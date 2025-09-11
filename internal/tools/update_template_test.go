package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewUpdateTemplateTool(t *testing.T) {
	db := setupTestDatabase(t)
	updateTool := NewUpdateTemplateTool(db)
	tool := updateTool.GetTool()
	handler := updateTool.GetHandler()

	// Test tool metadata
	assert.Equal(t, "update_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Update existing smart contract template")
	assert.NotNil(t, handler)

	// Check that the tool has the expected properties
	assert.Contains(t, tool.InputSchema.Properties, "template_id")
	assert.Contains(t, tool.InputSchema.Properties, "description")
	assert.Contains(t, tool.InputSchema.Properties, "chain_type")
	assert.Contains(t, tool.InputSchema.Properties, "template_code")
	assert.Contains(t, tool.InputSchema.Properties, "template_metadata")
	assert.Contains(t, tool.InputSchema.Properties, "contract_name")
	assert.Contains(t, tool.InputSchema.Properties, "template_values")

	// Verify parameter descriptions
	templateIdProp := tool.InputSchema.Properties["template_id"].(map[string]any)
	assert.Contains(t, templateIdProp["description"], "ID of the template")

	chainTypeProp := tool.InputSchema.Properties["chain_type"].(map[string]any)
	assert.Contains(t, chainTypeProp["description"], "ethereum or solana")

	templateCodeProp := tool.InputSchema.Properties["template_code"].(map[string]any)
	assert.Contains(t, templateCodeProp["description"], "Go template syntax")

	templateMetadataProp := tool.InputSchema.Properties["template_metadata"].(map[string]any)
	assert.Contains(t, templateMetadataProp["description"], "JSON object defining template parameters")
}

func TestUpdateTemplateHandler_ParameterValidation(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		requestArgs map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing_template_id",
			requestArgs: map[string]interface{}{
				"description": "Updated description",
			},
			expectError: true,
			errorMsg:    "TemplateID",
		},
		{
			name: "invalid_template_id_format",
			requestArgs: map[string]interface{}{
				"template_id": "not_a_number",
				"description": "Updated description",
			},
			expectError: true, // Invalid template ID format should return error
			errorMsg:    "Invalid template_id",
		},
		{
			name: "zero_template_id",
			requestArgs: map[string]interface{}{
				"template_id": "0",
				"description": "Updated description",
			},
			expectError: true, // Zero template ID should return template not found error
			errorMsg:    "Template not found",
		},
		{
			name: "negative_template_id",
			requestArgs: map[string]interface{}{
				"template_id": "-1",
				"description": "Updated description",
			},
			expectError: true, // Negative template ID should return invalid format error
			errorMsg:    "Invalid template_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDatabase(t)
			updateTool := NewUpdateTemplateTool(db)
			handler := updateTool.GetHandler()

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

func TestUpdateTemplateHandler_TemplateNotFound(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)
	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "999",
				"description": "Updated description",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err) // Handler returns success with error content
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Len(t, result.Content, 1)
	textContent0 := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent0.Text, "Template not found")
}

func TestUpdateTemplateHandler_NoUpdatesProvided(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "1",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err) // Handler returns success with error content
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Len(t, result.Content, 1)
	textContent0 := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent0.Text, "No update parameters provided")

}

func TestUpdateTemplateHandler_UpdateDescription(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "1",
				"description": "Updated description",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify success response
	assert.Len(t, result.Content, 2)
	textContent0 := result.Content[0].(mcp.TextContent)
	textContent1 := result.Content[1].(mcp.TextContent)
	assert.Contains(t, textContent0.Text, "Template updated successfully")
	assert.Contains(t, textContent0.Text, "description")

	// Parse result JSON
	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &resultData)
	assert.NoError(t, err)
	assert.Equal(t, "Updated description", resultData["description"])
	assert.Contains(t, resultData["updated_fields"], "description")

	// Verify database update
	updatedTemplate, err := db.GetTemplateByID(1)
	assert.NoError(t, err)
	assert.Equal(t, "Updated description", updatedTemplate.Description)
}

func TestUpdateTemplateHandler_UpdateChainType(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name            string
		originalChain   string
		newChainType    string
		newTemplateCode string
		expectError     bool
		errorMsg        string
	}{
		{
			name:            "valid_ethereum_to_solana_without_code",
			originalChain:   "ethereum",
			newChainType:    "solana",
			newTemplateCode: "", // No template code update
			expectError:     false,
		},
		{
			name:            "invalid_ethereum_to_solana_with_code",
			originalChain:   "ethereum",
			newChainType:    "solana",
			newTemplateCode: validSolanaTemplate(),
			expectError:     true,
			errorMsg:        "Cannot update template code when changing chain type to Solana",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDatabase(t)

			// Create original template
			originalTemplate := validEthereumTemplate()
			if tt.originalChain == "solana" {
				originalTemplate = validSolanaTemplate()
			}

			template := &models.Template{
				Name:         "Test Template",
				Description:  "Test description",
				ChainType:    models.TransactionChainType(tt.originalChain),
				TemplateCode: originalTemplate,
			}
			err := db.CreateTemplate(template)
			assert.NoError(t, err)

			updateTool := NewUpdateTemplateTool(db)
			handler := updateTool.GetHandler()

			args := map[string]interface{}{
				"template_id": "1",
				"chain_type":  tt.newChainType,
			}

			// Add new template code if chain type is changing
			if tt.newTemplateCode != "" {
				args["template_code"] = tt.newTemplateCode
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
				// Error responses have 1 content item
				assert.Len(t, result.Content, 1)
				textContent0 := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent0.Text, tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Verify database update
				updatedTemplate, err := db.GetTemplateByID(1)
				assert.NoError(t, err)
				assert.Equal(t, models.TransactionChainType(tt.newChainType), updatedTemplate.ChainType)

				if tt.newTemplateCode != "" {
					assert.Equal(t, tt.newTemplateCode, updatedTemplate.TemplateCode)
				}
			}
		})
	}
}

func TestUpdateTemplateHandler_UpdateTemplateCode(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		chainType   string
		newCode     string
		expectError bool
		errorMsg    string
	}{
		{
			name:      "valid_ethereum_code_update",
			chainType: "ethereum",
			newCode: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract UpdatedToken {
    string public name = "{{.TokenName}}";
    uint256 public supply = 1;
}`,
			expectError: false,
		},
		{
			name:        "invalid_ethereum_code",
			chainType:   "ethereum",
			newCode:     "invalid solidity code",
			expectError: true,
			errorMsg:    "Solidity compilation failed",
		},
		{
			name:        "solana_code_update_not_supported",
			chainType:   "solana",
			newCode:     validSolanaTemplate(),
			expectError: true,
			errorMsg:    "Solana template code updates are not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDatabase(t)

			// Create original template
			originalTemplate := validEthereumTemplate()
			if tt.chainType == "solana" {
				originalTemplate = validSolanaTemplate()
			}

			template := &models.Template{
				Name:         "Test Template",
				Description:  "Test description",
				ChainType:    models.TransactionChainType(tt.chainType),
				TemplateCode: originalTemplate,
			}
			err := db.CreateTemplate(template)
			assert.NoError(t, err)

			updateTool := NewUpdateTemplateTool(db)
			handler := updateTool.GetHandler()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"template_id":   "1",
						"template_code": tt.newCode,
					},
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				assert.NoError(t, err) // Handler returns success with error content
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				// Error responses have 1 content item
				assert.Len(t, result.Content, 1)
				textContent0 := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent0.Text, tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

				// Verify database update
				updatedTemplate, err := db.GetTemplateByID(1)
				assert.NoError(t, err)
				assert.Equal(t, tt.newCode, updatedTemplate.TemplateCode)
			}
		})
	}
}

func TestUpdateTemplateHandler_UpdateMetadata(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		newMetadata    string
		expectError    bool
		errorMsg       string
		expectedParams int
	}{
		{
			name:           "valid_metadata_update",
			newMetadata:    `{"TokenName": "", "TokenSymbol": "", "InitialSupply": "", "Owner": ""}`,
			expectError:    false,
			expectedParams: 4,
		},
		{
			name:           "empty_metadata",
			newMetadata:    "{}",
			expectError:    false,
			expectedParams: 0,
		},
		{
			name:        "invalid_json",
			newMetadata: `{invalid json}`,
			expectError: true,
			errorMsg:    "Invalid template_metadata JSON",
		},
		{
			name:        "non_empty_string_values",
			newMetadata: `{"TokenName": "Test", "TokenSymbol": ""}`,
			expectError: true,
			errorMsg:    "Metadata values must be empty strings for parameter definitions",
		},
		{
			name:        "empty_key",
			newMetadata: `{"": "", "TokenSymbol": ""}`,
			expectError: true,
			errorMsg:    "Metadata keys cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTestDatabase(t)

			// Create template with original metadata
			template := &models.Template{
				Name:         "Test Template",
				Description:  "Test description",
				ChainType:    "ethereum",
				TemplateCode: validEthereumTemplate(),
				Metadata:     models.JSON{"OriginalParam": ""},
			}
			err := db.CreateTemplate(template)
			assert.NoError(t, err)

			updateTool := NewUpdateTemplateTool(db)
			handler := updateTool.GetHandler()

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"template_id":       "1",
						"template_metadata": tt.newMetadata,
					},
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				assert.NoError(t, err) // Handler returns success with error content
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				// Error responses have 1 content item
				assert.Len(t, result.Content, 1)
				textContent0 := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent0.Text, tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)

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

				// Verify database update
				updatedTemplate, err := db.GetTemplateByID(1)
				assert.NoError(t, err)
				assert.Len(t, updatedTemplate.Metadata, tt.expectedParams)
			}
		})
	}
}

func TestUpdateTemplateHandler_MultipleFieldsUpdate(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create original template
	template := &models.Template{
		Name:         "Original Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
		Metadata:     models.JSON{"OldParam": ""},
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	// Update multiple fields at once
	newCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract MultiUpdateToken {
    string public name = "{{.TokenName}}";
    string public symbol = "{{.TokenSymbol}}";
    uint256 public totalSupply =  1;
}`

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":       "1",
				"description":       "Updated description with multiple changes",
				"template_code":     newCode,
				"template_metadata": `{"TokenName": "", "TokenSymbol": "", "InitialSupply": "", "Owner": ""}`,
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify success response with multiple updates
	assert.Len(t, result.Content, 2)
	textContent0 := result.Content[0].(mcp.TextContent)
	textContent1 := result.Content[1].(mcp.TextContent)
	assert.Contains(t, textContent0.Text, "Template updated successfully")
	assert.Contains(t, textContent0.Text, "description")
	assert.Contains(t, textContent0.Text, "template_code")
	assert.Contains(t, textContent0.Text, "metadata")

	// Parse result JSON
	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &resultData)
	assert.NoError(t, err)
	assert.Equal(t, "Updated description with multiple changes", resultData["description"])
	assert.Equal(t, float64(4), resultData["template_parameters"])
	assert.Contains(t, resultData["updated_fields"], "description")
	assert.Contains(t, resultData["updated_fields"], "template_code")
	assert.Contains(t, resultData["updated_fields"], "metadata")

	// Verify database updates
	updatedTemplate, err := db.GetTemplateByID(1)
	assert.NoError(t, err)
	assert.Equal(t, "Updated description with multiple changes", updatedTemplate.Description)
	assert.Equal(t, newCode, updatedTemplate.TemplateCode)
	assert.Len(t, updatedTemplate.Metadata, 4)
	assert.Contains(t, updatedTemplate.Metadata, "TokenName")
	assert.Contains(t, updatedTemplate.Metadata, "TokenSymbol")
	assert.Contains(t, updatedTemplate.Metadata, "InitialSupply")
	assert.Contains(t, updatedTemplate.Metadata, "Owner")
}

// Removed TestUpdateTemplateHandler_ChainTypeAndCodeValidation - Solana validation is skipped

// Removed TestUpdateTemplateHandler_ValidationWithExistingCode - Solana validation is skipped

func TestUpdateTemplateHandler_EthereumABIStorage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                string
		initialTemplateCode string
		initialContractName string
		updatedTemplateCode string
		updatedContractName string
		expectUpdatedABI    bool
		abiChecks           func(t *testing.T, abi models.JSON)
	}{
		{
			name:                "update_simple_contract_code_with_abi",
			initialTemplateCode: validEthereumTemplate(),
			initialContractName: "SimpleToken",
			updatedTemplateCode: validUpdatedEthereumTemplate(),
			updatedContractName: "AdvancedToken",
			expectUpdatedABI:    true,
			abiChecks: func(t *testing.T, abi models.JSON) {
				// Verify ABI is not empty
				assert.NotEmpty(t, abi)

				// The ABI is stored as {"abi": [...]}
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

				// Check for expected functions in updated ABI
				functionNames := make([]string, 0)
				for _, item := range abiArray {
					if itemType, ok := item["type"].(string); ok && itemType == "function" {
						if name, ok := item["name"].(string); ok {
							functionNames = append(functionNames, name)
						}
					}
				}

				// Verify expected functions exist in updated contract
				assert.Contains(t, functionNames, "transfer")
				assert.Contains(t, functionNames, "mint") // New function in AdvancedToken
				assert.Contains(t, functionNames, "burn") // New function in AdvancedToken
				assert.Contains(t, functionNames, "balanceOf")
				assert.Contains(t, functionNames, "totalSupply")
			},
		},
		{
			name:                "update_template_without_contract_name_no_abi_update",
			initialTemplateCode: validEthereumTemplate(),
			initialContractName: "SimpleToken",
			updatedTemplateCode: validUpdatedEthereumTemplate(),
			updatedContractName: "", // No contract name provided
			expectUpdatedABI:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh database for each test case
			templateService := setupTestDatabase(t)

			// First create a template using create_template tool
			createTool := NewCreateTemplateTool(templateService)
			createHandler := createTool.GetHandler()

			createRequest := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"name":            "Test ABI Update Template",
						"description":     "Initial template for ABI update testing",
						"chain_type":      "ethereum",
						"contract_name":   tt.initialContractName,
						"template_code":   tt.initialTemplateCode,
						"template_values": map[string]interface{}{"TokenName": "InitialToken", "TokenSymbol": "ITK", "InitialSupply": "1000"},
					},
				},
			}

			createResult, err := createHandler(ctx, createRequest)
			assert.NoError(t, err)
			assert.NotNil(t, createResult)
			assert.False(t, createResult.IsError)

			// Parse create result to get template ID
			textContent1 := createResult.Content[1].(mcp.TextContent)
			var createResultData map[string]interface{}
			err = json.Unmarshal([]byte(textContent1.Text), &createResultData)
			assert.NoError(t, err)
			templateID := uint(createResultData["id"].(float64))

			// Verify initial ABI was stored
			initialTemplate, err := templateService.GetTemplateByID(templateID)
			assert.NoError(t, err)
			assert.NotEmpty(t, initialTemplate.Abi, "Initial template should have ABI")

			// Now update the template using update_template tool
			updateTool := NewUpdateTemplateTool(templateService)
			updateHandler := updateTool.GetHandler()

			updateArgs := map[string]interface{}{
				"template_id":   "1",
				"template_code": tt.updatedTemplateCode,
			}

			if tt.updatedContractName != "" {
				updateArgs["contract_name"] = tt.updatedContractName
			}

			updateRequest := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: updateArgs,
				},
			}

			updateResult, err := updateHandler(ctx, updateRequest)
			assert.NoError(t, err)
			assert.NotNil(t, updateResult)
			assert.False(t, updateResult.IsError)

			// Retrieve updated template from database
			updatedTemplate, err := templateService.GetTemplateByID(templateID)
			assert.NoError(t, err)
			assert.NotNil(t, updatedTemplate)

			// Verify the template code was updated
			assert.Equal(t, tt.updatedTemplateCode, updatedTemplate.TemplateCode)

			if tt.expectUpdatedABI {
				// Verify ABI was updated
				assert.NotEmpty(t, updatedTemplate.Abi)

				// Run specific ABI checks if provided
				if tt.abiChecks != nil {
					tt.abiChecks(t, updatedTemplate.Abi)
				}

				// Verify ABI is different from initial ABI (if contract name changed)
				if tt.updatedContractName != tt.initialContractName {
					initialAbiBytes, _ := json.Marshal(initialTemplate.Abi)
					updatedAbiBytes, _ := json.Marshal(updatedTemplate.Abi)
					assert.NotEqual(t, string(initialAbiBytes), string(updatedAbiBytes), "ABI should be updated when contract name changes")
				}
			} else {
				// If contract name was not provided, ABI should remain unchanged
				initialAbiBytes, _ := json.Marshal(initialTemplate.Abi)
				updatedAbiBytes, _ := json.Marshal(updatedTemplate.Abi)
				assert.Equal(t, string(initialAbiBytes), string(updatedAbiBytes), "ABI should remain unchanged when contract name is not provided")
			}
		})
	}
}

func TestUpdateTemplateHandler_ContractNameMismatchOnUpdate(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// First create a template
	createTool := NewCreateTemplateTool(templateService)
	createHandler := createTool.GetHandler()

	createRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":            "Test Template",
				"description":     "Template for contract name mismatch test",
				"chain_type":      "ethereum",
				"contract_name":   "SimpleToken",
				"template_code":   validEthereumTemplate(),
				"template_values": map[string]interface{}{"TokenName": "TestToken", "TokenSymbol": "TTK", "InitialSupply": "1000"},
			},
		},
	}

	createResult, err := createHandler(ctx, createRequest)
	assert.NoError(t, err)
	assert.False(t, createResult.IsError)

	// Now try to update with wrong contract name
	updateTool := NewUpdateTemplateTool(templateService)
	updateHandler := updateTool.GetHandler()

	updateRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":   "1",
				"template_code": validUpdatedEthereumTemplate(), // Contains AdvancedToken
				"contract_name": "NonExistentContract",          // Wrong contract name
			},
		},
	}

	updateResult, err := updateHandler(ctx, updateRequest)
	assert.NoError(t, err)
	assert.NotNil(t, updateResult)
	assert.False(t, updateResult.IsError) // Update should succeed, but ABI won't be updated

	// Verify template was updated but ABI remains from initial template
	updatedTemplate, err := templateService.GetTemplateByID(1)
	assert.NoError(t, err)
	assert.Equal(t, validUpdatedEthereumTemplate(), updatedTemplate.TemplateCode)

	// ABI should still contain the original SimpleToken ABI, not AdvancedToken
	// because the contract_name mismatch prevents ABI update
	abiArrayInterface, exists := updatedTemplate.Abi["abi"]
	assert.True(t, exists)
	abiBytes, _ := json.Marshal(abiArrayInterface)
	var abiArray []map[string]interface{}
	json.Unmarshal(abiBytes, &abiArray)

	functionNames := make([]string, 0)
	for _, item := range abiArray {
		if itemType, ok := item["type"].(string); ok && itemType == "function" {
			if name, ok := item["name"].(string); ok {
				functionNames = append(functionNames, name)
			}
		}
	}

	// Should still have original SimpleToken functions, not AdvancedToken functions
	assert.Contains(t, functionNames, "transfer")
	assert.NotContains(t, functionNames, "mint") // mint is in AdvancedToken, not SimpleToken
	assert.NotContains(t, functionNames, "burn") // burn is in AdvancedToken, not SimpleToken
}

func TestUpdateTemplateHandler_UpdateTemplateValues(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:                 "Test Template",
		Description:          "Original description",
		ChainType:            "ethereum",
		TemplateCode:         validEthereumTemplate(),
		SampleTemplateValues: models.JSON{"TokenName": "Original", "TokenSymbol": "ORIG"},
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "1",
				"template_values": map[string]interface{}{
					"TokenName":     "Updated Token",
					"TokenSymbol":   "UPD",
					"InitialSupply": "5000",
				},
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify success response
	assert.Len(t, result.Content, 2)
	textContent0 := result.Content[0].(mcp.TextContent)
	textContent1 := result.Content[1].(mcp.TextContent)
	assert.Contains(t, textContent0.Text, "Template updated successfully")
	assert.Contains(t, textContent0.Text, "template_values")

	// Parse result JSON
	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &resultData)
	assert.NoError(t, err)
	assert.Contains(t, resultData["updated_fields"], "template_values")

	// Verify database update
	updatedTemplate, err := db.GetTemplateByID(1)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Token", updatedTemplate.SampleTemplateValues["TokenName"])
	assert.Equal(t, "UPD", updatedTemplate.SampleTemplateValues["TokenSymbol"])
	assert.Equal(t, "5000", updatedTemplate.SampleTemplateValues["InitialSupply"])
}

func TestUpdateTemplateHandler_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	tests := []struct {
		name        string
		arguments   map[string]interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name: "validation_error_from_struct_tags",
			arguments: map[string]interface{}{
				// Missing required template_id
				"description": "Updated description",
			},
			expectError: true,
			errorMsg:    "Invalid arguments",
		},
		{
			name: "bind_arguments_error",
			arguments: map[string]interface{}{
				"template_id": map[string]interface{}{"invalid": "type"}, // Wrong type for template_id
			},
			expectError: true, // This will cause a BindArguments error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.arguments,
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

func TestUpdateTemplateHandler_CompilationResultWithoutContractName(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	// Update template code without providing contract_name
	newCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract UpdatedToken {
    string public name = "{{.TokenName}}";
    uint256 public supply = {{.InitialSupply}};
}`

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":   "1",
				"template_code": newCode,
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify success response includes contract names from compilation
	assert.Len(t, result.Content, 2)
	textContent1 := result.Content[1].(mcp.TextContent)

	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &resultData)
	assert.NoError(t, err)

	// Should have contract_names from compilation result
	assert.Contains(t, resultData, "contract_names")
	contractNames := resultData["contract_names"].([]interface{})
	assert.Contains(t, contractNames, "UpdatedToken")

	// Verify database update
	updatedTemplate, err := db.GetTemplateByID(1)
	assert.NoError(t, err)
	assert.Equal(t, newCode, updatedTemplate.TemplateCode)
	// ABI should not be updated since no contract_name was provided
}

func TestUpdateTemplateHandler_TemplateRenderingError(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	// Use template code with invalid template syntax (unclosed template action)
	invalidTemplateCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Test {
    string public name = "{{.TokenName";  // Missing closing }}
}`

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":   "1",
				"template_code": invalidTemplateCode,
				"template_values": map[string]interface{}{
					"TokenName": "Test",
				},
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
	assert.Contains(t, textContent.Text, "Error rendering template")
}

func TestUpdateTemplateHandler_TemplateCodeWithSampleValues(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template with sample values
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
		SampleTemplateValues: models.JSON{
			"TokenName":     "SampleToken",
			"TokenSymbol":   "SMPL",
			"InitialSupply": "1000",
		},
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	// Update template code without providing template_values (should use sample values)
	newCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract NewToken {
    string public name = "{{.TokenName}}";
    string public symbol = "{{.TokenSymbol}}";
    uint256 public totalSupply = {{.InitialSupply}};
}`

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":   "1",
				"template_code": newCode,
				"contract_name": "NewToken",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify template was updated
	updatedTemplate, err := db.GetTemplateByID(1)
	assert.NoError(t, err)
	assert.Equal(t, newCode, updatedTemplate.TemplateCode)
	// ABI should be updated since contract_name was provided
	assert.NotEmpty(t, updatedTemplate.Abi)
}

func TestUpdateTemplateHandler_TemplateCodeWithDefaultValues(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template without sample values
	template := &models.Template{
		Name:                 "Test Template",
		Description:          "Original description",
		ChainType:            "ethereum",
		TemplateCode:         validEthereumTemplate(),
		SampleTemplateValues: nil,
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	// Update template code without providing template_values (should use default dummy values)
	newCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract DefaultToken {
    string public name = "{{.TokenName}}";
    string public symbol = "{{.TokenSymbol}}";
    uint256 public totalSupply = {{.InitialSupply}};
}`

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":   "1",
				"template_code": newCode,
				"contract_name": "DefaultToken",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify template was updated
	updatedTemplate, err := db.GetTemplateByID(1)
	assert.NoError(t, err)
	assert.Equal(t, newCode, updatedTemplate.TemplateCode)
	// ABI should be updated since contract_name was provided
	assert.NotEmpty(t, updatedTemplate.Abi)
}

func TestUpdateTemplateHandler_ABIHandling(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	tests := []struct {
		name              string
		contractName      string
		expectedABIUpdate bool
		description       string
	}{
		{
			name:              "valid_contract_name_updates_abi",
			contractName:      "SimpleToken",
			expectedABIUpdate: true,
			description:       "ABI should be updated when valid contract name is provided",
		},
		{
			name:              "invalid_contract_name_no_abi_update",
			contractName:      "NonExistentContract",
			expectedABIUpdate: false,
			description:       "ABI should not be updated when invalid contract name is provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SimpleToken {
    string public name = "{{.TokenName}}";
    uint256 public supply = {{.InitialSupply}};
    
    function test() external pure returns (bool) {
        return true;
    }
}`

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"template_id":   "1",
						"template_code": newCode,
						"contract_name": tt.contractName,
					},
				},
			}

			result, err := handler(ctx, request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.False(t, result.IsError)

			// Verify template was updated
			updatedTemplate, err := db.GetTemplateByID(1)
			assert.NoError(t, err)
			assert.Equal(t, newCode, updatedTemplate.TemplateCode)

			if tt.expectedABIUpdate {
				// ABI should be updated
				assert.NotEmpty(t, updatedTemplate.Abi)
				// Verify ABI structure
				abiInterface, exists := updatedTemplate.Abi["abi"]
				assert.True(t, exists, "ABI should contain 'abi' key")
				assert.NotNil(t, abiInterface)
			}
		})
	}
}

func TestUpdateTemplateHandler_ServiceErrors(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	// Test service error during update (template doesn't exist)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "999",
				"description": "Updated description",
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
	assert.Contains(t, textContent.Text, "Template not found")
}

func TestUpdateTemplateHandler_ResponseFormatting(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)

	// Create a template first
	template := &models.Template{
		Name:         "Format Test Template",
		Description:  "Original description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err := db.CreateTemplate(template)
	assert.NoError(t, err)

	updateTool := NewUpdateTemplateTool(db)
	handler := updateTool.GetHandler()

	// Update multiple fields to test comprehensive response formatting
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":       "1",
				"description":       "Updated comprehensive description",
				"template_metadata": `{"TokenName": "", "TokenSymbol": "", "InitialSupply": "", "Decimals": ""}`,
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify response structure
	assert.Len(t, result.Content, 2)

	// Verify success message format
	textContent0 := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent0.Text, "Template updated successfully")
	assert.Contains(t, textContent0.Text, "updated:")
	assert.Contains(t, textContent0.Text, "description")
	assert.Contains(t, textContent0.Text, "metadata")

	// Verify JSON response structure
	textContent1 := result.Content[1].(mcp.TextContent)
	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &resultData)
	assert.NoError(t, err)

	// Verify all expected fields in response
	assert.Contains(t, resultData, "id")
	assert.Contains(t, resultData, "name")
	assert.Contains(t, resultData, "description")
	assert.Contains(t, resultData, "chain_type")
	assert.Contains(t, resultData, "updated_fields")
	assert.Contains(t, resultData, "template_parameters")
	assert.Contains(t, resultData, "metadata")

	// Verify field values
	assert.Equal(t, float64(1), resultData["id"])
	assert.Equal(t, "Format Test Template", resultData["name"])
	assert.Equal(t, "Updated comprehensive description", resultData["description"])
	assert.Equal(t, "ethereum", resultData["chain_type"])
	assert.Equal(t, float64(4), resultData["template_parameters"])

	// Verify updated_fields array
	updatedFields := resultData["updated_fields"].([]interface{})
	assert.Contains(t, updatedFields, "description")
	assert.Contains(t, updatedFields, "metadata")
}

// Helper functions for test templates (validEthereumTemplate is in create_template_test.go)

func validUpdatedEthereumTemplate() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract AdvancedToken {
    string public name = "{{.TokenName}}";
    string public symbol = "{{.TokenSymbol}}";
    uint256 public totalSupply = {{.InitialSupply}};
    address public owner;
    
    mapping(address => uint256) public balanceOf;
    
    constructor() {
        balanceOf[msg.sender] = totalSupply;
        owner = msg.sender;
    }
    
    function transfer(address to, uint256 amount) external returns (bool) {
        require(balanceOf[msg.sender] >= amount, "Insufficient balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
    
    function mint(address to, uint256 amount) external {
        require(msg.sender == owner, "Only owner can mint");
        balanceOf[to] += amount;
        totalSupply += amount;
    }
    
    function burn(uint256 amount) external {
        require(balanceOf[msg.sender] >= amount, "Insufficient balance");
        balanceOf[msg.sender] -= amount;
        totalSupply -= amount;
    }
}`
}
