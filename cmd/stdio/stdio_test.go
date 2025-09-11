package main

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type StdioServerTestSuite struct {
	suite.Suite
	db        *gorm.DB
	apiServer *api.APIServer
	port      int
}

func (suite *StdioServerTestSuite) SetupSuite() {
	suite.T().Setenv("JWT_SECRET", "test-secret")
	// Create in-memory database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)
	suite.db = db

	// Create database service wrapper
	dbService := services.NewDBServiceFromDB(db)

	// Configure and start server using the refactored function
	apiServer, port, err := configureAndStartServer(dbService, 0) // 0 for random port
	suite.Require().NoError(err)
	suite.Require().NotZero(port, "Port should not be 0")

	suite.apiServer = apiServer
	suite.port = port

	// Wait for server to be ready
	time.Sleep(100 * time.Millisecond)
}

func (suite *StdioServerTestSuite) TearDownSuite() {
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

func (suite *StdioServerTestSuite) TestRoutesAccessibleWithoutAuth() {
	// Test that all routes are accessible without authentication in stdio mode
	client := &http.Client{Timeout: 10 * time.Second}

	// Define test routes that should be accessible without authentication
	testRoutes := []struct {
		method      string
		path        string
		description string
	}{
		{"GET", "/tx/test-session-id", "transaction page"},
		{"POST", "/api/tx/test-session-id/transaction/0", "transaction API"},
		{"GET", "/static/tx/app.js", "static JavaScript"},
		{"GET", "/static/tx/app.css", "static CSS"},
		{"POST", "/api/test/sign-transaction", "test endpoint"},
		{"GET", "/health", "health check"},
		{"GET", "/.well-known/oauth-protected-resource/mcp", "well-known endpoint"},
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

		// Should NOT return 401 Unauthorized for any route in stdio mode
		suite.NotEqual(http.StatusUnauthorized, resp.StatusCode,
			"Route %s %s (%s) should be accessible without authentication in stdio mode",
			testRoute.method, testRoute.path, testRoute.description)

		_ = resp.Body.Close()
	}
}

func (suite *StdioServerTestSuite) TestNoMCPEndpointsInStdioMode() {
	// Test that MCP endpoints are not registered in stdio mode
	client := &http.Client{Timeout: 10 * time.Second}

	// These endpoints should not exist in stdio mode
	mcpPaths := []string{
		"/mcp",
		"/mcp/sse",
		"/mcp/ws",
		"/mcp/status",
	}

	for _, path := range mcpPaths {
		url := suite.getBaseURL() + path
		req, err := http.NewRequest("GET", url, nil)
		suite.Require().NoError(err)

		resp, err := client.Do(req)
		suite.Require().NoError(err)

		// Should return 404 Not Found because these endpoints don't exist in stdio mode
		suite.Equal(http.StatusNotFound, resp.StatusCode,
			"MCP endpoint %s should not exist in stdio mode", path)

		_ = resp.Body.Close()
	}
}

func (suite *StdioServerTestSuite) TestHealthEndpointWorks() {
	// Test that health endpoint specifically returns expected response
	client := &http.Client{Timeout: 10 * time.Second}

	url := suite.getBaseURL() + "/health"
	req, err := http.NewRequest("GET", url, nil)
	suite.Require().NoError(err)

	resp, err := client.Do(req)
	suite.Require().NoError(err)

	// Health endpoint should return 200 OK
	suite.Equal(http.StatusOK, resp.StatusCode, "Health endpoint should return 200 OK")

	_ = resp.Body.Close()
}

func (suite *StdioServerTestSuite) TestStaticAssetsAccessible() {
	// Test that static assets are accessible without authentication
	client := &http.Client{Timeout: 10 * time.Second}

	staticPaths := []struct {
		path        string
		contentType string
	}{
		{"/static/tx/app.js", "application/javascript"},
		{"/static/tx/app.css", "text/css"},
	}

	for _, asset := range staticPaths {
		url := suite.getBaseURL() + asset.path
		req, err := http.NewRequest("GET", url, nil)
		suite.Require().NoError(err)

		resp, err := client.Do(req)
		suite.Require().NoError(err)

		// Should return 200 OK for static assets
		suite.Equal(http.StatusOK, resp.StatusCode,
			"Static asset %s should be accessible", asset.path)

		// Check content type
		contentType := resp.Header.Get("Content-Type")
		suite.Equal(asset.contentType, contentType,
			"Static asset %s should have correct content type", asset.path)

		_ = resp.Body.Close()
	}
}

func (suite *StdioServerTestSuite) TestAuthenticationMiddlewareNotActive() {
	// Test that providing Authorization header doesn't change behavior
	// (authentication middleware is not active in stdio mode)
	client := &http.Client{Timeout: 10 * time.Second}

	url := suite.getBaseURL() + "/health"

	// Test without auth header
	req1, err := http.NewRequest("GET", url, nil)
	suite.Require().NoError(err)

	resp1, err := client.Do(req1)
	suite.Require().NoError(err)
	status1 := resp1.StatusCode
	_ = resp1.Body.Close()

	// Test with auth header
	req2, err := http.NewRequest("GET", url, nil)
	suite.Require().NoError(err)
	req2.Header.Set("Authorization", "Bearer some-token")

	resp2, err := client.Do(req2)
	suite.Require().NoError(err)
	status2 := resp2.StatusCode
	_ = resp2.Body.Close()

	// Both requests should return the same status code
	// (authentication middleware is not processing the token)
	suite.Equal(status1, status2,
		"Response should be identical with and without auth header in stdio mode")
}

func (suite *StdioServerTestSuite) getBaseURL() string {
	return fmt.Sprintf("http://localhost:%d", suite.port)
}

func TestStdioServerTestSuite(t *testing.T) {
	suite.Run(t, new(StdioServerTestSuite))
}
