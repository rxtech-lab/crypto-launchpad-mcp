package api

import (
	"fmt"
	"log"
	"net"
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

// handleWalletJS serves the embedded wallet.js file
func (s *APIServer) handleWalletJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.WalletJS)
}
