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
	CreateTransactionSessionWithUser(req CreateTransactionSessionRequest, userID *string) (string, error)
	GetTransactionSession(sessionID string) (*models.TransactionSession, error)
	UpdateTransactionSession(sessionID string, session *models.TransactionSession) error
	ListTransactionSessionsByUser(userID string) ([]models.TransactionSession, error)

	// Legacy methods for backward compatibility with database.go
	CreateTransactionSessionLegacy(sessionType string, chainType models.TransactionChainType, chainID, data string) (string, error)
	CreateTransactionSessionWithUserLegacy(sessionType string, chainType models.TransactionChainType, chainID, data string, userID *string) (string, error)
	UpdateTransactionSessionStatus(sessionID string, status models.TransactionStatus, txHash string) error
}

type transactionService struct {
	db *gorm.DB
}

type CreateTransactionSessionRequest struct {
	Metadata               []models.TransactionMetadata   `json:"metadata"`
	TransactionDeployments []models.TransactionDeployment `json:"transaction_deployments"`
	ChainType              models.TransactionChainType    `json:"chain_type"`
	ChainID                uint                           `json:"chain_id"`
	UserID                 *string                        `json:"user_id,omitempty"`
}

func NewTransactionService(db *gorm.DB) TransactionService {
	return &transactionService{db: db}
}

func (s *transactionService) CreateTransactionSession(req CreateTransactionSessionRequest) (string, error) {
	return s.CreateTransactionSessionWithUser(req, nil)
}

func (s *transactionService) CreateTransactionSessionWithUser(req CreateTransactionSessionRequest, userID *string) (string, error) {
	sessionID := uuid.New().String()

	// Use provided userID or from request
	finalUserID := userID
	if finalUserID == nil && req.UserID != nil {
		finalUserID = req.UserID
	}

	session := &models.TransactionSession{
		ID:                     sessionID,
		UserID:                 finalUserID,
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
func (s *transactionService) GetTransactionSession(sessionID string) (*models.TransactionSession, error) {
	var session models.TransactionSession
	err := s.db.Where("id = ?", sessionID).Preload("Chain").First(&session).Error
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
func (s *transactionService) UpdateTransactionSession(sessionID string, session *models.TransactionSession) error {
	// Ensure the session ID matches
	session.ID = sessionID
	session.UpdatedAt = time.Now()

	// Update the session in the database by ID
	err := s.db.Model(&models.TransactionSession{}).Where("id = ?", sessionID).Updates(session).Error
	if err != nil {
		return err
	}

	return nil
}

// ListTransactionSessionsByUser returns all transaction sessions for a specific user
func (s *transactionService) ListTransactionSessionsByUser(userID string) ([]models.TransactionSession, error) {
	var sessions []models.TransactionSession
	err := s.db.Preload("Chain").Where("user_id = ?", userID).Find(&sessions).Error
	return sessions, err
}

// CreateTransactionSessionLegacy creates a transaction session with backward compatibility signature
func (s *transactionService) CreateTransactionSessionLegacy(sessionType string, chainType models.TransactionChainType, chainID, data string) (string, error) {
	return s.CreateTransactionSessionWithUserLegacy(sessionType, chainType, chainID, data, nil)
}

// CreateTransactionSessionWithUserLegacy creates a transaction session with optional user ID (legacy signature)
func (s *transactionService) CreateTransactionSessionWithUserLegacy(sessionType string, chainType models.TransactionChainType, chainID, data string, userID *string) (string, error) {
	// Generate a UUID for the session ID
	sessionID := fmt.Sprintf("%s-%d", sessionType, time.Now().UnixNano())

	// Parse chainID to uint
	var chainIDUint uint
	if _, err := fmt.Sscanf(chainID, "%d", &chainIDUint); err != nil {
		// If chainID is not a number, try to find the chain by type and chainID string
		var chain models.Chain
		if err := s.db.Where("chain_type = ? AND chain_id = ?", chainType, chainID).First(&chain).Error; err != nil {
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
		UserID:               userID,
		Metadata:             metadata,
		TransactionStatus:    models.TransactionStatusPending,
		TransactionChainType: chainType,
		ChainID:              chainIDUint,
		ExpiresAt:            time.Now().Add(30 * time.Minute),
	}

	if err := s.db.Create(session).Error; err != nil {
		return "", err
	}

	return session.ID, nil
}

// UpdateTransactionSessionStatus updates the status of a transaction session
func (s *transactionService) UpdateTransactionSessionStatus(sessionID string, status models.TransactionStatus, txHash string) error {
	updates := map[string]interface{}{
		"transaction_status": status,
	}
	if txHash != "" {
		updates["transaction_hash"] = txHash
	}

	return s.db.Model(&models.TransactionSession{}).Where("id = ?", sessionID).Updates(updates).Error
}
