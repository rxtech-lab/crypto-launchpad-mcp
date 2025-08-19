package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

// Pool creation handlers
func (s *APIServer) handleCreatePoolPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != "create_pool" {
		return c.Status(400).SendString("Invalid session type")
	}

	// Parse session data and generate transaction data
	var transactionData map[string]interface{}
	if sessionData, err := s.parseLiquiditySessionData(session.TransactionData); err == nil {
		if poolID, ok := sessionData["pool_id"].(float64); ok {
			if pool, err := s.db.GetLiquidityPoolByID(uint(poolID)); err == nil {
				if activeChain, err := s.db.GetActiveChain(); err == nil {
					transactionData = s.generateCreatePoolTransactionData(pool, sessionData, activeChain)
				}
			}
		}
	}

	html := s.renderTemplate("create_pool", map[string]interface{}{
		"SessionID":       session.ID,
		"TransactionData": transactionData,
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (s *APIServer) handleCreatePoolAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "create_pool")
}

func (s *APIServer) handleCreatePoolConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "pool")
}

// Add liquidity handlers
func (s *APIServer) handleAddLiquidityPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != "add_liquidity" {
		return c.Status(400).SendString("Invalid session type")
	}

	// Parse session data and generate transaction data
	var transactionData map[string]interface{}
	if sessionData, err := s.parseLiquiditySessionData(session.TransactionData); err == nil {
		if positionID, ok := sessionData["position_id"].(float64); ok {
			if position, err := s.db.GetLiquidityPositionByID(uint(positionID)); err == nil {
				if pool, err := s.db.GetLiquidityPoolByID(position.PoolID); err == nil {
					if activeChain, err := s.db.GetActiveChain(); err == nil {
						transactionData = s.generateAddLiquidityTransactionData(position, pool, sessionData, activeChain)
					}
				}
			}
		}
	}

	// Serialize transactionData to JSON for embedding
	var transactionDataJSON interface{}
	if transactionData != nil {
		transactionDataJSON = transactionData
	}

	html := s.renderTemplate("add_liquidity", map[string]interface{}{
		"SessionID":       session.ID,
		"TransactionData": transactionDataJSON,
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (s *APIServer) handleAddLiquidityAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "add_liquidity")
}

func (s *APIServer) handleAddLiquidityConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "liquidity_position")
}

// Remove liquidity handlers
func (s *APIServer) handleRemoveLiquidityPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != "remove_liquidity" {
		return c.Status(400).SendString("Invalid session type")
	}

	// Parse session data and generate transaction data
	var transactionData map[string]interface{}
	if sessionData, err := s.parseLiquiditySessionData(session.TransactionData); err == nil {
		if positionID, ok := sessionData["position_id"].(float64); ok {
			if position, err := s.db.GetLiquidityPositionByID(uint(positionID)); err == nil {
				if pool, err := s.db.GetLiquidityPoolByID(position.PoolID); err == nil {
					if activeChain, err := s.db.GetActiveChain(); err == nil {
						transactionData = s.generateRemoveLiquidityTransactionData(position, pool, sessionData, activeChain)
					}
				}
			}
		}
	}

	html := s.renderTemplate("remove_liquidity", map[string]interface{}{
		"SessionID":       session.ID,
		"TransactionData": transactionData,
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (s *APIServer) handleRemoveLiquidityAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "remove_liquidity")
}

func (s *APIServer) handleRemoveLiquidityConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "liquidity_position")
}

// Swap handlers
func (s *APIServer) handleSwapPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != "swap" {
		return c.Status(400).SendString("Invalid session type")
	}

	// Parse session data and generate transaction data
	var transactionData map[string]interface{}
	if sessionData, err := s.parseLiquiditySessionData(session.TransactionData); err == nil {
		if swapID, ok := sessionData["swap_id"].(float64); ok {
			if swap, err := s.db.GetSwapTransactionByID(uint(swapID)); err == nil {
				if activeChain, err := s.db.GetActiveChain(); err == nil {
					transactionData = s.generateSwapTransactionData(swap, sessionData, activeChain)
				}
			}
		}
	}

	html := s.renderTemplate("swap", map[string]interface{}{
		"SessionID":       session.ID,
		"TransactionData": transactionData,
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (s *APIServer) handleSwapAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "swap")
}

func (s *APIServer) handleSwapConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "swap")
}

// Generic handlers for common functionality
func (s *APIServer) handleGenericPage(c *fiber.Ctx, sessionType string) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != sessionType {
		return c.Status(400).SendString("Invalid session type")
	}

	html := s.renderTemplate("generic", map[string]interface{}{
		"SessionID": session.ID,
		"Title":     s.getPageTitle(sessionType),
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (s *APIServer) handleGenericAPI(c *fiber.Ctx, sessionType string) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(map[string]string{"error": "Session not found or expired"})
	}

	if session.SessionType != sessionType {
		return c.Status(400).JSON(map[string]string{"error": "Invalid session type"})
	}

	var transactionData map[string]interface{}
	if err := json.Unmarshal([]byte(session.TransactionData), &transactionData); err != nil {
		return c.Status(500).JSON(map[string]string{"error": "Invalid transaction data"})
	}

	return c.JSON(map[string]interface{}{
		"session_id":       session.ID,
		"session_type":     session.SessionType,
		"chain_type":       session.ChainType,
		"chain_id":         session.ChainID,
		"transaction_data": transactionData,
		"status":           session.Status,
		"created_at":       session.CreatedAt,
		"expires_at":       session.ExpiresAt,
	})
}

func (s *APIServer) handleGenericConfirm(c *fiber.Ctx, recordType string) error {
	sessionID := c.Params("session_id")

	var body struct {
		TransactionHash string                   `json:"transaction_hash"`
		Status          models.TransactionStatus `json:"status"`
		PairAddress     string                   `json:"pair_address,omitempty"`     // For pool creation
		ContractAddress string                   `json:"contract_address,omitempty"` // For other deployments
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(map[string]string{"error": "Invalid request body"})
	}

	// Update session status
	if err := s.db.UpdateTransactionSessionStatus(sessionID, body.Status, body.TransactionHash); err != nil {
		return c.Status(500).JSON(map[string]string{"error": "Failed to update session"})
	}

	// Update related records based on type
	if body.Status == "confirmed" && body.TransactionHash != "" {
		session, err := s.db.GetTransactionSession(sessionID)
		if err == nil {
			var transactionData map[string]interface{}
			if err := json.Unmarshal([]byte(session.TransactionData), &transactionData); err == nil {
				// Pass additional data for specific record types
				extraData := map[string]string{
					"pair_address":     body.PairAddress,
					"contract_address": body.ContractAddress,
				}
				s.updateRelatedRecord(recordType, transactionData, body.TransactionHash, extraData)
			}
		}
	}

	return c.JSON(map[string]string{"status": "success"})
}

func (s *APIServer) updateRelatedRecord(recordType string, transactionData map[string]interface{}, txHash string, extraData map[string]string) {
	switch recordType {
	case "pool":
		if poolID, ok := transactionData["pool_id"].(float64); ok {
			pairAddress := extraData["pair_address"]
			s.db.UpdateLiquidityPoolStatus(uint(poolID), "confirmed", pairAddress, txHash)
		}
	case "liquidity_position":
		if positionID, ok := transactionData["position_id"].(float64); ok {
			s.db.UpdateLiquidityPositionStatus(uint(positionID), "confirmed", txHash)
		}
	case "swap":
		if swapID, ok := transactionData["swap_id"].(float64); ok {
			s.db.UpdateSwapTransactionStatus(uint(swapID), "confirmed", txHash)
		}
	}
}
