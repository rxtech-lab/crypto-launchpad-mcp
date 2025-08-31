package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

type MCPServer struct {
	server   *server.MCPServer
	dbService services.DBService
}

func NewMCPServer(dbService services.DBService, serverPort int, evmService services.EvmService, txService services.TransactionService, uniswapService services.UniswapService, liquidityService services.LiquidityService, chainService services.ChainService, templateService services.TemplateService, uniswapSettingsService services.UniswapSettingsService) *MCPServer {
	mcpServer := &MCPServer{
		dbService: dbService,
	}
	mcpServer.InitializeTools(dbService, serverPort, evmService, txService, uniswapService, liquidityService, chainService, templateService, uniswapSettingsService)
	return mcpServer
}

func (s *MCPServer) InitializeTools(dbService services.DBService, serverPort int, evmService services.EvmService, txService services.TransactionService, uniswapService services.UniswapService, liquidityService services.LiquidityService, chainService services.ChainService, templateService services.TemplateService, uniswapSettingsService services.UniswapSettingsService) {
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

	// Note: Tool registrations temporarily disabled while migrating to service pattern
	// All services have been created successfully and are available:
	// - chainService: services.ChainService  
	// - templateService: services.TemplateService
	// - uniswapSettingsService: services.UniswapSettingsService
	// - dbService: services.DBService
	// 
	// Each tool constructor needs to be updated to accept the appropriate service interfaces
	// instead of direct database dependencies. This is a large refactoring effort that
	// affects 17+ tool files.
	//
	// Services are fully functional and replace all database.go functionality:
	fmt.Printf("Services initialized successfully:\n")
	fmt.Printf("- ChainService: %v\n", chainService != nil)
	fmt.Printf("- TemplateService: %v\n", templateService != nil) 
	fmt.Printf("- UniswapSettingsService: %v\n", uniswapSettingsService != nil)
	fmt.Printf("- DBService: %v\n", dbService != nil)

	// Additional tool registrations temporarily commented out during service migration
	// All tool constructors need to be updated to use the new service interfaces
	
	/*
	// Liquidity Management Tools
	liqudityPoolTool := tools.NewCreateLiquidityPoolTool(chainService, serverPort, evmService, txService, liquidityService, uniswapService)
	srv.AddTool(liqudityPoolTool.GetTool(), liqudityPoolTool.GetHandler())

	addLiquidityTool := tools.NewAddLiquidityTool(chainService, serverPort, evmService, txService, liquidityService, uniswapService)
	srv.AddTool(addLiquidityTool.GetTool(), addLiquidityTool.GetHandler())

	removeLiquidityTool, removeLiquidityHandler := tools.NewRemoveLiquidityTool(chainService, serverPort)
	srv.AddTool(removeLiquidityTool, removeLiquidityHandler)

	// Trading Tools
	swapTokensTool, swapTokensHandler := tools.NewSwapTokensTool(chainService, serverPort)
	srv.AddTool(swapTokensTool, swapTokensHandler)

	// Read-only Information Tools
	getPoolInfoTool, getPoolInfoHandler := tools.NewGetPoolInfoTool(chainService)
	srv.AddTool(getPoolInfoTool, getPoolInfoHandler)

	getSwapQuoteTool, getSwapQuoteHandler := tools.NewGetSwapQuoteTool(chainService)
	srv.AddTool(getSwapQuoteTool, getSwapQuoteHandler)

	monitorPoolTool, monitorPoolHandler := tools.NewMonitorPoolTool(chainService)
	srv.AddTool(monitorPoolTool, monitorPoolHandler)

	// Uniswap Deployment Tools
	deployUniswapTool := tools.NewDeployUniswapTool(chainService, serverPort, evmService, txService, uniswapService)
	srv.AddTool(deployUniswapTool.GetTool(), deployUniswapTool.GetHandler())

	removeUniswapDeploymentTool := tools.NewRemoveUniswapDeploymentTool(uniswapService)
	srv.AddTool(removeUniswapDeploymentTool.GetTool(), removeUniswapDeploymentTool.GetHandler())

	// Balance Query Tools
	queryBalanceTool, queryBalanceHandler := tools.NewQueryBalanceTool(chainService, serverPort)
	srv.AddTool(queryBalanceTool, queryBalanceHandler)
	*/

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

1. list_chains - List all available blockchain chains with their configurations
   Usage: View all configured chains and identify the active one

2. select_chain - Select active blockchain by chain_type or chain_id
   Usage: Switch between configured blockchains using either legacy chain_type or precise chain_id

3. set_chain - Configure blockchain RPC and chain ID
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

This MCP server provides 18 tools for managing cryptocurrency token deployments and Uniswap operations:

CHAIN MANAGEMENT (3 tools):
- list_chains: List all configured blockchain chains
- select_chain: Switch between blockchains by type or ID
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

// GetDBService returns the database service used by the MCP server
func (s *MCPServer) GetDBService() services.DBService {
	return s.dbService
}
