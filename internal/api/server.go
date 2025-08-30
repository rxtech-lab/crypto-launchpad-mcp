package api

import (
	"fmt"
	"log"
	"net"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

type APIServer struct {
	app         *fiber.App
	db          *database.Database
	txService   services.TransactionService
	hookService services.HookService
	mcpServer   *mcp.MCPServer
	port        int
}

func NewAPIServer(db *database.Database, txService services.TransactionService, hookService services.HookService) *APIServer {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	// Add middleware
	app.Use(cors.New())
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "15:04:05",
		TimeZone:   "Local",
	}))

	server := &APIServer{
		app:         app,
		db:          db,
		txService:   txService,
		hookService: hookService,
	}
	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {

	// Universal transaction signing routes
	s.app.Get("/tx/:session_id", s.handleTransactionPage)
	s.app.Post("/api/tx/:session_id/transaction/:index", s.handleTransactionAPI)

	// Static assets for signing app
	s.app.Get("/static/tx/app.js", s.handleSigningAppJS)
	s.app.Get("/static/tx/app.css", s.handleSigningAppCSS)

	// Legacy balance query routes (redirect to new tx routes)
	s.app.Get("/balance/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/balance/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	// Test API for E2E testing
	s.app.Post("/api/test/sign-transaction", s.handleTestSignTransaction)

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

// SetMCPServer sets the MCP server instance for accessing MCP methods
func (s *APIServer) SetMCPServer(mcpServer *mcp.MCPServer) {
	s.mcpServer = mcpServer
}

// GetMCPServer returns the MCP server instance
func (s *APIServer) GetMCPServer() *mcp.MCPServer {
	return s.mcpServer
}

// handleSigningAppJS serves the embedded signing app.js file
func (s *APIServer) handleSigningAppJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.SigningAppJS)
}

// handleSigningAppCSS serves the embedded signing app.css file
func (s *APIServer) handleSigningAppCSS(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/css")
	return c.Send(assets.SigningAppCSS)
}
