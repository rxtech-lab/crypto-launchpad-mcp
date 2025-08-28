package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

type addLiquidityTool struct {
	db               *database.Database
	evmService       services.EvmService
	txService        services.TransactionService
	liquidityService services.LiquidityService
	serverPort       int
}

type AddLiquidityArguments struct {
	// Required fields
	TokenAddress   string `json:"token_address" validate:"required"`
	TokenAmount    string `json:"token_amount" validate:"required"`
	ETHAmount      string `json:"eth_amount" validate:"required"`
	MinTokenAmount string `json:"min_token_amount" validate:"required"`
	MinETHAmount   string `json:"min_eth_amount" validate:"required"`

	// Optional fields
	Metadata []models.TransactionMetadata `json:"metadata,omitempty"`
}

func NewAddLiquidityTool(db *database.Database, serverPort int, evmService services.EvmService, txService services.TransactionService, liquidityService services.LiquidityService) *addLiquidityTool {
	return &addLiquidityTool{
		db:               db,
		evmService:       evmService,
		txService:        txService,
		liquidityService: liquidityService,
		serverPort:       serverPort,
	}
}

func (a *addLiquidityTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("add_liquidity",
		mcp.WithDescription("Add liquidity to existing Uniswap pool with signing interface. Generates a URL where users can connect wallet and sign the liquidity addition transaction."),
		mcp.WithString("token_address",
			mcp.Required(),
			mcp.Description("Address of the token in the pool"),
		),
		mcp.WithString("token_amount",
			mcp.Required(),
			mcp.Description("Amount of tokens to add to the pool"),
		),
		mcp.WithString("eth_amount",
			mcp.Required(),
			mcp.Description("Amount of ETH to add to the pool"),
		),
		mcp.WithString("min_token_amount",
			mcp.Required(),
			mcp.Description("Minimum amount of tokens (slippage protection)"),
		),
		mcp.WithString("min_eth_amount",
			mcp.Required(),
			mcp.Description("Minimum amount of ETH (slippage protection)"),
		),
		mcp.WithArray("metadata",
			mcp.Description("JSON array of metadata for the transaction (e.g., [{\"key\": \"Liquidity Action\", \"value\": \"Add Liquidity\"}]). Optional."),
			mcp.Items(map[string]any{
				"key": map[string]any{
					"type":        "string",
					"description": "Key of the metadata",
				},
				"value": map[string]any{
					"type":        "string",
					"description": "Value of the metadata",
				},
			}),
		),
	)
	return tool
}

func (a *addLiquidityTool) GetHandler() server.ToolHandlerFunc {

	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args AddLiquidityArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Get active chain configuration
		activeChain, err := a.db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Currently only support Ethereum
		if activeChain.ChainType != models.TransactionChainTypeEthereum {
			return mcp.NewToolResultError(fmt.Sprintf("Uniswap pools are only supported on Ethereum, got %s", activeChain.ChainType)), nil
		}

		// Check if pool exists
		pool, err := a.liquidityService.GetLiquidityPoolByTokenAddress(args.TokenAddress)
		if err != nil {
			return mcp.NewToolResultError("Liquidity pool not found. Create pool first using create_liquidity_pool"), nil
		}

		// Check if pool is confirmed
		if pool.Status != models.TransactionStatusConfirmed {
			return mcp.NewToolResultError("Liquidity pool is not confirmed. Create pool first using create_liquidity_pool"), nil
		}

		// Verify Uniswap settings exist
		_, err = a.db.GetActiveUniswapSettings()
		if err != nil {
			return mcp.NewToolResultError("No Uniswap version selected. Please use set_uniswap_version tool first"), nil
		}

		// Create liquidity position record
		// User address will be set when wallet connects on the web interface
		position := &models.LiquidityPosition{
			PoolID:       pool.ID,
			UserAddress:  "", // Will be populated when wallet connects
			Token0Amount: args.TokenAmount,
			Token1Amount: args.ETHAmount,
			Action:       "add",
			Status:       models.TransactionStatusPending,
		}

		positionID, err := a.liquidityService.CreateLiquidityPosition(position)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating liquidity position record: %v", err)), nil
		}

		// Add position ID to metadata
		enhancedMetadata := append(args.Metadata, models.TransactionMetadata{
			Key:   "position_id",
			Value: strconv.FormatUint(uint64(positionID), 10),
		})

		// Create transaction session with empty deployment (will be populated by handler)
		sessionID, err := a.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
			TransactionDeployments: []models.TransactionDeployment{}, // Empty - will be populated on signing page
			ChainType:              models.TransactionChainTypeEthereum,
			ChainID:                activeChain.ID,
			Metadata:               enhancedMetadata,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating transaction session: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Transaction session created: %s", sessionID)),
				mcp.NewTextContent("Please sign the add liquidity transaction in the URL"),
				mcp.NewTextContent(fmt.Sprintf("http://localhost:%d/tx/%s", a.serverPort, sessionID)),
			},
		}, nil
	}
}
