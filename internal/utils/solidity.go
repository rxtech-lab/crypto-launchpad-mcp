package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/rxtech-lab/launchpad-mcp/internal/contracts"
	"github.com/rxtech-lab/solc-go"
)

type CompilationResult struct {
	Bytecode map[string]string
	Abi      map[string]any
}

func CompileSolidity(version string, code string) (CompilationResult, error) {
	compiler, err := solc.NewWithVersion(version)
	if err != nil {
		return CompilationResult{}, err
	}

	opts := solc.CompileOptions{
		ImportCallback: func(u string) solc.ImportResult {
			if contractPath, ok := strings.CutPrefix(u, "@openzeppelin/"); ok {
				embeddedPath := filepath.Join("openzeppelin-contracts", contractPath)
				// Read the contract content from the embedded filesystem
				content, err := contracts.OpenZeppelinFS.ReadFile(embeddedPath)
				if err != nil {
					return solc.ImportResult{
						Error: fmt.Sprintf("OpenZeppelin contract %s not found: %v", u, err),
					}
				}

				return solc.ImportResult{
					Contents: string(content),
				}
			}

			return solc.ImportResult{
				Error: fmt.Sprintf("Import %s not found", u),
			}
		},
	}
	result, err := compiler.CompileWithOptions(&solc.Input{
		Language: "Solidity",
		Sources: map[string]solc.SourceIn{
			"contract.sol": {
				Content: code,
			},
		},
		Settings: solc.Settings{
			OutputSelection: map[string]map[string][]string{
				"*": {
					"*": []string{"abi", "evm.bytecode"},
				},
			},
		},
	}, &opts)
	if err != nil {
		return CompilationResult{}, err
	}

	if len(result.Errors) > 0 {
		return CompilationResult{}, errors.New(fmt.Sprintf("compilation errors: %v", result.Errors))
	}

	bytecodeMap := make(map[string]string)
	abiMap := make(map[string]any)

	for fileName, contract := range result.Contracts {
		if fileName != "contract.sol" {
			continue
		}
		for contractName, contract := range contract {
			bytecode := contract.EVM.Bytecode.Object
			abi := contract.ABI // Store the full ABI array, not just the first element

			bytecodeMap[contractName] = bytecode
			abiMap[contractName] = abi
		}
	}

	return CompilationResult{
		Bytecode: bytecodeMap,
		Abi:      abiMap,
	}, nil
}

// EncodeConstructorArgs encodes constructor arguments for ERC20 contracts and appends them to bytecode
func EncodeConstructorArgs(bytecode, tokenName, tokenSymbol string) (string, error) {
	// Remove 0x prefix if present
	cleanBytecode := strings.TrimPrefix(bytecode, "0x")

	// Create a simple ABI for string constructor (name, symbol)
	constructorABI := abi.Arguments{
		{Type: abi.Type{T: abi.StringTy}}, // name
		{Type: abi.Type{T: abi.StringTy}}, // symbol
	}

	// Pack the constructor arguments
	encodedArgs, err := constructorABI.Pack(tokenName, tokenSymbol)
	if err != nil {
		return "", fmt.Errorf("failed to encode constructor arguments: %w", err)
	}

	// Convert to hex and append to bytecode
	encodedArgsHex := hex.EncodeToString(encodedArgs)

	// Return with 0x prefix
	return "0x" + cleanBytecode + encodedArgsHex, nil
}
