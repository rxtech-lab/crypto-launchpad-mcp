package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

func NewQueryBalanceTool(db interface{}, serverPort int) (mcp.Tool, server.ToolHandlerFunc) {
	tool := mcp.NewTool("query_balance",
		mcp.WithDescription("Query wallet balance for native tokens and ERC-20 tokens. Can return results directly or display in browser interface."),
		mcp.WithString("wallet_address",
			mcp.Description("Wallet address to query balance for (required when show_browser=false)"),
		),
		mcp.WithBoolean("show_browser",
			mcp.Required(),
			mcp.Description("If true, display balance in web interface. If false, return balance directly in response."),
		),
		mcp.WithString("token_address",
			mcp.Description("ERC-20 token contract address (optional, for token balance queries)"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		showBrowser := request.GetBool("show_browser", false)
		walletAddress := request.GetString("wallet_address", "")
		tokenAddress := request.GetString("token_address", "")

		// Get active chain configuration
		activeChain, err := db.GetActiveChain()
		if err != nil {
			return mcp.NewToolResultError("No active chain selected. Please use select_chain tool first"), nil
		}

		// Validate chain type
		if activeChain.ChainType != "ethereum" {
			return mcp.NewToolResultError("Balance queries currently only supported on Ethereum-compatible chains"), nil
		}

		if showBrowser {
			// Web mode - create session and return URL
			return handleBrowserMode(db, serverPort, activeChain, walletAddress, tokenAddress)
		} else {
			// Direct mode - require wallet address and return balance immediately
			if walletAddress == "" {
				return mcp.NewToolResultError("wallet_address is required when show_browser=false"), nil
			}
			return handleDirectMode(activeChain, walletAddress, tokenAddress)
		}
	}

	return tool, handler
}

// handleBrowserMode creates a session for web-based balance display
func handleBrowserMode(db interface{}, serverPort int, activeChain *models.Chain, walletAddress, tokenAddress string) (*mcp.CallToolResult, error) {
	// Prepare session data
	sessionData := map[string]interface{}{
		"query_type":     "balance",
		"wallet_address": walletAddress, // Can be empty - will be set by frontend
		"token_address":  tokenAddress,  // Optional
		"chain_type":     activeChain.ChainType,
		"chain_id":       activeChain.NetworkID,
		"rpc_url":        activeChain.RPC,
	}

	sessionDataJSON, err := json.Marshal(sessionData)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error encoding session data: %v", err)), nil
	}

	// Create transaction session
	sessionID, err := db.CreateTransactionSession(
		"balance_query",
		activeChain.ChainType,
		activeChain.NetworkID,
		string(sessionDataJSON),
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error creating session: %v", err)), nil
	}

	// Generate URL
	balanceURL := fmt.Sprintf("http://localhost:%d/balance/%s", serverPort, sessionID)

	result := map[string]interface{}{
		"session_id":  sessionID,
		"balance_url": balanceURL,
		"chain_type":  activeChain.ChainType,
		"chain_id":    activeChain.NetworkID,
		"mode":        "browser",
		"message":     "Balance query session created. Use the URL to view wallet balance in browser.",
	}

	if walletAddress != "" {
		result["wallet_address"] = walletAddress
	}
	if tokenAddress != "" {
		result["token_address"] = tokenAddress
	}

	resultJSON, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(fmt.Sprintf("Balance query URL generated: %s", balanceURL)),
			mcp.NewTextContent("Please render the url using markdown link format"),
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}

// handleDirectMode queries balance immediately and returns in response
func handleDirectMode(activeChain *models.Chain, walletAddress, tokenAddress string) (*mcp.CallToolResult, error) {
	result := map[string]interface{}{
		"wallet_address": walletAddress,
		"chain_type":     activeChain.ChainType,
		"chain_id":       activeChain.NetworkID,
		"mode":           "direct",
	}

	// Query native balance
	nativeBalance, err := utils.QueryNativeBalance(activeChain.RPC, walletAddress, string(activeChain.ChainType))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to query native balance: %v", err)), nil
	}

	result["native_balance"] = nativeBalance

	// Query token balance if token address provided
	if tokenAddress != "" {
		tokenBalance, err := utils.QueryERC20Balance(activeChain.RPC, tokenAddress, walletAddress)
		if err != nil {
			// Don't fail completely, just note the error
			result["token_balance_error"] = fmt.Sprintf("Failed to query token balance: %v", err)
		} else {
			result["token_balance"] = tokenBalance
		}
	}

	resultJSON, _ := json.Marshal(result)

	// Format human-readable summary
	summary := fmt.Sprintf("Balance for %s on %s (Chain ID: %s):\n", walletAddress, activeChain.ChainType, activeChain.NetworkID)
	summary += fmt.Sprintf("• Native Balance: %s\n", nativeBalance.FormattedBalance)

	if tokenAddress != "" && result["token_balance"] != nil {
		tokenBal := result["token_balance"].(*utils.ERC20BalanceResult)
		summary += fmt.Sprintf("• Token Balance: %s\n", tokenBal.FormattedBalance)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(summary),
			mcp.NewTextContent(string(resultJSON)),
		},
	}, nil
}
