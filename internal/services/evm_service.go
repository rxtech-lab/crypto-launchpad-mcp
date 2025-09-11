package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	"github.com/rxtech-lab/launchpad-mcp/internal/constants"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type CallReadOnlyEthereumFunctionArgs struct {
	ContractAddress string `validate:"required,eth_addr"`
	FunctionName    string `validate:"required"`
	FunctionArgs    []any  `validate:"required"`
	Abi             string `validate:"required"`
	RpcURL          string `validate:"required,url"`
	Value           string `validate:"omitempty,number"` // Optional value, defaults to "0"
}

type EvmService interface {
	GetContractDeploymentTransactionWithContractCode(args ContractDeploymentWithContractCodeTransactionArgs) (models.TransactionDeployment, abi.ABI, error)
	GetContractDeploymentTransactionWithBytecodeAndAbi(args ContractDeploymentWithBytecodeAndAbiTransactionArgs) (models.TransactionDeployment, abi.ABI, error)
	GetTransactionData(args GetTransactionDataArgs) (string, error)
	GetContractFunctionCallTransaction(args GetContractFunctionCallTransactionArgs) (models.TransactionDeployment, error)
	GetAllAbiMethods(abi models.JSON) ([]abi.Method, error)
	GetAbiMethod(abi models.JSON, methodName string) (abi.Method, error)
	CallReadOnlyEthereumFunction(args CallReadOnlyEthereumFunctionArgs) ([]any, error)
}

type evmService struct {
	validator *validator.Validate
}

func NewEvmService() EvmService {
	validator := validator.New()
	return &evmService{
		validator: validator,
	}
}

// GetContractDeploymentTransactionWithContractCode returns a transaction deployment for a contract deployment with contract code
func (s *evmService) GetContractDeploymentTransactionWithContractCode(args ContractDeploymentWithContractCodeTransactionArgs) (models.TransactionDeployment, abi.ABI, error) {
	err := s.validator.Struct(args)
	if err != nil {
		return models.TransactionDeployment{}, abi.ABI{}, err
	}

	txData, abiData, err := s.getContractDeploymentTransactionData(args.ContractName, args.ConstructorArgs, args.ContractCode)
	if err != nil {
		return models.TransactionDeployment{}, abi.ABI{}, err
	}

	return models.TransactionDeployment{
		Data:            txData,
		Title:           args.Title,
		Description:     args.Description,
		Value:           args.Value,
		Receiver:        args.Receiver,
		TransactionType: args.TransactionType,
	}, abiData, nil
}

// GetContractDeploymentWithBytecodeAndAbi returns a transaction deployment for a contract deployment with bytecode and abi
func (s *evmService) GetContractDeploymentTransactionWithBytecodeAndAbi(args ContractDeploymentWithBytecodeAndAbiTransactionArgs) (models.TransactionDeployment, abi.ABI, error) {
	err := s.validator.Struct(args)
	if err != nil {
		return models.TransactionDeployment{}, abi.ABI{}, err
	}

	txData, abiData, err := s.getContractDeploymentTransactionDataWithBytecodeAndAbi(args.Abi, args.Bytecode, args.ConstructorArgs)
	if err != nil {
		return models.TransactionDeployment{}, abi.ABI{}, err
	}

	return models.TransactionDeployment{
		Data:            txData,
		Title:           args.Title,
		Description:     args.Description,
		Value:           args.Value,
		Receiver:        args.Receiver,
		TransactionType: args.TransactionType,
	}, abiData, nil
}

// GetTransactionData returns the transaction data interacting with a contract
func (s *evmService) GetTransactionData(args GetTransactionDataArgs) (string, error) {
	err := s.validator.Struct(args)
	if err != nil {
		return "", err
	}

	encodedData, err := utils.EncodeContractFunctionCall(args.Abi, args.FunctionName, args.FunctionArgs)
	if err != nil {
		return "", fmt.Errorf("failed to encode function call: %w", err)
	}

	return encodedData, nil
}

// GetContractFunctionCallTransaction returns a transaction deployment for a contract function call
func (s *evmService) GetContractFunctionCallTransaction(args GetContractFunctionCallTransactionArgs) (models.TransactionDeployment, error) {
	err := s.validator.Struct(args)
	if err != nil {
		return models.TransactionDeployment{}, err
	}

	encodedData, err := utils.EncodeContractFunctionCall(args.Abi, args.FunctionName, args.FunctionArgs)
	if err != nil {
		return models.TransactionDeployment{}, fmt.Errorf("failed to encode function call: %w", err)
	}

	return models.TransactionDeployment{
		Data:            encodedData,
		Title:           args.Title,
		Description:     args.Description,
		Value:           args.Value,
		Receiver:        args.ContractAddress,
		TransactionType: args.TransactionType,
	}, nil
}

func (s *evmService) getContractDeploymentTransactionDataWithBytecodeAndAbi(abiString string, bytecode string, constructorArgs []any) (string, abi.ABI, error) {
	// Encode constructor arguments if provided
	encodedArgs, err := utils.EncodeContractConstructorArgs(abiString, constructorArgs)
	if err != nil {
		return "", abi.ABI{}, fmt.Errorf("failed to encode constructor arguments: %w", err)
	}

	// Build deployment transaction data
	txData := utils.BuildDeploymentTransactionData(bytecode, encodedArgs)

	parsedABI, err := abi.JSON(strings.NewReader(abiString))
	if err != nil {
		return "", abi.ABI{}, fmt.Errorf("failed to parse ABI: %w", err)
	}

	return txData, parsedABI, nil
}

func (s *evmService) getContractDeploymentTransactionData(contractName string, constructorArgs []any, contractCode string) (string, abi.ABI, error) {
	compilationResult, err := utils.CompileSolidity(constants.SolidityCompilerVersion, contractCode)
	if err != nil {
		return "", abi.ABI{}, err
	}

	bytecode, exists := compilationResult.Bytecode[contractName]
	if !exists {
		return "", abi.ABI{}, fmt.Errorf("contract %s not found in compilation result", contractName)
	}

	abiData, exists := compilationResult.Abi[contractName]
	if !exists {
		return "", abi.ABI{}, fmt.Errorf("ABI for contract %s not found", contractName)
	}

	// Convert ABI to JSON string
	abiBytes, err := json.Marshal(abiData)
	if err != nil {
		return "", abi.ABI{}, fmt.Errorf("failed to marshal ABI: %w", err)
	}
	abiJSON := string(abiBytes)

	encodedArgs, err := utils.EncodeContractConstructorArgs(abiJSON, constructorArgs)
	if err != nil {
		return "", abi.ABI{}, fmt.Errorf("failed to encode constructor arguments: %w", err)
	}

	txData := utils.BuildDeploymentTransactionData(bytecode, encodedArgs)

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", abi.ABI{}, fmt.Errorf("failed to parse ABI: %w", err)
	}
	return txData, parsedABI, nil
}

func (s *evmService) GetAllAbiMethods(abiJSON models.JSON) ([]abi.Method, error) {
	// Extract ABI string from the JSON
	abiString := ""
	if abiData, exists := abiJSON["abi"]; exists {
		// If ABI is stored as "abi" field, marshal it
		abiBytes, err := json.Marshal(abiData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ABI data: %w", err)
		}
		abiString = string(abiBytes)
	} else {
		// Otherwise use the whole JSON as ABI string
		abiString = abiJSON.String()
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiString))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Convert map to slice for consistent ordering
	methods := make([]abi.Method, 0, len(parsedABI.Methods))
	for _, method := range parsedABI.Methods {
		methods = append(methods, method)
	}

	return methods, nil
}

func (s *evmService) GetAbiMethod(abiJSON models.JSON, methodName string) (abi.Method, error) {
	// Extract ABI string from the JSON
	abiString := ""
	if abiData, exists := abiJSON["abi"]; exists {
		// If ABI is stored as "abi" field, marshal it
		abiBytes, err := json.Marshal(abiData)
		if err != nil {
			return abi.Method{}, fmt.Errorf("failed to marshal ABI data: %w", err)
		}
		abiString = string(abiBytes)
	} else {
		// Otherwise use the whole JSON as ABI string
		abiString = abiJSON.String()
	}

	parsedABI, err := abi.JSON(strings.NewReader(abiString))
	if err != nil {
		return abi.Method{}, fmt.Errorf("failed to parse ABI: %w", err)
	}

	method, exists := parsedABI.Methods[methodName]
	if !exists {
		return abi.Method{}, fmt.Errorf("method '%s' not found in ABI", methodName)
	}

	return method, nil
}

func (s *evmService) CallReadOnlyEthereumFunction(args CallReadOnlyEthereumFunctionArgs) ([]any, error) {
	// Validate input arguments
	err := s.validator.Struct(args)
	if err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Create Ethereum client from provided RPC URL
	ethClient, err := ethclient.Dial(args.RpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum RPC: %w", err)
	}
	defer ethClient.Close()

	// Parse the contract address
	contractAddress := common.HexToAddress(args.ContractAddress)

	// Parse the ABI
	parsedABI, err := abi.JSON(strings.NewReader(args.Abi))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Check if the method exists
	method, exists := parsedABI.Methods[args.FunctionName]
	if !exists {
		return nil, fmt.Errorf("method '%s' not found in ABI", args.FunctionName)
	}

	// Encode the function call data
	callData, err := utils.EncodeContractFunctionCall(args.Abi, args.FunctionName, args.FunctionArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode function call: %w", err)
	}

	// Convert hex string to bytes
	callDataBytes := common.FromHex(callData)

	// Make the call to the contract
	ctx := context.Background()
	result, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &contractAddress,
		Data: callDataBytes,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result based on the method's output types
	if len(method.Outputs) == 0 {
		return nil, nil
	}

	// For single return value, unpack directly
	if len(method.Outputs) == 1 {
		output := method.Outputs[0]
		switch output.Type.T {
		case abi.StringTy:
			var str string
			err = parsedABI.UnpackIntoInterface(&str, args.FunctionName, result)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack string result: %w", err)
			}
			return []any{str}, nil
		case abi.UintTy, abi.IntTy:
			var num interface{}
			err = parsedABI.UnpackIntoInterface(&num, args.FunctionName, result)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack numeric result: %w", err)
			}
			return []any{fmt.Sprintf("%v", num)}, nil
		case abi.BoolTy:
			var b bool
			err = parsedABI.UnpackIntoInterface(&b, args.FunctionName, result)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack boolean result: %w", err)
			}
			return []any{fmt.Sprintf("%t", b)}, nil
		case abi.AddressTy:
			var addr common.Address
			err = parsedABI.UnpackIntoInterface(&addr, args.FunctionName, result)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack address result: %w", err)
			}
			return []any{addr.Hex()}, nil
		default:
			// For other types, try to unpack as interface{} and convert to string
			var value interface{}
			err = parsedABI.UnpackIntoInterface(&value, args.FunctionName, result)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack result: %w", err)
			}
			return []any{fmt.Sprintf("%v", value)}, nil
		}
	}

	// For multiple return values, unpack into a slice and format as JSON
	results := make([]interface{}, len(method.Outputs))
	err = parsedABI.UnpackIntoInterface(&results, args.FunctionName, result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack multiple results: %w", err)
	}

	return results, nil
}
