package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewDeployUniswapTool(db *database.Database, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("deploy_uniswap",
		mcp.WithDescription("Deploy Uniswap infrastructure contracts (factory, router, WETH) with version selection. Creates a transaction session and returns a URL where users can sign and deploy the contracts."),
		mcp.WithString("version",
			mcp.Required(),
			mcp.Description("Uniswap version to deploy (v2 only currently supported)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		version, err := request.RequireString("version")
		if err != nil {
			return nil, fmt.Errorf("version parameter is required: %w", err)
		}

		// Validate version
		if err := utils.ValidateUniswapVersion(version); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Get active chain configuration
		activeChain, err := db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Check if Uniswap is already deployed for this chain
		existingDeployment, err := db.GetUniswapDeploymentByChain(activeChain.ChainType, activeChain.ChainID)
		if err == nil && existingDeployment != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Uniswap %s is already deployed on %s (Chain ID: %s)",
				existingDeployment.Version, activeChain.ChainType, activeChain.ChainID)), nil
		}

		// Prepare deployment data based on version
		var deploymentData interface{}
		var metadata []utils.DeploymentMetadata

		switch version {
		case "v2":
			v2Data, err := utils.DeployV2Uniswap(activeChain.ChainType, activeChain.ChainID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to prepare V2 deployment: %v", err)), nil
			}
			deploymentData = v2Data
			metadata = v2Data.Metadata
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Unsupported version: %s", version)), nil
		}

		// Create Uniswap deployment record
		uniswapDeployment := &models.UniswapDeployment{
			Version:   version,
			ChainType: activeChain.ChainType,
			ChainID:   activeChain.ChainID,
			Status:    "pending",
		}

		if err := db.CreateUniswapDeployment(uniswapDeployment); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating Uniswap deployment record: %v", err)), nil
		}

		// Prepare session data
		sessionData := map[string]interface{}{
			"uniswap_deployment_id": uniswapDeployment.ID,
			"version":               version,
			"deployment_data":       deploymentData,
			"metadata":              metadata,
		}

		sessionDataJSON, err := json.Marshal(sessionData)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error encoding session data: %v", err)), nil
		}

		// Create transaction session
		sessionID, err := db.CreateTransactionSession(
			"deploy_uniswap",
			activeChain.ChainType,
			activeChain.ChainID,
			string(sessionDataJSON),
		)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error creating transaction session: %v", err)), nil
		}

		// Generate signing URL
		signingURL := fmt.Sprintf("http://localhost:%d/deploy-uniswap/%s", serverPort, sessionID)

		// Prepare result with metadata
		result := map[string]interface{}{
			"uniswap_deployment_id": uniswapDeployment.ID,
			"session_id":            sessionID,
			"signing_url":           signingURL,
			"version":               version,
			"chain_type":            activeChain.ChainType,
			"chain_id":              activeChain.ChainID,
			"metadata":              metadata,
			"message":               fmt.Sprintf("Uniswap %s deployment session created. Use the signing URL to connect wallet and deploy contracts.", version),
		}

		// Add gas estimates
		if version == "v2" {
			gasEstimates := utils.EstimateUniswapV2DeploymentGas()
			result["gas_estimates"] = gasEstimates
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(fmt.Sprintf("Uniswap %s deployment URL generated: %s", version, signingURL)),
				mcp.NewTextContent("Please render the url using markdown link format"),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
