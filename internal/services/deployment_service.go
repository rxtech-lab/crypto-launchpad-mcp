package services

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

// DeploymentService handles deployment-related operations
type DeploymentService struct {
	db *gorm.DB
}

// NewDeploymentService creates a new DeploymentService
func NewDeploymentService(db *gorm.DB) *DeploymentService {
	return &DeploymentService{db: db}
}

// CreateDeployment creates a new deployment
func (s *DeploymentService) CreateDeployment(deployment *models.Deployment) error {
	return s.db.Create(deployment).Error
}

// CreateDeploymentWithUser creates a new deployment with an optional user ID
func (s *DeploymentService) CreateDeploymentWithUser(deployment *models.Deployment, userID *string) error {
	deployment.UserID = userID
	return s.db.Create(deployment).Error
}

// GetDeploymentByID returns a deployment by its ID
func (s *DeploymentService) GetDeploymentByID(id uint) (*models.Deployment, error) {
	var deployment models.Deployment
	err := s.db.Preload("Template").Preload("Chain").First(&deployment, id).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// ListDeployments returns all deployments
func (s *DeploymentService) ListDeployments() ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Find(&deployments).Error
	return deployments, err
}

// ListDeploymentsByUser returns all deployments for a specific user
func (s *DeploymentService) ListDeploymentsByUser(userID string) ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("user_id = ?", userID).Find(&deployments).Error
	return deployments, err
}

// UpdateDeploymentStatus updates the status of a deployment
func (s *DeploymentService) UpdateDeploymentStatus(id uint, status models.TransactionStatus, contractAddress string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if contractAddress != "" {
		updates["contract_address"] = contractAddress
	}

	return s.db.Model(&models.Deployment{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateDeploymentStatusWithTxHash updates the status of a deployment with transaction hash
func (s *DeploymentService) UpdateDeploymentStatusWithTxHash(id uint, status models.TransactionStatus, contractAddress, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if contractAddress != "" {
		updates["contract_address"] = contractAddress
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return s.db.Model(&models.Deployment{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteDeployment deletes a deployment by its ID
func (s *DeploymentService) DeleteDeployment(id uint) error {
	return s.db.Delete(&models.Deployment{}, id).Error
}

// GetDeploymentByContractAddress returns a deployment by its contract address
func (s *DeploymentService) GetDeploymentByContractAddress(contractAddress string) (*models.Deployment, error) {
	var deployment models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("contract_address = ?", contractAddress).First(&deployment).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// GetDeploymentsByTemplate returns all deployments for a specific template
func (s *DeploymentService) GetDeploymentsByTemplate(templateID uint) ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("template_id = ?", templateID).Find(&deployments).Error
	return deployments, err
}

// GetDeploymentsByChain returns all deployments for a specific chain
func (s *DeploymentService) GetDeploymentsByChain(chainID uint) ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("chain_id = ?", chainID).Find(&deployments).Error
	return deployments, err
}

// GetDeploymentByTransactionHash returns a deployment by its transaction hash
func (s *DeploymentService) GetDeploymentByTransactionHash(txHash string) (*models.Deployment, error) {
	var deployment models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("transaction_hash = ?", txHash).First(&deployment).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}
