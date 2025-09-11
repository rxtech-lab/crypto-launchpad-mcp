package middleware

import (
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_UserStoredInContext(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		config         AuthConfig
		expectedStatus int
		expectedUser   *utils.AuthenticatedUser
		shouldHaveUser bool
	}{
		{
			name:       "Valid token - user should be stored in context",
			authHeader: "Bearer valid-token",
			config: AuthConfig{
				SkipWellKnown: true,
				TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
					if token == "valid-token" {
						return &utils.AuthenticatedUser{
							Sub:      "user123",
							ClientId: "client123",
							Aud:      []string{"api://test"},
							Roles:    []string{"admin", "user"},
							Scopes:   []string{"read", "write"},
						}, nil
					}
					return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
				},
			},
			expectedStatus: fiber.StatusOK,
			expectedUser: &utils.AuthenticatedUser{
				Sub:      "user123",
				ClientId: "client123",
				Aud:      []string{"api://test"},
				Roles:    []string{"admin", "user"},
				Scopes:   []string{"read", "write"},
			},
			shouldHaveUser: true,
		},
		{
			name:       "Invalid token - no user in context",
			authHeader: "Bearer invalid-token",
			config: AuthConfig{
				SkipWellKnown: true,
				TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
					return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
				},
			},
			expectedStatus: fiber.StatusUnauthorized,
			shouldHaveUser: false,
		},
		{
			name:       "Missing token - no user in context",
			authHeader: "",
			config: AuthConfig{
				SkipWellKnown: true,
				TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
					return &utils.AuthenticatedUser{Sub: "user123"}, nil
				},
			},
			expectedStatus: fiber.StatusUnauthorized,
			shouldHaveUser: false,
		},
		{
			name:       "Well-known endpoint - should skip auth",
			authHeader: "",
			config: AuthConfig{
				SkipWellKnown: true,
				TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
					return nil, fiber.NewError(fiber.StatusUnauthorized, "Should not be called")
				},
			},
			expectedStatus: fiber.StatusOK,
			shouldHaveUser: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Add the auth middleware
			app.Use(OauthAuthMiddleware(tt.config))

			// Add a test route that checks for the authenticated user in context
			app.Get("/test", func(c *fiber.Ctx) error {
				user := c.Locals(AuthenticatedUserContextKey)

				if tt.shouldHaveUser {
					require.NotNil(t, user, "Expected user to be in context but got nil")

					authenticatedUser, ok := user.(*utils.AuthenticatedUser)
					require.True(t, ok, "Expected user to be of type *utils.AuthenticatedUser")

					// Verify the user details
					assert.Equal(t, tt.expectedUser.Sub, authenticatedUser.Sub)
					assert.Equal(t, tt.expectedUser.ClientId, authenticatedUser.ClientId)
					assert.Equal(t, tt.expectedUser.Aud, authenticatedUser.Aud)
					assert.Equal(t, tt.expectedUser.Roles, authenticatedUser.Roles)
					assert.Equal(t, tt.expectedUser.Scopes, authenticatedUser.Scopes)

					return c.JSON(fiber.Map{
						"user_found": true,
						"user_sub":   authenticatedUser.Sub,
						"user_roles": authenticatedUser.Roles,
					})
				} else {
					assert.Nil(t, user, "Expected no user in context but found one")
					return c.JSON(fiber.Map{"user_found": false})
				}
			})

			// Add a well-known endpoint for testing skip behavior
			app.Get("/.well-known/test", func(c *fiber.Ctx) error {
				user := c.Locals(AuthenticatedUserContextKey)
				return c.JSON(fiber.Map{
					"well_known": true,
					"user_found": user != nil,
				})
			})

			// Create request
			path := "/test"
			if tt.name == "Well-known endpoint - should skip auth" {
				path = "/.well-known/test"
			}

			req := httptest.NewRequest("GET", path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Execute request
			resp, err := app.Test(req)
			require.NoError(t, err)

			// Check status code
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			// For successful requests, verify the response body
			if resp.StatusCode == fiber.StatusOK {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				t.Logf("Response body: %s", string(body))
			}
		})
	}
}

func TestAuthMiddleware_ResourceIDValidation(t *testing.T) {
	testUser := &utils.AuthenticatedUser{
		Sub:      "user123",
		ClientId: "client123",
		Aud:      []string{"api://resource1"},
	}

	tests := []struct {
		name           string
		resourceID     string
		expectedStatus int
		tokenValidator func(token string, audience []string) (*utils.AuthenticatedUser, error)
	}{
		{
			name:           "Valid audience - should pass",
			resourceID:     "api://resource1",
			expectedStatus: fiber.StatusOK,
			tokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
				// Verify the audience was passed correctly
				assert.Contains(t, audience, "api://resource1")
				return testUser, nil
			},
		},
		{
			name:           "No resource ID - should still pass with empty audience",
			resourceID:     "",
			expectedStatus: fiber.StatusOK,
			tokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
				// Verify empty audience
				assert.Empty(t, audience)
				return testUser, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			config := AuthConfig{
				SkipWellKnown:  true,
				ResourceID:     tt.resourceID,
				TokenValidator: tt.tokenValidator,
			}

			app.Use(OauthAuthMiddleware(config))

			app.Get("/test", func(c *fiber.Ctx) error {
				user := c.Locals(AuthenticatedUserContextKey)
				authenticatedUser, ok := user.(*utils.AuthenticatedUser)
				assert.True(t, ok)
				assert.Equal(t, testUser.Sub, authenticatedUser.Sub)

				return c.JSON(fiber.Map{"success": true})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer valid-token")

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestAuthMiddleware_WWWAuthenticateHeaders(t *testing.T) {
	tests := []struct {
		name               string
		authHeader         string
		expectedWWWAuth    string
		expectScalekitMeta bool
	}{
		{
			name:               "Missing token - should include Scalekit metadata",
			authHeader:         "",
			expectScalekitMeta: true,
		},
		{
			name:               "Invalid token - should include basic realm",
			authHeader:         "Bearer invalid",
			expectedWWWAuth:    `Bearer realm="Access to protected resource"`,
			expectScalekitMeta: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable for Scalekit test
			if tt.expectScalekitMeta {
				t.Setenv("SCALEKIT_RESOURCE_METADATA_URL", "https://example.com/metadata")
			}

			app := fiber.New()

			config := AuthConfig{
				SkipWellKnown: true,
				TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
					return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
				},
			}

			app.Use(OauthAuthMiddleware(config))

			app.Get("/test", func(c *fiber.Ctx) error {
				return c.JSON(fiber.Map{"success": true})
			})

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

			wwwAuth := resp.Header.Get("WWW-Authenticate")
			if tt.expectScalekitMeta {
				assert.Contains(t, wwwAuth, `Bearer realm="Oauth"`)
				assert.Contains(t, wwwAuth, `resource_metadata="https://example.com/metadata"`)
			} else if tt.expectedWWWAuth != "" {
				assert.Equal(t, tt.expectedWWWAuth, wwwAuth)
			}
		})
	}
}

func TestAuthMiddleware_DefaultConfig(t *testing.T) {
	config := DefaultAuthConfig()

	assert.True(t, config.SkipWellKnown)
	assert.NotNil(t, config.TokenValidator)

	// Test default token validator
	user, err := config.TokenValidator("", []string{})
	assert.Nil(t, user)
	assert.Error(t, err)

	user, err = config.TokenValidator("some-token", []string{})
	assert.NotNil(t, user)
	assert.NoError(t, err)
	assert.IsType(t, &utils.AuthenticatedUser{}, user)
}

func TestAuthMiddleware_ContextKeyConstant(t *testing.T) {
	// Test that the context key constant is properly defined
	assert.Equal(t, "authenticatedUser", AuthenticatedUserContextKey)

	// Test using the constant
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		// Manually set a user in context using the constant
		testUser := &utils.AuthenticatedUser{Sub: "test123"}
		c.Locals(AuthenticatedUserContextKey, testUser)
		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		user := c.Locals(AuthenticatedUserContextKey)
		authenticatedUser, ok := user.(*utils.AuthenticatedUser)
		assert.True(t, ok)
		assert.Equal(t, "test123", authenticatedUser.Sub)

		return c.JSON(fiber.Map{"sub": authenticatedUser.Sub})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestAuthMiddleware_IntegrationWithActualHandler(t *testing.T) {
	// This test simulates a real-world scenario where a protected endpoint
	// needs to access the authenticated user

	app := fiber.New()

	// Configure auth middleware
	config := AuthConfig{
		SkipWellKnown: true,
		ResourceID:    "api://test-resource",
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			if token == "admin-token" {
				return &utils.AuthenticatedUser{
					Sub:    "admin123",
					Roles:  []string{"admin"},
					Scopes: []string{"read", "write", "delete"},
				}, nil
			} else if token == "user-token" {
				return &utils.AuthenticatedUser{
					Sub:    "user456",
					Roles:  []string{"user"},
					Scopes: []string{"read"},
				}, nil
			}
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		},
	}

	app.Use(OauthAuthMiddleware(config))

	// Protected endpoint that requires admin role
	app.Delete("/admin/users/:id", func(c *fiber.Ctx) error {
		// Get authenticated user from context
		userInterface := c.Locals(AuthenticatedUserContextKey)
		if userInterface == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "No authenticated user found",
			})
		}

		user, ok := userInterface.(*utils.AuthenticatedUser)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Invalid user type in context",
			})
		}

		// Check if user has admin role
		hasAdminRole := false
		for _, role := range user.Roles {
			if role == "admin" {
				hasAdminRole = true
				break
			}
		}

		if !hasAdminRole {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Insufficient privileges",
			})
		}

		userID := c.Params("id")
		return c.JSON(fiber.Map{
			"message":      fmt.Sprintf("User %s deleted by admin %s", userID, user.Sub),
			"deleted_by":   user.Sub,
			"admin_roles":  user.Roles,
			"admin_scopes": user.Scopes,
		})
	})

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Admin token - should allow deletion",
			token:          "admin-token",
			expectedStatus: fiber.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "deleted by admin admin123")
				assert.Contains(t, string(body), "admin")
			},
		},
		{
			name:           "User token - should deny deletion",
			token:          "user-token",
			expectedStatus: fiber.StatusForbidden,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "Insufficient privileges")
			},
		},
		{
			name:           "No token - should deny access",
			token:          "",
			expectedStatus: fiber.StatusUnauthorized,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "Missing or invalid Bearer token")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", "/admin/users/123", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}
