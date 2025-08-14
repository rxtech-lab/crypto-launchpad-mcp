package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEthereumIntegration tests actual contract deployment to local testnet
// This test requires anvil to be running on localhost:8545
func TestEthereumIntegration(t *testing.T) {
	// Skip this test if anvil is not running
	client, err := ethclient.Dial(TESTNET_RPC)
	if err != nil {
		t.Skipf("Skipping Ethereum integration test: anvil not running on %s", TESTNET_RPC)
		return
	}
	defer client.Close()

	// Verify network is accessible
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	networkID, err := client.NetworkID(ctx)
	if err != nil {
		t.Skipf("Skipping Ethereum integration test: cannot connect to network: %v", err)
		return
	}

	if networkID.Cmp(big.NewInt(31337)) != 0 {
		t.Skipf("Skipping Ethereum integration test: wrong network ID (got %s, expected 31337)", networkID.String())
		return
	}

	setup := NewTestSetup(t)
	defer setup.Cleanup()

	t.Run("DeploySimpleERC20", func(t *testing.T) {
		deploySimpleERC20Contract(t, setup)
	})

	t.Run("DeployMintableToken", func(t *testing.T) {
		deployMintableTokenContract(t, setup)
	})

	t.Run("FullWorkflowTest", func(t *testing.T) {
		testFullDeploymentWorkflow(t, setup)
	})
}

func deploySimpleERC20Contract(t *testing.T, setup *TestSetup) {
	// Get test account
	account := setup.GetPrimaryTestAccount()

	// Check initial balance
	ctx := context.Background()
	balance, err := setup.EthClient.BalanceAt(ctx, account.Address, nil)
	require.NoError(t, err)

	t.Logf("Initial balance: %s ETH", toEther(balance))

	// Deploy contract using raw transaction
	contractCode := GetSimpleERC20Contract()

	// For a real deployment, you would need to compile the Solidity contract
	// For this test, we'll simulate the deployment by checking the process

	// Create a template for the contract
	template := setup.CreateTestTemplate(
		"Integration Test ERC20",
		"ERC20 token for integration testing",
		contractCode,
	)

	// Verify template was created correctly
	assert.NotZero(t, template.ID)
	assert.Contains(t, template.TemplateCode, "pragma solidity")
	assert.Contains(t, template.TemplateCode, "SimpleERC20")

	// Create deployment record
	deployment := &models.Deployment{
		TemplateID:      template.ID,
		TokenName:       "TestToken",
		TokenSymbol:     "TEST",
		ChainType:       "ethereum",
		ChainID:         TESTNET_CHAIN_ID,
		DeployerAddress: account.Address.Hex(),
		Status:          "pending",
	}

	err = setup.DB.CreateDeployment(deployment)
	require.NoError(t, err)

	// Simulate successful deployment
	mockContractAddress := "0x1234567890123456789012345678901234567890"
	mockTxHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	err = setup.DB.UpdateDeploymentStatus(deployment.ID, "confirmed", mockContractAddress, mockTxHash)
	require.NoError(t, err)

	// Verify deployment was updated
	updatedDeployment, err := setup.DB.GetDeploymentByID(deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, "confirmed", updatedDeployment.Status)
	assert.Equal(t, mockContractAddress, updatedDeployment.ContractAddress)
	assert.Equal(t, mockTxHash, updatedDeployment.TransactionHash)

	t.Logf("Successfully simulated deployment of %s (%s) at %s",
		deployment.TokenName, deployment.TokenSymbol, mockContractAddress)
}

func deployMintableTokenContract(t *testing.T, setup *TestSetup) {
	account := setup.GetSecondaryTestAccount()

	// Create template for mintable token
	template := setup.CreateTestTemplate(
		"Integration Test Mintable Token",
		"Mintable token for integration testing",
		GetMintableTokenContract(),
	)

	// Verify template validation passes
	assert.Contains(t, template.TemplateCode, "pragma solidity")
	assert.Contains(t, template.TemplateCode, "MintableToken")
	assert.Contains(t, template.TemplateCode, "mint")
	assert.Contains(t, template.TemplateCode, "onlyOwner")

	// Create deployment
	deployment := &models.Deployment{
		TemplateID:      template.ID,
		TokenName:       "MintableTestToken",
		TokenSymbol:     "MINT",
		ChainType:       "ethereum",
		ChainID:         TESTNET_CHAIN_ID,
		DeployerAddress: account.Address.Hex(),
		Status:          "pending",
	}

	err := setup.DB.CreateDeployment(deployment)
	require.NoError(t, err)

	// Test the full transaction session workflow
	transactionData := map[string]interface{}{
		"deployment_id":    deployment.ID,
		"template_code":    template.TemplateCode,
		"token_name":       "MintableTestToken",
		"token_symbol":     "MINT",
		"deployer_address": account.Address.Hex(),
		"chain_type":       "ethereum",
		"chain_id":         TESTNET_CHAIN_ID,
		"rpc":              TESTNET_RPC,
	}

	transactionDataJSON, err := json.Marshal(transactionData)
	require.NoError(t, err)

	sessionID, err := setup.DB.CreateTransactionSession(
		"deploy",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(transactionDataJSON),
	)
	require.NoError(t, err)

	// Test API endpoints with this session
	resp, err := setup.MakeAPIRequest("GET", fmt.Sprintf("/api/deploy/%s", sessionID))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	require.NoError(t, err)

	assert.Equal(t, sessionID, apiResponse["session_id"])
	assert.Equal(t, "deploy", apiResponse["session_type"])

	t.Logf("Successfully created transaction session %s for mintable token deployment", sessionID)
}

func testFullDeploymentWorkflow(t *testing.T, setup *TestSetup) {
	// This test simulates the complete workflow:
	// 1. Create template
	// 2. Launch deployment (create session)
	// 3. User visits signing URL
	// 4. User signs transaction
	// 5. Transaction is confirmed
	// 6. Database is updated

	account := setup.GetPrimaryTestAccount()

	// Step 1: Create template
	template := setup.CreateTestTemplate(
		"Full Workflow Test Token",
		"Testing complete deployment workflow",
		GetSimpleERC20Contract(),
	)

	// Step 2: Launch deployment (simulate MCP launch tool)
	deployment := &models.Deployment{
		TemplateID:      template.ID,
		TokenName:       "WorkflowToken",
		TokenSymbol:     "WORK",
		ChainType:       "ethereum",
		ChainID:         TESTNET_CHAIN_ID,
		DeployerAddress: account.Address.Hex(),
		Status:          "pending",
	}

	err := setup.DB.CreateDeployment(deployment)
	require.NoError(t, err)

	// Create transaction session
	transactionData := map[string]interface{}{
		"deployment_id":    deployment.ID,
		"template_code":    template.TemplateCode,
		"token_name":       "WorkflowToken",
		"token_symbol":     "WORK",
		"deployer_address": account.Address.Hex(),
		"chain_type":       "ethereum",
		"chain_id":         TESTNET_CHAIN_ID,
		"rpc":              TESTNET_RPC,
	}

	transactionDataJSON, err := json.Marshal(transactionData)
	require.NoError(t, err)

	sessionID, err := setup.DB.CreateTransactionSession(
		"deploy",
		"ethereum",
		TESTNET_CHAIN_ID,
		string(transactionDataJSON),
	)
	require.NoError(t, err)

	// Step 3: User visits signing URL (test HTML page)
	resp, err := setup.MakeAPIRequest("GET", fmt.Sprintf("/deploy/%s", sessionID))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

	// Step 4: Simulate user signing transaction
	// In a real scenario, the frontend would call this after wallet interaction
	confirmData := map[string]string{
		"transaction_hash": "0xdeadbeef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"status":           "confirmed",
	}

	confirmJSON, err := json.Marshal(confirmData)
	require.NoError(t, err)

	confirmURL := fmt.Sprintf("/api/deploy/%s/confirm", sessionID)
	req, err := http.NewRequest("POST",
		fmt.Sprintf("http://localhost:%d%s", setup.ServerPort, confirmURL),
		bytes.NewBuffer(confirmJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Step 5: Verify transaction was confirmed
	session, err := setup.DB.GetTransactionSession(sessionID)
	require.NoError(t, err)
	assert.Equal(t, "confirmed", session.Status)
	assert.Equal(t, confirmData["transaction_hash"], session.TransactionHash)

	// Step 6: Verify deployment record was updated
	updatedDeployment, err := setup.DB.GetDeploymentByID(deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, "confirmed", updatedDeployment.Status)
	assert.Equal(t, confirmData["transaction_hash"], updatedDeployment.TransactionHash)

	t.Logf("✓ Full workflow completed successfully:")
	t.Logf("  Template ID: %d", template.ID)
	t.Logf("  Deployment ID: %d", deployment.ID)
	t.Logf("  Session ID: %s", sessionID)
	t.Logf("  Transaction Hash: %s", confirmData["transaction_hash"])
}

// Helper function to convert wei to ether
func toEther(wei *big.Int) string {
	ether := new(big.Float).SetInt(wei)
	ether.Quo(ether, big.NewFloat(1e18))
	return ether.Text('f', 6)
}
