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
	dbService      services.DBService
	uniswapService services.UniswapService
	chainService   services.ChainService
	tool           *removeUniswapDeploymentTool
}

func (suite *RemoveUniswapDeploymentTestSuite) SetupSuite() {
	// Initialize in-memory database for testing
	dbService, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.dbService = dbService

	// Initialize services
	suite.uniswapService = services.NewUniswapService(dbService.GetDB())
	suite.chainService = services.NewChainService(dbService.GetDB())

	// Initialize tool
	suite.tool = NewRemoveUniswapDeploymentTool(suite.uniswapService)

	// Setup test data
	suite.setupTestData()
}

func (suite *RemoveUniswapDeploymentTestSuite) TearDownSuite() {
	if suite.dbService != nil {
		suite.dbService.Close()
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
	err := suite.chainService.CreateChain(testChain)
	suite.Require().NoError(err)
}

func (suite *RemoveUniswapDeploymentTestSuite) cleanupTestData() {
	// Clean up deployments and sessions
	suite.dbService.GetDB().Where("1 = 1").Delete(&models.UniswapDeployment{})
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
	testChain, err := suite.chainService.GetChainByType(string(models.TransactionChainTypeEthereum))
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
	testChain, err := suite.chainService.GetChainByType(string(models.TransactionChainTypeEthereum))
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
	testChain, err := suite.chainService.GetChainByType(string(models.TransactionChainTypeEthereum))
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
	testChain, err := suite.chainService.GetChainByType(string(models.TransactionChainTypeEthereum))
	suite.Require().NoError(err)

	// Create test deployment using service
	deploymentID, err := suite.uniswapService.CreateUniswapDeployment(testChain.ID, "v2")
	suite.Require().NoError(err)

	// Verify it exists
	foundDeployment, err := suite.uniswapService.GetUniswapDeployment(deploymentID)
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
	_, err = suite.uniswapService.GetUniswapDeployment(deploymentID)
	suite.Error(err)
	suite.Equal(gorm.ErrRecordNotFound, err)
}

func TestRemoveUniswapDeploymentTestSuite(t *testing.T) {
	suite.Run(t, new(RemoveUniswapDeploymentTestSuite))
}
