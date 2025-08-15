package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

func NewGetSwapQuoteTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_swap_quote",
		mcp.WithDescription("Get swap estimates and price impact for token swaps. This is a read-only operation that provides estimated output amounts and price impact calculations."),
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

		amountStr, err := request.RequireString("amount")
		if err != nil {
			return nil, fmt.Errorf("amount parameter is required: %w", err)
		}

		// Get active chain configuration
		activeChain, err := db.GetActiveChain()
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
		uniswapSettings, err := db.GetActiveUniswapSettings()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("No Uniswap version selected. Please use set_uniswap_version tool first"),
				},
			}, nil
		}

		// Determine which token pool to check
		var poolToken string
		if fromToken == "0x0" || fromToken == "0x0000000000000000000000000000000000000000" {
			poolToken = toToken
		} else if toToken == "0x0" || toToken == "0x0000000000000000000000000000000000000000" {
			poolToken = fromToken
		} else {
			// For token-to-token swaps, we need to check if a direct pool exists
			// For simplicity, we'll assume ETH is the intermediary
			poolToken = fromToken
		}

		// Get pool information
		pool, err := db.GetLiquidityPoolByTokenAddress(poolToken)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("No liquidity pool found for these tokens"),
				},
			}, nil
		}

		// Simple constant product formula calculation (x * y = k)
		// This is a basic estimation - real implementation would use on-chain data
		estimatedOutput, priceImpact, err := calculateSwapQuote(pool, fromToken, toToken, amountStr)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error calculating swap quote: %v", err)),
				},
			}, nil
		}

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
			"quote": map[string]interface{}{
				"from_token":         fromToken,
				"to_token":           toToken,
				"from_token_display": fromTokenDisplay,
				"to_token_display":   toTokenDisplay,
				"input_amount":       amountStr,
				"estimated_output":   estimatedOutput,
				"price_impact":       priceImpact,
				"uniswap_version":    uniswapSettings.Version,
			},
			"pool_info": map[string]interface{}{
				"pool_id":        pool.ID,
				"token_address":  pool.TokenAddress,
				"pair_address":   pool.PairAddress,
				"token0_reserve": pool.InitialToken0,
				"token1_reserve": pool.InitialToken1,
			},
			"warnings": []string{
				"This is an estimated quote based on current pool reserves",
				"Actual output may vary due to slippage and other transactions",
				"For accurate quotes, use on-chain RPC calls or Uniswap SDK",
			},
			"recommendations": map[string]interface{}{
				"suggested_slippage": "0.5%",
				"min_output":         fmt.Sprintf("%.6f", estimatedOutput*0.995), // 0.5% slippage
			},
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

// calculateSwapQuote provides a basic estimation using constant product formula
func calculateSwapQuote(pool *models.LiquidityPool, fromToken, _ /* toToken */, amountStr string) (float64, float64, error) {
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid amount: %v", err)
	}

	// Parse pool reserves
	token0Reserve, err := strconv.ParseFloat(pool.InitialToken0, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid token0 reserve: %v", err)
	}

	token1Reserve, err := strconv.ParseFloat(pool.InitialToken1, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid token1 reserve: %v", err)
	}

	// Determine direction of swap
	var inputReserve, outputReserve float64
	if fromToken == pool.Token0 || (fromToken == "0x0" && pool.Token1 == pool.TokenAddress) {
		inputReserve = token0Reserve
		outputReserve = token1Reserve
	} else {
		inputReserve = token1Reserve
		outputReserve = token0Reserve
	}

	// Constant product formula: (x + dx) * (y - dy) = x * y
	// dy = (y * dx) / (x + dx)
	// With 0.3% fee: dx_after_fee = dx * 0.997
	amountAfterFee := amount * 0.997

	// Calculate output using constant product formula
	outputAmount := (outputReserve * amountAfterFee) / (inputReserve + amountAfterFee)

	// Calculate price impact
	priceImpact := (amount / inputReserve) * 100

	return outputAmount, priceImpact, nil
}
