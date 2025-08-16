package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/tools"
)

type MCPServer struct {
	server *server.MCPServer
	db     *database.Database
}

func NewMCPServer(db *database.Database, serverPort int) *MCPServer {
	mcpServer := &MCPServer{
		db: db,
	}
	mcpServer.InitializeTools(db, serverPort)
	return mcpServer
}

func (s *MCPServer) InitializeTools(db *database.Database, serverPort int) {
	srv := server.NewMCPServer(
		"Crypto Launchpad MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
	)
	srv.EnableSampling()

	srv.AddPrompt(mcp.NewPrompt("launchpad-mcp-usage",
		mcp.WithPromptDescription("Instructions and guidance for using launchpad MCP tools"),
		mcp.WithArgument("tool_category",
			mcp.ArgumentDescription("Category of tools to get instructions for (chain, template, deployment, uniswap, balance, or all)"),
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

	deleteTemplateTool, deleteTemplateHandler := tools.NewDeleteTemplateTool(db)
	srv.AddTool(deleteTemplateTool, deleteTemplateHandler)

	// Deployment Tools
	launchTool, launchHandler := tools.NewLaunchTool(db, serverPort)
	srv.AddTool(launchTool, launchHandler)

	listDeploymentsTool, listDeploymentsHandler := tools.NewListDeploymentsTool(db)
	srv.AddTool(listDeploymentsTool, listDeploymentsHandler)

	// Uniswap Configuration Tools
	setUniswapVersionTool, setUniswapVersionHandler := tools.NewSetUniswapVersionTool(db)
	srv.AddTool(setUniswapVersionTool, setUniswapVersionHandler)

	getUniswapAddressesTool, getUniswapAddressesHandler := tools.NewGetUniswapAddressesTool(db)
	srv.AddTool(getUniswapAddressesTool, getUniswapAddressesHandler)

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

	// Uniswap Deployment Tools
	deployUniswapTool, deployUniswapHandler := tools.NewDeployUniswapTool(db, serverPort)
	srv.AddTool(deployUniswapTool, deployUniswapHandler)

	// Balance Query Tools
	queryBalanceTool, queryBalanceHandler := tools.NewQueryBalanceTool(db, serverPort)
	srv.AddTool(queryBalanceTool, queryBalanceHandler)

	s.server = srv
}

func (s *MCPServer) SendMessageToAiClient(messages []mcp.SamplingMessage) error {
	samplingRequest := mcp.CreateMessageRequest{
		CreateMessageParams: mcp.CreateMessageParams{
			Messages: messages,
		},
	}

	samplingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	serverFromCtx := server.ServerFromContext(samplingCtx)
	_, err := serverFromCtx.RequestSampling(samplingCtx, samplingRequest)
	if err != nil {
		return err
	}
	return nil
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
   Usage: Modify existing contract templates

4. delete_template - Delete templates by ID(s)
   Usage: Remove one or multiple templates (supports bulk deletion)`

	case "deployment":
		return `Deployment Tools:

1. launch - Generate deployment URL with signing interface
   Usage: Deploy contracts through a web interface that opens for wallet signing

2. list_deployments - List all token deployments with filtering options
   Usage: View all deployed contracts with status, addresses, and transaction details`

	case "uniswap":
		return `Uniswap Integration Tools:

1. deploy_uniswap - Deploy Uniswap infrastructure contracts (factory, router, WETH)
   Usage: Deploy complete Uniswap V2 infrastructure to enable trading

2. set_uniswap_version - Configure Uniswap version and contract addresses
   Usage: Set Uniswap version (v2/v3/v4) and all required contract addresses

3. get_uniswap_addresses - Get current Uniswap configuration
   Usage: Retrieve the active Uniswap version and contract addresses

4. create_liquidity_pool - Create new liquidity pool with signing interface
   Usage: Initialize new trading pairs on Uniswap

5. add_liquidity - Add liquidity to existing pool with signing interface
   Usage: Provide liquidity to earn trading fees

6. remove_liquidity - Remove liquidity from pool with signing interface
   Usage: Withdraw liquidity positions

7. swap_tokens - Execute token swaps with signing interface
   Usage: Trade tokens through Uniswap

8. get_pool_info - Retrieve pool metrics (read-only)
   Usage: Get current pool statistics and information

9. get_swap_quote - Get swap estimates and price impact (read-only)
   Usage: Calculate swap amounts and price impact before trading

10. monitor_pool - Real-time pool monitoring and event tracking (read-only)
    Usage: Track pool activity and events`

	case "balance":
		return `Balance Query Tools:

1. query_balance - Query wallet balance for native tokens and ERC-20 tokens
   Usage: Get wallet balances either directly in response or through web interface
   Parameters:
   - wallet_address (optional): Target wallet address 
   - show_browser (required): true for web interface, false for direct response
   - token_address (optional): ERC-20 token contract address for token balance`

	case "all":
		return `Crypto Launchpad MCP Tools Overview:

This MCP server provides 17 tools for managing cryptocurrency token deployments and Uniswap operations:

CHAIN MANAGEMENT (2 tools):
- select_chain: Switch between ethereum/solana
- set_chain: Configure RPC endpoints

TEMPLATE MANAGEMENT (4 tools):
- list_template: Browse contract templates
- create_template: Add new templates
- update_template: Modify existing templates
- delete_template: Delete templates by ID(s)

DEPLOYMENT (2 tools):
- launch: Deploy contracts via web interface
- list_deployments: View all deployed contracts

UNISWAP INTEGRATION (10 tools):
- deploy_uniswap: Deploy Uniswap infrastructure contracts
- set_uniswap_version: Configure Uniswap version and addresses
- get_uniswap_addresses: Get current Uniswap configuration
- create_liquidity_pool: Create new pools
- add_liquidity: Provide liquidity
- remove_liquidity: Withdraw liquidity
- swap_tokens: Trade tokens
- get_pool_info: View pool metrics
- get_swap_quote: Calculate swap estimates
- monitor_pool: Track pool activity

BALANCE QUERY (1 tool):
- query_balance: Query wallet balances with browser/direct modes

All signing operations open a web interface for secure wallet interaction.
No private keys are handled by the server - all signing is client-side.`

	default:
		return `Invalid category. Available categories: chain, template, deployment, uniswap, balance, all`
	}
}

func (s *MCPServer) Start() error {
	return server.ServeStdio(s.server)
}

// GetDatabase returns the database instance used by the MCP server
func (s *MCPServer) GetDatabase() *database.Database {
	return s.db
}

// CallMCPMethod provides a way to execute MCP tool functionality from the API server
// This is a helper method that demonstrates how to access MCP functionality
func (s *MCPServer) CallMCPMethod(method string, params map[string]interface{}) (interface{}, error) {
	switch method {
	case "list_templates":
		// Example: get templates by chain type
		chainType, ok := params["chain_type"].(string)
		if !ok {
			chainType = ""
		}
		keyword, _ := params["keyword"].(string)
		limit, ok := params["limit"].(int)
		if !ok {
			limit = 0 // 0 means no limit
		}
		return s.db.ListTemplates(chainType, keyword, limit)

	case "list_deployments":
		// Example: get all deployments
		return s.db.ListDeployments()

	case "get_active_chain":
		// Example: get active chain configuration
		return s.db.GetActiveChain()

	case "list_chains":
		// Example: get all chain configurations
		return s.db.ListChains()

	default:
		return nil, fmt.Errorf("unsupported MCP method: %s", method)
	}
}
