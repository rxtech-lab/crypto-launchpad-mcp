package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewSetUniswapVersionTool(db interface{}) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("set_uniswap_version",
		mcp.WithDescription("Set Uniswap version and contract addresses for liquidity operations. Stores configuration in database as active setting."),
		mcp.WithString("version",
			mcp.Required(),
			mcp.Description("Uniswap version to use (v2, v3, v4)"),
		),
		mcp.WithString("router_address",
			mcp.Required(),
			mcp.Description("Uniswap router contract address"),
		),
		mcp.WithString("factory_address",
			mcp.Required(),
			mcp.Description("Uniswap factory contract address"),
		),
		mcp.WithString("weth_address",
			mcp.Required(),
			mcp.Description("WETH contract address"),
		),
		mcp.WithString("quoter_address",
			mcp.Description("Quoter contract address (required for v3/v4, optional for v2)"),
		),
		mcp.WithString("position_manager",
			mcp.Description("Position manager contract address (required for v3/v4, optional for v2)"),
		),
		mcp.WithString("swap_router02",
			mcp.Description("SwapRouter02 contract address (optional for v3/v4)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		version, err := request.RequireString("version")
		if err != nil {
			return nil, fmt.Errorf("version parameter is required: %w", err)
		}

		routerAddress, err := request.RequireString("router_address")
		if err != nil {
			return nil, fmt.Errorf("router_address parameter is required: %w", err)
		}

		factoryAddress, err := request.RequireString("factory_address")
		if err != nil {
			return nil, fmt.Errorf("factory_address parameter is required: %w", err)
		}

		wethAddress, err := request.RequireString("weth_address")
		if err != nil {
			return nil, fmt.Errorf("weth_address parameter is required: %w", err)
		}

		// Optional parameters
		quoterAddress := request.GetString("quoter_address", "")
		positionManager := request.GetString("position_manager", "")
		swapRouter02 := request.GetString("swap_router02", "")

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

		// Validate v3/v4 requirements
		if (version == "v3" || version == "v4") && (quoterAddress == "" || positionManager == "") {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("quoter_address and position_manager are required for v3/v4"),
				},
			}, nil
		}

		// Set the Uniswap configuration
		if err := db.SetUniswapConfiguration(version, routerAddress, factoryAddress, wethAddress, quoterAddress, positionManager, swapRouter02); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error setting Uniswap configuration: %v", err)),
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
			"version":          settings.Version,
			"router_address":   settings.RouterAddress,
			"factory_address":  settings.FactoryAddress,
			"weth_address":     settings.WETHAddress,
			"quoter_address":   settings.QuoterAddress,
			"position_manager": settings.PositionManager,
			"swap_router02":    settings.SwapRouter02,
			"is_active":        settings.IsActive,
			"created_at":       settings.CreatedAt,
			"message":          fmt.Sprintf("Successfully configured Uniswap %s with contract addresses", version),
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
