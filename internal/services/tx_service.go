package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

type TransactionService struct {
	db *gorm.DB
}

type CreateTransactionSessionRequest struct {
	Metadata               []models.TransactionMetadata    `json:"metadata"`
	TransactionDeployments []models.TransactionDeployment `json:"transaction_deployments"`
	ChainType              string                          `json:"chain_type"`
	ChainID                uint                            `json:"chain_id"`
}

func NewTransactionService(db *gorm.DB) *TransactionService {
	return &TransactionService{db: db}
}

func (s *TransactionService) CreateTransactionSession(req CreateTransactionSessionRequest) (string, error) {
	sessionID := uuid.New().String()
	
	session := &models.TransactionSession{
		ID:                     sessionID,
		Metadata:               req.Metadata,
		TransactionStatus:      models.TransactionStatusPending,
		TransactionChainType:   models.TransactionChainType(req.ChainType),
		TransactionDeployments: req.TransactionDeployments,
		ChainID:                req.ChainID,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
		ExpiresAt:              time.Now().Add(30 * time.Minute),
	}

	err := s.db.Create(session).Error
	if err != nil {
		return "", err
	}

	return sessionID, nil
}
