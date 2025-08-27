package api

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type APIServer struct {
	app       *fiber.App
	db        *database.Database
	txService services.TransactionService
	mcpServer *mcp.MCPServer
	port      int
}

func NewAPIServer(db *database.Database, txService services.TransactionService) *APIServer {
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
		app:       app,
		db:        db,
		txService: txService,
	}
	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {

	// Universal transaction signing routes
	s.app.Get("/tx/:session_id", s.handleTransactionPage)
	s.app.Get("/api/tx/:session_id", s.handleTransactionAPI)
	s.app.Get("/api/session/:session_id", s.handleTransactionAPI) // Alias for React app compatibility

	// Static assets for signing app
	s.app.Get("/static/tx/app.js", s.handleSigningAppJS)
	s.app.Get("/static/tx/app.css", s.handleSigningAppCSS)

	// Legacy deployment signing routes (redirect to new tx routes)
	s.app.Get("/deploy/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/deploy/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	// Legacy Uniswap deployment routes (redirect to new tx routes)
	s.app.Get("/deploy-uniswap/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/deploy-uniswap/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	// Legacy balance query routes (redirect to new tx routes)
	s.app.Get("/balance/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/balance/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	// Legacy liquidity pool creation routes (redirect to new tx routes)
	s.app.Get("/pool/create/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/pool/create/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	// Legacy liquidity management routes (redirect to new tx routes)
	s.app.Get("/liquidity/add/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/liquidity/add/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	s.app.Get("/liquidity/remove/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/liquidity/remove/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	// Legacy swap routes (redirect to new tx routes)
	s.app.Get("/swap/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/tx/" + c.Params("session_id"))
	})
	s.app.Get("/api/swap/:session_id", func(c *fiber.Ctx) error {
		return c.Redirect("/api/tx/" + c.Params("session_id"))
	})

	// Test API for E2E testing
	s.app.Post("/api/test/sign-transaction", s.handleTestSignTransaction)

	// Contract artifacts API
	s.app.Get("/api/contracts/:name", s.handleContractArtifact)

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

// verifyTransactionOnChain verifies that a transaction exists and was successful on-chain
func (s *APIServer) verifyTransactionOnChain(txHash, chainID string) error { // Get active chain configuration to get RPC URL
	activeChain, err := s.db.GetActiveChain()
	if err != nil {
		return fmt.Errorf("failed to get active chain: %w", err)
	}

	// Verify chain ID matches
	if activeChain.NetworkID != chainID {
		return fmt.Errorf("chain ID mismatch: session has %s but active chain is %s", chainID, activeChain.NetworkID)
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
