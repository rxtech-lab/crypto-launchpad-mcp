package api

import (
	"bytes"
	"fmt"
	"log"
	"strings"
)

// renderTemplate renders a template with the given data
func (s *APIServer) renderTemplate(templateName string, data interface{}) string {
	tmpl, exists := s.templates[templateName]
	if !exists {
		log.Printf("Template %s not found", templateName)
		return fmt.Sprintf("Template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Error executing template %s: %v", templateName, err)
		return fmt.Sprintf("Error rendering template: %v", err)
	}

	return buf.String()
}

// getPageTitle returns a human-readable title for the given session type
func (s *APIServer) getPageTitle(sessionType string) string {
	switch sessionType {
	case "add_liquidity":
		return "Add Liquidity"
	case "remove_liquidity":
		return "Remove Liquidity"
	case "swap":
		return "Swap Tokens"
	default:
		// Capitalize first letter and replace underscores with spaces
		title := strings.ReplaceAll(sessionType, "_", " ")
		if len(title) > 0 {
			title = strings.ToUpper(string(title[0])) + title[1:]
		}
		return title
	}
}
