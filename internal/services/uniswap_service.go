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
	CreateUniswapDeployment(chainID uint, version string) (uint, error)
	UpdateFactoryAddress(deploymentID uint, factoryAddress string) error
	UpdateRouterAddress(deploymentID uint, routerAddress string) error
	UpdateWETHAddress(deploymentID uint, wethAddress string) error
	UpdateDeployerAddress(deploymentID uint, deployerAddress string) error
	UpdateStatus(deploymentID uint, status models.TransactionStatus) error
	ListUniswapDeployments(skip, limit int) ([]models.UniswapDeployment, error)
	DeleteUniswapDeployment(deploymentID uint) error
	DeleteUniswapDeployments(deploymentIDs []uint) error
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
		if deployment.DeployerAddress == "" {
			missingAddresses = append(missingAddresses, "deployer_address")
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

func (u *uniswapService) CreateUniswapDeployment(chainID uint, version string) (uint, error) {
	deployment := &models.UniswapDeployment{
		ChainID: chainID,
		Version: version,
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

func (u *uniswapService) DeleteUniswapDeployment(deploymentID uint) error {
	return u.db.Delete(&models.UniswapDeployment{}, deploymentID).Error
}

func (u *uniswapService) DeleteUniswapDeployments(deploymentIDs []uint) error {
	return u.db.Delete(&models.UniswapDeployment{}, deploymentIDs).Error
}
