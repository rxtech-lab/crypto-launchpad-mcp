package hooks

import (
	"testing"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

type UniswapDeploymentHookTestSuite struct {
	suite.Suite
	dbService      services.DBService
	uniswapService services.UniswapService
	chainService   services.ChainService
	hook           *UniswapDeploymentHook
	chain          *models.Chain
}

func (suite *UniswapDeploymentHookTestSuite) SetupSuite() {
	// Initialize in-memory database for testing
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.dbService = db

	// Initialize services
	suite.uniswapService = services.NewUniswapService(db.GetDB())
	suite.chainService = services.NewChainService(db.GetDB())

	// Initialize hook
	suite.hook = &UniswapDeploymentHook{
		db:             db.GetDB(),
		uniswapService: suite.uniswapService,
	}

	// Setup test chain
	suite.setupTestData()
}

func (suite *UniswapDeploymentHookTestSuite) TearDownSuite() {
	if suite.dbService != nil {
		suite.dbService.Close()
	}
}

func (suite *UniswapDeploymentHookTestSuite) SetupTest() {
	// Clean up test data between tests
	suite.cleanupTestData()
}

func (suite *UniswapDeploymentHookTestSuite) setupTestData() {
	// Create test chain
	chain := &models.Chain{
		ID:        1,
		Name:      "Ethereum",
		NetworkID: "1",
		ChainType: models.TransactionChainTypeEthereum,
		IsActive:  true,
		RPC:       "http://localhost:8545",
	}
	err := suite.dbService.GetDB().Create(chain).Error
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *UniswapDeploymentHookTestSuite) cleanupTestData() {
	suite.dbService.GetDB().Where("1 = 1").Delete(&models.UniswapDeployment{})
	suite.dbService.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})
}

func (suite *UniswapDeploymentHookTestSuite) TestCanHandle() {
	// Test that hook can handle Uniswap deployment transaction types
	suite.True(suite.hook.CanHandle(models.TransactionTypeUniswapV2TokenDeployment))
	suite.True(suite.hook.CanHandle(models.TransactionTypeUniswapV2FactoryDeployment))
	suite.True(suite.hook.CanHandle(models.TransactionTypeUniswapV2RouterDeployment))

	// Test that hook cannot handle other transaction types
	suite.False(suite.hook.CanHandle(models.TransactionTypeTokenDeployment))
	suite.False(suite.hook.CanHandle("unknown_type"))
}

func (suite *UniswapDeploymentHookTestSuite) TestOnTransactionConfirmed_WETHDeployment() {
	// Create a Uniswap deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Create transaction session
	session := models.TransactionSession{
		ID:                   "test-session",
		ChainID:              suite.chain.ID,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
	}

	// Test WETH deployment
	contractAddress := "0x1234567890123456789012345678901234567890"
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2TokenDeployment,
		"0xabcd1234",
		contractAddress,
		session,
	)
	suite.NoError(err)

	// Verify WETH address was updated
	deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Equal(contractAddress, deployment.WETHAddress)
	suite.Empty(deployment.FactoryAddress)
	suite.Empty(deployment.RouterAddress)
	suite.Equal(models.TransactionStatusPending, deployment.Status)
}

func (suite *UniswapDeploymentHookTestSuite) TestOnTransactionConfirmed_FactoryDeployment() {
	// Create a Uniswap deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Create transaction session
	session := models.TransactionSession{
		ID:                   "test-session",
		ChainID:              suite.chain.ID,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
	}

	// Test Factory deployment
	contractAddress := "0xabcdef1234567890123456789012345678901234"
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2FactoryDeployment,
		"0xdef5678",
		contractAddress,
		session,
	)
	suite.NoError(err)

	// Verify Factory address was updated
	deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Equal(contractAddress, deployment.FactoryAddress)
	suite.Empty(deployment.WETHAddress)
	suite.Empty(deployment.RouterAddress)
	suite.Equal(models.TransactionStatusPending, deployment.Status)
}

func (suite *UniswapDeploymentHookTestSuite) TestOnTransactionConfirmed_RouterDeployment() {
	// Create a Uniswap deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Create transaction session
	session := models.TransactionSession{
		ID:                   "test-session",
		ChainID:              suite.chain.ID,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
	}

	// Test Router deployment
	contractAddress := "0x9876543210987654321098765432109876543210"
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2RouterDeployment,
		"0x987654",
		contractAddress,
		session,
	)
	suite.NoError(err)

	// Verify Router address was updated
	deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Equal(contractAddress, deployment.RouterAddress)
	suite.Empty(deployment.WETHAddress)
	suite.Empty(deployment.FactoryAddress)
	suite.Equal(models.TransactionStatusPending, deployment.Status)
}

func (suite *UniswapDeploymentHookTestSuite) TestOnTransactionConfirmed_AllAddressesSet() {
	// Create a Uniswap deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Create transaction session
	session := models.TransactionSession{
		ID:                   "test-session",
		ChainID:              suite.chain.ID,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
	}

	// Deploy all three contracts
	wethAddress := "0x1111111111111111111111111111111111111111"
	factoryAddress := "0x2222222222222222222222222222222222222222"
	routerAddress := "0x3333333333333333333333333333333333333333"

	// Deploy WETH
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2TokenDeployment,
		"0xweth",
		wethAddress,
		session,
	)
	suite.NoError(err)

	// Verify status is still pending
	deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Equal(models.TransactionStatusPending, deployment.Status)

	// Deploy Factory
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2FactoryDeployment,
		"0xfactory",
		factoryAddress,
		session,
	)
	suite.NoError(err)

	// Verify status is still pending
	deployment, err = suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Equal(models.TransactionStatusPending, deployment.Status)

	// Deploy Router (this should trigger confirmation)
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2RouterDeployment,
		"0xrouter",
		routerAddress,
		session,
	)
	suite.NoError(err)

	// Verify all addresses are set and status is confirmed
	deployment, err = suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Equal(wethAddress, deployment.WETHAddress)
	suite.Equal(factoryAddress, deployment.FactoryAddress)
	suite.Equal(routerAddress, deployment.RouterAddress)
	suite.Equal(models.TransactionStatusConfirmed, deployment.Status)
}

func (suite *UniswapDeploymentHookTestSuite) TestOnTransactionConfirmed_AllAddressesSetDifferentOrder() {
	// Create a Uniswap deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Create transaction session
	session := models.TransactionSession{
		ID:                   "test-session",
		ChainID:              suite.chain.ID,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
	}

	// Deploy contracts in different order: Factory -> Router -> WETH
	wethAddress := "0x1111111111111111111111111111111111111111"
	factoryAddress := "0x2222222222222222222222222222222222222222"
	routerAddress := "0x3333333333333333333333333333333333333333"

	// Deploy Factory first
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2FactoryDeployment,
		"0xfactory",
		factoryAddress,
		session,
	)
	suite.NoError(err)

	// Deploy Router second
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2RouterDeployment,
		"0xrouter",
		routerAddress,
		session,
	)
	suite.NoError(err)

	// Deploy WETH last (this should trigger confirmation)
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2TokenDeployment,
		"0xweth",
		wethAddress,
		session,
	)
	suite.NoError(err)

	// Verify all addresses are set and status is confirmed
	deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Equal(wethAddress, deployment.WETHAddress)
	suite.Equal(factoryAddress, deployment.FactoryAddress)
	suite.Equal(routerAddress, deployment.RouterAddress)
	suite.Equal(models.TransactionStatusConfirmed, deployment.Status)
}

func (suite *UniswapDeploymentHookTestSuite) TestOnTransactionConfirmed_NoDeploymentFound() {
	// Create transaction session with non-existent chain
	session := models.TransactionSession{
		ID:                   "test-session",
		ChainID:              999, // Non-existent chain
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
	}

	// Test should return error when no deployment found
	err := suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2TokenDeployment,
		"0xabcd1234",
		"0x1234567890123456789012345678901234567890",
		session,
	)
	suite.Error(err)
	suite.Contains(err.Error(), "record not found")
}

func (suite *UniswapDeploymentHookTestSuite) TestOnTransactionConfirmed_ServiceError() {
	// Create a Uniswap deployment
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.chain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Create transaction session
	session := models.TransactionSession{
		ID:                   "test-session",
		ChainID:              suite.chain.ID,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
	}

	// Test with invalid contract address (empty string)
	err = suite.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2TokenDeployment,
		"0xabcd1234",
		"", // Empty contract address
		session,
	)

	// The hook should still succeed as it's just storing the address
	suite.NoError(err)

	// Verify empty address was stored
	deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.Empty(deployment.WETHAddress)
}

func TestUniswapDeploymentHookTestSuite(t *testing.T) {
	suite.Run(t, new(UniswapDeploymentHookTestSuite))
}
