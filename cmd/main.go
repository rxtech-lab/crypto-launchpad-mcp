package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/database"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
)

// Build information (set via ldflags)
var (
	Version    = "dev"
	CommitHash = "unknown"
	BuildTime  = "unknown"
)

func main() {
	// Command line flags
	var showVersion = flag.Bool("version", false, "Show version information")
	var showHelp = flag.Bool("help", false, "Show help information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Crypto Launchpad MCP Server\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Commit: %s\n", CommitHash)
		fmt.Printf("Built: %s\n", BuildTime)
		return
	}

	if *showHelp {
		fmt.Printf("Crypto Launchpad MCP Server\n\n")
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		fmt.Printf("  --version    Show version information\n")
		fmt.Printf("  --help       Show this help message\n\n")
		fmt.Printf("Description:\n")
		fmt.Printf("  AI-powered crypto launchpad supporting Ethereum and Solana blockchains.\n")
		fmt.Printf("  Provides 14 MCP tools for token deployment and Uniswap integration.\n\n")
		fmt.Printf("Database: ~/launchpad.db (SQLite)\n")
		fmt.Printf("Web Interface: http://localhost:[random-port]\n")
		return
	}

	// Get home directory for database
	homePath, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to get home directory:", err)
	}

	// Initialize database
	dbPath := homePath + "/launchpad.db"
	db, err := database.NewDatabase(dbPath)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Initialize and start API server (HTTP server for transaction signing)
	apiServer := api.NewAPIServer(db)

	// Start API server and get the assigned port
	port, err := apiServer.Start()
	if err != nil {
		log.Fatal("Failed to start API server:", err)
	}

	log.Printf("API server started on port %d\n", port)

	// Initialize MCP server with the API server port
	mcpServer := mcp.NewMCPServer(db, port)

	// Start MCP server in a goroutine
	go func() {
		if err := mcpServer.Start(); err != nil {
			log.SetOutput(os.Stderr)
			log.SetFlags(0)
			log.Fatal("Failed to start MCP server:", err)
		}
	}()

	// Set up graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nShutting down servers...")

	// Shutdown API server
	if err := apiServer.Shutdown(); err != nil {
		log.SetOutput(os.Stderr)
		log.SetFlags(0)
		log.Printf("Error shutting down API server: %v", err)
	}

	fmt.Println("Servers shut down successfully")
}
