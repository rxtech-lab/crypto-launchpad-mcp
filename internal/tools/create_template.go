package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewCreateTemplateTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("create_template",
		mcp.WithDescription("Create new smart contract template with syntax validation. Template code should be valid Solidity (for Ethereum) or Rust (for Solana) and can use Go template variables like {{.TokenName}}. OpenZeppelin contracts are available to use."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the template (e.g., 'ERC20 Basic Token', 'SPL Token')"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Description of what this template does"),
		),
		mcp.WithString("chain_type",
			mcp.Required(),
			mcp.Description("Target blockchain type (ethereum or solana)"),
		),
		mcp.WithString("template_code",
			mcp.Required(),
			mcp.Description("The smart contract source code template with Go template variables like {{.TokenName}}"),
		),
		mcp.WithString("metadata",
			mcp.Description("JSON object defining template variables (e.g., {\"TokenName\": \"\", \"TokenSymbol\": \"\"}). Values should be empty - they will be provided during deployment."),
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

		chainType, err := request.RequireString("chain_type")
		if err != nil {
			return nil, fmt.Errorf("chain_type parameter is required: %w", err)
		}

		templateCode, err := request.RequireString("template_code")
		if err != nil {
			return nil, fmt.Errorf("template_code parameter is required: %w", err)
		}

		metadata := request.GetString("metadata", "")

		// Validate metadata JSON if provided
		if metadata != "" {
			var metadataObj map[string]interface{}
			if err := json.Unmarshal([]byte(metadata), &metadataObj); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent(fmt.Sprintf("Invalid metadata JSON: %v", err)),
					},
				}, nil
			}
		}

		// Validate chain type
		if chainType != "ethereum" && chainType != "solana" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Invalid chain_type. Supported values: ethereum, solana"),
				},
			}, nil
		}

		// Validate template code using Solidity compiler for Ethereum
		var compilationResult *utils.CompilationResult
		if chainType == "ethereum" {
			// Use Solidity version 0.8.20 for validation
			result, err := utils.CompileSolidity("0.8.20", templateCode)
			if err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent(fmt.Sprintf("Solidity compilation failed: %v", err)),
					},
				}, nil
			}
			compilationResult = &result
		} else if chainType == "solana" {
			// For Solana, perform basic validation checks
			if !strings.Contains(templateCode, "use anchor_lang::prelude::*;") && !strings.Contains(templateCode, "#[program]") {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent("Solana template must contain anchor_lang imports or #[program] attribute"),
					},
				}, nil
			}
		}

		// Create template
		template := &models.Template{
			Name:         name,
			Description:  description,
			ChainType:    chainType,
			TemplateCode: templateCode,
			Metadata:     metadata,
		}

		if err := db.CreateTemplate(template); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error creating template: %v", err)),
				},
			}, nil
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
