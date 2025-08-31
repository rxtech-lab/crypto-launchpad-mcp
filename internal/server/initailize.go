package server

import (
	"log"

	"github.com/rxtech-lab/launchpad-mcp/internal/hooks"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

func InitializeServices(db *gorm.DB) (services.EvmService, services.TransactionService, services.UniswapService, services.LiquidityService, services.HookService) {
	evmService := services.NewEvmService()
	txService := services.NewTransactionService(db)
	uniswapService := services.NewUniswapService(db)
	liquidityService := services.NewLiquidityService(db)
	hookService := services.NewHookService()

	return evmService, txService, uniswapService, liquidityService, hookService
}

func InitializeHooks(db *gorm.DB, hookService services.HookService) (services.Hook, services.Hook) {
	tokenDeploymentHook := hooks.NewTokenDeploymentHook(db)
	uniswapDeploymentHook := hooks.NewUniswapDeploymentHook(db)

	return tokenDeploymentHook, uniswapDeploymentHook
}

func RegisterHooks(hookService services.HookService, tokenDeploymentHook services.Hook, uniswapDeploymentHook services.Hook) {
	if err := hookService.AddHook(tokenDeploymentHook); err != nil {
		log.Fatal("Failed to register token deployment hook:", err)
	}
	if err := hookService.AddHook(uniswapDeploymentHook); err != nil {
		log.Fatal("Failed to register uniswap deployment hook:", err)
	}
}
