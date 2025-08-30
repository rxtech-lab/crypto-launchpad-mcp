package hooks

import (
	"fmt"
	"strconv"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

type LiquidityPoolHook struct {
	db               *gorm.DB
	liquidityService services.LiquidityService
}

// CanHandle implements Hook.
func (l *LiquidityPoolHook) CanHandle(txType models.TransactionType) bool {
	return txType == models.TransactionTypeLiquidityPoolCreation ||
		txType == models.TransactionTypeAddLiquidity ||
		txType == models.TransactionTypeRemoveLiquidity
}

// OnTransactionConfirmed implements Hook.
func (l *LiquidityPoolHook) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress string, session models.TransactionSession) error {
	switch txType {
	case models.TransactionTypeLiquidityPoolCreation:
		return l.handleLiquidityPoolCreation(txHash, contractAddress, session)
	case models.TransactionTypeAddLiquidity:
		return l.handleAddLiquidity(txHash, session)
	case models.TransactionTypeRemoveLiquidity:
		return l.handleRemoveLiquidity(txHash, session)
	default:
		return fmt.Errorf("unsupported transaction type: %s", txType)
	}
}

// handleLiquidityPoolCreation updates the liquidity pool with the confirmed transaction details
func (l *LiquidityPoolHook) handleLiquidityPoolCreation(txHash string, pairAddress string, session models.TransactionSession) error {
	// Find pool ID from metadata
	var poolID uint
	for _, metadata := range session.Metadata {
		if metadata.Key == "pool_id" {
			id, err := strconv.ParseUint(metadata.Value, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid pool_id in metadata: %v", err)
			}
			poolID = uint(id)
			break
		}
	}

	if poolID == 0 {
		return fmt.Errorf("pool_id not found in session metadata")
	}

	// Update the pool with transaction hash and pair address
	return l.liquidityService.UpdateLiquidityPoolStatus(poolID, models.TransactionStatusConfirmed, pairAddress, txHash)
}

// handleAddLiquidity updates the liquidity position with the confirmed transaction details
func (l *LiquidityPoolHook) handleAddLiquidity(txHash string, session models.TransactionSession) error {
	// Find position ID from metadata
	var positionID uint
	for _, metadata := range session.Metadata {
		if metadata.Key == "position_id" {
			id, err := strconv.ParseUint(metadata.Value, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid position_id in metadata: %v", err)
			}
			positionID = uint(id)
			break
		}
	}

	if positionID == 0 {
		return fmt.Errorf("position_id not found in session metadata")
	}

	// Update the position with transaction hash
	return l.liquidityService.UpdateLiquidityPositionStatus(positionID, models.TransactionStatusConfirmed, txHash)
}

// handleRemoveLiquidity updates the liquidity position with the confirmed transaction details
func (l *LiquidityPoolHook) handleRemoveLiquidity(txHash string, session models.TransactionSession) error {
	// Find position ID from metadata
	var positionID uint
	for _, metadata := range session.Metadata {
		if metadata.Key == "position_id" {
			id, err := strconv.ParseUint(metadata.Value, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid position_id in metadata: %v", err)
			}
			positionID = uint(id)
			break
		}
	}

	if positionID == 0 {
		return fmt.Errorf("position_id not found in session metadata")
	}

	// Update the position with transaction hash
	return l.liquidityService.UpdateLiquidityPositionStatus(positionID, models.TransactionStatusConfirmed, txHash)
}

func NewLiquidityPoolHook(db *gorm.DB, liquidityService services.LiquidityService) services.Hook {
	return &LiquidityPoolHook{
		db:               db,
		liquidityService: liquidityService,
	}
}
