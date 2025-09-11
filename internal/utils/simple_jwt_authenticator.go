package utils

import (
	"fmt"
	"log"

	"github.com/golang-jwt/jwt/v4"
)

type SimpleJwtAuthenticator struct {
	Secret string
}

func NewSimpleJwtAuthenticator(secret string) (SimpleJwtAuthenticator, error) {
	if len(secret) == 0 {
		return SimpleJwtAuthenticator{}, fmt.Errorf("JWT secret not configured")
	}
	return SimpleJwtAuthenticator{
		Secret: secret,
	}, nil
}

func (sja *SimpleJwtAuthenticator) ValidateToken(tokenString string) (*AuthenticatedUser, error) {
	// Parse the JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify that the signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(sja.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse/validate token: %w", err)
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to parse token claims")
	}

	// Convert claims to AuthenticatedUser
	user, err := sja.mapClaimsToUser(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to map claims to user: %w", err)
	}

	log.Printf("Successfully validated JWT token for user: %s", user.Sub)
	return user, nil
}

func (sja *SimpleJwtAuthenticator) mapClaimsToUser(claims jwt.MapClaims) (*AuthenticatedUser, error) {
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
