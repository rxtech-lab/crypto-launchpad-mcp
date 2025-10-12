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
	// Not needed anymore since templates are auto-created
	// But keep for backward compatibility with some tests
}

func (suite *AddDeploymentTestSuite) cleanupTestData() {
	// Clean up deployments and templates
	suite.db.GetDB().Where("1 = 1").Delete(&models.Deployment{})
	suite.db.GetDB().Where("1 = 1").Delete(&models.Template{})
}

func (suite *AddDeploymentTestSuite) getTestContractCode() string {
	return `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract TestToken {
    string public name;
    string public symbol;

    constructor(string memory _name, string memory _symbol) {
        name = _name;
        symbol = _symbol;
    }
}`
}

// TestGetTool tests the tool definition
func (suite *AddDeploymentTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("add_deployment", tool.Name)
	suite.NotEmpty(tool.Description)
	suite.NotNil(tool.InputSchema)

	// Verify required parameters
	properties := tool.InputSchema.Properties
	suite.Contains(properties, "contract_code")
	suite.Contains(properties, "chain_id")
	suite.Contains(properties, "contract_address")
	suite.Contains(properties, "owner_address")

	// Verify optional parameters
	suite.Contains(properties, "transaction_hash")
	suite.Contains(properties, "template_values")
	suite.Contains(properties, "solc_version")
	suite.Contains(properties, "template_name")
	suite.Contains(properties, "description")

	// Verify required fields
	required := tool.InputSchema.Required
	suite.Contains(required, "contract_code")
	suite.Contains(required, "chain_id")
	suite.Contains(required, "contract_address")
	suite.Contains(required, "owner_address")
}

// TestHandlerSuccess tests successful deployment creation
func (suite *AddDeploymentTestSuite) TestHandlerSuccess() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"contract_code":    suite.getTestContractCode(),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"owner_address":    "0x9876543210987654321098765432109876543210",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
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
		suite.NotNil(response["template_id"])
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
	suite.NotEmpty(deployments[0].SessionId) // Session ID should be auto-generated

	// Verify template was auto-created
	templates, err := suite.templateService.ListTemplates(nil, "", "", 0)
	suite.NoError(err)
	suite.Len(templates, 1)
	suite.Equal("TestToken", templates[0].Name)
}

// TestHandlerWithTemplateValues tests deployment creation with template values
func (suite *AddDeploymentTestSuite) TestHandlerWithTemplateValues() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"contract_code":    suite.getTestContractCode(),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"owner_address":    "0x9876543210987654321098765432109876543210",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
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

// TestHandlerDefaultStatus tests that deployment is created with confirmed status by default
func (suite *AddDeploymentTestSuite) TestHandlerDefaultStatus() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"contract_code":    suite.getTestContractCode(),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"owner_address":    "0x9876543210987654321098765432109876543210",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
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
		suite.Equal("confirmed", response["status"])
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
				"contract_code":    suite.getTestContractCode(),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"owner_address":    "0x9876543210987654321098765432109876543210",
				"transaction_hash": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd",
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

// TestHandlerInvalidContractCode tests error handling for invalid contract code
func (suite *AddDeploymentTestSuite) TestHandlerInvalidContractCode() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"contract_code":    "invalid solidity code",
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"owner_address":    "0x9876543210987654321098765432109876543210",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Failed to compile contract")
	}
}

// TestHandlerInvalidChainID tests error handling for invalid chain ID
func (suite *AddDeploymentTestSuite) TestHandlerInvalidChainID() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"contract_code":    suite.getTestContractCode(),
				"chain_id":         "999",
				"contract_address": "0x1234567890123456789012345678901234567890",
				"owner_address":    "0x9876543210987654321098765432109876543210",
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

// TestHandlerWithCustomTemplateName tests deployment with custom template name
func (suite *AddDeploymentTestSuite) TestHandlerWithCustomTemplateName() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"contract_code":    suite.getTestContractCode(),
				"chain_id":         fmt.Sprintf("%d", suite.chain.ID),
				"contract_address": "0x1234567890123456789012345678901234567890",
				"owner_address":    "0x9876543210987654321098765432109876543210",
				"template_name":    "CustomTokenName",
				"description":      "A custom token template",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.False(result.IsError)

	// Verify template was created with custom name
	templates, err := suite.templateService.ListTemplates(nil, "", "", 0)
	suite.NoError(err)
	suite.Len(templates, 1)
	suite.Equal("CustomTokenName", templates[0].Name)
	suite.Equal("A custom token template", templates[0].Description)
}

// TestHandlerMissingRequiredFields tests error handling for missing required fields
func (suite *AddDeploymentTestSuite) TestHandlerMissingRequiredFields() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"chain_id": fmt.Sprintf("%d", suite.chain.ID),
				// Missing required fields: contract_code, contract_address, owner_address
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
