package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/constants"
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
	Token0Address       string `json:"token0_address" validate:"required"`
	Token1Address       string `json:"token1_address" validate:"required"`
	InitialToken0Amount string `json:"initial_token0_amount" validate:"required"`
	InitialToken1Amount string `json:"initial_token1_amount" validate:"required"`
	OwnerAddress        string `json:"owner_address" validate:"required"`

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
		mcp.WithDescription("Create new Uniswap liquidity pool with signing interface. Supports both ETH-to-Token pairs (using addLiquidityETH) and Token-to-Token pairs (using addLiquidity). Generates a URL where users can connect wallet and sign the pool creation transaction."),
		mcp.WithString("token0_address",
			mcp.Required(),
			mcp.Description(fmt.Sprintf("Address of the first token in the pair. Use %s address for ETH pairs.", services.EthTokenAddress)),
		),
		mcp.WithString("token1_address",
			mcp.Required(),
			mcp.Description(fmt.Sprintf("Address of the second token in the pair. Use %s address for ETH pairs.", services.EthTokenAddress)),
		),
		mcp.WithString("initial_token0_amount",
			mcp.Required(),
			mcp.Description("Initial amount of first token to add to the pool"),
		),
		mcp.WithString("initial_token1_amount",
			mcp.Required(),
			mcp.Description("Initial amount of second token to add to the pool. For ETH pairs, this represents the ETH amount."),
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
	// Validate all addresses are valid ethereum addresses
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

	// Determine pair type: ETH pair or Token pair
	isETHPair := args.Token0Address == services.EthTokenAddress || args.Token1Address == services.EthTokenAddress
	// make sure not all of the token addresses are the same
	if args.Token0Address == args.Token1Address {
		return mcp.NewToolResultError("Token0 and Token1 addresses cannot be the same"), nil
	}

	// Check if pool already exists - for now check based on token0 (could be enhanced to check both tokens)
	existingPool, err := c.liquidityService.GetLiquidityPoolByTokenAddress(args.Token0Address, args.Token1Address)
	if err == nil && existingPool != nil {
		// Check if pool is already confirmed
		if existingPool.Status == models.TransactionStatusConfirmed {
			return mcp.NewToolResultError("Liquidity pool already exists for this token pair"), nil
		}
	}

	// Create liquidity pool record
	// Creator address will be set when wallet connects on the web interface
	_, _, err = utils.CalculateInitialTokenPrice(args.InitialToken0Amount, args.InitialToken1Amount, 18)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error calculating initial price: %v", err)), nil
	}

	// Create transaction deployments for liquidity pool creation based on pair type
	var transactionDeployments []models.TransactionDeployment
	if isETHPair {
		// get the non-ETH token address
		nonEthTokenAddress := args.Token0Address
		nonEthTokenAmount := args.InitialToken0Amount
		ethTokenAmount := args.InitialToken1Amount
		if args.Token0Address == services.EthTokenAddress {
			nonEthTokenAddress = args.Token1Address
			nonEthTokenAmount = args.InitialToken1Amount
			ethTokenAmount = args.InitialToken0Amount
		}
		// ETH pair: use addLiquidityETH
		transactionDeployments, err = c.createETHPairTransactions(
			uniswapDeployment.RouterAddress,
			nonEthTokenAddress,
			nonEthTokenAmount,
			ethTokenAmount,
			args.OwnerAddress,
		)
	} else {
		// Token pair: use addLiquidity
		transactionDeployments, err = c.createTokenPairTransactions(
			uniswapDeployment.RouterAddress,
			args.Token0Address,
			args.Token1Address,
			args.InitialToken0Amount,
			args.InitialToken1Amount,
			args.OwnerAddress,
		)
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating liquidity pool transactions: %v", err)), nil
	}

	enhancedMetadata := append(args.Metadata, models.TransactionMetadata{
		Key:   services.MetadataToken0Address,
		Value: args.Token0Address,
	})

	enhancedMetadata = append(enhancedMetadata, models.TransactionMetadata{
		Key:   services.MetadataToken1Address,
		Value: args.Token1Address,
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

	pool := &models.LiquidityPool{
		TokenAddress:   args.Token0Address, // Use token0 as the primary token address for backward compatibility
		UniswapVersion: uniswapSettings.Version,
		Token0:         args.Token0Address,
		Token1:         args.Token1Address,
		InitialToken0:  args.InitialToken0Amount,
		InitialToken1:  args.InitialToken1Amount,
		CreatorAddress: "",
		Status:         models.TransactionStatusPending,
		SessionId:      sessionID,
	}
	_, err = c.liquidityService.CreateLiquidityPool(pool)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating liquidity pool record: %v", err)), nil
	}

	url, err := utils.GetTransactionSessionUrl(c.serverPort, sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get transaction session url: %v", err)), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Transaction session created: %s", sessionID)),
			mcp.NewTextContent("Please sign the liquidity pool creation transaction in the URL"),
			mcp.NewTextContent(url),
		},
	}, nil
}

// createETHPairTransactions creates transactions for ETH-to-Token liquidity pools using addLiquidityETH
// routerAddress is the address of the Uniswap router contract
// token0Address is the address of the custom token (not WETH)
// token1Address is the WETH address
// token0Amount is the amount of custom token to add to the pool
// token1Amount is the amount of ETH to add to the pool (will be sent as transaction value)
// ownerAddress is the address that will receive the liquidity pool tokens
func (c *createLiquidityPoolTool) createETHPairTransactions(routerAddress, nonEthTokenAddress, nonEthTokenAmount, ethTokenAmount, ownerAddress string) ([]models.TransactionDeployment, error) {
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

	// Calculate deadline (10 minutes from now)
	deadline := time.Now().Unix() + 600

	var transactionDeployments []models.TransactionDeployment

	// Validate addresses before creating transactions
	if !utils.IsValidEthereumAddress(nonEthTokenAddress) {
		return nil, fmt.Errorf("invalid token0 address: %s", nonEthTokenAddress)
	}
	if !utils.IsValidEthereumAddress(routerAddress) {
		return nil, fmt.Errorf("invalid router address: %s", routerAddress)
	}

	// Transaction 1: Approve Token0 (custom token) for Router
	// Note: We don't need to approve WETH for ETH pairs since we send ETH directly
	approveTx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: nonEthTokenAddress,
		FunctionName:    "approve",
		FunctionArgs:    []any{routerAddress, constants.MaxUint256.String()},
		Abi:             erc20ABI,
		Value:           "0",
		Title:           "Approve Token for Router",
		Description:     fmt.Sprintf("Approve unlimited token spending for Uniswap Router at %s", routerAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create token approval transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, approveTx)

	// Calculate minimum amounts with 1% slippage protection
	minTokenAmount, minETHAmount, err := utils.CalculateMinimumLiquidityAmounts(nonEthTokenAmount, ethTokenAmount, 1.0)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate minimum amounts with slippage: %w", err)
	}

	// Transaction 2: Add Liquidity with ETH
	// addLiquidityETH(address token, uint amountTokenDesired, uint amountTokenMin, uint amountETHMin, address to, uint deadline)
	addLiquidityETHTx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: routerAddress,
		FunctionName:    "addLiquidityETH",
		FunctionArgs: []any{
			nonEthTokenAddress,          // token
			nonEthTokenAmount,           // amountTokenDesired
			minTokenAmount,              // amountTokenMin (1% slippage protection)
			minETHAmount,                // amountETHMin (1% slippage protection)
			ownerAddress,                // to (address that will receive the LP tokens)
			fmt.Sprintf("%d", deadline), // deadline
		},
		Abi:             string(routerAbi),
		Value:           ethTokenAmount, // ETH amount to send with transaction
		Title:           "Add Liquidity with ETH",
		Description:     fmt.Sprintf("Add liquidity to the ETH pair %s/ETH", nonEthTokenAddress),
		TransactionType: models.TransactionTypeLiquidityPoolCreation,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create add liquidity transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, addLiquidityETHTx)

	return transactionDeployments, nil
}

// createTokenPairTransactions creates transactions for Token-to-Token liquidity pools using addLiquidity
// routerAddress is the address of the Uniswap router contract
// token0Address is the address of the first token
// token1Address is the address of the second token (not WETH)
// token0Amount is the amount of first token to add to the pool
// token1Amount is the amount of second token to add to the pool
// ownerAddress is the address that will receive the liquidity pool tokens
func (c *createLiquidityPoolTool) createTokenPairTransactions(routerAddress, token0Address, token1Address, token0Amount, token1Amount, ownerAddress string) ([]models.TransactionDeployment, error) {
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

	// Calculate deadline (10 minutes from now)
	deadline := time.Now().Unix() + 600

	// Validate addresses before creating transactions
	if !utils.IsValidEthereumAddress(token0Address) {
		return nil, fmt.Errorf("invalid token0 address: %s", token0Address)
	}
	if !utils.IsValidEthereumAddress(token1Address) {
		return nil, fmt.Errorf("invalid token1 address: %s", token1Address)
	}
	if !utils.IsValidEthereumAddress(routerAddress) {
		return nil, fmt.Errorf("invalid router address: %s", routerAddress)
	}

	var transactionDeployments []models.TransactionDeployment

	// Transaction 1: Approve Token0 for Router
	approve0Tx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: token0Address,
		FunctionName:    "approve",
		FunctionArgs:    []any{routerAddress, constants.MaxUint256.String()},
		Abi:             erc20ABI,
		Value:           "0",
		Title:           "Approve First Token for Router",
		Description:     fmt.Sprintf("Approve unlimited first token spending for Uniswap Router at %s", routerAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create first token approval transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, approve0Tx)

	// Transaction 2: Approve Token1 for Router
	functionArgs := []any{routerAddress, constants.MaxUint256.String()}
	approve1Tx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: token1Address,
		FunctionName:    "approve",
		FunctionArgs:    functionArgs,
		Abi:             erc20ABI,
		Value:           "0",
		Title:           "Approve Second Token for Router",
		Description:     fmt.Sprintf("Approve unlimited second token spending for Uniswap Router at %s", routerAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create second token approval transaction: %w", err)
	}
	functionArgsString, err := utils.EncodeFunctionArgsToStringMapWithStringABI("approve", functionArgs, erc20ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw contract arguments: %w", err)
	}

	approve1Tx.ShowBalanceBeforeDeployment = true
	approve1Tx.ContractAddress = &token1Address
	approve1Tx.RawContractArguments = &functionArgsString
	transactionDeployments = append(transactionDeployments, approve1Tx)

	// Calculate minimum amounts with 1% slippage protection
	minAmount0, minAmount1, err := utils.CalculateMinimumLiquidityAmounts(token0Amount, token1Amount, 1.0)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate minimum amounts with slippage: %w", err)
	}

	// Transaction 3: Add Liquidity for Token Pair
	// addLiquidity(address tokenA, address tokenB, uint amountADesired, uint amountBDesired, uint amountAMin, uint amountBMin, address to, uint deadline)
	addLiquidityFunctionArgs := []any{
		token0Address,               // tokenA
		token1Address,               // tokenB
		token0Amount,                // amountADesired
		token1Amount,                // amountBDesired
		minAmount0,                  // amountAMin (1% slippage protection)
		minAmount1,                  // amountBMin (1% slippage protection)
		ownerAddress,                // to (address that will receive the LP tokens)
		fmt.Sprintf("%d", deadline), // deadline
	}
	addLiquidityTx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: routerAddress,
		FunctionName:    "addLiquidity",
		FunctionArgs:    addLiquidityFunctionArgs,
		Abi:             string(routerAbi),
		Value:           "0", // No ETH value for token-to-token pairs
		Title:           "Add Token Liquidity to Pool",
		Description:     fmt.Sprintf("Add liquidity to the token pair %s/%s", token0Address, token1Address),
		TransactionType: models.TransactionTypeAddLiquidity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create add liquidity transaction: %w", err)
	}
	addLiquidityFunctionArgsString, err := utils.EncodeFunctionArgsToStringMapWithStringABI("addLiquidity", addLiquidityFunctionArgs, string(routerAbi))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw contract arguments: %w", err)
	}
	addLiquidityTx.RawContractArguments = &addLiquidityFunctionArgsString
	transactionDeployments = append(transactionDeployments, addLiquidityTx)

	return transactionDeployments, nil
}
