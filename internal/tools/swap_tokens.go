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
)

func NewSwapTokensTool(chainService services.ChainService, liquidityService services.LiquidityService, uniswapSettingsService services.UniswapSettingsService, txService services.TransactionService, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("swap_tokens",
		mcp.WithDescription("Execute token swaps via Uniswap with signing interface. Generates a URL where users can connect wallet and sign the swap transaction."),
		mcp.WithString("from_token",
			mcp.Required(),
			mcp.Description("Address of the token to swap from (use '0x0' for ETH)"),
		),
		mcp.WithString("to_token",
			mcp.Required(),
			mcp.Description("Address of the token to swap to (use '0x0' for ETH)"),
		),
		mcp.WithString("amount",
			mcp.Required(),
			mcp.Description("Amount of tokens to swap"),
		),
		mcp.WithString("slippage_tolerance",
			mcp.Required(),
			mcp.Description("Maximum slippage tolerance as percentage (e.g., '0.5' for 0.5%)"),
		),
		mcp.WithString("user_address",
			mcp.Required(),
			mcp.Description("Address that will execute the swap"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		fromToken, err := request.RequireString("from_token")
		if err != nil {
			return nil, fmt.Errorf("from_token parameter is required: %w", err)
		}

		toToken, err := request.RequireString("to_token")
		if err != nil {
			return nil, fmt.Errorf("to_token parameter is required: %w", err)
		}

		amount, err := request.RequireString("amount")
		if err != nil {
			return nil, fmt.Errorf("amount parameter is required: %w", err)
		}

		slippageTolerance, err := request.RequireString("slippage_tolerance")
		if err != nil {
			return nil, fmt.Errorf("slippage_tolerance parameter is required: %w", err)
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
					mcp.NewTextContent("Uniswap swaps are only supported on Ethereum"),
				},
			}, nil
		}

		// Validate that from_token and to_token are different
		if fromToken == toToken {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Cannot swap token to itself"),
				},
			}, nil
		}

		// Get active Uniswap settings
		uniswapSettings, err := uniswapSettingsService.GetActiveUniswapSettings()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("No Uniswap version selected. Please use set_uniswap_version tool first"),
				},
			}, nil
		}

		// Create swap transaction record
		swap := &models.SwapTransaction{
			UserAddress:       userAddress,
			FromToken:         fromToken,
			ToToken:           toToken,
			FromAmount:        amount,
			SlippageTolerance: slippageTolerance,
			Status:            "pending",
		}

		if _, err := liquidityService.CreateSwapTransaction(swap); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error creating swap transaction record: %v", err)),
				},
			}, nil
		}

		// Prepare transaction data for signing
		transactionData := map[string]interface{}{
			"swap_id":            swap.ID,
			"from_token":         fromToken,
			"to_token":           toToken,
			"amount":             amount,
			"slippage_tolerance": slippageTolerance,
			"user_address":       userAddress,
			"uniswap_version":    uniswapSettings.Version,
			"chain_type":         activeChain.ChainType,
			"chain_id":           activeChain.NetworkID,
			"rpc":                activeChain.RPC,
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
				{Key: "session_type", Value: "swap"},
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
		signingURL := fmt.Sprintf("http://localhost:%d/swap/%s", serverPort, sessionID)

		// Format token names for display
		fromTokenDisplay := fromToken
		toTokenDisplay := toToken
		if fromToken == "0x0" || fromToken == "0x0000000000000000000000000000000000000000" {
			fromTokenDisplay = "ETH"
		}
		if toToken == "0x0" || toToken == "0x0000000000000000000000000000000000000000" {
			toTokenDisplay = "ETH"
		}

		result := map[string]interface{}{
			"swap_id":            swap.ID,
			"session_id":         sessionID,
			"signing_url":        signingURL,
			"from_token":         fromToken,
			"to_token":           toToken,
			"from_token_display": fromTokenDisplay,
			"to_token_display":   toTokenDisplay,
			"amount":             amount,
			"slippage_tolerance": slippageTolerance,
			"uniswap_version":    uniswapSettings.Version,
			"user_address":       userAddress,
			"message":            fmt.Sprintf("Token swap session created. Swapping %s %s to %s.", amount, fromTokenDisplay, toTokenDisplay),
			"instructions":       "1. Open the signing URL in your browser\n2. Connect your wallet using EIP-6963\n3. Review the swap details and price impact\n4. Sign and send the transaction to execute the swap",
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
