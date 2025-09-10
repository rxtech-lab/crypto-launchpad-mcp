package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/go-playground/validator/v10"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type EvmService interface {
	GetContractDeploymentTransactionWithContractCode(args ContractDeploymentWithContractCodeTransactionArgs) (models.TransactionDeployment, abi.ABI, error)
	GetContractDeploymentTransactionWithBytecodeAndAbi(args ContractDeploymentWithBytecodeAndAbiTransactionArgs) (models.TransactionDeployment, abi.ABI, error)
	GetTransactionData(args GetTransactionDataArgs) (string, error)
	GetContractFunctionCallTransaction(args GetContractFunctionCallTransactionArgs) (models.TransactionDeployment, error)
}

type evmService struct {
	validator *validator.Validate
}

func NewEvmService() EvmService {
	validator := validator.New()
	return &evmService{validator: validator}
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
	compilationResult, err := utils.CompileSolidity("0.8.24", contractCode)
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
