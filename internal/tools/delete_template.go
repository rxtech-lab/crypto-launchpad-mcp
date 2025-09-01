package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

func NewDeleteTemplateTool(templateService services.TemplateService) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("delete_template",
		mcp.WithDescription("Delete one or multiple smart contract templates by their IDs. Accepts a single ID or comma-separated list of IDs for bulk deletion."),
		mcp.WithString("ids",
			mcp.Required(),
			mcp.Description("Template ID(s) to delete. Can be a single ID (e.g., '1') or comma-separated list (e.g., '1,2,3,4')"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		idsStr, err := request.RequireString("ids")
		if err != nil {
			return nil, fmt.Errorf("ids parameter is required: %w", err)
		}

		// Parse IDs from comma-separated string
		idStrings := strings.Split(strings.TrimSpace(idsStr), ",")
		if len(idStrings) == 0 {
			return mcp.NewToolResultError("No template IDs provided"), nil
		}

		var ids []uint
		var invalidIds []string

		for _, idStr := range idStrings {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}

			id, err := strconv.ParseUint(idStr, 10, 32)
			if err != nil {
				invalidIds = append(invalidIds, idStr)
				continue
			}
			ids = append(ids, uint(id))
		}

		if len(invalidIds) > 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid ID(s): %s. IDs must be positive integers", strings.Join(invalidIds, ", "))), nil
		}

		if len(ids) == 0 {
			return mcp.NewToolResultError("No valid template IDs provided"), nil
		}

		// Check if templates exist before deletion
		var existingTemplates []uint
		for _, id := range ids {
			template, err := templateService.GetTemplateByID(id)
			if err == nil && template != nil {
				existingTemplates = append(existingTemplates, id)
			}
		}

		if len(existingTemplates) == 0 {
			result := map[string]interface{}{
				"requested_ids": ids,
				"deleted_count": 0,
				"not_found_ids": ids,
				"message":       "No templates found with the provided IDs",
			}
			resultJSON, _ := json.Marshal(result)
			return mcp.NewToolResultText(fmt.Sprintf("No templates deleted: %s", string(resultJSON))), nil
		}

		// Perform bulk deletion
		deletedCount, err := templateService.DeleteTemplates(existingTemplates)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error deleting templates: %v", err)), nil
		}

		// Find which IDs were not found
		var notFoundIds []uint
		for _, requestedId := range ids {
			found := false
			for _, existingId := range existingTemplates {
				if requestedId == existingId {
					found = true
					break
				}
			}
			if !found {
				notFoundIds = append(notFoundIds, requestedId)
			}
		}

		// Prepare result
		result := map[string]interface{}{
			"requested_ids": ids,
			"deleted_count": deletedCount,
			"deleted_ids":   existingTemplates,
		}

		if len(notFoundIds) > 0 {
			result["not_found_ids"] = notFoundIds
		}

		var message string
		if len(ids) == 1 {
			if deletedCount == 1 {
				message = "Template deleted successfully"
			} else {
				message = "Template not found"
			}
		} else {
			message = fmt.Sprintf("%d template(s) deleted successfully", deletedCount)
			if len(notFoundIds) > 0 {
				message += fmt.Sprintf(", %d template(s) not found", len(notFoundIds))
			}
		}

		result["message"] = message

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(message + ": "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
