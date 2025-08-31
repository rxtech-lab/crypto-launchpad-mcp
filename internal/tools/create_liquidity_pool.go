package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type createLiquidityPoolTool struct {
	chainService     services.ChainService
	evmService       services.EvmService
	txService        services.TransactionService
	liquidityService services.LiquidityService
	uniswapService   services.UniswapService
	serverPort       int
}

type CreateLiquidityPoolArguments struct {
	// Required fields
	TokenAddress       string `json:"token_address" validate:"required"`
	InitialTokenAmount string `json:"initial_token_amount" validate:"required"`
	InitialETHAmount   string `json:"initial_eth_amount" validate:"required"`

	// Optional fields
	Metadata []models.TransactionMetadata `json:"metadata,omitempty"`
}

func NewCreateLiquidityPoolTool(chainService services.ChainService, serverPort int, evmService services.EvmService, txService services.TransactionService, liquidityService services.LiquidityService, uniswapService services.UniswapService) *createLiquidityPoolTool {
	return &createLiquidityPoolTool{
		chainService:     chainService,
		evmService:       evmService,
		txService:        txService,
		liquidityService: liquidityService,
		uniswapService:   uniswapService,
		serverPort:       serverPort,
	}
}

func (c *createLiquidityPoolTool) GetTool() mcp.Tool {
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
		mcp.WithArray("metadata",
			mcp.Description("JSON array of metadata for the transaction (e.g., [{\"key\": \"Pool Type\", \"value\": \"Liquidity Pool\"}]). Optional."),
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

func (c *createLiquidityPoolTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args CreateLiquidityPoolArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Get active chain configuration
		activeChain, err := c.db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Currently only support Ethereum
		if activeChain.ChainType != models.TransactionChainTypeEthereum {
			return mcp.NewToolResultError(fmt.Sprintf("Uniswap pools are only supported on Ethereum, got %s", activeChain.ChainType)), nil
		}

		return c.createEthereumLiquidityPool(ctx, args, activeChain)
	}
}

func (c *createLiquidityPoolTool) createEthereumLiquidityPool(ctx context.Context, args CreateLiquidityPoolArguments, activeChain *models.Chain) (*mcp.CallToolResult, error) {
	// Get active Uniswap settings
	uniswapSettings, err := c.db.GetActiveUniswapSettings()
	if err != nil {
		return mcp.NewToolResultError("No Uniswap version selected. Please use set_uniswap_version tool first"), nil
	}

	// Get Uniswap deployment to retrieve WETH address
	uniswapDeployment, err := c.uniswapService.GetUniswapDeploymentByChain(activeChain.ID)
	if err != nil {
		return mcp.NewToolResultError("No Uniswap deployment found for this chain. Please deploy Uniswap first using deploy_uniswap tool"), nil
	}

	// Verify WETH address is available
	if uniswapDeployment.WETHAddress == "" {
		return mcp.NewToolResultError("WETH address not found in Uniswap deployment. Please ensure Uniswap deployment is completed"), nil
	}

	// Check if pool already exists
	existingPool, err := c.liquidityService.GetLiquidityPoolByTokenAddress(args.TokenAddress)
	if err == nil && existingPool != nil {
		// Delete the pool if not confirmed
		if existingPool.Status != models.TransactionStatusConfirmed {
			if err := c.db.DeleteLiquidityPool(existingPool.ID); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to delete pending pool: %v", err)), nil
			}
		} else {
			return mcp.NewToolResultError("Liquidity pool already exists for this token"), nil
		}
	}

	// Create liquidity pool record
	// Creator address will be set when wallet connects on the web interface
	_, _, err = utils.CalculateInitialTokenPrice(args.InitialTokenAmount, args.InitialETHAmount, 18)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error calculating initial price: %v", err)), nil
	}

	pool := &models.LiquidityPool{
		TokenAddress:   args.TokenAddress,
		UniswapVersion: uniswapSettings.Version,
		Token0:         args.TokenAddress,
		Token1:         uniswapDeployment.WETHAddress, // Use WETH from Uniswap deployment
		InitialToken0:  args.InitialTokenAmount,
		InitialToken1:  args.InitialETHAmount,
		CreatorAddress: "", // Will be populated when wallet connects
		Status:         models.TransactionStatusPending,
	}

	poolID, err := c.liquidityService.CreateLiquidityPool(pool)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating liquidity pool record: %v", err)), nil
	}

	// Add pool ID to metadata
	enhancedMetadata := append(args.Metadata, models.TransactionMetadata{
		Key:   "pool_id",
		Value: strconv.FormatUint(uint64(poolID), 10),
	})

	// Create transaction session with empty deployment (will be populated by handler)
	sessionID, err := c.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
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
			mcp.NewTextContent("Please sign the liquidity pool creation transaction in the URL"),
			mcp.NewTextContent(fmt.Sprintf("http://localhost:%d/tx/%s", c.serverPort, sessionID)),
		},
	}, nil
}
