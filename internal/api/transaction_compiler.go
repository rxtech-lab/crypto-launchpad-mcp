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
			transactionData["bytecode"] = result.Bytecode
			transactionData["abi"] = result.Abi
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
