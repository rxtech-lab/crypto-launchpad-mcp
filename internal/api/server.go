package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"text/template"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
)

type APIServer struct {
	app       *fiber.App
	db        *database.Database
	port      int
	templates map[string]*template.Template
}

func NewAPIServer(db *database.Database) *APIServer {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Add middleware
	app.Use(cors.New())
	app.Use(logger.New())

	server := &APIServer{
		app: app,
		db:  db,
	}

	server.initTemplates()
	server.setupRoutes()
	return server
}

func (s *APIServer) initTemplates() {
	s.templates = make(map[string]*template.Template)

	// Parse deployment template
	deployTmpl, err := template.New("deploy").Parse(string(assets.DeployHTML))
	if err != nil {
		log.Printf("Error parsing deploy template: %v", err)
	} else {
		s.templates["deploy"] = deployTmpl
	}

	// Parse create pool template
	poolTmpl, err := template.New("create_pool").Parse(string(assets.CreatePoolHTML))
	if err != nil {
		log.Printf("Error parsing create pool template: %v", err)
	} else {
		s.templates["create_pool"] = poolTmpl
	}

	// Parse generic template
	genericTmpl, err := template.New("generic").Parse(string(assets.GenericHTML))
	if err != nil {
		log.Printf("Error parsing generic template: %v", err)
	} else {
		s.templates["generic"] = genericTmpl
	}
}

func (s *APIServer) setupRoutes() {
	// Static files
	s.app.Get("/js/wallet.js", s.handleWalletJS)

	// Deployment signing routes
	s.app.Get("/deploy/:session_id", s.handleDeploymentPage)
	s.app.Get("/api/deploy/:session_id", s.handleDeploymentAPI)
	s.app.Post("/api/deploy/:session_id/confirm", s.handleDeploymentConfirm)

	// Liquidity pool creation routes
	s.app.Get("/pool/create/:session_id", s.handleCreatePoolPage)
	s.app.Get("/api/pool/create/:session_id", s.handleCreatePoolAPI)
	s.app.Post("/api/pool/create/:session_id/confirm", s.handleCreatePoolConfirm)

	// Liquidity management routes
	s.app.Get("/liquidity/add/:session_id", s.handleAddLiquidityPage)
	s.app.Get("/api/liquidity/add/:session_id", s.handleAddLiquidityAPI)
	s.app.Post("/api/liquidity/add/:session_id/confirm", s.handleAddLiquidityConfirm)

	s.app.Get("/liquidity/remove/:session_id", s.handleRemoveLiquidityPage)
	s.app.Get("/api/liquidity/remove/:session_id", s.handleRemoveLiquidityAPI)
	s.app.Post("/api/liquidity/remove/:session_id/confirm", s.handleRemoveLiquidityConfirm)

	// Swap routes
	s.app.Get("/swap/:session_id", s.handleSwapPage)
	s.app.Get("/api/swap/:session_id", s.handleSwapAPI)
	s.app.Post("/api/swap/:session_id/confirm", s.handleSwapConfirm)

	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(map[string]string{"status": "ok"})
	})
}

// Start starts the server on a random available port
func (s *APIServer) Start() (int, error) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}

	// Get the assigned port
	port := listener.Addr().(*net.TCPAddr).Port
	s.port = port

	// Close the listener so Fiber can use it
	listener.Close()

	// Start the server on the found port
	go func() {
		if err := s.app.Listen(fmt.Sprintf(":%d", port)); err != nil {
			log.Printf("Error starting API server: %v\n", err)
		}
	}()

	return port, nil
}

func (s *APIServer) Shutdown() error {
	return s.app.Shutdown()
}

func (s *APIServer) GetPort() int {
	return s.port
}

// Deployment handlers
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

	// Serve HTML page with embedded transaction data
	html := s.renderTemplate("deploy", map[string]interface{}{
		"SessionID": session.ID,
	})
	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

func (s *APIServer) handleDeploymentAPI(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).JSON(map[string]string{"error": "Session not found or expired"})
	}

	if session.SessionType != "deploy" {
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

	// If confirmed, update deployment record
	if body.Status == "confirmed" && body.TransactionHash != "" {
		session, err := s.db.GetTransactionSession(sessionID)
		if err == nil {
			var transactionData map[string]interface{}
			if err := json.Unmarshal([]byte(session.TransactionData), &transactionData); err == nil {
				if deploymentID, ok := transactionData["deployment_id"].(float64); ok {
					// Update deployment with transaction hash and contract address
					s.db.UpdateDeploymentStatus(uint(deploymentID), "confirmed", body.ContractAddress, body.TransactionHash)
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

// Similar handlers for other transaction types...
func (s *APIServer) handleCreatePoolPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != "create_pool" {
		return c.Status(400).SendString("Invalid session type")
	}

	html := s.renderTemplate("create_pool", map[string]interface{}{
		"SessionID": session.ID,
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

func (s *APIServer) handleAddLiquidityPage(c *fiber.Ctx) error {
	return s.handleGenericPage(c, "add_liquidity")
}

func (s *APIServer) handleAddLiquidityAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "add_liquidity")
}

func (s *APIServer) handleAddLiquidityConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "liquidity_position")
}

func (s *APIServer) handleRemoveLiquidityPage(c *fiber.Ctx) error {
	return s.handleGenericPage(c, "remove_liquidity")
}

func (s *APIServer) handleRemoveLiquidityAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "remove_liquidity")
}

func (s *APIServer) handleRemoveLiquidityConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "liquidity_position")
}

func (s *APIServer) handleSwapPage(c *fiber.Ctx) error {
	return s.handleGenericPage(c, "swap")
}

func (s *APIServer) handleSwapAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "swap")
}

func (s *APIServer) handleSwapConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "swap")
}

// Generic handlers
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
		TransactionHash string `json:"transaction_hash"`
		Status          string `json:"status"`
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
				s.updateRelatedRecord(recordType, transactionData, body.TransactionHash)
			}
		}
	}

	return c.JSON(map[string]string{"status": "success"})
}

func (s *APIServer) updateRelatedRecord(recordType string, transactionData map[string]interface{}, txHash string) {
	switch recordType {
	case "pool":
		if poolID, ok := transactionData["pool_id"].(float64); ok {
			s.db.UpdateLiquidityPoolStatus(uint(poolID), "confirmed", "", txHash)
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

// Template rendering helper methods
func (s *APIServer) renderTemplate(templateName string, data interface{}) string {
	tmpl, exists := s.templates[templateName]
	if !exists {
		log.Printf("Template %s not found", templateName)
		return fmt.Sprintf("Template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Error executing template %s: %v", templateName, err)
		return fmt.Sprintf("Error rendering template: %v", err)
	}

	return buf.String()
}

func (s *APIServer) getPageTitle(sessionType string) string {
	switch sessionType {
	case "add_liquidity":
		return "Add Liquidity"
	case "remove_liquidity":
		return "Remove Liquidity"
	case "swap":
		return "Swap Tokens"
	default:
		// Capitalize first letter and replace underscores with spaces
		title := strings.ReplaceAll(sessionType, "_", " ")
		if len(title) > 0 {
			title = strings.ToUpper(string(title[0])) + title[1:]
		}
		return title
	}
}

// handleWalletJS serves the embedded wallet.js file
func (s *APIServer) handleWalletJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.WalletJS)
}
