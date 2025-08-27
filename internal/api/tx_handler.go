package api

import (
	"bytes"
	"log"
	"text/template"

	"github.com/gofiber/fiber/v2"
	"github.com/rxtech-lab/launchpad-mcp/internal/assets"
)

// handleTransactionPage serves the universal transaction signing page
func (s *APIServer) handleTransactionPage(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	// Get the session from database
	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		log.Printf("Error getting session %s: %v", sessionID, err)
		// Still serve the page but let the React app handle the error
	}

	// Prepare template data
	data := map[string]interface{}{
		"SessionID": sessionID,
	}

	// Only add SessionData if session exists
	if session != nil {
		data["SessionData"] = session
	}

	// Render the template with custom functions
	tmplBytes := assets.SigningHTML
	tmpl, err := template.New("signing").Funcs(GetTemplateFuncs()).Parse(string(tmplBytes))
	if err != nil {
		log.Printf("Error parsing signing template: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error parsing template")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Error rendering signing template: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error rendering template")
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.Send(buf.Bytes())
}

// handleTransactionAPI provides transaction data via API
func (s *APIServer) handleTransactionAPI(c *fiber.Ctx) error {
	sessionID := c.Params("session_id")

	// Get the session from database
	session, err := s.db.GetTransactionSession(sessionID)
	if err != nil {
		log.Printf("Error getting session %s: %v", sessionID, err)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Session not found",
		})
	}

	// Return the session data as JSON
	return c.JSON(session)
}
