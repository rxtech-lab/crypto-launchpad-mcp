package api

import (
	"fmt"
	"testing"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/suite"
)

// TokenDeploymentTestSuite tests the complete token deployment workflow
type TokenDeploymentTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *TokenDeploymentTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())

	// Verify Ethereum connection is available
	err := s.setup.TestSetup.VerifyEthereumConnection()
	s.Require().NoError(err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")
}

func (s *TokenDeploymentTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *TokenDeploymentTestSuite) TestOpenZeppelinTokenDeployment() {
	// Create OpenZeppelin template first
	templateID, err := s.setup.CreateOpenZeppelinTemplate()
	s.Require().NoError(err, "Failed to create OpenZeppelin template")

	// Create deployment session using the template
	sessionID, err := s.setup.CreateTokenDeploymentSession(templateID)
	s.Require().NoError(err, "Failed to create deployment session")

	// Create page object
	page := NewTokenDeploymentPage(s.setup.ctx)

	// Navigate to the deployment page
	baseURL := s.setup.GetBaseURL()
	err = page.NavigateToSession(baseURL, sessionID)
	s.Require().NoError(err, "Failed to navigate to deployment page")

	// Initialize test wallet immediately after navigation, before waiting for page load
	err = s.setup.InitializeTestWallet()
	s.Require().NoError(err, "Failed to initialize test wallet")

	// Wait for page to load (now that wallet provider is injected)
	err = page.WaitForPageLoad()
	s.Require().NoError(err, "Failed to wait for page load")

	// Verify page title
	title, err := page.GetPageTitle()
	s.Require().NoError(err)
	s.Assert().Contains(title, "Transaction Signing", "Page title should contain 'Transaction Signing'")

	// Test wallet connection workflow
	s.testWalletConnection(page)

	// Test deployment workflow
	s.testDeploymentWorkflow(page)

	// Test database state verification
	s.testDatabaseVerification(sessionID)

	// Log final page state for debugging
	err = page.LogPageState("token_deployment_openzeppelin_final")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

func (s *TokenDeploymentTestSuite) TestCustomTokenDeployment() {
	// Create custom template first
	templateID, err := s.setup.CreateCustomTemplate()
	s.Require().NoError(err, "Failed to create custom template")

	// Create deployment session using the template
	sessionID, err := s.setup.CreateTokenDeploymentSession(templateID)
	s.Require().NoError(err, "Failed to create deployment session")

	// Create page object
	page := NewTokenDeploymentPage(s.setup.ctx)

	// Navigate to the deployment page
	baseURL := s.setup.GetBaseURL()
	err = page.NavigateToSession(baseURL, sessionID)
	s.Require().NoError(err, "Failed to navigate to deployment page")

	// Initialize test wallet immediately after navigation, before waiting for page load
	err = s.setup.InitializeTestWallet()
	s.Require().NoError(err, "Failed to initialize test wallet")

	// Wait for page to load (now that wallet provider is injected)
	err = page.WaitForPageLoad()
	s.Require().NoError(err, "Failed to wait for page load")

	// Verify page title
	title, err := page.GetPageTitle()
	s.Require().NoError(err)
	s.Assert().Contains(title, "Transaction Signing", "Page title should contain 'Transaction Signing'")

	// Test wallet connection workflow
	s.testWalletConnection(page)

	// Test deployment workflow
	s.testDeploymentWorkflow(page)

	// Test database state verification
	s.testDatabaseVerification(sessionID)

	// Log final page state for debugging
	err = page.LogPageState("token_deployment_custom_final")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

func (s *TokenDeploymentTestSuite) testWalletConnection(page *TokenDeploymentPage) {
	// Wait for wallet selection to be available
	err := page.WaitForWalletSelection()
	s.Assert().NoError(err, "Wallet selection not available")

	// Select the test wallet by value (UUID)
	err = page.SelectWallet("test-wallet-e2e")
	s.Assert().NoError(err, "Failed to select test wallet")
}

func (s *TokenDeploymentTestSuite) testDeploymentWorkflow(page *TokenDeploymentPage) {
	// Click the deploy button
	err := page.ClickDeployButton()
	s.Require().NoError(err, "Failed to click deploy button")

	// Wait for success state
	err = page.WaitForSuccessState()
	s.Require().NoError(err, "Success state not reached")
}

func (s *TokenDeploymentTestSuite) testDatabaseVerification(sessionID string) {
	// sleep for a short time to ensure DB is updated
	time.Sleep(2 * time.Second)
	// Get session from database
	session, err := s.setup.TestSetup.TxService.GetTransactionSession(sessionID)
	s.Require().NoError(err, "Failed to get transaction session")

	// Verify session status
	s.Assert().Equal(models.TransactionStatusConfirmed, session.TransactionStatus, "Session status should be models.TransactionStatusConfirmed")

	s.Require().NoError(err, "Failed to list deployments")
	for _, deployment := range session.TransactionDeployments {
		s.Require().Equal(deployment.Status, models.TransactionStatusConfirmed, "Deployment status should be models.TransactionStatusConfirmed")
	}
}

// TokenDeploymentErrorTestSuite tests error scenarios for token deployment
type TokenDeploymentErrorTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *TokenDeploymentErrorTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())
}

func (s *TokenDeploymentErrorTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *TokenDeploymentErrorTestSuite) TestInvalidSession() {
	page := NewTokenDeploymentPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	// Navigate to invalid session ID
	err := page.NavigateToSession(baseURL, "invalid-session-id")
	s.Require().NoError(err)

	// Should get 404 or error page
	time.Sleep(2 * time.Second) // Allow page to load

	// Take screenshot for debugging
	page.TakeScreenshot("token_deployment_invalid_session.png")

	// Log page HTML for debugging
	err = page.LogPageState("token_deployment_invalid_session")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

func (s *TokenDeploymentErrorTestSuite) TestExpiredSession() {
	// Create template first
	templateID, err := s.setup.CreateCustomTemplate()
	s.Require().NoError(err)

	// Create session and immediately expire it
	sessionID, err := s.setup.CreateTokenDeploymentSession(templateID)
	s.Require().NoError(err)

	// Manually expire the session
	err = s.setup.TestSetup.TxService.UpdateTransactionSessionStatus(sessionID, "expired", "")
	s.Require().NoError(err)

	page := NewTokenDeploymentPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	// Navigate to expired session
	err = page.NavigateToSession(baseURL, sessionID)
	s.Require().NoError(err)

	// Should handle expired session gracefully
	time.Sleep(2 * time.Second)

	// Take screenshot for debugging
	page.TakeScreenshot("token_deployment_expired_session.png")

	// Log page HTML for debugging
	err = page.LogPageState("token_deployment_expired_session")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

// TokenDeploymentWalletTestSuite tests behavior when no wallet is available
type TokenDeploymentWalletTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *TokenDeploymentWalletTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())

	// Verify Ethereum connection
	err := s.setup.TestSetup.VerifyEthereumConnection()
	s.Require().NoError(err)
}

func (s *TokenDeploymentWalletTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *TokenDeploymentWalletTestSuite) TestDeploymentWithoutWallet() {
	// Create template first
	templateID, err := s.setup.CreateCustomTemplate()
	s.Require().NoError(err)

	// Create session but don't initialize wallet
	sessionID, err := s.setup.CreateTokenDeploymentSession(templateID)
	s.Require().NoError(err)

	page := NewTokenDeploymentPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	// Navigate to deployment page
	err = page.NavigateToSession(baseURL, sessionID)
	s.Require().NoError(err)

	// Wait for page to load
	err = page.WaitForPageLoad()
	s.Require().NoError(err)

	// Should show empty wallet selection or appropriate message
	time.Sleep(3 * time.Second) // Allow time for wallet discovery

	// Take screenshot to verify UI state
	page.TakeScreenshot("token_deployment_no_wallet.png")

	// Check if no wallets message is shown
	err = page.WaitForNoWalletsMessage()
	if err != nil {
		// If no specific no-wallets message, check connection status
		status, statusErr := page.CheckConnectionStatus()
		if statusErr != nil {
			s.T().Logf("Could not check connection status: %v", statusErr)
		} else {
			s.T().Logf("Connection status: %s", status)
			s.Assert().Equal("not_connected", status, "Should show not connected status when no wallet available")
		}
	}

	// Log page HTML for debugging
	err = page.LogPageState("token_deployment_no_wallet")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

// TokenDeploymentPageLoadTestSuite tests just the page loading functionality
type TokenDeploymentPageLoadTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *TokenDeploymentPageLoadTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())
}

func (s *TokenDeploymentPageLoadTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

// Test runner functions that testify expects
func TestTokenDeployment(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	suite.Run(t, new(TokenDeploymentTestSuite))
}

func TestTokenDeploymentErrorHandling(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	suite.Run(t, new(TokenDeploymentErrorTestSuite))
}

func TestTokenDeploymentWithoutWallet(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	suite.Run(t, new(TokenDeploymentWalletTestSuite))
}

func TestTokenDeploymentPageLoad(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	suite.Run(t, new(TokenDeploymentPageLoadTestSuite))
}
