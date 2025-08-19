package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
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

	// Deploy contract using real transaction
	contractCode := GetSimpleERC20Contract()

	// Compile the Solidity contract
	compilationResult, err := utils.CompileSolidity("0.8.19", contractCode)
	require.NoError(t, err)
	require.Contains(t, compilationResult.Bytecode, "SimpleERC20")

	// Get bytecode and ABI for deployment
	bytecodeHex := compilationResult.Bytecode["SimpleERC20"]
	bytecode, err := hex.DecodeString(strings.TrimPrefix(bytecodeHex, "0x"))
	require.NoError(t, err)

	// Parse ABI
	abiBytes, err := json.Marshal(compilationResult.Abi["SimpleERC20"])
	require.NoError(t, err)
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	require.NoError(t, err)

	// Pack constructor arguments (name, symbol, totalSupply)
	constructorData, err := parsedABI.Pack("", "TestToken", "TEST", big.NewInt(1000000))
	require.NoError(t, err)

	// Combine bytecode with constructor arguments
	fullBytecode := append(bytecode, constructorData...)

	// Create deployment transaction
	nonce, err := setup.EthClient.PendingNonceAt(ctx, account.Address)
	require.NoError(t, err)

	gasPrice, err := setup.EthClient.SuggestGasPrice(ctx)
	require.NoError(t, err)

	// Estimate gas for deployment
	gasLimit := uint64(3000000) // Sufficient for ERC20 deployment

	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, fullBytecode)

	// Sign and send transaction
	chainID := big.NewInt(31337)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), account.PrivateKey)
	require.NoError(t, err)

	err = setup.EthClient.SendTransaction(ctx, signedTx)
	require.NoError(t, err)

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(ctx, setup.EthClient, signedTx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.Status, "Transaction failed")

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
		ChainID:         setup.GetTestChainID(),
		TokenName:       "TestToken",
		TokenSymbol:     "TEST",
		DeployerAddress: account.Address.Hex(),
		Status:          "pending",
	}

	err = setup.DB.CreateDeployment(deployment)
	require.NoError(t, err)

	// Update with real deployment data
	realContractAddress := receipt.ContractAddress.Hex()
	realTxHash := signedTx.Hash().Hex()

	err = setup.DB.UpdateDeploymentStatus(deployment.ID, "confirmed", realContractAddress, realTxHash)
	require.NoError(t, err)

	// Verify deployment was updated
	updatedDeployment, err := setup.DB.GetDeploymentByID(deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, "confirmed", updatedDeployment.Status)
	assert.Equal(t, realContractAddress, updatedDeployment.ContractAddress)
	assert.Equal(t, realTxHash, updatedDeployment.TransactionHash)

	// Verify contract deployment by reading contract state
	contractInstance := bind.NewBoundContract(receipt.ContractAddress, parsedABI, setup.EthClient, setup.EthClient, setup.EthClient)

	// Call name() function to verify deployment
	var name string
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&name}, "name")
	require.NoError(t, err)
	assert.Equal(t, "TestToken", name)

	// Call symbol() function
	var symbol string
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&symbol}, "symbol")
	require.NoError(t, err)
	assert.Equal(t, "TEST", symbol)

	t.Logf("✓ Successfully deployed %s (%s) at %s",
		updatedDeployment.TokenName, updatedDeployment.TokenSymbol, realContractAddress)
	t.Logf("✓ Transaction hash: %s", realTxHash)
	t.Logf("✓ Gas used: %d", receipt.GasUsed)
}

func deployMintableTokenContract(t *testing.T, setup *TestSetup) {
	account := setup.GetSecondaryTestAccount()
	ctx := context.Background()

	// Deploy contract using real transaction
	contractCode := GetMintableTokenContract()

	// Compile the Solidity contract
	compilationResult, err := utils.CompileSolidity("0.8.19", contractCode)
	require.NoError(t, err)
	require.Contains(t, compilationResult.Bytecode, "MintableToken")

	// Get bytecode and ABI for deployment
	bytecodeHex := compilationResult.Bytecode["MintableToken"]
	bytecode, err := hex.DecodeString(strings.TrimPrefix(bytecodeHex, "0x"))
	require.NoError(t, err)

	// Parse ABI
	abiBytes, err := json.Marshal(compilationResult.Abi["MintableToken"])
	require.NoError(t, err)
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	require.NoError(t, err)

	// Pack constructor arguments (name, symbol, initialSupply)
	constructorData, err := parsedABI.Pack("", "MintableTestToken", "MINT", big.NewInt(500000))
	require.NoError(t, err)

	// Combine bytecode with constructor arguments
	fullBytecode := append(bytecode, constructorData...)

	// Create deployment transaction
	nonce, err := setup.EthClient.PendingNonceAt(ctx, account.Address)
	require.NoError(t, err)

	gasPrice, err := setup.EthClient.SuggestGasPrice(ctx)
	require.NoError(t, err)

	// Estimate gas for deployment
	gasLimit := uint64(3500000) // Sufficient for mintable token deployment

	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, fullBytecode)

	// Sign and send transaction
	chainID := big.NewInt(31337)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), account.PrivateKey)
	require.NoError(t, err)

	err = setup.EthClient.SendTransaction(ctx, signedTx)
	require.NoError(t, err)

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(ctx, setup.EthClient, signedTx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.Status, "Transaction failed")

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
		ChainID:         setup.GetTestChainID(),
		TokenName:       "MintableTestToken",
		TokenSymbol:     "MINT",
		DeployerAddress: account.Address.Hex(),
		Status:          "pending",
	}

	err = setup.DB.CreateDeployment(deployment)
	require.NoError(t, err)

	// Update with real deployment data
	realContractAddress := receipt.ContractAddress.Hex()
	realTxHash := signedTx.Hash().Hex()

	err = setup.DB.UpdateDeploymentStatus(deployment.ID, "confirmed", realContractAddress, realTxHash)
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
		"contract_address": realContractAddress,
		"transaction_hash": realTxHash,
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

	// Verify contract deployment by reading contract state
	contractInstance := bind.NewBoundContract(receipt.ContractAddress, parsedABI, setup.EthClient, setup.EthClient, setup.EthClient)

	// Call name() function to verify deployment
	var name string
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&name}, "name")
	require.NoError(t, err)
	assert.Equal(t, "MintableTestToken", name)

	// Call owner() function to verify deployer is owner
	var owner common.Address
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&owner}, "owner")
	require.NoError(t, err)
	assert.Equal(t, account.Address, owner)

	// Test minting functionality
	mintAmount := big.NewInt(1000)
	mintTx, err := contractInstance.Transact(account.Auth, "mint", account.Address, mintAmount)
	require.NoError(t, err)

	// Wait for mint transaction
	mintReceipt, err := bind.WaitMined(ctx, setup.EthClient, mintTx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), mintReceipt.Status, "Mint transaction failed")

	t.Logf("✓ Successfully deployed mintable token at %s", realContractAddress)
	t.Logf("✓ Transaction hash: %s", realTxHash)
	t.Logf("✓ Gas used for deployment: %d", receipt.GasUsed)
	t.Logf("✓ Mint transaction hash: %s", mintTx.Hash().Hex())
	t.Logf("✓ Successfully created transaction session %s", sessionID)
}

func testFullDeploymentWorkflow(t *testing.T, setup *TestSetup) {
	// This test demonstrates the complete workflow with REAL blockchain transactions:
	// 1. Create template
	// 2. Compile and deploy contract to blockchain
	// 3. Launch deployment (create session with real data)
	// 4. User visits signing URL
	// 5. Simulate transaction confirmation with real hash
	// 6. Database is updated with real data
	// 7. Verify on-chain contract state

	account := setup.GetPrimaryTestAccount()
	ctx := context.Background()

	// Step 1: Create template
	template := setup.CreateTestTemplate(
		"Full Workflow Test Token",
		"Testing complete deployment workflow",
		GetSimpleERC20Contract(),
	)

	// Step 2: Perform REAL contract deployment
	contractCode := GetSimpleERC20Contract()

	// Compile the Solidity contract
	compilationResult, err := utils.CompileSolidity("0.8.19", contractCode)
	require.NoError(t, err)
	require.Contains(t, compilationResult.Bytecode, "SimpleERC20")

	// Get bytecode and ABI for deployment
	bytecodeHex := compilationResult.Bytecode["SimpleERC20"]
	bytecode, err := hex.DecodeString(strings.TrimPrefix(bytecodeHex, "0x"))
	require.NoError(t, err)

	// Parse ABI
	abiBytes, err := json.Marshal(compilationResult.Abi["SimpleERC20"])
	require.NoError(t, err)
	parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	require.NoError(t, err)

	// Pack constructor arguments (name, symbol, totalSupply)
	constructorData, err := parsedABI.Pack("", "WorkflowToken", "WORK", big.NewInt(2000000))
	require.NoError(t, err)

	// Combine bytecode with constructor arguments
	fullBytecode := append(bytecode, constructorData...)

	// Create deployment transaction
	nonce, err := setup.EthClient.PendingNonceAt(ctx, account.Address)
	require.NoError(t, err)

	gasPrice, err := setup.EthClient.SuggestGasPrice(ctx)
	require.NoError(t, err)

	// Estimate gas for deployment
	gasLimit := uint64(3000000) // Sufficient for ERC20 deployment

	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, fullBytecode)

	// Sign and send transaction
	chainID := big.NewInt(31337)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), account.PrivateKey)
	require.NoError(t, err)

	err = setup.EthClient.SendTransaction(ctx, signedTx)
	require.NoError(t, err)

	// Wait for transaction to be mined
	receipt, err := bind.WaitMined(ctx, setup.EthClient, signedTx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), receipt.Status, "Transaction failed")

	// Real deployment data
	realContractAddress := receipt.ContractAddress.Hex()
	realTxHash := signedTx.Hash().Hex()

	// Step 3: Launch deployment (simulate MCP launch tool)
	deployment := &models.Deployment{
		TemplateID:      template.ID,
		ChainID:         setup.GetTestChainID(),
		TokenName:       "WorkflowToken",
		TokenSymbol:     "WORK",
		DeployerAddress: account.Address.Hex(),
		Status:          "pending",
	}

	err = setup.DB.CreateDeployment(deployment)
	require.NoError(t, err)

	// Create transaction session with real blockchain data
	transactionData := map[string]interface{}{
		"deployment_id":    deployment.ID,
		"template_code":    template.TemplateCode,
		"token_name":       "WorkflowToken",
		"token_symbol":     "WORK",
		"deployer_address": account.Address.Hex(),
		"chain_type":       "ethereum",
		"chain_id":         TESTNET_CHAIN_ID,
		"rpc":              TESTNET_RPC,
		"contract_address": realContractAddress,
		"transaction_hash": realTxHash,
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

	// Step 4: User visits signing URL (test HTML page)
	resp, err := setup.MakeAPIRequest("GET", fmt.Sprintf("/deploy/%s", sessionID))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))

	// Step 5: Simulate user confirming transaction with REAL hash
	// In a real scenario, the frontend would call this after wallet interaction
	confirmData := map[string]string{
		"transaction_hash": realTxHash,
		"contract_address": realContractAddress,
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

	// Step 6: Verify transaction was models.TransactionStatusConfirmed
	session, err := setup.DB.GetTransactionSession(sessionID)
	require.NoError(t, err)
	assert.Equal(t, models.TransactionStatusConfirmed, session.Status)
	assert.Equal(t, realTxHash, session.TransactionHash)

	// Update deployment record with real data
	err = setup.DB.UpdateDeploymentStatus(deployment.ID, models.TransactionStatusConfirmed, realContractAddress, realTxHash)
	require.NoError(t, err)

	// Verify deployment record was updated with real data
	updatedDeployment, err := setup.DB.GetDeploymentByID(deployment.ID)
	require.NoError(t, err)
	assert.Equal(t, string(models.TransactionStatusConfirmed), updatedDeployment.Status)
	assert.Equal(t, realContractAddress, updatedDeployment.ContractAddress)
	assert.Equal(t, realTxHash, updatedDeployment.TransactionHash)

	// Step 7: Verify on-chain contract state
	contractInstance := bind.NewBoundContract(receipt.ContractAddress, parsedABI, setup.EthClient, setup.EthClient, setup.EthClient)

	// Call name() function to verify deployment
	var name string
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&name}, "name")
	require.NoError(t, err)
	assert.Equal(t, "WorkflowToken", name)

	// Call symbol() function
	var symbol string
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&symbol}, "symbol")
	require.NoError(t, err)
	assert.Equal(t, "WORK", symbol)

	// Call totalSupply() function
	var totalSupply *big.Int
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&totalSupply}, "totalSupply")
	require.NoError(t, err)
	expectedSupply := new(big.Int).Mul(big.NewInt(2000000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	assert.Equal(t, expectedSupply, totalSupply)

	// Verify deployer has all tokens
	var deployerBalance *big.Int
	err = contractInstance.Call(&bind.CallOpts{}, &[]interface{}{&deployerBalance}, "balanceOf", account.Address)
	require.NoError(t, err)
	assert.Equal(t, expectedSupply, deployerBalance)

	t.Logf("✓ Full workflow completed successfully with REAL blockchain transactions:")
	t.Logf("  Template ID: %d", template.ID)
	t.Logf("  Deployment ID: %d", deployment.ID)
	t.Logf("  Session ID: %s", sessionID)
	t.Logf("  REAL Contract Address: %s", realContractAddress)
	t.Logf("  REAL Transaction Hash: %s", realTxHash)
	t.Logf("  Gas Used: %d", receipt.GasUsed)
	t.Logf("  Total Supply: %s tokens", totalSupply.String())
	t.Logf("  Deployer Balance: %s tokens", deployerBalance.String())
}

// Helper function to convert wei to ether
func toEther(wei *big.Int) string {
	ether := new(big.Float).SetInt(wei)
	ether.Quo(ether, big.NewFloat(1e18))
	return ether.Text('f', 6)
}
