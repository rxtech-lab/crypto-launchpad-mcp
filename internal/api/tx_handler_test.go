package api

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

const (
	TESTNET_RPC      = "http://localhost:8545"
	TESTNET_CHAIN_ID = "31337"
	TESTING_PK_1     = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	TESTING_PK_2     = "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
)

type TxHandlerTestSuite struct {
	suite.Suite
	db                services.DBService
	apiServer         *APIServer
	serverPort        int
	ethClient         *ethclient.Client
	chain             *models.Chain
	template          *models.Template
	deploymentService *services.DeploymentService
	chainService      services.ChainService
	templateService   services.TemplateService
}

func (suite *TxHandlerTestSuite) SetupSuite() {
	// Initialize in-memory database
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Initialize services
	txService := services.NewTransactionService(db.GetDB())
	hookService := services.NewHookService()
	suite.chainService = services.NewChainService(db.GetDB())
	suite.templateService = services.NewTemplateService(db.GetDB())
	suite.deploymentService = services.NewDeploymentService(db.GetDB())

	// Initialize API server
	apiServer := NewAPIServer(db, txService, hookService, suite.chainService)
	apiServer.SetupRoutes()
	port, err := apiServer.Start(nil) // Let it find an available port
	suite.Require().NoError(err)
	suite.apiServer = apiServer
	suite.serverPort = port

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)

	// Initialize Ethereum client
	ethClient, err := ethclient.Dial(TESTNET_RPC)
	suite.Require().NoError(err)
	suite.ethClient = ethClient

	// Setup test chain
	suite.setupTestChain()

	// Setup test template
	suite.setupTestTemplate()
}

func (suite *TxHandlerTestSuite) TearDownSuite() {
	if suite.apiServer != nil {
		suite.apiServer.Shutdown()
	}
	if suite.db != nil {
		suite.db.Close()
	}
	if suite.ethClient != nil {
		suite.ethClient.Close()
	}
}

func (suite *TxHandlerTestSuite) SetupTest() {
	// Clean up any existing sessions for each test
	suite.cleanupTestData()
}

func (suite *TxHandlerTestSuite) setupTestChain() {
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       TESTNET_RPC,
		NetworkID: TESTNET_CHAIN_ID,
		Name:      "Ethereum Testnet",
		IsActive:  true,
	}

	err := suite.chainService.CreateChain(chain)
	suite.Require().NoError(err)
	suite.chain = chain
}

func (suite *TxHandlerTestSuite) setupTestTemplate() {
	contractCode := `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

contract SimpleToken {
    string public name;
    string public symbol;
    uint256 public totalSupply;
    mapping(address => uint256) public balanceOf;

    constructor(string memory _name, string memory _symbol, uint256 _supply) {
        name = _name;
        symbol = _symbol;
        totalSupply = _supply;
        balanceOf[msg.sender] = _supply;
    }
}`

	template := &models.Template{
		Name:         "SimpleToken",
		Description:  "A simple token contract for testing",
		ChainType:    models.TransactionChainTypeEthereum,
		ContractName: "SimpleToken",
		TemplateCode: contractCode,
	}

	err := suite.templateService.CreateTemplate(template)
	suite.Require().NoError(err)
	suite.template = template
}

func (suite *TxHandlerTestSuite) cleanupTestData() {
	// Clean up transaction sessions
	suite.db.GetDB().Where("1 = 1").Delete(&models.TransactionSession{})

	// Clean up deployments
	suite.db.GetDB().Where("1 = 1").Delete(&models.Deployment{})
}

func (suite *TxHandlerTestSuite) verifyEthereumConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := suite.ethClient.NetworkID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get network ID: %w", err)
	}

	if networkID.Cmp(big.NewInt(31337)) != 0 {
		return fmt.Errorf("unexpected network ID: got %s, expected 31337", networkID.String())
	}

	return nil
}

func (suite *TxHandlerTestSuite) createTestSession() string {
	// Create a deployment record first
	deployment := &models.Deployment{
		TemplateID:      suite.template.ID,
		ChainID:         suite.chain.ID,
		ContractAddress: "",
		TransactionHash: "",
		Status:          string(models.TransactionStatusPending),
		TemplateValues: models.JSON{
			"name":   "TestToken",
			"symbol": "TEST",
			"supply": 1000000,
		},
	}

	err := suite.deploymentService.CreateDeployment(deployment)
	suite.Require().NoError(err)

	// Create transaction session using the service
	req := services.CreateTransactionSessionRequest{
		Metadata: []models.TransactionMetadata{
			{Key: "deployment_id", Value: fmt.Sprintf("%d", deployment.ID)},
			{Key: "template_id", Value: fmt.Sprintf("%d", suite.template.ID)},
		},
		TransactionDeployments: []models.TransactionDeployment{
			{
				Title:       "Deploy SimpleToken",
				Description: "Deploy a simple token contract",
				Data:        "0x608060405234801561001057600080fd5b50",
				Value:       "0",
				Receiver:    "0x0000000000000000000000000000000000000000",
				Status:      models.TransactionStatusPending,
			},
		},
		ChainType: models.TransactionChainTypeEthereum,
		ChainID:   suite.chain.ID,
	}

	txService := services.NewTransactionService(suite.db.GetDB())
	sessionID, err := txService.CreateTransactionSession(req)
	suite.Require().NoError(err)

	return sessionID
}

func (suite *TxHandlerTestSuite) deployTestContract() (common.Hash, common.Address, error) {
	// Verify Ethereum connection
	err := suite.verifyEthereumConnection()
	if err != nil {
		return common.Hash{}, common.Address{}, err
	}

	// For testing purposes, we'll use a simple RPC call to deploy a basic contract
	rpcClient := utils.NewRPCClient(TESTNET_RPC)

	// Simple contract bytecode (empty contract that just stores constructor parameters)
	// This is a minimal test contract that will deploy successfully
	deploymentBytecode := "0x608060405234801561001057600080fd5b50336000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555061003e8061005d6000396000f3fe6080604052600080fdfea26469706673582212205c6b0e6b8f8e1a4c5e8e9f4d8f3b6c4e8f9f1c8e5f4c8e6f4c8e6f4c8e6f4c8e64736f6c63430008130033"

	// Use RPC client to send transaction
	params := []interface{}{
		map[string]interface{}{
			"from":     "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", // First anvil account
			"data":     deploymentBytecode,
			"gas":      "0x7A120",    // 500000 in hex
			"gasPrice": "0x3B9ACA00", // 1 gwei in hex
			"value":    "0x0",
		},
	}

	response, err := rpcClient.Call("eth_sendTransaction", params)
	if err != nil {
		return common.Hash{}, common.Address{}, fmt.Errorf("failed to send deployment transaction: %w", err)
	}

	if response.Error != nil {
		return common.Hash{}, common.Address{}, fmt.Errorf("RPC error: %s", response.Error.Message)
	}

	txHashStr, ok := response.Result.(string)
	if !ok {
		return common.Hash{}, common.Address{}, fmt.Errorf("invalid transaction hash response")
	}

	txHash := common.HexToHash(txHashStr)

	// Wait for transaction to be mined
	receipt, err := suite.waitForTransaction(txHash, 30*time.Second)
	if err != nil {
		return common.Hash{}, common.Address{}, fmt.Errorf("failed to wait for transaction: %w", err)
	}

	if receipt.Status != 1 {
		return common.Hash{}, common.Address{}, fmt.Errorf("deployment transaction failed")
	}

	return txHash, receipt.ContractAddress, nil
}

func (suite *TxHandlerTestSuite) waitForTransaction(txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash.Hex())
		default:
			receipt, err := suite.ethClient.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (suite *TxHandlerTestSuite) makeRequest(method, path string, body interface{}) (*http.Response, error) {
	url := fmt.Sprintf("http://localhost:%d%s", suite.serverPort, path)

	var reqBody io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add dummy authentication token for testing
	req.Header.Set("Authorization", "Bearer test-token")

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

func (suite *TxHandlerTestSuite) TestHandleTransactionPage_Success() {
	sessionID := suite.createTestSession()

	// Test the transaction page endpoint
	resp, err := suite.makeRequest("GET", fmt.Sprintf("/tx/%s", sessionID), nil)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)
	suite.Equal("text/html; charset=utf-8", resp.Header.Get("Content-Type"))

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	suite.Require().NoError(err)

	htmlContent := string(body)

	// Verify essential elements are present
	suite.Contains(htmlContent, sessionID, "Session ID should be embedded in HTML")
	suite.Contains(htmlContent, `meta name="session-id"`, "Session ID meta tag should be present")
	suite.Contains(htmlContent, `meta name="transaction-session"`, "Transaction session meta tag should be present")
	suite.Contains(htmlContent, TESTNET_RPC, "RPC URL should be embedded")
	suite.Contains(htmlContent, TESTNET_CHAIN_ID, "Chain ID should be embedded")
	suite.Contains(htmlContent, "Deploy SimpleToken", "Contract transaction title should be embedded")
}

func (suite *TxHandlerTestSuite) TestHandleTransactionPage_SessionNotFound() {
	nonExistentSessionID := uuid.New().String()

	// Test the transaction page endpoint with non-existent session
	resp, err := suite.makeRequest("GET", fmt.Sprintf("/tx/%s", nonExistentSessionID), nil)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)
}

func (suite *TxHandlerTestSuite) TestHandleTransactionAPI_Success() {
	// Skip if Ethereum connection is not available
	err := suite.verifyEthereumConnection()
	suite.Require().NoError(err)

	sessionID := suite.createTestSession()

	// Deploy a real contract to get a valid transaction hash
	txHash, contractAddress, err := suite.deployTestContract()
	suite.Require().NoError(err)

	// Generate a consistent signing message and create signature
	// The frontend signs the hex-encoded message but sends the original message to backend
	originalMessage := "I am signing into Launchpad at 1234567890"
	signature, err := utils.PersonalSignFromHex(originalMessage, TESTING_PK_1)
	suite.Require().NoError(err)
	signedMessage := originalMessage // Backend receives the original message

	// Test the transaction API endpoint
	contractAddrStr := contractAddress.Hex()
	requestBody := TransactionCompleteRequest{
		TransactionHash: txHash.Hex(),
		Status:          models.TransactionStatusConfirmed,
		ContractAddress: &contractAddrStr,
		SignedMessage:   signedMessage,
		Signature:       signature,
	}

	resp, err := suite.makeRequest("POST", fmt.Sprintf("/api/tx/%s/transaction/0", sessionID), requestBody)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)
	suite.Equal("application/json", resp.Header.Get("Content-Type"))

	// Parse response
	var responseData TransactionCompleteRequest
	err = json.NewDecoder(resp.Body).Decode(&responseData)
	suite.Require().NoError(err)

	suite.Equal(txHash.Hex(), responseData.TransactionHash)
	suite.Equal(models.TransactionStatusConfirmed, responseData.Status)
	suite.Equal(contractAddress.Hex(), *responseData.ContractAddress)

	// Verify database was updated
	txService := services.NewTransactionService(suite.db.GetDB())
	session, err := txService.GetTransactionSession(sessionID)
	suite.Require().NoError(err)
	suite.Equal(models.TransactionStatusConfirmed, session.TransactionStatus)
	suite.Equal(models.TransactionStatusConfirmed, session.TransactionDeployments[0].Status)
}

func (suite *TxHandlerTestSuite) TestHandleTransactionAPI_InvalidTransactionHash() {
	// Skip if Ethereum connection is not available
	err := suite.verifyEthereumConnection()
	suite.Require().NoError(err)

	sessionID := suite.createTestSession()

	// Use an invalid transaction hash
	invalidTxHash := "0x1234567890123456789012345678901234567890123456789012345678901234"

	// Generate signing message and create signature (even though tx is invalid)
	originalMessage := "I am signing into Launchpad at 1234567890"
	hexEncodedMessage := "0x" + hex.EncodeToString([]byte(originalMessage))
	signature, err := utils.PersonalSignFromHex(hexEncodedMessage, TESTING_PK_1)
	suite.Require().NoError(err)
	signedMessage := originalMessage // Backend receives the original message

	requestBody := TransactionCompleteRequest{
		TransactionHash: invalidTxHash,
		Status:          models.TransactionStatusConfirmed,
		SignedMessage:   signedMessage,
		Signature:       signature,
	}

	resp, err := suite.makeRequest("POST", fmt.Sprintf("/api/tx/%s/transaction/0", sessionID), requestBody)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusInternalServerError, resp.StatusCode)

	// Parse error response
	var errorResponse fiber.Map
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	suite.Require().NoError(err)

	suite.Contains(errorResponse["error"], "Failed to verify transaction")

	// Verify session status was updated to failed
	txService := services.NewTransactionService(suite.db.GetDB())
	session, err := txService.GetTransactionSession(sessionID)
	suite.Require().NoError(err)
	suite.Equal(models.TransactionStatusFailed, session.TransactionStatus)
}

func (suite *TxHandlerTestSuite) TestHandleTransactionAPI_SessionNotFound() {
	nonExistentSessionID := uuid.New().String()

	// Generate a consistent signing message and create signature
	// The frontend signs the hex-encoded message but sends the original message to backend
	originalMessage := "I am signing into Launchpad at 1234567890"
	hexEncodedMessage := "0x" + hex.EncodeToString([]byte(originalMessage))
	signature, err := utils.PersonalSignFromHex(hexEncodedMessage, TESTING_PK_1)
	suite.Require().NoError(err)
	signedMessage := originalMessage // Backend receives the original message

	requestBody := TransactionCompleteRequest{
		TransactionHash: "0x1234567890123456789012345678901234567890123456789012345678901234",
		Status:          models.TransactionStatusConfirmed,
		SignedMessage:   signedMessage,
		Signature:       signature,
	}

	resp, err := suite.makeRequest("POST", fmt.Sprintf("/api/tx/%s/transaction/0", nonExistentSessionID), requestBody)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusNotFound, resp.StatusCode)

	// Parse error response
	var errorResponse fiber.Map
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	suite.Require().NoError(err)

	suite.Contains(errorResponse["error"], "Session not found")
}

func (suite *TxHandlerTestSuite) TestHandleTransactionAPI_InvalidRequestBody() {
	sessionID := suite.createTestSession()

	// Send invalid JSON
	invalidJson := `{"invalid": "json", "missing_required_fields": true`

	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d/api/tx/%s/transaction/0", suite.serverPort, sessionID), strings.NewReader(invalidJson))
	suite.Require().NoError(err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	// Parse error response
	var errorResponse fiber.Map
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	suite.Require().NoError(err)

	suite.Contains(errorResponse["error"], "Invalid request body")
}

func (suite *TxHandlerTestSuite) TestHandleTransactionAPI_InvalidIndex() {
	sessionID := suite.createTestSession()

	requestBody := TransactionCompleteRequest{
		TransactionHash: "0x1234567890123456789012345678901234567890123456789012345678901234",
		Status:          models.TransactionStatusConfirmed,
	}

	// Use invalid index (non-numeric)
	resp, err := suite.makeRequest("POST", fmt.Sprintf("/api/tx/%s/transaction/invalid", sessionID), requestBody)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)

	// Parse error response
	var errorResponse fiber.Map
	err = json.NewDecoder(resp.Body).Decode(&errorResponse)
	suite.Require().NoError(err)

	suite.Contains(errorResponse["error"], "Invalid index")
}

func (suite *TxHandlerTestSuite) TestHandleTransactionAPI_PartialConfirmation() {
	// Skip if Ethereum connection is not available
	err := suite.verifyEthereumConnection()
	suite.Require().NoError(err)

	// Create a session with multiple deployments
	deployment := &models.Deployment{
		TemplateID:      suite.template.ID,
		ChainID:         suite.chain.ID,
		ContractAddress: "",
		TransactionHash: "",
		Status:          string(models.TransactionStatusPending),
		TemplateValues: models.JSON{
			"name":   "TestToken",
			"symbol": "TEST",
			"supply": 1000000,
		},
	}

	err = suite.deploymentService.CreateDeployment(deployment)
	suite.Require().NoError(err)

	// Create session with multiple transaction deployments
	req := services.CreateTransactionSessionRequest{
		Metadata: []models.TransactionMetadata{
			{Key: "deployment_id", Value: fmt.Sprintf("%d", deployment.ID)},
		},
		TransactionDeployments: []models.TransactionDeployment{
			{
				Title:       "Deploy Contract 1",
				Description: "First contract deployment",
				Data:        "0x608060405234801561001057600080fd5b50",
				Value:       "0",
				Receiver:    "0x0000000000000000000000000000000000000000",
				Status:      models.TransactionStatusPending,
			},
			{
				Title:       "Deploy Contract 2",
				Description: "Second contract deployment",
				Data:        "0x608060405234801561001057600080fd5b50",
				Value:       "0",
				Receiver:    "0x0000000000000000000000000000000000000000",
				Status:      models.TransactionStatusPending,
			},
		},
		ChainType: models.TransactionChainTypeEthereum,
		ChainID:   suite.chain.ID,
	}

	txService := services.NewTransactionService(suite.db.GetDB())
	sessionID, err := txService.CreateTransactionSession(req)
	suite.Require().NoError(err)

	// Deploy a real contract to get a valid transaction hash
	txHash, contractAddress, err := suite.deployTestContract()
	suite.Require().NoError(err)

	// Generate a consistent signing message and create signature
	// The frontend signs the hex-encoded message but sends the original message to backend
	originalMessage := "I am signing into Launchpad at 1234567890"
	signature, err := utils.PersonalSignFromHex(originalMessage, TESTING_PK_1)
	suite.Require().NoError(err)
	signedMessage := originalMessage // Backend receives the original message

	// Confirm only the first deployment
	contractAddrStr := contractAddress.Hex()
	requestBody := TransactionCompleteRequest{
		TransactionHash: txHash.Hex(),
		Status:          models.TransactionStatusConfirmed,
		ContractAddress: &contractAddrStr,
		SignedMessage:   signedMessage,
		Signature:       signature,
	}

	resp, err := suite.makeRequest("POST", fmt.Sprintf("/api/tx/%s/transaction/0", sessionID), requestBody)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	// Verify session is not fully confirmed yet (only partial)
	updatedSession, err := txService.GetTransactionSession(sessionID)
	suite.Require().NoError(err)
	suite.Equal(models.TransactionStatusPending, updatedSession.TransactionStatus, "Session should still be pending with partial confirmation")
	suite.Equal(models.TransactionStatusConfirmed, updatedSession.TransactionDeployments[0].Status)
	suite.Equal(models.TransactionStatusPending, updatedSession.TransactionDeployments[1].Status)

	// Confirm the second deployment
	txHash2, contractAddress2, err := suite.deployTestContract()
	suite.Require().NoError(err)

	// Generate new signing message and signature for second transaction
	originalMessage2 := "I am signing into Launchpad at 1234567890" // Use same consistent message
	signature2, err := utils.PersonalSignFromHex(originalMessage2, TESTING_PK_1)
	suite.Require().NoError(err)
	signedMessage2 := originalMessage2 // Backend receives the original message

	contractAddr2Str := contractAddress2.Hex()
	requestBody2 := TransactionCompleteRequest{
		TransactionHash: txHash2.Hex(),
		Status:          models.TransactionStatusConfirmed,
		ContractAddress: &contractAddr2Str,
		SignedMessage:   signedMessage2,
		Signature:       signature2,
	}

	resp2, err := suite.makeRequest("POST", fmt.Sprintf("/api/tx/%s/transaction/1", sessionID), requestBody2)
	suite.Require().NoError(err)
	defer resp2.Body.Close()

	suite.Equal(http.StatusOK, resp2.StatusCode)

	// Now verify session is fully confirmed
	finalSession, err := txService.GetTransactionSession(sessionID)
	suite.Require().NoError(err)
	suite.Equal(models.TransactionStatusConfirmed, finalSession.TransactionStatus, "Session should be fully confirmed")
	suite.Equal(models.TransactionStatusConfirmed, finalSession.TransactionDeployments[0].Status)
	suite.Equal(models.TransactionStatusConfirmed, finalSession.TransactionDeployments[1].Status)
}

func TestTxHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(TxHandlerTestSuite))
}
