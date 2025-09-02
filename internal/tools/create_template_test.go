package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/assert"
)

func setupTestDatabase(t *testing.T) services.TemplateService {
	db, err := services.NewSqliteDBService(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}
	return services.NewTemplateService(db.GetDB())
}

func TestNewCreateTemplateTool(t *testing.T) {
	templateService := setupTestDatabase(t)
	createTool := NewCreateTemplateTool(templateService)
	tool := createTool.GetTool()
	handler := createTool.GetHandler()

	// Test tool metadata
	assert.Equal(t, "create_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.NotNil(t, handler)

	// Check that the tool has the expected properties
	assert.Contains(t, tool.InputSchema.Properties, "name")
	assert.Contains(t, tool.InputSchema.Properties, "description")
	assert.Contains(t, tool.InputSchema.Properties, "chain_type")
	assert.Contains(t, tool.InputSchema.Properties, "template_code")
	assert.Contains(t, tool.InputSchema.Properties, "template_metadata")
}

func TestCreateTemplateHandler_ParameterValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		requestArgs map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing_name",
			requestArgs: map[string]interface{}{
				"description":     "Test description",
				"contract_name":   "TestToken",
				"chain_type":      "ethereum",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
			expectError: true,
			errorMsg:    "Name",
		},
		{
			name: "missing_description",
			requestArgs: map[string]interface{}{
				"name":            "Test Template",
				"contract_name":   "TestToken",
				"chain_type":      "ethereum",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
			expectError: true,
			errorMsg:    "Description",
		},
		{
			name: "missing_chain_type",
			requestArgs: map[string]interface{}{
				"name":            "Test Template",
				"description":     "Test description",
				"contract_name":   "TestToken",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
			expectError: true,
			errorMsg:    "ChainType",
		},
		{
			name: "missing_contract_name",
			requestArgs: map[string]interface{}{
				"name":            "Test Template",
				"description":     "Test description",
				"chain_type":      "ethereum",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
			expectError: true,
			errorMsg:    "ContractName",
		},
		{
			name: "missing_template_code",
			requestArgs: map[string]interface{}{
				"name":            "Test Template",
				"description":     "Test description",
				"contract_name":   "TestToken",
				"chain_type":      "ethereum",
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
			expectError: true,
			errorMsg:    "TemplateCode",
		},
		{
			name: "missing_template_values",
			requestArgs: map[string]interface{}{
				"name":          "Test Template",
				"description":   "Test description",
				"contract_name": "TestToken",
				"chain_type":    "ethereum",
				"template_code": validEthereumTemplate(),
			},
			expectError: true,
			errorMsg:    "TemplateValues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateService := setupTestDatabase(t)
			createTemplateTool := NewCreateTemplateTool(templateService)
			handler := createTemplateTool.GetHandler()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.requestArgs,
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				if err != nil {
					// BindArguments error case
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "failed to bind arguments")
					assert.Nil(t, result)
				} else {
					// Validation error case
					assert.NoError(t, err)
					assert.NotNil(t, result)
					assert.True(t, result.IsError)
					if len(result.Content) > 0 {
						if textContent, ok := result.Content[0].(mcp.TextContent); ok {
							assert.Contains(t, textContent.Text, tt.errorMsg)
						}
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.False(t, result.IsError)
			}
		})
	}
}

func TestCreateTemplateHandler_ChainTypeValidation(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)
	createTemplateTool := NewCreateTemplateTool(templateService)
	handler := createTemplateTool.GetHandler()

	tests := []struct {
		name        string
		chainType   string
		expectError bool
	}{
		{
			name:        "valid_ethereum",
			chainType:   "ethereum",
			expectError: false,
		},
		{
			name:        "valid_solana",
			chainType:   "solana",
			expectError: false,
		},
		{
			name:        "invalid_chain_type",
			chainType:   "bitcoin",
			expectError: true,
		},
		{
			name:        "empty_chain_type",
			chainType:   "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"name":          "Test Template",
						"description":   "Test description",
						"contract_name": "TestToken",
						"chain_type":    tt.chainType,
						"template_code": func() string {
							if tt.chainType == "solana" {
								return validSolanaTemplate()
							}
							return validEthereumTemplate()
						}(),
						"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
					},
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				assert.NoError(t, err) // Handler returns success with error content
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				assert.Len(t, result.Content, 1)
				textContent0 := result.Content[0].(mcp.TextContent)
				if tt.chainType == "" {
					assert.Contains(t, textContent0.Text, "ChainType")
				} else {
					assert.Contains(t, textContent0.Text, "Invalid chain_type")
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestCreateTemplateHandler_EthereumTemplateValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		templateCode string
		contractName string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid_ethereum_template",
			templateCode: validEthereumTemplate(),
			contractName: "SimpleToken",
			expectError:  false,
		},
		{
			name: "valid_openzeppelin_template",
			templateCode: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin-contracts/contracts/token/ERC20/ERC20.sol";

contract MyToken is ERC20 {
    constructor() ERC20("{{.TokenName}}", "{{.TokenSymbol}}") {
        _mint(msg.sender, {{.InitialSupply}} * 10**decimals());
    }
}`,
			contractName: "MyToken",
			expectError:  false,
		},
		{
			name:         "invalid_solidity_syntax",
			templateCode: "invalid solidity code {{}",
			contractName: "Test",
			expectError:  true,
			errorMsg:     "Error rendering template",
		},
		{
			name: "missing_pragma",
			templateCode: `contract Test {
    constructor() {}
}`,
			contractName: "Test",
			expectError:  true,
			errorMsg:     "Solidity compilation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh database for each test case
			templateService := setupTestDatabase(t)
			createTemplateTool := NewCreateTemplateTool(templateService)
			handler := createTemplateTool.GetHandler()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"name":            "Test Template",
						"description":     "Test description",
						"chain_type":      "ethereum",
						"contract_name":   tt.contractName,
						"template_code":   tt.templateCode,
						"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
					},
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				assert.NoError(t, err) // Handler returns success with error content
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				assert.Len(t, result.Content, 1)
				textContent0 := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent0.Text, tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				// Verify template was created in database
				testUser := "test-user"
				templates, err := templateService.ListTemplates(&testUser, "", "", 10)
				assert.NoError(t, err)
				assert.Len(t, templates, 1)
				if len(templates) > 0 {
					assert.Equal(t, "ethereum", string(templates[0].ChainType))
				}
			}
		})
	}
}

// Removed TestCreateTemplateHandler_SolanaTemplateValidation - Solana validation is skipped

func TestCreateTemplateHandler_MetadataValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name             string
		templateMetadata string
		expectError      bool
		errorMsg         string
		expectedParams   int
	}{
		{
			name:             "valid_metadata",
			templateMetadata: `{"TokenName": "", "TokenSymbol": "", "InitialSupply": ""}`,
			expectError:      false,
			expectedParams:   3,
		},
		{
			name:             "empty_metadata",
			templateMetadata: "",
			expectError:      false,
			expectedParams:   0,
		},
		{
			name:             "empty_json_object",
			templateMetadata: "{}",
			expectError:      false,
			expectedParams:   0,
		},
		{
			name:             "invalid_json",
			templateMetadata: `{invalid json}`,
			expectError:      true,
			errorMsg:         "Invalid template_metadata JSON",
		},
		{
			name:             "non_empty_string_values",
			templateMetadata: `{"TokenName": "MyToken", "TokenSymbol": ""}`,
			expectError:      true,
			errorMsg:         "Metadata values must be empty strings for parameter definitions",
		},
		{
			name:             "non_string_values",
			templateMetadata: `{"TokenName": "", "TokenSymbol": "", "InitialSupply": 1000}`,
			expectError:      true,
			errorMsg:         "Metadata values must be empty strings for parameter definitions",
		},
		{
			name:             "empty_key",
			templateMetadata: `{"": "", "TokenSymbol": ""}`,
			expectError:      true,
			errorMsg:         "Metadata keys cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh database for each test case
			templateService := setupTestDatabase(t)
			createTemplateTool := NewCreateTemplateTool(templateService)
			handler := createTemplateTool.GetHandler()

			args := map[string]interface{}{
				"name":            "Test Template",
				"description":     "Test description",
				"chain_type":      "ethereum",
				"contract_name":   "SimpleToken",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			}

			if tt.templateMetadata != "" {
				args["template_metadata"] = tt.templateMetadata
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: args,
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				assert.NoError(t, err) // Handler returns success with error content
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				assert.Len(t, result.Content, 1)
				textContent0 := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent0.Text, tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.False(t, result.IsError)

				// Parse result JSON to verify metadata handling
				assert.Len(t, result.Content, 2)
				textContent1 := result.Content[1].(mcp.TextContent)

				var resultData map[string]interface{}
				err = json.Unmarshal([]byte(textContent1.Text), &resultData)
				assert.NoError(t, err)

				if tt.expectedParams > 0 {
					assert.Equal(t, float64(tt.expectedParams), resultData["template_parameters"])
					assert.NotNil(t, resultData["metadata"])
				}

				// Verify template was created in database
				testUser := "test-user"
				templates, err := templateService.ListTemplates(&testUser, "", "", 10)
				assert.NoError(t, err)
				assert.Len(t, templates, 1)

				if tt.expectedParams > 0 {
					assert.NotEmpty(t, templates[0].Metadata)
					assert.Len(t, templates[0].Metadata, tt.expectedParams)
				}
			}
		})
	}
}

func TestCreateTemplateHandler_DatabaseIntegration(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)
	createTemplateTool := NewCreateTemplateTool(templateService)
	handler := createTemplateTool.GetHandler()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":              "Integration Test Template",
				"description":       "This is a test template for database integration",
				"chain_type":        "ethereum",
				"contract_name":     "SimpleToken",
				"template_code":     validEthereumTemplate(),
				"template_metadata": `{"TokenName": "", "TokenSymbol": "", "InitialSupply": ""}`,
				"template_values":   map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify success response format
	assert.Len(t, result.Content, 2)
	textContent0 := result.Content[0].(mcp.TextContent)
	textContent1 := result.Content[1].(mcp.TextContent)
	assert.Contains(t, textContent0.Text, "Template created successfully")
	assert.Contains(t, textContent0.Text, "(Solidity compilation validated)")

	// Parse result JSON
	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &resultData)
	assert.NoError(t, err)
	assert.Equal(t, "Integration Test Template", resultData["name"])
	assert.Equal(t, "This is a test template for database integration", resultData["description"])
	assert.Equal(t, "ethereum", resultData["chain_type"])
	assert.Equal(t, "Template created successfully", resultData["message"])
	assert.Equal(t, "success", resultData["compilation_status"])
	assert.Equal(t, float64(3), resultData["template_parameters"])
	assert.NotNil(t, resultData["metadata"])
	assert.NotNil(t, resultData["id"])
	assert.NotNil(t, resultData["created_at"])

	// Verify template exists in database
	testUser := "test-user"
	templates, err := templateService.ListTemplates(&testUser, "", "", 10)
	assert.NoError(t, err)
	assert.Len(t, templates, 1)

	template := templates[0]
	assert.Equal(t, "Integration Test Template", template.Name)
	assert.Equal(t, "This is a test template for database integration", template.Description)
	assert.Equal(t, models.TransactionChainType("ethereum"), template.ChainType)
	assert.Equal(t, validEthereumTemplate(), template.TemplateCode)
	assert.Len(t, template.Metadata, 3)
	assert.Contains(t, template.Metadata, "TokenName")
	assert.Contains(t, template.Metadata, "TokenSymbol")
	assert.Contains(t, template.Metadata, "InitialSupply")
}

func TestCreateTemplateHandler_MultipleTemplates(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)
	createTemplateTool := NewCreateTemplateTool(templateService)
	handler := createTemplateTool.GetHandler()

	// Create first template
	request1 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":            "Ethereum Template",
				"description":     "Ethereum ERC20 Template",
				"chain_type":      "ethereum",
				"contract_name":   "SimpleToken",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
		},
	}

	result1, err := handler(ctx, request1)
	assert.NoError(t, err)
	assert.NotNil(t, result1)

	// Create second template
	request2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":            "Solana Template",
				"description":     "Solana SPL Token Template",
				"chain_type":      "solana",
				"contract_name":   "spl_token",
				"template_code":   validSolanaTemplate(),
				"template_values": map[string]interface{}{"TokenName": "Test", "TokenSymbol": "TST", "InitialSupply": "1000"},
			},
		},
	}

	result2, err := handler(ctx, request2)
	assert.NoError(t, err)
	assert.NotNil(t, result2)

	// Verify both templates exist
	testUser := "test-user"
	templates, err := templateService.ListTemplates(&testUser, "", "", 10)
	assert.NoError(t, err)
	assert.Len(t, templates, 2)

	// Find templates by name
	var ethTemplate, solTemplate *models.Template
	for i := range templates {
		switch templates[i].Name {
		case "Ethereum Template":
			ethTemplate = &templates[i]
		case "Solana Template":
			solTemplate = &templates[i]
		}
	}

	assert.NotNil(t, ethTemplate)
	assert.NotNil(t, solTemplate)
	assert.Equal(t, models.TransactionChainType("ethereum"), ethTemplate.ChainType)
	assert.Equal(t, models.TransactionChainType("solana"), solTemplate.ChainType)
}

// Helper functions for test templates
func validEthereumTemplate() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SimpleToken {
    string public name = "{{.TokenName}}";
    string public symbol = "{{.TokenSymbol}}";
    uint256 public totalSupply = {{.InitialSupply}};
    
    mapping(address => uint256) public balanceOf;
    
    constructor() {
        balanceOf[msg.sender] = totalSupply;
    }
    
    function transfer(address to, uint256 amount) external returns (bool) {
        require(balanceOf[msg.sender] >= amount, "Insufficient balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
}`
}

func validSolanaTemplate() string {
	return `use anchor_lang::prelude::*;

#[program]
pub mod spl_token {
    use super::*;
    
    pub fn initialize(
        ctx: Context<Initialize>,
        name: String,
        symbol: String,
        decimals: u8,
    ) -> Result<()> {
        let token = &mut ctx.accounts.token;
        token.name = name;
        token.symbol = symbol;
        token.decimals = decimals;
        token.total_supply = 0;
        token.authority = ctx.accounts.authority.key();
        Ok(())
    }
    
    pub fn mint(ctx: Context<Mint>, amount: u64) -> Result<()> {
        let token = &mut ctx.accounts.token;
        token.total_supply += amount;
        Ok(())
    }
}

#[derive(Accounts)]
pub struct Initialize<'info> {
    #[account(init, payer = authority, space = 8 + 32 + 4 + 10 + 4 + 10 + 1 + 8 + 32)]
    pub token: Account<'info, Token>,
    #[account(mut)]
    pub authority: Signer<'info>,
    pub system_program: Program<'info, System>,
}

#[derive(Accounts)]
pub struct Mint<'info> {
    #[account(mut, has_one = authority)]
    pub token: Account<'info, Token>,
    pub authority: Signer<'info>,
}

#[account]
pub struct Token {
    pub name: String,
    pub symbol: String,
    pub decimals: u8,
    pub total_supply: u64,
    pub authority: Pubkey,
}`
}
