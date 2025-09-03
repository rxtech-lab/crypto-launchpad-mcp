package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"time"

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
	OwnerAddress       string `json:"owner_address" validate:"required"`

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
		mcp.WithString("owner_address",
			mcp.Required(),
			mcp.Description("Address that will own the liquidity pool tokens and receive them. Ask user to provide this address."),
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
		activeChain, err := c.chainService.GetActiveChain()
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
	// validate the owner address is a valid ethereum address
	if !utils.IsValidEthereumAddress(args.OwnerAddress) {
		return mcp.NewToolResultError("Owner address is not a valid Ethereum address"), nil
	}

	// Get the active Uniswap settings
	user, _ := utils.GetAuthenticatedUser(ctx)
	var userId *string
	if user != nil {
		userId = &user.Sub
	}

	// get active chain
	chain, err := c.chainService.GetActiveChain()
	if err != nil {
		return mcp.NewToolResultError("Unable to get active chain. Is there any chain selected?"), nil
	}
	// Get active Uniswap settings
	uniswapSettings, err := c.uniswapService.GetActiveUniswapDeployment(userId, *chain)
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
		// Check if pool is already confirmed
		if existingPool.Status == models.TransactionStatusConfirmed {
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

	// Create transaction deployments for liquidity pool creation
	transactionDeployments, err := c.createEthereumLiquidityPoolTransactions(
		uniswapDeployment.RouterAddress,
		args.TokenAddress,
		uniswapDeployment.WETHAddress,
		args.InitialTokenAmount,
		args.InitialETHAmount,
		args.OwnerAddress,
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating liquidity pool transactions: %v", err)), nil
	}

	// Add pool ID to metadata
	enhancedMetadata := append(args.Metadata, models.TransactionMetadata{
		Key:   "pool_id",
		Value: strconv.FormatUint(uint64(poolID), 10),
	})

	// Create transaction session with the liquidity pool transactions
	sessionID, err := c.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		TransactionDeployments: transactionDeployments,
		ChainType:              models.TransactionChainTypeEthereum,
		ChainID:                activeChain.ID,
		Metadata:               enhancedMetadata,
		UserID:                 userId,
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

// createEthereumLiquidityPoolTransactions creates the necessary transactions to create an Uniswap liquidity pool on Ethereum.
// It includes approve transactions for both tokens and the add liquidity transaction. Get router abi from embedded files.
// routerAddress is the address of the Uniswap router contract
// token1Address is the address of the first token (the custom token)
// token2Address is the address of the second token (for example, WETH)
// token1Amount is the amount of token1 to add to the pool
// token2Amount is the amount of token2 to add to the pool
// ownerAddress is the address that will receive the liquidity pool tokens
func (c *createLiquidityPoolTool) createEthereumLiquidityPoolTransactions(routerAddress, token1Address, token2Address, token1Amount, token2Amount, ownerAddress string) ([]models.TransactionDeployment, error) {
	// Get Uniswap V2 contracts to extract Router ABI
	v2Contracts, err := utils.FetchUniswapV2Contracts()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Uniswap V2 contracts: %w", err)
	}

	// Get Router ABI
	routerAbi, err := json.Marshal(v2Contracts.Router.ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Router ABI: %w", err)
	}

	// Standard ERC20 ABI for approve function
	erc20ABI := `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

	// MaxUint256 for unlimited approval
	maxUint256 := new(big.Int)
	maxUint256.SetString("115792089237316195423570985008687907853269984665640564039457584007913129639935", 10)

	// Calculate deadline (10 minutes from now)
	deadline := time.Now().Unix() + 600

	var transactionDeployments []models.TransactionDeployment

	// Transaction 1: Approve Token1 (custom token) for Router
	approve1Tx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: token1Address,
		FunctionName:    "approve",
		FunctionArgs:    []any{routerAddress, maxUint256.String()},
		Abi:             erc20ABI,
		Value:           "0",
		Title:           "Approve Token for Router",
		Description:     fmt.Sprintf("Approve unlimited token spending for Uniswap Router at %s", routerAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create token approval transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, approve1Tx)

	// Transaction 2: Approve Token2 (WETH) for Router
	approve2Tx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: token2Address,
		FunctionName:    "approve",
		FunctionArgs:    []any{routerAddress, maxUint256.String()},
		Abi:             erc20ABI,
		Value:           "0",
		Title:           "Approve WETH for Router",
		Description:     fmt.Sprintf("Approve unlimited WETH spending for Uniswap Router at %s", routerAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create WETH approval transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, approve2Tx)

	// Calculate minimum amounts with 1% slippage protection
	minAmount1, minAmount2, err := utils.CalculateMinimumLiquidityAmounts(token1Amount, token2Amount, 1.0)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate minimum amounts with slippage: %w", err)
	}

	// Transaction 3: Add Liquidity
	// addLiquidity(address tokenA, address tokenB, uint amountADesired, uint amountBDesired, uint amountAMin, uint amountBMin, address to, uint deadline)
	addLiquidityTx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: routerAddress,
		FunctionName:    "addLiquidity",
		FunctionArgs: []any{
			token1Address,               // tokenA
			token2Address,               // tokenB
			token1Amount,                // amountADesired
			token2Amount,                // amountBDesired
			minAmount1,                  // amountAMin (1% slippage protection)
			minAmount2,                  // amountBMin (1% slippage protection)
			ownerAddress,                // to (address that will receive the LP tokens)
			fmt.Sprintf("%d", deadline), // deadline
		},
		Abi:             string(routerAbi),
		Value:           "0",
		Title:           "Add Liquidity to Pool",
		Description:     fmt.Sprintf("Add liquidity to the token pair %s/%s", token1Address, token2Address),
		TransactionType: models.TransactionTypeAddLiquidity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create add liquidity transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, addLiquidityTx)

	return transactionDeployments, nil
}
