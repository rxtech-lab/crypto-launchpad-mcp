package hooks

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
)

type TokenDeploymentHook struct {
	deploymentService services.DeploymentService
}

// CanHandle implements Hook.
func (t *TokenDeploymentHook) CanHandle(txType models.TransactionType) bool {
	return txType == models.TransactionTypeTokenDeployment ||
		txType == models.TransactionTypeUniswapV2TokenDeployment
}

// OnTransactionConfirmed implements Hook.
func (t *TokenDeploymentHook) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress *string, session models.TransactionSession) error {
	// Update the deployment record with the contract address and confirmed status
	err := t.deploymentService.UpdateDeploymentStatusWithTxHashBySessionId(session.ID, models.TransactionStatusConfirmed, *contractAddress, txHash)

	if err != nil {
		return err
	}

	return nil
}

func NewTokenDeploymentHook(deploymentService services.DeploymentService) services.Hook {
	return &TokenDeploymentHook{
		deploymentService: deploymentService,
	}
}
