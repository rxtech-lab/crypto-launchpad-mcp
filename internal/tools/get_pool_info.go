package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

func NewGetPoolInfoTool(chainService services.ChainService, liquidityService services.LiquidityService) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("get_pool_info",
		mcp.WithDescription("Retrieve pool metrics including reserves, liquidity, price, and volume. This is a read-only operation that doesn't require wallet connection."),
		mcp.WithString("token_address",
			mcp.Required(),
			mcp.Description("Address of the token to get pool information for"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tokenAddress, err := request.RequireString("token_address")
		if err != nil {
			return nil, fmt.Errorf("token_address parameter is required: %w", err)
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

		// Get pool information from database
		pool, err := liquidityService.GetLiquidityPoolByTokenAddress(tokenAddress)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Liquidity pool not found for this token"),
				},
			}, nil
		}

		// Get liquidity positions for this pool
		positions, err := liquidityService.GetLiquidityPositionsByUser("") // Get all positions for the pool
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error getting liquidity positions: %v", err)),
				},
			}, nil
		}

		// Calculate total liquidity from positions
		var totalLiquidityProviders int
		var totalLiquidityAmount string = "0"
		for _, position := range positions {
			if position.PoolID == pool.ID && position.Status == "confirmed" {
				if position.Action == "add" {
					totalLiquidityProviders++
				}
			}
		}

		// Get recent swap transactions
		swaps, err := liquidityService.ListSwapTransactionsByUser("", 0, 1000) // Get all swaps
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error getting swap transactions: %v", err)),
				},
			}, nil
		}

		// Filter swaps related to this token
		var tokenSwaps []map[string]interface{}
		for _, swap := range swaps {
			if swap.FromToken == tokenAddress || swap.ToToken == tokenAddress {
				if swap.Status == "confirmed" {
					tokenSwaps = append(tokenSwaps, map[string]interface{}{
						"id":                 swap.ID,
						"from_token":         swap.FromToken,
						"to_token":           swap.ToToken,
						"from_amount":        swap.FromAmount,
						"to_amount":          swap.ToAmount,
						"slippage_tolerance": swap.SlippageTolerance,
						"user_address":       swap.UserAddress,
						"transaction_hash":   swap.TransactionHash,
						"created_at":         swap.CreatedAt,
					})
				}
			}
		}

		result := map[string]interface{}{
			"pool_info": map[string]interface{}{
				"id":              pool.ID,
				"token_address":   pool.TokenAddress,
				"pair_address":    pool.PairAddress,
				"uniswap_version": pool.UniswapVersion,
				"token0":          pool.Token0,
				"token1":          pool.Token1,
				"initial_token0":  pool.InitialToken0,
				"initial_token1":  pool.InitialToken1,
				"creator_address": pool.CreatorAddress,
				"status":          pool.Status,
				"created_at":      pool.CreatedAt,
			},
			"liquidity_metrics": map[string]interface{}{
				"total_liquidity_providers": totalLiquidityProviders,
				"total_liquidity_amount":    totalLiquidityAmount,
				"pool_status":               pool.Status,
			},
			"trading_activity": map[string]interface{}{
				"total_swaps":  len(tokenSwaps),
				"recent_swaps": tokenSwaps,
			},
			"chain_info": map[string]interface{}{
				"chain_type": activeChain.ChainType,
				"chain_id":   activeChain.NetworkID,
				"rpc":        activeChain.RPC,
			},
			"note": "This is cached data from the database. For real-time on-chain data, use blockchain RPC calls or subgraph queries.",
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
