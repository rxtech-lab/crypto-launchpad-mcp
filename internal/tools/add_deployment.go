package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type addDeploymentTool struct {
	deploymentService services.DeploymentService
	templateService   services.TemplateService
	chainService      services.ChainService
}

type AddDeploymentArguments struct {
	// Required fields
	ContractCode    string `json:"contract_code" validate:"required"`
	ContractAddress string `json:"contract_address" validate:"required"`
	OwnerAddress    string `json:"owner_address" validate:"required"`
	ChainID         string `json:"chain_id" validate:"required"`

	// Optional fields
	TransactionHash string         `json:"transaction_hash,omitempty"`
	TemplateValues  map[string]any `json:"template_values,omitempty"`
	SolcVersion     string         `json:"solc_version,omitempty"`
	TemplateName    string         `json:"template_name,omitempty"`
	Description     string         `json:"description,omitempty"`
}

func NewAddDeploymentTool(deploymentService services.DeploymentService, templateService services.TemplateService, chainService services.ChainService) *addDeploymentTool {
	return &addDeploymentTool{
		deploymentService: deploymentService,
		templateService:   templateService,
		chainService:      chainService,
	}
}

func (a *addDeploymentTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("add_deployment",
		mcp.WithDescription("Add a deployment entry by providing contract code, contract address, and owner address. The tool will compile the contract, create a template automatically, and track the deployment."),
		mcp.WithString("contract_code",
			mcp.Required(),
			mcp.Description("Solidity contract source code"),
		),
		mcp.WithString("contract_address",
			mcp.Required(),
			mcp.Description("Deployed contract address (e.g., 0x123...)"),
		),
		mcp.WithString("owner_address",
			mcp.Required(),
			mcp.Description("Address of the contract owner (e.g., 0xdef...)"),
		),
		mcp.WithString("chain_id",
			mcp.Required(),
			mcp.Description("ID of the chain where contract is deployed"),
		),
		mcp.WithString("transaction_hash",
			mcp.Description("Transaction hash of the deployment (e.g., 0xabc...)"),
		),
		mcp.WithObject("template_values",
			mcp.Description("JSON object with template parameter values used during deployment (e.g., {\"TokenName\": \"MyToken\", \"TokenSymbol\": \"MTK\"})"),
		),
		mcp.WithString("solc_version",
			mcp.Description("Solidity compiler version (e.g., 0.8.20). Defaults to 0.8.20"),
		),
		mcp.WithString("template_name",
			mcp.Description("Name for the auto-created template. If not provided, will use contract name"),
		),
		mcp.WithString("description",
			mcp.Description("Description for the auto-created template"),
		),
	)

	return tool
}

func (a *addDeploymentTool) GetHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args AddDeploymentArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		// Parse chain ID
		chainID, err := strconv.ParseUint(args.ChainID, 10, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid chain_id: %v", err)), nil
		}

		// Verify chain exists
		chains, err := a.chainService.ListChains()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to list chains: %v", err)), nil
		}

		var chain *models.Chain
		for i := range chains {
			if chains[i].ID == uint(chainID) {
				chain = &chains[i]
				break
			}
		}

		if chain == nil {
			return mcp.NewToolResultError("Chain not found"), nil
		}

		// Default solc version
		solcVersion := args.SolcVersion
		if solcVersion == "" {
			solcVersion = "0.8.20"
		}

		// Compile contract
		compilationResult, err := utils.CompileSolidity(solcVersion, args.ContractCode)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to compile contract: %v", err)), nil
		}

		// Extract first contract name and its ABI
		var contractName string
		var contractABI models.JSON
		for name, abi := range compilationResult.Abi {
			contractName = name
			// Convert the ABI to models.JSON format
			if abiMap, ok := abi.(models.JSON); ok {
				contractABI = abiMap
			} else {
				// The ABI from compilation is typically an array, but models.JSON is a map
				// Wrap the ABI array in a map structure to store it properly
				contractABI = models.JSON{
					"abi": abi,
				}
			}
			break
		}

		if contractName == "" {
			return mcp.NewToolResultError("No contract found in compilation result"), nil
		}

		// Determine template name
		templateName := args.TemplateName
		if templateName == "" {
			templateName = contractName
		}

		// Get user ID from context
		var userIDPtr *string
		if user, ok := utils.GetAuthenticatedUser(ctx); ok {
			userIDPtr = &user.Sub
		}

		// Create template
		template := &models.Template{
			Name:         templateName,
			Description:  args.Description,
			UserId:       userIDPtr,
			ChainType:    chain.ChainType,
			TemplateCode: args.ContractCode,
			Abi:          contractABI,
		}

		if err := a.templateService.CreateTemplate(template); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create template: %v", err)), nil
		}

		// Convert template values to JSON
		var templateValuesJSON models.JSON
		if args.TemplateValues != nil {
			templateValuesJSON = args.TemplateValues
		}

		// Generate session ID
		sessionID := uuid.New().String()

		// Create deployment with confirmed status by default
		deployment := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         uint(chainID),
			ContractAddress: args.ContractAddress,
			TransactionHash: args.TransactionHash,
			DeployerAddress: args.OwnerAddress,
			Status:          models.TransactionStatusConfirmed,
			TemplateValues:  templateValuesJSON,
			SessionId:       sessionID,
			UserID:          userIDPtr,
		}

		if err := a.deploymentService.CreateDeployment(deployment); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create deployment: %v", err)), nil
		}

		// Retrieve full deployment with relationships
		fullDeployment, err := a.deploymentService.GetDeploymentByID(deployment.ID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to retrieve created deployment: %v", err)), nil
		}

		// Format result
		result := map[string]interface{}{
			"id":               fullDeployment.ID,
			"template_id":      fullDeployment.TemplateID,
			"chain_id":         fullDeployment.ChainID,
			"contract_address": fullDeployment.ContractAddress,
			"transaction_hash": fullDeployment.TransactionHash,
			"deployer_address": fullDeployment.DeployerAddress,
			"status":           fullDeployment.Status,
			"session_id":       fullDeployment.SessionId,
			"user_id":          fullDeployment.UserID,
			"created_at":       fullDeployment.CreatedAt,
			"updated_at":       fullDeployment.UpdatedAt,
		}

		// Include template information
		if fullDeployment.Template.ID != 0 {
			result["template"] = map[string]interface{}{
				"id":          fullDeployment.Template.ID,
				"name":        fullDeployment.Template.Name,
				"description": fullDeployment.Template.Description,
				"chain_type":  fullDeployment.Template.ChainType,
			}
		}

		// Include chain information
		if fullDeployment.Chain.ID != 0 {
			result["chain"] = map[string]interface{}{
				"id":         fullDeployment.Chain.ID,
				"name":       fullDeployment.Chain.Name,
				"chain_id":   fullDeployment.Chain.NetworkID,
				"chain_type": fullDeployment.Chain.ChainType,
				"rpc":        fullDeployment.Chain.RPC,
			}
		}

		// Include template values if present
		if len(fullDeployment.TemplateValues) > 0 {
			result["template_values"] = fullDeployment.TemplateValues
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Deployment added successfully with auto-created template: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}
}
