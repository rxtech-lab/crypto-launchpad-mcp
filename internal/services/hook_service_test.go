package services_test

import (
	"fmt"
	"testing"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

// mockHook implements the Hook interface for testing
type mockHook struct {
	name             string
	supportedTypes   []models.TransactionType
	callCount        int
	lastTxType       models.TransactionType
	lastTxHash       string
	lastContractAddr *string
	lastSession      *models.TransactionSession
	shouldError      bool
	errorMessage     string
}

func newMockHook(name string, supportedTypes ...models.TransactionType) *mockHook {
	return &mockHook{
		name:           name,
		supportedTypes: supportedTypes,
		callCount:      0,
		shouldError:    false,
	}
}

func (m *mockHook) CanHandle(txType models.TransactionType) bool {
	for _, supportedType := range m.supportedTypes {
		if supportedType == txType {
			return true
		}
	}
	return false
}

func (m *mockHook) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress *string, session models.TransactionSession) error {
	m.callCount++
	m.lastTxType = txType
	m.lastTxHash = txHash
	m.lastContractAddr = contractAddress
	m.lastSession = &session

	if m.shouldError {
		return fmt.Errorf("%s", m.errorMessage)
	}
	return nil
}

func (m *mockHook) reset() {
	m.callCount = 0
	m.lastTxType = ""
	m.lastTxHash = ""
	m.lastContractAddr = nil
	m.lastSession = nil
	m.shouldError = false
	m.errorMessage = ""
}

func (m *mockHook) setError(shouldError bool, message string) {
	m.shouldError = shouldError
	m.errorMessage = message
}

type HookServiceTestSuite struct {
	suite.Suite
	hookService services.HookService
}

func (suite *HookServiceTestSuite) SetupSuite() {
	suite.hookService = services.NewHookService()
}

func (suite *HookServiceTestSuite) SetupTest() {
	// Create a fresh service for each test to avoid state leakage
	suite.hookService = services.NewHookService()
}

func (suite *HookServiceTestSuite) TestAddHook() {
	suite.Run("Add single hook", func() {
		hook := newMockHook("test-hook", models.TransactionTypeUniswapV2FactoryDeployment)

		err := suite.hookService.AddHook(hook)
		suite.NoError(err)

		// Test that the hook was added by triggering an event
		session := models.TransactionSession{
			ID:                   "test-session",
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			ChainID:              1,
		}
		err = suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0x123",
			nil,
			session,
		)
		suite.NoError(err)
		suite.Equal(1, hook.callCount)
	})

	suite.Run("Add multiple hooks", func() {
		hook1 := newMockHook("hook1", models.TransactionTypeUniswapV2FactoryDeployment)
		hook2 := newMockHook("hook2", models.TransactionTypeUniswapV2RouterDeployment)
		hook3 := newMockHook("hook3", models.TransactionTypeUniswapV2FactoryDeployment, models.TransactionTypeUniswapV2RouterDeployment)

		err := suite.hookService.AddHook(hook1)
		suite.NoError(err)
		err = suite.hookService.AddHook(hook2)
		suite.NoError(err)
		err = suite.hookService.AddHook(hook3)
		suite.NoError(err)

		// Trigger factory deployment - should call hook1 and hook3
		session := models.TransactionSession{
			ID:                   "test-session",
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			ChainID:              1,
		}
		err = suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0x123",
			nil,
			session,
		)
		suite.NoError(err)

		suite.Equal(1, hook1.callCount)
		suite.Equal(0, hook2.callCount) // Should not be called
		suite.Equal(1, hook3.callCount)
	})
}

func (suite *HookServiceTestSuite) TestOnTransactionConfirmed() {
	hook1 := newMockHook("hook1", models.TransactionTypeUniswapV2FactoryDeployment)
	hook2 := newMockHook("hook2", models.TransactionTypeUniswapV2RouterDeployment)
	hook3 := newMockHook("hook3", models.TransactionTypeUniswapV2FactoryDeployment, models.TransactionTypeUniswapV2RouterDeployment)

	err := suite.hookService.AddHook(hook1)
	suite.NoError(err)
	err = suite.hookService.AddHook(hook2)
	suite.NoError(err)
	err = suite.hookService.AddHook(hook3)
	suite.NoError(err)

	contractAddr := "0xContractAddress123"
	session := models.TransactionSession{
		ID:                   "test-session-123",
		TransactionStatus:    models.TransactionStatusConfirmed,
		TransactionChainType: models.TransactionChainTypeEthereum,
		ChainID:              1,
	}

	suite.Run("Factory deployment event", func() {
		// Reset all hooks
		hook1.reset()
		hook2.reset()
		hook3.reset()

		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0xFactoryTxHash",
			&contractAddr,
			session,
		)
		suite.NoError(err)

		// Verify correct hooks were called
		suite.Equal(1, hook1.callCount)
		suite.Equal(0, hook2.callCount)
		suite.Equal(1, hook3.callCount)

		// Verify parameters were passed correctly to hook1
		suite.Equal(models.TransactionTypeUniswapV2FactoryDeployment, hook1.lastTxType)
		suite.Equal("0xFactoryTxHash", hook1.lastTxHash)
		suite.Require().NotNil(hook1.lastContractAddr)
		suite.Equal(contractAddr, *hook1.lastContractAddr)
		suite.Equal("test-session-123", hook1.lastSession.ID)

		// Verify parameters were passed correctly to hook3
		suite.Equal(models.TransactionTypeUniswapV2FactoryDeployment, hook3.lastTxType)
		suite.Equal("0xFactoryTxHash", hook3.lastTxHash)
		suite.Require().NotNil(hook3.lastContractAddr)
		suite.Equal(contractAddr, *hook3.lastContractAddr)
		suite.Equal("test-session-123", hook3.lastSession.ID)
	})

	suite.Run("Router deployment event", func() {
		// Reset all hooks
		hook1.reset()
		hook2.reset()
		hook3.reset()

		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2RouterDeployment,
			"0xRouterTxHash",
			&contractAddr,
			session,
		)
		suite.NoError(err)

		// Verify correct hooks were called
		suite.Equal(0, hook1.callCount)
		suite.Equal(1, hook2.callCount)
		suite.Equal(1, hook3.callCount)

		// Verify parameters were passed correctly to hook2
		suite.Equal(models.TransactionTypeUniswapV2RouterDeployment, hook2.lastTxType)
		suite.Equal("0xRouterTxHash", hook2.lastTxHash)
		suite.Require().NotNil(hook2.lastContractAddr)
		suite.Equal(contractAddr, *hook2.lastContractAddr)
		suite.Equal("test-session-123", hook2.lastSession.ID)
	})

	suite.Run("Unsupported transaction type", func() {
		// Reset all hooks
		hook1.reset()
		hook2.reset()
		hook3.reset()

		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2TokenDeployment,
			"0xTokenTxHash",
			&contractAddr,
			session,
		)
		suite.NoError(err)

		// No hooks should be called
		suite.Equal(0, hook1.callCount)
		suite.Equal(0, hook2.callCount)
		suite.Equal(0, hook3.callCount)
	})
}

func (suite *HookServiceTestSuite) TestHookFiltering() {
	hook1 := newMockHook("hook1", models.TransactionTypeUniswapV2FactoryDeployment)
	hook2 := newMockHook("hook2", models.TransactionTypeUniswapV2RouterDeployment, models.TransactionTypeUniswapV2TokenDeployment)
	hook3 := newMockHook("hook3") // Supports no transaction types

	err := suite.hookService.AddHook(hook1)
	suite.NoError(err)
	err = suite.hookService.AddHook(hook2)
	suite.NoError(err)
	err = suite.hookService.AddHook(hook3)
	suite.NoError(err)

	session := models.TransactionSession{
		ID:                   "test-session",
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
		ChainID:              1,
	}

	suite.Run("Factory deployment filtering", func() {
		hook1.reset()
		hook2.reset()
		hook3.reset()

		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0x123",
			nil,
			session,
		)
		suite.NoError(err)

		suite.Equal(1, hook1.callCount) // Supports factory
		suite.Equal(0, hook2.callCount) // Doesn't support factory
		suite.Equal(0, hook3.callCount) // Supports nothing
	})

	suite.Run("Token deployment filtering", func() {
		hook1.reset()
		hook2.reset()
		hook3.reset()

		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2TokenDeployment,
			"0x456",
			nil,
			session,
		)
		suite.NoError(err)

		suite.Equal(0, hook1.callCount) // Doesn't support token
		suite.Equal(1, hook2.callCount) // Supports token
		suite.Equal(0, hook3.callCount) // Supports nothing
	})
}

func (suite *HookServiceTestSuite) TestHookErrors() {
	hook1 := newMockHook("hook1", models.TransactionTypeUniswapV2FactoryDeployment)
	hook2 := newMockHook("hook2", models.TransactionTypeUniswapV2FactoryDeployment)
	hook3 := newMockHook("hook3", models.TransactionTypeUniswapV2FactoryDeployment)

	// Set hook2 to return an error
	hook2.setError(true, "hook2 processing failed")

	err := suite.hookService.AddHook(hook1)
	suite.NoError(err)
	err = suite.hookService.AddHook(hook2)
	suite.NoError(err)
	err = suite.hookService.AddHook(hook3)
	suite.NoError(err)

	session := models.TransactionSession{
		ID:                   "test-session",
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
		ChainID:              1,
	}

	suite.Run("Hook error stops processing", func() {
		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0x123",
			nil,
			session,
		)
		suite.Error(err)
		suite.Contains(err.Error(), "hook2 processing failed")

		// First hook should have been called
		suite.Equal(1, hook1.callCount)
		// Second hook should have been called and failed
		suite.Equal(1, hook2.callCount)
		// Third hook should not have been called due to error in hook2
		suite.Equal(0, hook3.callCount)
	})
}

func (suite *HookServiceTestSuite) TestMultipleTransactionTypes() {
	// Create a hook that supports multiple transaction types
	hook := newMockHook("multi-hook",
		models.TransactionTypeUniswapV2FactoryDeployment,
		models.TransactionTypeUniswapV2RouterDeployment,
		models.TransactionTypeUniswapV2TokenDeployment,
	)

	err := suite.hookService.AddHook(hook)
	suite.NoError(err)

	session := models.TransactionSession{
		ID:                   "test-session",
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
		ChainID:              1,
	}

	// Test each supported transaction type
	testCases := []struct {
		name   string
		txType models.TransactionType
		txHash string
	}{
		{"Factory", models.TransactionTypeUniswapV2FactoryDeployment, "0xFactory"},
		{"Router", models.TransactionTypeUniswapV2RouterDeployment, "0xRouter"},
		{"Token", models.TransactionTypeUniswapV2TokenDeployment, "0xToken"},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			hook.reset()

			err := suite.hookService.OnTransactionConfirmed(tc.txType, tc.txHash, nil, session)
			suite.NoError(err)

			suite.Equal(1, hook.callCount)
			suite.Equal(tc.txType, hook.lastTxType)
			suite.Equal(tc.txHash, hook.lastTxHash)
		})
	}
}

func (suite *HookServiceTestSuite) TestEmptyHookService() {
	// Test behavior with no hooks registered
	session := models.TransactionSession{
		ID:                   "test-session",
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
		ChainID:              1,
	}

	suite.Run("No hooks registered", func() {
		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0x123",
			nil,
			session,
		)
		suite.NoError(err) // Should not error when no hooks are registered
	})
}

func (suite *HookServiceTestSuite) TestNilContractAddress() {
	hook := newMockHook("hook", models.TransactionTypeUniswapV2FactoryDeployment)
	err := suite.hookService.AddHook(hook)
	suite.NoError(err)

	session := models.TransactionSession{
		ID:                   "test-session",
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: models.TransactionChainTypeEthereum,
		ChainID:              1,
	}

	suite.Run("Nil contract address", func() {
		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0x123",
			nil, // Nil contract address
			session,
		)
		suite.NoError(err)

		suite.Equal(1, hook.callCount)
		suite.Nil(hook.lastContractAddr)
	})
}

func (suite *HookServiceTestSuite) TestComplexSession() {
	hook := newMockHook("hook", models.TransactionTypeUniswapV2FactoryDeployment)
	err := suite.hookService.AddHook(hook)
	suite.NoError(err)

	// Create a complex session with multiple fields
	session := models.TransactionSession{
		ID:                   "complex-session-123",
		TransactionStatus:    models.TransactionStatusConfirmed,
		TransactionChainType: models.TransactionChainTypeEthereum,
		ChainID:              31337,
	}

	contractAddr := "0xComplexContract"

	suite.Run("Complex session data", func() {
		err := suite.hookService.OnTransactionConfirmed(
			models.TransactionTypeUniswapV2FactoryDeployment,
			"0xComplexTxHash",
			&contractAddr,
			session,
		)
		suite.NoError(err)

		suite.Equal(1, hook.callCount)
		suite.Equal(models.TransactionTypeUniswapV2FactoryDeployment, hook.lastTxType)
		suite.Equal("0xComplexTxHash", hook.lastTxHash)
		suite.Require().NotNil(hook.lastContractAddr)
		suite.Equal(contractAddr, *hook.lastContractAddr)

		// Verify session data was passed correctly
		suite.Require().NotNil(hook.lastSession)
		suite.Equal("complex-session-123", hook.lastSession.ID)
		suite.Equal(models.TransactionChainTypeEthereum, hook.lastSession.TransactionChainType)
		suite.Equal(uint(31337), hook.lastSession.ChainID)
		suite.Equal(models.TransactionStatusConfirmed, hook.lastSession.TransactionStatus)
	})
}

func TestHookServiceTestSuite(t *testing.T) {
	suite.Run(t, new(HookServiceTestSuite))
}
