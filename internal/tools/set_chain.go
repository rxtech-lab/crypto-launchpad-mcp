package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

// fetchChainIDFromRPC fetches the chain ID from an Ethereum RPC endpoint
func fetchChainIDFromRPC(rpcURL string) (string, error) {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_chainId",
		"params":  []interface{}{},
		"id":      1,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal RPC payload: %w", err)
	}

	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to make RPC request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("RPC request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read RPC response: %w", err)
	}

	var result struct {
		Result string `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal RPC response: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("RPC error: %s", result.Error.Message)
	}

	// Convert hex chain ID to decimal string
	if strings.HasPrefix(result.Result, "0x") {
		chainIDInt, err := strconv.ParseInt(result.Result[2:], 16, 64)
		if err != nil {
			return "", fmt.Errorf("failed to parse chain ID hex: %w", err)
		}
		return strconv.FormatInt(chainIDInt, 10), nil
	}

	return result.Result, nil
}

func NewSetChainTool(chainService services.ChainService) (mcp.Tool, server.ToolHandlerFunc) {
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
			mcp.Description("The chain ID (e.g., '1' for Ethereum mainnet, '11155111' for Sepolia). If not provided, will be auto-detected from RPC endpoint."),
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

		chainID := request.GetString("chain_id", "")

		// If chain_id not provided and chain type is ethereum, try to fetch it from RPC
		if chainID == "" && chainType == "ethereum" {
			fetchedChainID, err := fetchChainIDFromRPC(rpc)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Could not auto-detect chain ID from RPC: %v. Please provide chain_id parameter.", err)), nil
			}
			chainID = fetchedChainID
		}

		// For Solana, chain_id is still required
		if chainID == "" && chainType == "solana" {
			return mcp.NewToolResultError("chain_id parameter is required for Solana chains"), nil
		}

		if chainID == "" {
			return mcp.NewToolResultError("chain_id parameter is required or could not be auto-detected"), nil
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
			return mcp.NewToolResultError("Invalid chain_type. Supported values: ethereum, solana"), nil
		}

		// Check if chain configuration already exists
		chains, err := chainService.ListChains()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error listing chains: %v", err)), nil
		}

		var existingChain *models.Chain
		for _, chain := range chains {
			if string(chain.ChainType) == chainType {
				existingChain = &chain
				break
			}
		}

		if existingChain != nil {
			// Update existing chain configuration
			if err := chainService.UpdateChainConfig(chainType, rpc, chainID); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error updating chain configuration: %v", err)), nil
			}
		} else {
			// Create new chain configuration
			newChain := &models.Chain{
				ChainType: models.TransactionChainType(chainType),
				RPC:       rpc,
				NetworkID: chainID,
				Name:      name,
				IsActive:  false,
			}
			if err := chainService.CreateChain(newChain); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error creating chain configuration: %v", err)), nil
			}
		}

		message := fmt.Sprintf("Successfully configured %s blockchain", chainType)
		if request.GetString("chain_id", "") == "" && chainType == "ethereum" {
			message += fmt.Sprintf(" (auto-detected chain ID: %s)", chainID)
		}

		result := map[string]interface{}{
			"chain_type": chainType,
			"rpc":        rpc,
			"chain_id":   chainID,
			"name":       name,
			"message":    message,
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
