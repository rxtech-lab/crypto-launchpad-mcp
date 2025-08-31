package middleware

import (
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

// AuthConfig holds configuration for the auth middleware
type AuthConfig struct {
	// ResourceID is the expected audience for token validation
	ResourceID string
	// TokenValidator is a function that validates the bearer token
	// It should return an error if the token is invalid
	TokenValidator func(token string, audience []string) error
	// JWTAuthenticator for JWT token validation (optional, takes precedence over TokenValidator)
	JWTAuthenticator *utils.JwtAuthenticator
	// SkipWellKnown determines if .well-known endpoints should bypass auth
	SkipWellKnown bool
}

// DefaultAuthConfig provides default configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		SkipWellKnown: true,
		TokenValidator: func(token string, audience []string) error {
			// Default implementation - should be overridden
			if token == "" {
				return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
			}
			return nil
		},
	}
}

// AuthMiddleware returns a Fiber middleware for Bearer token authentication
func AuthMiddleware(config ...AuthConfig) fiber.Handler {
	cfg := DefaultAuthConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// Allow public access to well-known endpoints for metadata discovery
		if cfg.SkipWellKnown && strings.Contains(c.Path(), ".well-known") {
			return c.Next()
		}

		// Extract Bearer token from Authorization header
		authHeader := c.Get("Authorization")
		var token string

		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}

		if token == "" {
			c.Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Oauth", resource_metadata="%s"`, os.Getenv("SCALEKIT_RESOURCE_METADATA_URL")))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing or invalid Bearer token",
			})
		}

		// Validate token using JWT authenticator if available, otherwise fall back to TokenValidator
		if cfg.JWTAuthenticator != nil {
			// Use JWT authenticator for validation
			user, err := cfg.JWTAuthenticator.ValidateToken(token)
			if err != nil {
				c.Set("WWW-Authenticate", `Bearer realm="Access to protected resource"`)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid token",
					"details": err.Error(),
				})
			}

			// Check if user has required audience (if specified)
			if cfg.ResourceID != "" {
				hasValidAudience := false
				for _, userAud := range user.Aud {
					if userAud == cfg.ResourceID {
						hasValidAudience = true
						break
					}
				}
				if !hasValidAudience {
					c.Set("WWW-Authenticate", `Bearer realm="Access to protected resource"`)
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error": "Invalid audience",
					})
				}
			}

			// Store authenticated user in context
			c.Locals("user", user)
		} else {
			// Fall back to custom token validator
			var audience []string
			if cfg.ResourceID != "" {
				audience = []string{cfg.ResourceID}
			}

			if err := cfg.TokenValidator(token, audience); err != nil {
				c.Set("WWW-Authenticate", `Bearer realm="Access to protected resource"`)
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid token",
				})
			}
		}

		return c.Next()
	}
}

// GetAuthenticatedUser retrieves the authenticated user from Fiber context
// Returns nil if no user is found or if user is not of correct type
func GetAuthenticatedUser(c *fiber.Ctx) *utils.AuthenticatedUser {
	userInterface := c.Locals("user")
	if userInterface == nil {
		return nil
	}
	
	user, ok := userInterface.(*utils.AuthenticatedUser)
	if !ok {
		return nil
	}
	
	return user
}
