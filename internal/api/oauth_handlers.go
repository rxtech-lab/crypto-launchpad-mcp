package api

import (
	"encoding/json"
	"os"

	"github.com/gofiber/fiber/v2"
)

func (s *APIServer) handleOAuthProtectedResource(c *fiber.Ctx) error {
	authenticationUrl := os.Getenv("OAUTH_AUTHENTICATION_SERVER")
	authenticationResourceUrl := os.Getenv("OAUTH_RESOURCE_URL")
	authenticationDocumentationUrl := os.Getenv("OAUTH_RESOURCE_DOCUMENTATION_URL")

	oauthContent := map[string]any{
		"authorization_servers":    []string{authenticationUrl},
		"bearer_methods_supported": []string{"header"},
		"resource":                 authenticationResourceUrl,
		"resource_documentation":   authenticationDocumentationUrl,
		"scopes_supported":         []string{},
	}

	jsonContent, err := json.Marshal(oauthContent)
	if err != nil {
		return c.Status(500).SendString("Failed to marshal OAuth content")
	}

	return c.SendString(string(jsonContent))
}
