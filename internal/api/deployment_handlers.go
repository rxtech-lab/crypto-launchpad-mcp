package api

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleDeploymentPage serves the deployment signing page
func (s *APIServer) handleDeploymentPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	// Validate session
	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != "deploy" {
		return c.Status(400).SendString("Invalid session type")
	}

	// Parse session data to get deployment ID and compile transaction data
	var sessionData map[string]interface{}
	var transactionData map[string]interface{}

	if err := json.Unmarshal([]byte(session.TransactionData), &sessionData); err == nil {
		if deploymentID, ok := sessionData["deployment_id"].(float64); ok {
			// Get deployment and template data for compilation
			if deployment, err := s.db.GetDeploymentByID(uint(deploymentID)); err == nil {
				if template, err := s.db.GetTemplateByID(deployment.TemplateID); err == nil {
					if activeChain, err := s.db.GetActiveChain(); err == nil {
						// Generate transaction data with bytecode during template rendering
						transactionData = s.generateTransactionData(deployment, template, activeChain)
					}
				}
			}
		}
	}

	// Serve HTML page with embedded transaction data
	html := s.renderTemplate("deploy", map[string]interface{}{
		"SessionID":       session.ID,
		"TransactionData": transactionData,
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// handleDeploymentAPI provides transaction data for deployment via API
func (s *APIServer) handleDeploymentAPI(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(map[string]string{"error": "Session not found or expired"})
	}

	if session.SessionType != "deploy" {
		return c.Status(400).JSON(map[string]string{"error": "Invalid session type"})
	}

	// Parse session data to get deployment ID
	var sessionData map[string]interface{}
	if err := json.Unmarshal([]byte(session.TransactionData), &sessionData); err != nil {
		return c.Status(500).JSON(map[string]string{"error": "Invalid session data"})
	}

	deploymentID, ok := sessionData["deployment_id"].(float64)
	if !ok {
		return c.Status(500).JSON(map[string]string{"error": "Invalid deployment ID in session"})
	}

	// Get deployment record
	deployment, err := s.db.GetDeploymentByID(uint(deploymentID))
	if err != nil {
		return c.Status(500).JSON(map[string]string{"error": "Deployment not found"})
	}

	// Get template
	template, err := s.db.GetTemplateByID(deployment.TemplateID)
	if err != nil {
		return c.Status(500).JSON(map[string]string{"error": "Template not found"})
	}

	// Get active chain configuration
	activeChain, err := s.db.GetActiveChain()
	if err != nil {
		return c.Status(500).JSON(map[string]string{"error": "No active chain configured"})
	}

	// Generate transaction data on-the-fly
	transactionData := s.generateTransactionData(deployment, template, activeChain)

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

// handleDeploymentConfirm processes deployment transaction confirmation
func (s *APIServer) handleDeploymentConfirm(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	var body struct {
		TransactionHash string `json:"transaction_hash"`
		Status          string `json:"status"`
		ContractAddress string `json:"contract_address"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(map[string]string{"error": "Invalid request body"})
	}

	// Update session status
	if err := s.db.UpdateTransactionSessionStatus(sessionID, body.Status, body.TransactionHash); err != nil {
		return c.Status(500).JSON(map[string]string{"error": "Failed to update session"})
	}

	// If confirmed, verify transaction on-chain and update deployment record
	if body.Status == "confirmed" && body.TransactionHash != "" {
		session, err := s.db.GetTransactionSession(sessionID)
		if err == nil {
			var transactionData map[string]interface{}
			if err := json.Unmarshal([]byte(session.TransactionData), &transactionData); err == nil {
				if deploymentID, ok := transactionData["deployment_id"].(float64); ok {
					// Verify transaction on-chain before marking as confirmed
					if err := s.verifyTransactionOnChain(body.TransactionHash, session.ChainID); err != nil {
						log.Printf("Transaction verification failed for %s: %v", body.TransactionHash, err)
						// Update deployment status as failed due to verification failure
						s.db.UpdateDeploymentStatus(uint(deploymentID), "failed", "", body.TransactionHash)
						return c.Status(400).JSON(map[string]string{"error": "Transaction verification failed: " + err.Error()})
					}

					// Transaction verified successfully - update deployment
					s.db.UpdateDeploymentStatus(uint(deploymentID), "confirmed", body.ContractAddress, body.TransactionHash)
					s.mcpServer.SendMessageToAiClient(
						[]mcp.SamplingMessage{
							{
								Role: "user",
								Content: mcp.TextContent{
									Text: fmt.Sprintf("Deployment %d confirmed with transaction %s", uint(deploymentID), body.TransactionHash),
									Type: "text",
								},
							},
						},
					)
					log.Printf("Deployment %d confirmed with transaction %s", uint(deploymentID), body.TransactionHash)
				}
			}
		}
	}

	response := map[string]interface{}{
		"status": "success",
	}

	if body.ContractAddress != "" {
		response["contract_address"] = body.ContractAddress
	}

	return c.JSON(response)
}
