package hooks

import (
	"fmt"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"gorm.io/gorm"
)

type LiquidityPoolHook struct {
	db                     *gorm.DB
	liquidityService       services.LiquidityService
	uniswapContractService services.UniswapContractService
	chainService           services.ChainService
}

// CanHandle implements Hook.
func (l *LiquidityPoolHook) CanHandle(txType models.TransactionType) bool {
	return txType == models.TransactionTypeLiquidityPoolCreation
}

// OnTransactionConfirmed implements Hook.
func (l *LiquidityPoolHook) OnTransactionConfirmed(txType models.TransactionType, txHash string, contractAddress *string, session models.TransactionSession) error {
	switch txType {
	case models.TransactionTypeLiquidityPoolCreation:
		// only handle the creation of the liquidity pool
		// position should be fetched from the blockchain
		return l.handleLiquidityPoolCreation(txHash, session)
	default:
		return nil
	}
}

// handleLiquidityPoolCreation updates the liquidity pool with the confirmed transaction details
func (l *LiquidityPoolHook) handleLiquidityPoolCreation(txHash string, session models.TransactionSession) error {
	// find the pool by session id
	pool, err := l.liquidityService.GetLiquidityPoolBySessionId(session.ID)
	if err != nil {
		return err
	}

	// Get token addresses from session metadata
	token0Address, token1Address, err := l.getTokenAddressesFromSession(session)
	if err != nil {
		return fmt.Errorf("failed to get token addresses: %w", err)
	}

	// Get pair address from Uniswap Factory contract
	pairAddress, err := l.uniswapContractService.GetPairAddress(token0Address, token1Address, &session.Chain)
	if err != nil {
		return fmt.Errorf("failed to get pair address: %w", err)
	}

	// Update the pool with transaction hash and pair address
	return l.liquidityService.UpdateLiquidityPoolStatus(pool.ID, models.TransactionStatusConfirmed, pairAddress, txHash)
}

// getTokenAddressesFromSession extracts token addresses from transaction session metadata
func (l *LiquidityPoolHook) getTokenAddressesFromSession(session models.TransactionSession) (string, string, error) {
	var token0Address, token1Address string

	// Extract from metadata
	for _, meta := range session.Metadata {
		if meta.Key == services.MetadataToken0Address {
			token0Address = meta.Value
		}
		if meta.Key == services.MetadataToken1Address {
			token1Address = meta.Value
		}
	}

	if token0Address == "" || token1Address == "" {
		return "", "", fmt.Errorf("token addresses not found in session metadata")
	}

	return token0Address, token1Address, nil
}

func NewLiquidityPoolHook(db *gorm.DB, liquidityService services.LiquidityService, uniswapContractService services.UniswapContractService, chainService services.ChainService) services.Hook {
	return &LiquidityPoolHook{
		db:                     db,
		liquidityService:       liquidityService,
		uniswapContractService: uniswapContractService,
		chainService:           chainService,
	}
}
