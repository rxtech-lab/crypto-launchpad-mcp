package services

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

type LiquidityService interface {
	// Liquidity Pool operations
	CreateLiquidityPool(pool *models.LiquidityPool) (uint, error)
	GetLiquidityPool(poolID uint) (*models.LiquidityPool, error)
	GetLiquidityPoolByTokenAddress(tokenAddressA string, tokenAddressB string) (*models.LiquidityPool, error)
	UpdateLiquidityPoolStatus(poolID uint, status models.TransactionStatus, pairAddress, txHash string) error
	UpdateLiquidityPoolPairAddress(poolID uint, pairAddress string) error
	ListLiquidityPools(skip, limit int) ([]models.LiquidityPool, error)
	ListLiquidityPoolsByUser(userID string, skip, limit int) ([]models.LiquidityPool, error)
	GetLiquidityPoolBySessionId(sessionId string) (*models.LiquidityPool, error)
}

type liquidityService struct {
	db *gorm.DB
}

func NewLiquidityService(db *gorm.DB) LiquidityService {
	return &liquidityService{db: db}
}

// Liquidity Pool operations
// GetLiquidityPoolBySessionId implements LiquidityService.
func (l *liquidityService) GetLiquidityPoolBySessionId(sessionId string) (*models.LiquidityPool, error) {
	var pool models.LiquidityPool
	err := l.db.Where("session_id = ?", sessionId).First(&pool).Error
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

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

func (l *liquidityService) GetLiquidityPoolByTokenAddress(tokenAddressA string, tokenAddressB string) (*models.LiquidityPool, error) {
	var pool models.LiquidityPool
	err := l.db.Where("token_address = ? OR token_address = ?", tokenAddressA, tokenAddressB).First(&pool).Error
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
