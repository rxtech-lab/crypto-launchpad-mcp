package api

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/api/middleware"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

type APIServer struct {
	app                   *fiber.App
	dbService             services.DBService
	txService             services.TransactionService
	hookService           services.HookService
	chainService          services.ChainService
	mcpServer             *mcp.MCPServer
	authenticator         *utils.JwtAuthenticator
	port                  int
	authenticationEnabled bool
}

func NewAPIServer(dbService services.DBService, txService services.TransactionService, hookService services.HookService, chainService services.ChainService) *APIServer {
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
	// Get JWKS URI from environment variable
	jwksUri := os.Getenv("SCALEKIT_ENV_URL")
	var authenticator *utils.JwtAuthenticator
	if jwksUri != "" {
		auth := utils.NewJwtAuthenticator(jwksUri)
		authenticator = &auth
		log.Printf("JWT authenticator initialized with JWKS URI: %s", jwksUri)
	} else {
		log.Println("Warning: SCALEKIT_ENV_URL not set, JWT authentication disabled")
	}

	server := &APIServer{
		app:           app,
		dbService:     dbService,
		txService:     txService,
		hookService:   hookService,
		chainService:  chainService,
		authenticator: authenticator,
	}
	return server
}

func (s *APIServer) SetupRoutes() {
	// Universal transaction signing routes
	s.app.Get("/tx/:session_id", s.handleTransactionPage)
	s.app.Post("/api/tx/:session_id/transaction/:index", s.handleTransactionAPI)
	// Static assets for signing app
	s.app.Get("/static/tx/app.js", s.handleSigningAppJS)
	s.app.Get("/static/tx/app.css", s.handleSigningAppCSS)
	// Test API for E2E testing
	s.app.Post("/api/test/sign-transaction", s.handleTestSignTransaction)
	s.app.Post("/api/test/personal-sign", s.handleTestPersonalSign)

	// Health check
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(map[string]string{"status": "ok"})
	})
}

func (s *APIServer) EnableAuthentication() {
	// Set authentication enabled state
	s.authenticationEnabled = true

	// oauth routes
	s.app.Get("/.well-known/oauth-protected-resource/mcp", s.handleOAuthProtectedResource)
	// Add authentication middleware to all routes
	s.app.Use(middleware.AuthMiddleware(middleware.AuthConfig{
		SkipWellKnown: true,
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			if s.authenticator != nil {
				return s.authenticator.ValidateToken(token)
			}
			// Default validation when no authenticator is configured
			if token == "" {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
			}
			return &utils.AuthenticatedUser{}, nil
		},
	}))
}

// EnableStreamableHttp enables the MCP Streamable HTTP server conditionally with authentication
// on the /mcp and /mcp/* routes based on whether EnableAuthentication was called

func (s *APIServer) EnableStreamableHttp() {
	if s.mcpServer == nil {
		log.Fatal("MCP server not set. Cannot enable Streamable HTTP.")
		return
	}
	// Start the streamable HTTP server
	streamableServer := s.mcpServer.StartStreamableHTTPServer()

	// Create a custom handler based on authentication state
	var mcpHandler fiber.Handler
	if s.authenticationEnabled {
		// Use authenticated handler if authentication was enabled
		mcpHandler = s.createAuthenticatedMCPHandler(streamableServer, s.authenticator)
		log.Println("MCP handlers enabled with authentication")
	} else {
		// Use unauthenticated handler if authentication was not enabled
		mcpHandler = s.createUnauthenticatedMCPHandler(streamableServer)
		log.Println("MCP handlers enabled without authentication")
	}

	s.app.All("/mcp", mcpHandler)
	s.app.All("/mcp/*", mcpHandler)
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

// createAuthenticatedMCPHandler creates a Fiber handler that enforces authentication
// and passes authenticated context to the MCP streamable HTTP server
func (s *APIServer) createAuthenticatedMCPHandler(streamableServer *server.StreamableHTTPServer, authenticator *utils.JwtAuthenticator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract Bearer token from Authorization header
		authHeader := c.Get("Authorization")
		var authenticatedUser *utils.AuthenticatedUser

		// Check if Authorization header is present
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authorization header required",
			})
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

		// Check if token is empty
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		// Validate token if authenticator is available
		if authenticator != nil {
			if user, err := authenticator.ValidateToken(token); err == nil {
				authenticatedUser = user
				log.Printf("MCP request authenticated as user: %s", user.Sub)
			} else {
				log.Printf("MCP authentication failed: %v", err)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid token",
				})
			}
		} else {
			// No authenticator configured, but we still require a token to be present
			// This maintains security by default even when JWT validation is disabled
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication not configured",
			})
		}

		// Create a custom HTTP handler that injects authentication context
		httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Add authenticated user to context
			if authenticatedUser != nil {
				ctx = utils.WithAuthenticatedUser(ctx, authenticatedUser)
			}

			// Update request with authenticated context
			r = r.WithContext(ctx)

			// Forward to the actual MCP streamable server
			streamableServer.ServeHTTP(w, r)
		})

		// Use Fiber's adaptor to convert
		return adaptor.HTTPHandler(httpHandler)(c)
	}
}

// createUnauthenticatedMCPHandler creates a Fiber handler that does not enforce authentication
// and passes the request directly to the MCP streamable HTTP server
func (s *APIServer) createUnauthenticatedMCPHandler(streamableServer *server.StreamableHTTPServer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Create a simple HTTP handler that forwards directly to MCP server
		httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Forward to the actual MCP streamable server without authentication context
			streamableServer.ServeHTTP(w, r)
		})

		// Use Fiber's adaptor to convert
		return adaptor.HTTPHandler(httpHandler)(c)
	}
}
