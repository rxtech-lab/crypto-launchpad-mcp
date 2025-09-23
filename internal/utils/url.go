package utils

import (
	"fmt"
	"net/url"
	"os"
)

func GetTransactionSessionUrl(serverPort int, sessionId string) (string, error) {

	// Override baseUrl if BASE_URL env var is set
	if os.Getenv("BASE_URL") != "" {
		baseUrl := os.Getenv("BASE_URL")
		parsedUrl, err := url.Parse(baseUrl)
		if err != nil {
			return "", fmt.Errorf("invalid BASE_URL env var: %w", err)
		}
		parsedUrl.Path = fmt.Sprintf("/tx/%s", sessionId)
		return parsedUrl.String(), nil
	}

	url := fmt.Sprintf("http://localhost:%d/tx/%s", serverPort, sessionId)
	return url, nil
}
