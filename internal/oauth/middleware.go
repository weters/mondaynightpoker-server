package oauth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"mondaynightpoker-server/pkg/model"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type for context keys defined in this package.
type contextKey string

// callerContextKey stores the authenticated Caller on the request context.
const callerContextKey contextKey = "oauth_caller"

// Caller identifies the authenticated player making an MCP request.
type Caller struct {
	PlayerID    int64
	IsSiteAdmin bool // live value from the DB, not the token claim
}

// CallerFromContext returns the authenticated MCP caller, if any.
func CallerFromContext(ctx context.Context) (Caller, bool) {
	c, ok := ctx.Value(callerContextKey).(Caller)
	return c, ok
}

// ContextWithCaller returns a copy of ctx carrying the given Caller. RequireMCPAuth
// uses it to stash the authenticated caller; it is exported so tests and embedding
// packages can construct an authenticated context without driving the full token flow.
func ContextWithCaller(ctx context.Context, c Caller) context.Context {
	return context.WithValue(ctx, callerContextKey, c)
}

// PlayerIDFromContext returns the authenticated player ID stashed by RequireMCPAuth.
func PlayerIDFromContext(ctx context.Context) (int64, bool) {
	c, ok := CallerFromContext(ctx)
	return c.PlayerID, ok
}

// RequireMCPAuth wraps next with bearer-token authentication for the MCP resource. It
// verifies the RS256 signature, issuer, audience, expiry, and token_use claim, does a
// live load of the player, and stashes the caller identity on the request context.
func (s *Server) RequireMCPAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		caller, ok := s.authenticateBearer(r)
		if !ok {
			s.writeInvalidToken(w)
			return
		}

		ctx := ContextWithCaller(r.Context(), caller)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateBearer validates the Authorization bearer token and returns the caller.
func (s *Server) authenticateBearer(r *http.Request) (Caller, bool) {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return Caller{}, false
	}

	raw := strings.TrimSpace(auth[len(prefix):])

	token, err := jwtgo.Parse(raw, func(t *jwtgo.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtgo.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.pub, nil
	},
		jwtgo.WithValidMethods([]string{"RS256"}),
		jwtgo.WithIssuer(s.cfg.Issuer),
		jwtgo.WithAudience(s.cfg.Resource),
		jwtgo.WithExpirationRequired(),
	)
	if err != nil || !token.Valid {
		return Caller{}, false
	}

	claims, ok := token.Claims.(jwtgo.MapClaims)
	if !ok {
		return Caller{}, false
	}

	if tokenUse, _ := claims["token_use"].(string); tokenUse != tokenUseMCPAccess {
		return Caller{}, false
	}

	sub, _ := claims["sub"].(string)
	playerID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return Caller{}, false
	}

	// Live authorization check: the player must still exist and not be deleted. The
	// site-admin flag is read live so per-tool scoping never trusts the token claim.
	player, err := s.repos.Players.GetPlayerByID(r.Context(), playerID)
	if err != nil || player.Status == model.PlayerStatusDeleted {
		return Caller{}, false
	}

	return Caller{PlayerID: playerID, IsSiteAdmin: player.IsSiteAdmin}, true
}

// writeInvalidToken writes a 401 with an RFC 6750 WWW-Authenticate challenge pointing at
// the protected-resource metadata.
func (s *Server) writeInvalidToken(w http.ResponseWriter) {
	challenge := fmt.Sprintf(
		`Bearer resource_metadata="%s/.well-known/oauth-protected-resource", error="invalid_token"`,
		s.cfg.Issuer,
	)
	w.Header().Set("WWW-Authenticate", challenge)
	writeOAuthError(w, http.StatusUnauthorized, "invalid_token", "the access token is missing, invalid, or expired")
}
