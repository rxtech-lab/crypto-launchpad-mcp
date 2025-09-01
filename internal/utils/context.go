package utils

import (
	"context"
)

// MCPAuthenticatedUserContextKey is the context key for storing authenticated user in MCP contexts
// This is separate from the Fiber middleware context key to avoid confusion
const MCPAuthenticatedUserContextKey = "mcp_authenticated_user"

// WithAuthenticatedUser stores an authenticated user in the context
func WithAuthenticatedUser(ctx context.Context, user *AuthenticatedUser) context.Context {
	return context.WithValue(ctx, MCPAuthenticatedUserContextKey, user)
}

// GetAuthenticatedUser retrieves the authenticated user from context
// Returns the user and a boolean indicating if the user was found
func GetAuthenticatedUser(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(MCPAuthenticatedUserContextKey).(*AuthenticatedUser)
	return user, ok
}

// MustGetAuthenticatedUser retrieves the authenticated user from context
// Panics if no user is found - use only when user is guaranteed to exist
func MustGetAuthenticatedUser(ctx context.Context) *AuthenticatedUser {
	user, ok := GetAuthenticatedUser(ctx)
	if !ok {
		panic("no authenticated user found in context")
	}
	return user
}

// IsAuthenticated checks if there is an authenticated user in the context
func IsAuthenticated(ctx context.Context) bool {
	_, ok := GetAuthenticatedUser(ctx)
	return ok
}

// GetUserID returns the user ID (subject) if authenticated, empty string otherwise
func GetUserID(ctx context.Context) string {
	user, ok := GetAuthenticatedUser(ctx)
	if !ok {
		return ""
	}
	return user.Sub
}

// GetUserRoles returns the user roles if authenticated, empty slice otherwise
func GetUserRoles(ctx context.Context) []string {
	user, ok := GetAuthenticatedUser(ctx)
	if !ok {
		return []string{}
	}
	return user.Roles
}

// HasRole checks if the authenticated user has a specific role
func HasRole(ctx context.Context, role string) bool {
	roles := GetUserRoles(ctx)
	for _, userRole := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// GetUserScopes returns the user scopes if authenticated, empty slice otherwise
func GetUserScopes(ctx context.Context) []string {
	user, ok := GetAuthenticatedUser(ctx)
	if !ok {
		return []string{}
	}
	return user.Scopes
}

// HasScope checks if the authenticated user has a specific scope
func HasScope(ctx context.Context, scope string) bool {
	scopes := GetUserScopes(ctx)
	for _, userScope := range scopes {
		if userScope == scope {
			return true
		}
	}
	return false
}
