package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBService handles database connection and lifecycle management
type DBService interface {
	GetDB() *gorm.DB
	Close() error
}

type dbService struct {
	db *gorm.DB
}

// NewDBService creates a new DBService with SQLite connection
func NewDBService(dbPath string) (DBService, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Configure GORM logger - only log errors and slow queries
	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Only log errors and slow queries
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,        // Include params in SQL log
			Colorful:                  false,        // Disable color
		},
	)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	service := &dbService{db: db}
	if err := service.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return service, nil
}

// GetDB returns the underlying GORM database instance
func (s *dbService) GetDB() *gorm.DB {
	return s.db
}

// migrate runs database migrations
func (s *dbService) migrate() error {
	return s.db.AutoMigrate(
		&models.Chain{},
		&models.Template{},
		&models.Deployment{},
		&models.UniswapSettings{},
		&models.UniswapDeployment{},
		&models.LiquidityPool{},
		&models.LiquidityPosition{},
		&models.SwapTransaction{},
		&models.TransactionSession{},
	)
}

// Close closes the database connection
func (s *dbService) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}