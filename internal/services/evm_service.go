package services

import (
	"encoding/json"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type EvmService interface {
	GetContractDeploymentTransactionWithContractCode(args ContractDeploymentWithContractCodeTransactionArgs) (models.TransactionDeployment, error)
	GetContractDeploymentTransactionWithBytecodeAndAbi(args ContractDeploymentWithBytecodeAndAbiTransactionArgs) (models.TransactionDeployment, error)
	GetTransactionData(args GetTransactionDataArgs) (string, error)
}

type evmService struct {
	validator *validator.Validate
}

func NewEvmService() EvmService {
	validator := validator.New()
	return &evmService{validator: validator}
}

// GetContractDeploymentTransactionWithContractCode returns a transaction deployment for a contract deployment with contract code
func (s *evmService) GetContractDeploymentTransactionWithContractCode(args ContractDeploymentWithContractCodeTransactionArgs) (models.TransactionDeployment, error) {
	err := s.validator.Struct(args)
	if err != nil {
		return models.TransactionDeployment{}, err
	}

	txData, err := s.getContractDeploymentTransactionData(args.ContractName, args.ConstructorArgs, args.ContractCode)
	if err != nil {
		return models.TransactionDeployment{}, err
	}

	return models.TransactionDeployment{
		Data:        txData,
		Title:       args.Title,
		Description: args.Description,
		Value:       args.Value,
		Receiver:    args.Receiver,
	}, nil
}

// GetContractDeploymentWithBytecodeAndAbi returns a transaction deployment for a contract deployment with bytecode and abi
func (s *evmService) GetContractDeploymentTransactionWithBytecodeAndAbi(args ContractDeploymentWithBytecodeAndAbiTransactionArgs) (models.TransactionDeployment, error) {
	err := s.validator.Struct(args)
	if err != nil {
		return models.TransactionDeployment{}, err
	}

	txData, err := s.getContractDeploymentTransactionDataWithBytecodeAndAbi(args.Abi, args.Bytecode, args.ConstructorArgs)
	if err != nil {
		return models.TransactionDeployment{}, err
	}

	return models.TransactionDeployment{
		Data:        txData,
		Title:       args.Title,
		Description: args.Description,
		Value:       args.Value,
		Receiver:    args.Receiver,
	}, nil
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

func (s *evmService) getContractDeploymentTransactionDataWithBytecodeAndAbi(abi string, bytecode string, constructorArgs []any) (string, error) {
	// Encode constructor arguments if provided
	encodedArgs, err := utils.EncodeContractConstructorArgs(abi, constructorArgs)
	if err != nil {
		return "", fmt.Errorf("failed to encode constructor arguments: %w", err)
	}

	// Build deployment transaction data
	txData := utils.BuildDeploymentTransactionData(bytecode, encodedArgs)
	return txData, nil
}

func (s *evmService) getContractDeploymentTransactionData(contractName string, constructorArgs []any, contractCode string) (string, error) {
	compilationResult, err := utils.CompileSolidity("0.8.27", contractCode)
	if err != nil {
		return "", err
	}

	bytecode, exists := compilationResult.Bytecode[contractName]
	if !exists {
		return "", fmt.Errorf("contract %s not found in compilation result", contractName)
	}

	abiData, exists := compilationResult.Abi[contractName]
	if !exists {
		return "", fmt.Errorf("ABI for contract %s not found", contractName)
	}

	// Convert ABI to JSON string
	abiBytes, err := json.Marshal(abiData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal ABI: %w", err)
	}
	abiJSON := string(abiBytes)

	encodedArgs, err := utils.EncodeContractConstructorArgs(abiJSON, constructorArgs)
	if err != nil {
		return "", fmt.Errorf("failed to encode constructor arguments: %w", err)
	}

	txData := utils.BuildDeploymentTransactionData(bytecode, encodedArgs)
	return txData, nil
}
