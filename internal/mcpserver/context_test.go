package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mondaynightpoker-server/internal/oauth"
)

// probeOutput is the output of the context-propagation probe tool.
type probeOutput struct {
	PlayerID  int64 `json:"playerId"`
	IsAdmin   bool  `json:"isAdmin"`
	HasCaller bool  `json:"hasCaller"`
}

// TestContextPropagation empirically verifies that the go-sdk streamable HTTP handler
// (stateless mode) propagates the HTTP request context into tool handlers, so that a
// Caller stashed by outer middleware is visible via oauth.CallerFromContext inside a
// tool. If this ever regresses, per-player scoping would silently break.
func TestContextPropagation(t *testing.T) {
	m := mcp.NewServer(&mcp.Implementation{Name: "probe", Version: "test"}, nil)
	mcp.AddTool(m, &mcp.Tool{Name: "probe", Description: "echoes the stashed caller"},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, probeOutput, error) {
			c, ok := oauth.CallerFromContext(ctx)
			return nil, probeOutput{PlayerID: c.PlayerID, IsAdmin: c.IsSiteAdmin, HasCaller: ok}, nil
		})

	inner := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return m },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)

	// Outer middleware stashes a known Caller, mirroring RequireMCPAuth.
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := oauth.ContextWithCaller(r.Context(), oauth.Caller{PlayerID: 4242, IsSiteAdmin: true})
		inner.ServeHTTP(w, r.WithContext(ctx))
	})

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"probe","arguments":{}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	payload := rr.Body.String()
	assert.Contains(t, payload, `"hasCaller":true`)
	assert.Contains(t, payload, `"playerId":4242`)
	assert.Contains(t, payload, `"isAdmin":true`)
}
