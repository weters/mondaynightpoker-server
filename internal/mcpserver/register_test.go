package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"mondaynightpoker-server/internal/oauth"
)

// spyHandler records whether it ran and echoes the caller/input it received.
type spyHandler[In any] struct {
	ran    bool
	caller oauth.Caller
	in     In
}

func (h *spyHandler[In]) handle(_ context.Context, _ *mcp.CallToolRequest, caller oauth.Caller, in In) (struct{}, error) {
	h.ran = true
	h.caller = caller
	h.in = in
	return struct{}{}, nil
}

// -------------------- wrapper enforcement --------------------

func TestWrapHandler_Unauthenticated(t *testing.T) {
	a := assert.New(t)

	spy := &spyHandler[getPlayerInput]{}
	h := wrapHandler[getPlayerInput, struct{}](accessAuthenticated, spy.handle)

	// A context with no Caller (should never happen behind RequireMCPAuth) is rejected.
	_, _, err := h(cbg, nil, getPlayerInput{ID: 1})
	a.ErrorIs(err, errUnauthenticated)
	a.False(spy.ran, "handler must not run for an unauthenticated caller")
}

func TestWrapHandler_AdminOnly(t *testing.T) {
	a := assert.New(t)

	// non-admin is rejected before the handler runs
	spy := &spyHandler[getPlayerInput]{}
	h := wrapHandler[getPlayerInput, struct{}](accessAdminOnly, spy.handle)

	_, _, err := h(ctxForPlayer(7), nil, getPlayerInput{ID: 7})
	a.ErrorIs(err, errRequiresAdmin)
	a.False(spy.ran, "handler must not run for a non-admin under accessAdminOnly")

	// admin is allowed through
	spy = &spyHandler[getPlayerInput]{}
	h = wrapHandler[getPlayerInput, struct{}](accessAdminOnly, spy.handle)

	_, _, err = h(ctxForAdmin(7), nil, getPlayerInput{ID: 7})
	a.NoError(err)
	a.True(spy.ran)
	a.True(spy.caller.IsSiteAdmin)
}

func TestWrapHandler_SelfScoped(t *testing.T) {
	a := assert.New(t)

	// self: a non-admin targeting their own id is allowed
	spy := &spyHandler[getPlayerInput]{}
	h := wrapHandler[getPlayerInput, struct{}](accessSelfScoped, spy.handle)
	_, _, err := h(ctxForPlayer(5), nil, getPlayerInput{ID: 5})
	a.NoError(err)
	a.True(spy.ran)

	// other: a non-admin targeting a different id is denied before the handler runs
	spy = &spyHandler[getPlayerInput]{}
	h = wrapHandler[getPlayerInput, struct{}](accessSelfScoped, spy.handle)
	_, _, err = h(ctxForPlayer(5), nil, getPlayerInput{ID: 6})
	a.ErrorIs(err, errPermissionDenied)
	a.False(spy.ran, "handler must not run when a non-admin targets another player")

	// non-existent target id is denied identically (no existence leak)
	spy = &spyHandler[getPlayerInput]{}
	h = wrapHandler[getPlayerInput, struct{}](accessSelfScoped, spy.handle)
	_, _, err = h(ctxForPlayer(5), nil, getPlayerInput{ID: -999})
	a.ErrorIs(err, errPermissionDenied)
	a.False(spy.ran)

	// admin: may target any id
	spy = &spyHandler[getPlayerInput]{}
	h = wrapHandler[getPlayerInput, struct{}](accessSelfScoped, spy.handle)
	_, _, err = h(ctxForAdmin(5), nil, getPlayerInput{ID: 6})
	a.NoError(err)
	a.True(spy.ran)
}

func TestWrapHandler_Authenticated(t *testing.T) {
	a := assert.New(t)

	spy := &spyHandler[getPlayerInput]{}
	h := wrapHandler[getPlayerInput, struct{}](accessAuthenticated, spy.handle)

	_, _, err := h(ctxForPlayer(9), nil, getPlayerInput{ID: 123})
	a.NoError(err)
	a.True(spy.ran)
	a.Equal(int64(9), spy.caller.PlayerID)
}

// -------------------- registration panics (programmer errors) --------------------

func TestWrapHandler_PanicsOnInvalidPolicy(t *testing.T) {
	spy := &spyHandler[getPlayerInput]{}
	// The zero value (accessPolicy(0)) is invalid on purpose.
	assert.Panics(t, func() {
		wrapHandler[getPlayerInput, struct{}](accessPolicy(0), spy.handle)
	})
	// An out-of-range policy also panics.
	assert.Panics(t, func() {
		wrapHandler[getPlayerInput, struct{}](accessPolicy(99), spy.handle)
	})
}

func TestWrapHandler_PanicsOnSelfScopedWithoutInterface(t *testing.T) {
	// getTableInput does not implement targetScoped, so declaring it accessSelfScoped is a
	// programmer error caught at registration time.
	spy := &spyHandler[getTableInput]{}
	assert.Panics(t, func() {
		wrapHandler[getTableInput, struct{}](accessSelfScoped, spy.handle)
	})
}

func TestRegisterTool_PanicsPropagate(t *testing.T) {
	s := newServer()
	m := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)

	// registerTool delegates validation to wrapHandler, so the same programmer errors panic
	// at registration and nothing is recorded in the registry.
	assert.Panics(t, func() {
		registerTool[getTableInput, struct{}](s, m, &mcp.Tool{Name: "bad"}, accessSelfScoped,
			func(context.Context, *mcp.CallToolRequest, oauth.Caller, getTableInput) (struct{}, error) {
				return struct{}{}, nil
			})
	})
	assert.NotContains(t, s.registry, "bad")
}

// -------------------- exhaustiveness tripwire --------------------

// expectedPolicies is the authoritative tool -> policy table. Every tool the server exposes
// must appear here with the correct policy. When adding a new tool, classify it here AND add
// behavioral tests for its policy; otherwise this test fails.
var expectedPolicies = map[string]accessPolicy{
	"whoami":                   accessAuthenticated,
	"list_players":             accessAdminOnly,
	"get_player":               accessSelfScoped,
	"get_player_by_email":      accessAdminOnly,
	"get_player_stats":         accessSelfScoped,
	"get_player_profile":       accessSelfScoped,
	"list_player_tables":       accessSelfScoped,
	"list_tables":              accessAuthenticated,
	"get_table":                accessAuthenticated,
	"get_table_roster":         accessAuthenticated,
	"list_table_games":         accessAuthenticated,
	"get_game":                 accessAuthenticated,
	"get_table_stats":          accessAuthenticated,
	"list_player_transactions": accessSelfScoped,
	"leaderboard":              accessAuthenticated,
	"list_game_types":          accessAuthenticated,
	"get_hand_history":         accessAuthenticated,
	"get_player_tendencies":    accessSelfScoped,
	"get_player_variance":      accessSelfScoped,
}

func TestRegistry_MatchesExpectedPolicies(t *testing.T) {
	s := newServer()
	m := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	s.registerTools(m)

	// (b) the live registry populated by registerTool must match the table EXACTLY.
	for name, want := range expectedPolicies {
		got, ok := s.registry[name]
		if !ok {
			t.Errorf("tool %q is expected but was not registered; if it was removed, delete it from expectedPolicies", name)
			continue
		}
		if got != want {
			t.Errorf("tool %q registered with policy %d, expected %d; reclassify it and update its behavioral tests", name, got, want)
		}
	}
	for name := range s.registry {
		if _, ok := expectedPolicies[name]; !ok {
			t.Errorf("tool %q was registered but is not classified in expectedPolicies; classify the new tool AND add behavioral tests for its policy", name)
		}
	}
}

func TestRegistry_ToolsListMatchesTable(t *testing.T) {
	s := newServer()
	m := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "test"}, nil)
	s.registerTools(m)

	// (c) drive tools/list on the real MCP server so a tool added via raw mcp.AddTool (which
	// never touches the registry) would still be caught.
	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ss, err := m.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)
	defer ss.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	cs, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	res, err := cs.ListTools(ctx, nil)
	require.NoError(t, err)

	advertised := make(map[string]struct{}, len(res.Tools))
	for _, tool := range res.Tools {
		advertised[tool.Name] = struct{}{}
	}

	assert.Len(t, advertised, len(expectedPolicies))
	for name := range expectedPolicies {
		_, ok := advertised[name]
		assert.Truef(t, ok, "tool %q is in the expected table but not advertised via tools/list", name)
	}
	for name := range advertised {
		_, ok := expectedPolicies[name]
		assert.Truef(t, ok, "tool %q is advertised via tools/list but not classified (did it bypass registerTool?)", name)
	}
}
