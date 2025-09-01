package tools

import (
	"context"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

type AddLiquidityToolTestSuite struct {
	suite.Suite
	dbService              services.DBService
	tool                   *addLiquidityTool
	liquidityService       services.LiquidityService
	uniswapService         services.UniswapService
	txService              services.TransactionService
	evmService             services.EvmService
	chainService           services.ChainService
	uniswapSettingsService services.UniswapSettingsService
	chain                  *models.Chain
	pool                   *models.LiquidityPool
	uniswapDeployment      *models.UniswapDeployment
}

func (suite *AddLiquidityToolTestSuite) SetupSuite() {
	// Initialize in-memory database
	dbService, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.dbService = dbService

	// Initialize services
	suite.evmService = services.NewEvmService()
	suite.txService = services.NewTransactionService(dbService.GetDB())
	suite.liquidityService = services.NewLiquidityService(dbService.GetDB())
	suite.uniswapService = services.NewUniswapService(dbService.GetDB())
	suite.chainService = services.NewChainService(dbService.GetDB())
	suite.uniswapSettingsService = services.NewUniswapSettingsService(dbService.GetDB())

	// Initialize tool
	suite.tool = NewAddLiquidityTool(
		suite.chainService,
		9999, // test server port
		suite.evmService,
		suite.txService,
		suite.liquidityService,
		suite.uniswapService,
		suite.uniswapSettingsService,
	)

	// Setup test data
	suite.setupTestChain()
	suite.setupUniswapSettings()
	suite.setupUniswapDeployment()
	suite.setupTestPool()
}

func (suite *AddLiquidityToolTestSuite) TearDownSuite() {
	if suite.dbService != nil {
		suite.dbService.Close()
	}
}

func (suite *AddLiquidityToolTestSuite) SetupTest() {
	// Clean up any existing sessions for each test
	suite.cleanupTestData()
}

func (suite *AddLiquidityToolTestSuite) setupTestChain() {
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

func (suite *AddLiquidityToolTestSuite) setupUniswapSettings() {
	err := suite.uniswapSettingsService.SetUniswapVersion("v2")
	suite.Require().NoError(err)
}

func (suite *AddLiquidityToolTestSuite) setupUniswapDeployment() {
	// Create Uniswap deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2")
	suite.Require().NoError(err)

	// Update with addresses
	err = suite.uniswapService.UpdateWETHAddress(deploymentID, "0x1111111111111111111111111111111111111111")
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, "0x2222222222222222222222222222222222222222")
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateRouterAddress(deploymentID, "0x3333333333333333333333333333333333333333")
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateDeployerAddress(deploymentID, "0x4444444444444444444444444444444444444444")
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateStatus(deploymentID, models.TransactionStatusConfirmed)
	suite.Require().NoError(err)

	suite.uniswapDeployment, err = suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
}

func (suite *AddLiquidityToolTestSuite) setupTestPool() {
	pool := &models.LiquidityPool{
		TokenAddress:   "0x5555555555555555555555555555555555555555",
		UniswapVersion: "v2",
		Token0:         "0x5555555555555555555555555555555555555555",
		Token1:         "0x1111111111111111111111111111111111111111", // WETH
		InitialToken0:  "1000000000000000000",
		InitialToken1:  "1000000000000000000",
		PairAddress:    "0x6666666666666666666666666666666666666666",
		Status:         models.TransactionStatusConfirmed,
	}

	poolID, err := suite.liquidityService.CreateLiquidityPool(pool)
	suite.Require().NoError(err)

	pool.ID = poolID
	suite.pool = pool
}

func (suite *AddLiquidityToolTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.dbService.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})

	// Clean up liquidity positions
	suite.dbService.GetDB().Where("1 = 1").Delete(&models.LiquidityPosition{})
}

// Test cases

func (suite *AddLiquidityToolTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()

	suite.Equal("add_liquidity", tool.Name)
	suite.Contains(tool.Description, "Add liquidity to existing Uniswap pool")
	suite.Contains(tool.Description, "signing interface")

	// Check required parameters
	suite.NotNil(tool.InputSchema)
	properties := tool.InputSchema.Properties

	// Check token_address parameter
	tokenAddressProp, exists := properties["token_address"]
	suite.True(exists)
	if propMap, ok := tokenAddressProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}

	// Check token_amount parameter
	tokenAmountProp, exists := properties["token_amount"]
	suite.True(exists)
	if propMap, ok := tokenAmountProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}

	// Check eth_amount parameter
	ethAmountProp, exists := properties["eth_amount"]
	suite.True(exists)
	if propMap, ok := ethAmountProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}

	// Check min_token_amount parameter
	minTokenProp, exists := properties["min_token_amount"]
	suite.True(exists)
	if propMap, ok := minTokenProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}

	// Check min_eth_amount parameter
	minEthProp, exists := properties["min_eth_amount"]
	suite.True(exists)
	if propMap, ok := minEthProp.(map[string]any); ok {
		suite.Equal("string", propMap["type"])
	}
}

func (suite *AddLiquidityToolTestSuite) TestHandlerSuccess() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    suite.pool.TokenAddress,
				"token_amount":     "500000000000000000", // 0.5 token
				"eth_amount":       "500000000000000000", // 0.5 ETH
				"min_token_amount": "490000000000000000", // 0.49 token (2% slippage)
				"min_eth_amount":   "490000000000000000", // 0.49 ETH (2% slippage)
				"metadata": []interface{}{
					map[string]interface{}{
						"key":   "Liquidity Provider",
						"value": "Test Provider",
					},
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)

	// Check for errors
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

	// Verify response format matches launch.go pattern
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Transaction session created:")
	}

	if textContent, ok := result.Content[1].(mcp.TextContent); ok {
		suite.Equal("Please return the following url to the user:", textContent.Text)
	}

	if textContent, ok := result.Content[2].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "http://localhost:9999/tx/")
	}

	// Verify position was created
	positions, err := suite.liquidityService.GetLiquidityPositionsByPool(suite.pool.ID)
	suite.NoError(err)
	suite.Len(positions, 1)
	suite.Equal("500000000000000000", positions[0].Token0Amount)
	suite.Equal("500000000000000000", positions[0].Token1Amount)
	suite.Equal("add", positions[0].Action)
}

func (suite *AddLiquidityToolTestSuite) TestHandlerPoolNotFound() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"token_amount":     "1000000000000000000",
				"eth_amount":       "1000000000000000000",
				"min_token_amount": "990000000000000000",
				"min_eth_amount":   "990000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Liquidity pool not found")
	}
}

func (suite *AddLiquidityToolTestSuite) TestHandlerPoolNotConfirmed() {
	// Create a pending pool
	pendingPool := &models.LiquidityPool{
		TokenAddress:   "0x7777777777777777777777777777777777777777",
		UniswapVersion: "v2",
		Token0:         "0x7777777777777777777777777777777777777777",
		Token1:         "0x1111111111111111111111111111111111111111",
		Status:         models.TransactionStatusPending, // Not confirmed
	}

	_, err := suite.liquidityService.CreateLiquidityPool(pendingPool)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    pendingPool.TokenAddress,
				"token_amount":     "1000000000000000000",
				"eth_amount":       "1000000000000000000",
				"min_token_amount": "990000000000000000",
				"min_eth_amount":   "990000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "not confirmed yet")
	}
}

func (suite *AddLiquidityToolTestSuite) TestHandlerPoolNoPairAddress() {
	// Create a pool without pair address
	poolNoPair := &models.LiquidityPool{
		TokenAddress:   "0x8888888888888888888888888888888888888888",
		UniswapVersion: "v2",
		Token0:         "0x8888888888888888888888888888888888888888",
		Token1:         "0x1111111111111111111111111111111111111111",
		Status:         models.TransactionStatusConfirmed,
		PairAddress:    "", // No pair address
	}

	_, err := suite.liquidityService.CreateLiquidityPool(poolNoPair)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    poolNoPair.TokenAddress,
				"token_amount":     "1000000000000000000",
				"eth_amount":       "1000000000000000000",
				"min_token_amount": "990000000000000000",
				"min_eth_amount":   "990000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "does not have a pair address")
	}
}

func (suite *AddLiquidityToolTestSuite) TestHandlerNoRouterAddress() {
	// Delete the existing deployment and create one without router
	err := suite.uniswapService.DeleteUniswapDeployment(suite.uniswapDeployment.ID)
	suite.Require().NoError(err)

	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2")
	suite.Require().NoError(err)

	// Only set WETH and Factory, but not Router
	err = suite.uniswapService.UpdateWETHAddress(deploymentID, "0x1111111111111111111111111111111111111111")
	suite.Require().NoError(err)

	err = suite.uniswapService.UpdateFactoryAddress(deploymentID, "0x2222222222222222222222222222222222222222")
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    suite.pool.TokenAddress,
				"token_amount":     "1000000000000000000",
				"eth_amount":       "1000000000000000000",
				"min_token_amount": "990000000000000000",
				"min_eth_amount":   "990000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "router address not found")
	}

	// Clean up and restore the deployment
	err = suite.uniswapService.DeleteUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.setupUniswapDeployment()
}

func (suite *AddLiquidityToolTestSuite) TestHandlerInvalidArguments() {
	// Test missing token_address
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_amount":     "1000000000000000000",
				"eth_amount":       "1000000000000000000",
				"min_token_amount": "990000000000000000",
				"min_eth_amount":   "990000000000000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "Invalid arguments")
	}
}

func (suite *AddLiquidityToolTestSuite) TestHandlerNonEthereumChain() {
	// Create a Solana chain and activate it
	solanaChain := &models.Chain{
		ChainType: models.TransactionChainTypeSolana,
		RPC:       "https://api.devnet.solana.com",
		NetworkID: "devnet",
		Name:      "Solana Devnet",
		IsActive:  false,
	}

	err := suite.chainService.CreateChain(solanaChain)
	suite.Require().NoError(err)

	// Set Solana chain as active (this will deactivate the Ethereum chain)
	err = suite.chainService.SetActiveChainByID(solanaChain.ID)
	suite.Require().NoError(err)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    "DezXAZ8z7PnrnRJjz3wXBoRgixCa6xjnB7YaB1pPB263",
				"token_amount":     "1000000000",
				"eth_amount":       "1000000000",
				"min_token_amount": "990000000",
				"min_eth_amount":   "990000000",
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.True(result.IsError)

	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		suite.Contains(textContent.Text, "only supported on Ethereum")
	}

	// Clean up - reactivate Ethereum chain for other tests
	err = suite.chainService.SetActiveChainByID(suite.chain.ID)
	suite.Require().NoError(err)
}

func (suite *AddLiquidityToolTestSuite) TestMetadataEnhancement() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"token_address":    suite.pool.TokenAddress,
				"token_amount":     "1000000000000000000",
				"eth_amount":       "2000000000000000000",
				"min_token_amount": "990000000000000000",
				"min_eth_amount":   "1980000000000000000",
				"metadata": []interface{}{
					map[string]interface{}{
						"key":   "custom_key",
						"value": "custom_value",
					},
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Extract session ID from response
	var sessionID string
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		text := textContent.Text
		suite.Contains(text, "Transaction session created:")
		sessionID = text[len("Transaction session created: "):]
	}

	// Get session and verify metadata
	session, err := suite.txService.GetTransactionSession(sessionID)
	suite.NoError(err)
	suite.NotNil(session)

	// Check that metadata contains both user and system metadata
	metadataMap := make(map[string]string)
	for _, m := range session.Metadata {
		metadataMap[m.Key] = m.Value
	}

	// Check user metadata
	suite.Equal("custom_value", metadataMap["custom_key"])

	// Check system-added metadata
	suite.NotEmpty(metadataMap["position_id"])
	suite.Equal(fmt.Sprintf("%d", suite.pool.ID), metadataMap["pool_id"])
	suite.Equal(suite.pool.PairAddress, metadataMap["pool_pair_address"])
	suite.Equal("v2", metadataMap["uniswap_version"])
	suite.Equal("add_liquidity", metadataMap["action"])
	suite.Equal(suite.pool.TokenAddress, metadataMap["token_address"])
	suite.Equal("1000000000000000000", metadataMap["token0_amount"])
	suite.Equal("2000000000000000000", metadataMap["token1_amount"])

	// Check transaction deployment
	suite.Len(session.TransactionDeployments, 1)
	deployment := session.TransactionDeployments[0]
	suite.Equal("Add Liquidity", deployment.Title)
	suite.Contains(deployment.Description, suite.pool.TokenAddress)
	suite.Equal("2000000000000000000", deployment.Value) // ETH amount
	suite.Equal(suite.uniswapDeployment.RouterAddress, deployment.Receiver)
}

func (suite *AddLiquidityToolTestSuite) TestToolRegistration() {
	// Test that the tool can be registered with an MCP server
	mcpServer := server.NewMCPServer("test", "1.0.0")

	tool := suite.tool.GetTool()
	handler := suite.tool.GetHandler()

	// This should not panic
	suite.NotPanics(func() {
		mcpServer.AddTool(tool, handler)
	})
}

// Test runner
func TestAddLiquidityToolTestSuite(t *testing.T) {
	suite.Run(t, new(AddLiquidityToolTestSuite))
}
