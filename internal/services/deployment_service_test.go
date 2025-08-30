package services

import (
	"testing"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeploymentService(t *testing.T) {
	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto migrate
	err = db.AutoMigrate(&models.Deployment{}, &models.Template{}, &models.Chain{})
	require.NoError(t, err)

	// Create service
	service := NewDeploymentService(db)

	// Create test data
	chain := &models.Chain{
		ChainType: models.TransactionChainTypeEthereum,
		RPC:       "http://localhost:8545",
		NetworkID: "1",
		Name:      "Test Chain",
		IsActive:  true,
	}
	err = db.Create(chain).Error
	require.NoError(t, err)

	template := &models.Template{
		Name:         "Test Template",
		ChainType:    models.TransactionChainTypeEthereum,
		ContractName: "TestContract",
		TemplateCode: "pragma solidity ^0.8.0;",
	}
	err = db.Create(template).Error
	require.NoError(t, err)

	t.Run("CreateDeploymentWithUser", func(t *testing.T) {
		userID := "user123"
		deployment := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         chain.ID,
			ContractAddress: "0x123",
			Status:          string(models.TransactionStatusPending),
		}

		err := service.CreateDeploymentWithUser(deployment, &userID)
		assert.NoError(t, err)
		assert.NotNil(t, deployment.UserID)
		assert.Equal(t, userID, *deployment.UserID)
	})

	t.Run("ListDeploymentsByUser", func(t *testing.T) {
		// Create deployments for different users
		user1 := "user1"
		user2 := "user2"

		deployment1 := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         chain.ID,
			ContractAddress: "0x456",
			Status:          string(models.TransactionStatusConfirmed),
			UserID:          &user1,
		}
		err := db.Create(deployment1).Error
		require.NoError(t, err)

		deployment2 := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         chain.ID,
			ContractAddress: "0x789",
			Status:          string(models.TransactionStatusConfirmed),
			UserID:          &user2,
		}
		err = db.Create(deployment2).Error
		require.NoError(t, err)

		deployment3 := &models.Deployment{
			TemplateID:      template.ID,
			ChainID:         chain.ID,
			ContractAddress: "0xabc",
			Status:          string(models.TransactionStatusConfirmed),
			UserID:          &user1,
		}
		err = db.Create(deployment3).Error
		require.NoError(t, err)

		// Test filtering by user1
		deployments, err := service.ListDeploymentsByUser(user1)
		assert.NoError(t, err)
		assert.Len(t, deployments, 2)
		for _, d := range deployments {
			assert.NotNil(t, d.UserID)
			assert.Equal(t, user1, *d.UserID)
		}

		// Test filtering by user2
		deployments, err = service.ListDeploymentsByUser(user2)
		assert.NoError(t, err)
		assert.Len(t, deployments, 1)
		assert.NotNil(t, deployments[0].UserID)
		assert.Equal(t, user2, *deployments[0].UserID)
	})

	t.Run("ListDeployments", func(t *testing.T) {
		// Should return all deployments regardless of user
		deployments, err := service.ListDeployments()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(deployments), 3) // At least the ones we created
	})
}
