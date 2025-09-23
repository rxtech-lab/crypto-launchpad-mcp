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
	TokenValidator func(token string, audience []string) (*utils.AuthenticatedUser, error)
	// SkipWellKnown determines if .well-known endpoints should bypass auth
	SkipWellKnown bool
}

const (
	AuthenticatedUserContextKey = "authenticatedUser"
)

// DefaultAuthConfig provides default configuration
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		SkipWellKnown: true,
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			// Default implementation - should be overridden
			if token == "" {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
			}
			return &utils.AuthenticatedUser{}, nil
		},
	}
}

// OauthAuthMiddleware returns a Fiber middleware for Bearer token authentication
func OauthAuthMiddleware(config ...AuthConfig) fiber.Handler {
	cfg := DefaultAuthConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		// check if the ctx already has an authenticated user
		if c.Locals(AuthenticatedUserContextKey) != nil {
			return c.Next()
		}

		// skip /mcp routes
		if strings.HasPrefix(c.Path(), "/mcp") {
			return c.Next()
		}
		// skip /tx routes
		if strings.HasPrefix(c.Path(), "/tx") {
			return c.Next()
		}

		// skip /static routes
		if strings.HasPrefix(c.Path(), "/static") {
			return c.Next()
		}

		if strings.HasPrefix(c.Path(), "/api/tx") {
			return c.Next()
		}

		// skip /health route
		if c.Path() == "/health" {
			return c.Next()
		}

		// Skip auth for well-known endpoints if configured
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

		// Validate token against configured resource audience
		var audience []string
		if cfg.ResourceID != "" {
			audience = []string{cfg.ResourceID}
		}

		if usr, err := cfg.TokenValidator(token, audience); err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="Access to protected resource"`)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		} else {
			// Set the authenticated user in the context
			c.Locals(AuthenticatedUserContextKey, usr)
		}

		return c.Next()
	}
}
