package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

func NewListChainsTool(chainService services.ChainService) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("list_chains",
		mcp.WithDescription("List all available blockchain chains with their configurations"),
		mcp.WithString("chain_type",
			mcp.Description("Filter by chain type (ethereum, solana). Optional."),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chainType := request.GetString("chain_type", "")

		chains, err := chainService.ListChains()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error listing chains: %v", err)), nil
		}

		// Filter by chain type if specified
		var filteredChains []interface{}
		for _, chain := range chains {
			if chainType == "" || string(chain.ChainType) == chainType {
				filteredChains = append(filteredChains, map[string]interface{}{
					"id":         chain.ID,
					"name":       chain.Name,
					"chain_type": chain.ChainType,
					"rpc":        chain.RPC,
					"chain_id":   chain.NetworkID,
					"is_active":  chain.IsActive,
					"created_at": chain.CreatedAt,
					"updated_at": chain.UpdatedAt,
				})
			}
		}

		// Format the response
		response := map[string]interface{}{
			"chains": filteredChains,
			"total":  len(filteredChains),
		}

		// Find active chain if any
		for _, chain := range chains {
			if chain.IsActive {
				response["active_chain"] = map[string]interface{}{
					"id":         chain.ID,
					"name":       chain.Name,
					"chain_type": chain.ChainType,
					"rpc":        chain.RPC,
					"chain_id":   chain.NetworkID,
					"is_active":  chain.IsActive,
				}
				break
			}
		}

		responseJSON, _ := json.MarshalIndent(response, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent(string(responseJSON)),
			},
		}, nil
	}

	return tool, handler
}
