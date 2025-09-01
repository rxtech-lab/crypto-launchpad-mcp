package utils

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type JwtAuthenticator struct {
	JwksUri    string
	cachedKeys jwk.Set
	lastFetch  time.Time
	cacheTTL   time.Duration
	mu         sync.RWMutex
}

type AuthenticatedUser struct {
	Aud      []string `json:"aud"`
	ClientId string   `json:"client_id"`
	Exp      int      `json:"exp"`
	Iat      int      `json:"iat"`
	Iss      string   `json:"iss"`
	Jti      string   `json:"jti"`
	Nbf      int      `json:"nbf"`
	Oid      string   `json:"oid"`
	Resid    string   `json:"resid"`
	Roles    []string `json:"roles"`
	Scopes   []string `json:"scopes"`
	Sid      string   `json:"sid"`
	Sub      string   `json:"sub"`
}

func NewJwtAuthenticator(jwksUri string) JwtAuthenticator {
	return JwtAuthenticator{
		JwksUri:  jwksUri,
		cacheTTL: 5 * time.Minute, // Cache keys for 5 minutes
	}
}

func (ja *JwtAuthenticator) fetchKey(ctx context.Context, keyID string) (interface{}, error) {
	ja.mu.RLock()
	// Check if we have cached keys and they're still valid
	if ja.cachedKeys != nil && time.Since(ja.lastFetch) < ja.cacheTTL {
		if key, exists := ja.cachedKeys.LookupKeyID(keyID); exists {
			ja.mu.RUnlock()
			return ja.convertJWKToPublicKey(key)
		}
	}
	ja.mu.RUnlock()

	// Need to fetch fresh keys
	ja.mu.Lock()
	defer ja.mu.Unlock()

	// Double-check after acquiring write lock
	if ja.cachedKeys != nil && time.Since(ja.lastFetch) < ja.cacheTTL {
		if key, exists := ja.cachedKeys.LookupKeyID(keyID); exists {
			return ja.convertJWKToPublicKey(key)
		}
	}

	// Fetch JWKS from the endpoint
	keySet, err := jwk.Fetch(ctx, ja.JwksUri)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", ja.JwksUri, err)
	}

	ja.cachedKeys = keySet
	ja.lastFetch = time.Now()

	// Look for the specific key ID
	key, exists := keySet.LookupKeyID(keyID)
	if !exists {
		return nil, fmt.Errorf("key with ID '%s' not found in JWKS", keyID)
	}

	return ja.convertJWKToPublicKey(key)
}

func (ja *JwtAuthenticator) convertJWKToPublicKey(key jwk.Key) (interface{}, error) {
	var publicKey interface{}
	if err := key.Raw(&publicKey); err != nil {
		return nil, fmt.Errorf("failed to convert JWK to public key: %w", err)
	}

	// Ensure it's an RSA public key for RS256
	if rsaKey, ok := publicKey.(*rsa.PublicKey); ok {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("expected RSA public key, got %T", publicKey)
}

func (ja *JwtAuthenticator) ValidateToken(token string) (*AuthenticatedUser, error) {
	if ja.JwksUri == "" {
		return nil, fmt.Errorf("JWKS URI not configured")
	}

	// Parse the JWT token
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		// Verify that the signing method is RS256
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get the key ID from the token header
		keyID, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("missing or invalid 'kid' claim in token header")
		}

		// Fetch the public key for this key ID
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		return ja.fetchKey(ctx, keyID)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse/validate token: %w", err)
	}

	// Check if the token is valid
	if !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse token claims")
	}

	// Convert claims to AuthenticatedUser
	user, err := ja.mapClaimsToUser(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to map claims to user: %w", err)
	}

	log.Printf("Successfully validated token for user: %s", user.Sub)
	return user, nil
}

func (ja *JwtAuthenticator) mapClaimsToUser(claims jwt.MapClaims) (*AuthenticatedUser, error) {
	user := &AuthenticatedUser{}

	// Extract standard claims
	if sub, ok := claims["sub"].(string); ok {
		user.Sub = sub
	}

	if iss, ok := claims["iss"].(string); ok {
		user.Iss = iss
	}

	if clientId, ok := claims["client_id"].(string); ok {
		user.ClientId = clientId
	}

	if jti, ok := claims["jti"].(string); ok {
		user.Jti = jti
	}

	if oid, ok := claims["oid"].(string); ok {
		user.Oid = oid
	}

	if resid, ok := claims["resid"].(string); ok {
		user.Resid = resid
	}

	if sid, ok := claims["sid"].(string); ok {
		user.Sid = sid
	}

	// Extract numeric claims
	if exp, ok := claims["exp"].(float64); ok {
		user.Exp = int(exp)
	}

	if iat, ok := claims["iat"].(float64); ok {
		user.Iat = int(iat)
	}

	if nbf, ok := claims["nbf"].(float64); ok {
		user.Nbf = int(nbf)
	}

	// Extract array claims
	if aud, ok := claims["aud"].([]interface{}); ok {
		for _, a := range aud {
			if audStr, ok := a.(string); ok {
				user.Aud = append(user.Aud, audStr)
			}
		}
	} else if audStr, ok := claims["aud"].(string); ok {
		// Handle single audience as string
		user.Aud = []string{audStr}
	}

	if roles, ok := claims["roles"].([]interface{}); ok {
		for _, r := range roles {
			if roleStr, ok := r.(string); ok {
				user.Roles = append(user.Roles, roleStr)
			}
		}
	}

	if scopes, ok := claims["scopes"].([]interface{}); ok {
		for _, s := range scopes {
			if scopeStr, ok := s.(string); ok {
				user.Scopes = append(user.Scopes, scopeStr)
			}
		}
	} else if scopeStr, ok := claims["scope"].(string); ok {
		// Handle space-separated scopes (common OAuth format)
		user.Scopes = []string{scopeStr}
	}

	return user, nil
}
