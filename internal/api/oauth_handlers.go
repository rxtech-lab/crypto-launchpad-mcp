package api

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
)

func (s *APIServer) handleOAuthProtectedResource(c *fiber.Ctx) error {
	oauthContent := map[string]any{
		"authorization_servers":    []string{"https://mcp-ae4lqgxzaaaqw.scalekit.dev/resources/res_88196233813820681"},
		"bearer_methods_supported": []string{"header"},
		"resource":                 "https://launchpad.mcprouter.app",
		"resource_documentation":   "https://launchpad.mcprouter.app/docs",
		"scopes_supported":         []string{},
	}

	jsonContent, err := json.Marshal(oauthContent)
	if err != nil {
		return c.Status(500).SendString("Failed to marshal OAuth content")
	}

	return c.SendString(string(jsonContent))
}
