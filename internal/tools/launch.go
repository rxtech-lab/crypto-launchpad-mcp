package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewLaunchTool(db *database.Database, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("launch",
		mcp.WithDescription("Generate deployment URL with contract compilation and signing interface. Creates a transaction session and returns a URL where users can sign and deploy the contract using template parameter values."),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("ID of the template to deploy"),
		),
		mcp.WithString("template_values",
			mcp.Required(),
			mcp.Description("JSON object with runtime values for template parameters (e.g., {\"TokenName\": \"MyToken\", \"TokenSymbol\": \"MTK\"})"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateIDStr, err := request.RequireString("template_id")
		if err != nil {
			return nil, fmt.Errorf("template_id parameter is required: %w", err)
		}

		templateValuesStr, err := request.RequireString("template_values")
		if err != nil {
			return nil, fmt.Errorf("template_values parameter is required: %w", err)
		}

		// Parse template values
		var templateValues models.JSON
		if err := json.Unmarshal([]byte(templateValuesStr), &templateValues); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid template_values JSON: %v", err)), nil
		}

		templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid template_id: %v", err)), nil
		}

		// Get template
		template, err := db.GetTemplateByID(uint(templateID))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Template not found: %v", err)), nil
		}

		// Validate template values against metadata
		if template.Metadata != nil && len(template.Metadata) > 0 {
			for key := range template.Metadata {
				if _, exists := templateValues[key]; !exists {
					return mcp.NewToolResultError(fmt.Sprintf("Missing required template parameter: %s", key)), nil
				}
			}
		}

		// Validate that no extra parameters are provided
		if template.Metadata != nil {
			for key := range templateValues {
				if _, exists := template.Metadata[key]; !exists {
					return mcp.NewToolResultError(fmt.Sprintf("Unknown template parameter: %s", key)), nil
				}
			}
		}

		// Get active chain configuration
		activeChain, err := db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Validate that template chain type matches active chain
		if template.ChainType != activeChain.ChainType {
			return mcp.NewToolResultError(fmt.Sprintf("Template chain type (%s) doesn't match active chain (%s)", template.ChainType, activeChain.ChainType)), nil
		}

		// Check for Uniswap deployment if this is an Ethereum ERC-20 token
		var uniswapWarning string
		if activeChain.ChainType == "ethereum" && isERC20Template(template.TemplateCode) {
			_, err := db.GetUniswapDeploymentByChain(activeChain.ChainType, activeChain.ChainID)
			if err != nil {
				// No Uniswap deployment found - suggest deploying it
				uniswapWarning = "Note: No Uniswap infrastructure found for this chain. Consider using the deploy_uniswap tool after token deployment to enable trading."
			}
		}

		// Extract contract name for compilation
		contractName := extractContractName(template.TemplateCode)
		if contractName == "" {
			return mcp.NewToolResultError("Could not extract contract name from template code"), nil
		}

		// Compile contract for validation and bytecode generation (Ethereum only)
		var compilationResult *utils.CompilationResult
		if activeChain.ChainType == "ethereum" {
			// Replace template placeholders with actual values
			processedCode, err := renderContractTemplate(template.TemplateCode, templateValues)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Template rendering failed: %v", err)), nil
			}

			// Compile the contract using utils/solidity.go
			result, err := utils.CompileSolidity("0.8.20", processedCode)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Contract compilation failed: %v", err)), nil
			}
			compilationResult = &result
		}

		// Create deployment record
		deployment := &models.Deployment{
			TemplateID:      uint(templateID),
			ChainID:         activeChain.ID,
			TemplateValues:  templateValues,
			DeployerAddress: "", // Will be set by frontend when wallet connects
			Status:          "pending",
		}

		// For backward compatibility, set TokenName and TokenSymbol if they exist in templateValues
		if tokenName, ok := templateValues["TokenName"].(string); ok {
			deployment.TokenName = tokenName
		}
		if tokenSymbol, ok := templateValues["TokenSymbol"].(string); ok {
			deployment.TokenSymbol = tokenSymbol
		}

		if err := db.CreateDeployment(deployment); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating deployment record: %v", err)), nil
		}

		// Prepare minimal session data (compilation will be done on-demand)
		sessionData := map[string]interface{}{
			"deployment_id": deployment.ID,
		}

		sessionDataJSON, err := json.Marshal(sessionData)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error encoding session data: %v", err)), nil
		}

		// Create transaction session
		sessionID, err := db.CreateTransactionSession(
			"deploy",
			activeChain.ChainType,
			activeChain.ChainID,
			string(sessionDataJSON),
		)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating transaction session: %v", err)), nil
		}

		// Generate signing URL
		signingURL := fmt.Sprintf("http://localhost:%d/deploy/%s", serverPort, sessionID)

		// Prepare comprehensive result
		result := map[string]interface{}{
			"deployment_id":   deployment.ID,
			"session_id":      sessionID,
			"signing_url":     signingURL,
			"template_name":   template.Name,
			"contract_name":   contractName,
			"template_values": templateValues,
			"chain_type":      activeChain.ChainType,
			"chain_id":        activeChain.ChainID,
			"message":         "Deployment session created. Use the signing URL to connect wallet and deploy contract.",
		}

		// Include backward compatible token fields if they exist
		if tokenName, ok := templateValues["TokenName"].(string); ok {
			result["token_name"] = tokenName
		}
		if tokenSymbol, ok := templateValues["TokenSymbol"].(string); ok {
			result["token_symbol"] = tokenSymbol
		}

		// Add Uniswap warning if applicable
		if uniswapWarning != "" {
			result["uniswap_warning"] = uniswapWarning
		}

		// Add compilation information
		if compilationResult != nil {
			result["compilation_status"] = "success"
			result["contract_compiled"] = true
			result["compiled_contracts"] = len(compilationResult.Bytecode)
			var contractNames []string
			for contractName := range compilationResult.Bytecode {
				contractNames = append(contractNames, contractName)
			}
			result["contract_names"] = contractNames

			// Calculate contract size in bytes for the main contract
			mainBytecode := compilationResult.Bytecode[contractName]
			if mainBytecode != "" {
				bytecodeSize := len(mainBytecode) / 2 // Convert hex to bytes
				result["contract_size_bytes"] = bytecodeSize
				result["contract_size_limit"] = 24576 // EIP-170 limit
			}
		} else {
			result["compilation_status"] = "skipped"
			result["contract_compiled"] = false
			if activeChain.ChainType == "ethereum" {
				result["compilation_note"] = "Solidity compiler not available - contract will be compiled client-side"
			}
		}

		resultJSON, _ := json.Marshal(result)
		// Format success message based on compilation status
		successMessage := "Deployment URL generated"
		if compilationResult != nil {
			successMessage += " (contract pre-compiled and validated)"
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(successMessage + ": " + signingURL),
				mcp.NewTextContent("Please render the url using markdown link format"),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}

// renderContractTemplate renders the template code with provided values using Go template engine
func renderContractTemplate(templateCode string, values models.JSON) (string, error) {
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

// extractContractName extracts the contract name from Solidity source code
func extractContractName(sourceCode string) string {
	// Look for contract definition
	contractRegex := regexp.MustCompile(`contract\s+(\w+)`)
	matches := contractRegex.FindStringSubmatch(sourceCode)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// replaceTemplatePlaceholders replaces common template placeholders with actual values
func replaceTemplatePlaceholders(templateCode, tokenName, tokenSymbol string) string {
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

// isERC20Template checks if a template is an ERC-20 token template
func isERC20Template(templateCode string) bool {
	// Check for common ERC-20 patterns
	erc20Patterns := []string{
		"function transfer(",
		"function transferFrom(",
		"function balanceOf(",
		"function approve(",
		"function allowance(",
		"ERC20",
		"IERC20",
	}

	codeLower := strings.ToLower(templateCode)

	// Count matches - need at least 3 ERC-20 patterns to be confident
	matches := 0
	for _, pattern := range erc20Patterns {
		if strings.Contains(codeLower, strings.ToLower(pattern)) {
			matches++
		}
	}

	return matches >= 3
}
