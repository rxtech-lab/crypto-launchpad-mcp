package handler

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

var (
	apiServer   *api.APIServer
	initialized bool
)

// Handler is the main Vercel function handler
func Handler(w http.ResponseWriter, r *http.Request) {
	// Initialize the API server only once
	if !initialized {
		if err := initializeAPIServer(); err != nil {
			log.Printf("Failed to initialize API server: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		initialized = true
	}

	// Use Fiber's HTTP adaptor to handle the request with the existing API server
	// We need to access the fiber app directly since GetApp() method doesn't exist
	adaptor.FiberApp(apiServer.GetFiberApp())(w, r)
}

// initializeAPIServer initializes the API server using existing code
func initializeAPIServer() error {
	// Initialize database
	dbPath, err := getDatabasePath()
	if err != nil {
		return fmt.Errorf("failed to get database path: %w", err)
	}

	// Initialize database service using the same approach as main.go
	dbService, err := services.NewSqliteDBService(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize services using the server initialization helper
	_, txService, _, _, hookService, chainService, _, _, _ := server.InitializeServices(dbService.GetDB())

	// Create the API server using the existing constructor
	apiServer = api.NewAPIServer(dbService, txService, hookService, chainService)

	// Add a root route for Vercel
	apiServer.GetFiberApp().Get("/", func(c *fiber.Ctx) error {
		return c.JSON(map[string]interface{}{
			"message": "Launchpad MCP API",
			"status":  "running",
			"version": "1.0.0",
		})
	})

	return nil
}

// getDatabasePath returns the appropriate database path for Vercel environment
func getDatabasePath() (string, error) {
	// In Vercel, we need to use /tmp for writable storage
	// But for development, we can use the home directory
	if os.Getenv("VERCEL") == "1" {
		return "/tmp/launchpad.db", nil
	}

	// For local development, use home directory
	homePath, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homePath, "launchpad.db"), nil
}
