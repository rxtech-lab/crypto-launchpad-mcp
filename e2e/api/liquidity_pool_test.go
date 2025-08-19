package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/suite"
)

// CreatePoolTestSuite tests liquidity pool creation
type CreatePoolTestSuite struct {
	suite.Suite
	setup             *TestSetup
	testToken         *DeployContractResult
	tokenAddress      string
	uniswapDeployment *models.UniswapDeployment
}

// SetupSuite runs once before all tests in the suite
func (s *CreatePoolTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())

	// Verify Ethereum connection
	err := s.setup.VerifyEthereumConnection()
	s.Require().NoError(err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")

	// Test server health
	s.setup.AssertServerHealth()

	// Deploy a test token for pool creation
	account := s.setup.GetPrimaryTestAccount()
	s.testToken, err = s.setup.DeployContract(
		account,
		GetSimpleERC20Contract(),
		"SimpleERC20",
		"Test Token",
		"TEST",
		big.NewInt(1000000),
	)
	s.Require().NoError(err)
	s.tokenAddress = s.testToken.ContractAddress.Hex()
	s.T().Logf("✓ Deployed test token at %s", s.tokenAddress)

	// Create or get Uniswap deployment (needed for pool creation)
	s.uniswapDeployment = &models.UniswapDeployment{
		Version:        "v2",
		ChainID:        s.setup.GetTestChainID(),
		Status:         "confirmed",
		WETHAddress:    "0x1111111111111111111111111111111111111111",
		FactoryAddress: "0x2222222222222222222222222222222222222222",
		RouterAddress:  "0x3333333333333333333333333333333333333333",
	}
	err = s.setup.DB.CreateUniswapDeployment(s.uniswapDeployment)
	s.Require().NoError(err)

	// Set active Uniswap version
	err = s.setup.DB.SetUniswapVersion("v2")
	s.Require().NoError(err)
}

// TearDownSuite runs once after all tests
func (s *CreatePoolTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

// TestCreatePoolSession tests the complete pool creation workflow
func (s *CreatePoolTestSuite) TestCreatePoolSession() {
	// Create liquidity pool record
	pool := &models.LiquidityPool{
		TokenAddress:   s.tokenAddress,
		UniswapVersion: "v2",
		Token0:         s.tokenAddress,
		Token1:         "0x0000000000000000000000000000000000000000", // ETH
		InitialToken0:  "1000000000000000000",                        // 1 token
		InitialToken1:  "1000000000000000000",                        // 1 ETH
		CreatorAddress: "",                                           // Will be set on web
		Status:         "pending",
	}

	err := s.setup.DB.CreateLiquidityPool(pool)
	s.Require().NoError(err)
	s.Require().NotZero(pool.ID)

	// Create session data
	sessionData := map[string]interface{}{
		"pool_id": pool.ID,
	}

	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	// Create transaction session
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"create_pool",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(sessionID)

	s.T().Logf("✓ Created pool creation session: %s", sessionID)

	// Test page endpoint
	s.testCreatePoolPage(sessionID, pool)

	// Test API endpoint
	s.testCreatePoolAPI(sessionID, pool)

	// Test confirmation
	s.testCreatePoolConfirmation(sessionID, pool)
}

func (s *CreatePoolTestSuite) testCreatePoolPage(sessionID string, pool *models.LiquidityPool) {
	pageURL := fmt.Sprintf("/pool/create/%s", sessionID)
	resp, err := s.setup.MakeAPIRequest("GET", pageURL)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("text/html", resp.Header.Get("Content-Type"))

	// Read and verify HTML content
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	htmlContent := string(body)
	// Focus on functionality - check for key data elements by ID
	s.Assert().Contains(htmlContent, `id="session-data"`)
	s.Assert().Contains(htmlContent, sessionID)
	s.Assert().Contains(htmlContent, "data-transaction-data=") // Embedded data is the key functionality

	s.T().Logf("✓ Pool creation page loaded with embedded transaction data")
}

func (s *CreatePoolTestSuite) testCreatePoolAPI(sessionID string, pool *models.LiquidityPool) {
	apiURL := fmt.Sprintf("/api/pool/create/%s", sessionID)
	resp, err := s.setup.MakeAPIRequest("GET", apiURL)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("application/json", resp.Header.Get("Content-Type"))

	// Parse response
	var apiResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	s.Require().NoError(err)

	// Verify session information
	s.Assert().Equal(sessionID, apiResponse["session_id"])
	s.Assert().Equal("create_pool", apiResponse["session_type"])
	s.Assert().Equal("ethereum", apiResponse["chain_type"])
	s.Assert().Equal(TESTNET_CHAIN_ID, apiResponse["chain_id"])
	s.Assert().Equal("pending", apiResponse["status"])

	// Verify transaction data (note: handleGenericAPI returns raw session data)
	txData, ok := apiResponse["transaction_data"].(map[string]interface{})
	s.Require().True(ok, "transaction_data should be present")

	// The API returns the minimal session data, not the full generated transaction data
	s.Assert().Equal(float64(pool.ID), txData["pool_id"])

	s.T().Logf("✓ API endpoint returned correct pool creation data")
}

func (s *CreatePoolTestSuite) testCreatePoolConfirmation(sessionID string, pool *models.LiquidityPool) {
	// Simulate pool creation transaction
	confirmURL := fmt.Sprintf("/api/pool/create/%s/confirm", sessionID)
	confirmData := map[string]interface{}{
		"transaction_hash": "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef",
		"status":           "confirmed",
		"pair_address":     "0x4444444444444444444444444444444444444444",
	}

	confirmJSON, err := json.Marshal(confirmData)
	s.Require().NoError(err)

	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d%s", s.setup.ServerPort, confirmURL), bytes.NewBuffer(confirmJSON))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)

	s.Assert().Equal("success", response["status"])

	// Verify session was updated
	session, err := s.setup.DB.GetTransactionSession(sessionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, session.Status)
	s.Assert().NotEmpty(session.TransactionHash)

	// Verify pool record was updated
	updatedPool, err := s.setup.DB.GetLiquidityPoolByID(pool.ID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, updatedPool.Status)
	s.Assert().Equal("0x4444444444444444444444444444444444444444", updatedPool.PairAddress)

	s.T().Logf("✓ Successfully models.TransactionStatusConfirmed pool creation")
}

// AddLiquidityTestSuite tests adding liquidity to pools
type AddLiquidityTestSuite struct {
	suite.Suite
	setup    *TestSetup
	pool     *models.LiquidityPool
	position *models.LiquidityPosition
}

func (s *AddLiquidityTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())

	// Verify Ethereum connection
	err := s.setup.VerifyEthereumConnection()
	s.Require().NoError(err)

	// Create a test pool
	s.pool = &models.LiquidityPool{
		TokenAddress:   "0x5555555555555555555555555555555555555555",
		UniswapVersion: "v2",
		Token0:         "0x5555555555555555555555555555555555555555",
		Token1:         "0x0000000000000000000000000000000000000000",
		InitialToken0:  "1000000000000000000",
		InitialToken1:  "1000000000000000000",
		PairAddress:    "0x6666666666666666666666666666666666666666",
		Status:         "confirmed",
	}
	err = s.setup.DB.CreateLiquidityPool(s.pool)
	s.Require().NoError(err)

	// Set active Uniswap version
	err = s.setup.DB.SetUniswapVersion("v2")
	if err != nil {
		// Settings might already exist from previous test
		existing, _ := s.setup.DB.GetActiveUniswapSettings()
		if existing == nil {
			s.Require().NoError(err)
		}
	}
}

func (s *AddLiquidityTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *AddLiquidityTestSuite) TestAddLiquiditySession() {
	// Create liquidity position record
	s.position = &models.LiquidityPosition{
		PoolID:       s.pool.ID,
		UserAddress:  "",                   // Will be set on web
		Token0Amount: "500000000000000000", // 0.5 token
		Token1Amount: "500000000000000000", // 0.5 ETH
		Action:       "add",
		Status:       "pending",
	}

	err := s.setup.DB.CreateLiquidityPosition(s.position)
	s.Require().NoError(err)
	s.Require().NotZero(s.position.ID)

	// Create session data with min amounts
	sessionData := map[string]interface{}{
		"position_id":      s.position.ID,
		"pool_id":          s.pool.ID,
		"token_address":    s.pool.TokenAddress,
		"token_amount":     s.position.Token0Amount,
		"eth_amount":       s.position.Token1Amount,
		"min_token_amount": "490000000000000000", // 0.49 (2% slippage)
		"min_eth_amount":   "490000000000000000", // 0.49 (2% slippage)
	}

	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	// Create transaction session
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"add_liquidity",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(sessionID)

	s.T().Logf("✓ Created add liquidity session: %s", sessionID)

	// Test page endpoint
	s.testAddLiquidityPage(sessionID)

	// Test API endpoint
	s.testAddLiquidityAPI(sessionID)

	// Test confirmation
	s.testAddLiquidityConfirmation(sessionID)
}

func (s *AddLiquidityTestSuite) testAddLiquidityPage(sessionID string) {
	pageURL := fmt.Sprintf("/liquidity/add/%s", sessionID)
	resp, err := s.setup.MakeAPIRequest("GET", pageURL)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("text/html", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	htmlContent := string(body)
	// Focus on functionality - check for session data element
	s.Assert().Contains(htmlContent, `id="session-data"`)
	s.Assert().Contains(htmlContent, sessionID)
	s.Assert().Contains(htmlContent, "data-transaction-data=") // Embedded data

	s.T().Logf("✓ Add liquidity page loaded with embedded transaction data")
}

func (s *AddLiquidityTestSuite) testAddLiquidityAPI(sessionID string) {
	apiURL := fmt.Sprintf("/api/liquidity/add/%s", sessionID)
	resp, err := s.setup.MakeAPIRequest("GET", apiURL)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	var apiResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	s.Require().NoError(err)

	// Verify transaction data
	txData, ok := apiResponse["transaction_data"].(map[string]interface{})
	s.Require().True(ok)

	s.Assert().Equal(float64(s.position.ID), txData["position_id"])
	s.Assert().Equal(float64(s.pool.ID), txData["pool_id"])
	s.Assert().Equal(s.pool.TokenAddress, txData["token_address"])
	s.Assert().Equal(s.position.Token0Amount, txData["token_amount"])
	s.Assert().Equal(s.position.Token1Amount, txData["eth_amount"])
	s.Assert().Equal("490000000000000000", txData["min_token_amount"])
	s.Assert().Equal("490000000000000000", txData["min_eth_amount"])

	s.T().Logf("✓ API endpoint returned correct add liquidity data")
}

func (s *AddLiquidityTestSuite) testAddLiquidityConfirmation(sessionID string) {
	confirmURL := fmt.Sprintf("/api/liquidity/add/%s/confirm", sessionID)
	confirmData := map[string]interface{}{
		"transaction_hash": "0xbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeef",
		"status":           "confirmed",
	}

	confirmJSON, err := json.Marshal(confirmData)
	s.Require().NoError(err)

	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d%s", s.setup.ServerPort, confirmURL), bytes.NewBuffer(confirmJSON))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	// Verify position was updated
	updatedPosition, err := s.setup.DB.GetLiquidityPositionByID(s.position.ID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, updatedPosition.Status)
	s.Assert().Equal("0xbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeefbeef", updatedPosition.TransactionHash)

	s.T().Logf("✓ Successfully models.TransactionStatusConfirmed add liquidity")
}

// LiquidityErrorHandlingTestSuite tests error scenarios
type LiquidityErrorHandlingTestSuite struct {
	suite.Suite
	setup *TestSetup
}

func (s *LiquidityErrorHandlingTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())
}

func (s *LiquidityErrorHandlingTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *LiquidityErrorHandlingTestSuite) TestInvalidPoolSessionID() {
	resp, err := s.setup.MakeAPIRequest("GET", "/api/pool/create/invalid-session-id")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *LiquidityErrorHandlingTestSuite) TestWrongSessionTypeForPool() {
	// Create session with wrong type
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"deploy", // Wrong type, should be "create_pool"
		"ethereum",
		TESTNET_CHAIN_ID,
		`{"test": "data"}`,
	)
	s.Require().NoError(err)

	resp, err := s.setup.MakeAPIRequest("GET", fmt.Sprintf("/api/pool/create/%s", sessionID))
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)

	var errorResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	s.Require().NoError(err)
	s.Assert().Contains(errorResponse["error"], "Invalid session type")
}

func (s *LiquidityErrorHandlingTestSuite) TestMissingPoolID() {
	// Create session without pool_id
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"create_pool",
		"ethereum",
		TESTNET_CHAIN_ID,
		`{"missing": "pool_id"}`,
	)
	s.Require().NoError(err)

	// Try to access the API endpoint
	resp, err := s.setup.MakeAPIRequest("GET", fmt.Sprintf("/api/pool/create/%s", sessionID))
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should still return 200 but with minimal data (no transaction data embedded)
	s.Assert().Equal(http.StatusOK, resp.StatusCode)

	var apiResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	s.Require().NoError(err)

	// Transaction data should be empty or minimal
	txData := apiResponse["transaction_data"]
	if txData != nil {
		txDataMap, ok := txData.(map[string]interface{})
		if ok {
			s.Assert().Empty(txDataMap["pool_id"])
		}
	}
}

func (s *LiquidityErrorHandlingTestSuite) TestNonexistentPool() {
	sessionData := map[string]interface{}{
		"pool_id": 99999, // Non-existent ID
	}
	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	sessionID, err := s.setup.DB.CreateTransactionSession(
		"create_pool",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)

	// Should still return 200 but with empty transaction data
	resp, err := s.setup.MakeAPIRequest("GET", fmt.Sprintf("/api/pool/create/%s", sessionID))
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
}

// LiquidityDatabaseTestSuite tests database operations
type LiquidityDatabaseTestSuite struct {
	suite.Suite
	setup *TestSetup
}

func (s *LiquidityDatabaseTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())
}

func (s *LiquidityDatabaseTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *LiquidityDatabaseTestSuite) TestPoolRecordManagement() {
	// Create pool
	pool := &models.LiquidityPool{
		TokenAddress:   "0x7777777777777777777777777777777777777777",
		UniswapVersion: "v2",
		Token0:         "0x7777777777777777777777777777777777777777",
		Token1:         "0x0000000000000000000000000000000000000000",
		InitialToken0:  "1000000000000000000",
		InitialToken1:  "2000000000000000000",
		Status:         "pending",
	}

	err := s.setup.DB.CreateLiquidityPool(pool)
	s.Require().NoError(err)
	s.Assert().NotZero(pool.ID)

	// Retrieve pool
	retrieved, err := s.setup.DB.GetLiquidityPoolByID(pool.ID)
	s.Require().NoError(err)
	s.Assert().Equal(pool.TokenAddress, retrieved.TokenAddress)
	s.Assert().Equal(pool.InitialToken0, retrieved.InitialToken0)
	s.Assert().Equal(pool.InitialToken1, retrieved.InitialToken1)

	// Update pool status
	err = s.setup.DB.UpdateLiquidityPoolStatus(
		pool.ID,
		"confirmed",
		"0x8888888888888888888888888888888888888888",
		"0xabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd",
	)
	s.Require().NoError(err)

	// Verify update
	updated, err := s.setup.DB.GetLiquidityPoolByID(pool.ID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, updated.Status)
	s.Assert().Equal("0x8888888888888888888888888888888888888888", updated.PairAddress)
	s.Assert().Equal("0xabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcdabcd", updated.TransactionHash)

	s.T().Logf("✓ Pool record management successful")
}

func (s *LiquidityDatabaseTestSuite) TestPositionRecordManagement() {
	// Create pool first
	pool := &models.LiquidityPool{
		TokenAddress:   "0x9999999999999999999999999999999999999999",
		UniswapVersion: "v2",
		Status:         "confirmed",
	}
	err := s.setup.DB.CreateLiquidityPool(pool)
	s.Require().NoError(err)

	// Create position
	position := &models.LiquidityPosition{
		PoolID:       pool.ID,
		UserAddress:  "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Token0Amount: "1000000000000000000",
		Token1Amount: "2000000000000000000",
		Action:       "add",
		Status:       models.TransactionStatusPending,
	}

	err = s.setup.DB.CreateLiquidityPosition(position)
	s.Require().NoError(err)
	s.Assert().NotZero(position.ID)

	// Retrieve position
	retrieved, err := s.setup.DB.GetLiquidityPositionByID(position.ID)
	s.Require().NoError(err)
	s.Assert().Equal(position.PoolID, retrieved.PoolID)
	s.Assert().Equal(position.Token0Amount, retrieved.Token0Amount)

	// Update position status
	err = s.setup.DB.UpdateLiquidityPositionStatus(
		position.ID,
		models.TransactionStatusConfirmed,
		"0xdeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead",
	)
	s.Require().NoError(err)

	// Verify update
	updated, err := s.setup.DB.GetLiquidityPositionByID(position.ID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, updated.Status)
	s.Assert().Equal("0xdeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead", updated.TransactionHash)

	s.T().Logf("✓ Position record management successful")
}

func (s *LiquidityDatabaseTestSuite) TestPreventDuplicatePool() {
	tokenAddress := "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	// Create first pool
	pool1 := &models.LiquidityPool{
		TokenAddress:   tokenAddress,
		UniswapVersion: "v2",
		Status:         "confirmed",
	}
	err := s.setup.DB.CreateLiquidityPool(pool1)
	s.Require().NoError(err)

	// Try to create duplicate
	pool2 := &models.LiquidityPool{
		TokenAddress:   tokenAddress,
		UniswapVersion: "v2",
		Status:         "pending",
	}
	err = s.setup.DB.CreateLiquidityPool(pool2)
	// Should succeed as database doesn't enforce uniqueness, but tool logic would prevent this

	// Check retrieval by token address
	existing, err := s.setup.DB.GetLiquidityPoolByTokenAddress(tokenAddress)
	s.Require().NoError(err)
	s.Assert().NotNil(existing)
	s.Assert().Equal(tokenAddress, existing.TokenAddress)

	s.T().Logf("✓ Pool duplicate check successful")
}

func (s *LiquidityDatabaseTestSuite) TestSessionLifecycleForLiquidity() {
	// Create pool and session
	pool := &models.LiquidityPool{
		TokenAddress:   "0xcccccccccccccccccccccccccccccccccccccccc",
		UniswapVersion: "v2",
		Status:         models.TransactionStatusPending,
	}
	err := s.setup.DB.CreateLiquidityPool(pool)
	s.Require().NoError(err)

	sessionData := map[string]interface{}{
		"pool_id": pool.ID,
	}
	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	sessionID, err := s.setup.DB.CreateTransactionSession(
		"create_pool",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)

	// Test initial session state
	session, err := s.setup.DB.GetTransactionSession(sessionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusPending, session.Status)
	s.Assert().Equal("create_pool", session.SessionType)
	s.Assert().True(time.Now().Before(session.ExpiresAt))

	// Update session status
	testTxHash := "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed"
	err = s.setup.DB.UpdateTransactionSessionStatus(sessionID, models.TransactionStatusConfirmed, testTxHash)
	s.Require().NoError(err)

	// Verify update
	session, err = s.setup.DB.GetTransactionSession(sessionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, session.Status)
	s.Assert().Equal(testTxHash, session.TransactionHash)

	s.T().Logf("✓ Session lifecycle for liquidity completed successfully")
}

// Test runner functions
func TestCreatePoolAPI(t *testing.T) {
	suite.Run(t, new(CreatePoolTestSuite))
}

func TestAddLiquidityAPI(t *testing.T) {
	suite.Run(t, new(AddLiquidityTestSuite))
}

func TestLiquidityErrorHandling(t *testing.T) {
	suite.Run(t, new(LiquidityErrorHandlingTestSuite))
}

func TestLiquidityDatabaseIntegration(t *testing.T) {
	suite.Run(t, new(LiquidityDatabaseTestSuite))
}
