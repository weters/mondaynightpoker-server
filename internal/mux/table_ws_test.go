package mux

import (
	"net/http"
	"testing"

	"mondaynightpoker-server/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_checkOrigin(t *testing.T) {
	m := &Mux{cfg: config.Config{
		Host:           "https://mondaynight.bid",
		AllowedOrigins: []string{"https://example.com", "http://localhost:8080"},
	}}

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

			assert.Equal(t, tc.allowed, m.checkOrigin(r))
		})
	}
}

func Test_checkOrigin_fallsBackToHost(t *testing.T) {
	// with no AllowedOrigins configured, only Host is allowed
	m := &Mux{cfg: config.Config{Host: "https://mondaynight.bid"}}

	r, err := http.NewRequest(http.MethodGet, "/table/uuid/ws", nil)
	require.NoError(t, err)

	r.Header.Set("Origin", "https://mondaynight.bid")
	assert.True(t, m.checkOrigin(r))

	r.Header.Set("Origin", "https://other.example.com")
	assert.False(t, m.checkOrigin(r))
}
