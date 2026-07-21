package mcpserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mondaynightpoker-server/internal/oauth"
)

// Authorization errors surfaced as tool errors. They are intentionally vague so a
// non-admin caller cannot distinguish "forbidden" from "does not exist". The exact
// wording of these errors is asserted (via Contains) by internal/mux/mcp_test.go and must
// not change.
var (
	errUnauthenticated  = errors.New("unauthenticated")
	errRequiresAdmin    = errors.New("requires site admin")
	errPermissionDenied = errors.New("permission denied")
)

// accessPolicy declares who may call a tool. It is enforced structurally by registerTool
// before any handler runs, so authorization is a property of registration rather than
// per-handler discipline. The zero value is invalid on purpose: a tool must always declare
// a policy, and registerTool panics if it does not.
type accessPolicy int

const (
	// accessAdminOnly restricts a tool to site admins.
	accessAdminOnly accessPolicy = iota + 1
	// accessSelfScoped restricts a non-admin caller to targeting only their own player id;
	// admins may target any player. The tool's input type must implement targetScoped.
	accessSelfScoped
	// accessAuthenticated allows any authenticated player.
	accessAuthenticated
)

// targetScoped is implemented by the input types of accessSelfScoped tools. targetPlayerID
// returns the player id the request targets, which the wrapper compares against the caller.
type targetScoped interface {
	targetPlayerID() int64
}

// toolHandler is the shape of a policy-wrapped tool handler: it receives the
// already-authorized caller and performs zero authorization checks of its own.
type toolHandler[In, Out any] func(ctx context.Context, req *mcp.CallToolRequest, caller oauth.Caller, in In) (Out, error)

// registerTool installs a tool on the MCP server behind a mandatory access policy. It is the
// only way tools are added to the server: the policy is enforced in a wrapper (built by
// wrapHandler) that runs before the handler. The declared policy is recorded in the server's
// registry so an exhaustiveness test can verify every tool is classified.
//
// It fails fast at registration time (panics) for programmer errors: an unset/unknown policy
// or an accessSelfScoped tool whose input type does not implement targetScoped. Both are
// caught at server construction (startup), never in production traffic.
func registerTool[In, Out any](
	s *server,
	m *mcp.Server,
	tool *mcp.Tool,
	policy accessPolicy,
	handler toolHandler[In, Out],
) {
	// wrapHandler validates the policy and (for accessSelfScoped) the input type, panicking
	// on a programmer error before anything is registered.
	wrapped := wrapHandler(policy, handler)
	s.registry[tool.Name] = policy
	mcp.AddTool(m, tool, wrapped)
}

// wrapHandler turns a policy plus a caller-aware handler into an mcp.ToolHandlerFor that
// enforces the policy before the handler runs. It is the single structural chokepoint for
// authorization: the returned handler resolves the caller once, applies the policy, and only
// then invokes the wrapped handler with the authorized caller.
//
// It panics (a startup-time programmer error) if the policy is unset/unknown, or if the
// policy is accessSelfScoped but In does not implement targetScoped.
func wrapHandler[In, Out any](policy accessPolicy, handler toolHandler[In, Out]) mcp.ToolHandlerFor[In, Out] {
	switch policy {
	case accessAdminOnly, accessAuthenticated:
		// valid; no per-input requirement
	case accessSelfScoped:
		var zero In
		if _, ok := any(zero).(targetScoped); !ok {
			panic(fmt.Sprintf("wrapHandler: accessSelfScoped requires input type %T to implement targetScoped", zero))
		}
	default:
		panic(fmt.Sprintf("wrapHandler: invalid access policy %d", policy))
	}

	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var zeroOut Out

		caller, ok := oauth.CallerFromContext(ctx)
		if !ok {
			// Deny by default: a missing caller should never happen behind RequireMCPAuth,
			// so erroring here is defense in depth.
			return nil, zeroOut, errUnauthenticated
		}

		switch policy {
		case accessAdminOnly:
			if !caller.IsSiteAdmin {
				return nil, zeroOut, errRequiresAdmin
			}
		case accessSelfScoped:
			// Safe: validated above that In implements targetScoped.
			target := any(in).(targetScoped).targetPlayerID()
			if !caller.IsSiteAdmin && target != caller.PlayerID {
				return nil, zeroOut, errPermissionDenied
			}
		case accessAuthenticated:
			// Authentication alone suffices.
		}

		out, err := handler(ctx, req, caller, in)
		return nil, out, err
	}
}
