package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewUpdateTemplateTool(db interface{}) (mcp.Tool, server.ToolHandlerFunc) {
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
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateIDStr, err := request.RequireString("template_id")
		if err != nil {
			return nil, fmt.Errorf("template_id parameter is required: %w", err)
		}

		templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Invalid template_id: %v", err)),
				},
			}, nil
		}

		description := request.GetString("description", "")
		chainType := request.GetString("chain_type", "")
		contractName := request.GetString("contract_name", "")
		templateCode := request.GetString("template_code", "")
		templateMetadata := request.GetString("template_metadata", "")

		// Parse template metadata if provided
		var metadata models.JSON
		if templateMetadata != "" {
			if err := json.Unmarshal([]byte(templateMetadata), &metadata); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent(fmt.Sprintf("Invalid template_metadata JSON: %v", err)),
					},
				}, nil
			}

			// Validate metadata format - all values should be empty strings for parameter definitions
			for key, value := range metadata {
				if key == "" {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent("Error: "),
							mcp.NewTextContent("Metadata keys cannot be empty"),
						},
					}, nil
				}
				if str, ok := value.(string); !ok || str != "" {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent("Error: "),
							mcp.NewTextContent(fmt.Sprintf("Metadata values must be empty strings for parameter definitions, got %v for key %s", value, key)),
						},
					}, nil
				}
			}
		}

		// Get existing template
		template, err := db.GetTemplateByID(uint(templateID))
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Template not found: %v", err)),
				},
			}, nil
		}

		// Track what's being updated
		updates := make([]string, 0)

		// Update description if provided
		if description != "" {
			template.Description = description
			updates = append(updates, "description")
		}

		// Update chain type if provided
		if chainType != "" {
			if chainType != "ethereum" && chainType != "solana" {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent("Invalid chain_type. Supported values: ethereum, solana"),
					},
				}, nil
			}
			template.ChainType = models.TransactionChainType(chainType)
			updates = append(updates, "chain_type")
		}

		// Update contract name if provided
		if contractName != "" {
			template.ContractName = contractName
			updates = append(updates, "contract_name")
		}

		// Update metadata if provided
		if templateMetadata != "" {
			template.Metadata = metadata
			updates = append(updates, "metadata")
		}

		// Update template code if provided
		if templateCode != "" {
			// Use the current or new chain type for validation
			validationChainType := template.ChainType
			if chainType != "" {
				validationChainType = models.TransactionChainType(chainType)
			}

			// Use existing template code if not provided
			codeToValidate := templateCode
			if codeToValidate == "" {
				codeToValidate = template.TemplateCode
			}

			// Validate template code using Solidity compiler for Ethereum
			if validationChainType == "ethereum" {
				// Replace Go template variables with dummy values for compilation validation
				validationCode := utils.ReplaceTemplateVariables(codeToValidate)
				// Use Solidity version 0.8.20 for validation
				_, err := utils.CompileSolidity("0.8.20", validationCode)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent("Error: "),
							mcp.NewTextContent(fmt.Sprintf("Solidity compilation failed: %v", err)),
						},
					}, nil
				}
			} else if validationChainType == "solana" {
				// Solana validation skipped - accept any template code
			}

			// Update the template code if provided
			if templateCode != "" {
				template.TemplateCode = codeToValidate
				updates = append(updates, "template_code")
			}
		}

		// Check if any updates were provided
		if len(updates) == 0 {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("No update parameters provided. Specify description, chain_type, template_code, or template_metadata"),
				},
			}, nil
		}

		// Save updated template
		if err := db.UpdateTemplate(template); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error updating template: %v", err)),
				},
			}, nil
		}

		// Prepare result with comprehensive information
		result := map[string]interface{}{
			"id":             template.ID,
			"name":           template.Name,
			"description":    template.Description,
			"chain_type":     template.ChainType,
			"updated_at":     template.UpdatedAt,
			"updated_fields": updates,
			"message":        fmt.Sprintf("Template updated successfully. Fields updated: %v", updates),
		}

		// Add metadata information if updated
		if templateMetadata != "" && len(metadata) > 0 {
			result["metadata"] = metadata
			result["template_parameters"] = len(metadata)
		}

		// Format success message
		successMessage := "Template updated successfully"
		if len(updates) > 0 {
			successMessage += fmt.Sprintf(" (fields: %s)", strings.Join(updates, ", "))
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
