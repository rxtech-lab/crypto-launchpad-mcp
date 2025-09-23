package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rxtech-lab/launchpad-mcp/internal/api"
	"github.com/rxtech-lab/launchpad-mcp/internal/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/services"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Constants for testing
const (
	TESTNET_RPC      = "http://localhost:8545"
	TESTNET_CHAIN_ID = "31337"
)

// TestSetup provides the core test infrastructure for auth tests
type TestSetup struct {
	t                 *testing.T
	DBService         services.DBService
	ChainService      services.ChainService
	TemplateService   services.TemplateService
	DeploymentService services.DeploymentService
	UniswapService    services.UniswapService
	EthClient         *ethclient.Client
	TxService         services.TransactionService
	ServerPort        int
}

// NewTestSetup creates the base test infrastructure
func NewTestSetup(t *testing.T) *TestSetup {
	// Create in-memory database service
	dbService, err := services.NewSqliteDBService(":memory:")
	require.NoError(t, err)

	// Create other services
	chainService := services.NewChainService(dbService.GetDB())
	templateService := services.NewTemplateService(dbService.GetDB())
	deploymentService := services.NewDeploymentService(dbService.GetDB())
	uniswapService := services.NewUniswapService(dbService.GetDB())

	// Connect to Ethereum testnet
	ethClient, err := ethclient.Dial(TESTNET_RPC)
	require.NoError(t, err)

	// Create transaction service
	txService := services.NewTransactionService(dbService.GetDB())

	// Get a random port for test server
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	return &TestSetup{
		t:                 t,
		DBService:         dbService,
		ChainService:      chainService,
		TemplateService:   templateService,
		DeploymentService: deploymentService,
		UniswapService:    uniswapService,
		EthClient:         ethClient,
		TxService:         txService,
		ServerPort:        port,
	}
}

// VerifyEthereumConnection verifies connection to Ethereum testnet
func (s *TestSetup) VerifyEthereumConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.EthClient.NetworkID(ctx)
	return err
}

// Cleanup properly shuts down the TestSetup
func (s *TestSetup) Cleanup() {
	if s.EthClient != nil {
		s.EthClient.Close()
	}
	if s.DBService != nil {
		s.DBService.Close()
	}
}

// AuthTestSuite tests JWT and OAuth authentication using existing server endpoints
type AuthTestSuite struct {
	suite.Suite
	setup      *TestSetup
	apiServer  *api.APIServer
	baseURL    string
	authHelper *AuthTestHelper
	sessionID  string
}

func (s *AuthTestSuite) SetupSuite() {
	s.setup = NewTestSetup(s.T())
	// Setup authentication helper
	s.authHelper = NewAuthTestHelper(s.T())

	// Start mock OAuth server and setup environment
	s.authHelper.SetupMockOAuthServer()
	s.authHelper.SetupEnvironmentVariables()

	// Create and start API server with authentication enabled
	s.createAPIServerWithAuth()

	// Wait for servers to be ready
	time.Sleep(200 * time.Millisecond)
}

func (s *AuthTestSuite) TearDownSuite() {
	if s.apiServer != nil {
		s.apiServer.Shutdown()
	}

	if s.authHelper != nil {
		s.authHelper.Cleanup()
	}

	s.setup.Cleanup()
}

func (s *AuthTestSuite) createAPIServerWithAuth() {
	hookService := services.NewHookService()
	s.apiServer = api.NewAPIServer(s.setup.DBService, s.setup.TxService, hookService, s.setup.ChainService)

	// Create additional services needed for MCP server
	evmService := services.NewEvmService()
	liquidityService := services.NewLiquidityService(s.setup.DBService.GetDB())

	// Create and set MCP server for streamable HTTP
	mcpServer := mcp.NewMCPServer(
		s.setup.DBService,
		s.setup.ServerPort,
		evmService,
		s.setup.TxService,
		s.setup.UniswapService,
		liquidityService,
		s.setup.ChainService,
		s.setup.TemplateService,
		s.setup.DeploymentService,
	)
	s.apiServer.SetMCPServer(mcpServer)

	// Enable authentication - this is key for testing auth middleware
	s.apiServer.EnableAuthentication()

	// Enable MCP streamable HTTP to get routes that require authentication
	s.apiServer.EnableStreamableHttp()

	// Setup routes
	s.apiServer.SetupRoutes()

	// Start server
	port, err := s.apiServer.Start(nil)
	s.Require().NoError(err)

	s.baseURL = fmt.Sprintf("http://localhost:%d", port)
}

func (s *AuthTestSuite) makeRequest(method, path, authHeader string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, s.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}

// Test JWT Authentication on MCP API (requires auth)

func (s *AuthTestSuite) TestJWTAuth_MCPEndpoint_MissingToken() {
	resp, err := s.makeRequest("GET", "/authentication", "", nil)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should return 401 Unauthorized
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthTestSuite) TestJWTAuth_MCPEndpoint_ExpiredToken() {
	// Create expired token
	token := s.authHelper.CreateExpiredJWTToken()

	resp, err := s.makeRequest("GET", "/authentication", "Bearer "+token, nil)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should return 401 Unauthorized
	s.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthTestSuite) TestJWTAuth_TransactionAPI_ValidToken() {
	token := s.authHelper.CreateStandardUserToken("test-user-123")

	resp, err := s.makeRequest("GET", "/authentication", "Bearer "+token, nil)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// get the body
	body, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	s.T().Log(string(body))
	s.Equal(http.StatusOK, resp.StatusCode)

}

func (s *AuthTestSuite) TestJWTAuth_ShouldBeAbleToGetStaticAssets() {
	// /static/tx/app.js
	//static/tx/app.css

	resp, err := s.makeRequest("GET", "/static/tx/app.js", "", nil)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	resp, err = s.makeRequest("GET", "/static/tx/app.css", "", nil)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusOK, resp.StatusCode)

	resp, err = s.makeRequest("POST", "/api/tx/123/transaction/0", "", nil)
	s.Require().NoError(err)
	// We should be able to access the api/tx endpoint without authentication
	s.NotEqual(http.StatusUnauthorized, resp.StatusCode)
	defer resp.Body.Close()
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}
