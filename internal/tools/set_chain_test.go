package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rxtech-lab/launchpad-mcp/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDatabase(t *testing.T) interface{} {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := database.NewDatabase(dbPath)
	require.NoError(t, err)
	return db
}

func TestFetchChainIDFromRPC(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		statusCode   int
		expectedID   string
		expectError  bool
	}{
		{
			name:         "successful_hex_chain_id",
			responseBody: `{"jsonrpc":"2.0","id":1,"result":"0x1"}`,
			statusCode:   200,
			expectedID:   "1",
			expectError:  false,
		},
		{
			name:         "successful_hex_chain_id_sepolia",
			responseBody: `{"jsonrpc":"2.0","id":1,"result":"0xaa36a7"}`,
			statusCode:   200,
			expectedID:   "11155111",
			expectError:  false,
		},
		{
			name:         "successful_decimal_chain_id",
			responseBody: `{"jsonrpc":"2.0","id":1,"result":"1"}`,
			statusCode:   200,
			expectedID:   "1",
			expectError:  false,
		},
		{
			name:         "rpc_error_response",
			responseBody: `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`,
			statusCode:   200,
			expectedID:   "",
			expectError:  true,
		},
		{
			name:         "http_error_response",
			responseBody: `Internal Server Error`,
			statusCode:   500,
			expectedID:   "",
			expectError:  true,
		},
		{
			name:         "invalid_json_response",
			responseBody: `{invalid json}`,
			statusCode:   200,
			expectedID:   "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			chainID, err := fetchChainIDFromRPC(server.URL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, chainID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, chainID)
			}
		})
	}
}

func TestNewSetChainTool(t *testing.T) {
	db := setupTestDatabase(t)
	tool, handler := NewSetChainTool(db)

	// Test tool metadata
	assert.Equal(t, "set_chain", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.NotNil(t, handler)

	// Check that the tool has the expected properties
	assert.Contains(t, tool.InputSchema.Properties, "chain_type")
	assert.Contains(t, tool.InputSchema.Properties, "rpc")
}

func TestSetChainHandler(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name          string
		requestArgs   map[string]interface{}
		expectError   bool
		expectedChain *models.Chain
		setupMockRPC  bool
		mockChainID   string
	}{
		{
			name: "create_ethereum_chain_with_explicit_chain_id",
			requestArgs: map[string]interface{}{
				"chain_type": "ethereum",
				"rpc":        "https://eth-mainnet.alchemyapi.io/v2/test",
				"chain_id":   "1",
				"name":       "Ethereum Mainnet",
			},
			expectError: false,
			expectedChain: &models.Chain{
				ChainType: "ethereum",
				RPC:       "https://eth-mainnet.alchemyapi.io/v2/test",
				NetworkID: "1",
				Name:      "Ethereum Mainnet",
				IsActive:  false,
			},
		},
		{
			name: "create_ethereum_chain_auto_detect_chain_id",
			requestArgs: map[string]interface{}{
				"chain_type": "ethereum",
				"rpc":        "http://localhost:8545",
			},
			expectError:  false,
			setupMockRPC: true,
			mockChainID:  "1337",
			expectedChain: &models.Chain{
				ChainType: "ethereum",
				RPC:       "http://localhost:8545",
				NetworkID: "1337",
				Name:      "Ethereum Chain 1337",
				IsActive:  false,
			},
		},
		{
			name: "create_solana_chain",
			requestArgs: map[string]interface{}{
				"chain_type": "solana",
				"rpc":        "https://api.devnet.solana.com",
				"chain_id":   "devnet",
			},
			expectError: false,
			expectedChain: &models.Chain{
				ChainType: "solana",
				RPC:       "https://api.devnet.solana.com",
				NetworkID: "devnet",
				Name:      "Solana Devnet",
				IsActive:  false,
			},
		},
		{
			name: "missing_chain_type",
			requestArgs: map[string]interface{}{
				"rpc":      "https://eth-mainnet.alchemyapi.io/v2/test",
				"chain_id": "1",
			},
			expectError: true,
		},
		{
			name: "missing_rpc",
			requestArgs: map[string]interface{}{
				"chain_type": "ethereum",
				"chain_id":   "1",
			},
			expectError: true,
		},
		{
			name: "invalid_chain_type",
			requestArgs: map[string]interface{}{
				"chain_type": "invalid",
				"rpc":        "https://test.com",
				"chain_id":   "1",
			},
			expectError: false, // This returns success with error content
		},
		{
			name: "solana_missing_chain_id",
			requestArgs: map[string]interface{}{
				"chain_type": "solana",
				"rpc":        "https://api.devnet.solana.com",
			},
			expectError: false, // This returns success with error content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh database for each test
			db := setupTestDatabase(t)
			_, handler := NewSetChainTool(db)

			var mockServer *httptest.Server
			if tt.setupMockRPC {
				mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"result":"0x%x"}`, mustParseInt(tt.mockChainID))
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(response))
				}))
				defer mockServer.Close()
				// Update RPC URL to use mock server and expected chain in test data
				tt.requestArgs["rpc"] = mockServer.URL
				tt.expectedChain.RPC = mockServer.URL
			}

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.requestArgs,
				},
			}

			result, err := handler(ctx, request)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// For invalid inputs, check error content
			if tt.name == "invalid_chain_type" || tt.name == "solana_missing_chain_id" {
				assert.True(t, result.IsError)
				assert.Len(t, result.Content, 1)
				textContent0 := result.Content[0].(mcp.TextContent)
				assert.NotEmpty(t, textContent0.Text)
				return
			}

			// For successful cases, verify database record
			if tt.expectedChain != nil {
				chains, err := db.ListChains()
				assert.NoError(t, err)
				assert.Len(t, chains, 1)

				chain := chains[0]
				assert.Equal(t, tt.expectedChain.ChainType, chain.ChainType)
				assert.Equal(t, tt.expectedChain.RPC, chain.RPC)
				assert.Equal(t, tt.expectedChain.NetworkID, chain.NetworkID)
				assert.Equal(t, tt.expectedChain.Name, chain.Name)
				assert.Equal(t, tt.expectedChain.IsActive, chain.IsActive)

				// Verify result content contains success message with JSON
				assert.Len(t, result.Content, 2)
				textContent0 := result.Content[0].(mcp.TextContent)
				textContent1 := result.Content[1].(mcp.TextContent)
				assert.Equal(t, "Success message: ", textContent0.Text)

				var resultData map[string]interface{}
				err = json.Unmarshal([]byte(textContent1.Text), &resultData)
				assert.NoError(t, err)
				assert.Equal(t, string(tt.expectedChain.ChainType), resultData["chain_type"])
				assert.Equal(t, tt.expectedChain.RPC, resultData["rpc"])
				assert.Equal(t, tt.expectedChain.NetworkID, resultData["chain_id"])
				assert.Equal(t, tt.expectedChain.Name, resultData["name"])
			}
		})
	}
}

func TestSetChainUpdateExisting(t *testing.T) {
	db := setupTestDatabase(t)
	_, handler := NewSetChainTool(db)
	ctx := context.Background()

	// Create initial chain
	initialChain := &models.Chain{
		ChainType: "ethereum",
		RPC:       "https://old-rpc.com",
		NetworkID: "1",
		Name:      "Old Ethereum",
		IsActive:  true,
	}
	err := db.CreateChain(initialChain)
	require.NoError(t, err)

	// Update chain configuration
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]interface{}{
				"chain_type": "ethereum",
				"rpc":        "https://new-rpc.com",
				"chain_id":   "11155111",
				"name":       "Ethereum Sepolia",
			},
		},
	}

	result, err := handler(ctx, request)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify only one chain exists and it's updated
	chains, err := db.ListChains()
	assert.NoError(t, err)
	assert.Len(t, chains, 1)

	chain := chains[0]
	assert.Equal(t, models.TransactionChainType("ethereum"), chain.ChainType)
	assert.Equal(t, "https://new-rpc.com", chain.RPC)
	assert.Equal(t, "11155111", chain.NetworkID)
	// Note: Name is not updated by UpdateChainConfig, only RPC and ChainID
}

func TestDefaultChainNames(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		chainType    string
		chainID      string
		expectedName string
	}{
		{"ethereum", "1", "Ethereum Mainnet"},
		{"ethereum", "11155111", "Ethereum Sepolia"},
		{"ethereum", "5", "Ethereum Goerli"},
		{"ethereum", "999", "Ethereum Chain 999"},
		{"solana", "mainnet-beta", "Solana Mainnet"},
		{"solana", "devnet", "Solana Devnet"},
		{"solana", "testnet", "Solana Testnet"},
		{"solana", "custom", "Solana custom"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.chainType, tt.chainID), func(t *testing.T) {
			// Create fresh database for each test
			db := setupTestDatabase(t)
			_, handler := NewSetChainTool(db)

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"chain_type": tt.chainType,
						"rpc":        "https://test.com",
						"chain_id":   tt.chainID,
					},
				},
			}

			result, err := handler(ctx, request)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			chains, err := db.ListChains()
			assert.NoError(t, err)
			assert.Len(t, chains, 1)
			assert.Equal(t, tt.expectedName, chains[0].Name)
		})
	}
}

// Helper function to parse int for mock server
func mustParseInt(s string) int64 {
	switch s {
	case "1337":
		return 1337
	case "1":
		return 1
	default:
		return 1
	}
}
