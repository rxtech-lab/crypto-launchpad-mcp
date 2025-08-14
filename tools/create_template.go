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
)

func NewCreateTemplateTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("create_template",
		mcp.WithDescription("Create new smart contract template with syntax validation. Template code should be valid Solidity (for Ethereum) or Rust (for Solana)."),
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
			mcp.Description("The smart contract source code template"),
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

		// Validate chain type
		if chainType != "ethereum" && chainType != "solana" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Invalid chain_type. Supported values: ethereum, solana"),
				},
			}, nil
		}

		// Basic syntax validation
		if err := validateTemplateCode(chainType, templateCode); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Template validation failed: %v", err)),
				},
			}, nil
		}

		// Create template
		template := &models.Template{
			Name:         name,
			Description:  description,
			ChainType:    chainType,
			TemplateCode: templateCode,
		}

		if err := db.CreateTemplate(template); err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error creating template: %v", err)),
				},
			}, nil
		}

		result := map[string]interface{}{
			"id":          template.ID,
			"name":        template.Name,
			"description": template.Description,
			"chain_type":  template.ChainType,
			"created_at":  template.CreatedAt,
			"message":     "Template created successfully",
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Success message: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}

// validateTemplateCode performs basic syntax validation for template code
func validateTemplateCode(chainType, templateCode string) error {
	if strings.TrimSpace(templateCode) == "" {
		return fmt.Errorf("template code cannot be empty")
	}

	switch chainType {
	case "ethereum":
		// Basic Solidity validation
		if !strings.Contains(templateCode, "pragma solidity") {
			return fmt.Errorf("Ethereum templates must include 'pragma solidity' directive")
		}
		if !strings.Contains(templateCode, "contract") {
			return fmt.Errorf("Ethereum templates must contain at least one contract")
		}
	case "solana":
		// Basic Rust validation for Solana programs
		if !strings.Contains(templateCode, "use anchor_lang::prelude::*") &&
			!strings.Contains(templateCode, "use solana_program") {
			return fmt.Errorf("Solana templates should use Anchor framework or native Solana program library")
		}
		if !strings.Contains(templateCode, "#[program]") &&
			!strings.Contains(templateCode, "entrypoint!") {
			return fmt.Errorf("Solana templates must define a program entrypoint")
		}
	}

	// Check for common security issues
	if strings.Contains(strings.ToLower(templateCode), "selfdestruct") {
		return fmt.Errorf("templates containing 'selfdestruct' are not allowed for security reasons")
	}

	return nil
}
