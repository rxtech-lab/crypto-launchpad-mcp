package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
)

func NewSelectChainTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("select_chain",
		mcp.WithDescription("Select blockchain for token operations. Can select by uuid (recommended). Sets the selection as active in database."),
		mcp.WithString("chain_type",
			mcp.Description("The blockchain type to select (ethereum or solana). Legacy parameter."),
		),
		mcp.WithString("uuid",
			mcp.Description("The uuid from the database. Use this for precise selection."),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chainType := request.GetString("chain_type", "")
		chainIDStr := request.GetString("uuid", "")

		// Validate that at least one parameter is provided
		if chainType == "" && chainIDStr == "" {
			return mcp.NewToolResultError("Either chain_type or chain_id parameter is required"), nil
		}
		uuid, err := strconv.ParseUint(chainIDStr, 10, 32)
		// Set the active chain by uuid
		if err := db.SetActiveChainByID(uint(uuid)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error setting active chain: %v", err)), nil
		}
		// Get the active chain to return current state
		activeChain, err := db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error getting active chain: %v", err)), nil
		}

		result := map[string]interface{}{
			"id":         activeChain.ID,
			"chain_type": activeChain.ChainType,
			"name":       activeChain.Name,
			"rpc":        activeChain.RPC,
			"chain_id":   activeChain.ChainID,
			"is_active":  activeChain.IsActive,
			"message":    fmt.Sprintf("Successfully selected %s blockchain (ID: %d)", activeChain.Name, activeChain.ID),
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
