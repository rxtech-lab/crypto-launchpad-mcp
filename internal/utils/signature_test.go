package utils

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testnetRPC     = "http://localhost:8545"
	testnetChainID = int64(31337)
	// Test private key (Anvil account #0)
	testPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

// TestSignatureUtilities tests all signature utility functions with real blockchain interaction.
func TestSignatureUtilities(t *testing.T) {
	// Connect to testnet
	client, err := ethclient.Dial(testnetRPC)
	require.NoError(t, err, "Testnet should be running on localhost:8545 (run 'make e2e-network')")
	defer client.Close()

	// Verify network connectivity
	_, err = client.NetworkID(context.Background())
	require.NoError(t, err, "Failed to connect to testnet")

	// Setup test account
	privateKey, err := crypto.HexToECDSA(testPrivateKey[2:]) // Remove 0x prefix
	require.NoError(t, err)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	require.True(t, ok)

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	t.Run("GenerateMessage", func(t *testing.T) {
		message1 := GenerateMessage()
		assert.NotEmpty(t, message1)
		assert.Contains(t, message1, "I am signing into Launchpad at")

		// Wait at least one second to ensure different timestamps
		time.Sleep(1 * time.Second)
		message2 := GenerateMessage()
		assert.NotEqual(t, message1, message2, "Messages should have different timestamps")
	})

	t.Run("getAddressFromSignature_Success", func(t *testing.T) {
		message := "I am signing into Launchpad at 1234567890"

		// Create message hash
		messageHash := accounts.TextHash([]byte(message))

		// Sign the message
		signature, err := crypto.Sign(messageHash, privateKey)
		require.NoError(t, err)

		// Convert to hex format
		signatureHex := "0x" + hex.EncodeToString(signature)

		// Recover address from signature
		recoveredAddress, err := getAddressFromSignature(signatureHex, message)
		require.NoError(t, err)

		assert.Equal(t, address.Hex(), recoveredAddress)
	})

	t.Run("getAddressFromSignature_ValidationErrors", func(t *testing.T) {
		message := "test message"

		tests := []struct {
			name      string
			signature string
			wantError string
		}{
			{
				name:      "missing 0x prefix",
				signature: "1234567890abcdef",
				wantError: "signature must start with 0x",
			},
			{
				name:      "too short",
				signature: "0x1234",
				wantError: "signature must be 65 bytes",
			},
			{
				name:      "too long",
				signature: "0x" + hex.EncodeToString(make([]byte, 70)),
				wantError: "signature must be 65 bytes",
			},
			{
				name:      "invalid hex",
				signature: "0x" + "zz" + hex.EncodeToString(make([]byte, 64)),
				wantError: "failed to decode signature",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := getAddressFromSignature(tt.signature, message)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			})
		}
	})

	t.Run("getTransactionSender_WithRealTransaction", func(t *testing.T) {
		ctx := context.Background()

		// Create and send a transaction to get a real transaction hash
		nonce, err := client.PendingNonceAt(ctx, address)
		require.NoError(t, err)

		gasPrice, err := client.SuggestGasPrice(ctx)
		require.NoError(t, err)

		// Create a simple transaction (send 0 ETH to self)
		tx := types.NewTransaction(nonce, address, big.NewInt(0), 21000, gasPrice, nil)

		// Sign the transaction
		signer := types.NewEIP155Signer(big.NewInt(testnetChainID))
		signedTx, err := types.SignTx(tx, signer, privateKey)
		require.NoError(t, err)

		// Send transaction
		err = client.SendTransaction(ctx, signedTx)
		require.NoError(t, err)

		txHash := signedTx.Hash().Hex()
		t.Logf("Created test transaction: %s", txHash)

		// Wait for transaction to be mined
		receipt, err := waitForTransaction(client, signedTx.Hash(), 30*time.Second)
		require.NoError(t, err)
		require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)

		// Test getTransactionSender
		sender, err := getTransactionSender(txHash, testnetRPC)
		require.NoError(t, err)

		// Use case-insensitive comparison for Ethereum addresses
		assert.True(t, strings.EqualFold(address.Hex(), sender), "Expected %s, got %s", address.Hex(), sender)
	})

	t.Run("getTransactionSender_ValidationErrors", func(t *testing.T) {
		tests := []struct {
			name      string
			txHash    string
			wantError string
		}{
			{
				name:      "missing 0x prefix",
				txHash:    "1234567890abcdef",
				wantError: "transaction hash must start with 0x",
			},
			{
				name:      "too short",
				txHash:    "0x1234",
				wantError: "transaction hash must be 32 bytes",
			},
			{
				name:      "too long",
				txHash:    "0x" + hex.EncodeToString(make([]byte, 40)),
				wantError: "transaction hash must be 32 bytes",
			},
			{
				name:      "nonexistent transaction",
				txHash:    "0x" + hex.EncodeToString(make([]byte, 32)),
				wantError: "transaction not found",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := getTransactionSender(tt.txHash, testnetRPC)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			})
		}
	})

	t.Run("TransactionOwnershipBySignature_FullWorkflow", func(t *testing.T) {
		ctx := context.Background()

		// Create and send a transaction
		nonce, err := client.PendingNonceAt(ctx, address)
		require.NoError(t, err)

		gasPrice, err := client.SuggestGasPrice(ctx)
		require.NoError(t, err)

		tx := types.NewTransaction(nonce, address, big.NewInt(0), 21000, gasPrice, nil)
		signer := types.NewEIP155Signer(big.NewInt(testnetChainID))
		signedTx, err := types.SignTx(tx, signer, privateKey)
		require.NoError(t, err)

		err = client.SendTransaction(ctx, signedTx)
		require.NoError(t, err)

		txHash := signedTx.Hash().Hex()

		// Wait for transaction to be mined
		_, err = waitForTransaction(client, signedTx.Hash(), 30*time.Second)
		require.NoError(t, err)

		// Create a signature for the current message
		message := GenerateMessage()

		// Sign the hex-encoded message (as the frontend would do)
		hexMessageHash := accounts.TextHash([]byte(message))
		signature, err := crypto.Sign(hexMessageHash, privateKey)
		require.NoError(t, err)
		signatureHex := "0x" + hex.EncodeToString(signature)

		// Verify with the original message (as the backend receives)
		isOwner, err := VerifyTransactionOwnershipBySignature(testnetRPC, txHash, signatureHex, message)
		require.NoError(t, err)
		assert.True(t, isOwner, "Should verify ownership correctly")
	})

	t.Run("TransactionOwnershipBySignature_WrongOwner", func(t *testing.T) {
		ctx := context.Background()

		// Create a transaction with the test account
		nonce, err := client.PendingNonceAt(ctx, address)
		require.NoError(t, err)

		gasPrice, err := client.SuggestGasPrice(ctx)
		require.NoError(t, err)

		tx := types.NewTransaction(nonce, address, big.NewInt(0), 21000, gasPrice, nil)
		signer := types.NewEIP155Signer(big.NewInt(testnetChainID))
		signedTx, err := types.SignTx(tx, signer, privateKey)
		require.NoError(t, err)

		err = client.SendTransaction(ctx, signedTx)
		require.NoError(t, err)

		txHash := signedTx.Hash().Hex()

		// Wait for transaction to be mined
		_, err = waitForTransaction(client, signedTx.Hash(), 30*time.Second)
		require.NoError(t, err)

		// Create a signature with a different private key
		differentPrivateKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		message := GenerateMessage()
		hexMessage := "0x" + hex.EncodeToString([]byte(message))

		// Sign the hex-encoded message with different private key
		hexMessageHash := accounts.TextHash([]byte(hexMessage))
		signature, err := crypto.Sign(hexMessageHash, differentPrivateKey)
		require.NoError(t, err)
		signatureHex := "0x" + hex.EncodeToString(signature)

		// Test ownership verification with original message - should return false
		isOwner, err := VerifyTransactionOwnershipBySignature(testnetRPC, txHash, signatureHex, message)
		require.NoError(t, err)
		assert.False(t, isOwner, "Should not verify ownership for different signer")
	})

	t.Run("TransactionOwnershipBySignature_ValidationErrors", func(t *testing.T) {
		validTxHash := "0x" + hex.EncodeToString(make([]byte, 32))
		validSignature := "0x" + hex.EncodeToString(make([]byte, 65))

		tests := []struct {
			name      string
			rpcUrl    string
			txHash    string
			signature string
			wantError string
		}{
			{
				name:      "empty rpc url",
				rpcUrl:    "",
				txHash:    validTxHash,
				signature: validSignature,
				wantError: "RPC URL cannot be empty",
			},
			{
				name:      "empty tx hash",
				rpcUrl:    testnetRPC,
				txHash:    "",
				signature: validSignature,
				wantError: "transaction hash cannot be empty",
			},
			{
				name:      "empty signature",
				rpcUrl:    testnetRPC,
				txHash:    validTxHash,
				signature: "",
				wantError: "signature cannot be empty",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := VerifyTransactionOwnershipBySignature(tt.rpcUrl, tt.txHash, tt.signature, GenerateMessage())
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			})
		}
	})

	t.Run("PersonalSign_Success", func(t *testing.T) {
		message := "Test message for signing"

		// Sign the message using personalSign
		signature, err := personalSign(message, privateKey)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		assert.True(t, strings.HasPrefix(signature, "0x"))

		// Verify the signature by recovering the address
		recoveredAddress, err := getAddressFromSignature(signature, message)
		require.NoError(t, err)
		assert.Equal(t, address.Hex(), recoveredAddress)
	})

	t.Run("PersonalSign_ValidationErrors", func(t *testing.T) {
		tests := []struct {
			name       string
			message    string
			privateKey *ecdsa.PrivateKey
			wantError  string
		}{
			{
				name:       "nil private key",
				message:    "test message",
				privateKey: nil,
				wantError:  "private key cannot be nil",
			},
			{
				name:       "empty message",
				message:    "",
				privateKey: privateKey,
				wantError:  "message cannot be empty",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := personalSign(tt.message, tt.privateKey)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			})
		}
	})

	t.Run("PersonalSignFromHex_Success", func(t *testing.T) {
		message := "Test message for signing with hex key"

		// Sign the message using PersonalSignFromHex
		signature, err := PersonalSignFromHex(message, testPrivateKey)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		assert.True(t, strings.HasPrefix(signature, "0x"))

		// Verify the signature by recovering the address
		recoveredAddress, err := getAddressFromSignature(signature, message)
		require.NoError(t, err)
		assert.Equal(t, address.Hex(), recoveredAddress)
	})

	t.Run("PersonalSignFromHex_WithoutPrefix", func(t *testing.T) {
		message := "Test message for signing without 0x prefix"
		privateKeyWithoutPrefix := testPrivateKey[2:] // Remove 0x prefix

		// Should work with or without 0x prefix
		signature, err := PersonalSignFromHex(message, privateKeyWithoutPrefix)
		require.NoError(t, err)
		assert.NotEmpty(t, signature)
		assert.True(t, strings.HasPrefix(signature, "0x"))

		// Verify the signature by recovering the address
		recoveredAddress, err := getAddressFromSignature(signature, message)
		require.NoError(t, err)
		assert.Equal(t, address.Hex(), recoveredAddress)
	})

	t.Run("PersonalSignFromHex_ValidationErrors", func(t *testing.T) {
		tests := []struct {
			name          string
			message       string
			privateKeyHex string
			wantError     string
		}{
			{
				name:          "empty message",
				message:       "",
				privateKeyHex: testPrivateKey,
				wantError:     "message cannot be empty",
			},
			{
				name:          "empty private key",
				message:       "test message",
				privateKeyHex: "",
				wantError:     "private key hex cannot be empty",
			},
			{
				name:          "invalid hex format",
				message:       "test message",
				privateKeyHex: "0xzzzz",
				wantError:     "invalid private key hex format",
			},
			{
				name:          "invalid private key length",
				message:       "test message",
				privateKeyHex: "0x1234",
				wantError:     "failed to parse private key",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := PersonalSignFromHex(tt.message, tt.privateKeyHex)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			})
		}
	})

	t.Run("createPersonalMessageHash", func(t *testing.T) {
		message := "Test message for hashing"

		// Create hash using utility function
		hash := createPersonalMessageHash(message)
		assert.NotNil(t, hash)
		assert.Len(t, hash, 32) // Hash should be 32 bytes

		// Compare with accounts.TextHash to ensure consistency
		expectedHash := accounts.TextHash([]byte(message))
		assert.Equal(t, expectedHash, hash)

		// Different messages should produce different hashes
		hash2 := createPersonalMessageHash("Different message")
		assert.NotEqual(t, hash, hash2)
	})

	t.Run("VerifyPersonalSignature_Success", func(t *testing.T) {
		message := "Test message for verification"

		// Create a signature
		signature, err := personalSign(message, privateKey)
		require.NoError(t, err)

		// Verify the signature
		isValid, err := VerifyPersonalSignature(message, signature, address.Hex())
		require.NoError(t, err)
		assert.True(t, isValid)
	})

	t.Run("VerifyPersonalSignature_WrongSigner", func(t *testing.T) {
		message := "Test message for verification"

		// Create a signature with the test key
		signature, err := personalSign(message, privateKey)
		require.NoError(t, err)

		// Create a different address
		differentKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		differentAddress := crypto.PubkeyToAddress(differentKey.PublicKey)

		// Verify with wrong address - should fail
		isValid, err := VerifyPersonalSignature(message, signature, differentAddress.Hex())
		require.NoError(t, err)
		assert.False(t, isValid)
	})

	t.Run("VerifyPersonalSignature_ValidationErrors", func(t *testing.T) {
		validMessage := "test message"
		validSignature := "0x" + hex.EncodeToString(make([]byte, 65))
		validAddress := address.Hex()

		tests := []struct {
			name      string
			message   string
			signature string
			address   string
			wantError string
		}{
			{
				name:      "empty message",
				message:   "",
				signature: validSignature,
				address:   validAddress,
				wantError: "message cannot be empty",
			},
			{
				name:      "empty signature",
				message:   validMessage,
				signature: "",
				address:   validAddress,
				wantError: "signature cannot be empty",
			},
			{
				name:      "empty address",
				message:   validMessage,
				signature: validSignature,
				address:   "",
				wantError: "signer address cannot be empty",
			},
			{
				name:      "invalid address format",
				message:   validMessage,
				signature: validSignature,
				address:   "invalid-address",
				wantError: "invalid signer address format",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := VerifyPersonalSignature(tt.message, tt.signature, tt.address)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			})
		}
	})

	t.Run("PersonalSign_EndToEndWorkflow", func(t *testing.T) {
		message := "Complete end-to-end signing and verification test"

		// 1. Sign message with personalSign
		signature1, err := personalSign(message, privateKey)
		require.NoError(t, err)

		// 2. Sign same message with PersonalSignFromHex
		signature2, err := PersonalSignFromHex(message, testPrivateKey)
		require.NoError(t, err)

		// Both signatures should be valid for the same address
		isValid1, err := VerifyPersonalSignature(message, signature1, address.Hex())
		require.NoError(t, err)
		assert.True(t, isValid1)

		isValid2, err := VerifyPersonalSignature(message, signature2, address.Hex())
		require.NoError(t, err)
		assert.True(t, isValid2)

		// Verify with getAddressFromSignature as well
		recovered1, err := getAddressFromSignature(signature1, message)
		require.NoError(t, err)
		assert.True(t, strings.EqualFold(address.Hex(), recovered1))

		recovered2, err := getAddressFromSignature(signature2, message)
		require.NoError(t, err)
		assert.True(t, strings.EqualFold(address.Hex(), recovered2))
	})

	t.Run("verifyAddressWithMessage_Success", func(t *testing.T) {
		message := "Test message for address verification"

		// Create a signature
		signature, err := personalSign(message, privateKey)
		require.NoError(t, err)

		// Verify with correct address
		isValid, err := verifyAddressWithMessage(signature, address.Hex(), message)
		require.NoError(t, err)
		assert.True(t, isValid)
	})

	t.Run("verifyAddressWithMessage_WrongAddress", func(t *testing.T) {
		message := "Test message for address verification"

		// Create a signature with the test key
		signature, err := personalSign(message, privateKey)
		require.NoError(t, err)

		// Create a different address
		differentKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		differentAddress := crypto.PubkeyToAddress(differentKey.PublicKey)

		// Verify with wrong address - should fail
		isValid, err := verifyAddressWithMessage(signature, differentAddress.Hex(), message)
		require.NoError(t, err)
		assert.False(t, isValid)
	})

	t.Run("verifyAddressWithMessage_ValidationErrors", func(t *testing.T) {
		validMessage := "test message"
		validSignature := "0x" + hex.EncodeToString(make([]byte, 65))
		validAddress := address.Hex()

		tests := []struct {
			name      string
			signature string
			address   string
			message   string
			wantError string
		}{
			{
				name:      "empty signature",
				signature: "",
				address:   validAddress,
				message:   validMessage,
				wantError: "signature cannot be empty",
			},
			{
				name:      "empty address",
				signature: validSignature,
				address:   "",
				message:   validMessage,
				wantError: "address cannot be empty",
			},
			{
				name:      "empty message",
				signature: validSignature,
				address:   validAddress,
				message:   "",
				wantError: "message cannot be empty",
			},
			{
				name:      "invalid address format",
				signature: validSignature,
				address:   "invalid-address",
				message:   validMessage,
				wantError: "invalid address format",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := verifyAddressWithMessage(tt.signature, tt.address, tt.message)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantError)
			})
		}
	})
}

func TestVerifyRealAddressWithSignature(t *testing.T) {
	signature := "0x403163eee64372c6cc53777e08cd1360d61b7d7cb18f7390ad5089872de74d124865c46ec9c3c486b320f7450b25403c10587812623daa7eb1e57e7d2ad295b91c"
	address := "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	message := "I am signing into Launchpad at 1756811045"
	verified, err := verifyAddressWithMessage(signature, address, message)
	require.NoError(t, err)
	assert.True(t, verified, "The signature should be valid for the given address and message")
}

// waitForTransaction waits for a transaction to be mined.
func waitForTransaction(client *ethclient.Client, txHash common.Hash, timeout time.Duration) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			receipt, err := client.TransactionReceipt(ctx, txHash)
			if err == nil {
				return receipt, nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
