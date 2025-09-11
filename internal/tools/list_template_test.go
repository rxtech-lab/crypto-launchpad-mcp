package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewListTemplateTool(t *testing.T) {
	templateService := setupTestDatabase(t)
	tool, handler := NewListTemplateTool(templateService)

	// Test tool metadata
	assert.Equal(t, "list_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "List predefined smart contract templates")
	assert.NotNil(t, handler)

	// Check that the tool has the expected properties
	assert.Contains(t, tool.InputSchema.Properties, "chain_type")
	assert.Contains(t, tool.InputSchema.Properties, "keyword")
	assert.Contains(t, tool.InputSchema.Properties, "limit")

	// Verify parameter descriptions
	chainTypeProp := tool.InputSchema.Properties["chain_type"].(map[string]any)
	assert.Contains(t, chainTypeProp["description"], "blockchain type")

	keywordProp := tool.InputSchema.Properties["keyword"].(map[string]any)
	assert.Contains(t, keywordProp["description"], "Search keyword")

	limitProp := tool.InputSchema.Properties["limit"].(map[string]any)
	assert.Contains(t, limitProp["description"], "Maximum number")
}

func TestListTemplateHandler_NoTemplates(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)
	_, handler := NewListTemplateTool(templateService)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse the response
	assert.Len(t, result.Content, 1)
	textContent := result.Content[0].(mcp.TextContent)
	assert.Contains(t, textContent.Text, "Templates listed:")

	var response map[string]interface{}
	jsonStr := textContent.Text[len("Templates listed: "):]
	err = json.Unmarshal([]byte(jsonStr), &response)
	assert.NoError(t, err)

	assert.Equal(t, float64(0), response["count"])
	assert.Equal(t, "No templates found matching the criteria", response["message"])
	templates := response["templates"].([]interface{})
	assert.Len(t, templates, 0)
}

func TestListTemplateHandler_WithTemplates(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// Create test templates
	template1 := &models.Template{
		Name:         "Ethereum ERC20",
		Description:  "Standard ERC20 token template",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
		Metadata:     models.JSON{"TokenName": "", "TokenSymbol": ""},
	}
	err := templateService.CreateTemplate(template1)
	assert.NoError(t, err)

	template2 := &models.Template{
		Name:         "Solana SPL Token",
		Description:  "Solana SPL token template",
		ChainType:    models.TransactionChainType("solana"),
		TemplateCode: validSolanaTemplate(),
		Metadata:     models.JSON{"TokenName": ""},
	}
	err = templateService.CreateTemplate(template2)
	assert.NoError(t, err)

	_, handler := NewListTemplateTool(templateService)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse the response
	assert.Len(t, result.Content, 2)
	textContent0 := result.Content[0].(mcp.TextContent)
	textContent1 := result.Content[1].(mcp.TextContent)
	assert.Equal(t, "Templates listed successfully: ", textContent0.Text)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &response)
	assert.NoError(t, err)

	assert.Equal(t, float64(2), response["count"])
	templates := response["templates"].([]interface{})
	assert.Len(t, templates, 2)

	// Verify template structure
	template := templates[0].(map[string]interface{})
	assert.Contains(t, template, "id")
	assert.Contains(t, template, "name")
	assert.Contains(t, template, "description")
	assert.Contains(t, template, "chain_type")

	// Verify filters in response
	filters := response["filters"].(map[string]interface{})
	assert.Equal(t, "", filters["chain_type"])
	assert.Equal(t, "", filters["keyword"])
	assert.Equal(t, float64(10), filters["limit"])
}

func TestListTemplateHandler_ChainTypeFilter(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// Create templates for different chains
	ethTemplate := &models.Template{
		Name:         "Ethereum Template",
		Description:  "Ethereum contract",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
	}
	err := templateService.CreateTemplate(ethTemplate)
	assert.NoError(t, err)

	solTemplate := &models.Template{
		Name:         "Solana Template",
		Description:  "Solana program",
		ChainType:    models.TransactionChainType("solana"),
		TemplateCode: validSolanaTemplate(),
	}
	err = templateService.CreateTemplate(solTemplate)
	assert.NoError(t, err)

	_, handler := NewListTemplateTool(templateService)

	tests := []struct {
		name          string
		chainType     string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "filter_ethereum",
			chainType:     "ethereum",
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "filter_solana",
			chainType:     "solana",
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "invalid_chain_type",
			chainType:     "bitcoin",
			expectedCount: 0,
			expectError:   true,
		},
		{
			name:          "empty_chain_type",
			chainType:     "",
			expectedCount: 2,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"chain_type": tt.chainType,
					},
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				assert.NoError(t, err) // Handler returns success with error content
				assert.NotNil(t, result)
				assert.True(t, result.IsError)
				assert.Len(t, result.Content, 1)
				textContent := result.Content[0].(mcp.TextContent)
				assert.Contains(t, textContent.Text, "Invalid chain_type")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.False(t, result.IsError)

				// Parse the response
				responseText, err := extractListResponseJSON(result)
				assert.NoError(t, err)

				var response map[string]interface{}
				err = json.Unmarshal([]byte(responseText), &response)
				assert.NoError(t, err)

				if tt.expectedCount == 0 && tt.chainType == "" {
					// Special case: empty chain_type should return all templates
					assert.Equal(t, float64(2), response["count"])
				} else {
					assert.Equal(t, float64(tt.expectedCount), response["count"])
				}

				// Verify filter is applied correctly
				if filters, exists := response["filters"]; exists {
					filtersMap := filters.(map[string]interface{})
					assert.Equal(t, tt.chainType, filtersMap["chain_type"])
				}
			}
		})
	}
}

func TestListTemplateHandler_KeywordFilter(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// Create templates with different names and descriptions
	template1 := &models.Template{
		Name:         "ERC20 Token",
		Description:  "Standard Ethereum token contract",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
	}
	err := templateService.CreateTemplate(template1)
	assert.NoError(t, err)

	template2 := &models.Template{
		Name:         "NFT Contract",
		Description:  "ERC721 Non-Fungible Token",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
	}
	err = templateService.CreateTemplate(template2)
	assert.NoError(t, err)

	template3 := &models.Template{
		Name:         "Solana Program",
		Description:  "Basic Solana smart contract",
		ChainType:    models.TransactionChainType("solana"),
		TemplateCode: validSolanaTemplate(),
	}
	err = templateService.CreateTemplate(template3)
	assert.NoError(t, err)

	_, handler := NewListTemplateTool(templateService)

	tests := []struct {
		name          string
		keyword       string
		expectedCount int
	}{
		{
			name:          "search_token",
			keyword:       "token",
			expectedCount: 2, // ERC20 Token and NFT Contract (description contains "Token")
		},
		{
			name:          "search_ethereum",
			keyword:       "Ethereum",
			expectedCount: 1, // Only template1 has "Ethereum" in description
		},
		{
			name:          "search_nft",
			keyword:       "NFT",
			expectedCount: 1, // Only template2 has "NFT" in name
		},
		{
			name:          "search_solana",
			keyword:       "Solana",
			expectedCount: 1, // Only template3 has "Solana" in name and description
		},
		{
			name:          "search_nonexistent",
			keyword:       "nonexistent",
			expectedCount: 0, // No matches
		},
		{
			name:          "empty_keyword",
			keyword:       "",
			expectedCount: 3, // All templates
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"keyword": tt.keyword,
					},
				},
			}

			result, err := handler(ctx, request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.False(t, result.IsError)

			// Parse the response
			var responseText string
			if len(result.Content) == 1 {
				// No results case: "Templates listed: {JSON}"
				fullText := result.Content[0].(mcp.TextContent).Text
				responseText = fullText[len("Templates listed: "):]
			} else {
				// Results found case: separate content items
				responseText = result.Content[len(result.Content)-1].(mcp.TextContent).Text
			}

			var response map[string]interface{}
			err = json.Unmarshal([]byte(responseText), &response)
			assert.NoError(t, err)

			assert.Equal(t, float64(tt.expectedCount), response["count"])

			// Verify filter is applied correctly if filters exist
			if filters, exists := response["filters"]; exists {
				filtersMap := filters.(map[string]interface{})
				assert.Equal(t, tt.keyword, filtersMap["keyword"])
			}
		})
	}
}

func TestListTemplateHandler_LimitParameter(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// Create multiple templates
	for i := 0; i < 15; i++ {
		template := &models.Template{
			Name:         fmt.Sprintf("Template %d", i+1),
			Description:  fmt.Sprintf("Description for template %d", i+1),
			ChainType:    models.TransactionChainType("ethereum"),
			TemplateCode: validEthereumTemplate(),
		}
		err := templateService.CreateTemplate(template)
		assert.NoError(t, err)
	}

	_, handler := NewListTemplateTool(templateService)

	tests := []struct {
		name          string
		limit         string
		expectedCount int
	}{
		{
			name:          "limit_5",
			limit:         "5",
			expectedCount: 5,
		},
		{
			name:          "limit_10",
			limit:         "10",
			expectedCount: 10,
		},
		{
			name:          "limit_20",
			limit:         "20",
			expectedCount: 15, // Only 15 templates exist
		},
		{
			name:          "invalid_limit",
			limit:         "invalid",
			expectedCount: 10, // Should default to 10
		},
		{
			name:          "empty_limit",
			limit:         "",
			expectedCount: 10, // Should default to 10
		},
		{
			name:          "zero_limit",
			limit:         "0",
			expectedCount: 15, // Zero limit actually returns all results in current implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{}
			if tt.limit != "" {
				args["limit"] = tt.limit
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: args,
				},
			}

			result, err := handler(ctx, request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.False(t, result.IsError)

			// Parse the response
			responseText, err := extractListResponseJSON(result)
			assert.NoError(t, err)

			var response map[string]interface{}
			err = json.Unmarshal([]byte(responseText), &response)
			assert.NoError(t, err)

			assert.Equal(t, float64(tt.expectedCount), response["count"])
			templates := response["templates"].([]interface{})
			assert.Len(t, templates, tt.expectedCount)

			// Verify filter shows correct limit
			if filters, exists := response["filters"]; exists {
				filtersMap := filters.(map[string]interface{})
				if tt.limit == "invalid" || tt.limit == "" {
					assert.Equal(t, float64(10), filtersMap["limit"])
				} else {
					expectedLimit, _ := strconv.Atoi(tt.limit)
					assert.Equal(t, float64(expectedLimit), filtersMap["limit"])
				}
			}
		})
	}
}

func TestListTemplateHandler_CombinedFilters(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// Create templates with specific combinations
	ethToken := &models.Template{
		Name:         "Ethereum Token",
		Description:  "ERC20 token contract",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
	}
	err := templateService.CreateTemplate(ethToken)
	assert.NoError(t, err)

	ethNFT := &models.Template{
		Name:         "Ethereum NFT",
		Description:  "ERC721 contract",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
	}
	err = templateService.CreateTemplate(ethNFT)
	assert.NoError(t, err)

	solToken := &models.Template{
		Name:         "Solana Token",
		Description:  "SPL token program",
		ChainType:    models.TransactionChainType("solana"),
		TemplateCode: validSolanaTemplate(),
	}
	err = templateService.CreateTemplate(solToken)
	assert.NoError(t, err)

	_, handler := NewListTemplateTool(templateService)

	tests := []struct {
		name          string
		chainType     string
		keyword       string
		limit         string
		expectedCount int
	}{
		{
			name:          "ethereum_token",
			chainType:     "ethereum",
			keyword:       "Token",
			limit:         "10",
			expectedCount: 1, // Only "Ethereum Token"
		},
		{
			name:          "ethereum_all",
			chainType:     "ethereum",
			keyword:       "",
			limit:         "10",
			expectedCount: 2, // Both Ethereum templates
		},
		{
			name:          "solana_token",
			chainType:     "solana",
			keyword:       "Token",
			limit:         "10",
			expectedCount: 1, // Only "Solana Token"
		},
		{
			name:          "all_token_limited",
			chainType:     "",
			keyword:       "Token",
			limit:         "1",
			expectedCount: 1, // Limited to 1 result
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"chain_type": tt.chainType,
						"keyword":    tt.keyword,
						"limit":      tt.limit,
					},
				},
			}

			result, err := handler(ctx, request)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.False(t, result.IsError)

			// Parse the response
			responseText, err := extractListResponseJSON(result)
			assert.NoError(t, err)

			var response map[string]interface{}
			err = json.Unmarshal([]byte(responseText), &response)
			assert.NoError(t, err)

			assert.Equal(t, float64(tt.expectedCount), response["count"])

			// Verify all filters are applied correctly
			if filters, exists := response["filters"]; exists {
				filtersMap := filters.(map[string]interface{})
				assert.Equal(t, tt.chainType, filtersMap["chain_type"])
				assert.Equal(t, tt.keyword, filtersMap["keyword"])
				limitInt, _ := strconv.Atoi(tt.limit)
				assert.Equal(t, float64(limitInt), filtersMap["limit"])
			}
		})
	}
}

func TestListTemplateHandler_ResponseFormat(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// Create a template with all fields populated
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Comprehensive test template",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
		Metadata:     models.JSON{"param1": "", "param2": ""},
	}
	err := templateService.CreateTemplate(template)
	assert.NoError(t, err)

	_, handler := NewListTemplateTool(templateService)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify response structure
	assert.Len(t, result.Content, 2)

	// First content should be success message
	textContent0 := result.Content[0].(mcp.TextContent)
	assert.Equal(t, "Templates listed successfully: ", textContent0.Text)

	// Second content should be JSON data
	textContent1 := result.Content[1].(mcp.TextContent)
	var response map[string]interface{}
	err = json.Unmarshal([]byte(textContent1.Text), &response)
	assert.NoError(t, err)

	// Verify top-level structure
	assert.Contains(t, response, "templates")
	assert.Contains(t, response, "count")
	assert.Contains(t, response, "filters")

	// Verify templates array structure
	templates := response["templates"].([]interface{})
	assert.Len(t, templates, 1)

	templateData := templates[0].(map[string]interface{})
	assert.Contains(t, templateData, "id")
	assert.Contains(t, templateData, "name")
	assert.Contains(t, templateData, "description")
	assert.Contains(t, templateData, "chain_type")

	// Verify template data matches created template
	assert.Equal(t, "Test Template", templateData["name"])
	assert.Equal(t, "Comprehensive test template", templateData["description"])
	assert.Equal(t, "ethereum", templateData["chain_type"])

	// Verify template data does NOT contain sensitive fields
	assert.NotContains(t, templateData, "template_code")
	assert.NotContains(t, templateData, "metadata")
	assert.NotContains(t, templateData, "abi")

	// Verify filters structure
	filters := response["filters"].(map[string]interface{})
	assert.Contains(t, filters, "chain_type")
	assert.Contains(t, filters, "keyword")
	assert.Contains(t, filters, "limit")
}

func TestListTemplateHandler_UserScope(t *testing.T) {
	ctx := context.Background()
	templateService := setupTestDatabase(t)

	// Create templates with different user ownership (if supported)
	user1ID := "user1"
	user2ID := "user2"
	template1 := &models.Template{
		Name:         "User 1 Template",
		Description:  "Template owned by user 1",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
		UserId:       &user1ID,
	}
	err := templateService.CreateTemplate(template1)
	assert.NoError(t, err)

	template2 := &models.Template{
		Name:         "User 2 Template",
		Description:  "Template owned by user 2",
		ChainType:    models.TransactionChainType("ethereum"),
		TemplateCode: validEthereumTemplate(),
		UserId:       &user2ID,
	}
	err = templateService.CreateTemplate(template2)
	assert.NoError(t, err)

	_, handler := NewListTemplateTool(templateService)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Parse the response
	responseText, err := extractListResponseJSON(result)
	assert.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(responseText), &response)
	assert.NoError(t, err)

	// Should return all templates regardless of user (since no user context in this test)
	assert.Equal(t, float64(2), response["count"])
}

// Helper function to extract JSON response from list template results
func extractListResponseJSON(result *mcp.CallToolResult) (string, error) {
	if len(result.Content) == 1 {
		// No results case: "Templates listed: {JSON}"
		fullText := result.Content[0].(mcp.TextContent).Text
		return fullText[len("Templates listed: "):], nil
	} else {
		// Results found case: separate content items
		return result.Content[len(result.Content)-1].(mcp.TextContent).Text, nil
	}
}
