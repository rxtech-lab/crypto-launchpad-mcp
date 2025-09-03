package server

import (
	"log"

	"github.com/rxtech-lab/launchpad-mcp/internal/hooks"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

func InitializeServices(db *gorm.DB) (services.EvmService, services.TransactionService, services.UniswapService, services.LiquidityService, services.HookService, services.ChainService, services.TemplateService, services.DeploymentService) {
	evmService := services.NewEvmService()
	txService := services.NewTransactionService(db)
	uniswapService := services.NewUniswapService(db)
	liquidityService := services.NewLiquidityService(db)
	hookService := services.NewHookService()
	chainService := services.NewChainService(db)
	templateService := services.NewTemplateService(db)
	deploymentService := services.NewDeploymentService(db)

	return evmService, txService, uniswapService, liquidityService, hookService, chainService, templateService, deploymentService
}

func InitializeHooks(db *gorm.DB, hookService services.HookService, uniswapService services.UniswapService, deploymentService services.DeploymentService, liquidityService services.LiquidityService) (services.Hook, services.Hook, services.Hook) {
	tokenDeploymentHook := hooks.NewTokenDeploymentHook(deploymentService)
	uniswapDeploymentHook := hooks.NewUniswapDeploymentHook(db, uniswapService)
	liquidityHook := hooks.NewLiquidityPoolHook(db, liquidityService)

	return tokenDeploymentHook, uniswapDeploymentHook, liquidityHook
}

func RegisterHooks(hookService services.HookService, hooks ...services.Hook) {
	for _, hook := range hooks {
		if err := hookService.AddHook(hook); err != nil {
			log.Fatal("Failed to register hook:", err)
		}
	}
}
