package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/tools"
)

type MCPServer struct {
	server    *server.MCPServer
	dbService services.DBService
}

func NewMCPServer(dbService services.DBService, serverPort int, evmService services.EvmService, txService services.TransactionService, uniswapService services.UniswapService, liquidityService services.LiquidityService, chainService services.ChainService, templateService services.TemplateService, deploymentService services.DeploymentService) *MCPServer {
	mcpServer := &MCPServer{
		dbService: dbService,
	}
	mcpServer.InitializeTools(dbService, serverPort, evmService, txService, uniswapService, liquidityService, chainService, templateService, deploymentService)
	return mcpServer
}

func (s *MCPServer) InitializeTools(dbService services.DBService, serverPort int, evmService services.EvmService, txService services.TransactionService, uniswapService services.UniswapService, liquidityService services.LiquidityService, chainService services.ChainService, templateService services.TemplateService, deploymentService services.DeploymentService) {
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
	selectChainTool, selectChainHandler := tools.NewSelectChainTool(chainService)
	srv.AddTool(selectChainTool, selectChainHandler)

	setChainTool, setChainHandler := tools.NewSetChainTool(chainService)
	srv.AddTool(setChainTool, setChainHandler)

	listChainsTool, listChainsHandler := tools.NewListChainsTool(chainService)
	srv.AddTool(listChainsTool, listChainsHandler)

	// Template Management Tools
	listTemplateTool, listTemplateHandler := tools.NewListTemplateTool(templateService)
	srv.AddTool(listTemplateTool, listTemplateHandler)

	createTemplateToolInstance := tools.NewCreateTemplateTool(templateService)
	srv.AddTool(createTemplateToolInstance.GetTool(), createTemplateToolInstance.GetHandler())

	updateTemplateToolInstance := tools.NewUpdateTemplateTool(templateService)
	srv.AddTool(updateTemplateToolInstance.GetTool(), updateTemplateToolInstance.GetHandler())

	deleteTemplateTool, deleteTemplateHandler := tools.NewDeleteTemplateTool(templateService)
	srv.AddTool(deleteTemplateTool, deleteTemplateHandler)

	viewTemplateToolInstance := tools.NewViewTemplateTool(templateService, evmService)
	srv.AddTool(viewTemplateToolInstance.GetTool(), viewTemplateToolInstance.GetHandler())

	// Deployment Tools
	launchTool := tools.NewLaunchTool(templateService, chainService, serverPort, evmService, txService, deploymentService)
	srv.AddTool(launchTool.GetTool(), launchTool.GetHandler())

	listDeploymentsTool, listDeploymentsHandler := tools.NewListDeploymentsTool(deploymentService)
	srv.AddTool(listDeploymentsTool, listDeploymentsHandler)

	addDeploymentTool := tools.NewAddDeploymentTool(deploymentService, templateService, chainService)
	srv.AddTool(addDeploymentTool.GetTool(), addDeploymentTool.GetHandler())

	// Function Call Tool
	callFunctionTool := tools.NewCallFunctionTool(templateService, evmService, txService, chainService, deploymentService, serverPort)
	srv.AddTool(callFunctionTool.GetTool(), callFunctionTool.GetHandler())

	// Uniswap Deployment Tools
	deployUniswapTool := tools.NewDeployUniswapTool(chainService, serverPort, evmService, txService, uniswapService)
	srv.AddTool(deployUniswapTool.GetTool(), deployUniswapTool.GetHandler())

	removeUniswapDeploymentTool := tools.NewRemoveUniswapDeploymentTool(uniswapService)
	srv.AddTool(removeUniswapDeploymentTool.GetTool(), removeUniswapDeploymentTool.GetHandler())

	getUniswapAddressesTool, getUniswapAddressesHandler := tools.NewGetUniswapAddressesTool(uniswapService, chainService)
	srv.AddTool(getUniswapAddressesTool, getUniswapAddressesHandler)

	setUniswapAddressesTool := tools.NewSetUniswapAddressesTool(uniswapService, chainService)
	srv.AddTool(setUniswapAddressesTool.GetTool(), setUniswapAddressesTool.GetHandler())

	// Liquidity Management Tools
	createLiquidityPoolTool := tools.NewCreateLiquidityPoolTool(chainService, serverPort, evmService, txService, liquidityService, uniswapService)
	srv.AddTool(createLiquidityPoolTool.GetTool(), createLiquidityPoolTool.GetHandler())

	addLiquidityTool := tools.NewAddLiquidityTool(chainService, serverPort, evmService, txService, liquidityService, uniswapService)
	srv.AddTool(addLiquidityTool.GetTool(), addLiquidityTool.GetHandler())

	removeLiquidityTool, removeLiquidityHandler := tools.NewRemoveLiquidityTool(chainService, liquidityService, uniswapService, txService, serverPort)
	srv.AddTool(removeLiquidityTool, removeLiquidityHandler)

	// Trading Tools
	swapTokensTool := tools.NewSwapTokensTool(chainService, liquidityService, uniswapService, txService, serverPort, evmService)
	srv.AddTool(swapTokensTool.GetTool(), swapTokensTool.GetHandler())

	// Read-only Information Tools
	getPoolInfoTool, getPoolInfoHandler := tools.NewGetPoolInfoTool(chainService, liquidityService)
	srv.AddTool(getPoolInfoTool, getPoolInfoHandler)

	getSwapQuoteTool, getSwapQuoteHandler := tools.NewGetSwapQuoteTool(chainService, liquidityService, uniswapService)
	srv.AddTool(getSwapQuoteTool, getSwapQuoteHandler)

	// Balance Query Tools
	queryBalanceTool, queryBalanceHandler := tools.NewQueryBalanceTool(chainService, txService, serverPort)
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
   Usage: Remove one or multiple templates (supports bulk deletion)

5. view_template - View template details and ABI methods
   Usage: View template information and optionally display contract ABI methods
   Parameters:
   - template_id (required): ID of the template to view
   - show_abi_methods (optional): Display all available ABI methods
   - abi_method (optional): Display details for a specific method by name`

	case "deployment":
		return `Deployment Tools:

1. launch - Generate deployment URL with signing interface
   Usage: Deploy contracts through a web interface that opens for wallet signing

2. list_deployments - List all token deployments with filtering options
   Usage: View all deployed contracts with status, addresses, and transaction details

3. call_function - Call smart contract functions using deployment ID and ABI
   Usage: Call functions on deployed contracts; read-only functions return results directly, state-changing functions create signing sessions
   Parameters:
   - deployment_id (required): ID of the deployment containing contract address and ABI
   - function_name (required): Name of function to call from contract's ABI
   - function_args (optional): Array of function arguments in ABI order
   - value (optional): ETH value to send with call (default "0")
   - metadata (optional): Transaction metadata for state-changing functions`

	case "uniswap":
		return `Uniswap Integration Tools:

1. deploy_uniswap - Deploy Uniswap infrastructure contracts (factory, router, WETH)
   Usage: Deploy complete Uniswap V2 infrastructure to enable trading

2. get_uniswap_addresses - Get current Uniswap configuration
   Usage: Retrieve the active Uniswap version and contract addresses

3. set_uniswap_addresses - Set or update Uniswap contract addresses
   Usage: Manually configure factory, router, and WETH addresses for externally deployed contracts

4. remove_uniswap_deployment - Remove Uniswap deployments by IDs
   Usage: Delete one or multiple Uniswap deployment records

5. create_liquidity_pool - Create new liquidity pool with signing interface
   Usage: Initialize new trading pairs on Uniswap

6. add_liquidity - Add liquidity to existing pool with signing interface
   Usage: Provide liquidity to earn trading fees

7. remove_liquidity - Remove liquidity from pool with signing interface
   Usage: Withdraw liquidity positions

8. swap_tokens - Execute token swaps with signing interface
   Usage: Trade tokens through Uniswap

9. get_pool_info - Retrieve pool metrics (read-only)
   Usage: Get current pool statistics and information

10. get_swap_quote - Get swap estimates and price impact (read-only)
    Usage: Calculate swap amounts and price impact before trading

11. monitor_pool - Real-time pool monitoring and event tracking (read-only)
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

This MCP server provides 21 tools for managing cryptocurrency token deployments and Uniswap operations:

CHAIN MANAGEMENT (3 tools):
- list_chains: List all configured blockchain chains
- select_chain: Switch between blockchains by type or ID
- set_chain: Configure RPC endpoints

TEMPLATE MANAGEMENT (5 tools):
- list_template: Browse contract templates
- create_template: Add new templates
- update_template: Modify existing templates
- delete_template: Delete templates by ID(s)
- view_template: View template details and ABI methods

DEPLOYMENT (3 tools):
- launch: Deploy contracts via web interface
- list_deployments: View all deployed contracts
- call_function: Call smart contract functions using deployment ID and ABI

UNISWAP INTEGRATION (11 tools):
- deploy_uniswap: Deploy Uniswap infrastructure contracts
- get_uniswap_addresses: Get current Uniswap configuration
- set_uniswap_addresses: Set or update Uniswap contract addresses
- remove_uniswap_deployment: Remove Uniswap deployments by IDs
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

func (s *MCPServer) StartStdioServer() error {
	return server.ServeStdio(s.server)
}

// StartStreamableHTTPServer starts the MCP server with streamable HTTP interface on the specified port
func (s *MCPServer) StartStreamableHTTPServer() *server.StreamableHTTPServer {
	return server.NewStreamableHTTPServer(s.server)
}

// GetDBService returns the database service used by the MCP server
func (s *MCPServer) GetDBService() services.DBService {
	return s.dbService
}
