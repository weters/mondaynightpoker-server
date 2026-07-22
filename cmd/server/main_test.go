package main

import "testing"

func Test_isOpenCORSPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/mcp", true},
		{"/mcp/", true},
		{"/mcp/anything", true},
		{"/oauth/token", true},
		{"/oauth/register", true},
		{"/.well-known/oauth-protected-resource", true},
		{"/.well-known/oauth-authorization-server", true},
		{"/oauth/authorize", false},
		{"/.well-known/other", false},
		{"/health", false},
		{"/player", false},
		{"/table", false},
		{"/mcpfoo", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isOpenCORSPath(tt.path); got != tt.want {
				t.Errorf("isOpenCORSPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
