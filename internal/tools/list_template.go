package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewListTemplateTool(templateService services.TemplateService) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("list_template",
		mcp.WithDescription("List predefined smart contract templates with optional filtering by chain type and keyword search. Uses SQLite search for template names and descriptions. Will only return a list of templates with their names, descriptions, and chain types. Call view_template tool to get detailed information including all available methods and method parameters."),
		mcp.WithString("chain_type",
			mcp.Description("Filter by blockchain type (ethereum or solana). If not provided, lists templates for all chains."),
		),
		mcp.WithString("keyword",
			mcp.Description("Search keyword to filter templates by name or description"),
		),
		mcp.WithString("limit",
			mcp.Description("Maximum number of templates to return (default: 10)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chainType := request.GetString("chain_type", "")
		keyword := request.GetString("keyword", "")
		limitStr := request.GetString("limit", "10")

		// Parse limit
		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			limit = 10 // Default limit
		}

		// Validate chain type if provided
		if chainType != "" && chainType != "ethereum" && chainType != "solana" {
			return mcp.NewToolResultError("Invalid chain_type. Supported values: ethereum, solana"), nil
		}

		user, _ := utils.GetAuthenticatedUser(ctx)
		// List templates with filters
		var userId *string
		if user != nil {
			userId = &user.Sub
		}
		templates, err := templateService.ListTemplates(userId, chainType, keyword, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error listing templates: %v", err)), nil
		}

		if len(templates) == 0 {
			result := map[string]any{
				"templates": []interface{}{},
				"count":     0,
				"message":   "No templates found matching the criteria",
			}
			resultJSON, _ := json.Marshal(result)
			return mcp.NewToolResultText(fmt.Sprintf("Templates listed: %s", string(resultJSON))), nil
		}

		// Format templates for response
		templateList := make([]map[string]any, len(templates))
		for i, template := range templates {
			templateList[i] = map[string]any{
				"id":          template.ID,
				"name":        template.Name,
				"description": template.Description,
				"chain_type":  template.ChainType,
			}
		}

		result := map[string]any{
			"templates": templateList,
			"count":     len(templates),
			"filters": map[string]any{
				"chain_type": chainType,
				"keyword":    keyword,
				"limit":      limit,
			},
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Templates listed successfully: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
