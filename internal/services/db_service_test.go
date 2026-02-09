package services_test

import (
	"testing"

	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
)

type DBServiceTestSuite struct {
	suite.Suite
}

func (suite *DBServiceTestSuite) TestNewTursoDBServiceInvalidURL() {
	// Test that NewTursoDBService returns an error with an invalid URL
	_, err := services.NewTursoDBService("invalid-url", "fake-token")
	suite.Error(err)
}

func (suite *DBServiceTestSuite) TestNewTursoDBServiceEmptyURL() {
	// Test that NewTursoDBService returns an error with an empty URL
	_, err := services.NewTursoDBService("", "")
	suite.Error(err)
}

func (suite *DBServiceTestSuite) TestNewTursoDBServiceUnreachableHost() {
	// Test that NewTursoDBService returns an error when the host is unreachable
	_, err := services.NewTursoDBService("libsql://nonexistent-db.turso.io", "fake-token")
	suite.Error(err)
}

func (suite *DBServiceTestSuite) TestNewSqliteDBServiceInMemory() {
	// Verify the existing SQLite in-memory service works (baseline test)
	db, err := services.NewSqliteDBService(":memory:")
	suite.Require().NoError(err)
	suite.NotNil(db)
	suite.NotNil(db.GetDB())
	defer db.Close()
}

func TestDBServiceTestSuite(t *testing.T) {
	suite.Run(t, new(DBServiceTestSuite))
}
