package hooks

import (
	"testing"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

type TokenDeploymentHookTestSuite struct {
	suite.Suite
	dbService         services.DBService
	deploymentService services.DeploymentService
	templateService   services.TemplateService
	chainService      services.ChainService
	hook              *TokenDeploymentHook
}

func (s *TokenDeploymentHookTestSuite) SetupSuite() {
	// Use in-memory database for testing
	db, err := services.NewSqliteDBService(":memory:")
	s.Require().NoError(err)
	s.dbService = db

	// Initialize services
	s.deploymentService = services.NewDeploymentService(db.GetDB())
	s.templateService = services.NewTemplateService(db.GetDB())
	s.chainService = services.NewChainService(db.GetDB())

	// Create hook instance
	s.hook = &TokenDeploymentHook{
		deploymentService: s.deploymentService,
	}

	// Setup test data
	s.setupTestData()
}

func (s *TokenDeploymentHookTestSuite) TearDownSuite() {
	if s.dbService != nil {
		s.dbService.Close()
	}
}

func (s *TokenDeploymentHookTestSuite) SetupTest() {
	// Clean up test data between tests
	s.dbService.GetDB().Where("1 = 1").Delete(&models.Deployment{})
}

func (s *TokenDeploymentHookTestSuite) setupTestData() {
	// Create test chain
	chain := &models.Chain{
		ChainType: "ethereum",
		RPC:       "http://localhost:8545",
		NetworkID: "31337",
		Name:      "Test Chain",
		IsActive:  true,
	}
	err := s.chainService.CreateChain(chain)
	s.Require().NoError(err)

	// Create test template
	template := &models.Template{
		Name:         "Test ERC20",
		Description:  "Test ERC20 token template",
		ChainType:    "ethereum",
		ContractName: "TestToken",
		TemplateCode: `pragma solidity ^0.8.0; contract TestToken { constructor(string memory name) {} }`,
		Metadata:     models.JSON{"name": ""},
	}
	err = s.templateService.CreateTemplate(template)
	s.Require().NoError(err)
}

func (s *TokenDeploymentHookTestSuite) TestCanHandle() {
	// Test supported transaction types
	s.True(s.hook.CanHandle(models.TransactionTypeTokenDeployment))
	s.True(s.hook.CanHandle(models.TransactionTypeUniswapV2TokenDeployment))

	// Test unsupported transaction types
	s.False(s.hook.CanHandle(models.TransactionType("other_type")))
	s.False(s.hook.CanHandle(models.TransactionType("")))
}

func (s *TokenDeploymentHookTestSuite) TestOnTransactionConfirmed_TokenDeployment() {
	// Create test deployment record
	deployment := &models.Deployment{
		TemplateID:      1,
		ChainID:         1,
		TransactionHash: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Status:          models.TransactionStatusPending,
		CreatedAt:       time.Now(),
	}

	err := s.deploymentService.CreateDeployment(deployment)
	s.Require().NoError(err)

	// Create test session
	session := models.TransactionSession{
		ID:                "test-session-id",
		TransactionStatus: models.TransactionStatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(30 * time.Minute),
	}

	// Test successful transaction confirmation
	contractAddress := "0xabcdef1234567890abcdef1234567890abcdef12"
	err = s.hook.OnTransactionConfirmed(
		models.TransactionTypeTokenDeployment,
		deployment.TransactionHash,
		contractAddress,
		session,
	)
	s.NoError(err)

	// Verify deployment record was updated
	updatedDeployment, err := s.deploymentService.GetDeploymentByTransactionHash(deployment.TransactionHash)
	s.Require().NoError(err)

	s.Equal(contractAddress, updatedDeployment.ContractAddress)
	s.Equal(string(models.TransactionStatusConfirmed), updatedDeployment.Status)
}

func (s *TokenDeploymentHookTestSuite) TestOnTransactionConfirmed_UniswapV2TokenDeployment() {
	// Create test deployment record
	deployment := &models.Deployment{
		TemplateID:      1,
		ChainID:         1,
		TransactionHash: "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		Status:          models.TransactionStatusPending,
		CreatedAt:       time.Now(),
	}

	err := s.deploymentService.CreateDeployment(deployment)
	s.Require().NoError(err)

	// Create test session
	session := models.TransactionSession{
		ID:                "test-uniswap-session-id",
		TransactionStatus: models.TransactionStatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(30 * time.Minute),
	}

	// Test successful transaction confirmation for Uniswap V2 deployment
	contractAddress := "0x9876543210fedcba9876543210fedcba98765432"
	err = s.hook.OnTransactionConfirmed(
		models.TransactionTypeUniswapV2TokenDeployment,
		deployment.TransactionHash,
		contractAddress,
		session,
	)
	s.NoError(err)

	// Verify deployment record was updated
	updatedDeployment, err := s.deploymentService.GetDeploymentByTransactionHash(deployment.TransactionHash)
	s.Require().NoError(err)

	s.Equal(contractAddress, updatedDeployment.ContractAddress)
	s.Equal(string(models.TransactionStatusConfirmed), updatedDeployment.Status)
}

func (s *TokenDeploymentHookTestSuite) TestOnTransactionConfirmed_NonexistentTransaction() {
	// Create test session
	session := models.TransactionSession{
		ID:                "test-session-id",
		TransactionStatus: models.TransactionStatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(30 * time.Minute),
	}

	// Test with nonexistent transaction hash
	nonexistentTxHash := "0x0000000000000000000000000000000000000000000000000000000000000000"
	contractAddress := "0xabcdef1234567890abcdef1234567890abcdef12"

	err := s.hook.OnTransactionConfirmed(
		models.TransactionTypeTokenDeployment,
		nonexistentTxHash,
		contractAddress,
		session,
	)

	// Should not return error even if no records are updated
	s.NoError(err)
}

func (s *TokenDeploymentHookTestSuite) TestOnTransactionConfirmed_EmptyContractAddress() {
	// Create test deployment record
	deployment := &models.Deployment{
		TemplateID:      1,
		ChainID:         1,
		TransactionHash: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		Status:          models.TransactionStatusPending,
		CreatedAt:       time.Now(),
	}

	err := s.deploymentService.CreateDeployment(deployment)
	s.Require().NoError(err)

	// Create test session
	session := models.TransactionSession{
		ID:                "test-session-id",
		TransactionStatus: models.TransactionStatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(30 * time.Minute),
	}

	// Test with empty contract address
	err = s.hook.OnTransactionConfirmed(
		models.TransactionTypeTokenDeployment,
		deployment.TransactionHash,
		"", // Empty contract address
		session,
	)
	s.NoError(err)

	// Verify deployment record was updated with empty address
	updatedDeployment, err := s.deploymentService.GetDeploymentByTransactionHash(deployment.TransactionHash)
	s.Require().NoError(err)

	s.Equal("", updatedDeployment.ContractAddress)
	s.Equal(string(models.TransactionStatusConfirmed), updatedDeployment.Status)
}

func (s *TokenDeploymentHookTestSuite) TestOnTransactionConfirmed_UpdateMultipleRecords() {
	// Create multiple deployment records with same transaction hash
	txHash := "0x1111111111111111111111111111111111111111111111111111111111111111"

	deployment1 := &models.Deployment{
		TemplateID:      1,
		ChainID:         1,
		TransactionHash: txHash,
		Status:          models.TransactionStatusPending,
		CreatedAt:       time.Now(),
	}
	deployment2 := &models.Deployment{
		TemplateID:      1,
		ChainID:         1,
		TransactionHash: txHash,
		Status:          models.TransactionStatusPending,
		CreatedAt:       time.Now(),
	}

	err := s.deploymentService.CreateDeployment(deployment1)
	s.Require().NoError(err)
	err = s.deploymentService.CreateDeployment(deployment2)
	s.Require().NoError(err)

	// Create test session
	session := models.TransactionSession{
		ID:                "test-session-id",
		TransactionStatus: models.TransactionStatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(30 * time.Minute),
	}

	// Test successful transaction confirmation
	contractAddress := "0xabcdef1234567890abcdef1234567890abcdef12"
	err = s.hook.OnTransactionConfirmed(
		models.TransactionTypeTokenDeployment,
		txHash,
		contractAddress,
		session,
	)
	s.NoError(err)

	// Verify both deployment records were updated
	var updatedDeployments []models.Deployment
	err = s.dbService.GetDB().Where("transaction_hash = ?", txHash).Find(&updatedDeployments).Error
	s.Require().NoError(err)
	s.Len(updatedDeployments, 2)

	for _, deployment := range updatedDeployments {
		s.Equal(contractAddress, deployment.ContractAddress)
		s.Equal(string(models.TransactionStatusConfirmed), deployment.Status)
	}
}

func (s *TokenDeploymentHookTestSuite) TestNewTokenDeploymentHook() {
	// Test constructor function
	hook := NewTokenDeploymentHook(s.deploymentService)
	s.NotNil(hook)

	// Verify it implements the Hook interface
	s.True(hook.CanHandle(models.TransactionTypeTokenDeployment))
	s.True(hook.CanHandle(models.TransactionTypeUniswapV2TokenDeployment))
	s.False(hook.CanHandle(models.TransactionType("other")))

	// Test OnTransactionConfirmed works
	deployment := &models.Deployment{
		TemplateID:      1,
		ChainID:         1,
		TransactionHash: "0x2222222222222222222222222222222222222222222222222222222222222222",
		Status:          models.TransactionStatusPending,
		CreatedAt:       time.Now(),
	}

	err := s.deploymentService.CreateDeployment(deployment)
	s.Require().NoError(err)

	session := models.TransactionSession{
		ID:                "constructor-test-session",
		TransactionStatus: models.TransactionStatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(30 * time.Minute),
	}

	contractAddress := "0x1111111111111111111111111111111111111111"
	err = hook.OnTransactionConfirmed(
		models.TransactionTypeTokenDeployment,
		deployment.TransactionHash,
		contractAddress,
		session,
	)
	s.NoError(err)

	// Verify deployment was updated
	updatedDeployment, err := s.deploymentService.GetDeploymentByTransactionHash(deployment.TransactionHash)
	s.Require().NoError(err)
	s.Equal(contractAddress, updatedDeployment.ContractAddress)
	s.Equal(string(models.TransactionStatusConfirmed), updatedDeployment.Status)
}

func (s *TokenDeploymentHookTestSuite) TestDatabaseTransactionIntegrity() {
	// Test that database operations maintain consistency
	deployment := &models.Deployment{
		TemplateID:      1,
		ChainID:         1,
		TransactionHash: "0x3333333333333333333333333333333333333333333333333333333333333333",
		Status:          models.TransactionStatusPending,
		CreatedAt:       time.Now(),
	}

	err := s.deploymentService.CreateDeployment(deployment)
	s.Require().NoError(err)

	session := models.TransactionSession{
		ID:                "integrity-test-session",
		TransactionStatus: models.TransactionStatusPending,
		CreatedAt:         time.Now(),
		ExpiresAt:         time.Now().Add(30 * time.Minute),
	}

	// Simulate concurrent updates by calling multiple times
	contractAddress := "0x4444444444444444444444444444444444444444"
	for range 3 {
		err = s.hook.OnTransactionConfirmed(
			models.TransactionTypeTokenDeployment,
			deployment.TransactionHash,
			contractAddress,
			session,
		)
		s.NoError(err)
	}

	// Verify final state is consistent
	updatedDeployment, err := s.deploymentService.GetDeploymentByTransactionHash(deployment.TransactionHash)
	s.Require().NoError(err)
	s.Equal(contractAddress, updatedDeployment.ContractAddress)
	s.Equal(string(models.TransactionStatusConfirmed), updatedDeployment.Status)
}

func TestTokenDeploymentHook(t *testing.T) {
	suite.Run(t, new(TokenDeploymentHookTestSuite))
}
