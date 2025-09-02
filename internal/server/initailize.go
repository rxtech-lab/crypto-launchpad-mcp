package server

import (
	"log"

	"github.com/rxtech-lab/launchpad-mcp/internal/hooks"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

func InitializeServices(db *gorm.DB) (services.EvmService, services.TransactionService, services.UniswapService, services.LiquidityService, services.HookService, services.ChainService, services.TemplateService, services.UniswapSettingsService, *services.DeploymentService) {
	evmService := services.NewEvmService()
	txService := services.NewTransactionService(db)
	uniswapService := services.NewUniswapService(db)
	liquidityService := services.NewLiquidityService(db)
	hookService := services.NewHookService()
	chainService := services.NewChainService(db)
	templateService := services.NewTemplateService(db)
	uniswapSettingsService := services.NewUniswapSettingsService(db)
	deploymentService := services.NewDeploymentService(db)

	return evmService, txService, uniswapService, liquidityService, hookService, chainService, templateService, uniswapSettingsService, deploymentService
}

func InitializeHooks(db *gorm.DB, hookService services.HookService, uniswapService services.UniswapService) (services.Hook, services.Hook) {
	tokenDeploymentHook := hooks.NewTokenDeploymentHook(db)
	uniswapDeploymentHook := hooks.NewUniswapDeploymentHook(db, uniswapService)

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
