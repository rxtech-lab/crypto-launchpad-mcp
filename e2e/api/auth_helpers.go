package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/require"
)

// AuthTestHelper provides authentication testing utilities
type AuthTestHelper struct {
	t              *testing.T
	jwtSecret      string
	mockJWKSServer *http.Server
	jwksPort       int
}

// NewAuthTestHelper creates a new authentication test helper
func NewAuthTestHelper(t *testing.T) *AuthTestHelper {
	return &AuthTestHelper{
		t:         t,
		jwtSecret: "test-jwt-secret-for-auth-testing",
	}
}

// SetupMockOAuthServer starts a mock OAuth/JWKS server for testing
func (h *AuthTestHelper) SetupMockOAuthServer() {
	mux := http.NewServeMux()

	// Mock JWKS endpoint
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, r *http.Request) {
		jwks := map[string]interface{}{
			"keys": []map[string]interface{}{
				{
					"kty": "RSA",
					"kid": "test-key-id",
					"use": "sig",
					"alg": "RS256",
					"n":   "mock-modulus-for-oauth-testing",
					"e":   "AQAB",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	})

	// Mock OAuth resource metadata endpoint
	mux.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		metadata := map[string]interface{}{
			"resource": "test-resource",
			"audience": "test-audience",
			"issuer":   "test-issuer",
			"scopes":   []string{"read", "write", "admin"},
			"jwks_uri": fmt.Sprintf("http://localhost:%d/.well-known/jwks.json", h.jwksPort),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(metadata)
	})

	// Mock token validation endpoint (optional)
	mux.HandleFunc("/validate", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Simple mock validation
		token := authHeader[7:] // Remove "Bearer "
		if token == "valid_oauth_token" {
			response := map[string]interface{}{
				"valid":  true,
				"sub":    "oauth-test-user",
				"roles":  []string{"user"},
				"scopes": []string{"read", "write"},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid_token"})
		}
	})

	// Start server on random port
	listener, err := net.Listen("tcp", ":0")
	require.NoError(h.t, err)
	h.jwksPort = listener.Addr().(*net.TCPAddr).Port

	h.mockJWKSServer = &http.Server{Handler: mux}
	go func() {
		h.mockJWKSServer.Serve(listener)
	}()

	// Wait for server to be ready
	time.Sleep(50 * time.Millisecond)
}

// SetupEnvironmentVariables configures OAuth and JWT environment variables using t.Setenv
func (h *AuthTestHelper) SetupEnvironmentVariables() {
	h.t.Setenv("JWT_SECRET", h.jwtSecret)
	if h.jwksPort > 0 {
		h.t.Setenv("SCALEKIT_ENV_URL", fmt.Sprintf("http://localhost:%d/.well-known/jwks.json", h.jwksPort))
		h.t.Setenv("SCALEKIT_RESOURCE_METADATA_URL", fmt.Sprintf("http://localhost:%d/metadata", h.jwksPort))
		h.t.Setenv("OAUTH_AUTHENTICATION_SERVER", fmt.Sprintf("http://localhost:%d", h.jwksPort))
		h.t.Setenv("OAUTH_RESOURCE_URL", "http://localhost:3000/api")
		h.t.Setenv("OAUTH_RESOURCE_DOCUMENTATION_URL", "http://localhost:3000/docs")
	}
}

// Cleanup shuts down the mock server (environment variables cleaned up automatically by t.Setenv)
func (h *AuthTestHelper) Cleanup() {
	if h.mockJWKSServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		h.mockJWKSServer.Shutdown(ctx)
	}
	// Environment variables are automatically cleaned up by t.Setenv
}

// CreateValidJWTToken creates a valid JWT token with the given claims
func (h *AuthTestHelper) CreateValidJWTToken(claims map[string]interface{}) string {
	// Set default expiration if not provided
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	if _, ok := claims["iat"]; !ok {
		claims["iat"] = time.Now().Unix()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	require.NoError(h.t, err)
	return tokenString
}

// CreateExpiredJWTToken creates an expired JWT token
func (h *AuthTestHelper) CreateExpiredJWTToken() string {
	claims := map[string]interface{}{
		"sub":   "expired-user",
		"exp":   time.Now().Unix() - 3600, // Expired 1 hour ago
		"iat":   time.Now().Unix() - 7200, // Issued 2 hours ago
		"roles": []string{"user"},
	}
	return h.CreateValidJWTToken(claims)
}

// CreateMalformedJWTToken creates a malformed JWT token string
func (h *AuthTestHelper) CreateMalformedJWTToken() string {
	return "malformed.jwt.token.invalid"
}

// CreateJWTTokenWithMissingClaims creates a JWT token missing required claims
func (h *AuthTestHelper) CreateJWTTokenWithMissingClaims() string {
	claims := map[string]interface{}{
		// Missing 'sub' claim
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	return h.CreateValidJWTToken(claims)
}

// ValidateJWTAuthenticator tests the JWT authenticator directly
func (h *AuthTestHelper) ValidateJWTAuthenticator(token string) (*utils.AuthenticatedUser, error) {
	authenticator, err := utils.NewSimpleJwtAuthenticator(h.jwtSecret)
	require.NoError(h.t, err)
	return authenticator.ValidateToken(token)
}

// GetJWKSPort returns the port of the mock JWKS server
func (h *AuthTestHelper) GetJWKSPort() int {
	return h.jwksPort
}

// GetJWTSecret returns the JWT secret used for testing
func (h *AuthTestHelper) GetJWTSecret() string {
	return h.jwtSecret
}

// CreateStandardUserToken creates a JWT token for a standard user
func (h *AuthTestHelper) CreateStandardUserToken(userID string) string {
	claims := map[string]interface{}{
		"sub":       userID,
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"roles":     []string{"user"},
		"scopes":    []string{"read"},
		"client_id": "test-client",
		"iss":       "test-issuer",
	}
	return h.CreateValidJWTToken(claims)
}

// CreateAdminUserToken creates a JWT token for an admin user
func (h *AuthTestHelper) CreateAdminUserToken(userID string) string {
	claims := map[string]interface{}{
		"sub":       userID,
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"roles":     []string{"admin", "user"},
		"scopes":    []string{"read", "write", "admin"},
		"client_id": "admin-client",
		"iss":       "test-issuer",
		"aud":       []string{"test-audience"},
	}
	return h.CreateValidJWTToken(claims)
}

// CreateServiceAccountToken creates a JWT token for a service account
func (h *AuthTestHelper) CreateServiceAccountToken(serviceID string) string {
	claims := map[string]interface{}{
		"sub":          serviceID,
		"exp":          time.Now().Add(24 * time.Hour).Unix(), // Longer expiration for service accounts
		"iat":          time.Now().Unix(),
		"roles":        []string{"service"},
		"scopes":       []string{"read", "write"},
		"client_id":    "service-client",
		"iss":          "service-issuer",
		"account_type": "service",
	}
	return h.CreateValidJWTToken(claims)
}

// AuthTestScenario represents a test scenario for authentication
type AuthTestScenario struct {
	Name           string
	AuthHeader     string
	ExpectedStatus int
	Description    string
}

// GetCommonAuthScenarios returns common authentication test scenarios
func (h *AuthTestHelper) GetCommonAuthScenarios() []AuthTestScenario {
	return []AuthTestScenario{
		{
			Name:           "ValidJWTToken",
			AuthHeader:     "Bearer " + h.CreateStandardUserToken("test-user-123"),
			ExpectedStatus: http.StatusOK,
			Description:    "Valid JWT token should authenticate successfully",
		},
		{
			Name:           "ExpiredJWTToken",
			AuthHeader:     "Bearer " + h.CreateExpiredJWTToken(),
			ExpectedStatus: http.StatusUnauthorized,
			Description:    "Expired JWT token should be rejected",
		},
		{
			Name:           "MalformedJWTToken",
			AuthHeader:     "Bearer " + h.CreateMalformedJWTToken(),
			ExpectedStatus: http.StatusUnauthorized,
			Description:    "Malformed JWT token should be rejected",
		},
		{
			Name:           "MissingAuthHeader",
			AuthHeader:     "",
			ExpectedStatus: http.StatusUnauthorized,
			Description:    "Missing authorization header should be rejected",
		},
		{
			Name:           "InvalidAuthFormat",
			AuthHeader:     "Basic dGVzdDp0ZXN0", // Basic auth instead of Bearer
			ExpectedStatus: http.StatusUnauthorized,
			Description:    "Non-Bearer authorization should be rejected",
		},
		{
			Name:           "EmptyBearerToken",
			AuthHeader:     "Bearer ",
			ExpectedStatus: http.StatusUnauthorized,
			Description:    "Empty Bearer token should be rejected",
		},
	}
}
