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
)

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

	// initialize postgres database
	postgresUrl := os.Getenv("POSTGRES_URL")
	// Create database service wrapper
	dbService, err := services.NewPostgresDBService(postgresUrl)
	if err != nil {
		log.Fatal("Failed to initialize database service:", err)
	}
	// Initialize services and hooks
	evmService, txService, uniswapService, liquidityService, hookService, chainService, templateService, uniswapSettingsService, deploymentService := server.InitializeServices(dbService.GetDB())
	tokenDeploymentHook, uniswapDeploymentHook := server.InitializeHooks(dbService.GetDB(), hookService)
	server.RegisterHooks(hookService, tokenDeploymentHook, uniswapDeploymentHook)

	// Initialize MCP server
	mcpServer := mcp.NewMCPServer(dbService, parsedPort, evmService, txService, uniswapService, liquidityService, chainService, templateService, uniswapSettingsService, deploymentService)
	// Initialize API server for transaction signing (authenticator is created internally)
	apiServer := api.NewAPIServer(dbService, txService, hookService, chainService)
	apiServer.SetMCPServer(mcpServer)
	apiServer.EnableStreamableHttp()
	// Start API server
	startedPort, err := apiServer.Start(&parsedPort)
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
