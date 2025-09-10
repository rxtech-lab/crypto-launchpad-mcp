package services_test

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// Simple ERC20 contract for testing
	simpleERC20Contract = `
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SimpleToken {
    string public name;
    string public symbol;
    uint256 public totalSupply;
    mapping(address => uint256) public balanceOf;

    constructor(string memory _name, string memory _symbol, uint256 _totalSupply) {
        name = _name;
        symbol = _symbol;
        totalSupply = _totalSupply;
        balanceOf[msg.sender] = _totalSupply;
    }

    function transfer(address to, uint256 amount) public returns (bool) {
        require(balanceOf[msg.sender] >= amount, "Insufficient balance");
        balanceOf[msg.sender] -= amount;
        balanceOf[to] += amount;
        return true;
    }
}
`

	// Testnet configuration
	testnetRPC     = "http://localhost:8545"
	testnetChainID = 31337 // Anvil default chain ID
)

// Test helper functions
func getTestAccount() (*ecdsa.PrivateKey, common.Address, error) {
	// Anvil test account #0 private key
	privateKeyHex := "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, common.Address{}, err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, common.Address{}, fmt.Errorf("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	return privateKey, fromAddress, nil
}

func connectToTestnet() (*ethclient.Client, error) {
	return ethclient.Dial(testnetRPC)
}

func sendTransaction(client *ethclient.Client, privateKey *ecdsa.PrivateKey, txData string) (common.Hash, error) {
	ctx := context.Background()

	fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Get nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Decode transaction data
	data := common.FromHex(txData)

	// Estimate gas
	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From: fromAddress,
		Data: data,
	})
	if err != nil {
		// Use a default gas limit for contract deployment
		gasLimit = uint64(3000000)
	}

	// Create transaction
	tx := types.NewContractCreation(nonce, big.NewInt(0), gasLimit, gasPrice, data)

	// Sign transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(testnetChainID)), privateKey)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash(), nil
}

func waitForTransaction(client *ethclient.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for transaction %s", txHash.Hex())
		case <-ticker.C:
			receipt, err := client.TransactionReceipt(context.Background(), txHash)
			if err == nil && receipt != nil {
				return receipt, nil
			}
		}
	}
}

func getContractAddress(receipt *types.Receipt) common.Address {
	return receipt.ContractAddress
}

func verifyDeployedContract(t *testing.T, client *ethclient.Client, contractAddress, ownerAddress common.Address) {
	// Create ABI for the SimpleToken contract
	contractABI := `[
		{
			"inputs": [],
			"name": "name",
			"outputs": [{"internalType": "string", "name": "", "type": "string"}],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "symbol",
			"outputs": [{"internalType": "string", "name": "", "type": "string"}],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [],
			"name": "totalSupply",
			"outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
			"stateMutability": "view",
			"type": "function"
		},
		{
			"inputs": [{"internalType": "address", "name": "", "type": "address"}],
			"name": "balanceOf",
			"outputs": [{"internalType": "uint256", "name": "", "type": "uint256"}],
			"stateMutability": "view",
			"type": "function"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	require.NoError(t, err)

	// Call name() function
	nameData, err := parsedABI.Pack("name")
	require.NoError(t, err)

	result, err := client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &contractAddress,
		Data: nameData,
	}, nil)
	require.NoError(t, err)

	var name string
	err = parsedABI.UnpackIntoInterface(&name, "name", result)
	require.NoError(t, err)
	assert.Equal(t, "Test Token", name)

	// Call symbol() function
	symbolData, err := parsedABI.Pack("symbol")
	require.NoError(t, err)

	result, err = client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &contractAddress,
		Data: symbolData,
	}, nil)
	require.NoError(t, err)

	var symbol string
	err = parsedABI.UnpackIntoInterface(&symbol, "symbol", result)
	require.NoError(t, err)
	assert.Equal(t, "TEST", symbol)

	// Call totalSupply() function
	totalSupplyData, err := parsedABI.Pack("totalSupply")
	require.NoError(t, err)

	result, err = client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &contractAddress,
		Data: totalSupplyData,
	}, nil)
	require.NoError(t, err)

	var totalSupply *big.Int
	err = parsedABI.UnpackIntoInterface(&totalSupply, "totalSupply", result)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1000000), totalSupply)

	// Call balanceOf(owner) function
	balanceOfData, err := parsedABI.Pack("balanceOf", ownerAddress)
	require.NoError(t, err)

	result, err = client.CallContract(context.Background(), ethereum.CallMsg{
		To:   &contractAddress,
		Data: balanceOfData,
	}, nil)
	require.NoError(t, err)

	var balance *big.Int
	err = parsedABI.UnpackIntoInterface(&balance, "balanceOf", result)
	require.NoError(t, err)
	assert.Equal(t, big.NewInt(1000000), balance)

	t.Logf("Contract verified: name=%s, symbol=%s, totalSupply=%s, ownerBalance=%s",
		name, symbol, totalSupply.String(), balance.String())
}

// TestGetContractDeploymentTransaction tests the GetContractDeploymentTransaction method
func TestGetContractDeploymentTransactionWithContractCode(t *testing.T) {
	// Create EVM service
	evmService := services.NewEvmService()

	t.Run("Generate deployment transaction data", func(t *testing.T) {
		// Prepare arguments
		args := services.ContractDeploymentWithContractCodeTransactionArgs{
			ContractName:    "SimpleToken",
			ConstructorArgs: []any{"Test Token", "TEST", big.NewInt(1000000)},
			ContractCode:    simpleERC20Contract,
			Title:           "Deploy Test Token",
			Description:     "Deploying a test ERC20 token",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		// Generate deployment transaction
		deployment, abi, err := evmService.GetContractDeploymentTransactionWithContractCode(args)
		require.NoError(t, err)
		require.Greater(t, len(abi.Methods), 0)

		// Verify deployment data
		assert.NotEmpty(t, deployment.Data)
		assert.Equal(t, "Deploy Test Token", deployment.Title)
		assert.Equal(t, "Deploying a test ERC20 token", deployment.Description)
		assert.Equal(t, "0", deployment.Value)

		// Verify the data starts with valid bytecode (0x prefix)
		assert.True(t, strings.HasPrefix(deployment.Data, "0x"))
		assert.Greater(t, len(deployment.Data), 100) // Should be substantial bytecode
	})

	t.Run("Deploy contract to testnet", func(t *testing.T) {
		// Skip if testnet is not running
		client, err := connectToTestnet()
		if err != nil {
			t.Skipf("Testnet not running on %s (run 'make e2e-network'): %v", testnetRPC, err)
		}
		defer client.Close()

		// Get test account
		privateKey, fromAddress, err := getTestAccount()
		require.NoError(t, err)

		// Check balance
		balance, err := client.BalanceAt(context.Background(), fromAddress, nil)
		require.NoError(t, err)
		require.Greater(t, balance.Cmp(big.NewInt(0)), 0, "Test account has no balance")

		// Prepare deployment arguments
		args := services.ContractDeploymentWithContractCodeTransactionArgs{
			ContractName:    "SimpleToken",
			ConstructorArgs: []any{"Test Token", "TEST", big.NewInt(1000000)},
			ContractCode:    simpleERC20Contract,
			Title:           "Deploy Test Token",
			Description:     "Deploying a test ERC20 token to testnet",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		// Generate deployment transaction
		deployment, abi, err := evmService.GetContractDeploymentTransactionWithContractCode(args)
		require.Greater(t, len(abi.Methods), 0)
		require.NoError(t, err)

		// Send transaction to testnet
		txHash, err := sendTransaction(client, privateKey, deployment.Data)
		require.NoError(t, err)
		assert.NotEqual(t, common.Hash{}, txHash)

		t.Logf("Transaction sent: %s", txHash.Hex())

		// Wait for transaction confirmation
		receipt, err := waitForTransaction(client, txHash, 30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, receipt)

		// Verify transaction was successful
		assert.Equal(t, uint64(1), receipt.Status, "Transaction failed")

		// Get deployed contract address
		contractAddress := getContractAddress(receipt)
		assert.NotEqual(t, common.Address{}, contractAddress)

		t.Logf("Contract deployed at: %s", contractAddress.Hex())

		// Verify contract code exists at the address
		code, err := client.CodeAt(context.Background(), contractAddress, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, code, "No code at deployed contract address")

		// Interact with the deployed contract to verify it works
		verifyDeployedContract(t, client, contractAddress, fromAddress)
	})

	t.Run("Invalid contract code", func(t *testing.T) {
		args := services.ContractDeploymentWithContractCodeTransactionArgs{
			ContractName:    "InvalidContract",
			ConstructorArgs: []any{},
			ContractCode:    "invalid solidity code",
			Title:           "Invalid Deploy",
			Description:     "This should fail",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		_, _, err := evmService.GetContractDeploymentTransactionWithContractCode(args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "compilation errors")
	})

	t.Run("Missing required fields", func(t *testing.T) {
		args := services.ContractDeploymentWithContractCodeTransactionArgs{
			// Missing ContractName
			ConstructorArgs: []any{},
			ContractCode:    simpleERC20Contract,
		}

		_, _, err := evmService.GetContractDeploymentTransactionWithContractCode(args)
		assert.Error(t, err)
	})
}

// TestGetContractDeploymentTransactionWithBytecodeAndAbi tests the GetContractDeploymentTransactionWithBytecodeAndAbi method
func TestGetContractDeploymentTransactionWithBytecodeAndAbi(t *testing.T) {
	// Create EVM service
	evmService := services.NewEvmService()

	t.Run("Generate deployment with WETH9 bytecode and ABI", func(t *testing.T) {
		// Get real WETH9 contract artifact
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		// Convert ABI to JSON string
		abiBytes, err := json.Marshal(weth9Artifact.ABI)
		require.NoError(t, err)

		// Prepare arguments - WETH9 has no constructor arguments
		args := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			Abi:             string(abiBytes),
			Bytecode:        weth9Artifact.Bytecode,
			ConstructorArgs: []any{},
			Title:           "Deploy WETH9",
			Description:     "Deploying Wrapped Ether contract",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2FactoryDeployment,
		}

		// Generate deployment transaction
		deployment, abi, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(args)
		require.Greater(t, len(abi.Methods), 0)
		require.NoError(t, err)

		// Verify deployment data
		assert.NotEmpty(t, deployment.Data)
		assert.Equal(t, "Deploy WETH9", deployment.Title)
		assert.Equal(t, "Deploying Wrapped Ether contract", deployment.Description)
		assert.Equal(t, "0", deployment.Value)

		// Verify the data matches the bytecode (WETH9 has no constructor args)
		assert.Equal(t, weth9Artifact.Bytecode, deployment.Data)
	})

	t.Run("Deploy UniswapV2Factory to testnet", func(t *testing.T) {
		client, err := connectToTestnet()
		require.NoError(t, err)
		defer client.Close()

		// Get test account
		privateKey, fromAddress, err := getTestAccount()
		require.NoError(t, err)

		// Get real Factory contract artifact
		factoryArtifact, err := contracts.GetFactoryArtifact()
		require.NoError(t, err)

		// Convert ABI to JSON string
		abiBytes, err := json.Marshal(factoryArtifact.ABI)
		require.NoError(t, err)

		// Factory constructor takes feeToSetter address
		args := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			Abi:             string(abiBytes),
			Bytecode:        factoryArtifact.Bytecode,
			ConstructorArgs: []any{fromAddress}, // Use deployer as feeToSetter
			Title:           "Deploy Uniswap V2 Factory",
			Description:     "Deploying factory contract",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2FactoryDeployment,
		}

		// Generate deployment transaction
		deployment, abiData, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(args)
		require.Greater(t, len(abiData.Methods), 0)
		require.NoError(t, err)

		assert.NotEmpty(t, deployment.Data)
		// Check that deployment data contains bytecode (may have 0x prefix differences)
		assert.Contains(t, deployment.Data, strings.TrimPrefix(factoryArtifact.Bytecode, "0x"))
		// Should have constructor args appended
		assert.Greater(t, len(deployment.Data), len(factoryArtifact.Bytecode))

		// Deploy to testnet
		txHash, err := sendTransaction(client, privateKey, deployment.Data)
		require.NoError(t, err)
		assert.NotEqual(t, common.Hash{}, txHash)

		t.Logf("Factory transaction sent: %s", txHash.Hex())

		// Wait for transaction confirmation
		receipt, err := waitForTransaction(client, txHash, 30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, receipt)

		// Verify transaction was successful
		assert.Equal(t, uint64(1), receipt.Status, "Transaction failed")

		// Get deployed contract address
		contractAddress := getContractAddress(receipt)
		assert.NotEqual(t, common.Address{}, contractAddress)

		t.Logf("Factory deployed at: %s", contractAddress.Hex())

		// Verify contract code exists at the address
		code, err := client.CodeAt(context.Background(), contractAddress, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, code, "No code at deployed contract address")

		// Verify we can call the feeToSetter() function
		parsedABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
		require.NoError(t, err)

		feeToSetterData, err := parsedABI.Pack("feeToSetter")
		require.NoError(t, err)

		result, err := client.CallContract(context.Background(), ethereum.CallMsg{
			To:   &contractAddress,
			Data: feeToSetterData,
		}, nil)
		require.NoError(t, err)

		// Decode the address result
		var feeToSetterAddr common.Address
		err = parsedABI.UnpackIntoInterface(&feeToSetterAddr, "feeToSetter", result)
		require.NoError(t, err)
		assert.Equal(t, fromAddress, feeToSetterAddr, "feeToSetter address mismatch")
	})

	t.Run("Deploy UniswapV2Router to testnet", func(t *testing.T) {
		client, err := connectToTestnet()
		require.NoError(t, err)
		defer client.Close()

		// Get test account
		privateKey, fromAddress, err := getTestAccount()
		require.NoError(t, err)

		// First deploy WETH9 and Factory as dependencies
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		weth9AbiBytes, err := json.Marshal(weth9Artifact.ABI)
		require.NoError(t, err)

		// Deploy WETH9
		weth9Args := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			Abi:             string(weth9AbiBytes),
			Bytecode:        weth9Artifact.Bytecode,
			ConstructorArgs: []any{},
			Title:           "Deploy WETH9",
			Description:     "Deploying WETH9 for router",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2RouterDeployment,
		}

		weth9Deployment, abiData, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(weth9Args)
		require.Greater(t, len(abiData.Methods), 0)
		require.NoError(t, err)

		weth9TxHash, err := sendTransaction(client, privateKey, weth9Deployment.Data)
		require.NoError(t, err)

		weth9Receipt, err := waitForTransaction(client, weth9TxHash, 30*time.Second)
		require.NoError(t, err)
		require.Equal(t, uint64(1), weth9Receipt.Status)

		wethAddress := weth9Receipt.ContractAddress
		t.Logf("WETH9 deployed at: %s", wethAddress.Hex())

		// Deploy Factory
		factoryArtifact, err := contracts.GetFactoryArtifact()
		require.NoError(t, err)

		factoryAbiBytes, err := json.Marshal(factoryArtifact.ABI)
		require.NoError(t, err)

		factoryArgs := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			Abi:             string(factoryAbiBytes),
			Bytecode:        factoryArtifact.Bytecode,
			ConstructorArgs: []any{fromAddress},
			Title:           "Deploy Factory",
			Description:     "Deploying factory for router",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2FactoryDeployment,
		}

		factoryDeployment, abiData, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(factoryArgs)
		require.Greater(t, len(abiData.Methods), 0)
		require.NoError(t, err)

		factoryTxHash, err := sendTransaction(client, privateKey, factoryDeployment.Data)
		require.NoError(t, err)

		factoryReceipt, err := waitForTransaction(client, factoryTxHash, 30*time.Second)
		require.NoError(t, err)
		require.Equal(t, uint64(1), factoryReceipt.Status)

		factoryAddress := factoryReceipt.ContractAddress
		t.Logf("Factory deployed at: %s", factoryAddress.Hex())

		// Now deploy Router with actual Factory and WETH addresses
		routerArtifact, err := contracts.GetRouterArtifact()
		require.NoError(t, err)

		routerAbiBytes, err := json.Marshal(routerArtifact.ABI)
		require.NoError(t, err)

		args := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			Abi:             string(routerAbiBytes),
			Bytecode:        routerArtifact.Bytecode,
			ConstructorArgs: []any{factoryAddress, wethAddress},
			Title:           "Deploy Uniswap V2 Router",
			Description:     "Deploying router contract",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2RouterDeployment,
		}

		// Generate deployment transaction
		deployment, abiData, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(args)
		require.Greater(t, len(abiData.Methods), 0)
		require.NoError(t, err)

		assert.NotEmpty(t, deployment.Data)
		assert.Contains(t, deployment.Data, strings.TrimPrefix(routerArtifact.Bytecode, "0x"))
		// Should have constructor args appended
		assert.Greater(t, len(deployment.Data), len(routerArtifact.Bytecode))

		// Deploy to testnet
		txHash, err := sendTransaction(client, privateKey, deployment.Data)
		require.NoError(t, err)
		assert.NotEqual(t, common.Hash{}, txHash)

		t.Logf("Router transaction sent: %s", txHash.Hex())

		// Wait for transaction confirmation
		receipt, err := waitForTransaction(client, txHash, 30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, receipt)

		// Verify transaction was successful
		assert.Equal(t, uint64(1), receipt.Status, "Transaction failed")

		// Get deployed contract address
		contractAddress := getContractAddress(receipt)
		assert.NotEqual(t, common.Address{}, contractAddress)

		t.Logf("Router deployed at: %s", contractAddress.Hex())

		// Verify contract code exists at the address
		code, err := client.CodeAt(context.Background(), contractAddress, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, code, "No code at deployed contract address")

		// Verify we can call the factory() and WETH() functions
		parsedABI, err := abi.JSON(strings.NewReader(string(routerAbiBytes)))
		require.NoError(t, err)

		// Check factory address
		factoryData, err := parsedABI.Pack("factory")
		require.NoError(t, err)

		result, err := client.CallContract(context.Background(), ethereum.CallMsg{
			To:   &contractAddress,
			Data: factoryData,
		}, nil)
		require.NoError(t, err)

		var factoryAddrResult common.Address
		err = parsedABI.UnpackIntoInterface(&factoryAddrResult, "factory", result)
		require.NoError(t, err)
		assert.Equal(t, factoryAddress, factoryAddrResult, "factory address mismatch")

		// Check WETH address
		wethData, err := parsedABI.Pack("WETH")
		require.NoError(t, err)

		result, err = client.CallContract(context.Background(), ethereum.CallMsg{
			To:   &contractAddress,
			Data: wethData,
		}, nil)
		require.NoError(t, err)

		var wethAddrResult common.Address
		err = parsedABI.UnpackIntoInterface(&wethAddrResult, "WETH", result)
		require.NoError(t, err)
		assert.Equal(t, wethAddress, wethAddrResult, "WETH address mismatch")
	})

	t.Run("Deploy WETH9 to testnet", func(t *testing.T) {
		client, err := connectToTestnet()
		require.NoError(t, err)
		defer client.Close()

		// Get test account
		privateKey, fromAddress, err := getTestAccount()
		require.NoError(t, err)

		// Check balance
		balance, err := client.BalanceAt(context.Background(), fromAddress, nil)
		require.NoError(t, err)
		require.Greater(t, balance.Cmp(big.NewInt(0)), 0, "Test account has no balance")

		// Get WETH9 artifact
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		abiBytes, err := json.Marshal(weth9Artifact.ABI)
		require.NoError(t, err)

		// Prepare deployment arguments
		args := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			Abi:             string(abiBytes),
			Bytecode:        weth9Artifact.Bytecode,
			ConstructorArgs: []any{},
			Title:           "Deploy WETH9",
			Description:     "Deploying WETH9 to testnet",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		// Generate deployment transaction
		deployment, abiData, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(args)
		require.Greater(t, len(abiData.Methods), 0)
		require.NoError(t, err)

		// Send transaction to testnet
		txHash, err := sendTransaction(client, privateKey, deployment.Data)
		require.NoError(t, err)
		assert.NotEqual(t, common.Hash{}, txHash)

		t.Logf("Transaction sent: %s", txHash.Hex())

		// Wait for transaction confirmation
		receipt, err := waitForTransaction(client, txHash, 30*time.Second)
		require.NoError(t, err)
		require.NotNil(t, receipt)

		// Verify transaction was successful
		assert.Equal(t, uint64(1), receipt.Status, "Transaction failed")

		// Get deployed contract address
		contractAddress := getContractAddress(receipt)
		assert.NotEqual(t, common.Address{}, contractAddress)

		t.Logf("WETH9 deployed at: %s", contractAddress.Hex())

		// Verify contract code exists at the address
		code, err := client.CodeAt(context.Background(), contractAddress, nil)
		require.NoError(t, err)
		assert.NotEmpty(t, code, "No code at deployed contract address")
	})

	t.Run("Invalid ABI format", func(t *testing.T) {
		args := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			Abi:             "invalid json",
			Bytecode:        "0x608060405234801561001057600080fd5b50",
			ConstructorArgs: []any{"Test"},
			Title:           "Invalid Deploy",
			Description:     "Should fail",
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		_, _, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode constructor arguments")
	})

	t.Run("Missing required fields", func(t *testing.T) {
		args := services.ContractDeploymentWithBytecodeAndAbiTransactionArgs{
			// Missing Abi and Bytecode
			ConstructorArgs: []any{},
			Title:           "Deploy",
			// Missing Description
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		_, _, err := evmService.GetContractDeploymentTransactionWithBytecodeAndAbi(args)
		assert.Error(t, err)
	})
}

// TestGetTransactionData tests the GetTransactionData method
func TestGetTransactionData(t *testing.T) {
	// Create EVM service
	evmService := services.NewEvmService()

	t.Run("Generate function call data for WETH9 deposit", func(t *testing.T) {
		// Get WETH9 ABI
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		abiBytes, err := json.Marshal(weth9Artifact.ABI)
		require.NoError(t, err)

		args := services.GetTransactionDataArgs{
			ContractAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7",
			FunctionName:    "deposit",
			FunctionArgs:    []any{}, // deposit takes no arguments
			Abi:             string(abiBytes),
			Title:           "Deposit ETH",
			Description:     "Wrap ETH to WETH",
			Value:           "1000000000000000000", // 1 ETH
		}

		data, err := evmService.GetTransactionData(args)
		require.NoError(t, err)

		// Verify the data is not empty and starts with 0x
		assert.NotEmpty(t, data)
		assert.True(t, strings.HasPrefix(data, "0x"))
		// deposit() function selector is 0xd0e30db0
		assert.Equal(t, "0xd0e30db0", data)
	})

	t.Run("Generate function call data for WETH9 withdraw", func(t *testing.T) {
		// Get WETH9 ABI
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		abiBytes, err := json.Marshal(weth9Artifact.ABI)
		require.NoError(t, err)

		withdrawAmount := big.NewInt(500000000000000000) // 0.5 ETH

		args := services.GetTransactionDataArgs{
			ContractAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7",
			FunctionName:    "withdraw",
			FunctionArgs:    []any{withdrawAmount},
			Abi:             string(abiBytes),
			Title:           "Withdraw WETH",
			Description:     "Unwrap WETH to ETH",
			Value:           "0",
		}

		data, err := evmService.GetTransactionData(args)
		require.NoError(t, err)

		assert.NotEmpty(t, data)
		assert.True(t, strings.HasPrefix(data, "0x"))
		// withdraw(uint256) function selector is 0x2e1a7d4d
		assert.True(t, strings.HasPrefix(data, "0x2e1a7d4d"))
		// Data should be 4 bytes selector + 32 bytes amount = 36 bytes (72 hex chars + 0x)
		assert.Equal(t, 74, len(data))
	})

	t.Run("Generate function call data for UniswapV2Factory createPair", func(t *testing.T) {
		// Get Factory ABI
		factoryArtifact, err := contracts.GetFactoryArtifact()
		require.NoError(t, err)

		abiBytes, err := json.Marshal(factoryArtifact.ABI)
		require.NoError(t, err)

		tokenA := common.HexToAddress("0x5B38Da6a701c568545dCfcB03FcB875f56beddC4")
		tokenB := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7")

		args := services.GetTransactionDataArgs{
			ContractAddress: "0x1234567890123456789012345678901234567890",
			FunctionName:    "createPair",
			FunctionArgs:    []any{tokenA, tokenB},
			Abi:             string(abiBytes),
			Title:           "Create Pair",
			Description:     "Create Uniswap pair",
			Value:           "0",
		}

		data, err := evmService.GetTransactionData(args)
		require.NoError(t, err)

		assert.NotEmpty(t, data)
		assert.True(t, strings.HasPrefix(data, "0x"))
		// createPair function selector is 0xc9c65396
		assert.True(t, strings.HasPrefix(data, "0xc9c65396"))
		// Data should be 4 bytes selector + 32 bytes address + 32 bytes address = 68 bytes (136 hex chars + 0x)
		assert.Equal(t, 138, len(data))
	})

	t.Run("Generate function call data for UniswapV2Router addLiquidity", func(t *testing.T) {
		// Get Router ABI
		routerArtifact, err := contracts.GetRouterArtifact()
		require.NoError(t, err)

		abiBytes, err := json.Marshal(routerArtifact.ABI)
		require.NoError(t, err)

		tokenA := common.HexToAddress("0x5B38Da6a701c568545dCfcB03FcB875f56beddC4")
		tokenB := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7")
		amountADesired := big.NewInt(1000000000000000000) // 1 token
		amountBDesired := big.NewInt(2000000000000000000) // 2 tokens
		amountAMin := big.NewInt(900000000000000000)      // 0.9 token
		amountBMin := big.NewInt(1800000000000000000)     // 1.8 tokens
		to := common.HexToAddress("0x1234567890123456789012345678901234567890")
		deadline := big.NewInt(time.Now().Add(time.Hour).Unix())

		args := services.GetTransactionDataArgs{
			ContractAddress: "0x9876543210987654321098765432109876543210",
			FunctionName:    "addLiquidity",
			FunctionArgs:    []any{tokenA, tokenB, amountADesired, amountBDesired, amountAMin, amountBMin, to, deadline},
			Abi:             string(abiBytes),
			Title:           "Add Liquidity",
			Description:     "Add liquidity to pool",
			Value:           "0",
		}

		data, err := evmService.GetTransactionData(args)
		require.NoError(t, err)

		assert.NotEmpty(t, data)
		assert.True(t, strings.HasPrefix(data, "0x"))
		// addLiquidity function selector is 0xe8e33700
		assert.True(t, strings.HasPrefix(data, "0xe8e33700"))
		// Should have multiple parameters encoded
		assert.Greater(t, len(data), 500)
	})

	t.Run("Invalid function name", func(t *testing.T) {
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		abiBytes, err := json.Marshal(weth9Artifact.ABI)
		require.NoError(t, err)

		args := services.GetTransactionDataArgs{
			ContractAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7",
			FunctionName:    "invalidFunction", // Function not in ABI
			FunctionArgs:    []any{},
			Abi:             string(abiBytes),
			Title:           "Invalid Call",
			Description:     "Should fail",
		}

		_, err = evmService.GetTransactionData(args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode function call")
	})

	t.Run("Invalid ABI JSON", func(t *testing.T) {
		args := services.GetTransactionDataArgs{
			ContractAddress: "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7",
			FunctionName:    "transfer",
			FunctionArgs:    []any{},
			Abi:             "not valid json",
			Title:           "Invalid ABI",
			Description:     "Should fail",
		}

		_, err := evmService.GetTransactionData(args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to encode function call")
	})

	t.Run("Missing required fields", func(t *testing.T) {
		args := services.GetTransactionDataArgs{
			// Missing ContractAddress
			FunctionName: "transfer",
			FunctionArgs: []any{},
			Abi:          "[]",
			// Missing Title and Description
		}

		_, err := evmService.GetTransactionData(args)
		assert.Error(t, err)
	})

	t.Run("Invalid contract address format", func(t *testing.T) {
		args := services.GetTransactionDataArgs{
			ContractAddress: "invalid-address",
			FunctionName:    "transfer",
			FunctionArgs:    []any{},
			Abi:             "[]",
			Title:           "Test",
			Description:     "Test",
		}

		_, err := evmService.GetTransactionData(args)
		assert.Error(t, err)
	})
}

// TestGetAllAbiMethods tests the GetAllAbiMethods method
func TestGetAllAbiMethods(t *testing.T) {
	evmService := services.NewEvmService()

	t.Run("Get all methods from WETH9 ABI", func(t *testing.T) {
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		// The ABI from artifact should be the ABI array
		abiArray, ok := weth9Artifact.ABI.([]interface{})
		require.True(t, ok)

		// Convert to models.JSON - this will represent the raw ABI array
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		methods, err := evmService.GetAllAbiMethods(abiJSONData)
		require.NoError(t, err)
		require.NotEmpty(t, methods)

		// WETH9 should have these methods: allowance, approve, balanceOf, decimals, deposit, name, symbol, totalSupply, transfer, transferFrom, withdraw
		expectedMethods := []string{"allowance", "approve", "balanceOf", "decimals", "deposit", "name", "symbol", "totalSupply", "transfer", "transferFrom", "withdraw"}

		// Create a map of actual method names for easier checking
		actualMethods := make(map[string]bool)
		for _, method := range methods {
			actualMethods[method.Name] = true
		}

		// Verify all expected methods are present
		for _, expectedMethod := range expectedMethods {
			assert.True(t, actualMethods[expectedMethod], "Expected method %s not found", expectedMethod)
		}

		// Verify at least the expected number of methods
		assert.GreaterOrEqual(t, len(methods), len(expectedMethods))
	})

	t.Run("Get all methods from UniswapV2Factory ABI", func(t *testing.T) {
		factoryArtifact, err := contracts.GetFactoryArtifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		abiArray, ok := factoryArtifact.ABI.([]interface{})
		require.True(t, ok)
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		methods, err := evmService.GetAllAbiMethods(abiJSONData)
		require.NoError(t, err)
		require.NotEmpty(t, methods)

		// Factory should have these key methods
		expectedMethods := []string{"createPair", "feeTo", "feeToSetter", "getPair", "allPairs", "allPairsLength", "setFeeTo", "setFeeToSetter"}

		actualMethods := make(map[string]bool)
		for _, method := range methods {
			actualMethods[method.Name] = true
		}

		for _, expectedMethod := range expectedMethods {
			assert.True(t, actualMethods[expectedMethod], "Expected method %s not found", expectedMethod)
		}
	})

	t.Run("Get all methods from UniswapV2Router ABI", func(t *testing.T) {
		routerArtifact, err := contracts.GetRouterArtifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		abiArray, ok := routerArtifact.ABI.([]interface{})
		require.True(t, ok)
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		methods, err := evmService.GetAllAbiMethods(abiJSONData)
		require.NoError(t, err)
		require.NotEmpty(t, methods)

		// Router should have these key methods
		expectedMethods := []string{"WETH", "factory", "addLiquidity", "addLiquidityETH", "removeLiquidity", "removeLiquidityETH", "swapExactTokensForTokens", "swapTokensForExactTokens", "swapExactETHForTokens", "swapTokensForExactETH", "swapExactTokensForETH", "swapETHForExactTokens"}

		actualMethods := make(map[string]bool)
		for _, method := range methods {
			actualMethods[method.Name] = true
		}

		for _, expectedMethod := range expectedMethods {
			assert.True(t, actualMethods[expectedMethod], "Expected method %s not found", expectedMethod)
		}
	})

	t.Run("Get all methods from SimpleToken ABI", func(t *testing.T) {
		// Use the test SimpleToken contract
		evmService := services.NewEvmService()

		args := services.ContractDeploymentWithContractCodeTransactionArgs{
			ContractName:    "SimpleToken",
			ConstructorArgs: []any{"Test Token", "TEST", big.NewInt(1000000)},
			ContractCode:    simpleERC20Contract,
			Title:           "Deploy Test Token",
			Description:     "Deploying a test ERC20 token",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		_, abiData, err := evmService.GetContractDeploymentTransactionWithContractCode(args)
		require.NoError(t, err)

		// Convert parsed go-ethereum ABI back to ABI array format
		// Extract the methods from the parsed ABI and create a simple ABI array
		abiArray := []map[string]interface{}{}
		for _, method := range abiData.Methods {
			inputs := []map[string]interface{}{}
			for _, input := range method.Inputs {
				inputs = append(inputs, map[string]interface{}{
					"name": input.Name,
					"type": input.Type.String(),
				})
			}
			outputs := []map[string]interface{}{}
			for _, output := range method.Outputs {
				outputs = append(outputs, map[string]interface{}{
					"name": output.Name,
					"type": output.Type.String(),
				})
			}
			abiArray = append(abiArray, map[string]interface{}{
				"type":            "function",
				"name":            method.Name,
				"inputs":          inputs,
				"outputs":         outputs,
				"stateMutability": method.StateMutability,
				"constant":        method.Constant,
				"payable":         method.Payable,
			})
		}

		abiJSON := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		methods, err := evmService.GetAllAbiMethods(abiJSON)
		require.NoError(t, err)
		require.NotEmpty(t, methods)

		// SimpleToken should have these methods
		expectedMethods := []string{"name", "symbol", "totalSupply", "balanceOf", "transfer"}

		actualMethods := make(map[string]bool)
		for _, method := range methods {
			actualMethods[method.Name] = true
		}

		for _, expectedMethod := range expectedMethods {
			assert.True(t, actualMethods[expectedMethod], "Expected method %s not found", expectedMethod)
		}
	})

	t.Run("Invalid ABI JSON", func(t *testing.T) {
		// Create invalid JSON
		invalidJSON := models.JSON(map[string]interface{}{
			"invalid": "structure",
		})

		_, err := evmService.GetAllAbiMethods(invalidJSON)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse ABI")
	})

	t.Run("Empty ABI", func(t *testing.T) {
		emptyABI := models.JSON(map[string]interface{}{
			"abi": []interface{}{},
		})

		methods, err := evmService.GetAllAbiMethods(emptyABI)
		require.NoError(t, err)
		assert.Empty(t, methods)
	})
}

// TestGetAbiMethod tests the GetAbiMethod method
func TestGetAbiMethod(t *testing.T) {
	evmService := services.NewEvmService()

	t.Run("Get specific method from WETH9 ABI", func(t *testing.T) {
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		abiArray, ok := weth9Artifact.ABI.([]interface{})
		require.True(t, ok)
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		// Test getting the deposit method
		depositMethod, err := evmService.GetAbiMethod(abiJSONData, "deposit")
		require.NoError(t, err)

		assert.Equal(t, "deposit", depositMethod.Name)
		assert.Empty(t, depositMethod.Inputs)   // deposit takes no parameters
		assert.Empty(t, depositMethod.Outputs)  // deposit returns nothing
		assert.False(t, depositMethod.Constant) // deposit is not a view function
		assert.True(t, depositMethod.Payable)   // deposit is payable

		// Test getting the withdraw method
		withdrawMethod, err := evmService.GetAbiMethod(abiJSONData, "withdraw")
		require.NoError(t, err)

		assert.Equal(t, "withdraw", withdrawMethod.Name)
		assert.Len(t, withdrawMethod.Inputs, 1) // withdraw takes 1 parameter (amount)
		assert.Equal(t, "uint256", withdrawMethod.Inputs[0].Type.String())
		assert.Empty(t, withdrawMethod.Outputs)  // withdraw returns nothing
		assert.False(t, withdrawMethod.Constant) // withdraw is not a view function
		assert.False(t, withdrawMethod.Payable)  // withdraw is not payable

		// Test getting a view method
		balanceOfMethod, err := evmService.GetAbiMethod(abiJSONData, "balanceOf")
		require.NoError(t, err)

		assert.Equal(t, "balanceOf", balanceOfMethod.Name)
		assert.Len(t, balanceOfMethod.Inputs, 1) // balanceOf takes 1 parameter (address)
		assert.Equal(t, "address", balanceOfMethod.Inputs[0].Type.String())
		assert.Len(t, balanceOfMethod.Outputs, 1) // balanceOf returns 1 value (uint256)
		assert.Equal(t, "uint256", balanceOfMethod.Outputs[0].Type.String())
		assert.True(t, balanceOfMethod.Constant) // balanceOf is a view function
		assert.False(t, balanceOfMethod.Payable) // balanceOf is not payable
	})

	t.Run("Get specific method from UniswapV2Factory ABI", func(t *testing.T) {
		factoryArtifact, err := contracts.GetFactoryArtifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		abiArray, ok := factoryArtifact.ABI.([]interface{})
		require.True(t, ok)
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		// Test getting the createPair method
		createPairMethod, err := evmService.GetAbiMethod(abiJSONData, "createPair")
		require.NoError(t, err)

		assert.Equal(t, "createPair", createPairMethod.Name)
		assert.Len(t, createPairMethod.Inputs, 2) // createPair takes 2 parameters (tokenA, tokenB)
		assert.Equal(t, "address", createPairMethod.Inputs[0].Type.String())
		assert.Equal(t, "address", createPairMethod.Inputs[1].Type.String())
		assert.Len(t, createPairMethod.Outputs, 1) // createPair returns pair address
		assert.Equal(t, "address", createPairMethod.Outputs[0].Type.String())
		assert.False(t, createPairMethod.Constant) // createPair is not a view function
		assert.False(t, createPairMethod.Payable)  // createPair is not payable

		// Test getting a view method
		getPairMethod, err := evmService.GetAbiMethod(abiJSONData, "getPair")
		require.NoError(t, err)

		assert.Equal(t, "getPair", getPairMethod.Name)
		assert.Len(t, getPairMethod.Inputs, 2) // getPair takes 2 parameters (tokenA, tokenB)
		assert.Equal(t, "address", getPairMethod.Inputs[0].Type.String())
		assert.Equal(t, "address", getPairMethod.Inputs[1].Type.String())
		assert.Len(t, getPairMethod.Outputs, 1) // getPair returns pair address
		assert.Equal(t, "address", getPairMethod.Outputs[0].Type.String())
		assert.True(t, getPairMethod.Constant) // getPair is a view function
		assert.False(t, getPairMethod.Payable) // getPair is not payable
	})

	t.Run("Get specific method from UniswapV2Router ABI", func(t *testing.T) {
		routerArtifact, err := contracts.GetRouterArtifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		abiArray, ok := routerArtifact.ABI.([]interface{})
		require.True(t, ok)
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		// Test getting the addLiquidity method
		addLiquidityMethod, err := evmService.GetAbiMethod(abiJSONData, "addLiquidity")
		require.NoError(t, err)

		assert.Equal(t, "addLiquidity", addLiquidityMethod.Name)
		assert.Len(t, addLiquidityMethod.Inputs, 8) // addLiquidity takes 8 parameters
		// Verify some key parameter types
		assert.Equal(t, "address", addLiquidityMethod.Inputs[0].Type.String()) // tokenA
		assert.Equal(t, "address", addLiquidityMethod.Inputs[1].Type.String()) // tokenB
		assert.Equal(t, "uint256", addLiquidityMethod.Inputs[2].Type.String()) // amountADesired
		assert.Len(t, addLiquidityMethod.Outputs, 3)                           // returns amountA, amountB, liquidity
		assert.False(t, addLiquidityMethod.Constant)                           // addLiquidity is not a view function
		assert.False(t, addLiquidityMethod.Payable)                            // addLiquidity is not payable

		// Test getting the swapExactTokensForTokens method
		swapMethod, err := evmService.GetAbiMethod(abiJSONData, "swapExactTokensForTokens")
		require.NoError(t, err)

		assert.Equal(t, "swapExactTokensForTokens", swapMethod.Name)
		assert.Len(t, swapMethod.Inputs, 5)                              // swapExactTokensForTokens takes 5 parameters
		assert.Equal(t, "uint256", swapMethod.Inputs[0].Type.String())   // amountIn
		assert.Equal(t, "uint256", swapMethod.Inputs[1].Type.String())   // amountOutMin
		assert.Equal(t, "address[]", swapMethod.Inputs[2].Type.String()) // path
		assert.Equal(t, "address", swapMethod.Inputs[3].Type.String())   // to
		assert.Equal(t, "uint256", swapMethod.Inputs[4].Type.String())   // deadline
		assert.Len(t, swapMethod.Outputs, 1)                             // returns amounts[]
		assert.Equal(t, "uint256[]", swapMethod.Outputs[0].Type.String())
		assert.False(t, swapMethod.Constant) // swap is not a view function
		assert.False(t, swapMethod.Payable)  // this specific swap is not payable
	})

	t.Run("Get method from SimpleToken ABI", func(t *testing.T) {
		evmService := services.NewEvmService()

		args := services.ContractDeploymentWithContractCodeTransactionArgs{
			ContractName:    "SimpleToken",
			ConstructorArgs: []any{"Test Token", "TEST", big.NewInt(1000000)},
			ContractCode:    simpleERC20Contract,
			Title:           "Deploy Test Token",
			Description:     "Deploying a test ERC20 token",
			Value:           "0",
			TransactionType: models.TransactionTypeUniswapV2TokenDeployment,
		}

		_, abiData, err := evmService.GetContractDeploymentTransactionWithContractCode(args)
		require.NoError(t, err)

		// Convert parsed go-ethereum ABI back to ABI array format
		// Extract the methods from the parsed ABI and create a simple ABI array
		abiArray := []map[string]interface{}{}
		for _, method := range abiData.Methods {
			inputs := []map[string]interface{}{}
			for _, input := range method.Inputs {
				inputs = append(inputs, map[string]interface{}{
					"name": input.Name,
					"type": input.Type.String(),
				})
			}
			outputs := []map[string]interface{}{}
			for _, output := range method.Outputs {
				outputs = append(outputs, map[string]interface{}{
					"name": output.Name,
					"type": output.Type.String(),
				})
			}
			abiArray = append(abiArray, map[string]interface{}{
				"type":            "function",
				"name":            method.Name,
				"inputs":          inputs,
				"outputs":         outputs,
				"stateMutability": method.StateMutability,
				"constant":        method.Constant,
				"payable":         method.Payable,
			})
		}

		abiJSON := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		// Test getting the transfer method
		transferMethod, err := evmService.GetAbiMethod(abiJSON, "transfer")
		require.NoError(t, err)

		assert.Equal(t, "transfer", transferMethod.Name)
		assert.Len(t, transferMethod.Inputs, 2) // transfer takes 2 parameters (to, amount)
		assert.Equal(t, "address", transferMethod.Inputs[0].Type.String())
		assert.Equal(t, "uint256", transferMethod.Inputs[1].Type.String())
		assert.Len(t, transferMethod.Outputs, 1) // transfer returns bool
		assert.Equal(t, "bool", transferMethod.Outputs[0].Type.String())
		assert.False(t, transferMethod.Constant) // transfer is not a view function
		assert.False(t, transferMethod.Payable)  // transfer is not payable

		// Test getting a view method
		nameMethod, err := evmService.GetAbiMethod(abiJSON, "name")
		require.NoError(t, err)

		assert.Equal(t, "name", nameMethod.Name)
		assert.Empty(t, nameMethod.Inputs)   // name takes no parameters
		assert.Len(t, nameMethod.Outputs, 1) // name returns string
		assert.Equal(t, "string", nameMethod.Outputs[0].Type.String())
		// For SimpleToken, check StateMutability instead as it's more reliable
		assert.Contains(t, []string{"view", "pure"}, nameMethod.StateMutability) // name is a view/pure function
		assert.False(t, nameMethod.Payable)                                      // name is not payable
	})

	t.Run("Method not found", func(t *testing.T) {
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		abiArray, ok := weth9Artifact.ABI.([]interface{})
		require.True(t, ok)
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		_, err = evmService.GetAbiMethod(abiJSONData, "nonExistentMethod")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "method 'nonExistentMethod' not found in ABI")
	})

	t.Run("Invalid ABI JSON", func(t *testing.T) {
		invalidJSON := models.JSON(map[string]interface{}{
			"invalid": "structure",
		})

		_, err := evmService.GetAbiMethod(invalidJSON, "someMethod")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse ABI")
	})

	t.Run("Empty method name", func(t *testing.T) {
		weth9Artifact, err := contracts.GetWETH9Artifact()
		require.NoError(t, err)

		// Convert ABI directly to models.JSON
		abiArray, ok := weth9Artifact.ABI.([]interface{})
		require.True(t, ok)
		abiJSONData := models.JSON(map[string]interface{}{
			"abi": abiArray,
		})

		_, err = evmService.GetAbiMethod(abiJSONData, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "method '' not found in ABI")
	})
}
