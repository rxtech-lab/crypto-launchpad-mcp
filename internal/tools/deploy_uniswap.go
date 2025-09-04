package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type deployUniswapTool struct {
	chainService   services.ChainService
	evmService     services.EvmService
	txService      services.TransactionService
	uniswapService services.UniswapService
	serverPort     int
}

type DeployUniswapArguments struct {
	// Required fields
	Version string `json:"version" validate:"required"`

	// Optional fields
	DeployRouter *bool                        `json:"deploy_router,omitempty"`
	Metadata     []models.TransactionMetadata `json:"metadata,omitempty"`
}

func NewDeployUniswapTool(chainService services.ChainService, serverPort int, evmService services.EvmService, txService services.TransactionService, uniswapService services.UniswapService) *deployUniswapTool {
	return &deployUniswapTool{
		chainService:   chainService,
		evmService:     evmService,
		txService:      txService,
		uniswapService: uniswapService,
		serverPort:     serverPort,
	}
}

func (d *deployUniswapTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("deploy_uniswap",
		mcp.WithDescription("Deploy Uniswap infrastructure contracts (factory, router, WETH) with version selection. Creates a transaction session and returns a URL where users can sign and deploy the contracts."),
		mcp.WithString("version",
			mcp.Required(),
			mcp.Description("Uniswap version to deploy (v2 only currently supported)"),
		),
		mcp.WithBoolean("deploy_router",
			mcp.Description("Whether to deploy the router contract. If false, only factory and WETH will be deployed. Otherwise, only router will be deployed. However, it will check if factory and WETH are already deployed. Call this tool with deploy_router=false first, then call it with deploy_router=true to deploy the router."),
		),
		mcp.WithArray("metadata",
			mcp.Description("JSON array of metadata for the transaction (e.g., [{\"key\": \"Deploy Type\", \"value\": \"Uniswap V2\"}]). Optional."),
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

func (d *deployUniswapTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args DeployUniswapArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Validate version
		if err := utils.ValidateUniswapVersion(args.Version); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		user, _ := utils.GetAuthenticatedUser(ctx)
		var userId *string
		if user != nil {
			userId = &user.Sub
		}

		// Get active chain configuration
		activeChain, err := d.chainService.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Validate that only Ethereum is supported
		if activeChain.ChainType != models.TransactionChainTypeEthereum {
			return mcp.NewToolResultError(fmt.Sprintf("Uniswap deployment is only supported on Ethereum, got %s", activeChain.ChainType)), nil
		}

		// Check deployment status and validate deploy_router flag logic
		existingDeployment, err := d.uniswapService.GetUniswapDeploymentByChain(activeChain.ID)

		if deployRouter := args.DeployRouter; deployRouter != nil && *deployRouter {
			// Router deployment requested
			if err != nil || existingDeployment == nil {
				return mcp.NewToolResultError("Cannot deploy router: No existing Uniswap deployment found. Please deploy infrastructure first (deploy_router=false)"), nil
			}
			if existingDeployment.RouterAddress != "" {
				return mcp.NewToolResultError(fmt.Sprintf("Router is already deployed at address: %s", existingDeployment.RouterAddress)), nil
			}
		} else {
			// Infrastructure deployment (WETH + Factory)
			if err == nil && existingDeployment != nil {
				if existingDeployment.WETHAddress != "" && existingDeployment.FactoryAddress != "" {
					return mcp.NewToolResultError(fmt.Sprintf("Uniswap %s infrastructure is already deployed on %s (WETH: %s, Factory: %s)",
						existingDeployment.Version, string(activeChain.ChainType), existingDeployment.WETHAddress, existingDeployment.FactoryAddress)), nil
				}
			}
		}

		switch args.Version {
		case "v2":
			return d.createUniswapV2DeploymentSession(activeChain, args.DeployRouter, args.Metadata, userId)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Unsupported version: %s", args.Version)), nil
		}
	}
}

// createUniswapV2DeploymentSession creates a transaction session for Uniswap V2 deployment
func (d *deployUniswapTool) createUniswapV2DeploymentSession(activeChain *models.Chain, deployRouter *bool, metadata []models.TransactionMetadata, userId *string) (*mcp.CallToolResult, error) {
	// Get or create Uniswap deployment record
	existingDeployment, err := d.uniswapService.GetUniswapDeploymentByChain(activeChain.ID)
	var uniswapDeployment *models.UniswapDeployment

	if err != nil || existingDeployment == nil {
		// Create new deployment record
		uniswapDeployment = &models.UniswapDeployment{
			Version: "v2",
			Status:  "pending",
		}
		createdDeploymentId, createErr := d.uniswapService.CreateUniswapDeployment(activeChain.ID, "v2", userId)
		if createErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating Uniswap deployment record: %v", createErr)), nil
		}
		uniswapDeployment.ID = createdDeploymentId
	} else {
		uniswapDeployment = existingDeployment
	}

	// Get Uniswap V2 contracts
	v2Contracts, err := utils.FetchUniswapV2Contracts()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch Uniswap V2 contracts: %v", err)), nil
	}

	// Prepare transaction deployments based on deploy_router flag
	var transactionDeployments []models.TransactionDeployment

	if deployRouter == nil || !*deployRouter {
		// Deploy WETH9 and Factory (infrastructure contracts)
		wethTx, err := d.createWETH9Deployment(v2Contracts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to prepare WETH9 deployment: %v", err)), nil
		}
		transactionDeployments = append(transactionDeployments, wethTx)

		factoryTx, err := d.createFactoryDeployment(v2Contracts)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to prepare Factory deployment: %v", err)), nil
		}
		transactionDeployments = append(transactionDeployments, factoryTx)
	} else {
		// Deploy only Router (requires existing WETH and Factory addresses)
		if uniswapDeployment.WETHAddress == "" || uniswapDeployment.FactoryAddress == "" {
			return mcp.NewToolResultError("Cannot deploy router: WETH and Factory addresses not found. Please deploy infrastructure first (deploy_router=false)"), nil
		}

		routerTx, err := d.createRouterDeployment(v2Contracts, uniswapDeployment.FactoryAddress, uniswapDeployment.WETHAddress)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to prepare Router deployment: %v", err)), nil
		}
		transactionDeployments = append(transactionDeployments, routerTx)
	}

	// Add deployment ID to metadata
	enhancedMetadata := append(metadata, models.TransactionMetadata{
		Key:   "uniswap_deployment_id",
		Value: strconv.FormatUint(uint64(uniswapDeployment.ID), 10),
	})

	// Create transaction session
	sessionID, err := d.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		TransactionDeployments: transactionDeployments,
		ChainType:              models.TransactionChainTypeEthereum,
		ChainID:                activeChain.ID,
		Metadata:               enhancedMetadata,
		UserID:                 userId,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create transaction session: %v", err)), nil
	}

	deploymentType := "WETH9 and Factory"
	if deployRouter != nil && *deployRouter {
		deploymentType = "Router"
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Transaction session created: %s", sessionID)),
			mcp.NewTextContent(fmt.Sprintf("Please sign the Uniswap V2 %s deployment transactions in the URL", deploymentType)),
			mcp.NewTextContent(fmt.Sprintf("http://localhost:%d/tx/%s", d.serverPort, sessionID)),
		},
	}, nil
}

// createWETH9Deployment creates a transaction deployment for WETH9 contract
func (d *deployUniswapTool) createWETH9Deployment(v2Contracts *utils.UniswapV2Contracts) (models.TransactionDeployment, error) {
	wethAbi, _ := json.Marshal(v2Contracts.WETH9.ABI)
	tx, abiData, err := d.evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
		Abi:             string(wethAbi),
		Bytecode:        v2Contracts.WETH9.Bytecode,
		ConstructorArgs: []any{}, // WETH9 has no constructor args
		Value:           "0",
		Title:           "Deploy WETH9",
		Description:     "Deploy Wrapped Ether (WETH9) contract for Uniswap V2",
		Receiver:        "", // Empty for contract deployment
		TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
	})

	functionArgs, err := utils.EncodeFunctionArgsToStringMap("constructor", []any{}, abiData)
	tx.RawContractArguments = &functionArgs
	return tx, err
}

// createFactoryDeployment creates a transaction deployment for UniswapV2Factory contract
func (d *deployUniswapTool) createFactoryDeployment(v2Contracts *utils.UniswapV2Contracts) (models.TransactionDeployment, error) {
	factoryAbi, _ := json.Marshal(v2Contracts.Factory.ABI)
	args := []any{"0x0000000000000000000000000000000000000000"}
	// Factory constructor requires feeToSetter address (use zero address as placeholder)
	tx, abiData, err := d.evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
		Abi:             string(factoryAbi),
		Bytecode:        v2Contracts.Factory.Bytecode,
		ConstructorArgs: args, // feeToSetter address (placeholder)
		Value:           "0",
		Title:           "Deploy UniswapV2Factory",
		Description:     "Deploy Uniswap V2 Factory contract",
		Receiver:        "", // Empty for contract deployment
		TransactionType: models.TransactionTypeUniswapV2FactoryDeployment,
	})

	functionArgs, err := utils.EncodeFunctionArgsToStringMap("constructor", args, abiData)
	tx.RawContractArguments = &functionArgs
	return tx, err
}

// createRouterDeployment creates a transaction deployment for UniswapV2Router02 contract
func (d *deployUniswapTool) createRouterDeployment(v2Contracts *utils.UniswapV2Contracts, factoryAddress, wethAddress string) (models.TransactionDeployment, error) {
	routerAbi, _ := json.Marshal(v2Contracts.Router.ABI)
	args := []any{factoryAddress, wethAddress}
	// Router constructor requires factory and WETH addresses
	tx, abiData, err := d.evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
		Abi:             string(routerAbi),
		Bytecode:        v2Contracts.Router.Bytecode,
		ConstructorArgs: args, // Use actual deployed addresses
		Value:           "0",
		Title:           "Deploy UniswapV2Router02",
		Description:     "Deploy Uniswap V2 Router contract",
		Receiver:        "", // Empty for contract deployment
		TransactionType: models.TransactionTypeUniswapV2RouterDeployment,
	})

	functionArgs, err := utils.EncodeFunctionArgsToStringMap("constructor", args, abiData)
	tx.RawContractArguments = &functionArgs
	return tx, err
}
