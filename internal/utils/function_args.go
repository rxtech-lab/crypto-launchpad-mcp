package utils

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/rxtech-lab/launchpad-mcp/internal/constants"
)

func EncodeFunctionArgsToStringMapWithStringABI(functionName string, args []any, contractABI string) (string, error) {
	parsedABI, err := abi.JSON(strings.NewReader(contractABI))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	return EncodeFunctionArgsToStringMap(functionName, args, parsedABI)
}

// EncodeFunctionArgsToStringMap generates a JSON string representing a map of argument names to their stringified values for a given contract function or constructor.
// Special handling: if an argument value is the string representation of MAX_UINT256, it will be replaced with the literal "MAX_UINT256" in the output.
// Example usage:
//
//	args = [routerAddress, constants.MaxUint256.String()]
//	abi = erc20ABI
//	output = {"routerAddress": "0x1234567890123456789012345678901234567890", "value": "MAX_UINT256"}
func EncodeFunctionArgsToStringMap(functionName string, args []any, contractABI abi.ABI) (string, error) {
	if len(args) == 0 {
		return "{}", nil
	}

	var inputs abi.Arguments
	var err error

	// Handle constructor case
	if functionName == "constructor" {
		if contractABI.Constructor.Inputs == nil {
			return "", fmt.Errorf("no constructor found in ABI")
		}
		inputs = contractABI.Constructor.Inputs
	} else {
		// Find the specified method by name
		method, exists := contractABI.Methods[functionName]
		if !exists {
			return "", fmt.Errorf("method '%s' not found in ABI", functionName)
		}
		inputs = method.Inputs
	}

	// Validate argument count
	if len(args) != len(inputs) {
		return "", fmt.Errorf("expected %d arguments for %s, got %d", len(inputs), functionName, len(args))
	}

	result := make(map[string]string)

	for i, arg := range args {
		input := inputs[i]
		argName := input.Name
		if argName == "" {
			argName = fmt.Sprintf("arg%d", i)
		}

		argValue, err := formatArgValue(arg)
		if err != nil {
			return "", fmt.Errorf("failed to format argument %s: %w", argName, err)
		}

		result[argName] = argValue
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result to JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// formatArgValue formats an argument value to string, with special handling for MAX_UINT256
func formatArgValue(arg any) (string, error) {
	switch v := arg.(type) {
	case string:
		// Check if it's MAX_UINT256 string representation
		if v == constants.MaxUint256.String() {
			return "MAX_UINT256", nil
		}
		return v, nil
	case *big.Int:
		// Check if it's MAX_UINT256
		if v.Cmp(constants.MaxUint256) == 0 {
			return "MAX_UINT256", nil
		}
		return v.String(), nil
	case common.Address:
		return v.Hex(), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), nil
	case float32, float64:
		return fmt.Sprintf("%g", v), nil
	case []byte:
		return "0x" + strings.ToUpper(fmt.Sprintf("%x", v)), nil
	default:
		// Try to convert to string using fmt
		return fmt.Sprintf("%v", v), nil
	}
}
