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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIServer_TemplateWorkflow(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	// Verify Ethereum connection first
	err := setup.VerifyEthereumConnection()
	require.NoError(t, err, "Ethereum testnet should be running on localhost:8545")

	// Test server health
	setup.AssertServerHealth()

	// 1. Create a template
	t.Run("CreateTemplate", func(t *testing.T) {
		template := setup.CreateTestTemplate(
			"Test ERC20 Token",
			"A simple ERC20 token for testing",
			GetSimpleERC20Contract(),
		)

		assert.NotZero(t, template.ID)
		assert.Equal(t, "Test ERC20 Token", template.Name)
		assert.Equal(t, "ethereum", template.ChainType)
		assert.Contains(t, template.TemplateCode, "pragma solidity")
	})

	// 2. Launch deployment session
	t.Run("LaunchDeployment", func(t *testing.T) {
		// First create a template
		template := setup.CreateTestTemplate(
			"Deployment Test Token",
			"Token for testing deployment",
			GetSimpleERC20Contract(),
		)

		// Test MCP launch tool functionality via database operations
		// Since we can't directly call MCP tools in tests, we'll simulate the workflow

		// Create deployment record directly
		deployment := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         setup.GetTestChainID(),
			TokenName:       "TestToken",
			TokenSymbol:     "TEST",
			DeployerAddress: setup.GetPrimaryTestAccount().Address.Hex(),
			Status:          "pending",
		}

		err := setup.DB.CreateDeployment(deployment)
		require.NoError(t, err)

		// Create transaction session with minimal session data
		sessionData := map[string]interface{}{
			"deployment_id": deployment.ID,
		}

		sessionDataJSON, err := json.Marshal(sessionData)
		require.NoError(t, err)

		sessionID, err := setup.DB.CreateTransactionSession(
			"deploy",
			"ethereum",
			TESTNET_CHAIN_ID,
			string(sessionDataJSON),
		)
		require.NoError(t, err)

		// Test deployment page endpoint
		deployURL := fmt.Sprintf("/deploy/%s", sessionID)
		resp, err := setup.MakeAPIRequest("GET", deployURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

		// Read and verify HTML content
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		htmlContent := string(body)
		assert.Contains(t, htmlContent, "Deploy Contract")
		assert.Contains(t, htmlContent, sessionID)
	})

	// 3. Test deployment API endpoint
	t.Run("DeploymentAPI", func(t *testing.T) {
		// Create template and session
		template := setup.CreateTestTemplate(
			"API Test Token",
			"Token for testing API endpoints",
			GetMintableTokenContract(),
		)

		// Create deployment record with APIToken/API values
		deployment := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         setup.GetTestChainID(),
			TokenName:       "APIToken",
			TokenSymbol:     "API",
			DeployerAddress: setup.GetPrimaryTestAccount().Address.Hex(),
			Status:          "pending",
		}

		err := setup.DB.CreateDeployment(deployment)
		require.NoError(t, err)

		// Create minimal session data (transaction data will be generated on-demand)
		sessionData := map[string]interface{}{
			"deployment_id": deployment.ID,
		}

		sessionDataJSON, err := json.Marshal(sessionData)
		require.NoError(t, err)

		sessionID, err := setup.DB.CreateTransactionSession(
			"deploy",
			"ethereum",
			TESTNET_CHAIN_ID,
			string(sessionDataJSON),
		)
		require.NoError(t, err)

		// Test API endpoint
		apiURL := fmt.Sprintf("/api/deploy/%s", sessionID)
		resp, err := setup.MakeAPIRequest("GET", apiURL)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		// Parse response
		var apiResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&apiResponse)
		require.NoError(t, err)

		assert.Equal(t, sessionID, apiResponse["session_id"])
		assert.Equal(t, "deploy", apiResponse["session_type"])
		assert.Equal(t, "ethereum", apiResponse["chain_type"])
		assert.Equal(t, TESTNET_CHAIN_ID, apiResponse["chain_id"])
		assert.Equal(t, "pending", apiResponse["status"])

		// Verify transaction data
		txData, ok := apiResponse["transaction_data"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "APIToken", txData["token_name"])
		assert.Equal(t, "API", txData["token_symbol"])
	})

	// 4. Test transaction confirmation
	t.Run("TransactionConfirmation", func(t *testing.T) {
		// Skip this test if anvil is not running
		err := setup.VerifyEthereumConnection()
		if err != nil {
			t.Skipf("Skipping test: %v", err)
			return
		}

		// Deploy a real contract to get a real transaction hash
		account := setup.GetPrimaryTestAccount()
		result, err := setup.DeployContract(
			account,
			GetSimpleERC20Contract(),
			"SimpleERC20",
			"APITestToken",     // name
			"API",              // symbol
			big.NewInt(100000), // totalSupply
		)
		require.NoError(t, err)

		// Create a session for confirmation testing
		sessionID, err := setup.DB.CreateTransactionSession(
			"deploy",
			"ethereum",
			TESTNET_CHAIN_ID,
			`{"deployment_id": 1, "token_name": "ConfirmTest", "token_symbol": "CONF"}`,
		)
		require.NoError(t, err)

		// Test successful confirmation with real transaction hash
		confirmURL := fmt.Sprintf("/api/deploy/%s/confirm", sessionID)
		confirmData := map[string]any{
			"transaction_hash": result.TransactionHash.Hex(),
			"contract_address": result.ContractAddress.Hex(),
			"status":           models.TransactionStatusConfirmed,
		}

		confirmJSON, err := json.Marshal(confirmData)
		require.NoError(t, err)

		req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d%s", setup.ServerPort, confirmURL), bytes.NewBuffer(confirmJSON))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]string
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		assert.Equal(t, string(models.TransactionStatusConfirmed), response["status"])

		// Verify session was updated
		session, err := setup.DB.GetTransactionSession(sessionID)
		require.NoError(t, err)
		assert.Equal(t, models.TransactionStatusConfirmed, session.Status)
		assert.Equal(t, confirmData["transaction_hash"], session.TransactionHash)

		t.Logf("✓ Successfully models.TransactionStatusConfirmed transaction %s", result.TransactionHash.Hex())
	})
}

func TestAPIServer_ErrorHandling(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	// Test invalid session ID
	t.Run("InvalidSessionID", func(t *testing.T) {
		resp, err := setup.MakeAPIRequest("GET", "/deploy/invalid-session-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test expired session
	t.Run("ExpiredSession", func(t *testing.T) {
		// Create a session with past expiry
		sessionID, err := setup.DB.CreateTransactionSession(
			"deploy",
			"ethereum",
			TESTNET_CHAIN_ID,
			`{"test": "data"}`,
		)
		require.NoError(t, err)

		// Manually update the session to be expired
		err = setup.DB.DB.Model(&models.TransactionSession{}).
			Where("id = ?", sessionID).
			Update("expires_at", time.Now().Add(-1*time.Hour)).Error
		require.NoError(t, err)

		resp, err := setup.MakeAPIRequest("GET", fmt.Sprintf("/deploy/%s", sessionID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Test wrong session type
	t.Run("WrongSessionType", func(t *testing.T) {
		// Create a swap session
		sessionID, err := setup.DB.CreateTransactionSession(
			"swap",
			"ethereum",
			TESTNET_CHAIN_ID,
			`{"test": "data"}`,
		)
		require.NoError(t, err)

		// Try to access it as a deployment session
		resp, err := setup.MakeAPIRequest("GET", fmt.Sprintf("/deploy/%s", sessionID))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Test malformed JSON in confirmation
	t.Run("MalformedConfirmationData", func(t *testing.T) {
		sessionID, err := setup.DB.CreateTransactionSession(
			"deploy",
			"ethereum",
			TESTNET_CHAIN_ID,
			`{"test": "data"}`,
		)
		require.NoError(t, err)

		confirmURL := fmt.Sprintf("/api/deploy/%s/confirm", sessionID)

		req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost:%d%s", setup.ServerPort, confirmURL), bytes.NewBuffer([]byte("invalid json")))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestAPIServer_SessionManagement(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	// Test session lifecycle
	t.Run("SessionLifecycle", func(t *testing.T) {
		// Create session
		sessionID, err := setup.DB.CreateTransactionSession(
			"deploy",
			"ethereum",
			TESTNET_CHAIN_ID,
			`{"deployment_id": 1, "token_name": "LifecycleTest", "token_symbol": "LIFE"}`,
		)
		require.NoError(t, err)

		// Test initial session state
		session, err := setup.DB.GetTransactionSession(sessionID)
		require.NoError(t, err)
		assert.Equal(t, models.TransactionStatusPending, session.Status)
		assert.Equal(t, "deploy", session.SessionType)
		assert.True(t, time.Now().Before(session.ExpiresAt))

		// Test status updates
		err = setup.DB.UpdateTransactionSessionStatus(sessionID, models.TransactionStatusConfirmed, "")
		require.NoError(t, err)

		session, err = setup.DB.GetTransactionSession(sessionID)
		require.NoError(t, err)
		assert.Equal(t, models.TransactionStatusConfirmed, session.Status)

		// Test final confirmation
		txHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
		err = setup.DB.UpdateTransactionSessionStatus(sessionID, models.TransactionStatusConfirmed, txHash)
		require.NoError(t, err)

		session, err = setup.DB.GetTransactionSession(sessionID)
		require.NoError(t, err)
		assert.Equal(t, models.TransactionStatusConfirmed, session.Status)
		assert.Equal(t, txHash, session.TransactionHash)
	})

	// Test multiple concurrent sessions
	t.Run("ConcurrentSessions", func(t *testing.T) {
		var sessionIDs []string

		// Create multiple sessions
		for i := 0; i < 5; i++ {
			sessionData := fmt.Sprintf(`{"deployment_id": %d, "token_name": "Concurrent%d", "token_symbol": "CONC%d"}`, i+1, i+1, i+1)
			sessionID, err := setup.DB.CreateTransactionSession(
				"deploy",
				"ethereum",
				TESTNET_CHAIN_ID,
				sessionData,
			)
			require.NoError(t, err)
			sessionIDs = append(sessionIDs, sessionID)
		}

		// Verify all sessions exist and are independent
		for i, sessionID := range sessionIDs {
			session, err := setup.DB.GetTransactionSession(sessionID)
			require.NoError(t, err)
			assert.Equal(t, models.TransactionStatusPending, session.Status)

			var sessionData map[string]interface{}
			err = json.Unmarshal([]byte(session.TransactionData), &sessionData)
			require.NoError(t, err)

			expectedTokenName := fmt.Sprintf("Concurrent%d", i+1)
			assert.Equal(t, expectedTokenName, sessionData["token_name"])
		}

		// Update one session and verify others are unaffected
		err := setup.DB.UpdateTransactionSessionStatus(sessionIDs[0], models.TransactionStatusConfirmed, "0x123")
		require.NoError(t, err)

		// Check first session is updated
		session, err := setup.DB.GetTransactionSession(sessionIDs[0])
		require.NoError(t, err)
		assert.Equal(t, models.TransactionStatusConfirmed, session.Status)

		// Check others are still pending
		for _, sessionID := range sessionIDs[1:] {
			session, err := setup.DB.GetTransactionSession(sessionID)
			require.NoError(t, err)
			assert.Equal(t, models.TransactionStatusPending, session.Status)
		}
	})
}

func TestAPIServer_DatabaseIntegration(t *testing.T) {
	setup := NewTestSetup(t)
	defer setup.Cleanup()

	// Test template operations
	t.Run("TemplateOperations", func(t *testing.T) {
		// Create template
		template := setup.CreateTestTemplate(
			"Database Test Token",
			"Testing database integration",
			GetSimpleERC20Contract(),
		)

		// Retrieve template
		retrieved, err := setup.DB.GetTemplateByID(template.ID)
		require.NoError(t, err)
		assert.Equal(t, template.Name, retrieved.Name)
		assert.Equal(t, template.ChainType, retrieved.ChainType)

		// List templates
		templates, err := setup.DB.ListTemplates("ethereum", "", 10)
		require.NoError(t, err)
		assert.Greater(t, len(templates), 0)

		// Verify our template is in the list
		found := false
		for _, tmpl := range templates {
			if tmpl.ID == template.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Created template should be in the list")
	})

	// Test deployment operations
	t.Run("DeploymentOperations", func(t *testing.T) {
		// Create template first
		template := setup.CreateTestTemplate(
			"Deployment DB Test",
			"Testing deployment database operations",
			GetMintableTokenContract(),
		)

		// Create deployment
		deployment := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         setup.GetTestChainID(),
			TokenName:       "DBTestToken",
			TokenSymbol:     "DBT",
			DeployerAddress: setup.GetPrimaryTestAccount().Address.Hex(),
			Status:          "pending",
		}

		err := setup.DB.CreateDeployment(deployment)
		require.NoError(t, err)
		assert.NotZero(t, deployment.ID)

		// Retrieve deployment
		retrieved, err := setup.DB.GetDeploymentByID(deployment.ID)
		require.NoError(t, err)
		assert.Equal(t, deployment.TokenName, retrieved.TokenName)
		assert.Equal(t, deployment.TokenSymbol, retrieved.TokenSymbol)
		assert.Equal(t, template.Name, retrieved.Template.Name)

		// Update deployment status
		contractAddress := "0x1234567890123456789012345678901234567890"
		txHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

		err = setup.DB.UpdateDeploymentStatus(deployment.ID, "confirmed", contractAddress, txHash)
		require.NoError(t, err)

		// Verify update
		updated, err := setup.DB.GetDeploymentByID(deployment.ID)
		require.NoError(t, err)
		assert.Equal(t, "confirmed", updated.Status)
		assert.Equal(t, contractAddress, updated.ContractAddress)
		assert.Equal(t, txHash, updated.TransactionHash)
	})
}
