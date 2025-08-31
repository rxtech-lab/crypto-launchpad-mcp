package streamable_http

import (
	"fmt"
	"log"
	"os"

	"github.com/rxtech-lab/launchpad-mcp/internal/server"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// initialize postgres database with postgres://user:password@localhost:5432/launchpad?sslmode=disable
	postgresUrl := os.Getenv("POSTGRES_URL")
	if postgresUrl == "" {
		log.Fatal("POSTGRES_URL environment variable is required")
	}

	db, err := gorm.Open(postgres.Open(postgresUrl), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize services and hooks - capture all 9 return values
	evmService, txService, uniswapService, liquidityService, hookService, chainService, templateService, uniswapSettingsService, deploymentService := server.InitializeServices(db)
	tokenDeploymentHook, uniswapDeploymentHook := server.InitializeHooks(db, hookService)

	server.RegisterHooks(hookService, tokenDeploymentHook, uniswapDeploymentHook)

	// TODO: Add HTTP server implementation using the initialized services
	fmt.Printf("Services initialized successfully:\n")
	fmt.Printf("- EVM Service: %v\n", evmService != nil)
	fmt.Printf("- Transaction Service: %v\n", txService != nil)
	fmt.Printf("- Uniswap Service: %v\n", uniswapService != nil)
	fmt.Printf("- Liquidity Service: %v\n", liquidityService != nil)
	fmt.Printf("- Hook Service: %v\n", hookService != nil)
	fmt.Printf("- Chain Service: %v\n", chainService != nil)
	fmt.Printf("- Template Service: %v\n", templateService != nil)
	fmt.Printf("- Uniswap Settings Service: %v\n", uniswapSettingsService != nil)
	fmt.Printf("- Deployment Service: %v\n", deploymentService != nil)
}
