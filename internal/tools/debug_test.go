package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestDebugCreateTemplate(t *testing.T) {
	ctx := context.Background()
	db := setupTestDatabase(t)
	_, handler := NewCreateTemplateTool(db)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"name":          "Debug Template",
				"description":   "Debug description",
				"chain_type":    "ethereum",
				"template_code": validEthereumTemplate(),
			},
		},
	}

	result, err := handler(ctx, request)

	// Debug output
	t.Logf("Error: %v", err)
	t.Logf("Result: %+v", result)

	if result != nil {
		for i, content := range result.Content {
			if textContent, ok := content.(mcp.TextContent); ok {
				t.Logf("Content[%d]: %s", i, textContent.Text)
			}
		}
	}

	// Check database directly
	templates, dbErr := db.ListTemplates("", "", 10)
	t.Logf("Database error: %v", dbErr)
	t.Logf("Templates count: %d", len(templates))
	for i, template := range templates {
		t.Logf("Template[%d]: %+v", i, template)
	}

	assert.NoError(t, err)
	assert.NotNil(t, result)
}
