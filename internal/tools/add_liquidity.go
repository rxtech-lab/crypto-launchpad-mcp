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

type addLiquidityTool struct {
	chainService     services.ChainService
	evmService       services.EvmService
	txService        services.TransactionService
	liquidityService services.LiquidityService
	uniswapService   services.UniswapService
	serverPort       int
}

type AddLiquidityArguments struct {
	// Required fields
	TokenAddress   string `json:"token_address" validate:"required"`
	TokenAmount    string `json:"token_amount" validate:"required"`
	ETHAmount      string `json:"eth_amount" validate:"required"`
	MinTokenAmount string `json:"min_token_amount" validate:"required"`
	MinETHAmount   string `json:"min_eth_amount" validate:"required"`
	OwnerAddress   string `json:"owner_address" validate:"required"`

	// Optional fields
	Metadata []models.TransactionMetadata `json:"metadata,omitempty"`
}

func NewAddLiquidityTool(chainService services.ChainService, serverPort int, evmService services.EvmService, txService services.TransactionService, liquidityService services.LiquidityService, uniswapService services.UniswapService) *addLiquidityTool {
	return &addLiquidityTool{
		chainService:     chainService,
		evmService:       evmService,
		txService:        txService,
		liquidityService: liquidityService,
		uniswapService:   uniswapService,
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
		mcp.WithString("owner_address",
			mcp.Required(),
			mcp.Description("Address that will receive the liquidity pool tokens. Ask user to provide this address."),
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
	pool, err := a.liquidityService.GetLiquidityPoolByTokenAddress(args.TokenAddress, "")
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

	// Get the active Uniswap settings
	user, _ := utils.GetAuthenticatedUser(ctx)
	var userId *string
	if user != nil {
		userId = &user.Sub
	}

	// get active chain
	chain, err := a.chainService.GetActiveChain()
	if err != nil {
		return mcp.NewToolResultError("Unable to get active chain. Is there any chain selected?"), nil
	}

	// Verify Uniswap settings exist
	_, err = a.uniswapService.GetActiveUniswapDeployment(userId, *chain)
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

	// Prepare enhanced metadata
	enhancedMetadata := a.prepareMetadata(args.Metadata, pool)

	// Create transaction deployments for adding liquidity
	transactionDeployments, err := a.createEthereumAddLiquidityTransactions(
		uniswapDeployment.RouterAddress,
		pool.TokenAddress,
		uniswapDeployment.WETHAddress,
		args.TokenAmount,
		args.ETHAmount,
		args.MinTokenAmount,
		args.MinETHAmount,
		args.OwnerAddress,
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating add liquidity transactions: %v", err)), nil
	}

	// Create transaction session with the add liquidity transactions
	sessionID, err := a.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		TransactionDeployments: transactionDeployments,
		ChainType:              models.TransactionChainTypeEthereum,
		ChainID:                activeChain.ID,
		Metadata:               enhancedMetadata,
		UserID:                 userId,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating transaction session: %v", err)), nil
	}

	// Return success with URL
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Transaction session created: %s", sessionID)),
			mcp.NewTextContent("Please sign the add liquidity transactions in the URL"),
			mcp.NewTextContent(fmt.Sprintf("http://localhost:%d/tx/%s", a.serverPort, sessionID)),
		},
	}, nil
}

// prepareMetadata prepares enhanced metadata for the transaction
func (a *addLiquidityTool) prepareMetadata(
	userMetadata []models.TransactionMetadata,
	pool *models.LiquidityPool,
) []models.TransactionMetadata {
	// StartStdioServer with user-provided metadata
	metadata := append([]models.TransactionMetadata{}, userMetadata...)

	// Add position and pool information
	metadata = append(metadata,
		models.TransactionMetadata{
			Key:   "pool_id",
			Value: strconv.FormatUint(uint64(pool.ID), 10),
		},
		models.TransactionMetadata{
			Key:   "pool_pair_address",
			Value: pool.PairAddress,
		},
		models.TransactionMetadata{
			Key:   "action",
			Value: "add_liquidity",
		},
		models.TransactionMetadata{
			Key:   "token_address",
			Value: pool.TokenAddress,
		},
	)

	return metadata
}

// createEthereumAddLiquidityTransactions creates the necessary transactions to add liquidity to an existing Uniswap pool.
// It includes approve transactions for both tokens and the add liquidity transaction.
// routerAddress is the address of the Uniswap router contract
// tokenAddress is the address of the token in the pool
// wethAddress is the address of WETH
// tokenAmount is the amount of tokens to add to the pool
// ethAmount is the amount of ETH to add to the pool (converted to WETH)
// minTokenAmount is the minimum amount of tokens (slippage protection)
// minETHAmount is the minimum amount of ETH (slippage protection)
// ownerAddress is the address that will receive the liquidity pool tokens
func (a *addLiquidityTool) createEthereumAddLiquidityTransactions(routerAddress, tokenAddress, wethAddress, tokenAmount, ethAmount, minTokenAmount, minETHAmount, ownerAddress string) ([]models.TransactionDeployment, error) {
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

	// Transaction 1: Approve Token for Router
	approveTokenTx, err := a.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: tokenAddress,
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
	transactionDeployments = append(transactionDeployments, approveTokenTx)

	// Transaction 2: Add Liquidity ETH (this handles WETH conversion internally)
	// addLiquidityETH(address token, uint amountTokenDesired, uint amountTokenMin, uint amountETHMin, address to, uint deadline) payable
	addLiquidityETHTx, err := a.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: routerAddress,
		FunctionName:    "addLiquidityETH",
		FunctionArgs: []any{
			tokenAddress,                // token
			tokenAmount,                 // amountTokenDesired
			minTokenAmount,              // amountTokenMin (slippage protection)
			minETHAmount,                // amountETHMin (slippage protection)
			ownerAddress,                // to (address that will receive the LP tokens)
			fmt.Sprintf("%d", deadline), // deadline
		},
		Abi:             string(routerAbi),
		Value:           ethAmount, // ETH amount to send with transaction
		Title:           "Add Liquidity to Pool",
		Description:     fmt.Sprintf("Add liquidity to the token/%s pool", "ETH"),
		TransactionType: models.TransactionTypeAddLiquidity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create add liquidity transaction: %w", err)
	}
	transactionDeployments = append(transactionDeployments, addLiquidityETHTx)

	return transactionDeployments, nil
}
