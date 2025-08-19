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
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

// UniswapDeploymentTestSuite defines the test suite for Uniswap deployment API
type UniswapDeploymentTestSuite struct {
	suite.Suite
	setup *TestSetup
}

// SetupSuite runs once before all tests in the suite
func (s *UniswapDeploymentTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())

	// Verify Ethereum connection first
	err := s.setup.VerifyEthereumConnection()
	s.Require().NoError(err, "Ethereum testnet should be running on localhost:8545 (run 'make e2e-network')")

	// Test server health
	s.setup.AssertServerHealth()
}

// TearDownSuite runs once after all tests in the suite
func (s *UniswapDeploymentTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

// TestCreateUniswapDeploymentSession tests the main deployment workflow
func (s *UniswapDeploymentTestSuite) TestCreateUniswapDeploymentSession() {
	// Create Uniswap deployment record
	deployment := &models.UniswapDeployment{
		Version: "v2",
		ChainID: s.setup.GetTestChainID(),
		Status:  models.TransactionStatusPending,
	}

	err := s.setup.DB.CreateUniswapDeployment(deployment)
	s.Require().NoError(err)
	s.Require().NotZero(deployment.ID)

	// Generate deployment data using utils
	deploymentData, err := utils.DeployV2Uniswap("ethereum", TESTNET_CHAIN_ID)
	s.Require().NoError(err)

	// Create session data with deployment information
	sessionData := map[string]interface{}{
		"uniswap_deployment_id": deployment.ID,
		"deployment_data":       deploymentData,
		"metadata": []map[string]interface{}{
			{
				"title": "Deployment Type",
				"value": "Uniswap V2 Infrastructure",
			},
			{
				"title": "Chain",
				"value": "Ethereum Testnet",
			},
			{
				"title": "Version",
				"value": "v2",
			},
		},
	}

	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	// Create transaction session
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"deploy_uniswap",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(sessionID)

	s.T().Logf("✓ Created Uniswap deployment session: %s", sessionID)

	// Test deployment page endpoint
	s.testDeploymentPage(sessionID)

	// Test API endpoint
	s.testDeploymentAPI(sessionID)

	// Test confirmation endpoint
	s.testDeploymentConfirmation(sessionID, deployment)
}

func (s *UniswapDeploymentTestSuite) testDeploymentPage(sessionID string) {
	deployURL := fmt.Sprintf("/deploy-uniswap/%s", sessionID)
	resp, err := s.setup.MakeAPIRequest("GET", deployURL)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("text/html", resp.Header.Get("Content-Type"))

	// Read and verify HTML content
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	htmlContent := string(body)
	s.Assert().Contains(htmlContent, "Deploy Uniswap Infrastructure")
	s.Assert().Contains(htmlContent, sessionID)
	s.Assert().Contains(htmlContent, "wallet-connection.js")
	s.Assert().Contains(htmlContent, "deploy-uniswap.js")

	s.T().Logf("✓ Deployment page loaded successfully")
}

func (s *UniswapDeploymentTestSuite) testDeploymentAPI(sessionID string) {
	apiURL := fmt.Sprintf("/api/deploy-uniswap/%s", sessionID)
	resp, err := s.setup.MakeAPIRequest("GET", apiURL)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusOK, resp.StatusCode)
	s.Assert().Equal("application/json", resp.Header.Get("Content-Type"))

	// Parse response
	var apiResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	s.Require().NoError(err)

	// Verify basic session information
	s.Assert().Equal(sessionID, apiResponse["session_id"])
	s.Assert().Equal("deploy_uniswap", apiResponse["session_type"])
	s.Assert().Equal("ethereum", apiResponse["chain_type"])
	s.Assert().Equal(TESTNET_CHAIN_ID, apiResponse["chain_id"])
	s.Assert().Equal("pending", apiResponse["status"])
	s.Assert().Equal("v2", apiResponse["version"])

	// Verify deployment data structure (different from transaction_data)
	deploymentData, ok := apiResponse["deployment_data"].(map[string]interface{})
	s.Require().True(ok, "deployment_data should be present and be a map")

	// Verify contracts structure
	contracts, ok := deploymentData["contracts"].(map[string]interface{})
	s.Require().True(ok, "contracts should be present in deployment_data")

	// Check required contracts
	expectedContracts := []string{"weth9", "factory", "router"}
	for _, contractName := range expectedContracts {
		contract, ok := contracts[contractName].(map[string]interface{})
		s.Require().True(ok, fmt.Sprintf("%s contract should be present", contractName))

		// Each contract should have name (bytecode may be empty for placeholder contracts)
		s.Assert().NotEmpty(contract["name"], fmt.Sprintf("%s should have a name", contractName))
		// Note: bytecode is empty in current implementation as it's fetched separately
	}

	// Verify metadata
	metadata, ok := apiResponse["metadata"].([]interface{})
	s.Require().True(ok, "metadata should be present")
	s.Assert().Greater(len(metadata), 0, "metadata should not be empty")

	// Verify contracts to deploy list
	contractsToDeploy, ok := apiResponse["contracts_to_deploy"].([]interface{})
	s.Require().True(ok, "contracts_to_deploy should be present")
	s.Assert().Equal(3, len(contractsToDeploy), "should have 3 contracts to deploy")
	s.Assert().Contains(contractsToDeploy, "WETH9")
	s.Assert().Contains(contractsToDeploy, "Factory")
	s.Assert().Contains(contractsToDeploy, "Router")

	s.T().Logf("✓ API endpoint returned correct deployment data")
}

func (s *UniswapDeploymentTestSuite) testDeploymentConfirmation(sessionID string, deployment *models.UniswapDeployment) {
	// Deploy dummy contracts to simulate Uniswap deployment
	account := s.setup.GetPrimaryTestAccount()

	// Deploy simple ERC20 contracts as proxies for WETH9, Factory, and Router
	// This is just for testing the API endpoint - not real Uniswap contracts
	wethResult, err := s.setup.DeployContract(
		account,
		GetSimpleERC20Contract(),
		"SimpleERC20",
		"Wrapped ETH", // name
		"WETH",        // symbol
		big.NewInt(0), // totalSupply - WETH starts with 0
	)
	s.Require().NoError(err)
	s.T().Logf("✓ Deployed WETH proxy at %s", wethResult.ContractAddress.Hex())

	// Deploy Factory proxy
	factoryResult, err := s.setup.DeployContract(
		account,
		GetSimpleERC20Contract(),
		"SimpleERC20",
		"Factory Token", // name
		"FACT",          // symbol
		big.NewInt(0),   // totalSupply
	)
	s.Require().NoError(err)
	s.T().Logf("✓ Deployed Factory proxy at %s", factoryResult.ContractAddress.Hex())

	// Deploy Router proxy
	routerResult, err := s.setup.DeployContract(
		account,
		GetSimpleERC20Contract(),
		"SimpleERC20",
		"Router Token", // name
		"ROUT",         // symbol
		big.NewInt(0),  // totalSupply
	)
	s.Require().NoError(err)
	s.T().Logf("✓ Deployed Router proxy at %s", routerResult.ContractAddress.Hex())

	// Test successful confirmation with real transaction hashes
	confirmURL := fmt.Sprintf("/api/deploy-uniswap/%s/confirm", sessionID)
	confirmData := map[string]interface{}{
		"transaction_hashes": map[string]string{
			"weth":    wethResult.TransactionHash.Hex(),
			"factory": factoryResult.TransactionHash.Hex(),
			"router":  routerResult.TransactionHash.Hex(),
		},
		"contract_addresses": map[string]string{
			"weth":    wethResult.ContractAddress.Hex(),
			"factory": factoryResult.ContractAddress.Hex(),
			"router":  routerResult.ContractAddress.Hex(),
		},
		"deployer_address": account.Address.Hex(),
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

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	s.Require().NoError(err)

	s.Assert().True(response["success"].(bool))
	s.Assert().Equal(sessionID, response["session_id"])
	s.Assert().Equal(string(models.TransactionStatusConfirmed), response["status"])

	// Verify session was updated
	session, err := s.setup.DB.GetTransactionSession(sessionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, session.Status)
	s.Assert().NotEmpty(session.TransactionHash)

	// Verify deployment record was updated
	updatedDeployment, err := s.setup.DB.GetUniswapDeploymentByID(deployment.ID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, updatedDeployment.Status)
	s.Assert().Equal(wethResult.ContractAddress.Hex(), updatedDeployment.WETHAddress)
	s.Assert().Equal(factoryResult.ContractAddress.Hex(), updatedDeployment.FactoryAddress)
	s.Assert().Equal(routerResult.ContractAddress.Hex(), updatedDeployment.RouterAddress)
	s.Assert().Equal(account.Address.Hex(), updatedDeployment.DeployerAddress)

	// Verify transaction hashes were stored
	s.Assert().Equal(wethResult.TransactionHash.Hex(), updatedDeployment.WETHTxHash)
	s.Assert().Equal(factoryResult.TransactionHash.Hex(), updatedDeployment.FactoryTxHash)
	s.Assert().Equal(routerResult.TransactionHash.Hex(), updatedDeployment.RouterTxHash)

	s.T().Logf("✓ Successfully models.TransactionStatusConfirmed Uniswap deployment")
	s.T().Logf("  WETH9: %s (tx: %s)", updatedDeployment.WETHAddress, updatedDeployment.WETHTxHash)
	s.T().Logf("  Factory: %s (tx: %s)", updatedDeployment.FactoryAddress, updatedDeployment.FactoryTxHash)
	s.T().Logf("  Router: %s (tx: %s)", updatedDeployment.RouterAddress, updatedDeployment.RouterTxHash)
}

// ErrorHandlingTestSuite tests error scenarios
type ErrorHandlingTestSuite struct {
	suite.Suite
	setup *TestSetup
}

func (s *ErrorHandlingTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())
}

func (s *ErrorHandlingTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *ErrorHandlingTestSuite) TestInvalidSessionID() {
	// Test API endpoint specifically (not HTML page)
	resp, err := s.setup.MakeAPIRequest("GET", "/api/deploy-uniswap/invalid-session-id")
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ErrorHandlingTestSuite) TestWrongSessionType() {
	// Create a regular deployment session
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"deploy", // Wrong type, should be "deploy_uniswap"
		"ethereum",
		TESTNET_CHAIN_ID,
		`{"test": "data"}`,
	)
	s.Require().NoError(err)

	// Try to access it as a Uniswap deployment session
	resp, err := s.setup.MakeAPIRequest("GET", fmt.Sprintf("/api/deploy-uniswap/%s", sessionID))
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)

	var errorResponse map[string]string
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	s.Require().NoError(err)
	s.Assert().Contains(errorResponse["error"], "Invalid session type")
}

func (s *ErrorHandlingTestSuite) TestMissingDeploymentID() {
	// Create session without uniswap_deployment_id
	sessionID, err := s.setup.DB.CreateTransactionSession(
		"deploy_uniswap",
		"ethereum",
		TESTNET_CHAIN_ID,
		`{"missing": "deployment_id"}`,
	)
	s.Require().NoError(err)

	resp, err := s.setup.MakeAPIRequest("GET", fmt.Sprintf("/api/deploy-uniswap/%s", sessionID))
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusInternalServerError, resp.StatusCode)
}

func (s *ErrorHandlingTestSuite) TestNonexistentDeployment() {
	// Create session with invalid deployment ID
	sessionData := map[string]interface{}{
		"uniswap_deployment_id": 99999, // Non-existent ID
	}
	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	sessionID, err := s.setup.DB.CreateTransactionSession(
		"deploy_uniswap",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)

	resp, err := s.setup.MakeAPIRequest("GET", fmt.Sprintf("/api/deploy-uniswap/%s", sessionID))
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusNotFound, resp.StatusCode)
}

func (s *ErrorHandlingTestSuite) TestMalformedConfirmationData() {
	// Create valid session first
	deployment := &models.UniswapDeployment{
		Version: "v2",
		ChainID: s.setup.GetTestChainID(),
		Status:  "pending",
	}
	err := s.setup.DB.CreateUniswapDeployment(deployment)
	s.Require().NoError(err)

	sessionData := map[string]interface{}{
		"uniswap_deployment_id": deployment.ID,
	}
	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	sessionID, err := s.setup.DB.CreateTransactionSession(
		"deploy_uniswap",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)

	// Send malformed JSON
	confirmURL := fmt.Sprintf("/api/deploy-uniswap/%s/confirm", sessionID)
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d%s", s.setup.ServerPort, confirmURL), bytes.NewBuffer([]byte("invalid json")))
	s.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Assert().Equal(http.StatusBadRequest, resp.StatusCode)
}

// DatabaseIntegrationTestSuite tests database operations
type DatabaseIntegrationTestSuite struct {
	suite.Suite
	setup *TestSetup
}

func (s *DatabaseIntegrationTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())
}

func (s *DatabaseIntegrationTestSuite) TearDownSuite() {
	if s.setup != nil {
		s.setup.Cleanup()
	}
}

func (s *DatabaseIntegrationTestSuite) TestDeploymentRecordManagement() {
	// Create deployment
	chainID := s.setup.GetTestChainID()

	deployment := &models.UniswapDeployment{
		Version: "v2",
		ChainID: chainID,
		Status:  "pending",
	}

	err := s.setup.DB.CreateUniswapDeployment(deployment)
	s.Require().NoError(err)
	s.Assert().NotZero(deployment.ID)

	// Retrieve deployment
	retrieved, err := s.setup.DB.GetUniswapDeploymentByID(deployment.ID)
	s.Require().NoError(err)

	s.Assert().Equal(deployment.Version, retrieved.Version)
	s.Assert().Equal(deployment.ChainID, retrieved.ChainID)
	s.Assert().Equal("ethereum", retrieved.Chain.ChainType)

	// Update deployment with contract addresses
	contractAddresses := map[string]string{
		"weth":    "0x1111111111111111111111111111111111111111",
		"factory": "0x2222222222222222222222222222222222222222",
		"router":  "0x3333333333333333333333333333333333333333",
	}

	transactionHashes := map[string]string{
		"weth":    "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"factory": "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"router":  "0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}

	err = s.setup.DB.UpdateUniswapDeploymentStatus(deployment.ID, "confirmed", contractAddresses, transactionHashes)
	s.Require().NoError(err)

	// Verify update
	updated, err := s.setup.DB.GetUniswapDeploymentByID(deployment.ID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatus("confirmed"), updated.Status)
	s.Assert().Equal(contractAddresses["weth"], updated.WETHAddress)
	s.Assert().Equal(contractAddresses["factory"], updated.FactoryAddress)
	s.Assert().Equal(contractAddresses["router"], updated.RouterAddress)
	s.Assert().Equal(transactionHashes["weth"], updated.WETHTxHash)
	s.Assert().Equal(transactionHashes["factory"], updated.FactoryTxHash)
	s.Assert().Equal(transactionHashes["router"], updated.RouterTxHash)
}

func (s *DatabaseIntegrationTestSuite) TestPreventDuplicateDeployment() {
	// Create another chain for this test
	testChain2 := &models.Chain{
		ChainType: "ethereum",
		RPC:       "http://localhost:9999",
		ChainID:   "9999",
		Name:      "Test Chain 2",
		IsActive:  false,
	}
	err := s.setup.DB.CreateChain(testChain2)
	s.Require().NoError(err)

	// Create models.TransactionStatusConfirmed deployment for a specific chain
	deployment := &models.UniswapDeployment{
		Version: "v2",
		ChainID: testChain2.ID,
		Status:  models.TransactionStatusConfirmed,
	}

	err = s.setup.DB.CreateUniswapDeployment(deployment)
	s.Require().NoError(err)

	// Try to find existing deployment (simulating the tool's duplicate check)
	found, err := s.setup.DB.GetUniswapDeploymentByChain("ethereum", "9999")
	s.Require().NoError(err)
	s.Require().NotNil(found)
	s.Assert().Equal(deployment.ID, found.ID)
	s.Assert().Equal(models.TransactionStatus("confirmed"), found.Status)

	s.T().Logf("✓ Correctly found existing deployment: ID=%d", found.ID)
}

func (s *DatabaseIntegrationTestSuite) TestSessionLifecycle() {
	// Create deployment and session
	deployment := &models.UniswapDeployment{
		Version: "v2",
		ChainID: s.setup.GetTestChainID(),
		Status:  "pending",
	}
	err := s.setup.DB.CreateUniswapDeployment(deployment)
	s.Require().NoError(err)

	sessionData := map[string]interface{}{
		"uniswap_deployment_id": deployment.ID,
		"metadata": []map[string]interface{}{
			{"title": "Test", "value": "Value"},
		},
	}
	sessionDataJSON, err := json.Marshal(sessionData)
	s.Require().NoError(err)

	sessionID, err := s.setup.DB.CreateTransactionSession(
		"deploy_uniswap",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(sessionDataJSON),
	)
	s.Require().NoError(err)

	// Test initial session state
	session, err := s.setup.DB.GetTransactionSession(sessionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusPending, session.Status)
	s.Assert().Equal("deploy_uniswap", session.SessionType)
	s.Assert().True(time.Now().Before(session.ExpiresAt))

	// Test status updates
	testTxHash := "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	err = s.setup.DB.UpdateTransactionSessionStatus(sessionID, "confirmed", testTxHash)
	s.Require().NoError(err)

	session, err = s.setup.DB.GetTransactionSession(sessionID)
	s.Require().NoError(err)
	s.Assert().Equal(models.TransactionStatusConfirmed, session.Status)
	s.Assert().Equal(testTxHash, session.TransactionHash)

	s.T().Logf("✓ Session lifecycle completed successfully")
}

// Test runner functions that testify expects
func TestUniswapDeploymentAPI(t *testing.T) {
	suite.Run(t, new(UniswapDeploymentTestSuite))
}

func TestUniswapDeploymentAPI_ErrorHandling(t *testing.T) {
	suite.Run(t, new(ErrorHandlingTestSuite))
}

func TestUniswapDeploymentAPI_DatabaseIntegration(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}
