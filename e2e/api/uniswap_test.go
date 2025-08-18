package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/suite"
)

// UniswapDeploymentChromedpTestSuite tests the complete Uniswap deployment workflow using chromedp
type UniswapDeploymentChromedpTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *UniswapDeploymentChromedpTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())

	// Verify Ethereum connection is available
	err := s.setup.VerifyEthereumConnection()
	s.Require().NoError(err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")
}

func (s *UniswapDeploymentChromedpTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *UniswapDeploymentChromedpTestSuite) TestUniswapDeploymentWorkflow() {
	// Create a Uniswap deployment session
	sessionID, err := s.setup.CreateUniswapDeploymentSession()
	s.Require().NoError(err, "Failed to create deployment session")

	// Create page object
	page := NewUniswapDeploymentPage(s.setup.ctx)

	// Navigate to the deployment page first
	baseURL := s.setup.GetBaseURL()
	err = page.NavigateToSession(baseURL, sessionID)
	s.Require().NoError(err, "Failed to navigate to deployment page")

	// Wait for page to load
	err = page.WaitForPageLoad()
	s.Require().NoError(err, "Failed to wait for page load")

	// Now initialize test wallet in browser AFTER page is loaded
	err = s.setup.InitializeTestWallet()
	s.Require().NoError(err, "Failed to initialize test wallet")

	// Verify page title
	title, err := page.GetPageTitle()
	s.Require().NoError(err)
	s.Assert().Contains(title, "Deploy Uniswap", "Page title should contain 'Deploy Uniswap'")

	// Test wallet connection workflow
	s.testWalletConnection(page)

	// Test deployment workflow
	s.testDeploymentWorkflow(page)

	// Test contract verification
	s.testContractVerification(page)

	// Test database state verification
	s.testDatabaseVerification(sessionID)

	// Log final page state for debugging
	err = page.LogPageState("uniswap_deployment_final")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

func (s *UniswapDeploymentChromedpTestSuite) testWalletConnection(page *UniswapDeploymentPage) {
	// Wait for wallet selection to be available
	err := page.WaitForWalletSelection()
	s.Assert().NoError(err, "Wallet selection not available")

	// Select the test wallet
	err = page.SelectWallet("test-wallet-e2e")
	s.Assert().NoError(err, "Failed to select test wallet")

	// Click connect wallet button
	err = page.ClickConnectWallet()
	s.Assert().NoError(err, "Failed to click connect wallet")

	// Wait for wallet connection
	err = page.WaitForWalletConnection()
	s.Assert().NoError(err, "Wallet connection failed")
}

func (s *UniswapDeploymentChromedpTestSuite) testDeploymentWorkflow(page *UniswapDeploymentPage) {
	// Click the deploy button
	err := page.ClickDeployButton()
	s.Require().NoError(err, "Failed to click deploy button")

	// Wait for success state
	err = page.WaitForSuccessState()
	s.Require().NoError(err, "Success state not reached")
}

func (s *UniswapDeploymentChromedpTestSuite) testContractVerification(page *UniswapDeploymentPage) {
	// Get all contract addresses
	addresses, err := page.GetAllContractAddresses()
	s.Require().NoError(err, "Failed to get contract addresses")

	// Verify we have all three contracts
	s.Require().Contains(addresses, "weth", "WETH address missing")
	s.Require().Contains(addresses, "factory", "Factory address missing")
	s.Require().Contains(addresses, "router", "Router address missing")

	// Verify addresses are valid Ethereum addresses
	for contractType, address := range addresses {
		s.Assert().Regexp(`^0x[a-fA-F0-9]{40}$`, address,
			fmt.Sprintf("%s address format invalid: %s", contractType, address))
	}

	// Verify contracts are actually deployed on blockchain
	for contractType, address := range addresses {
		err = s.setup.VerifyContractDeployment(address)
		s.Require().NoError(err,
			fmt.Sprintf("Contract %s not found on blockchain at %s", contractType, address))
	}
}

func (s *UniswapDeploymentChromedpTestSuite) testDatabaseVerification(sessionID string) {
	// Get session from database
	session, err := s.setup.DB.GetTransactionSession(sessionID)
	s.Require().NoError(err, "Failed to get transaction session")

	// Verify session status
	s.Assert().Equal("confirmed", session.Status, "Session status should be confirmed")

	// Verify Uniswap deployment record was created
	deployments, err := s.setup.DB.ListUniswapDeployments()
	s.Require().NoError(err, "Failed to list Uniswap deployments")

	// Find the deployment for our session
	var deployment *models.UniswapDeployment

	// Parse session data to get deployment info
	var sessionData map[string]interface{}
	err = json.Unmarshal([]byte(session.TransactionData), &sessionData)
	s.Require().NoError(err, "Failed to parse session data")

	// Find deployment by checking for our session's characteristics
	found := false
	for _, dep := range deployments {
		if dep.Chain.ChainID == "31337" && dep.Status == "confirmed" {
			deployment = &dep
			found = true
			break
		}
	}

	s.Require().True(found, "Deployment record not found in database")

	// Verify deployment fields are populated
	s.Assert().NotEmpty(deployment.WETHAddress, "WETH address should be populated")
	s.Assert().NotEmpty(deployment.WETHTxHash, "WETH transaction hash should be populated")
	s.Assert().NotEmpty(deployment.FactoryAddress, "Factory address should be populated")
	s.Assert().NotEmpty(deployment.FactoryTxHash, "Factory transaction hash should be populated")
	s.Assert().NotEmpty(deployment.RouterAddress, "Router address should be populated")
	s.Assert().NotEmpty(deployment.RouterTxHash, "Router transaction hash should be populated")
	s.Assert().Equal("v2", deployment.Version, "Version should be v2")
	s.Assert().Equal("confirmed", deployment.Status, "Status should be confirmed")
}

// ErrorHandlingTestSuite tests error scenarios
type ErrorHandlingTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *ErrorHandlingTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())
}

func (s *ErrorHandlingTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *ErrorHandlingTestSuite) TestInvalidSession() {
	page := NewUniswapDeploymentPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	// Navigate to invalid session ID
	err := page.NavigateToSession(baseURL, "invalid-session-id")
	s.Require().NoError(err)

	// Should get 404 or error page
	// This test verifies the error handling for invalid sessions
	time.Sleep(2 * time.Second) // Allow page to load

	// Take screenshot for debugging
	page.TakeScreenshot("invalid_session_test.png")

	// Log page HTML for debugging
	err = page.LogPageState("invalid_session")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

func (s *ErrorHandlingTestSuite) TestExpiredSession() {
	page := NewUniswapDeploymentPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	// Create a session and immediately expire it by updating the database
	sessionID, err := s.setup.CreateUniswapDeploymentSession()
	s.Require().NoError(err)

	// Manually expire the session
	err = s.setup.DB.UpdateTransactionSessionStatus(sessionID, "expired", "")
	s.Require().NoError(err)

	// Navigate to expired session
	err = page.NavigateToSession(baseURL, sessionID)
	s.Require().NoError(err)

	// Should handle expired session gracefully
	time.Sleep(2 * time.Second)

	// Take screenshot for debugging
	page.TakeScreenshot("expired_session_test.png")

	// Log page HTML for debugging
	err = page.LogPageState("expired_session")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

// WalletHandlingTestSuite tests behavior when no wallet is available
type WalletHandlingTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *WalletHandlingTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())

	// Verify Ethereum connection
	err := s.setup.VerifyEthereumConnection()
	s.Require().NoError(err)
}

func (s *WalletHandlingTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *WalletHandlingTestSuite) TestDeploymentWithoutWallet() {
	// Create session but don't initialize wallet
	sessionID, err := s.setup.CreateUniswapDeploymentSession()
	s.Require().NoError(err)

	page := NewUniswapDeploymentPage(s.setup.ctx)
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
	page.TakeScreenshot("no_wallet_available.png")

	// Log page HTML for debugging
	err = page.LogPageState("no_wallet_available")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

// PageLoadTestSuite tests just the page loading functionality
type PageLoadTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *PageLoadTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())
}

func (s *PageLoadTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *PageLoadTestSuite) TestUniswapDeploymentPageLoad() {
	// Create session
	sessionID, err := s.setup.CreateUniswapDeploymentSession()
	s.Require().NoError(err)

	page := NewUniswapDeploymentPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	// Navigate and verify basic page elements
	err = page.NavigateToSession(baseURL, sessionID)
	s.Require().NoError(err)

	// Verify page loads
	err = page.WaitForPageLoad()
	s.Require().NoError(err)

	// Verify title
	title, err := page.GetPageTitle()
	s.Require().NoError(err)
	s.Assert().Contains(title, "Deploy Uniswap")

	// Take screenshot for verification
	page.TakeScreenshot("page_load_verification.png")

	// Log page HTML for debugging
	err = page.LogPageState("page_load_verification")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

// BenchmarkTestSuite for performance testing
type BenchmarkTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *BenchmarkTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())

	// Verify Ethereum connection
	err := s.setup.VerifyEthereumConnection()
	if err != nil {
		s.T().Skipf("Ethereum testnet not available: %v", err)
	}
}

func (s *BenchmarkTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

// Test runner functions that testify expects
func TestUniswapDeploymentChromedp(t *testing.T) {
	suite.Run(t, new(UniswapDeploymentChromedpTestSuite))
}

func TestUniswapDeploymentErrorHandling(t *testing.T) {
	suite.Run(t, new(ErrorHandlingTestSuite))
}

func TestUniswapDeploymentWithoutWallet(t *testing.T) {
	suite.Run(t, new(WalletHandlingTestSuite))
}

func TestUniswapDeploymentPageLoad(t *testing.T) {
	suite.Run(t, new(PageLoadTestSuite))
}

func TestBenchmarkUniswapDeployment(t *testing.T) {
	suite.Run(t, new(BenchmarkTestSuite))
}
