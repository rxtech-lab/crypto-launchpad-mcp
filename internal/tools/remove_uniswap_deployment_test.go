package tools

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type RemoveUniswapDeploymentTestSuite struct {
	suite.Suite
	db             interface{}
	uniswapService services.UniswapService
	tool           *removeUniswapDeploymentTool
}

func (suite *RemoveUniswapDeploymentTestSuite) SetupSuite() {
	// Initialize in-memory database for testing
	db, err := database.NewDatabase(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize services
	suite.uniswapService = services.NewUniswapService(db.DB)

	// Initialize tool
	suite.tool = NewRemoveUniswapDeploymentTool(db, suite.uniswapService)

	// Setup test data
	suite.setupTestData()
}

func (suite *RemoveUniswapDeploymentTestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

func (suite *RemoveUniswapDeploymentTestSuite) SetupTest() {
	// Clean up any test-specific data between tests
	suite.cleanupTestData()
}

func (suite *RemoveUniswapDeploymentTestSuite) setupTestData() {
	// Create test chain
	testChain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		Name:      "Test Ethereum",
		RPC:       "http://localhost:8545",
		NetworkID: "31337",
		IsActive:  true,
	}
	err := suite.db.DB.Create(testChain).Error
	suite.Require().NoError(err)
}

func (suite *RemoveUniswapDeploymentTestSuite) cleanupTestData() {
	// Clean up deployments and sessions
	suite.db.DB.Where("1 = 1").Delete(&models.UniswapDeployment{})
}

func (suite *RemoveUniswapDeploymentTestSuite) createTestDeployment(chainID uint, version string) uint {
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(chainID, version)
	suite.Require().NoError(err)
	return deploymentID
}

func (suite *RemoveUniswapDeploymentTestSuite) TestGetTool() {
	tool := suite.tool.GetTool()
	suite.Equal("remove_uniswap_deployment", tool.Name)
	suite.Contains(tool.Description, "Remove one or multiple Uniswap deployments")

	// Check parameters
	suite.Require().Len(tool.InputSchema.Properties, 1)

	idsParam, exists := tool.InputSchema.Properties["ids"]
	suite.True(exists)

	// Type assert to get the parameter properties
	if param, ok := idsParam.(map[string]any); ok {
		suite.Equal("array", param["type"])
		suite.Contains(param["description"], "Array of Uniswap deployment IDs")
	}
}

func (suite *RemoveUniswapDeploymentTestSuite) TestHandlerSuccess_SingleDeployment() {
	// Get test chain
	var testChain models.Chain
	err := suite.db.DB.Where("chain_type = ?", models.TransactionChainTypeEthereum).First(&testChain).Error
	suite.Require().NoError(err)

	// Create test deployment
	deploymentID := suite.createTestDeployment(testChain.ID, "v2")

	// Verify deployment exists
	deployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Require().NoError(err)
	suite.NotNil(deployment)

	// Create request with single ID
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"ids": []interface{}{float64(deploymentID)}, // JSON numbers come as float64
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Verify deployment was deleted from database
	deletedDeployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Error(err)
	suite.True(err == gorm.ErrRecordNotFound || deletedDeployment == nil)
}

func (suite *RemoveUniswapDeploymentTestSuite) TestHandlerSuccess_BulkDeployments() {
	// Get test chain
	var testChain models.Chain
	err := suite.db.DB.Where("chain_type = ?", models.TransactionChainTypeEthereum).First(&testChain).Error
	suite.Require().NoError(err)

	// Create multiple test deployments
	deployment1ID := suite.createTestDeployment(testChain.ID, "v2")
	deployment2ID := suite.createTestDeployment(testChain.ID, "v2")
	deployment3ID := suite.createTestDeployment(testChain.ID, "v2")

	// Verify all deployments exist
	for _, id := range []uint{deployment1ID, deployment2ID, deployment3ID} {
		deployment, err := suite.uniswapService.GetUniswapDeployment(id)
		suite.Require().NoError(err)
		suite.NotNil(deployment)
	}

	// Create request with multiple IDs
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"ids": []interface{}{
					float64(deployment1ID),
					float64(deployment2ID),
					float64(deployment3ID),
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Verify all deployments were deleted from database
	for _, id := range []uint{deployment1ID, deployment2ID, deployment3ID} {
		deletedDeployment, err := suite.uniswapService.GetUniswapDeployment(id)
		suite.Error(err)
		suite.True(err == gorm.ErrRecordNotFound || deletedDeployment == nil)
	}
}

func (suite *RemoveUniswapDeploymentTestSuite) TestHandlerSuccess_MixedExistingAndNonExisting() {
	// Get test chain
	var testChain models.Chain
	err := suite.db.DB.Where("chain_type = ?", models.TransactionChainTypeEthereum).First(&testChain).Error
	suite.Require().NoError(err)

	// Create one test deployment
	existingDeploymentID := suite.createTestDeployment(testChain.ID, "v2")
	nonExistingID1 := uint(99999)
	nonExistingID2 := uint(88888)

	// Verify existing deployment exists
	deployment, err := suite.uniswapService.GetUniswapDeployment(existingDeploymentID)
	suite.Require().NoError(err)
	suite.NotNil(deployment)

	// Create request with mix of existing and non-existing IDs
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"ids": []interface{}{
					float64(existingDeploymentID),
					float64(nonExistingID1),
					float64(nonExistingID2),
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError)

	// Verify existing deployment was deleted
	deletedDeployment, err := suite.uniswapService.GetUniswapDeployment(existingDeploymentID)
	suite.Error(err)
	suite.True(err == gorm.ErrRecordNotFound || deletedDeployment == nil)
}

func (suite *RemoveUniswapDeploymentTestSuite) TestHandlerError_NoDeploymentsFound() {
	nonExistingID1 := uint(99999)
	nonExistingID2 := uint(88888)

	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"ids": []interface{}{
					float64(nonExistingID1),
					float64(nonExistingID2),
				},
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.NotNil(result)
	suite.False(result.IsError) // Should not be an error, just no deployments found

	// Check that the result indicates no deployments were found
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			suite.Contains(textContent.Text, "No deployments removed")
		}
	}
}

func (suite *RemoveUniswapDeploymentTestSuite) TestHandlerError_InvalidBindArguments() {
	// Test with invalid argument type
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"ids": "invalid_string", // Should be array
			},
		},
	}

	handler := suite.tool.GetHandler()
	_, err := handler(context.Background(), request)

	// Should return binding error
	suite.Error(err)
	suite.Contains(err.Error(), "failed to bind arguments")
}

func (suite *RemoveUniswapDeploymentTestSuite) TestHandlerError_MissingRequiredFields() {
	// Test with missing ids field
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				// Missing "ids" field
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

func (suite *RemoveUniswapDeploymentTestSuite) TestHandlerError_EmptyIdsArray() {
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"ids": []interface{}{}, // Empty array
			},
		},
	}

	handler := suite.tool.GetHandler()
	result, err := handler(context.Background(), request)

	suite.NoError(err)
	suite.True(result.IsError)
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			// The validator catches this first with the min=1 validation
			suite.Contains(textContent.Text, "Invalid arguments")
		}
	}
}

func (suite *RemoveUniswapDeploymentTestSuite) TestDatabaseIntegration() {
	// Get test chain
	var testChain models.Chain
	err := suite.db.DB.Where("chain_type = ?", models.TransactionChainTypeEthereum).First(&testChain).Error
	suite.Require().NoError(err)

	// Create test deployment directly in database
	deployment := &models.UniswapDeployment{
		ChainID: testChain.ID,
		Version: "v2",
		Status:  models.TransactionStatusPending,
	}
	err = suite.db.DB.Create(deployment).Error
	suite.Require().NoError(err)

	deploymentID := deployment.ID

	// Verify it exists
	var foundDeployment models.UniswapDeployment
	err = suite.db.DB.First(&foundDeployment, deploymentID).Error
	suite.Require().NoError(err)
	suite.Equal(deploymentID, foundDeployment.ID)

	// Remove using the tool
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"ids": []interface{}{float64(deploymentID)},
			},
		},
	}

	handler := suite.tool.GetHandler()
	_, err = handler(context.Background(), request)

	suite.NoError(err)

	// Verify it's deleted from database
	err = suite.db.DB.First(&foundDeployment, deploymentID).Error
	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

func TestRemoveUniswapDeploymentTestSuite(t *testing.T) {
	suite.Run(t, new(RemoveUniswapDeploymentTestSuite))
}
