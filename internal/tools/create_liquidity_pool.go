package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

func NewCreateLiquidityPoolTool(db *database.Database, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("create_liquidity_pool",
		mcp.WithDescription("Create new Uniswap liquidity pool with signing interface. Generates a URL where users can connect wallet and sign the pool creation transaction."),
		mcp.WithString("token_address",
			mcp.Required(),
			mcp.Description("Address of the token to create a pool for"),
		),
		mcp.WithString("initial_token_amount",
			mcp.Required(),
			mcp.Description("Initial amount of tokens to add to the pool"),
		),
		mcp.WithString("initial_eth_amount",
			mcp.Required(),
			mcp.Description("Initial amount of ETH to add to the pool"),
		),
		mcp.WithString("creator_address",
			mcp.Required(),
			mcp.Description("Address that will create the pool"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tokenAddress, err := request.RequireString("token_address")
		if err != nil {
			return nil, fmt.Errorf("token_address parameter is required: %w", err)
		}

		initialTokenAmount, err := request.RequireString("initial_token_amount")
		if err != nil {
			return nil, fmt.Errorf("initial_token_amount parameter is required: %w", err)
		}

		initialETHAmount, err := request.RequireString("initial_eth_amount")
		if err != nil {
			return nil, fmt.Errorf("initial_eth_amount parameter is required: %w", err)
		}

		creatorAddress, err := request.RequireString("creator_address")
		if err != nil {
			return nil, fmt.Errorf("creator_address parameter is required: %w", err)
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
					mcp.NewTextContent("Uniswap pools are only supported on Ethereum"),
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

		// Check if pool already exists
		existingPool, err := db.GetLiquidityPoolByTokenAddress(tokenAddress)
		if err == nil && existingPool != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Liquidity pool already exists for this token"),
				},
			}, nil
		}

		// Create liquidity pool record
		pool := &models.LiquidityPool{
			TokenAddress:   tokenAddress,
			UniswapVersion: uniswapSettings.Version,
			Token0:         tokenAddress,
			Token1:         "0x0000000000000000000000000000000000000000", // ETH placeholder
			InitialToken0:  initialTokenAmount,
			InitialToken1:  initialETHAmount,
			CreatorAddress: creatorAddress,
			Status:         "pending",
		}

		if err := db.CreateLiquidityPool(pool); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error creating liquidity pool record: %v", err)),
				},
			}, nil
		}

		// Prepare minimal session data (transaction data will be generated on-demand)
		sessionData := map[string]interface{}{
			"pool_id": pool.ID,
		}

		sessionDataJSON, err := json.Marshal(sessionData)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error encoding session data: %v", err)),
				},
			}, nil
		}

		// Create transaction session
		sessionID, err := db.CreateTransactionSession(
			"create_pool",
			activeChain.ChainType,
			activeChain.ChainID,
			string(sessionDataJSON),
		)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error creating transaction session: %v", err)),
				},
			}, nil
		}

		// Generate signing URL
		signingURL := fmt.Sprintf("http://localhost:%d/pool/create/%s", serverPort, sessionID)

		result := map[string]interface{}{
			"pool_id":              pool.ID,
			"session_id":           sessionID,
			"signing_url":          signingURL,
			"token_address":        tokenAddress,
			"initial_token_amount": initialTokenAmount,
			"initial_eth_amount":   initialETHAmount,
			"uniswap_version":      uniswapSettings.Version,
			"chain_type":           activeChain.ChainType,
			"creator_address":      creatorAddress,
			"message":              "Liquidity pool creation session created. Use the signing URL to connect wallet and create pool.",
			"instructions":         "1. Open the signing URL in your browser\n2. Connect your wallet using EIP-6963\n3. Review the pool creation details\n4. Sign and send the transaction to create the pool",
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
