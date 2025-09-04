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
				assert.Equal(t, "https://api.example.com/tx/3000/test-session-123", url)
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
			name:        "with BASE_URL but invalid PORT",
			serverPort:  8080,
			sessionId:   "test-session-789",
			baseUrl:     "https://api.example.com",
			port:        "invalid-port",
			expectError: true,
			setup: func() {
				os.Setenv("BASE_URL", "https://api.example.com")
				os.Setenv("PORT", "invalid-port")
			},
			cleanup: func() {
				os.Unsetenv("BASE_URL")
				os.Unsetenv("PORT")
			},
			validate: func(t *testing.T, url string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid PORT env var")
				assert.Equal(t, "", url)
			},
		},
		{
			name:        "with BASE_URL but empty PORT",
			serverPort:  8080,
			sessionId:   "test-session-empty-port",
			baseUrl:     "https://api.example.com",
			port:        "",
			expectError: true,
			setup: func() {
				os.Setenv("BASE_URL", "https://api.example.com")
				os.Setenv("PORT", "")
			},
			cleanup: func() {
				os.Unsetenv("BASE_URL")
				os.Unsetenv("PORT")
			},
			validate: func(t *testing.T, url string, err error) {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid PORT env var")
				assert.Equal(t, "", url)
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
				assert.Equal(t, "https://api.example.com//tx/4000/test-session-trailing", url)
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

func TestGetTransactionSessionUrl_EdgeCases(t *testing.T) {
	t.Run("zero port number", func(t *testing.T) {
		os.Unsetenv("BASE_URL")
		os.Unsetenv("PORT")

		url, err := GetTransactionSessionUrl(0, "test-session")
		require.NoError(t, err)

		hostname, hostnameErr := os.Hostname()
		require.NoError(t, hostnameErr)
		expected := "http://" + hostname + ":0/tx/test-session"
		assert.Equal(t, expected, url)
	})

	t.Run("empty session ID", func(t *testing.T) {
		os.Unsetenv("BASE_URL")
		os.Unsetenv("PORT")

		url, err := GetTransactionSessionUrl(8080, "")
		require.NoError(t, err)

		hostname, hostnameErr := os.Hostname()
		require.NoError(t, hostnameErr)
		expected := "http://" + hostname + ":8080/tx/"
		assert.Equal(t, expected, url)
	})

	t.Run("negative port number", func(t *testing.T) {
		os.Unsetenv("BASE_URL")
		os.Unsetenv("PORT")

		url, err := GetTransactionSessionUrl(-1, "test-session")
		require.NoError(t, err)

		hostname, hostnameErr := os.Hostname()
		require.NoError(t, hostnameErr)
		expected := "http://" + hostname + ":-1/tx/test-session"
		assert.Equal(t, expected, url)
	})

	t.Run("BASE_URL set but PORT not set", func(t *testing.T) {
		os.Setenv("BASE_URL", "https://api.example.com")
		os.Unsetenv("PORT")
		defer func() {
			os.Unsetenv("BASE_URL")
		}()

		url, err := GetTransactionSessionUrl(8080, "test-session")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PORT env var")
		assert.Equal(t, "", url)
	})
}
