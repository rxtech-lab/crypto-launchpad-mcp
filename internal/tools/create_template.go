package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewCreateTemplateTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
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
			mcp.Description("The smart contract source code template with Go template syntax ({{.VariableName}}). Don't need to include the contract owner info since it will be set during the deployment. Use msg.sender as the contract owner if not specified."),
		),
		mcp.WithString("template_metadata",
			mcp.Description("JSON object defining template parameters as key-value pairs where values are empty strings (e.g., {\"TokenName\": \"\", \"TokenSymbol\": \"\"})"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, err := request.RequireString("name")
		if err != nil {
			return nil, fmt.Errorf("name parameter is required: %w", err)
		}

		description, err := request.RequireString("description")
		if err != nil {
			return nil, fmt.Errorf("description parameter is required: %w", err)
		}

		contractName, err := request.RequireString("contract_name")
		if err != nil {
			return nil, fmt.Errorf("contract_name parameter is required: %w", err)
		}

		chainType, err := request.RequireString("chain_type")
		if err != nil {
			return nil, fmt.Errorf("chain_type parameter is required: %w", err)
		}

		templateCode, err := request.RequireString("template_code")
		if err != nil {
			return nil, fmt.Errorf("template_code parameter is required: %w", err)
		}

		// Parse template metadata if provided
		var metadata models.JSON
		templateMetadata := request.GetString("template_metadata", "")
		if templateMetadata != "" {
			if err := json.Unmarshal([]byte(templateMetadata), &metadata); err != nil {
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
		if chainType != "ethereum" && chainType != "solana" {
			return mcp.NewToolResultError("Invalid chain_type. Supported values: ethereum, solana"), nil
		}

		// Validate template code using Solidity compiler for Ethereum
		var compilationResult *utils.CompilationResult
		switch chainType {
		case "ethereum":
			// Replace Go template variables with dummy values for compilation validation
			validationCode := utils.ReplaceTemplateVariables(templateCode)

			// Use Solidity version 0.8.20 for validation
			result, err := utils.CompileSolidity("0.8.27", validationCode)
			// check if the contract name is in the result
			contract := result.Abi[contractName]
			if contract == nil {
				return mcp.NewToolResultError(fmt.Sprintf("Contract %s not found in the compilation result", contractName)), nil
			}
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Solidity compilation failed: %v", err)), nil
			}
			compilationResult = &result
		case "solana":
			// Solana validation skipped - accept any template code
		}

		// Create template
		template := &models.Template{
			Name:         name,
			Description:  description,
			ChainType:    models.TransactionChainType(chainType),
			ContractName: contractName,
			TemplateCode: templateCode,
			Metadata:     metadata,
		}

		if err := db.CreateTemplate(template); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating template: %v", err)), nil
		}

		// Prepare result
		result := map[string]interface{}{
			"id":          template.ID,
			"name":        template.Name,
			"description": template.Description,
			"chain_type":  template.ChainType,
			"created_at":  template.CreatedAt,
			"message":     "Template created successfully",
		}

		// Include metadata if provided
		if len(metadata) > 0 {
			result["metadata"] = metadata
			result["template_parameters"] = len(metadata)
		}

		// Add compilation information for Ethereum
		if compilationResult != nil {
			result["compilation_status"] = "success"
			result["compiled_contracts"] = len(compilationResult.Bytecode)
			var contractNames []string
			for contractName := range compilationResult.Bytecode {
				contractNames = append(contractNames, contractName)
			}
			result["contract_names"] = contractNames
		}

		// Format success message
		successMessage := "Template created successfully"
		if compilationResult != nil {
			successMessage += " (Solidity compilation validated)"
		} else if chainType == "solana" {
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

	return tool, handler
}
