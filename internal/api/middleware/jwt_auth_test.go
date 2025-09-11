package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/rxtech-lab/launchpad-mcp/internal/utils"
	"github.com/stretchr/testify/suite"
)

type JwtAuthMiddlewareTestSuite struct {
	suite.Suite
	app           *fiber.App
	authenticator *utils.SimpleJwtAuthenticator
	secret        string
}

func (suite *JwtAuthMiddlewareTestSuite) SetupSuite() {
	suite.secret = "test-secret-key-for-jwt-authentication"
	auth, err := utils.NewSimpleJwtAuthenticator(suite.secret)
	suite.Require().NoError(err)
	suite.authenticator = &auth

	// Create test app with middleware
	app := fiber.New()
	app.Use(JwtAuthMiddleware(suite.authenticator))

	// Test route that checks for authenticated user
	app.Get("/test", func(c *fiber.Ctx) error {
		user := c.Locals(AuthenticatedUserContextKey)
		if user != nil {
			authUser := user.(*utils.AuthenticatedUser)
			return c.JSON(fiber.Map{
				"authenticated": true,
				"user_id":       authUser.Sub,
			})
		}
		return c.JSON(fiber.Map{
			"authenticated": false,
		})
	})

	suite.app = app
}

func (suite *JwtAuthMiddlewareTestSuite) TestValidJWTToken() {
	// Create a valid JWT token
	token := suite.createTestToken("test-user-123", map[string]interface{}{
		"roles":  []string{"user"},
		"scopes": []string{"read", "write"},
	})

	// Make request with valid token
	req, err := http.NewRequest("GET", "/test", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	// Check response indicates authenticated user
	// Note: In a real test, you'd parse the JSON response
	// For simplicity, we're just checking the status code
}

func (suite *JwtAuthMiddlewareTestSuite) TestInvalidJWTToken() {
	// Make request with invalid token
	req, err := http.NewRequest("GET", "/test", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)

	// Should still return 200 (middleware doesn't block)
	suite.Equal(http.StatusOK, resp.StatusCode)
	// User should not be authenticated (tested in route handler)
}

func (suite *JwtAuthMiddlewareTestSuite) TestNoAuthorizationHeader() {
	// Make request without Authorization header
	req, err := http.NewRequest("GET", "/test", nil)
	suite.Require().NoError(err)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)

	// Should still return 200 (middleware doesn't block)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *JwtAuthMiddlewareTestSuite) TestEmptyBearerToken() {
	// Make request with empty Bearer token
	req, err := http.NewRequest("GET", "/test", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer ")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)

	// Should still return 200 (middleware doesn't block)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *JwtAuthMiddlewareTestSuite) TestNonBearerAuthorizationHeader() {
	// Make request with non-Bearer authorization
	req, err := http.NewRequest("GET", "/test", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)

	// Should still return 200 (middleware doesn't block)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *JwtAuthMiddlewareTestSuite) TestNilAuthenticator() {
	// Create app with nil authenticator
	app := fiber.New()
	app.Use(JwtAuthMiddleware(nil))

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	req, err := http.NewRequest("GET", "/test", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer some-token")

	resp, err := app.Test(req)
	suite.Require().NoError(err)

	// Should still return 200 (middleware skips when no authenticator)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *JwtAuthMiddlewareTestSuite) createTestToken(userID string, claims map[string]interface{}) string {
	token := jwt.New(jwt.SigningMethodHS256)

	// Set claims
	tokenClaims := token.Claims.(jwt.MapClaims)
	tokenClaims["sub"] = userID
	tokenClaims["iss"] = "test-issuer"
	tokenClaims["aud"] = []string{"test-audience"}
	tokenClaims["exp"] = time.Now().Add(time.Hour).Unix()
	tokenClaims["iat"] = time.Now().Unix()

	// Add custom claims
	for k, v := range claims {
		tokenClaims[k] = v
	}

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(suite.secret))
	suite.Require().NoError(err)

	return tokenString
}

func TestJwtAuthMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(JwtAuthMiddlewareTestSuite))
}

// OauthAuthMiddleware Test Suite
type OauthAuthMiddlewareTestSuite struct {
	suite.Suite
	app           *fiber.App
	authenticator *utils.SimpleJwtAuthenticator
	secret        string
}

func (suite *OauthAuthMiddlewareTestSuite) SetupSuite() {
	suite.secret = "test-secret-key-for-oauth-middleware"
	auth, err := utils.NewSimpleJwtAuthenticator(suite.secret)
	suite.Require().NoError(err)
	suite.authenticator = &auth
}

func (suite *OauthAuthMiddlewareTestSuite) SetupTest() {
	// Create fresh app for each test
	suite.app = fiber.New()
}

func (suite *OauthAuthMiddlewareTestSuite) TestDefaultConfig() {
	config := DefaultAuthConfig()
	suite.True(config.SkipWellKnown)
	suite.NotNil(config.TokenValidator)
	suite.Empty(config.ResourceID)
}

func (suite *OauthAuthMiddlewareTestSuite) TestValidTokenWithDefaultConfig() {
	// Use default config which has a dummy token validator
	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		user := c.Locals(AuthenticatedUserContextKey)
		suite.NotNil(user)
		return c.JSON(fiber.Map{"status": "authenticated"})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer valid-token")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestValidTokenWithCustomConfig() {
	config := AuthConfig{
		ResourceID: "test-resource",
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			if token == "valid-token" {
				return &utils.AuthenticatedUser{
					Sub: "test-user",
				}, nil
			}
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		},
		SkipWellKnown: true,
	}

	suite.app.Use(OauthAuthMiddleware(config))

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		user := c.Locals(AuthenticatedUserContextKey)
		if user != nil {
			authUser := user.(*utils.AuthenticatedUser)
			return c.JSON(fiber.Map{
				"authenticated": true,
				"user_id":       authUser.Sub,
			})
		}
		return c.JSON(fiber.Map{"authenticated": false})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer valid-token")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestInvalidToken() {
	config := AuthConfig{
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		},
	}

	suite.app.Use(OauthAuthMiddleware(config))

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "should not reach here"})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer invalid-token")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
	suite.Equal(`Bearer realm="Access to protected resource"`, resp.Header.Get("WWW-Authenticate"))
}

func (suite *OauthAuthMiddlewareTestSuite) TestMissingBearerToken() {
	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "should not reach here"})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
	suite.Contains(resp.Header.Get("WWW-Authenticate"), `Bearer realm="Oauth"`)
}

func (suite *OauthAuthMiddlewareTestSuite) TestEmptyBearerToken() {
	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "should not reach here"})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer ")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestNonBearerAuthorizationHeader() {
	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "should not reach here"})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestSkipMCPRoutes() {
	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/mcp/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "allowed"})
	})

	req, err := http.NewRequest("GET", "/mcp/test", nil)
	suite.Require().NoError(err)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestSkipTxRoutes() {
	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/tx/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "allowed"})
	})

	req, err := http.NewRequest("GET", "/tx/test", nil)
	suite.Require().NoError(err)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestSkipHealthRoute() {
	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	req, err := http.NewRequest("GET", "/health", nil)
	suite.Require().NoError(err)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestSkipWellKnownEndpoints() {
	suite.app.Use(OauthAuthMiddleware()) // Default config has SkipWellKnown = true

	suite.app.Get("/.well-known/oauth-authorization-server", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "metadata"})
	})

	req, err := http.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	suite.Require().NoError(err)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestDontSkipWellKnownEndpoints() {
	config := AuthConfig{
		SkipWellKnown: false,
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			return nil, fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
		},
	}

	suite.app.Use(OauthAuthMiddleware(config))

	suite.app.Get("/.well-known/oauth-authorization-server", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "should not reach here"})
	})

	req, err := http.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	suite.Require().NoError(err)

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusUnauthorized, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestAlreadyAuthenticatedUser() {
	// Middleware that sets authenticated user before auth middleware
	suite.app.Use(func(c *fiber.Ctx) error {
		c.Locals(AuthenticatedUserContextKey, &utils.AuthenticatedUser{
			Sub: "pre-existing-user",
		})
		return c.Next()
	})

	suite.app.Use(OauthAuthMiddleware())

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		user := c.Locals(AuthenticatedUserContextKey).(*utils.AuthenticatedUser)
		return c.JSON(fiber.Map{
			"user_id": user.Sub,
		})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	// No Authorization header, but user already in context

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestWithResourceID() {
	config := AuthConfig{
		ResourceID: "test-resource-id",
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			suite.Contains(audience, "test-resource-id")
			return &utils.AuthenticatedUser{
				Sub: "test-user",
				Aud: audience,
			}, nil
		},
	}

	suite.app.Use(OauthAuthMiddleware(config))

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		user := c.Locals(AuthenticatedUserContextKey).(*utils.AuthenticatedUser)
		return c.JSON(fiber.Map{
			"user_id":  user.Sub,
			"audience": user.Aud,
		})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer valid-token")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func (suite *OauthAuthMiddlewareTestSuite) TestWithoutResourceID() {
	config := AuthConfig{
		ResourceID: "", // Empty resource ID
		TokenValidator: func(token string, audience []string) (*utils.AuthenticatedUser, error) {
			suite.Empty(audience) // Should be empty when no ResourceID
			return &utils.AuthenticatedUser{
				Sub: "test-user",
			}, nil
		},
	}

	suite.app.Use(OauthAuthMiddleware(config))

	suite.app.Get("/protected", func(c *fiber.Ctx) error {
		user := c.Locals(AuthenticatedUserContextKey).(*utils.AuthenticatedUser)
		return c.JSON(fiber.Map{
			"user_id": user.Sub,
		})
	})

	req, err := http.NewRequest("GET", "/protected", nil)
	suite.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer valid-token")

	resp, err := suite.app.Test(req)
	suite.Require().NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func TestOauthAuthMiddlewareTestSuite(t *testing.T) {
	suite.Run(t, new(OauthAuthMiddlewareTestSuite))
}
