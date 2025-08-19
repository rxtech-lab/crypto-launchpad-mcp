package api

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
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
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "15:04:05",
		TimeZone:   "Local",
	}))

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

	// Define custom functions
	funcMap := template.FuncMap{
		"json": func(v interface{}) (string, error) {
			b, err := json.Marshal(v)
			return string(b), err
		},
	}

	// Parse deployment template with custom functions
	deployTmpl, err := template.New("deploy").Funcs(funcMap).Parse(string(assets.DeployHTML))
	if err != nil {
		log.Printf("Error parsing deploy template: %v", err)
	} else {
		s.templates["deploy"] = deployTmpl
	}

	// Parse create pool template with custom functions
	poolTmpl, err := template.New("create_pool").Funcs(funcMap).Parse(string(assets.CreatePoolHTML))
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

	// Parse Uniswap deployment template with custom functions
	uniswapTmpl, err := template.New("deploy_uniswap").Funcs(funcMap).Parse(string(assets.DeployUniswapHTML))
	if err != nil {
		log.Printf("Error parsing deploy uniswap template: %v", err)
	} else {
		s.templates["deploy_uniswap"] = uniswapTmpl
	}

	// Parse balance query template
	balanceTmpl, err := template.New("balance").Parse(string(assets.BalanceHTML))
	if err != nil {
		log.Printf("Error parsing balance template: %v", err)
	} else {
		s.templates["balance"] = balanceTmpl
	}

	// Parse add liquidity template with custom functions
	addLiquidityTmpl, err := template.New("add_liquidity").Funcs(funcMap).Parse(string(assets.AddLiquidityHTML))
	if err != nil {
		log.Printf("Error parsing add liquidity template: %v", err)
	} else {
		s.templates["add_liquidity"] = addLiquidityTmpl
	}

	// Parse remove liquidity template with custom functions
	removeLiquidityTmpl, err := template.New("remove_liquidity").Funcs(funcMap).Parse(string(assets.RemoveLiquidityHTML))
	if err != nil {
		log.Printf("Error parsing remove liquidity template: %v", err)
	} else {
		s.templates["remove_liquidity"] = removeLiquidityTmpl
	}

	// Parse swap template with custom functions
	swapTmpl, err := template.New("swap").Funcs(funcMap).Parse(string(assets.SwapHTML))
	if err != nil {
		log.Printf("Error parsing swap template: %v", err)
	} else {
		s.templates["swap"] = swapTmpl
	}
}

func (s *APIServer) setupRoutes() {
	// Static files
	s.app.Get("/js/wallet.js", s.handleWalletJS)
	s.app.Get("/js/wallet-connection.js", s.handleWalletConnectionJS)
	s.app.Get("/js/deploy-tokens.js", s.handleDeployTokensJS)
	s.app.Get("/js/deploy-uniswap.js", s.handleDeployUniswapJS)
	s.app.Get("/js/balance-query.js", s.handleBalanceQueryJS)
	s.app.Get("/js/create-pool.js", s.handleCreatePoolJS)
	s.app.Get("/js/liquidity.js", s.handleLiquidityJS)

	// Deployment signing routes
	s.app.Get("/deploy/:session_id", s.handleDeploymentPage)
	s.app.Get("/api/deploy/:session_id", s.handleDeploymentAPI)
	s.app.Post("/api/deploy/:session_id/confirm", s.handleDeploymentConfirm)

	// Uniswap deployment routes
	s.app.Get("/deploy-uniswap/:session_id", s.handleUniswapDeploymentPage)
	s.app.Get("/api/deploy-uniswap/:session_id", s.handleUniswapDeploymentAPI)
	s.app.Post("/api/deploy-uniswap/:session_id/confirm", s.handleUniswapDeploymentConfirm)

	// Balance query routes
	s.app.Get("/balance/:session_id", s.handleBalancePage)
	s.app.Get("/api/balance/:session_id", s.handleBalanceAPI)

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

// handleWalletJS serves the embedded wallet.js file
func (s *APIServer) handleWalletJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.WalletJS)
}

// handleWalletConnectionJS serves the embedded wallet-connection.js file
func (s *APIServer) handleWalletConnectionJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.WalletConnectionJS)
}

// handleDeployTokensJS serves the embedded deploy-tokens.js file
func (s *APIServer) handleDeployTokensJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.DeployTokensJS)
}

// handleDeployUniswapJS serves the embedded deploy-uniswap.js file
func (s *APIServer) handleDeployUniswapJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.DeployUniswapJS)
}

// handleBalanceQueryJS serves the embedded balance-query.js file
func (s *APIServer) handleBalanceQueryJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.BalanceQueryJS)
}

// handleCreatePoolJS serves the embedded create-pool.js file
func (s *APIServer) handleCreatePoolJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.CreatePoolJS)
}

// handleLiquidityJS serves the embedded liquidity.js file
func (s *APIServer) handleLiquidityJS(c *fiber.Ctx) error {
	c.Set("Content-Type", "application/javascript")
	return c.Send(assets.LiquidityJS)
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
