package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	// Use in-memory SQLite database for testing
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err, "Failed to connect to in-memory database")

	// Only migrate required models for TransactionSession
	err = db.AutoMigrate(
		&models.Chain{},
		&models.TransactionSession{},
	)
	require.NoError(t, err, "Failed to run migrations")

	// Enable debug mode to see SQL queries during test
	if testing.Verbose() {
		db = db.Debug()
	}

	return db
}

func TestGetTransactionSession(t *testing.T) {
	db := setupTestDB(t)
	service := &transactionService{db: db}

	t.Run("successful retrieval with chain preload", func(t *testing.T) {
		// Create a chain first
		chain := &models.Chain{
			ChainType: models.TransactionChainTypeEthereum,
			RPC:       "https://localhost:8545",
			NetworkID: "1",
			Name:      "Ethereum Mainnet",
			IsActive:  true,
		}
		err := db.Create(chain).Error
		require.NoError(t, err)

		// Create a transaction session
		sessionID := uuid.New().String()
		session := &models.TransactionSession{
			ID: sessionID,
			Metadata: []models.TransactionMetadata{
				{Key: "test_key", Value: "test_value"},
			},
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			TransactionDeployments: []models.TransactionDeployment{
				{
					Title:       "Test Deployment",
					Description: "Test Description",
					Data:        "0x1234",
					Value:       "0",
					Receiver:    "0x0000000000000000000000000000000000000000",
				},
			},
			ChainID:   chain.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		err = db.Create(session).Error
		require.NoError(t, err)

		// Retrieve the session
		retrievedSession, err := service.GetTransactionSession(sessionID)
		require.NoError(t, err)
		require.NotNil(t, retrievedSession)

		// Verify the session data
		assert.Equal(t, sessionID, retrievedSession.ID)
		assert.Equal(t, models.TransactionStatusPending, retrievedSession.TransactionStatus)
		assert.Equal(t, models.TransactionChainTypeEthereum, retrievedSession.TransactionChainType)
		assert.Equal(t, chain.ID, retrievedSession.ChainID)
		assert.Len(t, retrievedSession.Metadata, 1)
		assert.Equal(t, "test_key", retrievedSession.Metadata[0].Key)
		assert.Equal(t, "test_value", retrievedSession.Metadata[0].Value)
		assert.Len(t, retrievedSession.TransactionDeployments, 1)

		// Verify chain is preloaded
		// The Preload should load the Chain based on ChainID foreign key
		assert.Equal(t, chain.ID, retrievedSession.Chain.ID)
		assert.Equal(t, chain.Name, retrievedSession.Chain.Name)
		assert.Equal(t, chain.RPC, retrievedSession.Chain.RPC)
		assert.Equal(t, chain.NetworkID, retrievedSession.Chain.NetworkID)
	})

	t.Run("session not found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		retrievedSession, err := service.GetTransactionSession(nonExistentID)

		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("expired session", func(t *testing.T) {
		// Create an expired session
		sessionID := uuid.New().String()
		session := &models.TransactionSession{
			ID:                   sessionID,
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			ChainID:              1,
			CreatedAt:            time.Now().Add(-2 * time.Hour),
			UpdatedAt:            time.Now().Add(-2 * time.Hour),
			ExpiresAt:            time.Now().Add(-1 * time.Hour), // Expired 1 hour ago
		}
		err := db.Create(session).Error
		require.NoError(t, err)

		// Try to retrieve the expired session
		retrievedSession, err := service.GetTransactionSession(sessionID)

		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
		assert.Contains(t, err.Error(), "session expired")
	})

	t.Run("session with empty chain (chain not found)", func(t *testing.T) {
		// Create a session with non-existent chain ID
		sessionID := uuid.New().String()
		session := &models.TransactionSession{
			ID:                   sessionID,
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeSolana,
			ChainID:              999, // Non-existent chain ID
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
			ExpiresAt:            time.Now().Add(30 * time.Minute),
		}
		err := db.Create(session).Error
		require.NoError(t, err)

		// Retrieve the session
		retrievedSession, err := service.GetTransactionSession(sessionID)
		require.NoError(t, err)
		require.NotNil(t, retrievedSession)

		// Verify chain is empty when chain doesn't exist
		assert.Equal(t, sessionID, retrievedSession.ID)
		assert.Equal(t, uint(999), retrievedSession.ChainID)

		// Check if Chain is empty (zero value)
		assert.Equal(t, models.Chain{}, retrievedSession.Chain)
		assert.Equal(t, uint(0), retrievedSession.Chain.ID)
		assert.Empty(t, retrievedSession.Chain.Name)
		assert.Empty(t, retrievedSession.Chain.RPC)
		assert.Empty(t, retrievedSession.Chain.NetworkID)
		assert.Empty(t, retrievedSession.Chain.ChainType)
		assert.False(t, retrievedSession.Chain.IsActive)
	})

	t.Run("session with complex metadata and deployments", func(t *testing.T) {
		// Create a chain
		chain := &models.Chain{
			ChainType: models.TransactionChainTypeSolana,
			RPC:       "https://api.devnet.solana.com",
			NetworkID: "devnet",
			Name:      "Solana Devnet",
			IsActive:  false,
		}
		err := db.Create(chain).Error
		require.NoError(t, err)

		// Create a session with multiple metadata and deployments
		sessionID := uuid.New().String()
		session := &models.TransactionSession{
			ID: sessionID,
			Metadata: []models.TransactionMetadata{
				{Key: "type", Value: "token_deployment"},
				{Key: "version", Value: "1.0.0"},
				{Key: "environment", Value: "test"},
			},
			TransactionStatus:    models.TransactionStatusConfirmed,
			TransactionChainType: models.TransactionChainTypeSolana,
			TransactionDeployments: []models.TransactionDeployment{
				{
					Title:       "Deploy Token",
					Description: "Deploy ERC20 Token",
					Data:        "0xabcdef",
					Value:       "1000000000",
					Receiver:    "0x1234567890123456789012345678901234567890",
				},
				{
					Title:       "Initialize Pool",
					Description: "Initialize Liquidity Pool",
					Data:        "0xfedcba",
					Value:       "2000000000",
					Receiver:    "0x9876543210987654321098765432109876543210",
				},
			},
			ChainID:   chain.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		err = db.Create(session).Error
		require.NoError(t, err)

		// Retrieve the session
		retrievedSession, err := service.GetTransactionSession(sessionID)
		require.NoError(t, err)
		require.NotNil(t, retrievedSession)

		// Verify all metadata
		assert.Len(t, retrievedSession.Metadata, 3)
		assert.Equal(t, models.TransactionStatusConfirmed, retrievedSession.TransactionStatus)
		assert.Equal(t, models.TransactionChainTypeSolana, retrievedSession.TransactionChainType)

		// Verify all deployments
		assert.Len(t, retrievedSession.TransactionDeployments, 2)
		assert.Equal(t, "Deploy Token", retrievedSession.TransactionDeployments[0].Title)
		assert.Equal(t, "Initialize Pool", retrievedSession.TransactionDeployments[1].Title)
		assert.Equal(t, "0xabcdef", retrievedSession.TransactionDeployments[0].Data)
		assert.Equal(t, "0xfedcba", retrievedSession.TransactionDeployments[1].Data)

		// Verify chain is properly loaded
		assert.Equal(t, chain.ID, retrievedSession.Chain.ID)
		assert.Equal(t, chain.Name, retrievedSession.Chain.Name)
		assert.Equal(t, models.TransactionChainTypeSolana, retrievedSession.Chain.ChainType)
		assert.False(t, retrievedSession.Chain.IsActive)
	})

	t.Run("empty session ID", func(t *testing.T) {
		retrievedSession, err := service.GetTransactionSession("")

		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("session at exact expiry time", func(t *testing.T) {
		// Create a session that expires exactly now
		sessionID := uuid.New().String()
		expiryTime := time.Now()
		session := &models.TransactionSession{
			ID:                   sessionID,
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			ChainID:              1,
			CreatedAt:            time.Now().Add(-30 * time.Minute),
			UpdatedAt:            time.Now().Add(-30 * time.Minute),
			ExpiresAt:            expiryTime,
		}
		err := db.Create(session).Error
		require.NoError(t, err)

		// Small delay to ensure we're past expiry
		time.Sleep(10 * time.Millisecond)

		// Try to retrieve the session
		retrievedSession, err := service.GetTransactionSession(sessionID)

		assert.Error(t, err)
		assert.Nil(t, retrievedSession)
		assert.Contains(t, err.Error(), "session expired")
	})
}

func TestUpdateTransactionSession(t *testing.T) {
	db := setupTestDB(t)
	service := &transactionService{db: db}

	t.Run("successful update", func(t *testing.T) {
		// Create a chain first
		chain := &models.Chain{
			ChainType: models.TransactionChainTypeEthereum,
			RPC:       "https://localhost:8545",
			NetworkID: "1",
			Name:      "Ethereum Mainnet",
			IsActive:  true,
		}
		err := db.Create(chain).Error
		require.NoError(t, err)

		// Create initial transaction session
		sessionID := uuid.New().String()
		originalTime := time.Now().Add(-1 * time.Hour)
		session := &models.TransactionSession{
			ID: sessionID,
			Metadata: []models.TransactionMetadata{
				{Key: "old_key", Value: "old_value"},
			},
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			TransactionDeployments: []models.TransactionDeployment{
				{
					Title:       "Original Deployment",
					Description: "Original Description",
					Data:        "0x1234",
					Value:       "0",
					Receiver:    "0x0000000000000000000000000000000000000000",
					Status:      models.TransactionStatusPending,
				},
			},
			ChainID:   chain.ID,
			CreatedAt: originalTime,
			UpdatedAt: originalTime,
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		err = db.Create(session).Error
		require.NoError(t, err)

		// Update the session
		updatedSession := &models.TransactionSession{
			ID: sessionID,
			Metadata: []models.TransactionMetadata{
				{Key: "new_key", Value: "new_value"},
				{Key: "another_key", Value: "another_value"},
			},
			TransactionStatus:    models.TransactionStatusConfirmed,
			TransactionChainType: models.TransactionChainTypeEthereum,
			TransactionDeployments: []models.TransactionDeployment{
				{
					Title:       "Updated Deployment",
					Description: "Updated Description",
					Data:        "0xabcd",
					Value:       "1000",
					Receiver:    "0x1234567890123456789012345678901234567890",
					Status:      models.TransactionStatusConfirmed,
				},
			},
			ChainID: chain.ID,
		}

		beforeUpdate := time.Now()
		err = service.UpdateTransactionSession(sessionID, updatedSession)
		require.NoError(t, err)

		// Retrieve and verify the updated session
		retrieved, err := service.GetTransactionSession(sessionID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		// Verify updates
		assert.Equal(t, sessionID, retrieved.ID)
		assert.Equal(t, models.TransactionStatusConfirmed, retrieved.TransactionStatus)
		assert.Len(t, retrieved.Metadata, 2)
		assert.Equal(t, "new_key", retrieved.Metadata[0].Key)
		assert.Equal(t, "new_value", retrieved.Metadata[0].Value)
		assert.Equal(t, "another_key", retrieved.Metadata[1].Key)
		assert.Equal(t, "another_value", retrieved.Metadata[1].Value)

		// Verify deployments were updated
		assert.Len(t, retrieved.TransactionDeployments, 1)
		assert.Equal(t, "Updated Deployment", retrieved.TransactionDeployments[0].Title)
		assert.Equal(t, "Updated Description", retrieved.TransactionDeployments[0].Description)
		assert.Equal(t, "0xabcd", retrieved.TransactionDeployments[0].Data)
		assert.Equal(t, "1000", retrieved.TransactionDeployments[0].Value)
		assert.Equal(t, models.TransactionStatusConfirmed, retrieved.TransactionDeployments[0].Status)

		// Verify UpdatedAt was updated
		assert.True(t, retrieved.UpdatedAt.After(beforeUpdate) || retrieved.UpdatedAt.Equal(beforeUpdate))
		assert.True(t, retrieved.UpdatedAt.After(originalTime))

		// Verify CreatedAt and ExpiresAt were not changed
		assert.Equal(t, originalTime.Unix(), retrieved.CreatedAt.Unix())
	})

	t.Run("update non-existent session", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		session := &models.TransactionSession{
			ID:                   nonExistentID,
			TransactionStatus:    models.TransactionStatusConfirmed,
			TransactionChainType: models.TransactionChainTypeEthereum,
			ChainID:              1,
		}

		err := service.UpdateTransactionSession(nonExistentID, session)
		// GORM Updates doesn't return error for non-existent records
		// It just doesn't update anything
		require.NoError(t, err)

		// Verify the session doesn't exist
		retrieved, err := service.GetTransactionSession(nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Equal(t, gorm.ErrRecordNotFound, err)
	})

	t.Run("update with partial data", func(t *testing.T) {
		// Create initial session
		sessionID := uuid.New().String()
		originalSession := &models.TransactionSession{
			ID: sessionID,
			Metadata: []models.TransactionMetadata{
				{Key: "key1", Value: "value1"},
			},
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			TransactionDeployments: []models.TransactionDeployment{
				{
					Title:       "Original",
					Description: "Original Desc",
					Data:        "0x1111",
					Value:       "0",
					Receiver:    "0xaaaa",
					Status:      models.TransactionStatusPending,
				},
			},
			ChainID:   1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		err := db.Create(originalSession).Error
		require.NoError(t, err)

		// Update only status
		partialUpdate := &models.TransactionSession{
			TransactionStatus: models.TransactionStatusFailed,
		}

		err = service.UpdateTransactionSession(sessionID, partialUpdate)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := service.GetTransactionSession(sessionID)
		require.NoError(t, err)

		// Status should be updated
		assert.Equal(t, models.TransactionStatusFailed, retrieved.TransactionStatus)

		// Other fields should remain unchanged
		assert.Len(t, retrieved.Metadata, 1)
		assert.Equal(t, "key1", retrieved.Metadata[0].Key)
		assert.Len(t, retrieved.TransactionDeployments, 1)
		assert.Equal(t, "Original", retrieved.TransactionDeployments[0].Title)
	})

	t.Run("update deployment status", func(t *testing.T) {
		// Create a chain
		chain := &models.Chain{
			ChainType: models.TransactionChainTypeEthereum,
			RPC:       "https://localhost:8545",
			NetworkID: "1",
			Name:      "Ethereum Mainnet",
			IsActive:  true,
		}
		err := db.Create(chain).Error
		require.NoError(t, err)

		// Create session with multiple deployments
		sessionID := uuid.New().String()
		session := &models.TransactionSession{
			ID:                   sessionID,
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			TransactionDeployments: []models.TransactionDeployment{
				{
					Title:       "Deploy Token",
					Description: "Deploy ERC20",
					Data:        "0xaaa",
					Value:       "0",
					Receiver:    "0x111",
					Status:      models.TransactionStatusPending,
				},
				{
					Title:       "Initialize Pool",
					Description: "Init LP",
					Data:        "0xbbb",
					Value:       "100",
					Receiver:    "0x222",
					Status:      models.TransactionStatusPending,
				},
			},
			ChainID:   chain.ID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		err = db.Create(session).Error
		require.NoError(t, err)

		// Update to mark first deployment as confirmed
		updatedSession := &models.TransactionSession{
			TransactionStatus:    models.TransactionStatusPending,
			TransactionChainType: models.TransactionChainTypeEthereum,
			TransactionDeployments: []models.TransactionDeployment{
				{
					Title:       "Deploy Token",
					Description: "Deploy ERC20",
					Data:        "0xaaa",
					Value:       "0",
					Receiver:    "0x111",
					Status:      models.TransactionStatusConfirmed, // Changed
				},
				{
					Title:       "Initialize Pool",
					Description: "Init LP",
					Data:        "0xbbb",
					Value:       "100",
					Receiver:    "0x222",
					Status:      models.TransactionStatusPending, // Unchanged
				},
			},
			ChainID: chain.ID,
		}

		err = service.UpdateTransactionSession(sessionID, updatedSession)
		require.NoError(t, err)

		// Retrieve and verify
		retrieved, err := service.GetTransactionSession(sessionID)
		require.NoError(t, err)

		assert.Len(t, retrieved.TransactionDeployments, 2)
		assert.Equal(t, models.TransactionStatusConfirmed, retrieved.TransactionDeployments[0].Status)
		assert.Equal(t, models.TransactionStatusPending, retrieved.TransactionDeployments[1].Status)
	})
}
