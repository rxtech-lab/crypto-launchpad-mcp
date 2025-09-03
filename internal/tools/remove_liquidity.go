package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewRemoveLiquidityTool(chainService services.ChainService, liquidityService services.LiquidityService, uniswapService services.UniswapService, txService services.TransactionService, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("remove_liquidity",
		mcp.WithDescription("Remove liquidity from Uniswap pool with signing interface. Generates a URL where users can connect wallet and sign the liquidity removal transaction."),
		mcp.WithString("token_address",
			mcp.Required(),
			mcp.Description("Address of the token in the pool"),
		),
		mcp.WithString("liquidity_amount",
			mcp.Required(),
			mcp.Description("Amount of liquidity tokens to remove"),
		),
		mcp.WithString("min_token_amount",
			mcp.Required(),
			mcp.Description("Minimum amount of tokens to receive (slippage protection)"),
		),
		mcp.WithString("min_eth_amount",
			mcp.Required(),
			mcp.Description("Minimum amount of ETH to receive (slippage protection)"),
		),
		mcp.WithString("user_address",
			mcp.Required(),
			mcp.Description("Address that will remove liquidity"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tokenAddress, err := request.RequireString("token_address")
		if err != nil {
			return nil, fmt.Errorf("token_address parameter is required: %w", err)
		}

		liquidityAmount, err := request.RequireString("liquidity_amount")
		if err != nil {
			return nil, fmt.Errorf("liquidity_amount parameter is required: %w", err)
		}

		minTokenAmount, err := request.RequireString("min_token_amount")
		if err != nil {
			return nil, fmt.Errorf("min_token_amount parameter is required: %w", err)
		}

		minETHAmount, err := request.RequireString("min_eth_amount")
		if err != nil {
			return nil, fmt.Errorf("min_eth_amount parameter is required: %w", err)
		}

		userAddress, err := request.RequireString("user_address")
		if err != nil {
			return nil, fmt.Errorf("user_address parameter is required: %w", err)
		}

		// Get active chain configuration
		activeChain, err := chainService.GetActiveChain()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("No active chain selected. Please use select_chain tool first"),
				},
			}, nil
		}

		// Currently only support Ethereum
		if activeChain.ChainType != "ethereum" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Uniswap pools are only supported on Ethereum"),
				},
			}, nil
		}

		// Check if pool exists
		pool, err := liquidityService.GetLiquidityPoolByTokenAddress(tokenAddress, "")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Liquidity pool not found"),
				},
			}, nil
		}

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

		// Get active Uniswap settings
		uniswapSettings, err := uniswapService.GetActiveUniswapDeployment(userId, *chain)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("No Uniswap version selected. Please use set_uniswap_version tool first"),
				},
			}, nil
		}

		// Prepare transaction data for signing
		transactionData := map[string]interface{}{
			"pool_id":          pool.ID,
			"token_address":    tokenAddress,
			"liquidity_amount": liquidityAmount,
			"min_token_amount": minTokenAmount,
			"min_eth_amount":   minETHAmount,
			"user_address":     userAddress,
			"chain_type":       activeChain.ChainType,
			"chain_id":         activeChain.NetworkID,
			"rpc":              activeChain.RPC,
		}

		transactionDataJSON, err := json.Marshal(transactionData)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error encoding transaction data: %v", err)),
				},
			}, nil
		}

		// Create transaction session
		chainIDUint, _ := strconv.ParseUint(activeChain.NetworkID, 10, 64)
		req := services.CreateTransactionSessionRequest{
			Metadata: []models.TransactionMetadata{
				{Key: "session_type", Value: "remove_liquidity"},
				{Key: "session_data", Value: string(transactionDataJSON)},
			},
			ChainType: activeChain.ChainType,
			ChainID:   uint(chainIDUint),
		}
		sessionID, err := txService.CreateTransactionSession(req)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error creating transaction session: %v", err)),
				},
			}, nil
		}

		// Generate signing URL
		signingURL := fmt.Sprintf("http://localhost:%d/liquidity/remove/%s", serverPort, sessionID)

		result := map[string]interface{}{
			"session_id":       sessionID,
			"signing_url":      signingURL,
			"token_address":    tokenAddress,
			"liquidity_amount": liquidityAmount,
			"min_token_amount": minTokenAmount,
			"min_eth_amount":   minETHAmount,
			"uniswap_version":  uniswapSettings.Version,
			"user_address":     userAddress,
			"message":          "Remove liquidity session created. Use the signing URL to connect wallet and remove liquidity.",
			"instructions":     "1. Open the signing URL in your browser\n2. Connect your wallet using EIP-6963\n3. Review the liquidity removal details\n4. Sign and send the transaction to remove liquidity",
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
