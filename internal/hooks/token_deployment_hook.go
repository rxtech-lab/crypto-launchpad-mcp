package hooks

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

type TokenDeploymentHook struct {
	db *gorm.DB
}

// CanHandle implements Hook.
func (t *TokenDeploymentHook) CanHandle(txType models.TransactionType) bool {
	return txType == models.TransactionTypeTokenDeployment ||
		txType == models.TransactionTypeUniswapV2TokenDeployment
}

// OnTransactionConfirmed implements Hook.
func (t *TokenDeploymentHook) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress string, session models.TransactionSession) error {
	// Update the deployment record with the contract address and confirmed status
	err := t.db.Model(&models.Deployment{}).
		Where("transaction_hash = ?", txHash).
		Updates(map[string]interface{}{
			"contract_address": contractAddress,
			"status":           string(models.TransactionStatusConfirmed),
		}).Error

	if err != nil {
		return err
	}

	return nil
}

func NewTokenDeploymentHook(db *gorm.DB) services.Hook {
	return &TokenDeploymentHook{
		db: db,
	}
}
