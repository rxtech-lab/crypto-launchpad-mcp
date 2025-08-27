package utils

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func EncodeContractConstructorArgs(abiJSON string, args []any) ([]byte, error) {
	if len(args) == 0 {
		return []byte{}, nil
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	constructor := parsedABI.Constructor
	// Check if constructor exists by checking if it has inputs
	// In go-ethereum, Constructor is always present but may have empty inputs

	processedArgs, err := processConstructorArgs(constructor.Inputs, args)
	if err != nil {
		return nil, fmt.Errorf("failed to process constructor arguments: %w", err)
	}

	encodedArgs, err := constructor.Inputs.Pack(processedArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to encode constructor arguments: %w", err)
	}

	return encodedArgs, nil
}

func processConstructorArgs(inputs abi.Arguments, args []any) ([]any, error) {
	if len(args) != len(inputs) {
		return nil, fmt.Errorf("expected %d arguments, got %d", len(inputs), len(args))
	}

	processedArgs := make([]any, len(args))
	for i, input := range inputs {
		processedArg, err := processArg(input.Type, args[i])
		if err != nil {
			return nil, fmt.Errorf("failed to process argument %d (%s): %w", i, input.Name, err)
		}
		processedArgs[i] = processedArg
	}
	return processedArgs, nil
}

func processArg(argType abi.Type, value any) (any, error) {
	switch argType.T {
	case abi.AddressTy:
		switch v := value.(type) {
		case string:
			if !common.IsHexAddress(v) {
				return nil, fmt.Errorf("invalid address: %s", v)
			}
			return common.HexToAddress(v), nil
		case common.Address:
			return v, nil
		default:
			return nil, fmt.Errorf("unsupported address type: %T", value)
		}

	case abi.UintTy, abi.IntTy:
		switch v := value.(type) {
		case string:
			bigInt, ok := new(big.Int).SetString(v, 10)
			if !ok {
				bigInt, ok = new(big.Int).SetString(v, 16)
				if !ok {
					return nil, fmt.Errorf("invalid integer: %s", v)
				}
			}
			return bigInt, nil
		case *big.Int:
			return v, nil
		case int64:
			return big.NewInt(v), nil
		case int:
			return big.NewInt(int64(v)), nil
		case uint64:
			return new(big.Int).SetUint64(v), nil
		case float64:
			return big.NewInt(int64(v)), nil
		default:
			return nil, fmt.Errorf("unsupported integer type: %T", value)
		}

	case abi.BoolTy:
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			return strings.ToLower(v) == "true", nil
		default:
			return nil, fmt.Errorf("unsupported bool type: %T", value)
		}

	case abi.StringTy:
		switch v := value.(type) {
		case string:
			return v, nil
		default:
			return nil, fmt.Errorf("unsupported string type: %T", value)
		}

	case abi.BytesTy, abi.FixedBytesTy:
		switch v := value.(type) {
		case string:
			if strings.HasPrefix(v, "0x") {
				v = v[2:]
			}
			bytes, err := hex.DecodeString(v)
			if err != nil {
				return nil, fmt.Errorf("invalid hex string: %w", err)
			}
			if argType.T == abi.FixedBytesTy && len(bytes) != argType.Size {
				return nil, fmt.Errorf("expected %d bytes, got %d", argType.Size, len(bytes))
			}
			if argType.T == abi.FixedBytesTy {
				var fixedBytes [32]byte
				copy(fixedBytes[:], bytes)
				return fixedBytes, nil
			}
			return bytes, nil
		case []byte:
			if argType.T == abi.FixedBytesTy {
				if len(v) != argType.Size {
					return nil, fmt.Errorf("expected %d bytes, got %d", argType.Size, len(v))
				}
				var fixedBytes [32]byte
				copy(fixedBytes[:], v)
				return fixedBytes, nil
			}
			return v, nil
		default:
			return nil, fmt.Errorf("unsupported bytes type: %T", value)
		}

	case abi.ArrayTy, abi.SliceTy:
		slice, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("expected array, got %T", value)
		}
		processedSlice := make([]any, len(slice))
		for i, elem := range slice {
			processed, err := processArg(*argType.Elem, elem)
			if err != nil {
				return nil, fmt.Errorf("failed to process array element %d: %w", i, err)
			}
			processedSlice[i] = processed
		}
		return processedSlice, nil

	default:
		return nil, fmt.Errorf("unsupported argument type: %v", argType)
	}
}

func EncodeContractFunctionCall(abiJSON, functionName string, args []any) (string, error) {
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	method, ok := parsedABI.Methods[functionName]
	if !ok {
		return "", fmt.Errorf("function %s not found in ABI", functionName)
	}

	// Process arguments to match ABI types
	processedArgs := make([]any, len(args))
	for i, input := range method.Inputs {
		if i >= len(args) {
			return "", fmt.Errorf("missing argument for %s", input.Name)
		}
		processedArg, err := processArg(input.Type, args[i])
		if err != nil {
			return "", fmt.Errorf("failed to process argument %d (%s): %w", i, input.Name, err)
		}
		processedArgs[i] = processedArg
	}

	// Pack the method call
	encodedData, err := parsedABI.Pack(functionName, processedArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to encode function call: %w", err)
	}

	return "0x" + hex.EncodeToString(encodedData), nil
}

func BuildDeploymentTransactionData(bytecode string, encodedConstructorArgs []byte) string {
	if strings.HasPrefix(bytecode, "0x") {
		bytecode = bytecode[2:]
	}

	txData := bytecode + hex.EncodeToString(encodedConstructorArgs)
	return "0x" + txData
}
