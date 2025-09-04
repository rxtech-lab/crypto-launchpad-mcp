package utils

import (
	"fmt"
	"os"
	"strconv"
)

func GetTransactionSessionUrl(serverPort int, sessionId string) (string, error) {
	hostName, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}

	// Override baseUrl if BASE_URL env var is set
	if os.Getenv("BASE_URL") != "" {
		baseUrl := os.Getenv("BASE_URL")
		port, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return "", fmt.Errorf("invalid PORT env var: %w", err)
		}
		return fmt.Sprintf("%s/tx/%d/%s", baseUrl, port, sessionId), nil
	}

	url := fmt.Sprintf("http://%s:%d/tx/%s", hostName, serverPort, sessionId)
	return url, nil
}
