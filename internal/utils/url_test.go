package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransactionSessionUrl(t *testing.T) {
	tests := []struct {
		name        string
		serverPort  int
		sessionId   string
		baseUrl     string
		port        string
		expectError bool
		setup       func()
		cleanup     func()
		validate    func(t *testing.T, url string, err error)
	}{
		{
			name:       "with BASE_URL and PORT env vars",
			serverPort: 8080,
			sessionId:  "test-session-123",
			baseUrl:    "https://api.example.com",
			port:       "3000",
			setup: func() {
				os.Setenv("BASE_URL", "https://api.example.com")
				os.Setenv("PORT", "3000")
			},
			cleanup: func() {
				os.Unsetenv("BASE_URL")
				os.Unsetenv("PORT")
			},
			validate: func(t *testing.T, url string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "https://api.example.com/tx/test-session-123", url)
			},
		},
		{
			name:       "without BASE_URL env var",
			serverPort: 9000,
			sessionId:  "session-456",
			setup: func() {
				os.Unsetenv("BASE_URL")
				os.Unsetenv("PORT")
			},
			cleanup: func() {},
			validate: func(t *testing.T, url string, err error) {
				expected := "http://" + "localhost" + ":9000/tx/session-456"
				assert.Equal(t, expected, url)
			},
		},
		{
			name:       "with BASE_URL containing trailing slash",
			serverPort: 8080,
			sessionId:  "test-session-trailing",
			baseUrl:    "https://api.example.com/",
			port:       "4000",
			setup: func() {
				os.Setenv("BASE_URL", "https://api.example.com/")
				os.Setenv("PORT", "4000")
			},
			cleanup: func() {
				os.Unsetenv("BASE_URL")
				os.Unsetenv("PORT")
			},
			validate: func(t *testing.T, url string, err error) {
				require.NoError(t, err)
				assert.Equal(t, "https://api.example.com/tx/test-session-trailing", url)
			},
		},
		{
			name:       "with special characters in session ID",
			serverPort: 7000,
			sessionId:  "session-with-special-chars-!@#",
			setup: func() {
				os.Unsetenv("BASE_URL")
				os.Unsetenv("PORT")
			},
			cleanup: func() {},
			validate: func(t *testing.T, url string, err error) {
				expected := "http://" + "localhost" + ":7000/tx/session-with-special-chars-!@#"
				assert.Equal(t, expected, url)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			url, err := GetTransactionSessionUrl(tt.serverPort, tt.sessionId)
			tt.validate(t, url, err)
		})
	}
}
