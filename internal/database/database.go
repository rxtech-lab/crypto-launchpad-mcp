package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	// Configure GORM logger - only log errors and slow queries
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Only log errors and slow queries
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,        // Include params in SQL log
			Colorful:                  false,        // Disable color
		},
	)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
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
		&models.UniswapDeployment{},
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

func (d *Database) SetActiveChainByID(chainID uint) error {
	// Deactivate all chains
	if err := d.DB.Model(&models.Chain{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Activate the selected chain by ID
	return d.DB.Model(&models.Chain{}).Where("id = ?", chainID).Update("is_active", true).Error
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

func (d *Database) DeleteTemplates(ids []uint) (int64, error) {
	result := d.DB.Delete(&models.Template{}, ids)
	return result.RowsAffected, result.Error
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

	// Manually load the Chain relationship
	var chain models.Chain
	err = d.DB.First(&chain, deployment.ChainID).Error
	if err != nil {
		return nil, err
	}
	deployment.Chain = chain

	return &deployment, nil
}

func (d *Database) UpdateDeploymentStatus(id uint, status models.TransactionStatus, contractAddress, txHash string) error {
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
	if err != nil {
		return nil, err
	}

	// Manually load Chain relationships
	for i := range deployments {
		var chain models.Chain
		err = d.DB.First(&chain, deployments[i].ChainID).Error
		if err != nil {
			return nil, err
		}
		deployments[i].Chain = chain
	}

	return deployments, nil
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

func (d *Database) SetUniswapConfiguration(version, routerAddress, factoryAddress, wethAddress, quoterAddress, positionManager, swapRouter02 string) error {
	// Deactivate all versions
	if err := d.DB.Model(&models.UniswapSettings{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Create or update the selected version with addresses
	var setting models.UniswapSettings
	err := d.DB.Where("version = ?", version).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		// Create new setting
		setting = models.UniswapSettings{
			Version:         version,
			RouterAddress:   routerAddress,
			FactoryAddress:  factoryAddress,
			WETHAddress:     wethAddress,
			QuoterAddress:   quoterAddress,
			PositionManager: positionManager,
			SwapRouter02:    swapRouter02,
			IsActive:        true,
		}
		return d.DB.Create(&setting).Error
	} else if err != nil {
		return err
	}

	// Update existing setting with new addresses and activate
	updates := map[string]interface{}{
		"router_address":   routerAddress,
		"factory_address":  factoryAddress,
		"weth_address":     wethAddress,
		"quoter_address":   quoterAddress,
		"position_manager": positionManager,
		"swap_router02":    swapRouter02,
		"is_active":        true,
	}
	return d.DB.Model(&setting).Updates(updates).Error
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

func (d *Database) GetLiquidityPoolByID(id uint) (*models.LiquidityPool, error) {
	var pool models.LiquidityPool
	err := d.DB.First(&pool, id).Error
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

func (d *Database) DeleteLiquidityPool(id uint) error {
	return d.DB.Delete(&models.LiquidityPool{}, id).Error
}

// Liquidity Position operations
func (d *Database) CreateLiquidityPosition(position *models.LiquidityPosition) error {
	return d.DB.Create(position).Error
}

func (d *Database) GetLiquidityPositionByID(id uint) (*models.LiquidityPosition, error) {
	var position models.LiquidityPosition
	err := d.DB.First(&position, id).Error
	if err != nil {
		return nil, err
	}
	return &position, nil
}

func (d *Database) UpdateLiquidityPositionStatus(id uint, status models.TransactionStatus, txHash string) error {
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

func (d *Database) GetSwapTransactionByID(id uint) (*models.SwapTransaction, error) {
	var swap models.SwapTransaction
	err := d.DB.First(&swap, id).Error
	if err != nil {
		return nil, err
	}
	return &swap, nil
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

func (d *Database) CreateTransactionSession(sessionType string, chainType models.TransactionChainType, chainID, data string) (string, error) {
	// Generate a UUID for the session ID
	sessionID := fmt.Sprintf("%s-%d", sessionType, time.Now().UnixNano())

	// Parse chainID to uint
	var chainIDUint uint
	if _, err := fmt.Sscanf(chainID, "%d", &chainIDUint); err != nil {
		// If chainID is not a number, try to find the chain by type and chainID string
		var chain models.Chain
		if err := d.DB.Where("chain_type = ? AND chain_id = ?", chainType, chainID).First(&chain).Error; err != nil {
			return "", fmt.Errorf("failed to find chain: %w", err)
		}
		chainIDUint = chain.ID
	}

	// Create metadata for session type
	metadata := []models.TransactionMetadata{
		{Key: "session_type", Value: sessionType},
		{Key: "data", Value: data},
	}

	session := &models.TransactionSession{
		ID:                   sessionID,
		Metadata:             metadata,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: chainType,
		ChainID:              chainIDUint,
		ExpiresAt:            time.Now().Add(30 * time.Minute),
	}

	if err := d.DB.Create(session).Error; err != nil {
		return "", err
	}

	return session.ID, nil
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

func (d *Database) UpdateTransactionSessionStatus(sessionID string, status models.TransactionStatus, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return d.DB.Model(&models.TransactionSession{}).Where("id = ?", sessionID).Updates(updates).Error
}

// UniswapDeployment operations
func (d *Database) CreateUniswapDeployment(deployment *models.UniswapDeployment) error {
	return d.DB.Create(deployment).Error
}

func (d *Database) GetUniswapDeploymentByChain(chainType, chainID string) (*models.UniswapDeployment, error) {
	var deployment models.UniswapDeployment
	err := d.DB.
		Joins("JOIN chains ON chains.id = uniswap_deployments.chain_id").
		Where("chains.chain_type = ? AND chains.chain_id = ? AND uniswap_deployments.status = ?", chainType, chainID, models.TransactionStatusConfirmed).
		First(&deployment).Error
	if err != nil {
		return nil, err
	}

	// Manually load the Chain relationship
	var chain models.Chain
	err = d.DB.First(&chain, deployment.ChainID).Error
	if err != nil {
		return nil, err
	}
	deployment.Chain = chain

	return &deployment, nil
}

func (d *Database) GetUniswapDeploymentByID(id uint) (*models.UniswapDeployment, error) {
	var deployment models.UniswapDeployment
	err := d.DB.First(&deployment, id).Error
	if err != nil {
		return nil, err
	}

	// Manually load the Chain relationship
	var chain models.Chain
	err = d.DB.First(&chain, deployment.ChainID).Error
	if err != nil {
		return nil, err
	}
	deployment.Chain = chain

	return &deployment, nil
}

func (d *Database) UpdateUniswapDeploymentStatus(id uint, status models.TransactionStatus, addresses map[string]string, txHashes map[string]string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Update addresses if provided
	if factoryAddr, ok := addresses["factory"]; ok && factoryAddr != "" {
		updates["factory_address"] = factoryAddr
	}
	if routerAddr, ok := addresses["router"]; ok && routerAddr != "" {
		updates["router_address"] = routerAddr
	}
	if wethAddr, ok := addresses["weth"]; ok && wethAddr != "" {
		updates["weth_address"] = wethAddr
	}
	if deployerAddr, ok := addresses["deployer"]; ok && deployerAddr != "" {
		updates["deployer_address"] = deployerAddr
	}

	// Update transaction hashes if provided
	if factoryTx, ok := txHashes["factory"]; ok && factoryTx != "" {
		updates["factory_tx_hash"] = factoryTx
	}
	if routerTx, ok := txHashes["router"]; ok && routerTx != "" {
		updates["router_tx_hash"] = routerTx
	}
	if wethTx, ok := txHashes["weth"]; ok && wethTx != "" {
		updates["weth_tx_hash"] = wethTx
	}

	return d.DB.Model(&models.UniswapDeployment{}).Where("id = ?", id).Updates(updates).Error
}

func (d *Database) ListUniswapDeployments() ([]models.UniswapDeployment, error) {
	var deployments []models.UniswapDeployment
	err := d.DB.Find(&deployments).Error
	if err != nil {
		return nil, err
	}

	// Manually load Chain relationships
	for i := range deployments {
		var chain models.Chain
		err = d.DB.First(&chain, deployments[i].ChainID).Error
		if err != nil {
			return nil, err
		}
		deployments[i].Chain = chain
	}

	return deployments, nil
}

func (d *Database) DeleteUniswapDeployments(ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	result := d.DB.Where("id IN ?", ids).Delete(&models.UniswapDeployment{})
	return result.RowsAffected, result.Error
}

func (d *Database) ClearUniswapConfiguration(version string) error {
	// Delete all Uniswap settings for the specified version
	return d.DB.Where("version = ?", version).Delete(&models.UniswapSettings{}).Error
}

func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
