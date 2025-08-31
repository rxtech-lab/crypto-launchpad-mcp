package utils

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func TestNewJwtAuthenticator(t *testing.T) {
	jwksUri := "https://example.com/.well-known/jwks.json"
	auth := NewJwtAuthenticator(jwksUri)

	if auth.JwksUri != jwksUri {
		t.Errorf("Expected JwksUri to be %s, got %s", jwksUri, auth.JwksUri)
	}

	if auth.cacheTTL.Minutes() != 5 {
		t.Errorf("Expected cacheTTL to be 5 minutes, got %v", auth.cacheTTL)
	}
}

func TestValidateTokenWithoutJwksUri(t *testing.T) {
	auth := NewJwtAuthenticator("")
	
	_, err := auth.ValidateToken("dummy.jwt.token")
	if err == nil {
		t.Error("Expected error when JWKS URI is not configured")
	}
	
	expectedError := "JWKS URI not configured"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

func TestMapClaimsToUser(t *testing.T) {
	auth := NewJwtAuthenticator("https://example.com/.well-known/jwks.json")
	
	// Test claims mapping
	claims := map[string]interface{}{
		"sub":       "user123",
		"iss":       "https://auth.example.com",
		"client_id": "client123",
		"exp":       1234567890.0,
		"iat":       1234567800.0,
		"aud":       []interface{}{"audience1", "audience2"},
		"roles":     []interface{}{"admin", "user"},
		"scopes":    []interface{}{"read", "write"},
	}
	
	user, err := auth.mapClaimsToUser(claims)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if user.Sub != "user123" {
		t.Errorf("Expected Sub to be 'user123', got '%s'", user.Sub)
	}
	
	if user.Iss != "https://auth.example.com" {
		t.Errorf("Expected Iss to be 'https://auth.example.com', got '%s'", user.Iss)
	}
	
	if user.ClientId != "client123" {
		t.Errorf("Expected ClientId to be 'client123', got '%s'", user.ClientId)
	}
	
	if user.Exp != 1234567890 {
		t.Errorf("Expected Exp to be 1234567890, got %d", user.Exp)
	}
	
	if len(user.Aud) != 2 || user.Aud[0] != "audience1" || user.Aud[1] != "audience2" {
		t.Errorf("Expected Aud to be ['audience1', 'audience2'], got %v", user.Aud)
	}
	
	if len(user.Roles) != 2 || user.Roles[0] != "admin" || user.Roles[1] != "user" {
		t.Errorf("Expected Roles to be ['admin', 'user'], got %v", user.Roles)
	}
	
	if len(user.Scopes) != 2 || user.Scopes[0] != "read" || user.Scopes[1] != "write" {
		t.Errorf("Expected Scopes to be ['read', 'write'], got %v", user.Scopes)
	}
}

func TestMapClaimsToUserWithSingleAudience(t *testing.T) {
	auth := NewJwtAuthenticator("https://example.com/.well-known/jwks.json")
	
	// Test single audience as string
	claims := map[string]interface{}{
		"aud": "single-audience",
	}
	
	user, err := auth.mapClaimsToUser(claims)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(user.Aud) != 1 || user.Aud[0] != "single-audience" {
		t.Errorf("Expected Aud to be ['single-audience'], got %v", user.Aud)
	}
}

func TestValidateTokenWithRealSignature(t *testing.T) {
	// Generate RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	
	publicKey := &privateKey.PublicKey
	
	// Create a JWK set with the public key
	keyID := "test-key-1"
	jwkKey, err := jwk.FromRaw(publicKey)
	if err != nil {
		t.Fatalf("Failed to create JWK from RSA public key: %v", err)
	}
	
	// Set key ID and algorithm
	err = jwkKey.Set(jwk.KeyIDKey, keyID)
	if err != nil {
		t.Fatalf("Failed to set key ID: %v", err)
	}
	
	err = jwkKey.Set(jwk.AlgorithmKey, "RS256")
	if err != nil {
		t.Fatalf("Failed to set algorithm: %v", err)
	}
	
	err = jwkKey.Set(jwk.KeyUsageKey, "sig")
	if err != nil {
		t.Fatalf("Failed to set key usage: %v", err)
	}
	
	// Create JWK set
	set := jwk.NewSet()
	set.AddKey(jwkKey)
	
	// Create mock JWKS endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		
		// Convert JWK set to JSON
		jwksJSON, err := json.Marshal(set)
		if err != nil {
			http.Error(w, "Failed to marshal JWKS", http.StatusInternalServerError)
			return
		}
		
		w.Write(jwksJSON)
	}))
	defer mockServer.Close()
	
	// Create JWT token with the private key
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       "user123",
		"iss":       "https://test-auth.example.com",
		"aud":       []string{"test-audience"},
		"exp":       now.Add(time.Hour).Unix(),
		"iat":       now.Unix(),
		"nbf":       now.Unix(),
		"jti":       "test-jwt-id",
		"client_id": "test-client",
		"roles":     []string{"admin", "user"},
		"scopes":    []string{"read", "write"},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign JWT token: %v", err)
	}
	
	t.Logf("Generated JWT token: %s", tokenString)
	t.Logf("Mock JWKS endpoint: %s", mockServer.URL)
	
	// Create authenticator with mock JWKS endpoint
	auth := NewJwtAuthenticator(mockServer.URL)
	
	// Test token validation
	user, err := auth.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	
	// Verify extracted claims
	if user.Sub != "user123" {
		t.Errorf("Expected Sub to be 'user123', got '%s'", user.Sub)
	}
	
	if user.Iss != "https://test-auth.example.com" {
		t.Errorf("Expected Iss to be 'https://test-auth.example.com', got '%s'", user.Iss)
	}
	
	if user.ClientId != "test-client" {
		t.Errorf("Expected ClientId to be 'test-client', got '%s'", user.ClientId)
	}
	
	if len(user.Aud) != 1 || user.Aud[0] != "test-audience" {
		t.Errorf("Expected Aud to be ['test-audience'], got %v", user.Aud)
	}
	
	if len(user.Roles) != 2 || user.Roles[0] != "admin" || user.Roles[1] != "user" {
		t.Errorf("Expected Roles to be ['admin', 'user'], got %v", user.Roles)
	}
	
	if len(user.Scopes) != 2 || user.Scopes[0] != "read" || user.Scopes[1] != "write" {
		t.Errorf("Expected Scopes to be ['read', 'write'], got %v", user.Scopes)
	}
	
	t.Logf("Token validation successful! User: %+v", user)
}

func TestValidateTokenWithInvalidSignature(t *testing.T) {
	// Generate two different RSA key pairs
	privateKey1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate first RSA key pair: %v", err)
	}
	
	privateKey2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate second RSA key pair: %v", err)
	}
	
	// Use public key from second pair for JWKS endpoint
	publicKey2 := &privateKey2.PublicKey
	
	// Create a JWK set with the second public key
	keyID := "test-key-1"
	jwkKey, err := jwk.FromRaw(publicKey2)
	if err != nil {
		t.Fatalf("Failed to create JWK from RSA public key: %v", err)
	}
	
	err = jwkKey.Set(jwk.KeyIDKey, keyID)
	if err != nil {
		t.Fatalf("Failed to set key ID: %v", err)
	}
	
	err = jwkKey.Set(jwk.AlgorithmKey, "RS256")
	if err != nil {
		t.Fatalf("Failed to set algorithm: %v", err)
	}
	
	// Create JWK set
	set := jwk.NewSet()
	set.AddKey(jwkKey)
	
	// Create mock JWKS endpoint with second public key
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jwksJSON, err := json.Marshal(set)
		if err != nil {
			http.Error(w, "Failed to marshal JWKS", http.StatusInternalServerError)
			return
		}
		w.Write(jwksJSON)
	}))
	defer mockServer.Close()
	
	// Create JWT token signed with first private key (different from JWKS)
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "https://test-auth.example.com",
		"aud": "test-audience",
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	
	tokenString, err := token.SignedString(privateKey1) // Sign with different key
	if err != nil {
		t.Fatalf("Failed to sign JWT token: %v", err)
	}
	
	// Create authenticator with mock JWKS endpoint
	auth := NewJwtAuthenticator(mockServer.URL)
	
	// Test token validation - should fail due to signature mismatch
	_, err = auth.ValidateToken(tokenString)
	if err == nil {
		t.Errorf("Expected token validation to fail due to signature mismatch, but it succeeded")
	}
	
	t.Logf("Token validation correctly failed with error: %v", err)
}

func TestValidateTokenWithExpiredToken(t *testing.T) {
	// Generate RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	
	publicKey := &privateKey.PublicKey
	
	// Create a JWK set with the public key
	keyID := "test-key-1"
	jwkKey, err := jwk.FromRaw(publicKey)
	if err != nil {
		t.Fatalf("Failed to create JWK from RSA public key: %v", err)
	}
	
	err = jwkKey.Set(jwk.KeyIDKey, keyID)
	if err != nil {
		t.Fatalf("Failed to set key ID: %v", err)
	}
	
	err = jwkKey.Set(jwk.AlgorithmKey, "RS256")
	if err != nil {
		t.Fatalf("Failed to set algorithm: %v", err)
	}
	
	// Create JWK set
	set := jwk.NewSet()
	set.AddKey(jwkKey)
	
	// Create mock JWKS endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jwksJSON, err := json.Marshal(set)
		if err != nil {
			http.Error(w, "Failed to marshal JWKS", http.StatusInternalServerError)
			return
		}
		w.Write(jwksJSON)
	}))
	defer mockServer.Close()
	
	// Create expired JWT token
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "https://test-auth.example.com",
		"aud": "test-audience",
		"exp": now.Add(-time.Hour).Unix(), // Expired 1 hour ago
		"iat": now.Add(-2*time.Hour).Unix(), // Issued 2 hours ago
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign JWT token: %v", err)
	}
	
	// Create authenticator with mock JWKS endpoint
	auth := NewJwtAuthenticator(mockServer.URL)
	
	// Test token validation - should fail due to expiration
	_, err = auth.ValidateToken(tokenString)
	if err == nil {
		t.Errorf("Expected token validation to fail due to expiration, but it succeeded")
	}
	
	t.Logf("Token validation correctly failed with error: %v", err)
}

func TestJWKSCaching(t *testing.T) {
	// Generate RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key pair: %v", err)
	}
	
	publicKey := &privateKey.PublicKey
	
	// Create a JWK set with the public key
	keyID := "test-key-1"
	jwkKey, err := jwk.FromRaw(publicKey)
	if err != nil {
		t.Fatalf("Failed to create JWK from RSA public key: %v", err)
	}
	
	err = jwkKey.Set(jwk.KeyIDKey, keyID)
	if err != nil {
		t.Fatalf("Failed to set key ID: %v", err)
	}
	
	err = jwkKey.Set(jwk.AlgorithmKey, "RS256")
	if err != nil {
		t.Fatalf("Failed to set algorithm: %v", err)
	}
	
	// Create JWK set
	set := jwk.NewSet()
	set.AddKey(jwkKey)
	
	// Track number of requests to JWKS endpoint
	requestCount := 0
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		t.Logf("JWKS endpoint called (request #%d)", requestCount)
		w.Header().Set("Content-Type", "application/json")
		jwksJSON, err := json.Marshal(set)
		if err != nil {
			http.Error(w, "Failed to marshal JWKS", http.StatusInternalServerError)
			return
		}
		w.Write(jwksJSON)
	}))
	defer mockServer.Close()
	
	// Create JWT token
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user123",
		"iss": "https://test-auth.example.com",
		"aud": "test-audience",
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = keyID
	
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign JWT token: %v", err)
	}
	
	// Create authenticator with mock JWKS endpoint
	auth := NewJwtAuthenticator(mockServer.URL)
	
	// Validate token multiple times
	for i := 0; i < 3; i++ {
		_, err := auth.ValidateToken(tokenString)
		if err != nil {
			t.Fatalf("Token validation %d failed: %v", i+1, err)
		}
	}
	
	// Should only make one request to JWKS endpoint due to caching
	if requestCount != 1 {
		t.Errorf("Expected 1 request to JWKS endpoint due to caching, got %d", requestCount)
	}
	
	t.Logf("JWKS caching test passed - made %d request(s) for 3 token validations", requestCount)
}

func TestFetchKeyWithTimeout(t *testing.T) {
	// Create a slow/hanging JWKS endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response that exceeds timeout
		time.Sleep(35 * time.Second) // Longer than our 30s timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()
	
	// Create authenticator with mock JWKS endpoint
	auth := NewJwtAuthenticator(mockServer.URL)
	
	// Test fetchKey with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	
	_, err := auth.fetchKey(ctx, "test-key")
	if err == nil {
		t.Error("Expected fetchKey to fail due to timeout, but it succeeded")
	}
	
	t.Logf("fetchKey correctly failed with timeout error: %v", err)
}