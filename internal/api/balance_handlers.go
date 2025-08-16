package api

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

// handleBalancePage serves the balance query page
func (s *APIServer) handleBalancePage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")
	if sessionID == "" {
		return c.Status(400).SendString("Session ID is required")
	}

	// Get template
	template := s.templates["balance"]
	if template == nil {
		return c.Status(500).SendString("Balance template not found")
	}

	// Template data
	data := map[string]interface{}{
		"SessionID": sessionID,
	}

	// Render template
	var rendered strings.Builder
	if err := template.Execute(&rendered, data); err != nil {
		log.Printf("Error rendering balance template: %v", err)
		return c.Status(500).SendString("Error rendering page")
	}

	c.Set("Content-Type", "text/html")
	return c.SendString(rendered.String())
}

// handleBalanceAPI provides session data for balance queries
func (s *APIServer) handleBalanceAPI(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")
	if sessionID == "" {
		return c.Status(400).JSON(map[string]string{
			"error": "Session ID is required",
		})
	}

	// Get session from database
	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(map[string]string{
			"error": "Session not found or expired",
		})
	}

	// Verify session type
	if session.SessionType != "balance_query" {
		return c.Status(400).JSON(map[string]string{
			"error": "Invalid session type for balance query",
		})
	}

	// Parse session data
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(session.TransactionData), &sessionData); err != nil {
		return c.Status(500).JSON(map[string]string{
			"error": "Failed to parse session data",
		})
	}

	// Check if we need to query balance with provided wallet address
	walletAddress := ""
	if addr, exists := sessionData["wallet_address"]; exists && addr != nil {
		if addrStr, ok := addr.(string); ok && addrStr != "" {
			walletAddress = addrStr
		}
	}

	// Query string parameter for wallet address (from frontend)
	if walletAddress == "" {
		walletAddress = c.Query("wallet_address", "")
	}

	// Prepare response data
	response := map[string]interface{}{
		"session_id":     sessionID,
		"query_type":     sessionData["query_type"],
		"chain_type":     sessionData["chain_type"],
		"chain_id":       sessionData["chain_id"],
		"rpc_url":        sessionData["rpc_url"],
		"session_type":   session.SessionType,
		"wallet_address": walletAddress,
	}

	// Include token address if provided
	if tokenAddr, exists := sessionData["token_address"]; exists && tokenAddr != nil {
		if tokenAddrStr, ok := tokenAddr.(string); ok && tokenAddrStr != "" {
			response["token_address"] = tokenAddrStr
		}
	}

	// If wallet address is provided, query the balance
	if walletAddress != "" {
		balanceData, err := s.queryWalletBalance(sessionData, walletAddress)
		if err != nil {
			response["balance_error"] = fmt.Sprintf("Failed to query balance: %v", err)
		} else {
			response["balance_data"] = balanceData
		}
	}

	return c.JSON(response)
}

// queryWalletBalance queries the wallet balance using session data
func (s *APIServer) queryWalletBalance(sessionData map[string]interface{}, walletAddress string) (map[string]interface{}, error) {
	rpcURL, ok := sessionData["rpc_url"].(string)
	if !ok || rpcURL == "" {
		return nil, fmt.Errorf("missing RPC URL in session data")
	}

	chainType, ok := sessionData["chain_type"].(string)
	if !ok || chainType == "" {
		return nil, fmt.Errorf("missing chain type in session data")
	}

	result := map[string]interface{}{
		"wallet_address": walletAddress,
		"chain_type":     chainType,
		"chain_id":       sessionData["chain_id"],
	}

	// Query native balance
	nativeBalance, err := utils.QueryNativeBalance(rpcURL, walletAddress, chainType)
	if err != nil {
		return nil, fmt.Errorf("failed to query native balance: %w", err)
	}

	result["native_balance"] = nativeBalance

	// Query token balance if token address provided
	if tokenAddr, exists := sessionData["token_address"]; exists && tokenAddr != nil {
		if tokenAddrStr, ok := tokenAddr.(string); ok && tokenAddrStr != "" {
			tokenBalance, err := utils.QueryERC20Balance(rpcURL, tokenAddrStr, walletAddress)
			if err != nil {
				// Don't fail completely, just note the error
				result["token_balance_error"] = fmt.Sprintf("Failed to query token balance: %v", err)
			} else {
				result["token_balance"] = tokenBalance
				result["token_address"] = tokenAddrStr
			}
		}
	}

	return result, nil
}
