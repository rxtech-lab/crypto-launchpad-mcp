package tools

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

const (
	DEPLOY_UNISWAP_TESTNET_RPC      = "http://localhost:8545"
	DEPLOY_UNISWAP_TESTNET_CHAIN_ID = "31337"
	DEPLOY_UNISWAP_SERVER_PORT      = 9998 // Use different port for testing
)

type DeployUniswapToolTestSuite struct {
	suite.Suite
	db                services.DBService
	ethClient         *ethclient.Client
	deployUniswapTool *deployUniswapTool
	chain             *models.Chain
	chainService      services.ChainService
	uniswapService    services.UniswapService
	txService         services.TransactionService
	evmService        services.EvmService
}

func (suite *DeployUniswapToolTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize services
	suite.chainService = services.NewChainService(db.GetDB())
	suite.uniswapService = services.NewUniswapService(db.GetDB())
	suite.txService = services.NewTransactionService(db.GetDB())
	suite.evmService = services.NewEvmService()

	// Initialize Ethereum client
	ethClient, err := ethclient.Dial(DEPLOY_UNISWAP_TESTNET_RPC)
	suite.Require().NoError(err)
	suite.ethClient = ethClient

	// Verify Ethereum connection
	err = suite.verifyEthereumConnection()
	suite.Require().NoError(err)

	// Initialize deploy uniswap tool
	suite.deployUniswapTool = NewDeployUniswapTool(suite.chainService, DEPLOY_UNISWAP_SERVER_PORT, suite.evmService, suite.txService, suite.uniswapService)

	// Setup test data
	suite.setupTestChain()
}

func (suite *DeployUniswapToolTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.ethClient != nil {
		suite.ethClient.Close()
	}
}

func (suite *DeployUniswapToolTestSuite) SetupTest() {
	// Clean up any existing sessions for each test
	suite.cleanupTestData()
}

func (suite *DeployUniswapToolTestSuite) verifyEthereumConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := suite.ethClient.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network ID: %w", err)
	}

	if networkID.Cmp(big.NewInt(31337)) != 0 {
		return fmt.Errorf("unexpected network ID: got %s, expected 31337", networkID.String())
	}

	return nil
}

func (suite *DeployUniswapToolTestSuite) setupTestChain() {
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       DEPLOY_UNISWAP_TESTNET_RPC,
		NetworkID: DEPLOY_UNISWAP_TESTNET_CHAIN_ID,
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := suite.chainService.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *DeployUniswapToolTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.db.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})

	// Clean up uniswap deployments
	suite.db.GetDB().Where("1 = 1").Delete(&models.UniswapDeployment{})
}

func (suite *DeployUniswapToolTestSuite) TestGetTool() {
	tool := suite.deployUniswapTool.GetTool()

	suite.Equal("deploy_uniswap", tool.Name)
	suite.Contains(tool.Description, "Deploy Uniswap infrastructure contracts")
	suite.Contains(tool.Description, "version selection")
	suite.Contains(tool.Description, "transaction session")

	// Check required parameters
	suite.NotNil(tool.InputSchema)
	properties := tool.InputSchema.Properties

	// Check version parameter
	versionProp, exists := properties["version"]
	suite.True(exists)
	if propMap, ok := versionProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}

	// Check optional parameters
	deployRouterProp, exists := properties["deploy_router"]
	suite.True(exists)
	if propMap, ok := deployRouterProp.(map[string]any); ok {
		suite.Equal("boolean", propMap["type"])
	}

	metadataProp, exists := properties["metadata"]
	suite.True(exists)
	if propMap, ok := metadataProp.(map[string]any); ok {
		suite.Equal("array", propMap["type"])
	}
}

func (suite *DeployUniswapToolTestSuite) TestHandlerWETHFactoryDeploymentTransactionTypes() {
	// Create test request for infrastructure deployment (WETH + Factory)
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":       "v2",
				"deploy_router": false,
				"metadata": []interface{}{
					map[string]interface{}{
						"key":   "Deploy Type",
						"value": "Uniswap V2 Infrastructure",
					},
				},
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Verify response is successful
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.T().Logf("Unexpected error: %s", textContent.Text)
				suite.FailNow("Expected successful result but got error", textContent.Text)
			}
		}
		suite.FailNow("Expected successful result but got error with no content")
	}

	suite.Len(result.Content, 3)

	// Extract session ID from response
	var sessionIDContent string
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			sessionIDContent = textContent.Text
			suite.Contains(sessionIDContent, "Transaction session created:")
		}
	}

	sessionID := sessionIDContent[len("Transaction session created: "):]

	// Verify session was created in database
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)
	suite.Equal(models.TransactionStatusPending, session.TransactionStatus)
	suite.Equal(models.TransactionChainTypeEthereum, session.TransactionChainType)
	suite.Equal(suite.chain.ID, session.ChainID)

	// Verify we have 2 deployments for infrastructure (WETH + Factory)
	suite.Len(session.TransactionDeployments, 2)

	// Verify WETH9 deployment transaction type
	wethDeployment := session.TransactionDeployments[0]
	suite.Equal("Deploy WETH9", wethDeployment.Title)
	suite.Equal("Deploy Wrapped Ether (WETH9) contract for Uniswap V2", wethDeployment.Description)
	suite.Equal("0", wethDeployment.Value)
	suite.Equal("", wethDeployment.Receiver)
	suite.NotEmpty(wethDeployment.Data)
	suite.Equal(models.TransactionTypeUniswapV2TokenDeployment, wethDeployment.TransactionType)

	// Verify Factory deployment transaction type
	factoryDeployment := session.TransactionDeployments[1]
	suite.Equal("Deploy UniswapV2Factory", factoryDeployment.Title)
	suite.Equal("Deploy Uniswap V2 Factory contract", factoryDeployment.Description)
	suite.Equal("0", factoryDeployment.Value)
	suite.Equal("", factoryDeployment.Receiver)
	suite.NotEmpty(factoryDeployment.Data)
	suite.Equal(models.TransactionTypeUniswapV2FactoryDeployment, factoryDeployment.TransactionType)

	// Verify the transaction data contains compiled bytecode
	suite.True(len(wethDeployment.Data) > 10, "WETH transaction data should contain compiled bytecode")
	suite.True(wethDeployment.Data[:2] == "0x", "WETH transaction data should be hex encoded")
	suite.True(len(factoryDeployment.Data) > 10, "Factory transaction data should contain compiled bytecode")
	suite.True(factoryDeployment.Data[:2] == "0x", "Factory transaction data should be hex encoded")
}

func (suite *DeployUniswapToolTestSuite) TestHandlerRouterDeploymentTransactionType() {
	// First, create infrastructure deployment to get WETH and Factory addresses
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2")
	suite.Require().NoError(err)

	// Update with mock addresses (simulating successful infrastructure deployment)
	err = suite.uniswapService.UpdateWETHAddress(deploymentID, "0x1234567890123456789012345678901234567890")
	suite.Require().NoError(err)
	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, "0x0987654321098765432109876543210987654321")
	suite.Require().NoError(err)

	// Create test request for router deployment
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":       "v2",
				"deploy_router": true,
				"metadata": []interface{}{
					map[string]interface{}{
						"key":   "Deploy Type",
						"value": "Uniswap V2 Router",
					},
				},
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Verify response is successful
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.T().Logf("Unexpected error: %s", textContent.Text)
				suite.FailNow("Expected successful result but got error", textContent.Text)
			}
		}
		suite.FailNow("Expected successful result but got error with no content")
	}

	// Extract session ID from response
	var sessionIDContent string
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			sessionIDContent = textContent.Text
		}
	}
	sessionID := sessionIDContent[len("Transaction session created: "):]

	// Verify session was created in database
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)

	// Verify we have 1 deployment for router only
	suite.Len(session.TransactionDeployments, 1)

	// Verify Router deployment transaction type
	routerDeployment := session.TransactionDeployments[0]
	suite.Equal("Deploy UniswapV2Router02", routerDeployment.Title)
	suite.Equal("Deploy Uniswap V2 Router contract", routerDeployment.Description)
	suite.Equal("0", routerDeployment.Value)
	suite.Equal("", routerDeployment.Receiver)
	suite.NotEmpty(routerDeployment.Data)
	suite.Equal(models.TransactionTypeUniswapV2RouterDeployment, routerDeployment.TransactionType)

	// Verify the transaction data contains compiled bytecode
	suite.True(len(routerDeployment.Data) > 10, "Router transaction data should contain compiled bytecode")
	suite.True(routerDeployment.Data[:2] == "0x", "Router transaction data should be hex encoded")
}

func (suite *DeployUniswapToolTestSuite) TestHandlerInvalidVersion() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version": "v3", // Unsupported version
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "unsupported Uniswap version: v3")
	}
}

func (suite *DeployUniswapToolTestSuite) TestHandlerNoActiveChain() {
	// Deactivate the chain
	err := suite.db.GetDB().Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", false).Error
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version": "v2",
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "No active chain selected")
	}

	// Reactivate the chain for other tests
	err = suite.db.GetDB().Model(&models.Chain{}).Where("id = ?", suite.chain.ID).Update("is_active", true).Error
	suite.Require().NoError(err)
}

func (suite *DeployUniswapToolTestSuite) TestHandlerNonEthereumChain() {
	// Create a Solana chain and set it as active
	solanaChain := &models.Chain{
		ChainType: models.TransactionChainTypeSolana,
		RPC:       "https://solana-rpc.com",
		NetworkID: "101",
		Name:      "Solana Mainnet",
		IsActive:  false,
	}

	err := suite.chainService.CreateChain(solanaChain)
	suite.Require().NoError(err)

	err = suite.chainService.SetActiveChainByID(solanaChain.ID)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version": "v2",
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Uniswap deployment is only supported on Ethereum")
	}

	// Reactivate the Ethereum chain for other tests
	err = suite.chainService.SetActiveChainByID(suite.chain.ID)
	suite.Require().NoError(err)
}

func (suite *DeployUniswapToolTestSuite) TestHandlerRouterWithoutInfrastructure() {
	// Try to deploy router without existing WETH and Factory
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":       "v2",
				"deploy_router": true,
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Cannot deploy router: No existing Uniswap deployment found")
	}
}

func (suite *DeployUniswapToolTestSuite) TestHandlerDuplicateInfrastructureDeployment() {
	// First, create and mark infrastructure as deployed
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2")
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateWETHAddress(deploymentID, "0x1234567890123456789012345678901234567890")
	suite.Require().NoError(err)
	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, "0x0987654321098765432109876543210987654321")
	suite.Require().NoError(err)

	// Try to deploy infrastructure again
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":       "v2",
				"deploy_router": false,
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Uniswap v2 infrastructure is already deployed")
	}
}

func (suite *DeployUniswapToolTestSuite) TestHandlerDuplicateRouterDeployment() {
	// First, create infrastructure and router
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2")
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateWETHAddress(deploymentID, "0x1234567890123456789012345678901234567890")
	suite.Require().NoError(err)
	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, "0x0987654321098765432109876543210987654321")
	suite.Require().NoError(err)
	err = suite.uniswapService.UpdateRouterAddress(deploymentID, "0x1111111111111111111111111111111111111111")
	suite.Require().NoError(err)

	// Try to deploy router again
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":       "v2",
				"deploy_router": true,
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Router is already deployed at address")
	}
}

func (suite *DeployUniswapToolTestSuite) TestHandlerMissingRequiredFields() {
	// Test missing version
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				// version is intentionally omitted
				"deploy_router": false,
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid arguments")
	}
}

func (suite *DeployUniswapToolTestSuite) TestHandlerInvalidBindArguments() {
	// Create request with invalid argument structure
	invalidRequest := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: "invalid-json-structure", // Should be map[string]interface{}
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), invalidRequest)

	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to bind arguments")
}

func (suite *DeployUniswapToolTestSuite) TestHandlerMetadataInclusion() {
	metadata := []interface{}{
		map[string]interface{}{
			"key":   "Deploy Type",
			"value": "Uniswap V2 Infrastructure",
		},
		map[string]interface{}{
			"key":   "Network",
			"value": "Ethereum Testnet",
		},
	}

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":       "v2",
				"deploy_router": false,
				"metadata":      metadata,
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Extract session ID from response
	var sessionIDContent string
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			sessionIDContent = textContent.Text
		}
	}
	sessionID := sessionIDContent[len("Transaction session created: "):]

	// Verify session was created with metadata
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)

	// Verify metadata is included (plus the uniswap_deployment_id metadata added by the tool)
	suite.Len(session.Metadata, 3) // 2 from request + 1 added by tool

	// Check that our metadata is there
	foundDeployType := false
	foundNetwork := false
	for _, md := range session.Metadata {
		if md.Key == "Deploy Type" && md.Value == "Uniswap V2 Infrastructure" {
			foundDeployType = true
		}
		if md.Key == "Network" && md.Value == "Ethereum Testnet" {
			foundNetwork = true
		}
	}
	suite.True(foundDeployType, "Deploy Type metadata should be included")
	suite.True(foundNetwork, "Network metadata should be included")
}

func (suite *DeployUniswapToolTestSuite) TestToolRegistration() {
	// Test that the tool can be registered with an MCP server
	mcpServer := server.NewMCPServer("test", "1.0.0")

	tool := suite.deployUniswapTool.GetTool()
	handler := suite.deployUniswapTool.GetHandler()

	// This should not panic
	suite.NotPanics(func() {
		mcpServer.AddTool(tool, handler)
	})
}

func (suite *DeployUniswapToolTestSuite) TestURLGeneration() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"version":       "v2",
				"deploy_router": false,
			},
		},
	}

	handler := suite.deployUniswapTool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Check for errors first
	if result.IsError {
		if len(result.Content) > 0 {
			if textContent, ok := result.Content[0].(mcp.TextContent); ok {
				suite.T().Logf("Error content: %s", textContent.Text)
				suite.FailNow("Expected successful result but got error", textContent.Text)
			}
		}
		suite.FailNow("Expected successful result but got error with no content")
	}

	suite.Require().Len(result.Content, 3, "Expected 3 content items for successful deployment")

	// Extract the URL from the response
	var urlContent string
	if len(result.Content) > 2 {
		if textContent, ok := result.Content[2].(mcp.TextContent); ok {
			urlContent = textContent.Text
		}
	}
	expectedURLPrefix := fmt.Sprintf("http://localhost:%d/tx/", DEPLOY_UNISWAP_SERVER_PORT)
	suite.True(len(urlContent) > len(expectedURLPrefix))
	suite.Contains(urlContent, expectedURLPrefix)

	// Extract session ID from URL
	sessionID := urlContent[len(expectedURLPrefix):]
	suite.NotEmpty(sessionID)

	// Verify the session ID is a valid UUID format
	suite.True(len(sessionID) > 10) // Basic length check for UUID
}

func TestDeployUniswapToolTestSuite(t *testing.T) {
	suite.Run(t, new(DeployUniswapToolTestSuite))
}
