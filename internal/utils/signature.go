package utils

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func GenerateMessage() string {
	return fmt.Sprintf("I am signing into Launchpad at %d", time.Now().Unix())
}

// VerifyTransactionOwnershipBySignature verifies if the provided signature corresponds to the owner of the transaction identified by txHash.
// It will fetch the transaction sender address and verify the signature was created by that address.
func VerifyTransactionOwnershipBySignature(rpcUrl, txHash, signature, message string) (bool, error) {
	// Validate inputs
	if rpcUrl == "" {
		return false, fmt.Errorf("RPC URL cannot be empty")
	}
	if txHash == "" {
		return false, fmt.Errorf("transaction hash cannot be empty")
	}
	if signature == "" {
		return false, fmt.Errorf("signature cannot be empty")
	}

	// Encode the message to hex format (same as ethers.js does)
	encodedMessage := "0x" + hex.EncodeToString([]byte(message))

	// Get transaction sender from blockchain
	txSender, err := getTransactionSender(txHash, rpcUrl)
	if err != nil {
		return false, fmt.Errorf("failed to get transaction sender: %w", err)
	}
	// Verify the signature was created by the transaction sender
	return verifyAddressWithMessage(signature, txSender, encodedMessage)
}

// getAddressFromSignature recovers the Ethereum address from a signature and message
func getAddressFromSignature(signature, message string) (string, error) {
	// Validate signature format
	if !strings.HasPrefix(signature, "0x") {
		return "", fmt.Errorf("signature must start with 0x")
	}

	// Remove 0x prefix and validate length
	sigBytes := strings.TrimPrefix(signature, "0x")
	if len(sigBytes) != 130 { // 65 bytes * 2 hex chars = 130
		return "", fmt.Errorf("signature must be 65 bytes (130 hex characters)")
	}

	// Decode hex signature
	sigData, err := hexutil.Decode(signature)
	if err != nil {
		return "", fmt.Errorf("failed to decode signature: %w", err)
	}

	// Signature should be 65 bytes: r(32) + s(32) + v(1)
	if len(sigData) != 65 {
		return "", fmt.Errorf("signature must be exactly 65 bytes")
	}

	// Create message hash using Ethereum's personal message format
	messageHash := accounts.TextHash([]byte(message))

	// Recover public key from signature
	// Note: go-ethereum expects v to be 0 or 1, but MetaMask returns 27 or 28
	// We need to adjust the recovery ID
	if sigData[64] >= 27 {
		sigData[64] -= 27
	}

	publicKey, err := crypto.SigToPub(messageHash, sigData)
	if err != nil {
		return "", fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert public key to address
	address := crypto.PubkeyToAddress(*publicKey)

	return address.Hex(), nil
}

// getTransactionSender retrieves the sender address of a transaction from the blockchain
func getTransactionSender(txHash, rpcURL string) (string, error) {
	// Validate transaction hash format
	if !strings.HasPrefix(txHash, "0x") {
		return "", fmt.Errorf("transaction hash must start with 0x")
	}
	if len(txHash) != 66 { // 32 bytes * 2 hex chars + 0x = 66
		return "", fmt.Errorf("transaction hash must be 32 bytes (66 hex characters including 0x)")
	}

	// Create RPC client
	client := NewRPCClient(rpcURL)

	// Get transaction by hash
	response, err := client.Call("eth_getTransactionByHash", []interface{}{txHash})
	if err != nil {
		return "", fmt.Errorf("failed to get transaction: %w", err)
	}

	if response.Result == nil {
		return "", fmt.Errorf("transaction not found: %s", txHash)
	}

	// Parse the transaction data
	txData, ok := response.Result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid transaction data format")
	}

	// Extract the 'from' field
	fromAddr, exists := txData["from"]
	if !exists {
		return "", fmt.Errorf("transaction does not have 'from' field")
	}

	fromAddrStr, ok := fromAddr.(string)
	if !ok {
		return "", fmt.Errorf("'from' field is not a string")
	}

	// Validate address format
	if !common.IsHexAddress(fromAddrStr) {
		return "", fmt.Errorf("invalid address format: %s", fromAddrStr)
	}

	return fromAddrStr, nil
}

// personalSign signs a message using Ethereum's personal message format with the provided private key.
// This creates a signature that can be verified using personal_ecRecover or eth_sign.
func personalSign(message string, privateKey *ecdsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", fmt.Errorf("private key cannot be nil")
	}
	if message == "" {
		return "", fmt.Errorf("message cannot be empty")
	}

	// Create message hash using Ethereum's personal message format
	messageHash := accounts.TextHash([]byte(message))

	// Sign the message hash
	signature, err := crypto.Sign(messageHash, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign message: %w", err)
	}

	// Convert to hex string with 0x prefix
	return "0x" + hex.EncodeToString(signature), nil
}

// PersonalSignFromHex signs a message using a private key provided as a hex string.
// This is a convenience function that parses the private key and calls personalSign.
func PersonalSignFromHex(message string, privateKeyHex string) (string, error) {
	if message == "" {
		return "", fmt.Errorf("message cannot be empty")
	}
	if privateKeyHex == "" {
		return "", fmt.Errorf("private key hex cannot be empty")
	}

	// Remove 0x prefix if present
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")

	// Parse private key from hex
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key hex format: %w", err)
	}

	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %w", err)
	}

	return personalSign(message, privateKey)
}

// createPersonalMessageHash creates the hash for a personal message using Ethereum's format.
// This is used internally by personalSign and can be useful for testing or advanced use cases.
func createPersonalMessageHash(message string) []byte {
	return accounts.TextHash([]byte(message))
}

// VerifyPersonalSignature verifies that a signature was created for the given message by the given address.
// This is an alternative to VerifyTransactionOwnershipBySignature for cases where you have the signer address directly.
func VerifyPersonalSignature(message string, signature string, signerAddress string) (bool, error) {
	if message == "" {
		return false, fmt.Errorf("message cannot be empty")
	}
	if signature == "" {
		return false, fmt.Errorf("signature cannot be empty")
	}
	if signerAddress == "" {
		return false, fmt.Errorf("signer address cannot be empty")
	}

	// Validate address format
	if !common.IsHexAddress(signerAddress) {
		return false, fmt.Errorf("invalid signer address format: %s", signerAddress)
	}

	// Recover address from signature
	recoveredAddress, err := getAddressFromSignature(signature, message)
	if err != nil {
		return false, fmt.Errorf("failed to recover address from signature: %w", err)
	}

	// Compare addresses (case-insensitive)
	return strings.EqualFold(recoveredAddress, signerAddress), nil
}

// verifyAddressWithMessage verifies that a signature was created by the given address for the given message.
// This is a helper method used internally by transaction ownership verification.
func verifyAddressWithMessage(signature string, address string, message string) (bool, error) {
	if signature == "" {
		return false, fmt.Errorf("signature cannot be empty")
	}
	if address == "" {
		return false, fmt.Errorf("address cannot be empty")
	}
	if message == "" {
		return false, fmt.Errorf("message cannot be empty")
	}

	// Validate address format
	if !common.IsHexAddress(address) {
		return false, fmt.Errorf("invalid address format: %s", address)
	}

	// Use the existing VerifyPersonalSignature method
	return VerifyPersonalSignature(message, signature, address)
}
