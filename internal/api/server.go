package api

import (
	"fmt"
	"log"
	"net"
	"text/template"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type APIServer struct {
	app       *fiber.App
	db        *database.Database
	mcpServer *mcp.MCPServer
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

// SetMCPServer sets the MCP server instance for accessing MCP methods
func (s *APIServer) SetMCPServer(mcpServer *mcp.MCPServer) {
	s.mcpServer = mcpServer
}

// GetMCPServer returns the MCP server instance
func (s *APIServer) GetMCPServer() *mcp.MCPServer {
	return s.mcpServer
}

// handleWalletJS serves the embedded wallet.js file
func (s *APIServer) handleWalletJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.WalletJS)
}

// verifyTransactionOnChain verifies that a transaction exists and was successful on-chain
func (s *APIServer) verifyTransactionOnChain(txHash, chainID string) error { // Get active chain configuration to get RPC URL
	activeChain, err := s.db.GetActiveChain()
	if err != nil {
		return fmt.Errorf("failed to get active chain: %w", err)
	}

	// Verify chain ID matches
	if activeChain.ChainID != chainID {
		return fmt.Errorf("chain ID mismatch: session has %s but active chain is %s", chainID, activeChain.ChainID)
	}

	// Create RPC client
	rpcClient := utils.NewRPCClient(activeChain.RPC)
	rpcClient.SetTimeout(15 * time.Second)

	// Verify transaction success
	success, receipt, err := rpcClient.VerifyTransactionSuccess(txHash)
	if err != nil {
		return fmt.Errorf("failed to verify transaction: %w", err)
	}

	if !success {
		return fmt.Errorf("transaction failed on-chain (status: %s)", receipt.Status)
	}

	log.Printf("Transaction %s verified successfully on chain %s (block: %s)", txHash, chainID, receipt.BlockNumber)
	return nil
}
