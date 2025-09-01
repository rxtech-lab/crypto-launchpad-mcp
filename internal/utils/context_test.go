package utils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithAuthenticatedUser(t *testing.T) {
	user := &AuthenticatedUser{
		Sub:      "user123",
		ClientId: "client456",
		Aud:      []string{"api://test"},
		Roles:    []string{"admin", "user"},
		Scopes:   []string{"read", "write"},
	}

	ctx := context.Background()

	// Add user to context
	ctxWithUser := WithAuthenticatedUser(ctx, user)

	// Verify user is stored
	retrievedUser, ok := GetAuthenticatedUser(ctxWithUser)
	require.True(t, ok)
	assert.Equal(t, user.Sub, retrievedUser.Sub)
	assert.Equal(t, user.ClientId, retrievedUser.ClientId)
	assert.Equal(t, user.Aud, retrievedUser.Aud)
	assert.Equal(t, user.Roles, retrievedUser.Roles)
	assert.Equal(t, user.Scopes, retrievedUser.Scopes)
}

func TestGetAuthenticatedUser(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected bool
		userSub  string
	}{
		{
			name: "Context with authenticated user",
			setup: func() context.Context {
				user := &AuthenticatedUser{Sub: "user123"}
				return WithAuthenticatedUser(context.Background(), user)
			},
			expected: true,
			userSub:  "user123",
		},
		{
			name: "Context without authenticated user",
			setup: func() context.Context {
				return context.Background()
			},
			expected: false,
		},
		{
			name: "Context with wrong type in key",
			setup: func() context.Context {
				return context.WithValue(context.Background(), MCPAuthenticatedUserContextKey, "not-a-user")
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			user, ok := GetAuthenticatedUser(ctx)

			assert.Equal(t, tt.expected, ok)
			if tt.expected {
				assert.NotNil(t, user)
				assert.Equal(t, tt.userSub, user.Sub)
			} else {
				assert.Nil(t, user)
			}
		})
	}
}

func TestMustGetAuthenticatedUser(t *testing.T) {
	t.Run("With authenticated user - should return user", func(t *testing.T) {
		user := &AuthenticatedUser{Sub: "user123"}
		ctx := WithAuthenticatedUser(context.Background(), user)

		retrievedUser := MustGetAuthenticatedUser(ctx)
		assert.Equal(t, user.Sub, retrievedUser.Sub)
	})

	t.Run("Without authenticated user - should panic", func(t *testing.T) {
		ctx := context.Background()

		assert.Panics(t, func() {
			MustGetAuthenticatedUser(ctx)
		})
	})
}

func TestIsAuthenticated(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected bool
	}{
		{
			name: "Context with authenticated user",
			setup: func() context.Context {
				user := &AuthenticatedUser{Sub: "user123"}
				return WithAuthenticatedUser(context.Background(), user)
			},
			expected: true,
		},
		{
			name: "Context without authenticated user",
			setup: func() context.Context {
				return context.Background()
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := IsAuthenticated(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserID(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected string
	}{
		{
			name: "Context with authenticated user",
			setup: func() context.Context {
				user := &AuthenticatedUser{Sub: "user123"}
				return WithAuthenticatedUser(context.Background(), user)
			},
			expected: "user123",
		},
		{
			name: "Context without authenticated user",
			setup: func() context.Context {
				return context.Background()
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetUserID(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserRoles(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected []string
	}{
		{
			name: "Context with authenticated user having roles",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:   "user123",
					Roles: []string{"admin", "user"},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			expected: []string{"admin", "user"},
		},
		{
			name: "Context with authenticated user having no roles",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:   "user123",
					Roles: []string{},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			expected: []string{},
		},
		{
			name: "Context without authenticated user",
			setup: func() context.Context {
				return context.Background()
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetUserRoles(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasRole(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		role     string
		expected bool
	}{
		{
			name: "User has the role",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:   "user123",
					Roles: []string{"admin", "user"},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			role:     "admin",
			expected: true,
		},
		{
			name: "User does not have the role",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:   "user123",
					Roles: []string{"user"},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			role:     "admin",
			expected: false,
		},
		{
			name: "No authenticated user",
			setup: func() context.Context {
				return context.Background()
			},
			role:     "admin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := HasRole(ctx, tt.role)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetUserScopes(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		expected []string
	}{
		{
			name: "Context with authenticated user having scopes",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:    "user123",
					Scopes: []string{"read", "write", "admin"},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			expected: []string{"read", "write", "admin"},
		},
		{
			name: "Context with authenticated user having no scopes",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:    "user123",
					Scopes: []string{},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			expected: []string{},
		},
		{
			name: "Context without authenticated user",
			setup: func() context.Context {
				return context.Background()
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := GetUserScopes(ctx)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasScope(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() context.Context
		scope    string
		expected bool
	}{
		{
			name: "User has the scope",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:    "user123",
					Scopes: []string{"read", "write", "admin"},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			scope:    "admin",
			expected: true,
		},
		{
			name: "User does not have the scope",
			setup: func() context.Context {
				user := &AuthenticatedUser{
					Sub:    "user123",
					Scopes: []string{"read", "write"},
				}
				return WithAuthenticatedUser(context.Background(), user)
			},
			scope:    "admin",
			expected: false,
		},
		{
			name: "No authenticated user",
			setup: func() context.Context {
				return context.Background()
			},
			scope:    "admin",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup()
			result := HasScope(ctx, tt.scope)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContextPropagation(t *testing.T) {
	// Test that context with authenticated user can be passed through function calls
	user := &AuthenticatedUser{
		Sub:      "user123",
		ClientId: "client456",
		Aud:      []string{"api://test"},
		Roles:    []string{"admin"},
		Scopes:   []string{"read", "write"},
	}

	ctx := WithAuthenticatedUser(context.Background(), user)

	// Simulate passing context through function calls
	simulateToolHandler := func(ctx context.Context) (string, bool, []string, []string) {
		userID := GetUserID(ctx)
		isAuth := IsAuthenticated(ctx)
		roles := GetUserRoles(ctx)
		scopes := GetUserScopes(ctx)
		return userID, isAuth, roles, scopes
	}

	userID, isAuth, roles, scopes := simulateToolHandler(ctx)

	assert.Equal(t, "user123", userID)
	assert.True(t, isAuth)
	assert.Equal(t, []string{"admin"}, roles)
	assert.Equal(t, []string{"read", "write"}, scopes)
}

func TestContextKeyUniqueness(t *testing.T) {
	// Test that our context key doesn't conflict with other keys
	ctx := context.Background()

	// Add our authenticated user
	user := &AuthenticatedUser{Sub: "user123"}
	ctx = WithAuthenticatedUser(ctx, user)

	// Add some other data with a similar key name
	ctx = context.WithValue(ctx, "authenticated_user", "different_value")
	ctx = context.WithValue(ctx, "user", "another_value")

	// Our authenticated user should still be accessible
	retrievedUser, ok := GetAuthenticatedUser(ctx)
	require.True(t, ok)
	assert.Equal(t, "user123", retrievedUser.Sub)

	// Verify context key is what we expect
	assert.Equal(t, "mcp_authenticated_user", MCPAuthenticatedUserContextKey)
}
