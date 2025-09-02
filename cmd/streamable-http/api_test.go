package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
	suite.T().Setenv("SCALEKIT_ENV_URL", "https://env.scalekit.com")

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

func (suite *StreamableHTTPTestSuite) TestAllRoutesRequireAuthentication() {
	// Test that all regular routes require authentication
	client := &http.Client{Timeout: 10 * time.Second}

	// Define test routes with their expected status codes when unauthorized
	testRoutes := []struct {
		method         string
		path           string
		expectedStatus int
		description    string
	}{
		{"GET", "/static/tx/app.js", http.StatusUnauthorized, "static JavaScript"},
		{"GET", "/static/tx/app.css", http.StatusUnauthorized, "static CSS"},
		{"POST", "/api/test/sign-transaction", http.StatusUnauthorized, "test endpoint"},
		{"GET", "/health", http.StatusUnauthorized, "health check"},
	}

	for _, testRoute := range testRoutes {
		url := suite.getBaseURL() + testRoute.path

		var req *http.Request
		var err error

		if testRoute.method == "POST" {
			// For POST requests, send empty JSON body
			req, err = http.NewRequest(testRoute.method, url, strings.NewReader("{}"))
			if err == nil {
				req.Header.Set("Content-Type", "application/json")
			}
		} else {
			req, err = http.NewRequest(testRoute.method, url, nil)
		}

		suite.Require().NoError(err, "Failed to create request for %s", testRoute.description)

		// Intentionally NOT setting Authorization header
		resp, err := client.Do(req)
		suite.Require().NoError(err, "Failed to make request for %s", testRoute.description)

		// Should return 401 Unauthorized for all routes
		suite.Equal(testRoute.expectedStatus, resp.StatusCode,
			"Route %s %s (%s) should require authentication",
			testRoute.method, testRoute.path, testRoute.description)

		_ = resp.Body.Close()
	}
}

func (suite *StreamableHTTPTestSuite) TestWellKnownEndpointBypassesAuth() {
	// Test that .well-known endpoint is accessible without authentication
	client := &http.Client{Timeout: 10 * time.Second}

	url := suite.getBaseURL() + "/.well-known/oauth-protected-resource/mcp"
	req, err := http.NewRequest("GET", url, nil)
	suite.Require().NoError(err)

	// Intentionally NOT setting Authorization header
	resp, err := client.Do(req)
	suite.Require().NoError(err)

	// Should return 200 OK (or other success code, not 401)
	suite.NotEqual(http.StatusUnauthorized, resp.StatusCode,
		".well-known endpoint should bypass authentication")

	_ = resp.Body.Close()
}

func (suite *StreamableHTTPTestSuite) TestAuthenticatedRequestsWork() {
	// Test that valid tokens allow access to protected routes
	// Note: This test sets up a valid authenticator environment to test positive cases
	client := &http.Client{Timeout: 10 * time.Second}

	// Set up environment for JWT authentication
	suite.T().Setenv("SCALEKIT_ENV_URL", "https://example.com/.well-known/jwks.json")

	// Test a simple route that should work with any valid-looking token
	url := suite.getBaseURL() + "/health"
	req, err := http.NewRequest("GET", url, nil)
	suite.Require().NoError(err)

	// Set a mock valid token (the test environment will accept any non-empty token)
	req.Header.Set("Authorization", "Bearer valid-test-token")

	resp, err := client.Do(req)
	suite.Require().NoError(err)

	// With a valid token, should not return 401
	// Note: The actual response may vary (200, 404, etc.) based on the endpoint logic
	suite.NotEqual(http.StatusUnauthorized, resp.StatusCode,
		"Valid token should allow access to protected routes")

	_ = resp.Body.Close()
}

func TestStreamableHTTPTestSuite(t *testing.T) {
	suite.Run(t, new(StreamableHTTPTestSuite))
}
