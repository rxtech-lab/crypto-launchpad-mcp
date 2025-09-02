package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/server"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

// Build information (set via ldflags)
var (
	Version    = "dev"
	CommitHash = "unknown"
	BuildTime  = "unknown"
)

func configureAndStartServer(dbService services.DBService, port int) (*api.APIServer, int, error) {
	// Initialize services and hooks
	evmService, txService, uniswapService, liquidityService, hookService, chainService, templateService, uniswapSettingsService, deploymentService := server.InitializeServices(dbService.GetDB())
	tokenDeploymentHook, uniswapDeploymentHook := server.InitializeHooks(dbService.GetDB(), hookService, uniswapService)
	server.RegisterHooks(hookService, tokenDeploymentHook, uniswapDeploymentHook)

	// Initialize API server (HTTP server for transaction signing) - NO AUTHENTICATION
	apiServer := api.NewAPIServer(dbService, txService, hookService, chainService)

	// Setup routes WITHOUT enabling authentication (key difference from streamable-http)
	apiServer.SetupRoutes()
	// NOTE: NOT calling EnableAuthentication() or EnableStreamableHttp()

	// Start API server first to get the actual port
	var portPtr *int
	if port != 0 {
		portPtr = &port
	}
	startedPort, err := apiServer.Start(portPtr)
	if err != nil {
		return nil, 0, err
	}

	// Now initialize MCP server with the actual port
	mcpServer := mcp.NewMCPServer(dbService, startedPort, evmService, txService, uniswapService, liquidityService, chainService, templateService, uniswapSettingsService, deploymentService)
	apiServer.SetMCPServer(mcpServer)

	return apiServer, startedPort, nil
}

func main() {
	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	var enableLog = flag.Bool("log", false, "Enable logging output")
	flag.Parse()

	// Disable logging by default
	if !*enableLog {
		log.SetOutput(io.Discard)
	}

	// Show version information
	if *showVersion {
		log.Printf("Crypto Launchpad MCP Server\n")
		log.Printf("Version: %s\n", Version)
		log.Printf("Commit: %s\n", CommitHash)
		log.Printf("Built: %s\n", BuildTime)
		return
	}

	if *showHelp {
		log.Printf("Crypto Launchpad MCP Server\n\n")
		log.Printf("Usage: %s [options]\n\n", os.Args[0])
		log.Printf("Options:\n")
		log.Printf("  --version    Show version information\n")
		log.Printf("  --help       Show this help message\n")
		log.Printf("  --log        Enable logging output\n\n")
		log.Printf("Description:\n")
		log.Printf("  AI-powered crypto launchpad supporting Ethereum and Solana blockchains.\n")
		log.Printf("  Provides 17 MCP tools for token deployment and Uniswap integration.\n\n")
		log.Printf("Database: ~/launchpad.db (SQLite)\n")
		log.Printf("Web Interface: http://localhost:[random-port]\n")
		return
	}

	// Get home directory for database
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	// Initialize database
	dbPath := homePath + "/launchpad.db"
	dbService, err := services.NewSqliteDBService(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer dbService.Close()

	// Configure and start server
	apiServer, port, err := configureAndStartServer(dbService, 0) // 0 for random port
	if err != nil {
		log.Fatal("Failed to start API server:", err)
	}

	log.Printf("API server started on port %d\n", port)

	// Get MCP server for stdio communication
	mcpServer := apiServer.GetMCPServer()
	if mcpServer == nil {
		log.Fatal("MCP server not found")
	}

	// StartStdioServer MCP server in a goroutine
	go func() {
		if err := mcpServer.StartStdioServer(); err != nil {
			log.SetOutput(os.Stderr)
			log.SetFlags(0)
			log.Fatal("Failed to start MCP server:", err)
		}
	}()

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("\nShutting down servers...")

	// Shutdown API server
	if err := apiServer.Shutdown(); err != nil {
		log.SetOutput(os.Stderr)
		log.SetFlags(0)
		log.Printf("Error shutting down API server: %v", err)
	}

	log.Println("Servers shut down successfully")
}
