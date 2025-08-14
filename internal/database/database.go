package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: nil, // Disable GORM logging to prevent color output
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	database := &Database{DB: db}
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return database, nil
}

func (d *Database) migrate() error {
	return d.DB.AutoMigrate(
		&models.Chain{},
		&models.Template{},
		&models.Deployment{},
		&models.UniswapSettings{},
		&models.LiquidityPool{},
		&models.LiquidityPosition{},
		&models.SwapTransaction{},
		&models.TransactionSession{},
	)
}

// Chain operations
func (d *Database) CreateChain(chain *models.Chain) error {
	return d.DB.Create(chain).Error
}

func (d *Database) GetActiveChain() (*models.Chain, error) {
	var chain models.Chain
	err := d.DB.Where("is_active = ?", true).First(&chain).Error
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

func (d *Database) SetActiveChain(chainType string) error {
	// Deactivate all chains
	if err := d.DB.Model(&models.Chain{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Activate the selected chain
	return d.DB.Model(&models.Chain{}).Where("chain_type = ?", chainType).Update("is_active", true).Error
}

func (d *Database) UpdateChainConfig(chainType, rpc, chainID string) error {
	return d.DB.Model(&models.Chain{}).
		Where("chain_type = ?", chainType).
		Updates(map[string]interface{}{
			"rpc":      rpc,
			"chain_id": chainID,
		}).Error
}

func (d *Database) ListChains() ([]models.Chain, error) {
	var chains []models.Chain
	err := d.DB.Find(&chains).Error
	return chains, err
}

// Template operations
func (d *Database) CreateTemplate(template *models.Template) error {
	return d.DB.Create(template).Error
}

func (d *Database) GetTemplateByID(id uint) (*models.Template, error) {
	var template models.Template
	err := d.DB.First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (d *Database) ListTemplates(chainType, keyword string, limit int) ([]models.Template, error) {
	query := d.DB.Model(&models.Template{})

	if chainType != "" {
		query = query.Where("chain_type = ?", chainType)
	}

	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	var templates []models.Template
	err := query.Find(&templates).Error
	return templates, err
}

func (d *Database) UpdateTemplate(template *models.Template) error {
	return d.DB.Save(template).Error
}

func (d *Database) DeleteTemplate(id uint) error {
	return d.DB.Delete(&models.Template{}, id).Error
}

// Deployment operations
func (d *Database) CreateDeployment(deployment *models.Deployment) error {
	return d.DB.Create(deployment).Error
}

func (d *Database) GetDeploymentByID(id uint) (*models.Deployment, error) {
	var deployment models.Deployment
	err := d.DB.Preload("Template").First(&deployment, id).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

func (d *Database) UpdateDeploymentStatus(id uint, status, contractAddress, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if contractAddress != "" {
		updates["contract_address"] = contractAddress
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return d.DB.Model(&models.Deployment{}).Where("id = ?", id).Updates(updates).Error
}

func (d *Database) ListDeployments() ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := d.DB.Preload("Template").Find(&deployments).Error
	return deployments, err
}

// Uniswap Settings operations
func (d *Database) SetUniswapVersion(version string) error {
	// Deactivate all versions
	if err := d.DB.Model(&models.UniswapSettings{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Create or activate the selected version
	var setting models.UniswapSettings
	err := d.DB.Where("version = ?", version).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		// Create new setting
		setting = models.UniswapSettings{
			Version:  version,
			IsActive: true,
		}
		return d.DB.Create(&setting).Error
	} else if err != nil {
		return err
	}

	// Activate existing setting
	return d.DB.Model(&setting).Update("is_active", true).Error
}

func (d *Database) GetActiveUniswapSettings() (*models.UniswapSettings, error) {
	var settings models.UniswapSettings
	err := d.DB.Where("is_active = ?", true).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// Liquidity Pool operations
func (d *Database) CreateLiquidityPool(pool *models.LiquidityPool) error {
	return d.DB.Create(pool).Error
}

func (d *Database) GetLiquidityPoolByTokenAddress(tokenAddress string) (*models.LiquidityPool, error) {
	var pool models.LiquidityPool
	err := d.DB.Where("token_address = ?", tokenAddress).First(&pool).Error
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

func (d *Database) UpdateLiquidityPoolStatus(id uint, status, pairAddress, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if pairAddress != "" {
		updates["pair_address"] = pairAddress
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return d.DB.Model(&models.LiquidityPool{}).Where("id = ?", id).Updates(updates).Error
}

func (d *Database) ListLiquidityPools() ([]models.LiquidityPool, error) {
	var pools []models.LiquidityPool
	err := d.DB.Find(&pools).Error
	return pools, err
}

// Liquidity Position operations
func (d *Database) CreateLiquidityPosition(position *models.LiquidityPosition) error {
	return d.DB.Create(position).Error
}

func (d *Database) UpdateLiquidityPositionStatus(id uint, status, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return d.DB.Model(&models.LiquidityPosition{}).Where("id = ?", id).Updates(updates).Error
}

func (d *Database) GetLiquidityPositionsByUser(userAddress string) ([]models.LiquidityPosition, error) {
	var positions []models.LiquidityPosition
	err := d.DB.Preload("Pool").Where("user_address = ?", userAddress).Find(&positions).Error
	return positions, err
}

// Swap Transaction operations
func (d *Database) CreateSwapTransaction(swap *models.SwapTransaction) error {
	return d.DB.Create(swap).Error
}

func (d *Database) UpdateSwapTransactionStatus(id uint, status, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return d.DB.Model(&models.SwapTransaction{}).Where("id = ?", id).Updates(updates).Error
}

func (d *Database) GetSwapTransactionsByUser(userAddress string) ([]models.SwapTransaction, error) {
	var swaps []models.SwapTransaction
	err := d.DB.Where("user_address = ?", userAddress).Find(&swaps).Error
	return swaps, err
}

// Transaction Session operations
func (d *Database) CreateTransactionSession(sessionType, chainType, chainID, transactionData string) (string, error) {
	sessionID := uuid.New().String()
	session := &models.TransactionSession{
		ID:              sessionID,
		SessionType:     sessionType,
		ChainType:       chainType,
		ChainID:         chainID,
		TransactionData: transactionData,
		Status:          "pending",
		ExpiresAt:       time.Now().Add(30 * time.Minute), // 30 minute expiry
	}

	err := d.DB.Create(session).Error
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

func (d *Database) GetTransactionSession(sessionID string) (*models.TransactionSession, error) {
	var session models.TransactionSession
	err := d.DB.Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

func (d *Database) UpdateTransactionSessionStatus(sessionID, status, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return d.DB.Model(&models.TransactionSession{}).Where("id = ?", sessionID).Updates(updates).Error
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
