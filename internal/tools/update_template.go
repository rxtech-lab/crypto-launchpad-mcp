package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewUpdateTemplateTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
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
		mcp.WithString("template_code",
			mcp.Description("New template code with Go template variables"),
		),
		mcp.WithString("metadata",
			mcp.Description("JSON object defining template variables (e.g., {\"TokenName\": \"\", \"TokenSymbol\": \"\"}). Values should be empty - they will be provided during deployment."),
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
		templateCode := request.GetString("template_code", "")
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

		// OpenZeppelin configuration
		useOpenZeppelinStr := request.GetString("use_openzeppelin", "")
		openzeppelinVersion := request.GetString("openzeppelin_version", "")
		openzeppelinContractsStr := request.GetString("openzeppelin_contracts", "")

		// Parse OpenZeppelin usage
		var useOpenZeppelin bool
		if useOpenZeppelinStr != "" {
			useOpenZeppelin = strings.ToLower(useOpenZeppelinStr) == "true"
		}

		// Convert comma-separated OpenZeppelin contracts to slice
		var ozContracts []string
		if openzeppelinContractsStr != "" {
			for _, contract := range strings.Split(openzeppelinContractsStr, ",") {
				contract = strings.TrimSpace(contract)
				if contract != "" {
					ozContracts = append(ozContracts, contract)
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
			template.ChainType = chainType
			updates = append(updates, "chain_type")
		}

		// Update metadata if provided
		if metadata != "" {
			template.Metadata = metadata
			updates = append(updates, "metadata")
		}

		// Update template code if provided
		if templateCode != "" || useOpenZeppelinStr != "" {
			// Use the current or new chain type for validation
			validationChainType := template.ChainType
			if chainType != "" {
				validationChainType = chainType
			}

			// OpenZeppelin validation for non-Ethereum chains
			if useOpenZeppelin && validationChainType != "ethereum" {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent("OpenZeppelin is only supported for Ethereum contracts"),
					},
				}, nil
			}

			// Use existing template code if not provided
			codeToValidate := templateCode
			if codeToValidate == "" {
				codeToValidate = template.TemplateCode
			}

			// Validate template code using Solidity compiler for Ethereum
			if validationChainType == "ethereum" {
				// Use Solidity version 0.8.20 for validation
				_, err := utils.CompileSolidity("0.8.20", codeToValidate)
				if err != nil {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent("Error: "),
							mcp.NewTextContent(fmt.Sprintf("Solidity compilation failed: %v", err)),
						},
					}, nil
				}
			} else if validationChainType == "solana" {
				// For Solana, perform basic validation checks
				if !strings.Contains(codeToValidate, "use anchor_lang::prelude::*;") && !strings.Contains(codeToValidate, "#[program]") {
					return &mcp.CallToolResult{
						Content: []mcp.Content{
							mcp.NewTextContent("Error: "),
							mcp.NewTextContent("Solana template must contain anchor_lang imports or #[program] attribute"),
						},
					}, nil
				}
			}

			// Update the template code if provided
			if templateCode != "" {
				template.TemplateCode = codeToValidate
				updates = append(updates, "template_code")
			}
		}

		// Check if any updates were provided
		if len(updates) == 0 && useOpenZeppelinStr == "" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("No update parameters provided. Specify description, chain_type, template_code, or OpenZeppelin options"),
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

		// Add OpenZeppelin information if configured
		if useOpenZeppelinStr != "" {
			result["use_openzeppelin"] = useOpenZeppelin
			if openzeppelinVersion != "" {
				result["openzeppelin_version"] = openzeppelinVersion
			}
			if len(ozContracts) > 0 {
				result["openzeppelin_contracts"] = ozContracts
			}
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
