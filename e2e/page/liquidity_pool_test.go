package api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/suite"
)

// CreatePoolChromedpTestSuite tests the complete create pool workflow using chromedp
type CreatePoolChromedpTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *CreatePoolChromedpTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())

	// Verify Ethereum connection is available
	err := s.setup.VerifyEthereumConnection()
	s.Require().NoError(err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")
}

func (s *CreatePoolChromedpTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *CreatePoolChromedpTestSuite) TestCreatePoolWorkflow() {
	// First deploy a test token that we'll create a pool for
	tokenAddress, err := s.deployTestToken()
	s.Require().NoError(err, "Failed to deploy test token")

	// Create Uniswap deployment first
	err = s.setupUniswapInfrastructure()
	s.Require().NoError(err, "Failed to setup Uniswap infrastructure")

	// Create a pool creation session
	sessionID, poolID, err := s.createPoolSession(tokenAddress)
	s.Require().NoError(err, "Failed to create pool session")

	// Create page object
	page := NewLiquidityPoolPage(s.setup.ctx)

	// Navigate to the create pool page
	baseURL := s.setup.GetBaseURL()
	err = page.NavigateToCreatePoolSession(baseURL, sessionID)
	s.Require().NoError(err, "Failed to navigate to create pool page")

	// Wait for page to load
	err = page.WaitForPageLoad()
	s.Require().NoError(err, "Failed to wait for page load")

	// Verify embedded transaction data
	err = page.VerifyEmbeddedTransactionData()
	s.Assert().NoError(err, "Transaction data should be embedded in page")

	// Initialize test wallet
	err = s.setup.InitializeTestWallet()
	s.Require().NoError(err, "Failed to initialize test wallet")

	// Test wallet connection
	s.testWalletConnection(page)

	// Test pool creation
	s.testPoolCreation(page)

	// Verify database state
	s.verifyPoolDatabaseState(sessionID, poolID)

	// Log final page state
	err = page.LogPageState("create_pool_final")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

func (s *CreatePoolChromedpTestSuite) deployTestToken() (string, error) {
	account := s.setup.GetPrimaryTestAccount()

	// Simple ERC20 token contract
	tokenCode := `// SPDX-License-Identifier: MIT
	pragma solidity ^0.8.0;
	contract TestToken {
		mapping(address => uint256) public balanceOf;
		mapping(address => mapping(address => uint256)) public allowance;
		uint256 public totalSupply;
		string public name = "Test Token";
		string public symbol = "TEST";
		uint8 public decimals = 18;
		
		event Transfer(address indexed from, address indexed to, uint256 value);
		event Approval(address indexed owner, address indexed spender, uint256 value);
		
		constructor() {
			totalSupply = 1000000 * 10**18;
			balanceOf[msg.sender] = totalSupply;
		}
		
		function transfer(address to, uint256 amount) public returns (bool) {
			balanceOf[msg.sender] -= amount;
			balanceOf[to] += amount;
			emit Transfer(msg.sender, to, amount);
			return true;
		}
		
		function approve(address spender, uint256 amount) public returns (bool) {
			allowance[msg.sender][spender] = amount;
			emit Approval(msg.sender, spender, amount);
			return true;
		}
		
		function transferFrom(address from, address to, uint256 amount) public returns (bool) {
			allowance[from][msg.sender] -= amount;
			balanceOf[from] -= amount;
			balanceOf[to] += amount;
			emit Transfer(from, to, amount);
			return true;
		}
	}`

	result, err := s.setup.DeployContract(account, tokenCode, "TestToken")
	if err != nil {
		return "", err
	}

	return result.ContractAddress.Hex(), nil
}

func (s *CreatePoolChromedpTestSuite) setupUniswapInfrastructure() error {
	// Create Uniswap settings
	err := s.setup.DB.SetUniswapVersion("v2")
	if err != nil {
		return fmt.Errorf("failed to set Uniswap version: %w", err)
	}

	// Create a Uniswap deployment record
	deployment := &models.UniswapDeployment{
		ChainID:        1, // Will be updated after getting active chain
		Version:        "v2",
		FactoryAddress: "0x5FbDB2315678afecb367f032d93F642f64180aa3",
		RouterAddress:  "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		WETHAddress:    "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
		Status:         "confirmed",
	}

	// Get active chain
	activeChain, err := s.setup.DB.GetActiveChain()
	if err == nil {
		deployment.ChainID = activeChain.ID
	}

	err = s.setup.DB.CreateUniswapDeployment(deployment)
	if err != nil {
		return fmt.Errorf("failed to create Uniswap deployment: %w", err)
	}

	return nil
}

func (s *CreatePoolChromedpTestSuite) createPoolSession(tokenAddress string) (string, uint, error) {
	// Create pool record
	pool := &models.LiquidityPool{
		TokenAddress:   tokenAddress,
		UniswapVersion: "v2",
		Token0:         tokenAddress,
		Token1:         "0x0000000000000000000000000000000000000000", // ETH
		InitialToken0:  "1000",
		InitialToken1:  "1",
		CreatorAddress: "",
		Status:         "pending",
	}

	err := s.setup.DB.CreateLiquidityPool(pool)
	if err != nil {
		return "", 0, err
	}

	// Create session data
	sessionData := map[string]interface{}{
		"pool_id": pool.ID,
	}

	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", 0, err
	}

	// Get active chain
	activeChain, err := s.setup.DB.GetActiveChain()
	if err != nil {
		return "", 0, err
	}

	// Create transaction session
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"create_pool",
		activeChain.ChainType,
		activeChain.NetworkID,
		string(sessionDataJSON),
	)

	return sessionID, pool.ID, err
}

func (s *CreatePoolChromedpTestSuite) testWalletConnection(page *LiquidityPoolPage) {
	// Wait for wallet selection
	err := page.WaitForWalletSelection()
	s.Assert().NoError(err, "Wallet selection not available")

	// sleep for 1 second
	time.Sleep(1 * time.Second)

	// Select test wallet
	err = page.SelectWallet("test-wallet-e2e")
	s.Assert().NoError(err, "Failed to select test wallet")

	// Connect wallet
	err = page.ClickConnectWallet()
	s.Assert().NoError(err, "Failed to click connect wallet")

	// Wait for connection
	err = page.WaitForWalletConnection()
	s.Assert().NoError(err, "Wallet connection failed")
}

func (s *CreatePoolChromedpTestSuite) testPoolCreation(page *LiquidityPoolPage) {
	// Click create pool button
	err := page.ClickCreatePoolButton()
	s.Require().NoError(err, "Failed to click create pool button")

	// Wait for success state
	err = page.WaitForSuccessState()
	s.Require().NoError(err, "Success state not reached")

	// Get pair address
	pairAddress, err := page.GetPairAddress()
	s.Require().NoError(err, "Failed to get pair address")
	s.Assert().Regexp(`^0x[a-fA-F0-9]{40}$`, pairAddress, "Invalid pair address format")

	// Get transaction hash
	txHash, err := page.GetTransactionHash()
	s.Require().NoError(err, "Failed to get transaction hash")
	s.Assert().Regexp(`^0x[a-fA-F0-9]{64}$`, txHash, "Invalid transaction hash format")
}

func (s *CreatePoolChromedpTestSuite) verifyPoolDatabaseState(sessionID string, poolID uint) {
	// Verify session status
	session, err := s.setup.TxService.GetTransactionSession(sessionID)
	s.Require().NoError(err, "Failed to get transaction session")
	s.Assert().Equal(models.TransactionStatusConfirmed, session.TransactionStatus, "Session should be confirmed")

	// Verify pool record
	pool, err := s.setup.DB.GetLiquidityPoolByID(poolID)
	s.Require().NoError(err, "Failed to get liquidity pool")
	s.Assert().Equal(models.TransactionStatusConfirmed, pool.Status, "Pool should be confirmed")
	s.Assert().NotEmpty(pool.PairAddress, "Pair address should be set")
	s.Assert().NotEmpty(pool.TransactionHash, "Transaction hash should be set")
}

// AddLiquidityChromedpTestSuite tests the add liquidity workflow
type AddLiquidityChromedpTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *AddLiquidityChromedpTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())

	err := s.setup.VerifyEthereumConnection()
	s.Require().NoError(err, "Ethereum testnet required")
}

func (s *AddLiquidityChromedpTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *AddLiquidityChromedpTestSuite) TestAddLiquidityWorkflow() {
	// Setup: Create a pool first
	poolID, err := s.createExistingPool()
	s.Require().NoError(err, "Failed to create existing pool")

	// Create add liquidity session
	sessionID, positionID, err := s.createAddLiquiditySession(poolID)
	s.Require().NoError(err, "Failed to create add liquidity session")

	// Create page object
	page := NewLiquidityPoolPage(s.setup.ctx)

	// Navigate to add liquidity page
	baseURL := s.setup.GetBaseURL()
	err = page.NavigateToAddLiquiditySession(baseURL, sessionID)
	s.Require().NoError(err, "Failed to navigate to add liquidity page")

	// Wait for page load
	err = page.WaitForPageLoad()
	s.Require().NoError(err, "Failed to wait for page load")

	// Verify embedded transaction data
	err = page.VerifyEmbeddedTransactionData()
	s.Assert().NoError(err, "Transaction data should be embedded")

	// Initialize wallet
	err = s.setup.InitializeTestWallet()
	s.Require().NoError(err, "Failed to initialize test wallet")

	// Connect wallet
	s.testWalletConnection(page)

	// Add liquidity
	s.testAddLiquidity(page)

	// Verify database
	s.verifyPositionDatabaseState(sessionID, positionID)

	// Log page state
	err = page.LogPageState("add_liquidity_final")
	if err != nil {
		fmt.Printf("WARNING: Failed to log page state: %v\n", err)
	}
}

func (s *AddLiquidityChromedpTestSuite) createExistingPool() (uint, error) {
	// Create a models.TransactionStatusConfirmed pool
	pool := &models.LiquidityPool{
		TokenAddress:    "0x5FbDB2315678afecb367f032d93F642f64180aa3",
		PairAddress:     "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512",
		UniswapVersion:  "v2",
		Token0:          "0x5FbDB2315678afecb367f032d93F642f64180aa3",
		Token1:          "0x0000000000000000000000000000000000000000",
		CreatorAddress:  "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
		Status:          "confirmed",
		TransactionHash: "0x1234567890abcdef",
	}

	err := s.setup.DB.CreateLiquidityPool(pool)
	if err != nil {
		return 0, err
	}

	return pool.ID, nil
}

func (s *AddLiquidityChromedpTestSuite) createAddLiquiditySession(poolID uint) (string, uint, error) {
	// Create position record
	position := &models.LiquidityPosition{
		PoolID:       poolID,
		UserAddress:  "",
		Token0Amount: "500",
		Token1Amount: "0.5",
		Action:       "add",
		Status:       "pending",
	}

	err := s.setup.DB.CreateLiquidityPosition(position)
	if err != nil {
		return "", 0, err
	}

	// Create session
	sessionData := map[string]interface{}{
		"position_id": position.ID,
	}

	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return "", 0, err
	}

	activeChain, err := s.setup.DB.GetActiveChain()
	if err != nil {
		return "", 0, err
	}

	sessionID, err := s.setup.DB.CreateTransactionSession(
		"add_liquidity",
		activeChain.ChainType,
		activeChain.NetworkID,
		string(sessionDataJSON),
	)

	return sessionID, position.ID, err
}

func (s *AddLiquidityChromedpTestSuite) testWalletConnection(page *LiquidityPoolPage) {
	err := page.WaitForWalletSelection()
	s.Assert().NoError(err)

	time.Sleep(1 * time.Second)
	err = page.SelectWallet("test-wallet-e2e")
	s.Assert().NoError(err)

	time.Sleep(1 * time.Second)
	err = page.ClickConnectWallet()
	s.Assert().NoError(err)

	time.Sleep(1 * time.Second)
	err = page.WaitForWalletConnection()
	s.Assert().NoError(err)
}

func (s *AddLiquidityChromedpTestSuite) testAddLiquidity(page *LiquidityPoolPage) {
	err := page.ClickAddLiquidityButton()
	s.Require().NoError(err)

	err = page.WaitForSuccessState()
	s.Require().NoError(err)

	txHash, err := page.GetTransactionHash()
	s.Require().NoError(err)
	s.Assert().Regexp(`^0x[a-fA-F0-9]{64}$`, txHash)
}

func (s *AddLiquidityChromedpTestSuite) verifyPositionDatabaseState(sessionID string, positionID uint) {
	session, err := s.setup.TxService.GetTransactionSession(sessionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, session.TransactionStatus)

	position, err := s.setup.DB.GetLiquidityPositionByID(positionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, position.Status)
	s.Assert().NotEmpty(position.TransactionHash)
}

// LiquidityErrorHandlingTestSuite tests error scenarios
type LiquidityErrorHandlingTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *LiquidityErrorHandlingTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())
}

func (s *LiquidityErrorHandlingTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *LiquidityErrorHandlingTestSuite) TestInvalidPoolSession() {
	page := NewLiquidityPoolPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	// Navigate to invalid session
	err := page.NavigateToCreatePoolSession(baseURL, "invalid-session-id")
	s.Require().NoError(err)

	time.Sleep(2 * time.Second)

	// Take screenshot
	page.TakeScreenshot("liquidity_invalid_session.png")
	page.LogPageState("liquidity_invalid_session")
}

func (s *LiquidityErrorHandlingTestSuite) TestExpiredLiquiditySession() {
	// Create and expire a session
	pool := &models.LiquidityPool{
		TokenAddress:   "0x123",
		UniswapVersion: "v2",
		Status:         "pending",
	}
	s.setup.DB.CreateLiquidityPool(pool)

	sessionData, _ := json.Marshal(map[string]interface{}{"pool_id": pool.ID})

	activeChain, _ := s.setup.DB.GetActiveChain()
	sessionID, _ := s.setup.DB.CreateTransactionSession(
		"create_pool",
		activeChain.ChainType,
		activeChain.NetworkID,
		string(sessionData),
	)

	// Expire the session
	s.setup.DB.UpdateTransactionSessionStatus(sessionID, "expired", "")

	page := NewLiquidityPoolPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	err := page.NavigateToCreatePoolSession(baseURL, sessionID)
	s.Require().NoError(err)

	time.Sleep(2 * time.Second)

	page.TakeScreenshot("liquidity_expired_session.png")
	page.LogPageState("liquidity_expired_session")
}

// LiquidityPageLoadTestSuite tests page loading
type LiquidityPageLoadTestSuite struct {
	suite.Suite
	setup *ChromedpTestSetup
}

func (s *LiquidityPageLoadTestSuite) SetupSuite() {
	s.setup = NewChromedpTestSetup(s.T())
}

func (s *LiquidityPageLoadTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *LiquidityPageLoadTestSuite) TestCreatePoolPageLoad() {
	// Create valid session
	pool := &models.LiquidityPool{
		TokenAddress:   "0x123",
		UniswapVersion: "v2",
		Status:         "pending",
		InitialToken0:  "1000",
		InitialToken1:  "1",
	}
	s.setup.DB.CreateLiquidityPool(pool)

	sessionData, _ := json.Marshal(map[string]interface{}{"pool_id": pool.ID})
	activeChain, _ := s.setup.DB.GetActiveChain()
	sessionID, _ := s.setup.DB.CreateTransactionSession(
		"create_pool",
		activeChain.ChainType,
		activeChain.NetworkID,
		string(sessionData),
	)

	page := NewLiquidityPoolPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	err := page.NavigateToCreatePoolSession(baseURL, sessionID)
	s.Require().NoError(err)

	err = page.WaitForPageLoad()
	s.Require().NoError(err)

	// Verify embedded data
	err = page.VerifyEmbeddedTransactionData()
	s.Assert().NoError(err, "Transaction data should be embedded")

	title, err := page.GetPageTitle()
	s.Require().NoError(err)
	s.Assert().Contains(title, "Create", "Should have create in title")

	page.TakeScreenshot("create_pool_page_load.png")
	page.LogPageState("create_pool_page_load")
}

func (s *LiquidityPageLoadTestSuite) TestAddLiquidityPageLoad() {
	// Create pool and position
	pool := &models.LiquidityPool{
		TokenAddress: "0x123",
		PairAddress:  "0x456",
		Status:       "confirmed",
	}
	s.setup.DB.CreateLiquidityPool(pool)

	position := &models.LiquidityPosition{
		PoolID:       pool.ID,
		Token0Amount: "100",
		Token1Amount: "0.1",
		Action:       "add",
		Status:       "pending",
	}
	s.setup.DB.CreateLiquidityPosition(position)

	sessionData, _ := json.Marshal(map[string]interface{}{"position_id": position.ID})
	activeChain, _ := s.setup.DB.GetActiveChain()
	sessionID, _ := s.setup.DB.CreateTransactionSession(
		"add_liquidity",
		activeChain.ChainType,
		activeChain.NetworkID,
		string(sessionData),
	)

	page := NewLiquidityPoolPage(s.setup.ctx)
	baseURL := s.setup.GetBaseURL()

	err := page.NavigateToAddLiquiditySession(baseURL, sessionID)
	s.Require().NoError(err)

	err = page.WaitForPageLoad()
	s.Require().NoError(err)

	// Verify embedded data
	err = page.VerifyEmbeddedTransactionData()
	s.Assert().NoError(err, "Transaction data should be embedded")

	page.TakeScreenshot("add_liquidity_page_load.png")
	page.LogPageState("add_liquidity_page_load")
}

// Test runner functions
func TestCreatePoolChromedp(t *testing.T) {
	suite.Run(t, new(CreatePoolChromedpTestSuite))
}

func TestAddLiquidityChromedp(t *testing.T) {
	suite.Run(t, new(AddLiquidityChromedpTestSuite))
}

func TestLiquidityErrorHandling(t *testing.T) {
	suite.Run(t, new(LiquidityErrorHandlingTestSuite))
}

func TestLiquidityPageLoad(t *testing.T) {
	suite.Run(t, new(LiquidityPageLoadTestSuite))
}
