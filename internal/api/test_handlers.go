package api

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gofiber/fiber/v2"
)

// TestSignTransactionRequest represents a request to sign a transaction for testing
type TestSignTransactionRequest struct {
	PrivateKey  string                 `json:"privateKey"`
	Transaction map[string]interface{} `json:"transaction"`
}

// TestSignTransactionResponse represents the response from signing a transaction
type TestSignTransactionResponse struct {
	Success bool   `json:"success"`
	TxHash  string `json:"txHash,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleTestSignTransaction handles real transaction signing for E2E tests
func (s *APIServer) handleTestSignTransaction(c *fiber.Ctx) error {
	var req TestSignTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(TestSignTransactionResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	// Parse private key
	privateKeyBytes, err := hex.DecodeString(req.PrivateKey)
	if err != nil {
		return c.Status(400).JSON(TestSignTransactionResponse{
			Success: false,
			Error:   "Invalid private key format",
		})
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return c.Status(400).JSON(TestSignTransactionResponse{
			Success: false,
			Error:   "Failed to parse private key",
		})
	}

	// Get the active chain configuration
	activeChain, err := s.chainService.GetActiveChain()
	if err != nil {
		return c.Status(500).JSON(TestSignTransactionResponse{
			Success: false,
			Error:   "Failed to get active chain configuration",
		})
	}

	// Connect to Ethereum client
	client, err := ethclient.Dial(activeChain.RPC)
	if err != nil {
		return c.Status(500).JSON(TestSignTransactionResponse{
			Success: false,
			Error:   "Failed to connect to Ethereum client",
		})
	}
	defer client.Close()

	// Create auth from private key
	chainIDInt, ok := new(big.Int).SetString(activeChain.NetworkID, 10)
	if !ok {
		return c.Status(500).JSON(TestSignTransactionResponse{
			Success: false,
			Error:   "Invalid chain ID format",
		})
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainIDInt)
	if err != nil {
		return c.Status(500).JSON(TestSignTransactionResponse{
			Success: false,
			Error:   "Failed to create transaction signer",
		})
	}

	// Parse transaction parameters
	tx := req.Transaction

	// Handle different types of transactions
	if dataHex, hasData := tx["data"].(string); hasData && dataHex != "" && dataHex != "0x" {
		// This is a contract deployment or contract call
		txHash, err := s.executeContractTransaction(client, auth, tx, privateKey)
		if err != nil {
			log.Printf("Contract transaction failed: %v", err)
			return c.Status(500).JSON(TestSignTransactionResponse{
				Success: false,
				Error:   fmt.Sprintf("Contract transaction failed: %v", err),
			})
		}

		return c.JSON(TestSignTransactionResponse{
			Success: true,
			TxHash:  txHash,
		})
	} else {
		// This is a regular ETH transfer
		txHash, err := s.executeETHTransfer(client, auth, tx, privateKey)
		if err != nil {
			log.Printf("ETH transfer failed: %v", err)
			return c.Status(500).JSON(TestSignTransactionResponse{
				Success: false,
				Error:   fmt.Sprintf("ETH transfer failed: %v", err),
			})
		}

		return c.JSON(TestSignTransactionResponse{
			Success: true,
			TxHash:  txHash,
		})
	}
}

// executeContractTransaction executes a contract deployment or call
func (s *APIServer) executeContractTransaction(client *ethclient.Client, auth *bind.TransactOpts, tx map[string]interface{}, privateKey *ecdsa.PrivateKey) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse transaction parameters
	dataHex := tx["data"].(string)
	if !strings.HasPrefix(dataHex, "0x") {
		dataHex = "0x" + dataHex
	}

	hexString := dataHex[2:]
	log.Printf("DEBUG: Attempting to decode hex string of length %d: %s...", len(hexString), hexString[:min(100, len(hexString))])

	data, err := hex.DecodeString(hexString)
	if err != nil {
		return "", fmt.Errorf("failed to decode transaction data (length: %d, odd: %t): %w", len(hexString), len(hexString)%2 == 1, err)
	}

	// Parse gas limit
	var gasLimit uint64 = 3000000 // Default gas limit
	if gasLimitStr, ok := tx["gas"].(string); ok {
		if strings.HasPrefix(gasLimitStr, "0x") {
			gasLimitBig, ok := new(big.Int).SetString(gasLimitStr[2:], 16)
			if ok {
				gasLimit = gasLimitBig.Uint64()
			}
		}
	}

	// Parse gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// Parse value
	var value *big.Int = big.NewInt(0)
	if valueStr, ok := tx["value"].(string); ok && valueStr != "" {
		if strings.HasPrefix(valueStr, "0x") {
			value, ok = new(big.Int).SetString(valueStr[2:], 16)
			if !ok {
				value = big.NewInt(0)
			}
		}
	}

	// Get sender address from private key for nonce
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to cast public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get nonce
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	// Set transaction options
	auth.GasLimit = gasLimit
	auth.GasPrice = gasPrice
	auth.Value = value
	auth.Context = ctx
	auth.Nonce = big.NewInt(int64(nonce))

	// Check if this is a contract deployment (no 'to' address)
	var toAddress *common.Address
	if toStr, ok := tx["to"].(string); ok && toStr != "" {
		addr := common.HexToAddress(toStr)
		toAddress = &addr
	}

	// Create transaction with proper nonce
	var signedTx *types.Transaction
	if toAddress == nil {
		// Contract deployment
		signedTx = types.NewContractCreation(
			nonce,
			value,
			gasLimit,
			gasPrice,
			data,
		)
	} else {
		// Contract call
		signedTx = types.NewTransaction(
			nonce,
			*toAddress,
			value,
			gasLimit,
			gasPrice,
			data,
		)
	}

	// Sign the transaction
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get network ID: %w", err)
	}

	signer := types.NewEIP155Signer(chainID)
	signedTx, err = types.SignTx(signedTx, signer, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send the transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	log.Printf("Contract transaction sent: %s", signedTx.Hash().Hex())
	return signedTx.Hash().Hex(), nil
}

// executeETHTransfer executes a regular ETH transfer
func (s *APIServer) executeETHTransfer(client *ethclient.Client, auth *bind.TransactOpts, txData map[string]interface{}, privateKey *ecdsa.PrivateKey) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse recipient address
	toStr, ok := txData["to"].(string)
	if !ok || toStr == "" {
		return "", fmt.Errorf("recipient address is required for ETH transfer")
	}
	toAddress := common.HexToAddress(toStr)

	// Parse value
	var value *big.Int = big.NewInt(0)
	if valueStr, ok := txData["value"].(string); ok && valueStr != "" {
		if strings.HasPrefix(valueStr, "0x") {
			value, ok = new(big.Int).SetString(valueStr[2:], 16)
			if !ok {
				value = big.NewInt(0)
			}
		}
	}

	// Get sender address from private key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("failed to cast public key to ECDSA")
	}
	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get gas price
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create and send transaction
	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	tx := types.NewTransaction(nonce, toAddress, value, 21000, gasPrice, nil)

	// Sign transaction
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get network ID: %w", err)
	}

	signer := types.NewEIP155Signer(chainID)
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	log.Printf("ETH transfer sent: %s", signedTx.Hash().Hex())
	return signedTx.Hash().Hex(), nil
}
