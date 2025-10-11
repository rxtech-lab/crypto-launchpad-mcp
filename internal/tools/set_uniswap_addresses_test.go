package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

type SetUniswapAddressesTestSuite struct {
	suite.Suite
	dbService      services.DBService
	uniswapService services.UniswapService
	chainService   services.ChainService
	tool           *setUniswapAddressesTool
	testChain      *models.Chain
}

func (suite *SetUniswapAddressesTestSuite) SetupSuite() {
	// Initialize in-memory database for testing
	dbService, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.dbService = dbService

	// Initialize services
	suite.uniswapService = services.NewUniswapService(dbService.GetDB())
	suite.chainService = services.NewChainService(dbService.GetDB())

	// Initialize tool
	suite.tool = NewSetUniswapAddressesTool(suite.uniswapService, suite.chainService)

	// Setup test data
	suite.setupTestData()
}

func (suite *SetUniswapAddressesTestSuite) TearDownSuite() {
	if suite.dbService != nil {
		suite.dbService.Close()
	}
}

func (suite *SetUniswapAddressesTestSuite) SetupTest() {
	// Clean up any test-specific data between tests
	suite.cleanupTestData()
}

func (suite *SetUniswapAddressesTestSuite) setupTestData() {
	// Create test chain
	testChain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		Name:      "Test Ethereum",
		RPC:       "http://localhost:8545",
		NetworkID: "31337",
		IsActive:  true,
	}
	err := suite.chainService.CreateChain(testChain)
	suite.Require().NoError(err)
	suite.testChain = testChain
}

func (suite *SetUniswapAddressesTestSuite) cleanupTestData() {
	// Clean up deployments
	suite.dbService.GetDB().Where("1 = 1").Delete(&models.UniswapDeployment{})
}

func (suite *SetUniswapAddressesTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()
	suite.Equal("set_uniswap_addresses", tool.Name)
	suite.Contains(tool.Description, "Set or update Uniswap contract addresses")

	// Check parameters
	suite.Require().Len(tool.InputSchema.Properties, 4)

	// Check version parameter
	versionParam, exists := tool.InputSchema.Properties["version"]
	suite.True(exists)
	if param, ok := versionParam.(map[string]any); ok {
		suite.Equal("string", param["type"])
		suite.Contains(param["description"], "Uniswap version")
	}

	// Check factory_address parameter
	factoryParam, exists := tool.InputSchema.Properties["factory_address"]
	suite.True(exists)
	if param, ok := factoryParam.(map[string]any); ok {
		suite.Equal("string", param["type"])
		suite.Contains(param["description"], "Factory contract address")
	}

	// Check router_address parameter
	routerParam, exists := tool.InputSchema.Properties["router_address"]
	suite.True(exists)
	if param, ok := routerParam.(map[string]any); ok {
		suite.Equal("string", param["type"])
		suite.Contains(param["description"], "Router contract address")
	}

	// Check weth_address parameter
	wethParam, exists := tool.InputSchema.Properties["weth_address"]
	suite.True(exists)
	if param, ok := wethParam.(map[string]any); ok {
		suite.Equal("string", param["type"])
		suite.Contains(param["description"], "WETH contract address")
	}
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerSuccess_SetSingleAddress() {
	factoryAddress := "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":         "v2",
				"factory_address": factoryAddress,
			},
		},
	}

	result, err := suite.tool.GetHandler()(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError, "Expected successful result")

	// Verify content
	suite.Require().Len(result.Content, 2)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Successfully updated Uniswap v2 addresses")
	}

	// Parse JSON response
	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		var response map[string]interface{}
		err = json.Unmarshal([]byte(textContent.Text), &response)
		suite.NoError(err)
		suite.Equal("v2", response["version"])
		suite.Equal(factoryAddress, response["factory_address"])
		suite.Contains(response["updated_fields"], "factory_address")
	}

	// Verify deployment was created
	deployment, err := suite.uniswapService.GetUniswapDeploymentByChain(suite.testChain.ID)
	suite.NoError(err)
	suite.NotNil(deployment)
	suite.Equal(factoryAddress, deployment.FactoryAddress)
	suite.Equal("", deployment.RouterAddress)
	suite.Equal("", deployment.WETHAddress)
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerSuccess_SetMultipleAddresses() {
	factoryAddress := "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f"
	routerAddress := "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
	wethAddress := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":         "v2",
				"factory_address": factoryAddress,
				"router_address":  routerAddress,
				"weth_address":    wethAddress,
			},
		},
	}

	result, err := suite.tool.GetHandler()(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Verify deployment was created with all addresses
	deployment, err := suite.uniswapService.GetUniswapDeploymentByChain(suite.testChain.ID)
	suite.NoError(err)
	suite.NotNil(deployment)
	suite.Equal(factoryAddress, deployment.FactoryAddress)
	suite.Equal(routerAddress, deployment.RouterAddress)
	suite.Equal(wethAddress, deployment.WETHAddress)
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerSuccess_UpdateExistingDeployment() {
	// Create existing deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Set initial factory address
	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f")
	suite.Require().NoError(err)

	// Update with router address
	newRouterAddress := "0x7a250d5630B4cF539739dF2C5dAcb4c659F2488D"
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":        "v2",
				"router_address": newRouterAddress,
			},
		},
	}

	result, err := suite.tool.GetHandler()(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Verify deployment was updated (not created new)
	deployment, err := suite.uniswapService.GetUniswapDeploymentByChain(suite.testChain.ID)
	suite.NoError(err)
	suite.NotNil(deployment)
	suite.Equal(deploymentID, deployment.ID)
	suite.Equal("0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f", deployment.FactoryAddress)
	suite.Equal(newRouterAddress, deployment.RouterAddress)
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerError_NoAddressProvided() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version": "v2",
			},
		},
	}

	result, err := suite.tool.GetHandler()(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "At least one address")
	}
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerError_InvalidAddressFormat() {
	testCases := []struct {
		name    string
		address string
		field   string
	}{
		{
			name:    "Invalid factory address - too short",
			address: "0x123",
			field:   "factory_address",
		},
		{
			name:    "Invalid factory address - invalid characters",
			address: "0xGGGGbEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
			field:   "factory_address",
		},
		{
			name:    "Invalid router address - no 0x prefix and wrong length",
			address: "7a250d5630B4cF539739dF2C5dAcb4c659F2488",
			field:   "router_address",
		},
		{
			name:    "Invalid weth address - too long",
			address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2FF",
			field:   "weth_address",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"version": "v2",
						tc.field:  tc.address,
					},
				},
			}

			result, err := suite.tool.GetHandler()(context.Background(), request)

			suite.NoError(err)
			suite.True(result.IsError)
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.Contains(textContent.Text, "Invalid")
			}
		})
	}
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerError_InvalidVersion() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":         "v5",
				"factory_address": "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
			},
		},
	}

	result, err := suite.tool.GetHandler()(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "version")
	}
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerError_NoActiveChain() {
	// Deactivate the test chain
	err := suite.dbService.GetDB().Model(&models.Chain{}).Where("id = ?", suite.testChain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":         "v2",
				"factory_address": "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
			},
		},
	}

	result, err := suite.tool.GetHandler()(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "No active chain selected")
	}

	// Reactivate for other tests
	err = suite.dbService.GetDB().Model(&models.Chain{}).Where("id = ?", suite.testChain.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
}

func (suite *SetUniswapAddressesTestSuite) TestHandlerError_MissingRequiredFields() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"factory_address": "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
				// Missing version
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

func (suite *SetUniswapAddressesTestSuite) TestValidateEthereumAddress() {
	testCases := []struct {
		name      string
		address   string
		expectErr bool
	}{
		{
			name:      "Valid address with 0x prefix",
			address:   "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
			expectErr: false,
		},
		{
			name:      "Valid address without 0x prefix",
			address:   "5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f",
			expectErr: false,
		},
		{
			name:      "Valid address lowercase",
			address:   "0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f",
			expectErr: false,
		},
		{
			name:      "Invalid - too short",
			address:   "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA",
			expectErr: true,
		},
		{
			name:      "Invalid - too long",
			address:   "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aA6f11",
			expectErr: true,
		},
		{
			name:      "Invalid - contains non-hex characters",
			address:   "0x5C69bEe701ef814a2B6a3EDD4B1652CB9cc5aAGZ",
			expectErr: true,
		},
		{
			name:      "Invalid - empty string",
			address:   "",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			err := validateEthereumAddress(tc.address)
			if tc.expectErr {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func TestSetUniswapAddressesTestSuite(t *testing.T) {
	suite.Run(t, new(SetUniswapAddressesTestSuite))
}
