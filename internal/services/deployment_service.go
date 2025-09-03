package services

import (
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

type DeploymentService interface {
	CreateDeployment(deployment *models.Deployment) error
	CreateDeploymentWithUser(deployment *models.Deployment, userID *string) error
	GetDeploymentByID(id uint) (*models.Deployment, error)
	ListDeployments() ([]models.Deployment, error)
	ListDeploymentsByUser(userID string) ([]models.Deployment, error)
	UpdateDeploymentStatus(id uint, status models.TransactionStatus, contractAddress string) error
	UpdateDeploymentStatusWithTxHashBySessionId(sessionId string, status models.TransactionStatus, contractAddress, txHash string) error
	DeleteDeployment(id uint) error
	GetDeploymentByContractAddress(contractAddress string) (*models.Deployment, error)
	GetDeploymentsByTemplate(templateID uint) ([]models.Deployment, error)
	GetDeploymentsByChain(chainID uint) ([]models.Deployment, error)
	GetDeploymentByTransactionHash(txHash string) (*models.Deployment, error)
}

// DeploymentService handles deployment-related operations
type deploymentService struct {
	db *gorm.DB
}

// NewDeploymentService creates a new DeploymentService
func NewDeploymentService(db *gorm.DB) DeploymentService {
	return &deploymentService{db: db}
}

// CreateDeployment creates a new deployment
func (s *deploymentService) CreateDeployment(deployment *models.Deployment) error {
	return s.db.Create(deployment).Error
}

// CreateDeploymentWithUser creates a new deployment with an optional user ID
func (s *deploymentService) CreateDeploymentWithUser(deployment *models.Deployment, userID *string) error {
	deployment.UserID = userID
	return s.db.Create(deployment).Error
}

// GetDeploymentByID returns a deployment by its ID
func (s *deploymentService) GetDeploymentByID(id uint) (*models.Deployment, error) {
	var deployment models.Deployment
	err := s.db.Preload("Template").Preload("Chain").First(&deployment, id).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// ListDeployments returns all deployments
func (s *deploymentService) ListDeployments() ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Find(&deployments).Error
	return deployments, err
}

// ListDeploymentsByUser returns all deployments for a specific user
func (s *deploymentService) ListDeploymentsByUser(userID string) ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("user_id = ?", userID).Find(&deployments).Error
	return deployments, err
}

// UpdateDeploymentStatus updates the status of a deployment
func (s *deploymentService) UpdateDeploymentStatus(id uint, status models.TransactionStatus, contractAddress string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if contractAddress != "" {
		updates["contract_address"] = contractAddress
	}

	return s.db.Model(&models.Deployment{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateDeploymentStatusWithTxHashBySessionId updates the status of a deployment with transaction hash by session ID
func (s *deploymentService) UpdateDeploymentStatusWithTxHashBySessionId(sessionId string, status models.TransactionStatus, contractAddress, txHash string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if contractAddress != "" {
		updates["contract_address"] = contractAddress
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return s.db.Model(&models.Deployment{}).Where("session_id = ?", sessionId).Updates(updates).Error
}

// DeleteDeployment deletes a deployment by its ID
func (s *deploymentService) DeleteDeployment(id uint) error {
	return s.db.Delete(&models.Deployment{}, id).Error
}

// GetDeploymentByContractAddress returns a deployment by its contract address
func (s *deploymentService) GetDeploymentByContractAddress(contractAddress string) (*models.Deployment, error) {
	var deployment models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("contract_address = ?", contractAddress).First(&deployment).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}

// GetDeploymentsByTemplate returns all deployments for a specific template
func (s *deploymentService) GetDeploymentsByTemplate(templateID uint) ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("template_id = ?", templateID).Find(&deployments).Error
	return deployments, err
}

// GetDeploymentsByChain returns all deployments for a specific chain
func (s *deploymentService) GetDeploymentsByChain(chainID uint) ([]models.Deployment, error) {
	var deployments []models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("chain_id = ?", chainID).Find(&deployments).Error
	return deployments, err
}

// GetDeploymentByTransactionHash returns a deployment by its transaction hash
func (s *deploymentService) GetDeploymentByTransactionHash(txHash string) (*models.Deployment, error) {
	var deployment models.Deployment
	err := s.db.Preload("Template").Preload("Chain").Where("transaction_hash = ?", txHash).First(&deployment).Error
	if err != nil {
		return nil, err
	}
	return &deployment, nil
}
