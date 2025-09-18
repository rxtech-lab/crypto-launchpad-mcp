package middleware

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
)

// JwtAuthMiddleware provides optional JWT authentication middleware
// If a valid JWT token is present, it sets the authenticated user in context
// Always calls next() - never blocks requests
func JwtAuthMiddleware(authenticator *utils.SimpleJwtAuthenticator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// check if the ctx already has an authenticated user
		if c.Locals(AuthenticatedUserContextKey) != nil {
			return c.Next()
		}

		// Skip if no authenticator configured
		if authenticator == nil {
			return c.Next()
		}

		// Extract Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			// No authorization header - continue without authentication
			return c.Next()
		}

		// Check for Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			// Not a Bearer token - continue without authentication
			return c.Next()
		}

		// Extract token
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		if token == "" {
			// Empty token - continue without authentication
			return c.Next()
		}

		// Validate JWT token
		user, err := authenticator.ValidateToken(token)
		if err != nil {
			// Invalid token - log and continue without authentication
			log.Printf("JWT validation failed: %v", err)
			return c.Next()
		}

		// Valid token - set authenticated user in context
		c.Locals(AuthenticatedUserContextKey, user)
		log.Printf("JWT authentication successful for user: %s", user.Sub)

		return c.Next()
	}
}
