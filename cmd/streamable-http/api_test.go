package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type StreamableHTTPTestSuite struct {
	suite.Suite
	db        *gorm.DB
	apiServer *api.APIServer
	port      int
}

func (suite *StreamableHTTPTestSuite) SetupSuite() {
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// Configure and start server using the refactored function
	apiServer, port, err := configureAndStartServer(db, 0) // 0 for random port
	suite.Require().NoError(err)
	suite.Require().NotZero(port, "Port should not be 0")

	suite.apiServer = apiServer
	suite.port = port

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)
}

func (suite *StreamableHTTPTestSuite) TearDownSuite() {
	if suite.apiServer != nil {
		suite.apiServer.Shutdown()
	}
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}
}

func (suite *StreamableHTTPTestSuite) TestMCPEndpointRequiresAuthentication() {
	// Test that /mcp endpoint returns error without authentication token
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a basic MCP request (initialize request)
	mcpRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	requestBody, err := json.Marshal(mcpRequest)
	suite.Require().NoError(err)

	// Make request to /mcp without Authorization header
	url := suite.getBaseURL() + "/mcp"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	suite.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")
	// Intentionally NOT setting Authorization header

	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Should return 401 Unauthorized when no token is present
	// This matches the behavior expected from server.go:109-110
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *StreamableHTTPTestSuite) TestMCPEndpointWithInvalidToken() {
	// Test that /mcp endpoint returns error with invalid authentication token
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a basic MCP request
	mcpRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	requestBody, err := json.Marshal(mcpRequest)
	suite.Require().NoError(err)

	// Make request to /mcp with invalid Authorization header
	url := suite.getBaseURL() + "/mcp"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	suite.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Should return 401 Unauthorized when invalid token is present
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *StreamableHTTPTestSuite) TestMCPEndpointWithEmptyBearerToken() {
	// Test that /mcp endpoint returns error with empty bearer token
	client := &http.Client{Timeout: 10 * time.Second}

	// Create a basic MCP request
	mcpRequest := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	requestBody, err := json.Marshal(mcpRequest)
	suite.Require().NoError(err)

	// Make request to /mcp with empty Bearer token
	url := suite.getBaseURL() + "/mcp"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	suite.Require().NoError(err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ")

	resp, err := client.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	// Should return 401 Unauthorized when empty token is present
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *StreamableHTTPTestSuite) TestMCPSubpathAuthentication() {
	// Test that /mcp/* endpoints also require authentication
	client := &http.Client{Timeout: 10 * time.Second}

	// Test various MCP subpaths without authentication
	testPaths := []string{
		"/mcp/sse",
		"/mcp/ws",
		"/mcp/status",
	}

	for _, path := range testPaths {
		url := suite.getBaseURL() + path
		req, err := http.NewRequest("GET", url, nil)
		suite.Require().NoError(err)

		// Intentionally NOT setting Authorization header
		resp, err := client.Do(req)
		suite.Require().NoError(err)
		resp.Body.Close()

		// Should return 401 Unauthorized for all MCP subpaths
		suite.Equal(http.StatusUnauthorized, resp.StatusCode, "Path %s should require authentication", path)
	}
}

func (suite *StreamableHTTPTestSuite) getBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", suite.port)
}

func TestStreamableHTTPTestSuite(t *testing.T) {
	suite.Run(t, new(StreamableHTTPTestSuite))
}
