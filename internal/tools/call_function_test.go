package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

const (
	CALL_FUNCTION_TEST_SERVER_PORT = 9998
)

type CallFunctionToolTestSuite struct {
	suite.Suite
	db                services.DBService
	tool              *callFunctionTool
	chain             *models.Chain
	template          *models.Template
	deployment        *models.Deployment
	templateService   services.TemplateService
	chainService      services.ChainService
	evmService        services.EvmService
	txService         services.TransactionService
	deploymentService services.DeploymentService
}

func (suite *CallFunctionToolTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize services
	suite.templateService = services.NewTemplateService(db.GetDB())
	suite.chainService = services.NewChainService(db.GetDB())
	suite.evmService = services.NewEvmService()
	suite.txService = services.NewTransactionService(db.GetDB())
	suite.deploymentService = services.NewDeploymentService(db.GetDB())

	// Initialize tool
	suite.tool = NewCallFunctionTool(
		suite.templateService,
		suite.evmService,
		suite.txService,
		suite.chainService,
		suite.deploymentService,
		CALL_FUNCTION_TEST_SERVER_PORT,
	)

	// Setup test data
	suite.setupTestData()
}

func (suite *CallFunctionToolTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *CallFunctionToolTestSuite) SetupTest() {
	// Clean up any test-specific data between tests
	suite.cleanupTestData()
}

func (suite *CallFunctionToolTestSuite) setupTestData() {
	// Create test chain
	chain := &models.Chain{
		Name:      "Test Ethereum",
		RPC:       "http://localhost:8545",
		NetworkID: "31337",
		ChainType: models.TransactionChainTypeEthereum,
		IsActive:  true,
	}
	err := suite.chainService.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain

	// Create test template with ABI
	abiArray := []interface{}{
		map[string]interface{}{
			"inputs": []interface{}{
				map[string]interface{}{"name": "account", "type": "address"},
			},
			"name":            "balanceOf",
			"outputs":         []interface{}{map[string]interface{}{"name": "", "type": "uint256"}},
			"stateMutability": "view",
			"type":            "function",
		},
		map[string]interface{}{
			"inputs": []interface{}{
				map[string]interface{}{"name": "to", "type": "address"},
				map[string]interface{}{"name": "amount", "type": "uint256"},
			},
			"name":            "transfer",
			"outputs":         []interface{}{map[string]interface{}{"name": "", "type": "bool"}},
			"stateMutability": "nonpayable",
			"type":            "function",
		},
	}

	abiData := models.JSON(map[string]interface{}{
		"abi": abiArray,
	})

	template := &models.Template{
		Name:        "Test ERC20",
		Description: "Test ERC20 contract",
		ChainType:   models.TransactionChainTypeEthereum,
		Abi:         abiData,
	}
	err = suite.templateService.CreateTemplate(template)
	suite.Require().NoError(err)
	suite.template = template

	// Create test deployment
	deployment := &models.Deployment{
		ChainID:         chain.ID,
		TemplateID:      template.ID,
		Status:          models.TransactionStatusConfirmed,
		ContractAddress: "0x1234567890123456789012345678901234567890",
	}
	err = suite.deploymentService.CreateDeployment(deployment)
	suite.Require().NoError(err)
	suite.deployment = deployment
}

func (suite *CallFunctionToolTestSuite) cleanupTestData() {
	// Clean up test data between tests if needed
	suite.db.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})
}

func (suite *CallFunctionToolTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("call_function", tool.Name)
	suite.Contains(tool.Description, "Call a smart contract function")
	suite.Contains(tool.InputSchema.Properties, "deployment_id")
	suite.Contains(tool.InputSchema.Properties, "function_name")
	suite.Contains(tool.InputSchema.Properties, "function_args")
	suite.Contains(tool.InputSchema.Properties, "value")
	suite.Contains(tool.InputSchema.Properties, "metadata")

	// Check required fields
	suite.Contains(tool.InputSchema.Required, "deployment_id")
	suite.Contains(tool.InputSchema.Required, "function_name")
}

func (suite *CallFunctionToolTestSuite) TestToolRegistration() {
	mcpServer := server.NewMCPServer("test", "1.0.0")
	tool := suite.tool.GetTool()
	handler := suite.tool.GetHandler()

	suite.NotPanics(func() {
		mcpServer.AddTool(tool, handler)
	})
}

func (suite *CallFunctionToolTestSuite) TestHandlerInvalidBindArguments() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: "invalid", // Should be a map, not a string
		},
	}

	handler := suite.tool.GetHandler()
	_, err := handler(context.Background(), request)

	suite.Error(err)
	suite.Contains(err.Error(), "failed to bind arguments")
}

func (suite *CallFunctionToolTestSuite) TestHandlerMissingRequiredFields() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"function_name": "balanceOf", // Missing deployment_id
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Invalid arguments")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerInvalidDeploymentID() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": "invalid",
				"function_name": "balanceOf",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Invalid deployment_id format")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerDeploymentNotFound() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": "999",
				"function_name": "balanceOf",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Deployment not found")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerDeploymentNotConfirmed() {
	// Create a pending deployment
	pendingDeployment := &models.Deployment{
		ChainID:    suite.chain.ID,
		TemplateID: suite.template.ID,
		Status:     models.TransactionStatusPending,
	}
	err := suite.deploymentService.CreateDeployment(pendingDeployment)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", pendingDeployment.ID),
				"function_name": "balanceOf",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Deployment is not confirmed yet")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerNoActiveChain() {
	// Deactivate the chain
	err := suite.db.GetDB().Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", suite.deployment.ID),
				"function_name": "balanceOf",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "No active chain selected")
		}
	}

	// Reactivate chain for other tests
	err = suite.db.GetDB().Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
}

func (suite *CallFunctionToolTestSuite) TestHandlerFunctionNotFound() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", suite.deployment.ID),
				"function_name": "nonExistentFunction",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Function 'nonExistentFunction' not found in ABI")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerWrongArgumentCount() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", suite.deployment.ID),
				"function_name": "balanceOf",
				"function_args": []interface{}{"0x123", "extra_arg"}, // balanceOf expects 1 arg, got 2
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Function 'balanceOf' expects 1 arguments, got 2")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerReadOnlyFunctionSuccess() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", suite.deployment.ID),
				"function_name": "balanceOf",
				"function_args": []interface{}{"0x1234567890123456789012345678901234567890"},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.False(result.IsError)
	suite.Len(result.Content, 2)

	// Check success message
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Function 'balanceOf' called successfully")
		}
	}

	// Check result JSON
	if len(result.Content) > 1 {
		if textContent, ok := result.Content[1].(mcp.TextContent); ok {
			var callResult CallFunctionResult
			err := json.Unmarshal([]byte(textContent.Text), &callResult)
			suite.NoError(err)
			suite.Equal(fmt.Sprintf("%d", suite.deployment.ID), callResult.DeploymentID)
			suite.Equal("balanceOf", callResult.FunctionName)
			suite.Equal(suite.deployment.ContractAddress, callResult.ContractAddress)
			suite.True(callResult.Success)
			suite.True(callResult.IsReadOnly)
			suite.Contains(callResult.Result, "placeholder implementation")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerStateChangingFunctionSuccess() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", suite.deployment.ID),
				"function_name": "transfer",
				"function_args": []interface{}{"0x1234567890123456789012345678901234567890", "1000000000000000000"},
				"value":         "0",
				"metadata": []interface{}{
					map[string]interface{}{
						"key":   "action",
						"value": "token_transfer",
					},
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.False(result.IsError)
	suite.Len(result.Content, 4)

	// Check session creation message
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Function call transaction session created")
		}
	}

	// Check contract address message
	if len(result.Content) > 1 {
		if textContent, ok := result.Content[1].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, suite.deployment.ContractAddress)
			suite.Contains(textContent.Text, "Deployment ID:")
		}
	}

	// Check instruction message
	if len(result.Content) > 2 {
		if textContent, ok := result.Content[2].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Please sign the function call in the URL")
		}
	}

	// Check URL
	if len(result.Content) > 3 {
		if textContent, ok := result.Content[3].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "http://localhost:")
			suite.Contains(textContent.Text, "/tx/")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerChainMismatch() {
	// Create another chain
	otherChain := &models.Chain{
		Name:      "Other Chain",
		RPC:       "http://localhost:8546",
		NetworkID: "31338",
		ChainType: models.TransactionChainTypeEthereum,
		IsActive:  false,
	}
	err := suite.chainService.CreateChain(otherChain)
	suite.Require().NoError(err)

	// Create deployment on the other chain
	otherDeployment := &models.Deployment{
		ChainID:         otherChain.ID,
		TemplateID:      suite.template.ID,
		Status:          models.TransactionStatusConfirmed,
		ContractAddress: "0x9876543210987654321098765432109876543210",
	}
	err = suite.deploymentService.CreateDeployment(otherDeployment)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", otherDeployment.ID),
				"function_name": "balanceOf",
				"function_args": []interface{}{"0x1234567890123456789012345678901234567890"},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Deployment is on different chain")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestHandlerTemplateWithoutABI() {
	// Create template without ABI
	templateWithoutABI := &models.Template{
		Name:        "Test Contract No ABI",
		Description: "Test contract without ABI",
		ChainType:   models.TransactionChainTypeEthereum,
		Abi:         nil,
	}
	err := suite.templateService.CreateTemplate(templateWithoutABI)
	suite.Require().NoError(err)

	// Create deployment with this template
	deploymentWithoutABI := &models.Deployment{
		ChainID:         suite.chain.ID,
		TemplateID:      templateWithoutABI.ID,
		Status:          models.TransactionStatusConfirmed,
		ContractAddress: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
	}
	err = suite.deploymentService.CreateDeployment(deploymentWithoutABI)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"deployment_id": fmt.Sprintf("%d", deploymentWithoutABI.ID),
				"function_name": "balanceOf",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "Template does not have ABI information")
		}
	}
}

func (suite *CallFunctionToolTestSuite) TestCallReadOnlyEthereumFunction() {
	// Test the helper function
	method := abi.Method{
		Name: "balanceOf",
	}
	result, err := suite.tool.callReadOnlyEthereumFunction("0x123", method, []any{"0x456"})

	suite.NoError(err)
	suite.Contains(result, "placeholder implementation")
	suite.Contains(result, "balanceOf")
}

func TestCallFunctionToolTestSuite(t *testing.T) {
	suite.Run(t, new(CallFunctionToolTestSuite))
}
