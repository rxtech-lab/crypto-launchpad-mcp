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

func TestCreateTemplateHandler_EthereumABIStorage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		templateCode string
		contractName string
		expectABI    bool
		abiChecks    func(t *testing.T, abi models.JSON)
	}{
		{
			name:         "simple_ethereum_contract_with_abi",
			templateCode: validEthereumTemplate(),
			contractName: "SimpleToken",
			expectABI:    true,
			abiChecks: func(t *testing.T, abi models.JSON) {
				// Verify ABI is not empty
				assert.NotEmpty(t, abi)

				// The ABI is now stored as {"abi": [...]}
				abiArrayInterface, exists := abi["abi"]
				assert.True(t, exists, "ABI should contain 'abi' key")

				// Convert the ABI array to JSON and parse it
				abiBytes, err := json.Marshal(abiArrayInterface)
				assert.NoError(t, err)

				// Parse ABI as array of function definitions
				var abiArray []map[string]interface{}
				err = json.Unmarshal(abiBytes, &abiArray)
				assert.NoError(t, err)
				assert.NotEmpty(t, abiArray)

				// Check for expected functions in ABI
				functionNames := make([]string, 0)
				for _, item := range abiArray {
					if itemType, ok := item["type"].(string); ok && itemType == "function" {
						if name, ok := item["name"].(string); ok {
							functionNames = append(functionNames, name)
						}
					}
				}

				// Verify expected functions exist
				assert.Contains(t, functionNames, "transfer")
				assert.Contains(t, functionNames, "name")
				assert.Contains(t, functionNames, "symbol")
				assert.Contains(t, functionNames, "totalSupply")
				assert.Contains(t, functionNames, "balanceOf")
			},
		},
		{
			name: "openzeppelin_contract_with_abi",
			templateCode: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import "@openzeppelin-contracts/contracts/token/ERC20/ERC20.sol";

contract MyToken is ERC20 {
    constructor() ERC20("{{.TokenName}}", "{{.TokenSymbol}}") {
        _mint(msg.sender, {{.InitialSupply}} * 10**decimals());
    }
}`,
			contractName: "MyToken",
			expectABI:    true,
			abiChecks: func(t *testing.T, abi models.JSON) {
				// Verify ABI is not empty
				assert.NotEmpty(t, abi)

				// The ABI is now stored as {"abi": [...]}
				abiArrayInterface, exists := abi["abi"]
				assert.True(t, exists, "ABI should contain 'abi' key")

				// Convert the ABI array to JSON and parse it
				abiBytes, err := json.Marshal(abiArrayInterface)
				assert.NoError(t, err)

				// Parse ABI as array of function definitions
				var abiArray []map[string]interface{}
				err = json.Unmarshal(abiBytes, &abiArray)
				assert.NoError(t, err)
				assert.NotEmpty(t, abiArray)

				// Check for expected ERC20 functions in ABI
				functionNames := make([]string, 0)
				for _, item := range abiArray {
					if itemType, ok := item["type"].(string); ok && itemType == "function" {
						if name, ok := item["name"].(string); ok {
							functionNames = append(functionNames, name)
						}
					}
				}

				// Verify standard ERC20 functions exist
				assert.Contains(t, functionNames, "transfer")
				assert.Contains(t, functionNames, "transferFrom")
				assert.Contains(t, functionNames, "approve")
				assert.Contains(t, functionNames, "balanceOf")
				assert.Contains(t, functionNames, "allowance")
				assert.Contains(t, functionNames, "totalSupply")
			},
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
						"name":            "Test ABI Template",
						"description":     "Test template for ABI storage verification",
						"chain_type":      "ethereum",
						"contract_name":   tt.contractName,
						"template_code":   tt.templateCode,
						"template_values": map[string]interface{}{"TokenName": "TestToken", "TokenSymbol": "TTK", "InitialSupply": "1000"},
					},
				},
			}

			result, err := handler(ctx, request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.False(t, result.IsError)

			// Verify template was created successfully
			assert.Len(t, result.Content, 2)

			// Parse the result to get template ID
			textContent1 := result.Content[1].(mcp.TextContent)
			var resultData map[string]interface{}
			err = json.Unmarshal([]byte(textContent1.Text), &resultData)
			assert.NoError(t, err)

			templateID := uint(resultData["id"].(float64))

			// Retrieve template from database to verify ABI storage
			template, err := templateService.GetTemplateByID(templateID)
			assert.NoError(t, err)
			assert.NotNil(t, template)

			if tt.expectABI {
				// Verify ABI was stored
				assert.NotEmpty(t, template.Abi)

				// Run specific ABI checks
				tt.abiChecks(t, template.Abi)
			} else {
				// Verify no ABI was stored
				assert.Empty(t, template.Abi)
			}
		})
	}
}

func TestCreateTemplateHandler_ContractNameMismatch(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)
	createTemplateTool := NewCreateTemplateTool(templateService)
	handler := createTemplateTool.GetHandler()

	// Use a valid contract but specify wrong contract name
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":            "Test Mismatch Template",
				"description":     "Test template for contract name mismatch",
				"chain_type":      "ethereum",
				"contract_name":   "NonExistentContract",   // Wrong contract name
				"template_code":   validEthereumTemplate(), // Contains SimpleToken
				"template_values": map[string]interface{}{"TokenName": "TestToken", "TokenSymbol": "TTK", "InitialSupply": "1000"},
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)

	// Verify error message
	assert.Len(t, result.Content, 1)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "Contract NonExistentContract not found in the compilation result")
	assert.Contains(t, textContent.Text, "AvailableContracts are: SimpleToken")
}

func TestCreateTemplateHandler_ABIVerification(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)
	createTemplateTool := NewCreateTemplateTool(templateService)
	handler := createTemplateTool.GetHandler()

	// Create a template with known contract structure
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":            "ABI Verification Template",
				"description":     "Template for detailed ABI verification",
				"chain_type":      "ethereum",
				"contract_name":   "SimpleToken",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "VerifyToken", "TokenSymbol": "VTK", "InitialSupply": "5000"},
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse result to get template ID
	textContent1 := result.Content[1].(mcp.TextContent)
	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &resultData)
	assert.NoError(t, err)

	templateID := uint(resultData["id"].(float64))

	// Retrieve template from database
	template, err := templateService.GetTemplateByID(templateID)
	assert.NoError(t, err)
	assert.NotNil(t, template)

	// Verify ABI structure in detail
	assert.NotEmpty(t, template.Abi)

	// The ABI is now stored as {"abi": [...]}
	abiArrayInterface, exists := template.Abi["abi"]
	assert.True(t, exists, "ABI should contain 'abi' key")

	// Convert the ABI array to JSON for detailed parsing
	abiBytes, err := json.Marshal(abiArrayInterface)
	assert.NoError(t, err)

	var abiArray []map[string]interface{}
	err = json.Unmarshal(abiBytes, &abiArray)
	assert.NoError(t, err)

	// Count different types of ABI entries
	functionCount := 0
	constructorCount := 0
	variableCount := 0

	functionInputs := make(map[string][]map[string]interface{})
	functionOutputs := make(map[string][]map[string]interface{})

	for _, item := range abiArray {
		itemType, ok := item["type"].(string)
		assert.True(t, ok, "Each ABI item should have a type")

		switch itemType {
		case "function":
			functionCount++
			if name, ok := item["name"].(string); ok {
				if inputs, ok := item["inputs"].([]interface{}); ok {
					inputsTyped := make([]map[string]interface{}, len(inputs))
					for i, input := range inputs {
						inputsTyped[i] = input.(map[string]interface{})
					}
					functionInputs[name] = inputsTyped
				}
				if outputs, ok := item["outputs"].([]interface{}); ok {
					outputsTyped := make([]map[string]interface{}, len(outputs))
					for i, output := range outputs {
						outputsTyped[i] = output.(map[string]interface{})
					}
					functionOutputs[name] = outputsTyped
				}
			}
		case "constructor":
			constructorCount++
		default:
			// Public variables become function getters
			if name, ok := item["name"].(string); ok && name != "" {
				variableCount++
			}
		}
	}

	// Verify expected counts and function signatures
	assert.Greater(t, functionCount, 0, "Should have at least one function")
	assert.Equal(t, 1, constructorCount, "Should have exactly one constructor")

	// Verify specific function signatures
	// transfer function should have 2 inputs (address, uint256) and 1 output (bool)
	if transferInputs, exists := functionInputs["transfer"]; exists {
		assert.Len(t, transferInputs, 2, "transfer function should have 2 inputs")
		assert.Equal(t, "address", transferInputs[0]["type"], "First input should be address")
		assert.Equal(t, "uint256", transferInputs[1]["type"], "Second input should be uint256")
	}

	if transferOutputs, exists := functionOutputs["transfer"]; exists {
		assert.Len(t, transferOutputs, 1, "transfer function should have 1 output")
		assert.Equal(t, "bool", transferOutputs[0]["type"], "Output should be bool")
	}

	// Verify balanceOf function
	if balanceOfInputs, exists := functionInputs["balanceOf"]; exists {
		assert.Len(t, balanceOfInputs, 1, "balanceOf function should have 1 input")
		assert.Equal(t, "address", balanceOfInputs[0]["type"], "Input should be address")
	}

	if balanceOfOutputs, exists := functionOutputs["balanceOf"]; exists {
		assert.Len(t, balanceOfOutputs, 1, "balanceOf function should have 1 output")
		assert.Equal(t, "uint256", balanceOfOutputs[0]["type"], "Output should be uint256")
	}
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
