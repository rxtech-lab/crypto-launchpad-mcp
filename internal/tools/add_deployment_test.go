package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

type AddDeploymentTestSuite struct {
	suite.Suite
	db                services.DBService
	tool              *addDeploymentTool
	chain             *models.Chain
	template          *models.Template
	deploymentService services.DeploymentService
	templateService   services.TemplateService
	chainService      services.ChainService
}

func (suite *AddDeploymentTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize services
	suite.deploymentService = services.NewDeploymentService(db.GetDB())
	suite.templateService = services.NewTemplateService(db.GetDB())
	suite.chainService = services.NewChainService(db.GetDB())

	// Initialize tool
	suite.tool = NewAddDeploymentTool(suite.deploymentService, suite.templateService, suite.chainService)

	// Setup test data
	suite.setupTestChain()
	suite.setupTestTemplate()
}

func (suite *AddDeploymentTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *AddDeploymentTestSuite) SetupTest() {
	// Clean up any test-specific data between tests
	suite.cleanupTestData()
}

func (suite *AddDeploymentTestSuite) setupTestChain() {
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       "http://localhost:8545",
		NetworkID: "31337",
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := suite.chainService.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *AddDeploymentTestSuite) setupTestTemplate() {
	contractCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract TestToken {
    string public name = "{{.TokenName}}";
    string public symbol = "{{.TokenSymbol}}";
}`

	template := &models.Template{
		Name:         "TestToken",
		Description:  "A test token template",
		ChainType:    models.TransactionChainTypeEthereum,
		TemplateCode: contractCode,
		SampleTemplateValues: map[string]any{
			"TokenName":   "Test",
			"TokenSymbol": "TST",
		},
	}

	err := suite.templateService.CreateTemplate(template)
	suite.Require().NoError(err)
	suite.template = template
}

func (suite *AddDeploymentTestSuite) cleanupTestData() {
	// Clean up deployments
	suite.db.GetDB().Where("1 = 1").Delete(&models.Deployment{})
}

// TestGetTool tests the tool definition
func (suite *AddDeploymentTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("add_deployment", tool.Name)
	suite.NotEmpty(tool.Description)
	suite.NotNil(tool.InputSchema)

	// Verify required parameters
	properties := tool.InputSchema.Properties
	suite.Contains(properties, "template_id")
	suite.Contains(properties, "chain_id")
	suite.Contains(properties, "contract_address")
	suite.Contains(properties, "transaction_hash")
	suite.Contains(properties, "deployer_address")

	// Verify optional parameters
	suite.Contains(properties, "status")
	suite.Contains(properties, "template_values")
	suite.Contains(properties, "session_id")
	suite.Contains(properties, "user_id")

	// Verify required fields
	required := tool.InputSchema.Required
	suite.Contains(required, "template_id")
	suite.Contains(required, "chain_id")
	suite.Contains(required, "contract_address")
	suite.Contains(required, "transaction_hash")
	suite.Contains(required, "deployer_address")
}

// TestHandlerSuccess tests successful deployment creation
func (suite *AddDeploymentTestSuite) TestHandlerSuccess() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      fmt.Sprintf("%d", suite.template.ID),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.False(result.IsError)
	suite.Require().Len(result.Content, 2)

	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		suite.NoError(err)
		suite.NotNil(response["id"])
		suite.Equal(float64(suite.template.ID), response["template_id"])
		suite.Equal(float64(suite.chain.ID), response["chain_id"])
		suite.Equal("0x1234567890123456789012345678901234567890", response["contract_address"])
		suite.Equal("0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd", response["transaction_hash"])
		suite.Equal("0x9876543210987654321098765432109876543210", response["deployer_address"])
		suite.Equal("confirmed", response["status"])
	}

	// Verify deployment was created in database
	deployments, err := suite.deploymentService.ListDeployments()
	suite.NoError(err)
	suite.Len(deployments, 1)
}

// TestHandlerWithTemplateValues tests deployment creation with template values
func (suite *AddDeploymentTestSuite) TestHandlerWithTemplateValues() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      fmt.Sprintf("%d", suite.template.ID),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
				"template_values": map[string]any{
					"TokenName":   "MyToken",
					"TokenSymbol": "MTK",
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.False(result.IsError)

	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		suite.NoError(err)
		suite.NotNil(response["template_values"])

		templateValues := response["template_values"].(map[string]interface{})
		suite.Equal("MyToken", templateValues["TokenName"])
		suite.Equal("MTK", templateValues["TokenSymbol"])
	}
}

// TestHandlerWithCustomStatus tests deployment creation with custom status
func (suite *AddDeploymentTestSuite) TestHandlerWithCustomStatus() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      fmt.Sprintf("%d", suite.template.ID),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
				"status":           "pending",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.False(result.IsError)

	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		suite.NoError(err)
		suite.Equal("pending", response["status"])
	}
}

// TestHandlerWithAuthenticatedUser tests deployment creation with authenticated user
func (suite *AddDeploymentTestSuite) TestHandlerWithAuthenticatedUser() {
	// Create authenticated context
	user := &utils.AuthenticatedUser{
		Sub: "test-user-123",
	}
	ctx := utils.WithAuthenticatedUser(context.Background(), user)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      fmt.Sprintf("%d", suite.template.ID),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(ctx, request)

	suite.NoError(err)
	suite.False(result.IsError)

	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		suite.NoError(err)
		suite.Equal("test-user-123", response["user_id"])
	}
}

// TestHandlerInvalidTemplateID tests error handling for invalid template ID
func (suite *AddDeploymentTestSuite) TestHandlerInvalidTemplateID() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      "999",
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Template not found")
	}
}

// TestHandlerInvalidChainID tests error handling for invalid chain ID
func (suite *AddDeploymentTestSuite) TestHandlerInvalidChainID() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      fmt.Sprintf("%d", suite.template.ID),
				"chain_id":         "999",
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Chain not found")
	}
}

// TestHandlerChainTypeMismatch tests error handling for chain type mismatch
func (suite *AddDeploymentTestSuite) TestHandlerChainTypeMismatch() {
	// Create a Solana chain
	solanaChain := &models.Chain{
		ChainType: models.TransactionChainTypeSolana,
		RPC:       "http://localhost:8899",
		NetworkID: "solana-testnet",
		Name:      "Solana Testnet",
		IsActive:  false,
	}
	err := suite.chainService.CreateChain(solanaChain)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      fmt.Sprintf("%d", suite.template.ID), // Ethereum template
				"chain_id":         fmt.Sprintf("%d", solanaChain.ID),    // Solana chain
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "doesn't match chain type")
	}
}

// TestHandlerInvalidStatus tests error handling for invalid status
func (suite *AddDeploymentTestSuite) TestHandlerInvalidStatus() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id":      fmt.Sprintf("%d", suite.template.ID),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				"deployer_address": "0x9876543210987654321098765432109876543210",
				"status":           "invalid_status",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid status")
	}
}

// TestHandlerMissingRequiredFields tests error handling for missing required fields
func (suite *AddDeploymentTestSuite) TestHandlerMissingRequiredFields() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"template_id": fmt.Sprintf("%d", suite.template.ID),
				// Missing required fields
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid arguments")
	}
}

// TestToolRegistration tests that the tool can be registered with MCP server
func (suite *AddDeploymentTestSuite) TestToolRegistration() {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	tool := suite.tool.GetTool()
	handler := suite.tool.GetHandler()

	suite.NotPanics(func() {
		mcpServer.AddTool(tool, handler)
	})
}

func TestAddDeploymentTestSuite(t *testing.T) {
	suite.Run(t, new(AddDeploymentTestSuite))
}
