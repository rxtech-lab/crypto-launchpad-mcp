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

func TestNewViewTemplateTool(t *testing.T) {
	// Setup test database
	db, err := services.NewSqliteDBService(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	templateService := services.NewTemplateService(db.GetDB())
	evmService := services.NewEvmService()

	// Create tool
	viewTool := NewViewTemplateTool(templateService, evmService)
	tool := viewTool.GetTool()
	handler := viewTool.GetHandler()

	// Test tool metadata
	assert.Equal(t, "view_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.NotNil(t, handler)

	// Check that the tool has the expected properties
	assert.Contains(t, tool.InputSchema.Properties, "template_id")
	assert.Contains(t, tool.InputSchema.Properties, "show_abi_methods")
	assert.Contains(t, tool.InputSchema.Properties, "abi_method")
}

func TestViewTemplateHandler_InvalidTemplateID(t *testing.T) {
	ctx := context.Background()

	// Setup test database
	db, err := services.NewSqliteDBService(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	templateService := services.NewTemplateService(db.GetDB())
	evmService := services.NewEvmService()

	viewTool := NewViewTemplateTool(templateService, evmService)
	handler := viewTool.GetHandler()

	// Test with invalid template ID
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "999",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Len(t, result.Content, 1)

	// Check error message
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		assert.Contains(t, textContent.Text, "Template not found")
	}
}

func TestViewTemplateHandler_MissingRequiredArgs(t *testing.T) {
	ctx := context.Background()

	// Setup test database
	db, err := services.NewSqliteDBService(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	templateService := services.NewTemplateService(db.GetDB())
	evmService := services.NewEvmService()

	viewTool := NewViewTemplateTool(templateService, evmService)
	handler := viewTool.GetHandler()

	// Test with missing required template_id
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Len(t, result.Content, 1)

	// Check validation error message
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		assert.Contains(t, textContent.Text, "Invalid arguments")
	}
}

func TestFunctionTypeToString(t *testing.T) {
	// Import abi for testing function types (this would require actual abi package)
	// For now, let's test the function exists and handles default case
	result := functionTypeToString(999) // Invalid function type
	assert.Equal(t, "unknown", result)
}

func TestConvertAbiArguments(t *testing.T) {
	// Test nil arguments - should return empty slice, not nil
	result := convertAbiArguments(nil)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
	assert.Equal(t, []AbiMethodArgument{}, result)
}

func TestViewTemplateHandler_ValidTemplate(t *testing.T) {
	ctx := context.Background()

	// Setup test database with a template
	db, err := services.NewSqliteDBService(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	templateService := services.NewTemplateService(db.GetDB())
	evmService := services.NewEvmService()

	// Create a test template
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Test description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
	}
	err = templateService.CreateTemplate(template)
	assert.NoError(t, err)

	viewTool := NewViewTemplateTool(templateService, evmService)
	handler := viewTool.GetHandler()

	// Test basic template viewing without ABI methods
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": "1",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 2)

	// Check success message
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		assert.Contains(t, textContent.Text, "Template 'Test Template' retrieved successfully")
	}

	// Parse result JSON
	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		var resultData map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultData)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), resultData["id"])
		assert.Equal(t, "Test Template", resultData["name"])
		assert.Equal(t, "Test description", resultData["description"])
		assert.Equal(t, "ethereum", resultData["chain_type"])
	}
}

func TestViewTemplateHandler_WithShowAbiMethodsNoAbi(t *testing.T) {
	ctx := context.Background()

	// Setup test database with a template that has no ABI
	db, err := services.NewSqliteDBService(":memory:")
	if err != nil {
		t.Fatalf("Failed to setup test database: %v", err)
	}

	templateService := services.NewTemplateService(db.GetDB())
	evmService := services.NewEvmService()

	// Create a test template without ABI
	template := &models.Template{
		Name:         "Test Template",
		Description:  "Test description",
		ChainType:    "ethereum",
		TemplateCode: validEthereumTemplate(),
		// No ABI provided - should not attempt to process ABI methods
	}
	err = templateService.CreateTemplate(template)
	assert.NoError(t, err)

	viewTool := NewViewTemplateTool(templateService, evmService)
	handler := viewTool.GetHandler()

	// Test template viewing with show_abi_methods = true but no ABI
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      "1",
				"show_abi_methods": true,
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 2)

	// Check success message
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		assert.Contains(t, textContent.Text, "Template 'Test Template' retrieved successfully")
		// Should not show ABI methods info since no ABI is present
		assert.NotContains(t, textContent.Text, "showing")
	}

	// Parse result JSON - should not have abi_methods field
	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		var resultData map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &resultData)
		assert.NoError(t, err)
		assert.Equal(t, float64(1), resultData["id"])
		assert.Equal(t, "Test Template", resultData["name"])
		// Should not have abi_methods field
		assert.NotContains(t, resultData, "abi_methods")
		assert.NotContains(t, resultData, "abi_method")
	}
}
