package services

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

// ChainService handles chain-related operations
type ChainService interface {
	CreateChain(chain *models.Chain) error
	GetActiveChain() (*models.Chain, error)
	SetActiveChain(chainType string) error
	SetActiveChainByID(chainID uint) error
	UpdateChainConfig(chainType, rpc, chainID string) error
	ListChains() ([]models.Chain, error)
}

type chainService struct {
	db *gorm.DB
}

// NewChainService creates a new ChainService
func NewChainService(db *gorm.DB) ChainService {
	return &chainService{db: db}
}

// CreateChain creates a new chain
func (s *chainService) CreateChain(chain *models.Chain) error {
	return s.db.Create(chain).Error
}

// GetActiveChain returns the currently active chain
func (s *chainService) GetActiveChain() (*models.Chain, error) {
	var chain models.Chain
	err := s.db.Where("is_active = ?", true).First(&chain).Error
	if err != nil {
		return nil, err
	}
	return &chain, nil
}

// SetActiveChain sets a chain as active by chain type
func (s *chainService) SetActiveChain(chainType string) error {
	// Deactivate all chains
	if err := s.db.Model(&models.Chain{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Activate the selected chain
	return s.db.Model(&models.Chain{}).Where("chain_type = ?", chainType).Update("is_active", true).Error
}

// SetActiveChainByID sets a chain as active by chain ID
func (s *chainService) SetActiveChainByID(chainID uint) error {
	// Deactivate all chains
	if err := s.db.Model(&models.Chain{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	// Activate the selected chain by ID
	return s.db.Model(&models.Chain{}).Where("id = ?", chainID).Update("is_active", true).Error
}

// UpdateChainConfig updates chain configuration
func (s *chainService) UpdateChainConfig(chainType, rpc, chainID string) error {
	return s.db.Model(&models.Chain{}).
		Where("chain_type = ?", chainType).
		Updates(map[string]interface{}{
			"rpc":      rpc,
			"chain_id": chainID,
		}).Error
}

// ListChains returns all chains
func (s *chainService) ListChains() ([]models.Chain, error) {
	var chains []models.Chain
	err := s.db.Find(&chains).Error
	return chains, err
}
