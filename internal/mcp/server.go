package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/tools"
)

type MCPServer struct {
	server *server.MCPServer
}

func NewMCPServer(db *database.Database, serverPort int) *MCPServer {
	mcpServer := &MCPServer{}
	mcpServer.InitializeTools(db, serverPort)
	return mcpServer
}

func (s *MCPServer) InitializeTools(db *database.Database, serverPort int) {
	srv := server.NewMCPServer(
		"Crypto Launchpad MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	srv.AddPrompt(mcp.NewPrompt("launchpad-mcp-usage",
		mcp.WithPromptDescription("Instructions and guidance for using launchpad MCP tools"),
		mcp.WithArgument("tool_category",
			mcp.ArgumentDescription("Category of tools to get instructions for (chain, template, deployment, uniswap, or all)"),
			mcp.RequiredArgument(),
		),
	), func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		category := request.Params.Arguments["tool_category"]
		if category == "" {
			return nil, fmt.Errorf("tool_category is required")
		}

		instructions := getToolInstructions(category)

		return mcp.NewGetPromptResult(
			fmt.Sprintf("Launchpad MCP Tools - %s", category),
			[]mcp.PromptMessage{
				mcp.NewPromptMessage(
					mcp.RoleUser,
					mcp.NewTextContent(instructions),
				),
			},
		), nil
	})

	// Chain Management Tools
	selectChainTool, selectChainHandler := tools.NewSelectChainTool(db)
	srv.AddTool(selectChainTool, selectChainHandler)

	setChainTool, setChainHandler := tools.NewSetChainTool(db)
	srv.AddTool(setChainTool, setChainHandler)

	// Template Management Tools
	listTemplateTool, listTemplateHandler := tools.NewListTemplateTool(db)
	srv.AddTool(listTemplateTool, listTemplateHandler)

	createTemplateTool, createTemplateHandler := tools.NewCreateTemplateTool(db)
	srv.AddTool(createTemplateTool, createTemplateHandler)

	updateTemplateTool, updateTemplateHandler := tools.NewUpdateTemplateTool(db)
	srv.AddTool(updateTemplateTool, updateTemplateHandler)

	// Deployment Tool
	launchTool, launchHandler := tools.NewLaunchTool(db, serverPort)
	srv.AddTool(launchTool, launchHandler)

	// Uniswap Configuration Tool
	setUniswapVersionTool, setUniswapVersionHandler := tools.NewSetUniswapVersionTool(db)
	srv.AddTool(setUniswapVersionTool, setUniswapVersionHandler)

	// Liquidity Management Tools
	createLiquidityPoolTool, createLiquidityPoolHandler := tools.NewCreateLiquidityPoolTool(db, serverPort)
	srv.AddTool(createLiquidityPoolTool, createLiquidityPoolHandler)

	addLiquidityTool, addLiquidityHandler := tools.NewAddLiquidityTool(db, serverPort)
	srv.AddTool(addLiquidityTool, addLiquidityHandler)

	removeLiquidityTool, removeLiquidityHandler := tools.NewRemoveLiquidityTool(db, serverPort)
	srv.AddTool(removeLiquidityTool, removeLiquidityHandler)

	// Trading Tools
	swapTokensTool, swapTokensHandler := tools.NewSwapTokensTool(db, serverPort)
	srv.AddTool(swapTokensTool, swapTokensHandler)

	// Read-only Information Tools
	getPoolInfoTool, getPoolInfoHandler := tools.NewGetPoolInfoTool(db)
	srv.AddTool(getPoolInfoTool, getPoolInfoHandler)

	getSwapQuoteTool, getSwapQuoteHandler := tools.NewGetSwapQuoteTool(db)
	srv.AddTool(getSwapQuoteTool, getSwapQuoteHandler)

	monitorPoolTool, monitorPoolHandler := tools.NewMonitorPoolTool(db)
	srv.AddTool(monitorPoolTool, monitorPoolHandler)

	s.server = srv
}

func getToolInstructions(category string) string {
	switch category {
	case "chain":
		return `Chain Management Tools:

1. select_chain - Select active blockchain (ethereum/solana)
   Usage: Use this to switch between supported blockchains

2. set_chain - Configure blockchain RPC and chain ID
   Usage: Set up custom RPC endpoints and chain configurations`

	case "template":
		return `Template Management Tools:

1. list_template - List smart contract templates with search
   Usage: Browse available contract templates by chain type

2. create_template - Create new contract template with validation
   Usage: Add custom smart contract templates for deployment

3. update_template - Update existing template
   Usage: Modify existing contract templates`

	case "deployment":
		return `Deployment Tools:

1. launch - Generate deployment URL with signing interface
   Usage: Deploy contracts through a web interface that opens for wallet signing`

	case "uniswap":
		return `Uniswap Integration Tools:

1. set_uniswap_version - Configure Uniswap version (v2/v3/v4)
   Usage: Set which Uniswap version to use for operations

2. create_liquidity_pool - Create new liquidity pool with signing interface
   Usage: Initialize new trading pairs on Uniswap

3. add_liquidity - Add liquidity to existing pool with signing interface
   Usage: Provide liquidity to earn trading fees

4. remove_liquidity - Remove liquidity from pool with signing interface
   Usage: Withdraw liquidity positions

5. swap_tokens - Execute token swaps with signing interface
   Usage: Trade tokens through Uniswap

6. get_pool_info - Retrieve pool metrics (read-only)
   Usage: Get current pool statistics and information

7. get_swap_quote - Get swap estimates and price impact (read-only)
   Usage: Calculate swap amounts and price impact before trading

8. monitor_pool - Real-time pool monitoring and event tracking (read-only)
   Usage: Track pool activity and events`

	case "all":
		return `Crypto Launchpad MCP Tools Overview:

This MCP server provides 14 tools for managing cryptocurrency token deployments and Uniswap operations:

CHAIN MANAGEMENT (2 tools):
- select_chain: Switch between ethereum/solana
- set_chain: Configure RPC endpoints

TEMPLATE MANAGEMENT (3 tools):
- list_template: Browse contract templates
- create_template: Add new templates
- update_template: Modify existing templates

DEPLOYMENT (1 tool):
- launch: Deploy contracts via web interface

UNISWAP INTEGRATION (8 tools):
- set_uniswap_version: Configure Uniswap version
- create_liquidity_pool: Create new pools
- add_liquidity: Provide liquidity
- remove_liquidity: Withdraw liquidity
- swap_tokens: Trade tokens
- get_pool_info: View pool metrics
- get_swap_quote: Calculate swap estimates
- monitor_pool: Track pool activity

All signing operations open a web interface for secure wallet interaction.
No private keys are handled by the server - all signing is client-side.`

	default:
		return `Invalid category. Available categories: chain, template, deployment, uniswap, all`
	}
}

func (s *MCPServer) Start() error {
	return server.ServeStdio(s.server)
}
