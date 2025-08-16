package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
)

func NewGetUniswapAddressesTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_uniswap_addresses",
		mcp.WithDescription("Get current Uniswap configuration including version and contract addresses. Returns the active Uniswap settings from database."),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get the active Uniswap settings
		settings, err := db.GetActiveUniswapSettings()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("No active Uniswap configuration found. Please use set_uniswap_version tool first: %v", err)),
				},
			}, nil
		}

		result := map[string]interface{}{
			"version":          settings.Version,
			"router_address":   settings.RouterAddress,
			"factory_address":  settings.FactoryAddress,
			"weth_address":     settings.WETHAddress,
			"quoter_address":   settings.QuoterAddress,
			"position_manager": settings.PositionManager,
			"swap_router02":    settings.SwapRouter02,
			"is_active":        settings.IsActive,
			"created_at":       settings.CreatedAt,
			"updated_at":       settings.UpdatedAt,
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Current Uniswap configuration: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
