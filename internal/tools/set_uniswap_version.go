package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
)

func NewSetUniswapVersionTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("set_uniswap_version",
		mcp.WithDescription("Set Uniswap version for liquidity operations. Stores version in database as active configuration. Currently only v2 is fully supported."),
		mcp.WithString("version",
			mcp.Required(),
			mcp.Description("Uniswap version to use (v2, v3, v4). Currently only v2 is fully supported"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		version, err := request.RequireString("version")
		if err != nil {
			return nil, fmt.Errorf("version parameter is required: %w", err)
		}

		// Validate version
		validVersions := []string{"v2", "v3", "v4"}
		isValid := false
		for _, v := range validVersions {
			if version == v {
				isValid = true
				break
			}
		}

		if !isValid {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Invalid version. Supported values: v2, v3, v4"),
				},
			}, nil
		}

		// Show warning for v3 and v4
		var warning string
		if version == "v3" || version == "v4" {
			warning = fmt.Sprintf("Warning: %s support is experimental. Only v2 is fully supported.", version)
		}

		// Set the Uniswap version
		if err := db.SetUniswapVersion(version); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error setting Uniswap version: %v", err)),
				},
			}, nil
		}

		// Get the active settings to confirm
		settings, err := db.GetActiveUniswapSettings()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error getting Uniswap settings: %v", err)),
				},
			}, nil
		}

		result := map[string]interface{}{
			"version":    settings.Version,
			"is_active":  settings.IsActive,
			"created_at": settings.CreatedAt,
			"message":    fmt.Sprintf("Successfully set Uniswap version to %s", version),
		}

		if warning != "" {
			result["warning"] = warning
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Success message: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
