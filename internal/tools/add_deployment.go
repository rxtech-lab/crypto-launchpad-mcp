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

type addDeploymentTool struct {
	deploymentService services.DeploymentService
	templateService   services.TemplateService
	chainService      services.ChainService
}

type AddDeploymentArguments struct {
	// Required fields
	TemplateID      string `json:"template_id" validate:"required"`
	ChainID         string `json:"chain_id" validate:"required"`
	ContractAddress string `json:"contract_address" validate:"required"`
	TransactionHash string `json:"transaction_hash" validate:"required"`
	DeployerAddress string `json:"deployer_address" validate:"required"`

	// Optional fields
	Status         string         `json:"status,omitempty"`
	TemplateValues map[string]any `json:"template_values,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	UserID         string         `json:"user_id,omitempty"`
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
		mcp.WithDescription("Manually add a deployment entry to the database for contracts deployed outside the system. Useful for tracking existing deployments."),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("ID of the template used for deployment"),
		),
		mcp.WithString("chain_id",
			mcp.Required(),
			mcp.Description("ID of the chain where contract is deployed"),
		),
		mcp.WithString("contract_address",
			mcp.Required(),
			mcp.Description("Deployed contract address (e.g., 0x123...)"),
		),
		mcp.WithString("transaction_hash",
			mcp.Required(),
			mcp.Description("Transaction hash of the deployment (e.g., 0xabc...)"),
		),
		mcp.WithString("deployer_address",
			mcp.Required(),
			mcp.Description("Address that deployed the contract (e.g., 0xdef...)"),
		),
		mcp.WithString("status",
			mcp.Description("Deployment status (pending, confirmed, failed). Defaults to 'confirmed'"),
		),
		mcp.WithObject("template_values",
			mcp.Description("JSON object with template parameter values used during deployment (e.g., {\"TokenName\": \"MyToken\", \"TokenSymbol\": \"MTK\"})"),
		),
		mcp.WithString("session_id",
			mcp.Description("Optional session ID if this deployment is associated with a transaction session"),
		),
		mcp.WithString("user_id",
			mcp.Description("Optional user ID to associate with this deployment. If not provided, will use authenticated user if available"),
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

		// Parse template ID
		templateID, err := strconv.ParseUint(args.TemplateID, 10, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid template_id: %v", err)), nil
		}

		// Parse chain ID
		chainID, err := strconv.ParseUint(args.ChainID, 10, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid chain_id: %v", err)), nil
		}

		// Verify template exists
		template, err := a.templateService.GetTemplateByID(uint(templateID))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Template not found: %v", err)), nil
		}

		// Verify chain exists by attempting to get all chains and finding the one with matching ID
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

		// Validate that template chain type matches chain
		if template.ChainType != chain.ChainType {
			return mcp.NewToolResultError(fmt.Sprintf("Template chain type (%s) doesn't match chain type (%s)", template.ChainType, chain.ChainType)), nil
		}

		// Determine status (default to confirmed)
		status := models.TransactionStatusConfirmed
		if args.Status != "" {
			status = models.TransactionStatus(args.Status)
			// Validate status
			if status != models.TransactionStatusPending && status != models.TransactionStatusConfirmed && status != models.TransactionStatusFailed {
				return mcp.NewToolResultError("Invalid status. Must be one of: pending, confirmed, failed"), nil
			}
		}

		// Convert template values to JSON
		var templateValuesJSON models.JSON
		if args.TemplateValues != nil {
			templateValuesJSON = args.TemplateValues
		}

		// Determine user ID
		var userIDPtr *string
		if args.UserID != "" {
			userIDPtr = &args.UserID
		} else if user, ok := utils.GetAuthenticatedUser(ctx); ok {
			userIDPtr = &user.Sub
		}

		// Create deployment
		deployment := &models.Deployment{
			TemplateID:      uint(templateID),
			ChainID:         uint(chainID),
			ContractAddress: args.ContractAddress,
			TransactionHash: args.TransactionHash,
			DeployerAddress: args.DeployerAddress,
			Status:          status,
			TemplateValues:  templateValuesJSON,
			SessionId:       args.SessionID,
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
				mcp.NewTextContent("Deployment added successfully: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}
}
