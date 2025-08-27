package api

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"

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
		"template_values":  deployment.TemplateValues,
		"deployer_address": deployment.DeployerAddress,
		"chain_type":       activeChain.ChainType,
		"chain_id":         activeChain.NetworkID,
		"rpc":              activeChain.RPC,
		"compiled":         false,
	}

	// Add backward compatible token fields if they exist in template values
	if deployment.TemplateValues != nil {
		if tokenName, ok := deployment.TemplateValues["TokenName"].(string); ok {
			transactionData["token_name"] = tokenName
		}
		if tokenSymbol, ok := deployment.TemplateValues["TokenSymbol"].(string); ok {
			transactionData["token_symbol"] = tokenSymbol
		}
	}

	// Fallback to deprecated fields if template values are not available
	if deployment.TokenName != "" {
		transactionData["token_name"] = deployment.TokenName
	}
	if deployment.TokenSymbol != "" {
		transactionData["token_symbol"] = deployment.TokenSymbol
	}

	// Compile contract for Ethereum
	if activeChain.ChainType == "ethereum" && contractName != "" {
		// Render template with actual values
		var processedCode string
		var err error

		if deployment.TemplateValues != nil {
			processedCode, err = s.renderContractTemplate(template.TemplateCode, deployment.TemplateValues)
		} else {
			// Fallback for old deployments without template values
			fallbackValues := models.JSON{
				"TokenName":   deployment.TokenName,
				"TokenSymbol": deployment.TokenSymbol,
			}
			processedCode, err = s.renderContractTemplate(template.TemplateCode, fallbackValues)
		}

		if err != nil {
			log.Printf("Template rendering failed for deployment %d: %v", deployment.ID, err)
			transactionData["compilation_error"] = err.Error()
			return transactionData
		}

		// Compile the contract using utils/solidity.go
		result, err := utils.CompileSolidity("0.8.20", processedCode)
		if err == nil {
			// Extract the bytecode for the specific contract name
			if bytecode, exists := result.Bytecode[contractName]; exists {
				// Ensure bytecode has 0x prefix for proper hex encoding
				if bytecode != "" && !strings.HasPrefix(bytecode, "0x") {
					bytecode = "0x" + bytecode
				}

				// For ERC20 contracts, encode constructor arguments if available
				var tokenName, tokenSymbol string
				if deployment.TemplateValues != nil {
					if name, ok := deployment.TemplateValues["TokenName"].(string); ok {
						tokenName = name
					}
					if symbol, ok := deployment.TemplateValues["TokenSymbol"].(string); ok {
						tokenSymbol = symbol
					}
				} else {
					// Fallback to deprecated fields
					tokenName = deployment.TokenName
					tokenSymbol = deployment.TokenSymbol
				}

				if tokenName != "" && tokenSymbol != "" {
					// Use utils to encode constructor arguments and append to bytecode
					if encodedBytecode, err := utils.EncodeConstructorArgs(bytecode, tokenName, tokenSymbol); err == nil {
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

// renderContractTemplate renders the contract template code with provided values using Go template engine
func (s *APIServer) renderContractTemplate(templateCode string, values models.JSON) (string, error) {
	tmpl, err := template.New("contract").Parse(templateCode)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
