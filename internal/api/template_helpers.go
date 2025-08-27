package api

import (
	"encoding/json"
	"html/template"
)

// GetTemplateFuncs returns the common template functions used across the application
func GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"json": func(v interface{}) (template.HTMLAttr, error) {
			bytes, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			// Return as HTMLAttr to properly escape for HTML attribute context
			return template.HTMLAttr(bytes), nil
		},
	}
}
