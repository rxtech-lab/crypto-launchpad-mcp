package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type callFunctionTool struct {
	templateService   services.TemplateService
	evmService        services.EvmService
	txService         services.TransactionService
	chainService      services.ChainService
	deploymentService services.DeploymentService
	serverPort        int
}

type CallFunctionArguments struct {
	// Required fields
	DeploymentID string `json:"deployment_id" validate:"required"`
	FunctionName string `json:"function_name" validate:"required"`

	// Optional fields
	FunctionArgs []any                        `json:"function_args,omitempty"`
	Value        string                       `json:"value,omitempty"`
	Metadata     []models.TransactionMetadata `json:"metadata,omitempty"`
}

type CallFunctionResult struct {
	DeploymentID    string `json:"deployment_id"`
	ContractAddress string `json:"contract_address"`
	FunctionName    string `json:"function_name"`
	Result          string `json:"result,omitempty"`
	Success         bool   `json:"success"`
	SessionID       string `json:"session_id,omitempty"`
	IsReadOnly      bool   `json:"is_read_only"`
}

func NewCallFunctionTool(templateService services.TemplateService, evmService services.EvmService, txService services.TransactionService, chainService services.ChainService, deploymentService services.DeploymentService, serverPort int) *callFunctionTool {
	return &callFunctionTool{
		templateService:   templateService,
		evmService:        evmService,
		txService:         txService,
		chainService:      chainService,
		deploymentService: deploymentService,
		serverPort:        serverPort,
	}
}

func (c *callFunctionTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("call_function",
		mcp.WithDescription("Call a smart contract function using deployment ID and ABI. For read-only functions (view/pure), returns result directly. For state-changing functions, creates a transaction session with signing URL."),
		mcp.WithString("deployment_id",
			mcp.Required(),
			mcp.Description("ID of the deployment containing the contract address and template with ABI"),
		),
		mcp.WithString("function_name",
			mcp.Required(),
			mcp.Description("Name of the function to call from the contract's ABI"),
		),
		mcp.WithArray("function_args",
			mcp.Description("JSON array of function arguments (e.g., [\"0x123...\", 1000000000000000000]). Optional. Provide arguments in the order they appear in the ABI."),
			mcp.Items(map[string]interface{}{
				"type":        "any",
				"description": "Function argument, provide the final value (e.g., for uint256 value of 1 ETH, provide \"1000000000000000000\")",
			}),
		),
		mcp.WithString("value",
			mcp.Description("ETH value to send with the function call in wei (e.g., \"1000000000000000000\" for 1 ETH). Optional, defaults to \"0\"."),
		),
		mcp.WithArray("metadata",
			mcp.Description("JSON array of metadata for the transaction (e.g., [{\"key\": \"Function Call\", \"value\": \"Transfer tokens\"}]). Optional."),
			mcp.Items(map[string]any{
				"type": "object",
				"properties": map[string]any{
					"key": map[string]any{
						"type":        "string",
						"description": "Key of the metadata",
					},
					"value": map[string]any{
						"type":        "string",
						"description": "Value of the metadata",
					},
				},
				"required": []string{"key", "value"},
			}),
		),
	)

	return tool
}

func (c *callFunctionTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Parse and validate arguments
		var args CallFunctionArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Parse deployment ID to uint
		deploymentID, err := strconv.ParseUint(args.DeploymentID, 10, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid deployment_id format: %v", err)), nil
		}

		// Get deployment from database
		deployment, err := c.deploymentService.GetDeploymentByID(uint(deploymentID))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Deployment not found: %v", err)), nil
		}

		// Verify deployment is confirmed and has contract address
		if deployment.Status != models.TransactionStatusConfirmed {
			return mcp.NewToolResultError("Deployment is not confirmed yet. Contract address not available"), nil
		}
		if deployment.ContractAddress == "" {
			return mcp.NewToolResultError("Deployment does not have a contract address"), nil
		}

		// Get active chain configuration
		activeChain, err := c.chainService.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Verify deployment chain matches active chain
		if deployment.ChainID != activeChain.ID {
			return mcp.NewToolResultError(fmt.Sprintf("Deployment is on different chain (ID: %d) than active chain (ID: %d)", deployment.ChainID, activeChain.ID)), nil
		}

		// Currently only support Ethereum
		if activeChain.ChainType != models.TransactionChainTypeEthereum {
			return mcp.NewToolResultError(fmt.Sprintf("Function calls are only supported on Ethereum, got %s", activeChain.ChainType)), nil
		}

		return c.makeEthereumFunctionCall(ctx, args, activeChain, deployment)
	}
}

func (c *callFunctionTool) makeEthereumFunctionCall(ctx context.Context, args CallFunctionArguments, activeChain *models.Chain, deployment *models.Deployment) (*mcp.CallToolResult, error) {
	// Get template to access ABI
	template, err := c.templateService.GetTemplateByID(deployment.TemplateID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Template not found: %v", err)), nil
	}

	// Verify template has ABI
	if template.Abi == nil {
		return mcp.NewToolResultError("Template does not have ABI information"), nil
	}

	// Get function from ABI
	method, err := c.evmService.GetAbiMethod(template.Abi, args.FunctionName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Function '%s' not found in ABI: %v", args.FunctionName, err)), nil
	}

	// Validate function arguments count
	if len(args.FunctionArgs) != len(method.Inputs) {
		return mcp.NewToolResultError(fmt.Sprintf("Function '%s' expects %d arguments, got %d", args.FunctionName, len(method.Inputs), len(args.FunctionArgs))), nil
	}

	// Determine if function is read-only (view/pure) or state-changing
	isReadOnly := method.StateMutability == "view" || method.StateMutability == "pure"

	if isReadOnly {
		// For read-only functions, call directly and return result
		result, err := c.callReadOnlyEthereumFunction(deployment.ContractAddress, method, args.FunctionArgs)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to call read-only function: %v", err)), nil
		}

		// Return direct result
		callResult := CallFunctionResult{
			DeploymentID:    args.DeploymentID,
			ContractAddress: deployment.ContractAddress,
			FunctionName:    args.FunctionName,
			Result:          result,
			Success:         true,
			IsReadOnly:      true,
		}

		resultJSON, _ := json.Marshal(callResult)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Function '%s' called successfully on deployment %s:", args.FunctionName, args.DeploymentID)),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil

	} else {
		// For state-changing functions, create transaction session
		sessionID, err := c.createFunctionCallTransaction(ctx, args, activeChain, deployment, template)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create function call transaction: %v", err)), nil
		}

		// Generate transaction session URL
		url, err := utils.GetTransactionSessionUrl(c.serverPort, sessionID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to get transaction session url: %v", err)), nil
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Function call transaction session created: %s", sessionID)),
				mcp.NewTextContent(fmt.Sprintf("Contract: %s (Deployment ID: %s)", deployment.ContractAddress, args.DeploymentID)),
				mcp.NewTextContent("Please sign the function call in the URL:"),
				mcp.NewTextContent(url),
			},
		}, nil
	}
}

func (c *callFunctionTool) callReadOnlyEthereumFunction(contractAddress string, method abi.Method, functionArgs []any) (string, error) {
	// For now, return a placeholder since we need to implement the actual blockchain call
	// This would use the evmService to make a read-only call to the contract
	// TODO: Implement actual read-only function call using evmService
	return fmt.Sprintf("Read-only call to %s with args %v (placeholder implementation)", method.Name, functionArgs), nil
}

func (c *callFunctionTool) createFunctionCallTransaction(ctx context.Context, args CallFunctionArguments, activeChain *models.Chain, deployment *models.Deployment, template *models.Template) (string, error) {
	// Set default value if not provided
	value := args.Value
	if value == "" {
		value = "0"
	}

	// Extract ABI array from template.Abi and convert to string
	var abiString string
	if abiData, exists := template.Abi["abi"]; exists {
		// If ABI is stored as "abi" field, marshal the array directly
		abiBytes, err := json.Marshal(abiData)
		if err != nil {
			return "", fmt.Errorf("failed to marshal ABI data: %w", err)
		}
		abiString = string(abiBytes)
	} else {
		// If template.Abi is the ABI array directly, marshal it
		abiBytes, err := json.Marshal(template.Abi)
		if err != nil {
			return "", fmt.Errorf("failed to marshal ABI: %w", err)
		}
		abiString = string(abiBytes)
	}

	// Create function call transaction
	tx, err := c.evmService.GetContractFunctionCallTransaction(services.GetContractFunctionCallTransactionArgs{
		ContractAddress: deployment.ContractAddress,
		FunctionName:    args.FunctionName,
		FunctionArgs:    args.FunctionArgs,
		Abi:             string(abiString),
		Value:           value,
		Title:           fmt.Sprintf("Call %s", args.FunctionName),
		Description:     fmt.Sprintf("Call function %s on contract %s", args.FunctionName, deployment.ContractAddress),
		TransactionType: models.TransactionTypeRegular,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create function call transaction: %w", err)
	}

	// Encode function arguments for raw contract arguments
	functionArgsString, err := utils.EncodeFunctionArgsToStringMapWithStringABI(args.FunctionName, args.FunctionArgs, string(abiString))
	if err != nil {
		return "", fmt.Errorf("failed to marshal raw contract arguments: %w", err)
	}
	tx.RawContractArguments = &functionArgsString
	tx.ContractAddress = &deployment.ContractAddress

	// Add metadata
	enhancedMetadata := append(args.Metadata, models.TransactionMetadata{
		Key:   "deployment_id",
		Value: args.DeploymentID,
	})
	enhancedMetadata = append(enhancedMetadata, models.TransactionMetadata{
		Key:   "function_name",
		Value: args.FunctionName,
	})
	enhancedMetadata = append(enhancedMetadata, models.TransactionMetadata{
		Key:   "contract_address",
		Value: deployment.ContractAddress,
	})

	// Get authenticated user
	user, _ := utils.GetAuthenticatedUser(ctx)
	var userId *string
	if user != nil {
		userId = &user.Sub
	}

	// Create transaction session
	sessionID, err := c.txService.CreateTransactionSession(services.CreateTransactionSessionRequest{
		TransactionDeployments: []models.TransactionDeployment{tx},
		ChainType:              models.TransactionChainTypeEthereum,
		ChainID:                activeChain.ID,
		Metadata:               enhancedMetadata,
		UserID:                 userId,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create transaction session: %w", err)
	}

	return sessionID, nil
}
