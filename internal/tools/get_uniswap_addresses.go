package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewGetUniswapAddressesTool(uniswapService services.UniswapService, chainService services.ChainService) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_uniswap_addresses",
		mcp.WithDescription("Get current Uniswap configuration including version and contract addresses. Returns the active Uniswap settings from database."),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get the active Uniswap settings
		user, _ := utils.GetAuthenticatedUser(ctx)
		var userId *string
		if user != nil {
			userId = &user.Sub
		}

		// get active chain
		chain, err := chainService.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("Unable to get active chain. Is there any chain selected?"), nil
		}

		// Fetch active Uniswap deployment
		settings, err := uniswapService.GetActiveUniswapDeployment(userId, *chain)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("No active Uniswap configuration found. Please use set_uniswap_version tool first: %v", err)),
				},
			}, nil
		}

		resultJSON, _ := json.Marshal(settings)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Current Uniswap configuration: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
