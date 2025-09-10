package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type createTemplateTool struct {
	templateService services.TemplateService
}

type CreateTemplateArguments struct {
	// Required fields
	Name           string         `json:"name" validate:"required"`
	Description    string         `json:"description" validate:"required"`
	ContractName   string         `json:"contract_name" validate:"required"`
	ChainType      string         `json:"chain_type" validate:"required"`
	TemplateCode   string         `json:"template_code" validate:"required"`
	TemplateValues map[string]any `json:"template_values" validate:"required"`

	// Optional fields
	TemplateMetadata string `json:"template_metadata,omitempty"`
}

type CreateTemplateResult struct {
	ID                 uint                        `json:"id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	ChainType          models.TransactionChainType `json:"chain_type"`
	ContractNames      []string                    `json:"contract_names,omitempty"`
	TemplateParameters int                         `json:"template_parameters,omitempty"`
	Metadata           models.JSON                 `json:"metadata,omitempty"`
}

type CompilationResult struct {
	Bytecode map[string]string `json:"bytecode"`
	Abi      map[string]any    `json:"abi"`
}

func NewCreateTemplateTool(templateService services.TemplateService) *createTemplateTool {
	return &createTemplateTool{
		templateService: templateService,
	}
}

func (c *createTemplateTool) GetTool() mcp.Tool {
	tool := mcp.NewTool("create_template",
		mcp.WithDescription("Create new smart contract template with syntax validation. Template code should use Go template syntax ({{.VariableName}}) for dynamic parameters. OpenZeppelin contracts are available to use."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the template (e.g., 'ERC20 Basic Token', 'SPL Token')"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Description of what this template does"),
		),
		mcp.WithString("contract_name",
			mcp.Required(),
			mcp.Description("Name of the contract to be deployed"),
		),
		mcp.WithString("chain_type",
			mcp.Required(),
			mcp.Description("Target blockchain type (ethereum or solana)"),
		),
		mcp.WithString("template_code",
			mcp.Required(),
			mcp.Description("The smart contract source code template with Go template syntax ({{.VariableName}}). Don't need to include the contract owner info since it will be set during the deployment. Use msg.sender as the contract owner if not specified."+
				"Please fix the SPDX-License-Identifier and pragma statements"),
		),
		mcp.WithString("template_metadata",
			mcp.Description("JSON object defining template parameters as key-value pairs where values are empty strings (e.g., {\"TokenName\": \"\", \"TokenSymbol\": \"\"})"),
		),
		mcp.WithObject("template_values",
			mcp.Required(),
			mcp.Description("JSON object with runtime values for template parameters (e.g., {\"TokenName\": \"MyToken\", \"TokenSymbol\": \"MTK\"})"),
		),
	)

	return tool
}

func (c *createTemplateTool) GetHandler() server.ToolHandlerFunc {

	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var userId *string
		user, _ := utils.GetAuthenticatedUser(ctx)
		if user != nil {
			userId = &user.Sub
		}

		// Parse and validate arguments

		var args CreateTemplateArguments
		if err := request.BindArguments(&args); err != nil {
			return nil, fmt.Errorf("failed to bind arguments: %w", err)
		}

		if err := validator.New().Struct(args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments: %v", err)), nil
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

		// Validate chain type
		if args.ChainType != "ethereum" && args.ChainType != "solana" {
			return mcp.NewToolResultError("Invalid chain_type. Supported values: ethereum, solana"), nil
		}

		// Validate template code using Solidity compiler for Ethereum
		var compilationResult *utils.CompilationResult
		switch args.ChainType {
		case "ethereum":
			// Render template with dummy values
			validationCode, err := utils.RenderContractTemplate(args.TemplateCode, args.TemplateValues)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error rendering template with provided values: %v", err)), nil
			}

			// Use Solidity version 0.8.20 for validation
			result, err := utils.CompileSolidity("0.8.27", validationCode)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Solidity compilation failed. Please fix the template code base on the error: %v", err)), nil
			}
			// check if the contract name is in the result
			contract := result.Abi[args.ContractName]
			if contract == nil {
				availableContracts := ""
				for name := range result.Abi {
					availableContracts += name + ","
				}
				return mcp.NewToolResultError(fmt.Sprintf("Contract %s not found in the compilation result. Make sure the contract name matches the contract name defined in your code. "+
					"AvailableContracts are: %s", args.ContractName, availableContracts)), nil
			}
			compilationResult = &result
		case "solana":
			// Solana validation skipped - accept any template code
		}

		// Create template
		template := &models.Template{
			Name:                 args.Name,
			Description:          args.Description,
			ChainType:            models.TransactionChainType(args.ChainType),
			TemplateCode:         args.TemplateCode,
			SampleTemplateValues: args.TemplateValues,
			Metadata:             metadata,
			UserId:               userId,
		}

		// Set ABI only for Ethereum contracts with successful compilation
		if compilationResult != nil && args.ChainType == "ethereum" {
			if abi, exists := compilationResult.Abi[args.ContractName]; exists {
				// Convert the ABI to models.JSON format
				if abiMap, ok := abi.(models.JSON); ok {
					template.Abi = abiMap
				} else {
					// If it's not already a models.JSON, convert it
					abiBytes, _ := json.Marshal(abi)
					var abiMap models.JSON
					json.Unmarshal(abiBytes, &abiMap)
					template.Abi = abiMap
				}
			}
		}

		if err := c.templateService.CreateTemplate(template); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating template: %v", err)), nil
		}

		// Prepare result
		result := CreateTemplateResult{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			ChainType:   template.ChainType,
		}

		// Add compilation information for Ethereum
		if compilationResult != nil {
			var contractNames []string
			for contractName := range compilationResult.Bytecode {
				contractNames = append(contractNames, contractName)
			}
			result.ContractNames = contractNames
		}

		// Add metadata information
		if len(metadata) > 0 {
			result.TemplateParameters = len(metadata)
			result.Metadata = metadata
		}

		// Format success message
		successMessage := "Template created successfully"
		if compilationResult != nil {
			successMessage += " (Solidity compilation validated)"
		} else if args.ChainType == "solana" {
			successMessage += " (Rust syntax validated)"
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
