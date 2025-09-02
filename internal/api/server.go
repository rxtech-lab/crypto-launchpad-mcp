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
	app           *fiber.App
	dbService     services.DBService
	txService     services.TransactionService
	hookService   services.HookService
	chainService  services.ChainService
	mcpServer     *mcp.MCPServer
	authenticator *utils.JwtAuthenticator
	port          int
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

	// Create a custom handler that extracts authentication and adds to context
	authenticatedHandler := s.createAuthenticatedMCPHandler(streamableServer, s.authenticator)

	s.app.All("/mcp", authenticatedHandler)
	s.app.All("/mcp/*", authenticatedHandler)

	// Add authentication middleware to all routes
	s.app.Use(middleware.AuthMiddleware(middleware.AuthConfig{
		SkipWellKnown: true,
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			if s.authenticator != nil {
				return s.authenticator.ValidateToken(token)
			}
			// Default validation when no authenticator is configured - reject all tokens
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		},
	}))
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

// createAuthenticatedMCPHandler creates a Fiber handler that extracts authentication
// and passes it to the MCP streamable HTTP server via context
func (s *APIServer) createAuthenticatedMCPHandler(streamableServer *server.StreamableHTTPServer, authenticator *utils.JwtAuthenticator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract Bearer token from Authorization header
		authHeader := c.Get("Authorization")
		var authenticatedUser *utils.AuthenticatedUser

		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
			if token != "" {
				if authenticator != nil {
					// Validate token using JWT authenticator
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
					// No authenticator configured - reject all tokens
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Invalid token",
					})
				}
			}
		} else {
			// No Authorization header or not Bearer token - reject
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing or invalid Bearer token",
			})
		}

		// Create a custom HTTP handler that injects authentication context
		httpHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Add authenticated user to context if available
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
