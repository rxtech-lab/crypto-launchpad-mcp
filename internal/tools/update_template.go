package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/constants"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type updateTemplateTool struct {
	templateService services.TemplateService
}

type UpdateTemplateArguments struct {
	// Required fields
	TemplateID string `json:"template_id" validate:"required"`

	// Optional fields
	Description      string         `json:"description,omitempty"`
	ChainType        string         `json:"chain_type,omitempty"`
	ContractName     string         `json:"contract_name,omitempty"`
	TemplateCode     string         `json:"template_code,omitempty"`
	TemplateMetadata string         `json:"template_metadata,omitempty"`
	TemplateValues   map[string]any `json:"template_values,omitempty"`
}

type UpdateTemplateResult struct {
	ID            uint                        `json:"id"`
	Name          string                      `json:"name"`
	Description   string                      `json:"description"`
	ChainType     models.TransactionChainType `json:"chain_type"`
	ContractNames []string                    `json:"contract_names"`
}

func NewUpdateTemplateTool(templateService services.TemplateService) *updateTemplateTool {
	return &updateTemplateTool{
		templateService: templateService,
	}
}

func (u *updateTemplateTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("update_template",
		mcp.WithDescription("Update existing smart contract template with new description, chain type, template code, or metadata. Performs syntax validation on updated code."),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("ID of the template to update"),
		),
		mcp.WithString("description",
			mcp.Description("New description for the template"),
		),
		mcp.WithString("chain_type",
			mcp.Description("New chain type (ethereum or solana)"),
		),
		mcp.WithString("contract_name",
			mcp.Description("New contract name"),
		),
		mcp.WithString("template_code",
			mcp.Description("New template code with Go template syntax ({{.VariableName}})"),
		),
		mcp.WithString("template_metadata",
			mcp.Description("JSON object defining template parameters as key-value pairs where values are empty strings (e.g., {\"TokenName\": \"\", \"TokenSymbol\": \"\"})"),
		),
		mcp.WithObject("template_values",
			mcp.Description("JSON object with runtime values for template parameters for validation (e.g., {\"TokenName\": \"MyToken\", \"TokenSymbol\": \"MTK\"})"),
		),
	)

	return tool
}

func (u *updateTemplateTool) GetHandler() server.ToolHandlerFunc {

	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var args UpdateTemplateArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
		}

		templateID, err := strconv.ParseUint(args.TemplateID, 10, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid template_id: %v", err)), nil
		}

		// Parse template metadata if provided
		var metadata models.JSON
		if args.TemplateMetadata != "" {
			if err := json.Unmarshal([]byte(args.TemplateMetadata), &metadata); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid template_metadata JSON: %v", err)), nil
			}

			// Validate metadata format - all values should be empty strings for parameter definitions
			for key, value := range metadata {
				if key == "" {
					return mcp.NewToolResultError("Metadata keys cannot be empty"), nil
				}
				if str, ok := value.(string); !ok || str != "" {
					return mcp.NewToolResultError(fmt.Sprintf("Metadata values must be empty strings for parameter definitions, got %v for key %s", value, key)), nil
				}
			}
		}

		// Get existing template
		template, err := u.templateService.GetTemplateByID(uint(templateID))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Template not found: %v", err)), nil
		}

		// Track which fields are being updated
		var updatedFields []string

		// Check if any updates are provided
		hasUpdates := args.Description != "" || args.ChainType != "" || args.TemplateCode != "" || args.TemplateMetadata != "" || args.TemplateValues != nil

		if !hasUpdates {
			return mcp.NewToolResultError("No update parameters provided"), nil
		}

		// Update description if provided
		if args.Description != "" {
			template.Description = args.Description
			updatedFields = append(updatedFields, "description")
		}

		// Update chain type if provided
		if args.ChainType != "" {
			if args.ChainType != "ethereum" && args.ChainType != "solana" {
				return mcp.NewToolResultError("Invalid chain_type. Supported values: ethereum, solana"), nil
			}

			// Check if changing to Solana with template code update
			if args.ChainType == "solana" && args.TemplateCode != "" {
				return mcp.NewToolResultError("Cannot update template code when changing chain type to Solana"), nil
			}

			template.ChainType = models.TransactionChainType(args.ChainType)
			updatedFields = append(updatedFields, "chain_type")
		}

		// Update metadata if provided
		if args.TemplateMetadata != "" {
			template.Metadata = metadata
			updatedFields = append(updatedFields, "metadata")
		}

		if args.TemplateValues != nil {
			template.SampleTemplateValues = args.TemplateValues
			updatedFields = append(updatedFields, "template_values")
		}

		var compilationResult *utils.CompilationResult
		// Update template code if provided
		if args.TemplateCode != "" {
			updatedFields = append(updatedFields, "template_code")

			// Use the current or new chain type for validation
			validationChainType := template.ChainType
			if args.ChainType != "" {
				validationChainType = models.TransactionChainType(args.ChainType)
			}

			// Validate template code using Solidity compiler for Ethereum
			switch validationChainType {
			case "ethereum":
				// For validation, use dummy values if TemplateValues not provided
				templateValues := args.TemplateValues
				if templateValues == nil {
					// Use sample values from existing template or provide dummy ones
					templateValues = template.SampleTemplateValues
					if templateValues == nil {
						templateValues = map[string]any{"TokenName": "TestToken", "TokenSymbol": "TEST", "InitialSupply": "1000"}
					}
				}

				renderedCode, err := utils.RenderContractTemplate(args.TemplateCode, templateValues)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Error rendering template: %v", err)), nil
				}

				// Use Solidity version 0.8.27 for validation
				result, err := utils.CompileSolidity(constants.SolidityCompilerVersion, renderedCode)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Solidity compilation failed: %v", err)), nil
				}

				compilationResult = &result

				// Update ABI if contract name provided and compilation successful
				if args.ContractName != "" {
					if abi, exists := compilationResult.Abi[args.ContractName]; exists {
						// Convert the ABI to models.JSON format
						if abiMap, ok := abi.(models.JSON); ok {
							template.Abi = abiMap
						} else {
							// The ABI from compilation is an array, but models.JSON is a map
							// Wrap the ABI array in a map structure to store it properly
							template.Abi = models.JSON{
								"abi": abi,
							}
						}
					}
				}
			case "solana":
				// Solana template code updates are not supported
				return mcp.NewToolResultError("Solana template code updates are not supported"), nil
			}

			// Update the template code
			template.TemplateCode = args.TemplateCode
		}

		// Save updated template
		if err := u.templateService.UpdateTemplate(template); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error updating template: %v", err)), nil
		}

		// Prepare result with comprehensive information
		result := map[string]interface{}{
			"id":             template.ID,
			"name":           template.Name,
			"description":    template.Description,
			"chain_type":     template.ChainType,
			"updated_fields": updatedFields,
		}

		// Add compilation information for Ethereum
		if compilationResult != nil {
			var contractNames []string
			for contractName := range compilationResult.Abi {
				contractNames = append(contractNames, contractName)
			}
			result["contract_names"] = contractNames
		}

		// Add metadata information if updated
		if args.TemplateMetadata != "" && metadata != nil && len(metadata) > 0 {
			result["template_parameters"] = len(metadata)
			result["metadata"] = metadata
		}

		// Format success message with updated fields
		successMessage := "Template updated successfully"
		if len(updatedFields) > 0 {
			successMessage += " (" + fmt.Sprintf("updated: %v", updatedFields) + ")"
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(successMessage + ": "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}
}
