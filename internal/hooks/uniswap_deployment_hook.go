package hooks

import (
	"fmt"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

type UniswapDeploymentHook struct {
	db             *gorm.DB
	uniswapService services.UniswapService
}

// CanHandle implements Hook.
func (u *UniswapDeploymentHook) CanHandle(txType models.TransactionType) bool {
	return txType == models.TransactionTypeUniswapV2TokenDeployment ||
		txType == models.TransactionTypeUniswapV2FactoryDeployment ||
		txType == models.TransactionTypeUniswapV2RouterDeployment
}

// OnTransactionConfirmed implements Hook.
func (u *UniswapDeploymentHook) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress *string, session models.TransactionSession) error {
	currentDeployment, err := u.uniswapService.GetUniswapDeploymentByChain(session.ChainID)
	if err != nil {
		return err
	}

	if currentDeployment == nil {
		return fmt.Errorf("no uniswap deployment found for chain %d", session.ChainID)
	}

	switch txType {
	case models.TransactionTypeUniswapV2TokenDeployment:
		if err := u.uniswapService.UpdateWETHAddress(currentDeployment.ID, *contractAddress); err != nil {
			return err
		}
	case models.TransactionTypeUniswapV2FactoryDeployment:
		if err := u.uniswapService.UpdateFactoryAddress(currentDeployment.ID, *contractAddress); err != nil {
			return err
		}
	case models.TransactionTypeUniswapV2RouterDeployment:
		if err := u.uniswapService.UpdateRouterAddress(currentDeployment.ID, *contractAddress); err != nil {
			return err
		}
	}

	// Check if all three addresses are now set, and if so, mark deployment as confirmed
	updatedDeployment, err := u.uniswapService.GetUniswapDeployment(currentDeployment.ID)
	if err != nil {
		return err
	}

	if updatedDeployment.WETHAddress != "" && updatedDeployment.FactoryAddress != "" && updatedDeployment.RouterAddress != "" {
		return u.uniswapService.UpdateStatus(currentDeployment.ID, models.TransactionStatusConfirmed)
	}

	return nil
}

func NewUniswapDeploymentHook(db *gorm.DB, uniswapService services.UniswapService) services.Hook {
	return &UniswapDeploymentHook{
		db:             db,
		uniswapService: uniswapService,
	}
}
