package utils

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rxtech-lab/launchpad-mcp/internal/constants"
)

func TestEncodeFunctionArgs(t *testing.T) {
	// Create a sample ERC20 ABI with approve method
	erc20ABI := createERC20ABI(t)

	tests := []struct {
		name         string
		functionName string
		args         []any
		contractABI  abi.ABI
		expected     map[string]string
		expectError  bool
	}{
		{
			name:         "approve with router address and MAX_UINT256 string",
			functionName: "approve",
			args: []any{
				"0x1234567890123456789012345678901234567890",
				constants.MaxUint256.String(),
			},
			contractABI: erc20ABI,
			expected: map[string]string{
				"spender": "0x1234567890123456789012345678901234567890",
				"amount":  "MAX_UINT256",
			},
			expectError: false,
		},
		{
			name:         "approve with router address and MAX_UINT256 big.Int",
			functionName: "approve",
			args: []any{
				common.HexToAddress("0x1234567890123456789012345678901234567890"),
				constants.MaxUint256,
			},
			contractABI: erc20ABI,
			expected: map[string]string{
				"spender": "0x1234567890123456789012345678901234567890",
				"amount":  "MAX_UINT256",
			},
			expectError: false,
		},
		{
			name:         "approve with regular amount",
			functionName: "approve",
			args: []any{
				"0x1234567890123456789012345678901234567890",
				"1000000000000000000", // 1 ether in wei
			},
			contractABI: erc20ABI,
			expected: map[string]string{
				"spender": "0x1234567890123456789012345678901234567890",
				"amount":  "1000000000000000000",
			},
			expectError: false,
		},
		{
			name:         "approve with big.Int amount",
			functionName: "approve",
			args: []any{
				common.HexToAddress("0x1234567890123456789012345678901234567890"),
				big.NewInt(1000000000000000000),
			},
			contractABI: erc20ABI,
			expected: map[string]string{
				"spender": "0x1234567890123456789012345678901234567890",
				"amount":  "1000000000000000000",
			},
			expectError: false,
		},
		{
			name:         "empty args",
			functionName: "approve",
			args:         []any{},
			contractABI:  erc20ABI,
			expected:     map[string]string{},
			expectError:  false,
		},
		{
			name:         "nonexistent method",
			functionName: "nonexistent",
			args: []any{
				"0x1234567890123456789012345678901234567890",
				"1000000000000000000",
			},
			contractABI: erc20ABI,
			expected:    nil,
			expectError: true,
		},
		{
			name:         "wrong argument count",
			functionName: "approve",
			args: []any{
				"0x1234567890123456789012345678901234567890",
				"1000000000000000000",
				"extra_arg",
			},
			contractABI: erc20ABI,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeFunctionArgsToStringMap(tt.functionName, tt.args, tt.contractABI)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if len(tt.args) == 0 {
				assert.Equal(t, "{}", result)
				return
			}

			// Parse JSON result
			var resultMap map[string]string
			err = json.Unmarshal([]byte(result), &resultMap)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, resultMap)
		})
	}
}

func TestFormatArgValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "string address",
			input:    "0x1234567890123456789012345678901234567890",
			expected: "0x1234567890123456789012345678901234567890",
		},
		{
			name:     "MAX_UINT256 string",
			input:    constants.MaxUint256.String(),
			expected: "MAX_UINT256",
		},
		{
			name:     "MAX_UINT256 big.Int",
			input:    constants.MaxUint256,
			expected: "MAX_UINT256",
		},
		{
			name:     "regular big.Int",
			input:    big.NewInt(12345),
			expected: "12345",
		},
		{
			name:     "common.Address",
			input:    common.HexToAddress("0x1234567890123456789012345678901234567890"),
			expected: "0x1234567890123456789012345678901234567890",
		},
		{
			name:     "bool true",
			input:    true,
			expected: "true",
		},
		{
			name:     "bool false",
			input:    false,
			expected: "false",
		},
		{
			name:     "int",
			input:    42,
			expected: "42",
		},
		{
			name:     "uint64",
			input:    uint64(42),
			expected: "42",
		},
		{
			name:     "float64",
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "bytes",
			input:    []byte{0xaa, 0xbb, 0xcc},
			expected: "0xAABBCC",
		},
		{
			name:     "slice",
			input:    []int{1, 2, 3},
			expected: "[1 2 3]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatArgValue(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEncodeFunctionArgsWithDifferentMethodTypes(t *testing.T) {
	// Test with a transfer method (2 args) and balanceOf method (1 arg)
	transferABI := createTransferABI(t)

	tests := []struct {
		name         string
		functionName string
		args         []any
		contractABI  abi.ABI
		expected     map[string]string
		expectError  bool
	}{
		{
			name:         "balanceOf with single address",
			functionName: "balanceOf",
			args: []any{
				"0x1234567890123456789012345678901234567890",
			},
			contractABI: transferABI,
			expected: map[string]string{
				"account": "0x1234567890123456789012345678901234567890",
			},
			expectError: false,
		},
		{
			name:         "transfer with address and amount",
			functionName: "transfer",
			args: []any{
				"0x1234567890123456789012345678901234567890",
				"1000000000000000000",
			},
			contractABI: transferABI,
			expected: map[string]string{
				"to":     "0x1234567890123456789012345678901234567890",
				"amount": "1000000000000000000",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeFunctionArgsToStringMap(tt.functionName, tt.args, tt.contractABI)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Parse JSON result
			var resultMap map[string]string
			err = json.Unmarshal([]byte(result), &resultMap)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, resultMap)
		})
	}
}

func TestEncodeFunctionArgsWithUnnamedParameters(t *testing.T) {
	// Create ABI with unnamed parameters
	unnamedABI := createUnnamedABI(t)

	args := []any{
		"0x1234567890123456789012345678901234567890",
		"1000000000000000000",
	}

	result, err := EncodeFunctionArgsToStringMap("someFunction", args, unnamedABI)
	require.NoError(t, err)

	var resultMap map[string]string
	err = json.Unmarshal([]byte(result), &resultMap)
	require.NoError(t, err)

	expected := map[string]string{
		"arg0": "0x1234567890123456789012345678901234567890",
		"arg1": "1000000000000000000",
	}
	assert.Equal(t, expected, resultMap)
}

func TestEncodeFunctionArgsWithConstructor(t *testing.T) {
	// Test constructor arguments
	constructorABI := createConstructorABI(t)

	tests := []struct {
		name         string
		functionName string
		args         []any
		contractABI  abi.ABI
		expected     map[string]string
		expectError  bool
	}{
		{
			name:         "constructor with name, symbol, and initial supply",
			functionName: "constructor",
			args: []any{
				"Test Token",
				"TEST",
				"1000000000000000000000000", // 1 million tokens
			},
			contractABI: constructorABI,
			expected: map[string]string{
				"name":          "Test Token",
				"symbol":        "TEST",
				"initialSupply": "1000000000000000000000000",
			},
			expectError: false,
		},
		{
			name:         "constructor with MAX_UINT256",
			functionName: "constructor",
			args: []any{
				"Max Token",
				"MAX",
				constants.MaxUint256.String(),
			},
			contractABI: constructorABI,
			expected: map[string]string{
				"name":          "Max Token",
				"symbol":        "MAX",
				"initialSupply": "MAX_UINT256",
			},
			expectError: false,
		},
		{
			name:         "constructor with wrong argument count",
			functionName: "constructor",
			args: []any{
				"Test Token",
				"TEST",
			},
			contractABI: constructorABI,
			expected:    nil,
			expectError: true,
		},
		{
			name:         "constructor with empty args",
			functionName: "constructor",
			args:         []any{},
			contractABI:  constructorABI,
			expected:     map[string]string{},
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeFunctionArgsToStringMap(tt.functionName, tt.args, tt.contractABI)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			if len(tt.args) == 0 {
				assert.Equal(t, "{}", result)
				return
			}

			// Parse JSON result
			var resultMap map[string]string
			err = json.Unmarshal([]byte(result), &resultMap)
			require.NoError(t, err)

			assert.Equal(t, tt.expected, resultMap)
		})
	}
}

func TestEncodeFunctionArgsConstructorNoABI(t *testing.T) {
	// Test constructor with no constructor in ABI
	noConstructorABI := createERC20ABI(t) // This ABI has no constructor

	args := []any{
		"Test Token",
		"TEST",
	}

	result, err := EncodeFunctionArgsToStringMap("constructor", args, noConstructorABI)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no constructor found in ABI")
	assert.Empty(t, result)
}

// Helper functions to create test ABIs

func createERC20ABI(t *testing.T) abi.ABI {
	abiJSON := `[
		{
			"constant": false,
			"inputs": [
				{
					"name": "spender",
					"type": "address"
				},
				{
					"name": "amount",
					"type": "uint256"
				}
			],
			"name": "approve",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"type": "function"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	require.NoError(t, err)
	return parsedABI
}

func createTransferABI(t *testing.T) abi.ABI {
	abiJSON := `[
		{
			"constant": true,
			"inputs": [
				{
					"name": "account",
					"type": "address"
				}
			],
			"name": "balanceOf",
			"outputs": [
				{
					"name": "",
					"type": "uint256"
				}
			],
			"type": "function"
		},
		{
			"constant": false,
			"inputs": [
				{
					"name": "to",
					"type": "address"
				},
				{
					"name": "amount",
					"type": "uint256"
				}
			],
			"name": "transfer",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"type": "function"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	require.NoError(t, err)
	return parsedABI
}

func createUnnamedABI(t *testing.T) abi.ABI {
	abiJSON := `[
		{
			"constant": false,
			"inputs": [
				{
					"name": "",
					"type": "address"
				},
				{
					"name": "",
					"type": "uint256"
				}
			],
			"name": "someFunction",
			"outputs": [
				{
					"name": "",
					"type": "bool"
				}
			],
			"type": "function"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	require.NoError(t, err)
	return parsedABI
}

func createConstructorABI(t *testing.T) abi.ABI {
	abiJSON := `[
		{
			"inputs": [
				{
					"name": "name",
					"type": "string"
				},
				{
					"name": "symbol",
					"type": "string"
				},
				{
					"name": "initialSupply",
					"type": "uint256"
				}
			],
			"stateMutability": "nonpayable",
			"type": "constructor"
		},
		{
			"constant": true,
			"inputs": [],
			"name": "name",
			"outputs": [
				{
					"name": "",
					"type": "string"
				}
			],
			"type": "function"
		}
	]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	require.NoError(t, err)
	return parsedABI
}
