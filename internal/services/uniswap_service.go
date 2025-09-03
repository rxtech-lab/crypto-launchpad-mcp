package services

import (
	"errors"
	"fmt"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

type UniswapService interface {
	GetUniswapDeployment(deploymentID uint) (*models.UniswapDeployment, error)
	GetUniswapDeploymentByChain(chainID uint) (*models.UniswapDeployment, error)
	GetUniswapDeploymentByChainString(chainType, chainID string) (*models.UniswapDeployment, error)
	CreateUniswapDeployment(chainID uint, version string, userId *string) (uint, error)
	UpdateFactoryAddress(deploymentID uint, factoryAddress string) error
	UpdateRouterAddress(deploymentID uint, routerAddress string) error
	UpdateWETHAddress(deploymentID uint, wethAddress string) error
	UpdateDeployerAddress(deploymentID uint, deployerAddress string) error
	UpdateStatus(deploymentID uint, status models.TransactionStatus) error
	ListUniswapDeployments(skip, limit int) ([]models.UniswapDeployment, error)
	ListUniswapDeploymentsByUser(userID string, skip, limit int) ([]models.UniswapDeployment, error)
	DeleteUniswapDeployment(deploymentID uint) error
	DeleteUniswapDeployments(deploymentIDs []uint) error
	GetActiveUniswapDeployment(userId *string, chain models.Chain) (*models.UniswapDeployment, error)
}

type uniswapService struct {
	db *gorm.DB
}

// UpdateStatus implements UniswapService.
func (u *uniswapService) UpdateStatus(deploymentID uint, status models.TransactionStatus) error {
	// If status is confirmed, validate that all required addresses are present
	if status == models.TransactionStatusConfirmed {
		var deployment models.UniswapDeployment
		if err := u.db.First(&deployment, deploymentID).Error; err != nil {
			return err
		}

		// Check for missing addresses
		var missingAddresses []string
		if deployment.FactoryAddress == "" {
			missingAddresses = append(missingAddresses, "factory_address")
		}
		if deployment.RouterAddress == "" {
			missingAddresses = append(missingAddresses, "router_address")
		}
		if deployment.WETHAddress == "" {
			missingAddresses = append(missingAddresses, "weth_address")
		}

		if len(missingAddresses) > 0 {
			return errors.New("cannot confirm deployment with missing addresses: " +
				fmt.Sprintf("%v", missingAddresses))
		}
	}

	return u.db.Model(&models.UniswapDeployment{}).
		Where("id = ?", deploymentID).
		Update("status", status).Error
}

func NewUniswapService(db *gorm.DB) UniswapService {
	return &uniswapService{db: db}
}

func (u *uniswapService) CreateUniswapDeployment(chainID uint, version string, userId *string) (uint, error) {
	return u.CreateUniswapDeploymentWithUser(chainID, version, userId)
}

func (u *uniswapService) CreateUniswapDeploymentWithUser(chainID uint, version string, userID *string) (uint, error) {
	deployment := &models.UniswapDeployment{
		ChainID: chainID,
		Version: version,
		UserID:  userID,
		Status:  models.TransactionStatusPending,
	}
	err := u.db.Create(deployment).Error
	if err != nil {
		return 0, err
	}
	return deployment.ID, nil
}

func (u *uniswapService) GetUniswapDeployment(deploymentID uint) (*models.UniswapDeployment, error) {
	var deployment models.UniswapDeployment
	err := u.db.First(&deployment, deploymentID).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

func (u *uniswapService) UpdateFactoryAddress(deploymentID uint, factoryAddress string) error {
	return u.db.Model(&models.UniswapDeployment{}).
		Where("id = ?", deploymentID).
		Update("factory_address", factoryAddress).Error
}

func (u *uniswapService) UpdateRouterAddress(deploymentID uint, routerAddress string) error {
	return u.db.Model(&models.UniswapDeployment{}).
		Where("id = ?", deploymentID).
		Update("router_address", routerAddress).Error
}

func (u *uniswapService) UpdateWETHAddress(deploymentID uint, wethAddress string) error {
	return u.db.Model(&models.UniswapDeployment{}).
		Where("id = ?", deploymentID).
		Update("weth_address", wethAddress).Error
}

func (u *uniswapService) UpdateDeployerAddress(deploymentID uint, deployerAddress string) error {
	return u.db.Model(&models.UniswapDeployment{}).
		Where("id = ?", deploymentID).
		Update("deployer_address", deployerAddress).Error
}

func (u *uniswapService) GetUniswapDeploymentByChain(chainID uint) (*models.UniswapDeployment, error) {
	var deployment models.UniswapDeployment
	err := u.db.Where("chain_id = ?", chainID).First(&deployment).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

func (u *uniswapService) ListUniswapDeployments(skip, limit int) ([]models.UniswapDeployment, error) {
	var deployments []models.UniswapDeployment
	err := u.db.Offset(skip).Limit(limit).Find(&deployments).Error
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (u *uniswapService) ListUniswapDeploymentsByUser(userID string, skip, limit int) ([]models.UniswapDeployment, error) {
	var deployments []models.UniswapDeployment
	err := u.db.Where("user_id = ?", userID).Offset(skip).Limit(limit).Find(&deployments).Error
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (u *uniswapService) DeleteUniswapDeployment(deploymentID uint) error {
	return u.db.Delete(&models.UniswapDeployment{}, deploymentID).Error
}

func (u *uniswapService) DeleteUniswapDeployments(deploymentIDs []uint) error {
	return u.db.Delete(&models.UniswapDeployment{}, deploymentIDs).Error
}

// GetUniswapDeploymentByChainString gets a Uniswap deployment by chain type and chain ID strings
func (u *uniswapService) GetUniswapDeploymentByChainString(chainType, chainID string) (*models.UniswapDeployment, error) {
	var deployment models.UniswapDeployment
	var chain models.Chain

	// First find the chain
	err := u.db.Where("chain_type = ? AND chain_id = ?", chainType, chainID).First(&chain).Error
	if err != nil {
		return nil, err
	}

	// Then find deployment for this chain
	err = u.db.Where("chain_id = ?", chain.ID).Preload("Chain").First(&deployment).Error
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}

func (u *uniswapService) GetActiveUniswapDeployment(userId *string, chain models.Chain) (*models.UniswapDeployment, error) {
	var deployment models.UniswapDeployment
	query := u.db.Where("chain_id = ?", chain.ID)

	// Add user filter if userId is provided
	if userId != nil {
		query = query.Where("user_id = ?", *userId)
	}

	err := query.First(&deployment).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}
