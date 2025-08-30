package services

import (
	"errors"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

type LiquidityService interface {
	// Liquidity Pool operations
	CreateLiquidityPool(pool *models.LiquidityPool) (uint, error)
	GetLiquidityPool(poolID uint) (*models.LiquidityPool, error)
	GetLiquidityPoolByTokenAddress(tokenAddress string) (*models.LiquidityPool, error)
	UpdateLiquidityPoolStatus(poolID uint, status models.TransactionStatus, pairAddress, txHash string) error
	UpdateLiquidityPoolPairAddress(poolID uint, pairAddress string) error
	ListLiquidityPools(skip, limit int) ([]models.LiquidityPool, error)
	ListLiquidityPoolsByUser(userID string, skip, limit int) ([]models.LiquidityPool, error)

	// Liquidity Position operations
	CreateLiquidityPosition(position *models.LiquidityPosition) (uint, error)
	GetLiquidityPosition(positionID uint) (*models.LiquidityPosition, error)
	GetLiquidityPositionsByPool(poolID uint) ([]models.LiquidityPosition, error)
	GetLiquidityPositionsByUser(userAddress string) ([]models.LiquidityPosition, error)
	GetLiquidityPositionsByUserID(userID string) ([]models.LiquidityPosition, error)
	UpdateLiquidityPositionStatus(positionID uint, status models.TransactionStatus, txHash string) error
	UpdateLiquidityPositionAmounts(positionID uint, liquidityAmount, token0Amount, token1Amount string) error

	// Swap operations
	CreateSwapTransaction(swap *models.SwapTransaction) (uint, error)
	GetSwapTransaction(swapID uint) (*models.SwapTransaction, error)
	ListSwapTransactionsByUser(userID string, skip, limit int) ([]models.SwapTransaction, error)
	UpdateSwapTransactionStatus(swapID uint, status models.TransactionStatus, txHash string) error
}

type liquidityService struct {
	db *gorm.DB
}

func NewLiquidityService(db *gorm.DB) LiquidityService {
	return &liquidityService{db: db}
}

// Liquidity Pool operations

func (l *liquidityService) CreateLiquidityPool(pool *models.LiquidityPool) (uint, error) {
	if pool.Status == "" {
		pool.Status = models.TransactionStatusPending
	}

	err := l.db.Create(pool).Error
	if err != nil {
		return 0, err
	}
	return pool.ID, nil
}

func (l *liquidityService) GetLiquidityPool(poolID uint) (*models.LiquidityPool, error) {
	var pool models.LiquidityPool
	err := l.db.First(&pool, poolID).Error
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

func (l *liquidityService) GetLiquidityPoolByTokenAddress(tokenAddress string) (*models.LiquidityPool, error) {
	var pool models.LiquidityPool
	err := l.db.Where("token_address = ?", tokenAddress).First(&pool).Error
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

func (l *liquidityService) UpdateLiquidityPoolStatus(poolID uint, status models.TransactionStatus, pairAddress, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if pairAddress != "" {
		updates["pair_address"] = pairAddress
	}

	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return l.db.Model(&models.LiquidityPool{}).
		Where("id = ?", poolID).
		Updates(updates).Error
}

func (l *liquidityService) UpdateLiquidityPoolPairAddress(poolID uint, pairAddress string) error {
	return l.db.Model(&models.LiquidityPool{}).
		Where("id = ?", poolID).
		Update("pair_address", pairAddress).Error
}

func (l *liquidityService) ListLiquidityPools(skip, limit int) ([]models.LiquidityPool, error) {
	var pools []models.LiquidityPool
	err := l.db.Offset(skip).Limit(limit).Find(&pools).Error
	if err != nil {
		return nil, err
	}
	return pools, nil
}

func (l *liquidityService) ListLiquidityPoolsByUser(userID string, skip, limit int) ([]models.LiquidityPool, error) {
	var pools []models.LiquidityPool
	err := l.db.Where("user_id = ?", userID).Offset(skip).Limit(limit).Find(&pools).Error
	if err != nil {
		return nil, err
	}
	return pools, nil
}

// Liquidity Position operations

func (l *liquidityService) CreateLiquidityPosition(position *models.LiquidityPosition) (uint, error) {
	if position.Status == "" {
		position.Status = models.TransactionStatusPending
	}

	err := l.db.Create(position).Error
	if err != nil {
		return 0, err
	}
	return position.ID, nil
}

func (l *liquidityService) GetLiquidityPosition(positionID uint) (*models.LiquidityPosition, error) {
	var position models.LiquidityPosition
	err := l.db.Preload("Pool").First(&position, positionID).Error
	if err != nil {
		return nil, err
	}
	return &position, nil
}

func (l *liquidityService) GetLiquidityPositionsByPool(poolID uint) ([]models.LiquidityPosition, error) {
	var positions []models.LiquidityPosition
	err := l.db.Where("pool_id = ?", poolID).Preload("Pool").Find(&positions).Error
	if err != nil {
		return nil, err
	}
	return positions, nil
}

func (l *liquidityService) GetLiquidityPositionsByUser(userAddress string) ([]models.LiquidityPosition, error) {
	var positions []models.LiquidityPosition
	err := l.db.Where("user_address = ?", userAddress).Preload("Pool").Find(&positions).Error
	if err != nil {
		return nil, err
	}
	return positions, nil
}

func (l *liquidityService) UpdateLiquidityPositionStatus(positionID uint, status models.TransactionStatus, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return l.db.Model(&models.LiquidityPosition{}).
		Where("id = ?", positionID).
		Updates(updates).Error
}

func (l *liquidityService) UpdateLiquidityPositionAmounts(positionID uint, liquidityAmount, token0Amount, token1Amount string) error {
	updates := map[string]interface{}{}

	if liquidityAmount != "" {
		updates["liquidity_amount"] = liquidityAmount
	}
	if token0Amount != "" {
		updates["token0_amount"] = token0Amount
	}
	if token1Amount != "" {
		updates["token1_amount"] = token1Amount
	}

	if len(updates) == 0 {
		return errors.New("no amounts to update")
	}

	return l.db.Model(&models.LiquidityPosition{}).
		Where("id = ?", positionID).
		Updates(updates).Error
}

func (l *liquidityService) GetLiquidityPositionsByUserID(userID string) ([]models.LiquidityPosition, error) {
	var positions []models.LiquidityPosition
	err := l.db.Where("user_id = ?", userID).Preload("Pool").Find(&positions).Error
	if err != nil {
		return nil, err
	}
	return positions, nil
}

// Swap operations

func (l *liquidityService) CreateSwapTransaction(swap *models.SwapTransaction) (uint, error) {
	if swap.Status == "" {
		swap.Status = models.TransactionStatusPending
	}

	err := l.db.Create(swap).Error
	if err != nil {
		return 0, err
	}
	return swap.ID, nil
}

func (l *liquidityService) GetSwapTransaction(swapID uint) (*models.SwapTransaction, error) {
	var swap models.SwapTransaction
	err := l.db.First(&swap, swapID).Error
	if err != nil {
		return nil, err
	}
	return &swap, nil
}

func (l *liquidityService) ListSwapTransactionsByUser(userID string, skip, limit int) ([]models.SwapTransaction, error) {
	var swaps []models.SwapTransaction
	err := l.db.Where("user_id = ?", userID).Offset(skip).Limit(limit).Find(&swaps).Error
	if err != nil {
		return nil, err
	}
	return swaps, nil
}

func (l *liquidityService) UpdateSwapTransactionStatus(swapID uint, status models.TransactionStatus, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return l.db.Model(&models.SwapTransaction{}).
		Where("id = ?", swapID).
		Updates(updates).Error
}
