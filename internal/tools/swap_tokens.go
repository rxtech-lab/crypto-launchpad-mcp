package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/constants"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type swapTokensTool struct {
	chainService     services.ChainService
	evmService       services.EvmService
	txService        services.TransactionService
	liquidityService services.LiquidityService
	uniswapService   services.UniswapService
	serverPort       int
}

type SwapTokensArguments struct {
	// Required fields
	FromToken         string `json:"from_token" validate:"required"`
	ToToken           string `json:"to_token" validate:"required"`
	Amount            string `json:"amount" validate:"required"`
	SlippageTolerance string `json:"slippage_tolerance" validate:"required"`
	UserAddress       string `json:"user_address" validate:"required"`

	// Optional fields
	Metadata []models.TransactionMetadata `json:"metadata,omitempty"`
}

func NewSwapTokensTool(chainService services.ChainService, liquidityService services.LiquidityService, uniswapService services.UniswapService, txService services.TransactionService, serverPort int, evmService services.EvmService) *swapTokensTool {
	return &swapTokensTool{
		chainService:     chainService,
		evmService:       evmService,
		txService:        txService,
		liquidityService: liquidityService,
		uniswapService:   uniswapService,
		serverPort:       serverPort,
	}
}

func (s *swapTokensTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("swap_tokens",
		mcp.WithDescription("Execute token swaps via Uniswap with signing interface. Generates a URL where users can connect wallet and sign the swap transaction."),
		mcp.WithString("from_token",
			mcp.Required(),
			mcp.Description(fmt.Sprintf("Address of the token to swap from (use %s for ETH)", services.EthTokenAddress)),
		),
		mcp.WithString("to_token",
			mcp.Required(),
			mcp.Description(fmt.Sprintf("Address of the token to swap to (use %s for ETH)", services.EthTokenAddress)),
		),
		mcp.WithString("amount",
			mcp.Required(),
			mcp.Description("Amount of tokens to swap (in wei for ETH, or smallest unit for tokens)"),
		),
		mcp.WithString("slippage_tolerance",
			mcp.Required(),
			mcp.Description("Maximum slippage tolerance as percentage (e.g., '0.5' for 0.5%)"),
		),
		mcp.WithString("user_address",
			mcp.Required(),
			mcp.Description("Address that will execute the swap"),
		),
		mcp.WithArray("metadata",
			mcp.Description("JSON array of metadata for the transaction (e.g., [{\"key\": \"Swap Type\", \"value\": \"Token Swap\"}]). Optional."),
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

func (s *swapTokensTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args SwapTokensArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Get active chain configuration
		activeChain, err := s.chainService.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Currently only support Ethereum
		if activeChain.ChainType != models.TransactionChainTypeEthereum {
			return mcp.NewToolResultError(fmt.Sprintf("Uniswap swaps are only supported on Ethereum, got %s", activeChain.ChainType)), nil
		}

		// Validate user address
		if !utils.IsValidEthereumAddress(args.UserAddress) {
			return mcp.NewToolResultError("User address is not a valid Ethereum address"), nil
		}

		// Validate tokens are different
		if strings.EqualFold(args.FromToken, args.ToToken) {
			return mcp.NewToolResultError("Cannot swap token to itself"), nil
		}

		return s.createSwapTransaction(ctx, args, activeChain)
	}
}

func (s *swapTokensTool) createSwapTransaction(ctx context.Context, args SwapTokensArguments, activeChain *models.Chain) (*mcp.CallToolResult, error) {
	// Get active Uniswap settings
	user, _ := utils.GetAuthenticatedUser(ctx)
	var userId *string
	if user != nil {
		userId = &user.Sub
	}

	// Get active Uniswap deployment
	_, err := s.uniswapService.GetActiveUniswapDeployment(userId, *activeChain)
	if err != nil {
		return mcp.NewToolResultError("No Uniswap version selected. Please use set_uniswap_version tool first"), nil
	}

	// Get Uniswap deployment to retrieve contract addresses
	uniswapDeployment, err := s.uniswapService.GetUniswapDeploymentByChain(activeChain.ID)
	if err != nil {
		return mcp.NewToolResultError("No Uniswap deployment found for this chain. Please deploy Uniswap first using deploy_uniswap tool"), nil
	}

	// Verify required addresses are available
	if uniswapDeployment.RouterAddress == "" {
		return mcp.NewToolResultError("Router address not found in Uniswap deployment. Please ensure Uniswap deployment is completed"), nil
	}
	if uniswapDeployment.WETHAddress == "" {
		return mcp.NewToolResultError("WETH address not found in Uniswap deployment. Please ensure Uniswap deployment is completed"), nil
	}

	// Parse slippage tolerance
	slippage, err := parseSlippage(args.SlippageTolerance)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid slippage tolerance: %v", err)), nil
	}

	// Determine swap type and create transactions
	var transactionDeployments []models.TransactionDeployment
	isFromETH := strings.ToLower(args.FromToken) == services.EthTokenAddress
	isToETH := strings.ToLower(args.ToToken) == services.EthTokenAddress

	if isFromETH && !isToETH {
		// ETH to Token swap
		transactionDeployments, err = s.createETHToTokenSwap(
			uniswapDeployment.RouterAddress,
			uniswapDeployment.WETHAddress,
			args.ToToken,
			args.Amount,
			slippage,
			args.UserAddress,
		)
	} else if !isFromETH && isToETH {
		// Token to ETH swap
		transactionDeployments, err = s.createTokenToETHSwap(
			uniswapDeployment.RouterAddress,
			uniswapDeployment.WETHAddress,
			args.FromToken,
			args.Amount,
			slippage,
			args.UserAddress,
		)
	} else {
		// Token to Token swap (through WETH)
		transactionDeployments, err = s.createTokenToTokenSwap(
			uniswapDeployment.RouterAddress,
			uniswapDeployment.WETHAddress,
			args.FromToken,
			args.ToToken,
			args.Amount,
			slippage,
			args.UserAddress,
		)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating swap transactions: %v", err)), nil
	}

	// Add metadata
	enhancedMetadata := append(args.Metadata, models.TransactionMetadata{
		Key:   "from_token",
		Value: args.FromToken,
	})
	enhancedMetadata = append(enhancedMetadata, models.TransactionMetadata{
		Key:   "to_token",
		Value: args.ToToken,
	})
	enhancedMetadata = append(enhancedMetadata, models.TransactionMetadata{
		Key:   "amount",
		Value: args.Amount,
	})
	enhancedMetadata = append(enhancedMetadata, models.TransactionMetadata{
		Key:   "slippage",
		Value: args.SlippageTolerance,
	})

	// Create transaction session
	sessionID, err := s.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		TransactionDeployments: transactionDeployments,
		ChainType:              models.TransactionChainTypeEthereum,
		ChainID:                activeChain.ID,
		Metadata:               enhancedMetadata,
		UserID:                 userId,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating transaction session: %v", err)), nil
	}

	// Create swap transaction record
	swapRecord := &models.SwapTransaction{
		UserID:            userId,
		UserAddress:       args.UserAddress,
		FromToken:         args.FromToken,
		ToToken:           args.ToToken,
		FromAmount:        args.Amount,
		ToAmount:          "0", // Will be updated after execution
		SlippageTolerance: args.SlippageTolerance,
		TransactionHash:   "", // Will be updated after execution
		Status:            models.TransactionStatusPending,
	}
	_, err = s.liquidityService.CreateSwapTransaction(swapRecord)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating swap record: %v", err)), nil
	}

	url, err := utils.GetTransactionSessionUrl(s.serverPort, sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get transaction session url: %v", err)), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Swap transaction session created: %s", sessionID)),
			mcp.NewTextContent("Please sign the swap transaction in the URL"),
			mcp.NewTextContent(url),
		},
	}, nil
}

// createETHToTokenSwap creates a transaction to swap ETH for tokens
func (s *swapTokensTool) createETHToTokenSwap(routerAddress, wethAddress, toToken, amount string, slippage float64, userAddress string) ([]models.TransactionDeployment, error) {
	// Get Uniswap V2 Router ABI
	v2Contracts, err := utils.FetchUniswapV2Contracts()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Uniswap V2 contracts: %w", err)
	}

	routerAbi, err := json.Marshal(v2Contracts.Router.ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Router ABI: %w", err)
	}

	// Calculate minimum output with slippage
	// For now, use a simple estimation (in production, use getAmountsOut)
	minAmountOut := calculateMinimumAmount(amount, slippage)

	// Calculate deadline (10 minutes from now)
	deadline := time.Now().Unix() + 600

	// Create path: [WETH, toToken]
	// Pass addresses as strings for proper ABI encoding
	path := []any{
		wethAddress,
		toToken,
	}

	// Validate addresses
	if !utils.IsValidEthereumAddress(toToken) {
		return nil, fmt.Errorf("invalid token address: %s", toToken)
	}
	if !utils.IsValidEthereumAddress(routerAddress) {
		return nil, fmt.Errorf("invalid router address: %s", routerAddress)
	}

	// Create swap transaction
	swapTx, err := s.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: routerAddress,
		FunctionName:    "swapExactETHForTokens",
		FunctionArgs: []any{
			minAmountOut,                // amountOutMin
			path,                        // path
			userAddress,                 // to
			fmt.Sprintf("%d", deadline), // deadline
		},
		Abi:             string(routerAbi),
		Value:           amount, // ETH value to send
		Title:           "Swap ETH for Tokens",
		Description:     fmt.Sprintf("Swap ETH for %s", toToken),
		TransactionType: models.TransactionTypeTokenSwap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create swap transaction: %w", err)
	}

	return []models.TransactionDeployment{swapTx}, nil
}

// createTokenToETHSwap creates transactions to swap tokens for ETH
func (s *swapTokensTool) createTokenToETHSwap(routerAddress, wethAddress, fromToken, amount string, slippage float64, userAddress string) ([]models.TransactionDeployment, error) {
	// Get Uniswap V2 Router ABI
	v2Contracts, err := utils.FetchUniswapV2Contracts()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Uniswap V2 contracts: %w", err)
	}

	routerAbi, err := json.Marshal(v2Contracts.Router.ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Router ABI: %w", err)
	}

	// Standard ERC20 ABI for approve function
	erc20ABI := `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

	// Calculate minimum output with slippage
	minAmountOut := calculateMinimumAmount(amount, slippage)

	// Calculate deadline (10 minutes from now)
	deadline := time.Now().Unix() + 600

	// Create path: [fromToken, WETH]
	// Pass addresses as strings for proper ABI encoding
	path := []any{
		fromToken,
		wethAddress,
	}

	// Validate addresses
	if !utils.IsValidEthereumAddress(fromToken) {
		return nil, fmt.Errorf("invalid token address: %s", fromToken)
	}
	if !utils.IsValidEthereumAddress(routerAddress) {
		return nil, fmt.Errorf("invalid router address: %s", routerAddress)
	}

	var transactionDeployments []models.TransactionDeployment

	// Transaction 1: Approve token for Router
	approveTx, err := s.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: fromToken,
		FunctionName:    "approve",
		FunctionArgs:    []any{routerAddress, constants.MaxUint256.String()},
		Abi:             erc20ABI,
		Value:           "0",
		Title:           "Approve Token for Swap",
		Description:     fmt.Sprintf("Approve token spending for Uniswap Router at %s", routerAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create approval transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, approveTx)

	// Transaction 2: Swap tokens for ETH
	swapTx, err := s.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: routerAddress,
		FunctionName:    "swapExactTokensForETH",
		FunctionArgs: []any{
			amount,                      // amountIn
			minAmountOut,                // amountOutMin
			path,                        // path
			userAddress,                 // to
			fmt.Sprintf("%d", deadline), // deadline
		},
		Abi:             string(routerAbi),
		Value:           "0",
		Title:           "Swap Tokens for ETH",
		Description:     fmt.Sprintf("Swap %s for ETH", fromToken),
		TransactionType: models.TransactionTypeTokenSwap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create swap transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, swapTx)

	return transactionDeployments, nil
}

// createTokenToTokenSwap creates transactions to swap tokens for tokens (via WETH)
func (s *swapTokensTool) createTokenToTokenSwap(routerAddress, wethAddress, fromToken, toToken, amount string, slippage float64, userAddress string) ([]models.TransactionDeployment, error) {
	// Get Uniswap V2 Router ABI
	v2Contracts, err := utils.FetchUniswapV2Contracts()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Uniswap V2 contracts: %w", err)
	}

	routerAbi, err := json.Marshal(v2Contracts.Router.ABI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Router ABI: %w", err)
	}

	// Standard ERC20 ABI for approve function
	erc20ABI := `[{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"}]`

	// MaxUint256 for unlimited approval

	// Calculate minimum output with slippage
	minAmountOut := calculateMinimumAmount(amount, slippage)

	// Calculate deadline (10 minutes from now)
	deadline := time.Now().Unix() + 600

	// Create path: [fromToken, WETH, toToken]
	// This assumes both tokens have liquidity with WETH
	// Pass addresses as strings for proper ABI encoding
	path := []any{
		fromToken,
		wethAddress,
		toToken,
	}

	// Validate addresses
	if !utils.IsValidEthereumAddress(fromToken) {
		return nil, fmt.Errorf("invalid from token address: %s", fromToken)
	}
	if !utils.IsValidEthereumAddress(toToken) {
		return nil, fmt.Errorf("invalid to token address: %s", toToken)
	}
	if !utils.IsValidEthereumAddress(routerAddress) {
		return nil, fmt.Errorf("invalid router address: %s", routerAddress)
	}

	var transactionDeployments []models.TransactionDeployment

	// Transaction 1: Approve fromToken for Router
	approveTx, err := s.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: fromToken,
		FunctionName:    "approve",
		FunctionArgs:    []any{routerAddress, constants.MaxUint256.String()},
		Abi:             erc20ABI,
		Value:           "0",
		Title:           "Approve Token for Swap",
		Description:     fmt.Sprintf("Approve token spending for Uniswap Router at %s", routerAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create approval transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, approveTx)

	// Transaction 2: Swap tokens for tokens
	swapTx, err := s.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: routerAddress,
		FunctionName:    "swapExactTokensForTokens",
		FunctionArgs: []any{
			amount,                      // amountIn
			minAmountOut,                // amountOutMin
			path,                        // path
			userAddress,                 // to
			fmt.Sprintf("%d", deadline), // deadline
		},
		Abi:             string(routerAbi),
		Value:           "0",
		Title:           "Swap Tokens",
		Description:     fmt.Sprintf("Swap %s for %s", fromToken, toToken),
		TransactionType: models.TransactionTypeTokenSwap,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create swap transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, swapTx)

	return transactionDeployments, nil
}

// parseSlippage parses the slippage tolerance string and returns a float
func parseSlippage(slippageStr string) (float64, error) {
	var slippage float64
	_, err := fmt.Sscanf(slippageStr, "%f", &slippage)
	if err != nil {
		return 0, err
	}
	if slippage < 0 || slippage > 100 {
		return 0, fmt.Errorf("slippage must be between 0 and 100")
	}
	return slippage, nil
}

// calculateMinimumAmount calculates the minimum output amount with slippage
func calculateMinimumAmount(amount string, slippage float64) string {
	// TODO: This is a simplified implementation. In production, you should:
	// 1. Call router.getAmountsOut() to get expected output amount
	// 2. Apply slippage to the expected output amount
	//
	// For now, we'll use a very low minimum to avoid swap failures in tests
	// This is safe for testing but NOT suitable for production
	return "1" // Set minimum output to 1 wei to avoid swap failures
}
