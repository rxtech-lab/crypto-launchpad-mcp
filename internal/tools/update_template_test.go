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
	assert.NotNil(t, handler)

	// Check that the tool has the expected properties
	assert.Contains(t, tool.InputSchema.Properties, "template_id")
	assert.Contains(t, tool.InputSchema.Properties, "description")
	assert.Contains(t, tool.InputSchema.Properties, "chain_type")
	assert.Contains(t, tool.InputSchema.Properties, "template_code")
	assert.Contains(t, tool.InputSchema.Properties, "template_metadata")
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
			name:            "valid_ethereum_to_solana",
			originalChain:   "ethereum",
			newChainType:    "solana",
			newTemplateCode: validSolanaTemplate(),
			expectError:     false,
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
    uint256 public supply = {{.Supply}};
}`,
			expectError: false,
		},
		{
			name:      "valid_solana_code_update",
			chainType: "solana",
			newCode: `use anchor_lang::prelude::*;

#[program]
pub mod updated_program {
    use super::*;
    
    pub fn initialize(ctx: Context<Initialize>) -> Result<()> {
        Ok(())
    }
}

#[derive(Accounts)]
pub struct Initialize {}`,
			expectError: false,
		},
		{
			name:        "invalid_ethereum_code",
			chainType:   "ethereum",
			newCode:     "invalid solidity code",
			expectError: true,
			errorMsg:    "Solidity compilation failed",
		},
		// Removed invalid_solana_code_missing_requirements - Solana validation is skipped
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
    uint256 public totalSupply = {{.InitialSupply}};
    address public owner = {{.Owner}};
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
