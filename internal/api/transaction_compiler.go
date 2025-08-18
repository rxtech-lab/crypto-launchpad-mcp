package api

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

// generateTransactionData creates transaction data on-the-fly with compilation
func (s *APIServer) generateTransactionData(deployment *models.Deployment, template *models.Template, activeChain *models.Chain) map[string]interface{} {
	// Extract contract name for compilation
	contractName := s.extractContractName(template.TemplateCode)

	// Base transaction data
	transactionData := map[string]interface{}{
		"deployment_id":    deployment.ID,
		"template_code":    template.TemplateCode,
		"contract_name":    contractName,
		"token_name":       deployment.TokenName,
		"token_symbol":     deployment.TokenSymbol,
		"deployer_address": deployment.DeployerAddress,
		"chain_type":       activeChain.ChainType,
		"chain_id":         activeChain.ChainID,
		"rpc":              activeChain.RPC,
		"compiled":         false,
	}

	// Compile contract for Ethereum
	if activeChain.ChainType == "ethereum" && contractName != "" {
		// Replace template placeholders with actual values
		processedCode := s.replaceTemplatePlaceholders(template.TemplateCode, deployment.TokenName, deployment.TokenSymbol)

		// Compile the contract using utils/solidity.go
		result, err := utils.CompileSolidity("0.8.20", processedCode)
		if err == nil {
			// Extract the bytecode for the specific contract name
			if bytecode, exists := result.Bytecode[contractName]; exists {
				// Ensure bytecode has 0x prefix for proper hex encoding
				if bytecode != "" && !strings.HasPrefix(bytecode, "0x") {
					bytecode = "0x" + bytecode
				}

				// For ERC20 contracts, encode constructor arguments (name, symbol)
				if deployment.TokenName != "" && deployment.TokenSymbol != "" {
					// Use utils to encode constructor arguments and append to bytecode
					if encodedBytecode, err := utils.EncodeConstructorArgs(bytecode, deployment.TokenName, deployment.TokenSymbol); err == nil {
						bytecode = encodedBytecode
						log.Printf("Constructor arguments encoded for deployment %d", deployment.ID)
					} else {
						log.Printf("Failed to encode constructor arguments for deployment %d: %v", deployment.ID, err)
						// Continue with raw bytecode - let the client handle it
					}
				}

				transactionData["bytecode"] = bytecode
			} else {
				log.Printf("Contract %s not found in compilation result for deployment %d", contractName, deployment.ID)
				transactionData["compilation_error"] = fmt.Sprintf("Contract %s not found in compilation result", contractName)
			}

			// Extract the ABI for the specific contract name
			if abi, exists := result.Abi[contractName]; exists {
				transactionData["abi"] = abi
			}

			transactionData["compiled"] = true
			transactionData["processed_code"] = processedCode
		} else {
			log.Printf("Contract compilation failed for deployment %d: %v", deployment.ID, err)
			transactionData["compilation_error"] = err.Error()
		}
	}

	return transactionData
}

// extractContractName extracts the contract name from Solidity source code
func (s *APIServer) extractContractName(sourceCode string) string {
	// Look for contract definition
	contractRegex := regexp.MustCompile(`contract\s+(\w+)`)
	matches := contractRegex.FindStringSubmatch(sourceCode)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// replaceTemplatePlaceholders replaces common template placeholders with actual values
func (s *APIServer) replaceTemplatePlaceholders(templateCode, tokenName, tokenSymbol string) string {
	// Replace common placeholders
	code := strings.ReplaceAll(templateCode, "{{TOKEN_NAME}}", tokenName)
	code = strings.ReplaceAll(code, "{{TOKEN_SYMBOL}}", tokenSymbol)
	code = strings.ReplaceAll(code, "{TOKEN_NAME}", tokenName)
	code = strings.ReplaceAll(code, "{TOKEN_SYMBOL}", tokenSymbol)
	code = strings.ReplaceAll(code, "$TOKEN_NAME", tokenName)
	code = strings.ReplaceAll(code, "$TOKEN_SYMBOL", tokenSymbol)

	// Replace placeholder values in constructor calls
	code = strings.ReplaceAll(code, "\"MyToken\"", fmt.Sprintf("\"%s\"", tokenName))
	code = strings.ReplaceAll(code, "\"MTK\"", fmt.Sprintf("\"%s\"", tokenSymbol))
	code = strings.ReplaceAll(code, "'MyToken'", fmt.Sprintf("'%s'", tokenName))
	code = strings.ReplaceAll(code, "'MTK'", fmt.Sprintf("'%s'", tokenSymbol))

	return code
}
