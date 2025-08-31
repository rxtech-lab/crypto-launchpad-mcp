package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewMonitorPoolTool(db interface{}) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("monitor_pool",
		mcp.WithDescription("Real-time pool monitoring and event tracking. Returns current pool status, recent transactions, and activity metrics. This is a read-only operation."),
		mcp.WithString("token_address",
			mcp.Required(),
			mcp.Description("Address of the token to monitor pool for"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range for monitoring data (1h, 24h, 7d). Default: 24h"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tokenAddress, err := request.RequireString("token_address")
		if err != nil {
			return nil, fmt.Errorf("token_address parameter is required: %w", err)
		}

		timeRange := request.GetString("time_range", "24h")

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
					mcp.NewTextContent("Uniswap pools are only supported on Ethereum"),
				},
			}, nil
		}

		// Get pool information
		pool, err := db.GetLiquidityPoolByTokenAddress(tokenAddress)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Liquidity pool not found for this token"),
				},
			}, nil
		}

		// Get all liquidity positions for this pool
		allPositions, err := db.GetLiquidityPositionsByUser("")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error getting liquidity positions: %v", err)),
				},
			}, nil
		}

		// Filter positions for this pool
		var poolPositions []map[string]interface{}
		var addLiquidityCount, removeLiquidityCount int
		for _, position := range allPositions {
			if position.PoolID == pool.ID {
				poolPositions = append(poolPositions, map[string]interface{}{
					"id":               position.ID,
					"user_address":     position.UserAddress,
					"action":           position.Action,
					"liquidity_amount": position.LiquidityAmount,
					"token0_amount":    position.Token0Amount,
					"token1_amount":    position.Token1Amount,
					"status":           position.Status,
					"transaction_hash": position.TransactionHash,
					"created_at":       position.CreatedAt,
				})

				if position.Status == "confirmed" {
					if position.Action == "add" {
						addLiquidityCount++
					} else if position.Action == "remove" {
						removeLiquidityCount++
					}
				}
			}
		}

		// Get all swap transactions related to this token
		allSwaps, err := db.GetSwapTransactionsByUser("")
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error getting swap transactions: %v", err)),
				},
			}, nil
		}

		// Filter swaps for this token
		var tokenSwaps []map[string]interface{}
		var swapCount int
		for _, swap := range allSwaps {
			if swap.FromToken == tokenAddress || swap.ToToken == tokenAddress {
				tokenSwaps = append(tokenSwaps, map[string]interface{}{
					"id":                 swap.ID,
					"user_address":       swap.UserAddress,
					"from_token":         swap.FromToken,
					"to_token":           swap.ToToken,
					"from_amount":        swap.FromAmount,
					"to_amount":          swap.ToAmount,
					"slippage_tolerance": swap.SlippageTolerance,
					"status":             swap.Status,
					"transaction_hash":   swap.TransactionHash,
					"created_at":         swap.CreatedAt,
				})

				if swap.Status == "confirmed" {
					swapCount++
				}
			}
		}

		// Get all deployments to check if this token was deployed through the system
		deployments, err := db.ListDeployments()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error getting deployments: %v", err)),
				},
			}, nil
		}

		var tokenDeployment map[string]interface{}
		for _, deployment := range deployments {
			if deployment.ContractAddress == tokenAddress {
				tokenDeployment = map[string]interface{}{
					"id":               deployment.ID,
					"token_name":       deployment.TokenName,
					"token_symbol":     deployment.TokenSymbol,
					"deployer_address": deployment.DeployerAddress,
					"transaction_hash": deployment.TransactionHash,
					"status":           deployment.Status,
					"created_at":       deployment.CreatedAt,
				}
				break
			}
		}

		result := map[string]interface{}{
			"monitoring_info": map[string]interface{}{
				"token_address": tokenAddress,
				"time_range":    timeRange,
				"monitored_at":  "now", // In real implementation, this would be current timestamp
			},
			"pool_status": map[string]interface{}{
				"id":              pool.ID,
				"status":          pool.Status,
				"pair_address":    pool.PairAddress,
				"uniswap_version": pool.UniswapVersion,
				"created_at":      pool.CreatedAt,
			},
			"activity_metrics": map[string]interface{}{
				"total_liquidity_operations": addLiquidityCount + removeLiquidityCount,
				"add_liquidity_count":        addLiquidityCount,
				"remove_liquidity_count":     removeLiquidityCount,
				"total_swaps":                swapCount,
				"unique_liquidity_providers": len(poolPositions),
			},
			"recent_liquidity_operations": poolPositions,
			"recent_swaps":                tokenSwaps,
			"token_deployment":            tokenDeployment,
			"chain_info": map[string]interface{}{
				"chain_type": activeChain.ChainType,
				"chain_id":   activeChain.NetworkID,
				"rpc":        activeChain.RPC,
			},
			"notes": []string{
				"This monitoring data is based on transactions recorded in the database",
				"For real-time on-chain monitoring, consider using event listeners or subgraph queries",
				"Transaction counts may not reflect all on-chain activity if done outside this system",
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
