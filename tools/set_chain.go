package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

func NewSetChainTool(db *database.Database) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("set_chain",
		mcp.WithDescription("Configure target blockchain with RPC endpoint and chain ID. Creates or updates chain configuration in database."),
		mcp.WithString("chain_type",
			mcp.Required(),
			mcp.Description("The blockchain type (ethereum or solana)"),
		),
		mcp.WithString("rpc",
			mcp.Required(),
			mcp.Description("The RPC endpoint URL for the blockchain"),
		),
		mcp.WithString("chain_id",
			mcp.Required(),
			mcp.Description("The chain ID (e.g., '1' for Ethereum mainnet, '11155111' for Sepolia)"),
		),
		mcp.WithString("name",
			mcp.Description("Optional name for the chain configuration (e.g., 'Ethereum Mainnet', 'Solana Devnet')"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		chainType, err := request.RequireString("chain_type")
		if err != nil {
			return nil, fmt.Errorf("chain_type parameter is required: %w", err)
		}

		rpc, err := request.RequireString("rpc")
		if err != nil {
			return nil, fmt.Errorf("rpc parameter is required: %w", err)
		}

		chainID, err := request.RequireString("chain_id")
		if err != nil {
			return nil, fmt.Errorf("chain_id parameter is required: %w", err)
		}

		name := request.GetString("name", "")
		if name == "" {
			// Generate default name based on chain type and ID
			switch chainType {
			case "ethereum":
				switch chainID {
				case "1":
					name = "Ethereum Mainnet"
				case "11155111":
					name = "Ethereum Sepolia"
				case "5":
					name = "Ethereum Goerli"
				default:
					name = fmt.Sprintf("Ethereum Chain %s", chainID)
				}
			case "solana":
				switch chainID {
				case "mainnet-beta":
					name = "Solana Mainnet"
				case "devnet":
					name = "Solana Devnet"
				case "testnet":
					name = "Solana Testnet"
				default:
					name = fmt.Sprintf("Solana %s", chainID)
				}
			default:
				name = fmt.Sprintf("%s Chain %s", chainType, chainID)
			}
		}

		// Validate chain type
		if chainType != "ethereum" && chainType != "solana" {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent("Invalid chain_type. Supported values: ethereum, solana"),
				},
			}, nil
		}

		// Check if chain configuration already exists
		chains, err := db.ListChains()
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.NewTextContent("Error: "),
					mcp.NewTextContent(fmt.Sprintf("Error listing chains: %v", err)),
				},
			}, nil
		}

		var existingChain *models.Chain
		for _, chain := range chains {
			if chain.ChainType == chainType {
				existingChain = &chain
				break
			}
		}

		if existingChain != nil {
			// Update existing chain configuration
			if err := db.UpdateChainConfig(chainType, rpc, chainID); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent(fmt.Sprintf("Error updating chain configuration: %v", err)),
					},
				}, nil
			}
		} else {
			// Create new chain configuration
			newChain := &models.Chain{
				ChainType: chainType,
				RPC:       rpc,
				ChainID:   chainID,
				Name:      name,
				IsActive:  false,
			}
			if err := db.CreateChain(newChain); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.NewTextContent("Error: "),
						mcp.NewTextContent(fmt.Sprintf("Error creating chain configuration: %v", err)),
					},
				}, nil
			}
		}

		result := map[string]interface{}{
			"chain_type": chainType,
			"rpc":        rpc,
			"chain_id":   chainID,
			"name":       name,
			"message":    fmt.Sprintf("Successfully configured %s blockchain", chainType),
		}

		resultJSON, _ := json.Marshal(result)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.NewTextContent("Success message: "),
				mcp.NewTextContent(string(resultJSON)),
			},
		}, nil
	}

	return tool, handler
}
