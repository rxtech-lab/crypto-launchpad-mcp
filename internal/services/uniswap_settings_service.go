package services

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

// UniswapSettingsService handles Uniswap settings operations
type UniswapSettingsService interface {
	SetUniswapVersion(version string) error
	SetUniswapConfiguration(version, routerAddress, factoryAddress, wethAddress, quoterAddress, positionManager, swapRouter02 string) error
	GetActiveUniswapSettings() (*models.UniswapSettings, error)
}

type uniswapSettingsService struct {
	db *gorm.DB
}

// NewUniswapSettingsService creates a new UniswapSettingsService
func NewUniswapSettingsService(db *gorm.DB) UniswapSettingsService {
	return &uniswapSettingsService{db: db}
}

// SetUniswapVersion sets the active Uniswap version
func (s *uniswapSettingsService) SetUniswapVersion(version string) error {
	// Deactivate all versions
	if err := s.db.Model(&models.UniswapSettings{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Create or activate the selected version
	var setting models.UniswapSettings
	err := s.db.Where("version = ?", version).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		// Create new setting
		setting = models.UniswapSettings{
			Version:  version,
			IsActive: true,
		}
		return s.db.Create(&setting).Error
	} else if err != nil {
		return err
	}

	// Activate existing setting
	return s.db.Model(&setting).Update("is_active", true).Error
}

// SetUniswapConfiguration sets the active Uniswap configuration with all addresses
func (s *uniswapSettingsService) SetUniswapConfiguration(version, routerAddress, factoryAddress, wethAddress, quoterAddress, positionManager, swapRouter02 string) error {
	// Deactivate all versions
	if err := s.db.Model(&models.UniswapSettings{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Create or update the selected version with addresses
	var setting models.UniswapSettings
	err := s.db.Where("version = ?", version).First(&setting).Error
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
		return s.db.Create(&setting).Error
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
	return s.db.Model(&setting).Updates(updates).Error
}

// GetActiveUniswapSettings returns the currently active Uniswap settings
func (s *uniswapSettingsService) GetActiveUniswapSettings() (*models.UniswapSettings, error) {
	var settings models.UniswapSettings
	err := s.db.Where("is_active = ?", true).First(&settings).Error
	if err != nil {
		return nil, err
	}
	return &settings, nil
}