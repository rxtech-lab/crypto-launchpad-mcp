package api

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/rxtech-lab/launchpad-mcp/internal/api/middleware"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type APIServer struct {
	app          *fiber.App
	dbService    services.DBService
	txService    services.TransactionService
	hookService  services.HookService
	chainService services.ChainService
	mcpServer    *mcp.MCPServer
	port         int
}

func NewAPIServer(dbService services.DBService, txService services.TransactionService, hookService services.HookService, chainService services.ChainService) *APIServer {
	// Get JWKS URI from environment variable
	jwksUri := os.Getenv("OAUTH_JWKS_URI")
	resourceID := os.Getenv("OAUTH_RESOURCE_ID")

	var authenticator *utils.JwtAuthenticator
	if jwksUri != "" {
		authenticator = utils.NewJwtAuthenticator(jwksUri)
		log.Printf("JWT authenticator initialized with JWKS URI: %s", jwksUri)
	} else {
		log.Println("Warning: OAUTH_JWKS_URI not set, JWT authentication disabled")
	}

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

	// Add authentication middleware to all routes
	app.Use(middleware.AuthMiddleware(middleware.AuthConfig{
		SkipWellKnown: true,
		TokenValidator: func(token string, audience []string) error {
			// TODO: Implement actual token validation (e.g., with Scalekit)
			if token == "" {
				return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
			}

			return nil
		},
	}))

	server := &APIServer{
		app:          app,
		dbService:    dbService,
		txService:    txService,
		hookService:  hookService,
		chainService: chainService,
	}
	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	// oauth routes
	s.app.Get("/.well-known/oauth-protected-resource/mcp", s.handleOAuthProtectedResource)
	// Universal transaction signing routes
	s.app.Get("/tx/:session_id", s.handleTransactionPage)
	s.app.Post("/api/tx/:session_id/transaction/:index", s.handleTransactionAPI)
	// Static assets for signing app
	s.app.Get("/static/tx/app.js", s.handleSigningAppJS)
	s.app.Get("/static/tx/app.css", s.handleSigningAppCSS)
	// Test API for E2E testing
	s.app.Post("/api/test/sign-transaction", s.handleTestSignTransaction)

	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(map[string]string{"status": "ok"})
	})
}

func (s *APIServer) EnableStreamableHttp() {
	if s.mcpServer == nil {
		log.Fatal("MCP server not set. Cannot enable Streamable HTTP.")
		return
	}
	// Start the streamable HTTP server
	streamableServer := s.mcpServer.StartStreamableHTTPServer()

	s.app.All("/mcp", adaptor.HTTPHandler(streamableServer))
	s.app.All("/mcp/*", adaptor.HTTPHandler(streamableServer))
}

// Start starts the server on a random available port
// if port is nil, otherwise starts on the specified port
func (s *APIServer) Start(port *int) (int, error) {
	// Find an available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}

	// Get the assigned port
	assignedPort := listener.Addr().(*net.TCPAddr).Port
	s.port = assignedPort

	if port != nil {
		s.port = *port
	}

	// Close the listener so Fiber can use it
	err = listener.Close()
	if err != nil {
		return 0, err
	}

	// Start the server on the found port
	go func() {
		if err := s.app.Listen(fmt.Sprintf(":%d", s.port)); err != nil {
			log.Printf("Error starting API server: %v\n", err)
		}
	}()

	return s.port, nil
}

func (s *APIServer) Shutdown() error {
	return s.app.Shutdown()
}

func (s *APIServer) GetPort() int {
	return s.port
}

func (s *APIServer) SetMCPServer(mcpServer *mcp.MCPServer) {
	s.mcpServer = mcpServer
}

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
