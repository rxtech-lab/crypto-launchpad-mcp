package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
)

func NewRemoveUniswapDeploymentTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("remove_uniswap_deployment",
		mcp.WithDescription("Remove one or multiple Uniswap deployments by their IDs. Accepts a single ID or comma-separated list of IDs for bulk removal. This will also clear any related Uniswap configuration settings."),
		mcp.WithString("ids",
			mcp.Required(),
			mcp.Description("Uniswap deployment ID(s) to remove. Can be a single ID (e.g., '1') or comma-separated list (e.g., '1,2,3,4')"),
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
			return mcp.NewToolResultError("No deployment IDs provided"), nil
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
			return mcp.NewToolResultError("No valid deployment IDs provided"), nil
		}

		// Check if deployments exist before removal
		var existingDeployments []uint
		var deploymentDetails []map[string]interface{}

		for _, id := range ids {
			deployment, err := db.GetUniswapDeploymentByID(id)
			if err == nil && deployment != nil {
				existingDeployments = append(existingDeployments, id)
				deploymentDetails = append(deploymentDetails, map[string]interface{}{
					"id":      deployment.ID,
					"version": deployment.Version,
					"chain":   deployment.Chain,
					"status":  deployment.Status,
				})
			}
		}

		if len(existingDeployments) == 0 {
			result := map[string]interface{}{
				"requested_ids": ids,
				"removed_count": 0,
				"not_found_ids": ids,
				"message":       "No Uniswap deployments found with the provided IDs",
			}
			resultJSON, _ := json.Marshal(result)
			return mcp.NewToolResultText(fmt.Sprintf("No deployments removed: %s", string(resultJSON))), nil
		}

		// Perform bulk removal
		removedCount, err := db.DeleteUniswapDeployments(existingDeployments)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error removing Uniswap deployments: %v", err)), nil
		}

		// Clear Uniswap configuration for removed deployments
		for _, deployment := range deploymentDetails {
			version := deployment["version"].(string)
			// Clear configuration if it matches any of the removed deployments
			if err := db.ClearUniswapConfiguration(version); err != nil {
				// Log error but don't fail the operation
				fmt.Printf("Warning: Failed to clear Uniswap configuration for version %s: %v\n", version, err)
			}
		}

		// Find which IDs were not found
		var notFoundIds []uint
		for _, requestedId := range ids {
			found := false
			for _, existingId := range existingDeployments {
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
			"requested_ids":       ids,
			"removed_count":       removedCount,
			"removed_ids":         existingDeployments,
			"removed_deployments": deploymentDetails,
		}

		if len(notFoundIds) > 0 {
			result["not_found_ids"] = notFoundIds
		}

		var message string
		if len(ids) == 1 {
			if removedCount == 1 {
				message = "Uniswap deployment removed successfully"
			} else {
				message = "Uniswap deployment not found"
			}
		} else {
			message = fmt.Sprintf("%d Uniswap deployment(s) removed successfully", removedCount)
			if len(notFoundIds) > 0 {
				message += fmt.Sprintf(", %d deployment(s) not found", len(notFoundIds))
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
