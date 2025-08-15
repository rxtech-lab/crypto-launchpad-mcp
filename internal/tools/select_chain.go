package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
)

func NewSelectChainTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("select_chain",
		mcp.WithDescription("Select blockchain for token operations. Stores the selection in database and sets it as active. Supported chains: ethereum, solana."),
		mcp.WithString("chain_type",
			mcp.Required(),
			mcp.Description("The blockchain type to select (ethereum or solana)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chainType, err := request.RequireString("chain_type")
		if err != nil {
			return nil, fmt.Errorf("chain_type parameter is required: %w", err)
		}

		// Validate chain type
		if chainType != "ethereum" && chainType != "solana" {
			return mcp.NewToolResultError("Invalid chain_type. Supported values: ethereum, solana"), nil
		}

		// Set the active chain
		if err := db.SetActiveChain(chainType); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error setting active chain: %v", err)), nil
		}

		// Get the active chain to return current state
		activeChain, err := db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error getting active chain: %v", err)), nil
		}

		result := map[string]interface{}{
			"chain_type": activeChain.ChainType,
			"name":       activeChain.Name,
			"rpc":        activeChain.RPC,
			"chain_id":   activeChain.ChainID,
			"is_active":  activeChain.IsActive,
			"message":    fmt.Sprintf("Successfully selected %s blockchain", chainType),
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Chain selected successfully: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
