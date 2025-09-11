package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSimpleJwtAuthenticator(t *testing.T) {
	t.Run("ValidSecret", func(t *testing.T) {
		secret := "test-secret"
		authenticator, err := NewSimpleJwtAuthenticator(secret)

		require.NoError(t, err)
		assert.Equal(t, secret, authenticator.Secret)
	})

	t.Run("EmptySecret", func(t *testing.T) {
		_, err := NewSimpleJwtAuthenticator("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "JWT secret not configured")
	})
}

func TestSimpleJwtAuthenticator_ValidateToken(t *testing.T) {
	secret := "test-secret-key"
	authenticator, err := NewSimpleJwtAuthenticator(secret)
	require.NoError(t, err)

	t.Run("ValidToken", func(t *testing.T) {
		// Create test token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":       "user123",
			"iss":       "test-issuer",
			"client_id": "client123",
			"jti":       "jwt123",
			"oid":       "object123",
			"resid":     "resource123",
			"sid":       "session123",
			"exp":       float64(time.Now().Add(time.Hour).Unix()),
			"iat":       float64(time.Now().Unix()),
			"nbf":       float64(time.Now().Unix()),
			"aud":       []interface{}{"audience1", "audience2"},
			"roles":     []interface{}{"admin", "user"},
			"scopes":    []interface{}{"read", "write"},
		})

		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		user, err := authenticator.ValidateToken(tokenString)

		require.NoError(t, err)
		assert.Equal(t, "user123", user.Sub)
		assert.Equal(t, "test-issuer", user.Iss)
		assert.Equal(t, "client123", user.ClientId)
		assert.Equal(t, "jwt123", user.Jti)
		assert.Equal(t, "object123", user.Oid)
		assert.Equal(t, "resource123", user.Resid)
		assert.Equal(t, "session123", user.Sid)
		assert.Equal(t, []string{"audience1", "audience2"}, user.Aud)
		assert.Equal(t, []string{"admin", "user"}, user.Roles)
		assert.Equal(t, []string{"read", "write"}, user.Scopes)
		assert.Greater(t, user.Exp, 0)
		assert.Greater(t, user.Iat, 0)
		assert.Greater(t, user.Nbf, 0)
	})

	t.Run("ValidTokenWithSingleAudience", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
			"aud": "single-audience",
		})

		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		user, err := authenticator.ValidateToken(tokenString)

		require.NoError(t, err)
		assert.Equal(t, []string{"single-audience"}, user.Aud)
	})

	t.Run("ValidTokenWithSingleScope", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub":   "user123",
			"scope": "read write execute",
		})

		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		user, err := authenticator.ValidateToken(tokenString)

		require.NoError(t, err)
		assert.Equal(t, []string{"read write execute"}, user.Scopes)
	})

	t.Run("TokenWithMissingClaims", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
		})

		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		user, err := authenticator.ValidateToken(tokenString)

		require.NoError(t, err)
		assert.Equal(t, "user123", user.Sub)
		assert.Empty(t, user.Iss)
		assert.Empty(t, user.ClientId)
		assert.Empty(t, user.Roles)
		assert.Empty(t, user.Scopes)
		assert.Empty(t, user.Aud)
	})

	t.Run("InvalidSigningMethod", func(t *testing.T) {
		// This will fail because we're using RS256 but the authenticator expects HMAC
		// We need to create a token string manually for this test
		tokenString := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyMTIzIn0.invalid"

		_, err := authenticator.ValidateToken(tokenString)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected signing method")
	})

	t.Run("InvalidTokenFormat", func(t *testing.T) {
		_, err := authenticator.ValidateToken("invalid-token")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse/validate token")
	})

	t.Run("EmptyToken", func(t *testing.T) {
		_, err := authenticator.ValidateToken("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse/validate token")
	})

	t.Run("TokenWithWrongSecret", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
		})

		// Sign with wrong secret
		tokenString, err := token.SignedString([]byte("wrong-secret"))
		require.NoError(t, err)

		_, err = authenticator.ValidateToken(tokenString)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse/validate token")
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "user123",
			"exp": float64(time.Now().Add(-time.Hour).Unix()), // Expired 1 hour ago
		})

		tokenString, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		_, err = authenticator.ValidateToken(tokenString)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse/validate token")
	})
}

func TestSimpleJwtAuthenticator_mapClaimsToUser(t *testing.T) {
	authenticator, err := NewSimpleJwtAuthenticator("test-secret")
	require.NoError(t, err)

	t.Run("AllClaims", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":       "user123",
			"iss":       "test-issuer",
			"client_id": "client123",
			"jti":       "jwt123",
			"oid":       "object123",
			"resid":     "resource123",
			"sid":       "session123",
			"exp":       float64(1234567890),
			"iat":       float64(1234567800),
			"nbf":       float64(1234567750),
			"aud":       []interface{}{"aud1", "aud2"},
			"roles":     []interface{}{"admin", "user"},
			"scopes":    []interface{}{"read", "write"},
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		assert.Equal(t, "user123", user.Sub)
		assert.Equal(t, "test-issuer", user.Iss)
		assert.Equal(t, "client123", user.ClientId)
		assert.Equal(t, "jwt123", user.Jti)
		assert.Equal(t, "object123", user.Oid)
		assert.Equal(t, "resource123", user.Resid)
		assert.Equal(t, "session123", user.Sid)
		assert.Equal(t, 1234567890, user.Exp)
		assert.Equal(t, 1234567800, user.Iat)
		assert.Equal(t, 1234567750, user.Nbf)
		assert.Equal(t, []string{"aud1", "aud2"}, user.Aud)
		assert.Equal(t, []string{"admin", "user"}, user.Roles)
		assert.Equal(t, []string{"read", "write"}, user.Scopes)
	})

	t.Run("SingleAudienceAsString", func(t *testing.T) {
		claims := jwt.MapClaims{
			"aud": "single-audience",
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		assert.Equal(t, []string{"single-audience"}, user.Aud)
	})

	t.Run("SingleScopeAsString", func(t *testing.T) {
		claims := jwt.MapClaims{
			"scope": "read write",
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		assert.Equal(t, []string{"read write"}, user.Scopes)
	})

	t.Run("InvalidAudienceTypes", func(t *testing.T) {
		claims := jwt.MapClaims{
			"aud": []interface{}{"valid", 123, "also-valid"},
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		// Only string values should be included
		assert.Equal(t, []string{"valid", "also-valid"}, user.Aud)
	})

	t.Run("InvalidRoleTypes", func(t *testing.T) {
		claims := jwt.MapClaims{
			"roles": []interface{}{"admin", 456, "user"},
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		// Only string values should be included
		assert.Equal(t, []string{"admin", "user"}, user.Roles)
	})

	t.Run("InvalidScopeTypes", func(t *testing.T) {
		claims := jwt.MapClaims{
			"scopes": []interface{}{"read", 789, "write"},
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		// Only string values should be included
		assert.Equal(t, []string{"read", "write"}, user.Scopes)
	})

	t.Run("WrongTypeForStringClaims", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":       123,    // Should be string
			"iss":       456,    // Should be string
			"client_id": 789,    // Should be string
			"jti":       "good", // Correct type
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		// Only correctly typed values should be set
		assert.Empty(t, user.Sub)
		assert.Empty(t, user.Iss)
		assert.Empty(t, user.ClientId)
		assert.Equal(t, "good", user.Jti)
	})

	t.Run("WrongTypeForNumericClaims", func(t *testing.T) {
		claims := jwt.MapClaims{
			"exp": "not-a-number",
			"iat": 1234567890.5, // Should work
			"nbf": "also-not-a-number",
		}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		// Only correctly typed values should be set
		assert.Equal(t, 0, user.Exp)
		assert.Equal(t, 1234567890, user.Iat)
		assert.Equal(t, 0, user.Nbf)
	})

	t.Run("EmptyClaims", func(t *testing.T) {
		claims := jwt.MapClaims{}

		user, err := authenticator.mapClaimsToUser(claims)

		require.NoError(t, err)
		assert.Equal(t, &AuthenticatedUser{}, user)
	})
}
