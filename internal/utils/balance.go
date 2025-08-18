package utils

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// BalanceResult represents the result of a balance query
type BalanceResult struct {
	Address          string `json:"address"`
	NativeBalance    string `json:"native_balance"`    // ETH balance in wei
	NativeSymbol     string `json:"native_symbol"`     // ETH, BNB, etc.
	FormattedBalance string `json:"formatted_balance"` // Human readable balance
	ChainID          string `json:"chain_id"`
	ChainType        string `json:"chain_type"`
}

// ERC20BalanceResult represents an ERC-20 token balance
type ERC20BalanceResult struct {
	TokenAddress     string `json:"token_address"`
	TokenBalance     string `json:"token_balance"` // Balance in wei
	TokenSymbol      string `json:"token_symbol"`
	TokenDecimals    int    `json:"token_decimals"`
	FormattedBalance string `json:"formatted_balance"` // Human readable balance
}

// QueryNativeBalance queries the native token balance (ETH) for an address
func QueryNativeBalance(rpcURL, address, chainType string) (*BalanceResult, error) {
	if !isValidAddress(address) {
		return nil, fmt.Errorf("invalid address format")
	}

	client := NewRPCClient(rpcURL)

	// Get balance in wei
	response, err := client.Call("eth_getBalance", []interface{}{address, "latest"})
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if response.Result == nil {
		return nil, fmt.Errorf("no balance returned")
	}

	balanceHex, ok := response.Result.(string)
	if !ok {
		return nil, fmt.Errorf("invalid balance format")
	}

	// Convert hex to decimal
	balanceWei, success := new(big.Int).SetString(strings.TrimPrefix(balanceHex, "0x"), 16)
	if !success {
		return nil, fmt.Errorf("failed to parse balance")
	}

	// Convert wei to ETH for display
	ethBalance := new(big.Float).SetInt(balanceWei)
	ethBalance.Quo(ethBalance, big.NewFloat(1e18))
	formattedBalance := ethBalance.Text('f', 6)

	// Determine native symbol based on chain type
	nativeSymbol := "ETH"
	switch chainType {
	case "ethereum":
		nativeSymbol = "ETH"
	case "bsc":
		nativeSymbol = "BNB"
	case "polygon":
		nativeSymbol = "MATIC"
	}

	return &BalanceResult{
		Address:          address,
		NativeBalance:    balanceWei.String(),
		NativeSymbol:     nativeSymbol,
		FormattedBalance: formattedBalance + " " + nativeSymbol,
		ChainType:        chainType,
	}, nil
}

// QueryERC20Balance queries an ERC-20 token balance for an address
func QueryERC20Balance(rpcURL, tokenAddress, walletAddress string) (*ERC20BalanceResult, error) {
	if !isValidAddress(tokenAddress) || !isValidAddress(walletAddress) {
		return nil, fmt.Errorf("invalid address format")
	}

	client := NewRPCClient(rpcURL)

	// ERC-20 balanceOf function signature: 0x70a08231
	// Encode the wallet address (32 bytes, padded)
	paddedAddress := strings.TrimPrefix(walletAddress, "0x")
	if len(paddedAddress) < 40 {
		return nil, fmt.Errorf("invalid wallet address length")
	}

	// Pad to 64 characters (32 bytes)
	for len(paddedAddress) < 64 {
		paddedAddress = "0" + paddedAddress
	}

	data := "0x70a08231" + paddedAddress

	// Call the contract
	response, err := client.Call("eth_call", []interface{}{
		map[string]string{
			"to":   tokenAddress,
			"data": data,
		},
		"latest",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	if response.Result == nil {
		return nil, fmt.Errorf("no result returned from contract call")
	}

	balanceHex, ok := response.Result.(string)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Convert hex to decimal
	balanceWei, success := new(big.Int).SetString(strings.TrimPrefix(balanceHex, "0x"), 16)
	if !success {
		return nil, fmt.Errorf("failed to parse balance")
	}

	// Get token symbol and decimals (optional, with fallbacks)
	symbol := "TOKEN"
	decimals := 18

	// Try to get symbol
	if tokenSymbol, err := getTokenSymbol(client, tokenAddress); err == nil {
		symbol = tokenSymbol
	}

	// Try to get decimals
	if tokenDecimals, err := getTokenDecimals(client, tokenAddress); err == nil {
		decimals = tokenDecimals
	}

	// Format balance
	divisor := new(big.Float).SetFloat64(1)
	for i := 0; i < decimals; i++ {
		divisor.Mul(divisor, big.NewFloat(10))
	}

	tokenBalance := new(big.Float).SetInt(balanceWei)
	tokenBalance.Quo(tokenBalance, divisor)
	formattedBalance := tokenBalance.Text('f', 6)

	return &ERC20BalanceResult{
		TokenAddress:     tokenAddress,
		TokenBalance:     balanceWei.String(),
		TokenSymbol:      symbol,
		TokenDecimals:    decimals,
		FormattedBalance: formattedBalance + " " + symbol,
	}, nil
}

// getTokenSymbol retrieves the symbol of an ERC-20 token
func getTokenSymbol(client *RPCClient, tokenAddress string) (string, error) {
	// ERC-20 symbol function signature: 0x95d89b41
	response, err := client.Call("eth_call", []interface{}{
		map[string]string{
			"to":   tokenAddress,
			"data": "0x95d89b41",
		},
		"latest",
	})
	if err != nil {
		return "", err
	}

	if response.Result == nil {
		return "", fmt.Errorf("no symbol returned")
	}

	symbolHex, ok := response.Result.(string)
	if !ok {
		return "", fmt.Errorf("invalid symbol format")
	}

	// Decode the string from hex
	return decodeStringFromHex(symbolHex), nil
}

// getTokenDecimals retrieves the decimals of an ERC-20 token
func getTokenDecimals(client *RPCClient, tokenAddress string) (int, error) {
	// ERC-20 decimals function signature: 0x313ce567
	response, err := client.Call("eth_call", []interface{}{
		map[string]string{
			"to":   tokenAddress,
			"data": "0x313ce567",
		},
		"latest",
	})
	if err != nil {
		return 0, err
	}

	if response.Result == nil {
		return 0, fmt.Errorf("no decimals returned")
	}

	decimalsHex, ok := response.Result.(string)
	if !ok {
		return 0, fmt.Errorf("invalid decimals format")
	}

	// Convert hex to int
	decimalsInt, err := strconv.ParseInt(strings.TrimPrefix(decimalsHex, "0x"), 16, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse decimals: %w", err)
	}

	return int(decimalsInt), nil
}

// isValidAddress checks if an Ethereum address is valid
func isValidAddress(address string) bool {
	if !strings.HasPrefix(address, "0x") {
		return false
	}
	if len(address) != 42 {
		return false
	}
	// Check if it's a valid hex string
	_, err := strconv.ParseInt(strings.TrimPrefix(address, "0x"), 16, 64)
	return err == nil || len(strings.TrimPrefix(address, "0x")) == 40
}

// decodeStringFromHex decodes a string from hex representation
func decodeStringFromHex(hexStr string) string {
	hexStr = strings.TrimPrefix(hexStr, "0x")
	if len(hexStr) < 128 { // 64 bytes minimum for string encoding
		return ""
	}

	// Skip the first 64 characters (offset and length)
	if len(hexStr) > 128 {
		hexStr = hexStr[128:]
	}

	// Convert pairs of hex characters to bytes
	var result []byte
	for i := 0; i < len(hexStr); i += 2 {
		if i+1 >= len(hexStr) {
			break
		}
		byteVal, err := strconv.ParseInt(hexStr[i:i+2], 16, 8)
		if err != nil {
			break
		}
		if byteVal == 0 {
			break // Stop at null terminator
		}
		result = append(result, byte(byteVal))
	}

	return string(result)
}
