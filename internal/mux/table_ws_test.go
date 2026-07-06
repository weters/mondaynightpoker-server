package mux

import (
	"net/http"
	"os"
	"testing"

	"mondaynightpoker-server/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_checkOrigin(t *testing.T) {
	// restore the config singleton after the test (runs after t.Setenv restores the env)
	t.Cleanup(func() {
		_ = config.Load()
	})

	t.Setenv("MNP_ALLOWED_ORIGINS", "https://example.com,http://localhost:8080")
	require.NoError(t, config.Load())

	testCases := []struct {
		name    string
		origin  string
		allowed bool
	}{
		{name: "no origin (non-browser client)", origin: "", allowed: true},
		{name: "exact match", origin: "https://example.com", allowed: true},
		{name: "case-insensitive host", origin: "https://EXAMPLE.com", allowed: true},
		{name: "second allowed origin", origin: "http://localhost:8080", allowed: true},
		{name: "scheme mismatch", origin: "http://example.com", allowed: false},
		{name: "port mismatch", origin: "http://localhost:9090", allowed: false},
		{name: "disallowed host", origin: "https://evil.example.net", allowed: false},
		{name: "unparseable origin", origin: "http://exa mple.com", allowed: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "/table/uuid/ws", nil)
			require.NoError(t, err)
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}

			assert.Equal(t, tc.allowed, checkOrigin(r))
		})
	}
}

func Test_checkOrigin_fallsBackToHost(t *testing.T) {
	t.Cleanup(func() {
		_ = config.Load()
	})

	if orig, ok := os.LookupEnv("MNP_ALLOWED_ORIGINS"); ok {
		_ = os.Unsetenv("MNP_ALLOWED_ORIGINS")
		t.Cleanup(func() {
			_ = os.Setenv("MNP_ALLOWED_ORIGINS", orig)
		})
	}

	t.Setenv("MNP_HOST", "https://mondaynight.bid")
	require.NoError(t, config.Load())

	r, err := http.NewRequest(http.MethodGet, "/table/uuid/ws", nil)
	require.NoError(t, err)

	r.Header.Set("Origin", "https://mondaynight.bid")
	assert.True(t, checkOrigin(r))

	r.Header.Set("Origin", "https://other.example.com")
	assert.False(t, checkOrigin(r))
}
