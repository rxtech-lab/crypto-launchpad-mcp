package utils

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	contractName       = "TestToken"
	TestAccountAddress = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
)

func TestEncodeContractConstructorArgs(t *testing.T) {
	// Compile the contract
	contractCode := `
		// SPDX-License-Identifier: MIT
		pragma solidity ^0.8.0;

		contract TestToken {
			string public name;
			string public symbol;
			uint256 public totalSupply;
			address public owner;

			constructor(string memory _name, string memory _symbol, uint256 _totalSupply, address _owner) {
				name = _name;
				symbol = _symbol;
				totalSupply = _totalSupply;
				owner = _owner;
			}
		}
	`
	result, err := CompileSolidity("0.8.20", contractCode)
	require.NoError(t, err)
	require.NotEmpty(t, result.Bytecode)
	require.NotEmpty(t, result.Abi)

	// Get the compiled contract ABI
	abiInterface, ok := result.Abi[contractName]
	require.True(t, ok, "Contract ABI should exist")

	// Convert ABI to JSON string
	abiJSON, err := json.Marshal(abiInterface)
	require.NoError(t, err)

	// Test encoding constructor arguments
	// Using Anvil test account #0 address
	args := []any{
		"Test Token",                // name
		"TEST",                      // symbol
		"1000000000000000000000000", // totalSupply (1M tokens with 18 decimals)
		TestAccountAddress,          // Anvil test account #0 address
	}

	encodedArgs, err := EncodeContractConstructorArgs(string(abiJSON), args)
	require.NoError(t, err)
	assert.NotEmpty(t, encodedArgs)

	// Build deployment transaction data
	bytecode := result.Bytecode[contractName]
	txData := BuildDeploymentTransactionData(bytecode, encodedArgs)
	assert.True(t, strings.HasPrefix(txData, "0x"))
	assert.True(t, len(txData) > len(bytecode)) // Should be longer than just bytecode

	// Verify the encoded args are appended to bytecode
	encodedArgsHex := hex.EncodeToString(encodedArgs)
	assert.True(t, strings.HasSuffix(strings.TrimPrefix(txData, "0x"), encodedArgsHex))
}

func TestEncodeContractConstructorArgs_EmptyConstructor(t *testing.T) {
	// Contract without constructor parameters
	contractCode := `
		// SPDX-License-Identifier: MIT
		pragma solidity ^0.8.0;

		contract TestToken {
			uint256 public value;

			function setValue(uint256 _value) public {
				value = _value;
			}

			function getValue() public view returns (uint256) {
				return value;
			}
		}
	`

	// Compile the contract
	result, err := CompileSolidity("0.8.20", contractCode)
	require.NoError(t, err)
	require.NotEmpty(t, result.Bytecode)
	require.NotEmpty(t, result.Abi)

	// Get the compiled contract ABI
	abiInterface, ok := result.Abi[contractName]
	require.True(t, ok, "Contract ABI should exist")

	// Convert ABI to JSON string
	abiJSON, err := json.Marshal(abiInterface)
	require.NoError(t, err)

	// Test encoding with empty constructor arguments
	args := []any{}

	encodedArgs, err := EncodeContractConstructorArgs(string(abiJSON), args)
	require.NoError(t, err)
	assert.Empty(t, encodedArgs, "Empty constructor should produce empty encoded args")

	// Build deployment transaction data
	bytecode := result.Bytecode[contractName]
	txData := BuildDeploymentTransactionData(bytecode, encodedArgs)
	assert.True(t, strings.HasPrefix(txData, "0x"))

	// For empty constructor, txData should be just the bytecode with 0x prefix
	expectedTxData := "0x" + strings.TrimPrefix(bytecode, "0x")
	assert.Equal(t, expectedTxData, txData)
}
