package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	_ "github.com/joho/godotenv/autoload" // Automatically load .env file if present
	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

func configureAndStartServer(db *gorm.DB, port int) (*api.APIServer, int, error) {
	// Create database service wrapper
	var dbService services.DBService
	var err error
	if db != nil {
		// Use provided DB connection (for testing)
		dbService = services.NewDBServiceFromDB(db)
	} else {
		// Initialize postgres database from environment
		postgresUrl := os.Getenv("POSTGRES_URL")
		dbService, err = services.NewPostgresDBService(postgresUrl)
		if err != nil {
			return nil, 0, err
		}
	}

	// Initialize services and hooks
	evmService, txService, uniswapService, liquidityService, hookService, chainService, templateService, deploymentService := server.InitializeServices(dbService.GetDB())
	tokenDeploymentHook, uniswapDeploymentHook, liquidityHook := server.InitializeHooks(dbService.GetDB(), hookService, uniswapService, deploymentService, liquidityService)
	server.RegisterHooks(hookService, tokenDeploymentHook, uniswapDeploymentHook, liquidityHook)

	// Initialize MCP server
	mcpServer := mcp.NewMCPServer(dbService, port, evmService, txService, uniswapService, liquidityService, chainService, templateService, deploymentService)
	// Initialize API server for transaction signing (authenticator is created internally)
	apiServer := api.NewAPIServer(dbService, txService, hookService, chainService)
	if os.Getenv("DISABLE_AUTHENTICATION") != "true" {
		apiServer.EnableAuthentication()
	} else {
		log.Println("Warning: Authentication is disabled")
	}
	apiServer.SetupRoutes()
	apiServer.SetMCPServer(mcpServer)
	apiServer.EnableStreamableHttp()
	// Start API server
	var portPtr *int
	if port != 0 {
		portPtr = &port
	}
	startedPort, err := apiServer.Start(portPtr)
	if err != nil {
		return nil, 0, err
	}

	return apiServer, startedPort, nil
}

func main() {
	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	parsedPort, err := strconv.Atoi(port)
	if err != nil {
		log.Fatal("Invalid port number:", err)
	}

	// Configure and start server
	apiServer, startedPort, err := configureAndStartServer(nil, parsedPort)
	if err != nil {
		log.Fatal("Failed to start API server:", err)
	}

	log.Printf("API server started on port %d\n", startedPort)

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("\nShutting down server...")

	// Shutdown API server
	if err := apiServer.Shutdown(); err != nil {
		log.Printf("Error shutting down API server: %v", err)
	}

	log.Println("Server shut down successfully")
}
