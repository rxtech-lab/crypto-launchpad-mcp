package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
)

type APIServer struct {
	app  *fiber.App
	db   *database.Database
	port int
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

	server.setupRoutes()
	return server
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
	html := s.generateDeploymentHTML(session)
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
					// Update deployment with transaction hash
					s.db.UpdateDeploymentStatus(uint(deploymentID), "confirmed", "", body.TransactionHash)
				}
			}
		}
	}

	return c.JSON(map[string]string{"status": "success"})
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

	html := s.generateCreatePoolHTML(session)
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
	return s.handleGenericPage(c, "add_liquidity", s.generateAddLiquidityHTML)
}

func (s *APIServer) handleAddLiquidityAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "add_liquidity")
}

func (s *APIServer) handleAddLiquidityConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "liquidity_position")
}

func (s *APIServer) handleRemoveLiquidityPage(c *fiber.Ctx) error {
	return s.handleGenericPage(c, "remove_liquidity", s.generateRemoveLiquidityHTML)
}

func (s *APIServer) handleRemoveLiquidityAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "remove_liquidity")
}

func (s *APIServer) handleRemoveLiquidityConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "liquidity_position")
}

func (s *APIServer) handleSwapPage(c *fiber.Ctx) error {
	return s.handleGenericPage(c, "swap", s.generateSwapHTML)
}

func (s *APIServer) handleSwapAPI(c *fiber.Ctx) error {
	return s.handleGenericAPI(c, "swap")
}

func (s *APIServer) handleSwapConfirm(c *fiber.Ctx) error {
	return s.handleGenericConfirm(c, "swap")
}

// Generic handlers
func (s *APIServer) handleGenericPage(c *fiber.Ctx, sessionType string, htmlGenerator func(*models.TransactionSession) string) error {
	sessionID := c.Params("session_id")

	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		return c.Status(404).SendString("Session not found or expired")
	}

	if session.SessionType != sessionType {
		return c.Status(400).SendString("Invalid session type")
	}

	html := htmlGenerator(session)
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

// HTML generation methods will be implemented in the next step
func (s *APIServer) generateDeploymentHTML(session *models.TransactionSession) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Deploy Contract</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <div class="max-w-2xl mx-auto bg-white rounded-lg shadow-lg p-6">
            <h1 class="text-3xl font-bold text-gray-800 mb-6">Deploy Contract</h1>
            <div id="session-data" data-session-id="%s" data-api-url="/api/deploy/%s"></div>
            <div id="content" class="space-y-6">
                <p class="text-gray-600">Loading transaction details...</p>
            </div>
        </div>
    </div>
    <script src="/js/wallet.js"></script>
</body>
</html>`, session.ID, session.ID)
}

func (s *APIServer) generateCreatePoolHTML(session *models.TransactionSession) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Create Liquidity Pool</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <div class="max-w-2xl mx-auto bg-white rounded-lg shadow-lg p-6">
            <h1 class="text-3xl font-bold text-gray-800 mb-6">Create Liquidity Pool</h1>
            <div id="session-data" data-session-id="%s" data-api-url="/api/pool/create/%s"></div>
            <div id="content" class="space-y-6">
                <p class="text-gray-600">Loading transaction details...</p>
            </div>
        </div>
    </div>
    <script src="/js/wallet.js"></script>
</body>
</html>`, session.ID, session.ID)
}

func (s *APIServer) generateAddLiquidityHTML(session *models.TransactionSession) string {
	return s.generateGenericHTML("Add Liquidity", session)
}

func (s *APIServer) generateRemoveLiquidityHTML(session *models.TransactionSession) string {
	return s.generateGenericHTML("Remove Liquidity", session)
}

func (s *APIServer) generateSwapHTML(session *models.TransactionSession) string {
	return s.generateGenericHTML("Swap Tokens", session)
}

func (s *APIServer) generateGenericHTML(title string, session *models.TransactionSession) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <div class="max-w-2xl mx-auto bg-white rounded-lg shadow-lg p-6">
            <h1 class="text-3xl font-bold text-gray-800 mb-6">%s</h1>
            <div id="session-data" data-session-id="%s"></div>
            <div id="content" class="space-y-6">
                <p class="text-gray-600">Loading transaction details...</p>
            </div>
        </div>
    </div>
    <script src="/js/wallet.js"></script>
</body>
</html>`, title, title, session.ID)
}

// handleWalletJS serves the wallet.js file
func (s *APIServer) handleWalletJS(c *fiber.Ctx) error {
	// Read the wallet.js file from templates directory
	walletJSPath := filepath.Join("templates", "wallet.js")
	content, err := ioutil.ReadFile(walletJSPath)
	if err != nil {
		return c.Status(404).SendString("File not found")
	}

	c.Set("Content-Type", "application/javascript")
	return c.Send(content)
}
