package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/gorm"
)

type TransactionService interface {
	CreateTransactionSession(req CreateTransactionSessionRequest) (string, error)
	GetTransactionSession(sessionID string) (*models.TransactionSession, error)
	UpdateTransactionSession(sessionID string, session *models.TransactionSession) error
}

type transactionService struct {
	db *gorm.DB
}

type CreateTransactionSessionRequest struct {
	Metadata               []models.TransactionMetadata   `json:"metadata"`
	TransactionDeployments []models.TransactionDeployment `json:"transaction_deployments"`
	ChainType              models.TransactionChainType    `json:"chain_type"`
	ChainID                uint                           `json:"chain_id"`
}

func NewTransactionService(db *gorm.DB) TransactionService {
	return &transactionService{db: db}
}

func (s *transactionService) CreateTransactionSession(req CreateTransactionSessionRequest) (string, error) {
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

	// Load the Chain association after creation
	err = s.db.Preload("Chain").First(session, "id = ?", sessionID).Error
	if err != nil {
		return "", err
	}

	return sessionID, nil
}

// GetTransactionSession returns the transaction session by sessionID
func (t *transactionService) GetTransactionSession(sessionID string) (*models.TransactionSession, error) {
	var session models.TransactionSession
	err := t.db.Where("id = ?", sessionID).Preload("Chain").First(&session).Error
	if err != nil {
		return nil, err
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

// UpdateTransactionSession updates the transaction session by sessionID
func (t *transactionService) UpdateTransactionSession(sessionID string, session *models.TransactionSession) error {
	// Ensure the session ID matches
	session.ID = sessionID
	session.UpdatedAt = time.Now()

	// Update the session in the database by ID
	err := t.db.Model(&models.TransactionSession{}).Where("id = ?", sessionID).Updates(session).Error
	if err != nil {
		return err
	}

	return nil
}
