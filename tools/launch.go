package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

func NewLaunchTool(db *database.Database, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("launch",
		mcp.WithDescription("Generate deployment URL with contract compilation and signing interface. Creates a transaction session and returns a URL where users can sign and deploy the contract."),
		mcp.WithString("template_id",
			mcp.Required(),
			mcp.Description("ID of the template to deploy"),
		),
		mcp.WithString("token_name",
			mcp.Required(),
			mcp.Description("Name of the token to deploy"),
		),
		mcp.WithString("token_symbol",
			mcp.Required(),
			mcp.Description("Symbol of the token to deploy"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		templateIDStr, err := request.RequireString("template_id")
		if err != nil {
			return nil, fmt.Errorf("template_id parameter is required: %w", err)
		}

		tokenName, err := request.RequireString("token_name")
		if err != nil {
			return nil, fmt.Errorf("token_name parameter is required: %w", err)
		}

		tokenSymbol, err := request.RequireString("token_symbol")
		if err != nil {
			return nil, fmt.Errorf("token_symbol parameter is required: %w", err)
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

		// Get active chain configuration
		activeChain, err := db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Validate that template chain type matches active chain
		if template.ChainType != activeChain.ChainType {
			return mcp.NewToolResultError(fmt.Sprintf("Template chain type (%s) doesn't match active chain (%s)", template.ChainType, activeChain.ChainType)), nil
		}

		// Create deployment record
		deployment := &models.Deployment{
			TemplateID:      uint(templateID),
			TokenName:       tokenName,
			TokenSymbol:     tokenSymbol,
			ChainType:       activeChain.ChainType,
			ChainID:         activeChain.ChainID,
			DeployerAddress: "", // Will be set by frontend when wallet connects
			Status:          "pending",
		}

		if err := db.CreateDeployment(deployment); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating deployment record: %v", err)), nil
		}

		// Prepare transaction data for signing
		transactionData := map[string]interface{}{
			"deployment_id":    deployment.ID,
			"template_code":    template.TemplateCode,
			"token_name":       tokenName,
			"token_symbol":     tokenSymbol,
			"deployer_address": "", // Will be populated by frontend wallet connection
			"chain_type":       activeChain.ChainType,
			"chain_id":         activeChain.ChainID,
			"rpc":              activeChain.RPC,
		}

		transactionDataJSON, err := json.Marshal(transactionData)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error encoding transaction data: %v", err)), nil
		}

		// Create transaction session
		sessionID, err := db.CreateTransactionSession(
			"deploy",
			activeChain.ChainType,
			activeChain.ChainID,
			string(transactionDataJSON),
		)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating transaction session: %v", err)), nil
		}

		// Generate signing URL
		signingURL := fmt.Sprintf("http://localhost:%d/deploy/%s", serverPort, sessionID)

		result := map[string]interface{}{
			"deployment_id": deployment.ID,
			"session_id":    sessionID,
			"signing_url":   signingURL,
			"template_name": template.Name,
			"token_name":    tokenName,
			"token_symbol":  tokenSymbol,
			"chain_type":    activeChain.ChainType,
			"chain_id":      activeChain.ChainID,
			"message":       "Deployment session created. Use the signing URL to connect wallet and deploy contract.",
			"instructions":  "1. Open the signing URL in your browser\n2. Connect your wallet using EIP-6963\n3. Review the transaction details\n4. Sign and send the deployment transaction",
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Deployment URL generated: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
