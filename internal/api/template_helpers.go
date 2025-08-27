package api

import (
	"encoding/json"
	"text/template"
)

// GetTemplateFuncs returns the common template functions used across the application
func GetTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"json": func(v interface{}) (string, error) {
			bytes, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(bytes), nil
		},
	}
}
