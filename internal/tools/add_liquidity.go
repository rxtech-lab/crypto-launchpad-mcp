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
)

type addLiquidityTool struct {
	chainService           services.ChainService
	evmService             services.EvmService
	txService              services.TransactionService
	liquidityService       services.LiquidityService
	uniswapService         services.UniswapService
	uniswapSettingsService services.UniswapSettingsService
	serverPort             int
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

func NewAddLiquidityTool(chainService services.ChainService, serverPort int, evmService services.EvmService, txService services.TransactionService, liquidityService services.LiquidityService, uniswapService services.UniswapService, uniswapSettingsService services.UniswapSettingsService) *addLiquidityTool {
	return &addLiquidityTool{
		chainService:           chainService,
		evmService:             evmService,
		txService:              txService,
		liquidityService:       liquidityService,
		uniswapService:         uniswapService,
		uniswapSettingsService: uniswapSettingsService,
		serverPort:             serverPort,
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
		activeChain, err := a.chainService.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Currently only support Ethereum
		if activeChain.ChainType != models.TransactionChainTypeEthereum {
			return mcp.NewToolResultError(fmt.Sprintf("Uniswap liquidity operations are only supported on Ethereum, got %s", activeChain.ChainType)), nil
		}

		// Delegate to Ethereum-specific implementation
		return a.createEthereumAddLiquidity(ctx, args, activeChain)
	}
}

func (a *addLiquidityTool) createEthereumAddLiquidity(ctx context.Context, args AddLiquidityArguments, activeChain *models.Chain) (*mcp.CallToolResult, error) {
	// Check if pool exists
	pool, err := a.liquidityService.GetLiquidityPoolByTokenAddress(args.TokenAddress)
	if err != nil {
		return mcp.NewToolResultError("Liquidity pool not found. Please create a pool first using create_liquidity_pool tool"), nil
	}

	// Check if pool is confirmed
	if pool.Status != models.TransactionStatusConfirmed {
		return mcp.NewToolResultError("Liquidity pool is not confirmed yet. Please wait for the pool creation transaction to be confirmed"), nil
	}

	// Verify pool has a pair address
	if pool.PairAddress == "" {
		return mcp.NewToolResultError("Liquidity pool does not have a pair address. Please ensure the pool was created successfully"), nil
	}

	// Verify Uniswap settings exist
	uniswapSettings, err := a.uniswapSettingsService.GetActiveUniswapSettings()
	if err != nil {
		return mcp.NewToolResultError("No Uniswap version selected. Please use set_uniswap_version tool first"), nil
	}

	// Get Uniswap deployment to retrieve router address
	uniswapDeployment, err := a.uniswapService.GetUniswapDeploymentByChain(activeChain.ID)
	if err != nil {
		return mcp.NewToolResultError("No Uniswap deployment found for this chain. Please deploy Uniswap first using deploy_uniswap tool"), nil
	}

	// Verify router address is available
	if uniswapDeployment.RouterAddress == "" {
		return mcp.NewToolResultError("Uniswap router address not found. Please ensure Uniswap deployment is completed"), nil
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
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create liquidity position record: %v", err)), nil
	}

	// Prepare enhanced metadata
	enhancedMetadata := a.prepareMetadata(args.Metadata, pool, position, positionID, uniswapSettings.Version)

	// Create transaction session
	sessionID, err := a.createAddLiquidityTransactionSession(
		activeChain,
		pool,
		uniswapDeployment,
		args,
		enhancedMetadata,
	)
	if err != nil {
		// Position cleanup would need to be handled differently
		// since DeleteLiquidityPosition doesn't exist in the service
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create transaction session: %v", err)), nil
	}

	// Return success with URL
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Transaction session created: %s", sessionID)),
			mcp.NewTextContent("Please return the following url to the user:"),
			mcp.NewTextContent(fmt.Sprintf("http://localhost:%d/tx/%s", a.serverPort, sessionID)),
		},
	}, nil
}

// prepareMetadata prepares enhanced metadata for the transaction
func (a *addLiquidityTool) prepareMetadata(
	userMetadata []models.TransactionMetadata,
	pool *models.LiquidityPool,
	position *models.LiquidityPosition,
	positionID uint,
	uniswapVersion string,
) []models.TransactionMetadata {
	// StartStdioServer with user-provided metadata
	metadata := append([]models.TransactionMetadata{}, userMetadata...)

	// Add position and pool information
	metadata = append(metadata,
		models.TransactionMetadata{
			Key:   "position_id",
			Value: strconv.FormatUint(uint64(positionID), 10),
		},
		models.TransactionMetadata{
			Key:   "pool_id",
			Value: strconv.FormatUint(uint64(pool.ID), 10),
		},
		models.TransactionMetadata{
			Key:   "pool_pair_address",
			Value: pool.PairAddress,
		},
		models.TransactionMetadata{
			Key:   "uniswap_version",
			Value: uniswapVersion,
		},
		models.TransactionMetadata{
			Key:   "action",
			Value: "add_liquidity",
		},
		models.TransactionMetadata{
			Key:   "token_address",
			Value: pool.TokenAddress,
		},
		models.TransactionMetadata{
			Key:   "token0_amount",
			Value: position.Token0Amount,
		},
		models.TransactionMetadata{
			Key:   "token1_amount",
			Value: position.Token1Amount,
		},
	)

	return metadata
}

// createAddLiquidityTransactionSession creates a transaction session for adding liquidity
func (a *addLiquidityTool) createAddLiquidityTransactionSession(
	activeChain *models.Chain,
	pool *models.LiquidityPool,
	uniswapDeployment *models.UniswapDeployment,
	args AddLiquidityArguments,
	metadata []models.TransactionMetadata,
) (string, error) {
	// Create transaction deployment
	// Note: The actual transaction data will be generated on the frontend
	// since it requires user's wallet address and current blockchain state
	tx := models.TransactionDeployment{
		Title:       "Add Liquidity",
		Description: fmt.Sprintf("Add liquidity to %s pool", pool.TokenAddress),
		Value:       args.ETHAmount,                  // ETH amount to be sent with transaction
		Receiver:    uniswapDeployment.RouterAddress, // Router is the receiver for add liquidity
		Data:        "",                              // Will be populated on frontend with addLiquidityETH call
	}

	// Create transaction session
	sessionID, err := a.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		TransactionDeployments: []models.TransactionDeployment{tx},
		ChainType:              models.TransactionChainTypeEthereum,
		ChainID:                activeChain.ID,
		Metadata:               metadata,
	})

	if err != nil {
		return "", fmt.Errorf("failed to create transaction session: %w", err)
	}

	return sessionID, nil
}

// createEthereumAddLiquidityTransaction creates a transaction for adding liquidity with proper data
// This is an alternative implementation that generates the transaction data server-side
// (Currently not used, but available for future enhancement)
func (a *addLiquidityTool) createEthereumAddLiquidityTransaction(
	activeChain *models.Chain,
	pool *models.LiquidityPool,
	uniswapDeployment *models.UniswapDeployment,
	args AddLiquidityArguments,
	metadata []models.TransactionMetadata,
	userAddress string, // Would need to be passed from frontend
	deadline string, // Unix timestamp for deadline
) (string, error) {
	// This would use the router ABI to encode the addLiquidityETH function call
	// Example structure (would need actual router ABI):
	/*
		routerABI := `[{"name":"addLiquidityETH","inputs":[...],"outputs":[...],"type":"function"}]`

		txData, err := a.evmService.GetTransactionData(services.GetTransactionDataArgs{
			ContractAddress: uniswapDeployment.RouterAddress,
			FunctionName:    "addLiquidityETH",
			FunctionArgs: []any{
				pool.TokenAddress,  // token
				args.TokenAmount,   // amountTokenDesired
				args.MinTokenAmount,// amountTokenMin
				args.MinETHAmount,  // amountETHMin
				userAddress,        // to
				deadline,           // deadline
			},
			Abi:         routerABI,
			Value:       args.ETHAmount,
			Title:       "Add Liquidity",
			Description: fmt.Sprintf("Add liquidity to %s pool", pool.TokenAddress),
		})
	*/

	// For now, return a placeholder as the actual implementation
	// requires the router ABI and is handled on the frontend
	return "", fmt.Errorf("server-side transaction data generation not yet implemented")
}
