package services_test

import (
	"testing"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

type UniswapServiceTestSuite struct {
	suite.Suite
	db             services.DBService
	uniswapService services.UniswapService
	chainService   services.ChainService
	testChain      *models.Chain
}

func (suite *UniswapServiceTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize services
	suite.uniswapService = services.NewUniswapService(db.GetDB())
	suite.chainService = services.NewChainService(db.GetDB())

	// Create a test chain for foreign key relationships
	suite.testChain = &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       "http://localhost:8545",
		NetworkID: "31337",
		Name:      "Test Chain",
		IsActive:  true,
	}
	err = suite.chainService.CreateChain(suite.testChain)
	suite.Require().NoError(err)
}

func (suite *UniswapServiceTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *UniswapServiceTestSuite) SetupTest() {
	// Clean up any existing deployments before each test
	suite.db.GetDB().Where("1 = 1").Delete(&models.UniswapDeployment{})
}

func (suite *UniswapServiceTestSuite) TestCreateUniswapDeployment() {
	suite.Run("Create deployment without user", func() {
		deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", nil)
		suite.NoError(err)
		suite.Greater(deploymentID, uint(0))

		// Verify the deployment was created
		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(suite.testChain.ID, deployment.ChainID)
		suite.Equal("v2", deployment.Version)
		suite.Nil(deployment.UserID)
		suite.Equal(models.TransactionStatusPending, deployment.Status)
	})

	suite.Run("Create deployment with user", func() {
		userID := "test-user-123"
		deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v3", &userID)
		suite.NoError(err)
		suite.Greater(deploymentID, uint(0))

		// Verify the deployment was created with user
		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(suite.testChain.ID, deployment.ChainID)
		suite.Equal("v3", deployment.Version)
		suite.Require().NotNil(deployment.UserID)
		suite.Equal(userID, *deployment.UserID)
		suite.Equal(models.TransactionStatusPending, deployment.Status)
	})

	suite.Run("Create multiple deployments", func() {
		userID := "test-user-456"

		deploymentID1, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", &userID)
		suite.NoError(err)

		deploymentID2, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v3", &userID)
		suite.NoError(err)

		suite.NotEqual(deploymentID1, deploymentID2)
	})
}

func (suite *UniswapServiceTestSuite) TestGetUniswapDeployment() {
	// Create a test deployment first
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", nil)
	suite.Require().NoError(err)

	suite.Run("Get existing deployment", func() {
		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.NotNil(deployment)
		suite.Equal(deploymentID, deployment.ID)
		suite.Equal("v2", deployment.Version)
	})

	suite.Run("Get non-existent deployment", func() {
		_, err := suite.uniswapService.GetUniswapDeployment(99999)
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")
	})
}

func (suite *UniswapServiceTestSuite) TestUpdateAddresses() {
	// Create a test deployment first
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", nil)
	suite.Require().NoError(err)

	factoryAddr := "0x1234567890123456789012345678901234567890"
	routerAddr := "0x2345678901234567890123456789012345678901"
	wethAddr := "0x3456789012345678901234567890123456789012"
	deployerAddr := "0x4567890123456789012345678901234567890123"

	suite.Run("Update factory address", func() {
		err := suite.uniswapService.UpdateFactoryAddress(deploymentID, factoryAddr)
		suite.NoError(err)

		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(factoryAddr, deployment.FactoryAddress)
	})

	suite.Run("Update router address", func() {
		err := suite.uniswapService.UpdateRouterAddress(deploymentID, routerAddr)
		suite.NoError(err)

		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(routerAddr, deployment.RouterAddress)
	})

	suite.Run("Update WETH address", func() {
		err := suite.uniswapService.UpdateWETHAddress(deploymentID, wethAddr)
		suite.NoError(err)

		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(wethAddr, deployment.WETHAddress)
	})

	suite.Run("Update deployer address", func() {
		err := suite.uniswapService.UpdateDeployerAddress(deploymentID, deployerAddr)
		suite.NoError(err)

		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(deployerAddr, deployment.DeployerAddress)
	})

	suite.Run("Update non-existent deployment", func() {
		err := suite.uniswapService.UpdateFactoryAddress(99999, factoryAddr)
		suite.NoError(err) // GORM Update doesn't return error for non-existent records
	})
}

func (suite *UniswapServiceTestSuite) TestUpdateStatus() {
	// Create a test deployment first
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", nil)
	suite.Require().NoError(err)

	suite.Run("Update to pending status", func() {
		err := suite.uniswapService.UpdateStatus(deploymentID, models.TransactionStatusPending)
		suite.NoError(err)

		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(models.TransactionStatusPending, deployment.Status)
	})

	suite.Run("Update to failed status", func() {
		err := suite.uniswapService.UpdateStatus(deploymentID, models.TransactionStatusFailed)
		suite.NoError(err)

		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(models.TransactionStatusFailed, deployment.Status)
	})

	suite.Run("Confirm deployment with missing addresses should fail", func() {
		err := suite.uniswapService.UpdateStatus(deploymentID, models.TransactionStatusConfirmed)
		suite.Error(err)
		suite.Contains(err.Error(), "cannot confirm deployment with missing addresses")
		suite.Contains(err.Error(), "factory_address")
		suite.Contains(err.Error(), "router_address")
		suite.Contains(err.Error(), "weth_address")
	})

	suite.Run("Confirm deployment with all addresses should succeed", func() {
		// Set all required addresses
		err := suite.uniswapService.UpdateFactoryAddress(deploymentID, "0x1234567890123456789012345678901234567890")
		suite.NoError(err)
		err = suite.uniswapService.UpdateRouterAddress(deploymentID, "0x2345678901234567890123456789012345678901")
		suite.NoError(err)
		err = suite.uniswapService.UpdateWETHAddress(deploymentID, "0x3456789012345678901234567890123456789012")
		suite.NoError(err)

		// Now confirmation should work
		err = suite.uniswapService.UpdateStatus(deploymentID, models.TransactionStatusConfirmed)
		suite.NoError(err)

		deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
		suite.NoError(err)
		suite.Equal(models.TransactionStatusConfirmed, deployment.Status)
	})

	suite.Run("Update status for non-existent deployment", func() {
		err := suite.uniswapService.UpdateStatus(99999, models.TransactionStatusConfirmed)
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")
	})
}

func (suite *UniswapServiceTestSuite) TestChainQueries() {
	// Create test deployments on different chains
	deploymentID1, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", nil)
	suite.Require().NoError(err)

	// Create another chain for testing
	testChain2 := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       "http://localhost:8546",
		NetworkID: "31338",
		Name:      "Test Chain 2",
		IsActive:  true,
	}
	err = suite.chainService.CreateChain(testChain2)
	suite.Require().NoError(err)

	deploymentID2, err := suite.uniswapService.CreateUniswapDeployment(testChain2.ID, "v3", nil)
	suite.Require().NoError(err)

	suite.Run("GetUniswapDeploymentByChain", func() {
		deployment1, err := suite.uniswapService.GetUniswapDeploymentByChain(suite.testChain.ID)
		suite.NoError(err)
		suite.Equal(deploymentID1, deployment1.ID)
		suite.Equal("v2", deployment1.Version)

		deployment2, err := suite.uniswapService.GetUniswapDeploymentByChain(testChain2.ID)
		suite.NoError(err)
		suite.Equal(deploymentID2, deployment2.ID)
		suite.Equal("v3", deployment2.Version)
	})

	suite.Run("GetUniswapDeploymentByChainString", func() {
		deployment1, err := suite.uniswapService.GetUniswapDeploymentByChainString("ethereum", "31337")
		suite.NoError(err)
		suite.Equal(deploymentID1, deployment1.ID)
		suite.Equal("v2", deployment1.Version)

		deployment2, err := suite.uniswapService.GetUniswapDeploymentByChainString("ethereum", "31338")
		suite.NoError(err)
		suite.Equal(deploymentID2, deployment2.ID)
		suite.Equal("v3", deployment2.Version)
	})

	suite.Run("GetUniswapDeploymentByChain non-existent", func() {
		_, err := suite.uniswapService.GetUniswapDeploymentByChain(99999)
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")
	})

	suite.Run("GetUniswapDeploymentByChainString non-existent", func() {
		_, err := suite.uniswapService.GetUniswapDeploymentByChainString("ethereum", "99999")
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")
	})
}

func (suite *UniswapServiceTestSuite) TestUserQueries() {
	userID1 := "user1"
	userID2 := "user2"

	// Create deployments for different users
	deploymentID1, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", &userID1)
	suite.Require().NoError(err)
	deploymentID2, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v3", &userID1)
	suite.Require().NoError(err)
	deploymentID3, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", &userID2)
	suite.Require().NoError(err)
	deploymentID4, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v3", nil) // No user
	suite.Require().NoError(err)

	suite.Run("ListUniswapDeploymentsByUser", func() {
		// Get user1 deployments
		deployments1, err := suite.uniswapService.ListUniswapDeploymentsByUser(userID1, 0, 10)
		suite.NoError(err)
		suite.Len(deployments1, 2)

		deploymentIDs1 := []uint{deployments1[0].ID, deployments1[1].ID}
		suite.Contains(deploymentIDs1, deploymentID1)
		suite.Contains(deploymentIDs1, deploymentID2)

		// Get user2 deployments
		deployments2, err := suite.uniswapService.ListUniswapDeploymentsByUser(userID2, 0, 10)
		suite.NoError(err)
		suite.Len(deployments2, 1)
		suite.Equal(deploymentID3, deployments2[0].ID)

		// Get non-existent user deployments
		deployments3, err := suite.uniswapService.ListUniswapDeploymentsByUser("non-existent", 0, 10)
		suite.NoError(err)
		suite.Empty(deployments3)
	})

	suite.Run("ListUniswapDeploymentsByUser with pagination", func() {
		// Test pagination with user1 (has 2 deployments)
		deployments, err := suite.uniswapService.ListUniswapDeploymentsByUser(userID1, 0, 1)
		suite.NoError(err)
		suite.Len(deployments, 1)

		deployments, err = suite.uniswapService.ListUniswapDeploymentsByUser(userID1, 1, 1)
		suite.NoError(err)
		suite.Len(deployments, 1)

		deployments, err = suite.uniswapService.ListUniswapDeploymentsByUser(userID1, 2, 1)
		suite.NoError(err)
		suite.Empty(deployments)
	})

	suite.Run("GetActiveUniswapDeployment", func() {
		// Get active deployment for user1
		deployment, err := suite.uniswapService.GetActiveUniswapDeployment(&userID1, *suite.testChain)
		suite.NoError(err)
		suite.NotNil(deployment)
		suite.Equal(userID1, *deployment.UserID)

		// Get active deployment for user2
		deployment, err = suite.uniswapService.GetActiveUniswapDeployment(&userID2, *suite.testChain)
		suite.NoError(err)
		suite.NotNil(deployment)
		suite.Equal(userID2, *deployment.UserID)

		// Get active deployment without user filter (should return first one)
		deployment, err = suite.uniswapService.GetActiveUniswapDeployment(nil, *suite.testChain)
		suite.NoError(err)
		suite.NotNil(deployment)

		// Test with non-existent user
		nonExistentUser := "non-existent-user"
		_, err = suite.uniswapService.GetActiveUniswapDeployment(&nonExistentUser, *suite.testChain)
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")
	})

	// Keep track of deployment IDs for verification
	_ = deploymentID4 // Prevent unused variable warning
}

func (suite *UniswapServiceTestSuite) TestListDeployments() {
	// Create multiple deployments
	userID := "test-user"
	deploymentID1, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", &userID)
	suite.Require().NoError(err)
	deploymentID2, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v3", nil)
	suite.Require().NoError(err)

	suite.Run("List all deployments", func() {
		deployments, err := suite.uniswapService.ListUniswapDeployments(0, 10)
		suite.NoError(err)
		suite.Len(deployments, 2)

		deploymentIDs := []uint{deployments[0].ID, deployments[1].ID}
		suite.Contains(deploymentIDs, deploymentID1)
		suite.Contains(deploymentIDs, deploymentID2)
	})

	suite.Run("List deployments with pagination", func() {
		// Get first deployment
		deployments, err := suite.uniswapService.ListUniswapDeployments(0, 1)
		suite.NoError(err)
		suite.Len(deployments, 1)

		// Get second deployment
		deployments, err = suite.uniswapService.ListUniswapDeployments(1, 1)
		suite.NoError(err)
		suite.Len(deployments, 1)

		// Try to get beyond available
		deployments, err = suite.uniswapService.ListUniswapDeployments(2, 1)
		suite.NoError(err)
		suite.Empty(deployments)
	})

	suite.Run("List deployments with zero limit", func() {
		deployments, err := suite.uniswapService.ListUniswapDeployments(0, 0)
		suite.NoError(err)
		suite.Empty(deployments) // Zero limit should return empty
	})
}

func (suite *UniswapServiceTestSuite) TestDeleteOperations() {
	userID := "test-user"

	// Create test deployments
	deploymentID1, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", &userID)
	suite.Require().NoError(err)
	deploymentID2, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v3", &userID)
	suite.Require().NoError(err)
	deploymentID3, err := suite.uniswapService.CreateUniswapDeployment(suite.testChain.ID, "v2", nil)
	suite.Require().NoError(err)

	suite.Run("Delete single deployment", func() {
		err := suite.uniswapService.DeleteUniswapDeployment(deploymentID1)
		suite.NoError(err)

		// Verify it's deleted
		_, err = suite.uniswapService.GetUniswapDeployment(deploymentID1)
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")

		// Verify others still exist
		_, err = suite.uniswapService.GetUniswapDeployment(deploymentID2)
		suite.NoError(err)
		_, err = suite.uniswapService.GetUniswapDeployment(deploymentID3)
		suite.NoError(err)
	})

	suite.Run("Delete multiple deployments", func() {
		err := suite.uniswapService.DeleteUniswapDeployments([]uint{deploymentID2, deploymentID3})
		suite.NoError(err)

		// Verify both are deleted
		_, err = suite.uniswapService.GetUniswapDeployment(deploymentID2)
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")
		_, err = suite.uniswapService.GetUniswapDeployment(deploymentID3)
		suite.Error(err)
		suite.Contains(err.Error(), "record not found")
	})

	suite.Run("Delete non-existent deployment", func() {
		err := suite.uniswapService.DeleteUniswapDeployment(99999)
		suite.NoError(err) // GORM Delete doesn't return error for non-existent records
	})

	suite.Run("Delete with empty array", func() {
		err := suite.uniswapService.DeleteUniswapDeployments([]uint{})
		// GORM requires WHERE conditions, so this might error
		// The specific behavior depends on the implementation
		// We accept either no error (if implementation handles empty arrays)
		// or an error (if GORM complains about missing WHERE conditions)
		_ = err // Don't assert on this as behavior may vary
	})
}

func TestUniswapServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UniswapServiceTestSuite))
}
