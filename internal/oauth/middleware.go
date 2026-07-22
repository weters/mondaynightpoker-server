package oauth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	jwtgo "github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type for context keys defined in this package.
type contextKey string

// playerIDContextKey stores the authenticated player ID on the request context.
const playerIDContextKey contextKey = "oauth_player_id"

// PlayerIDFromContext returns the authenticated player ID stashed by RequireMCPAuth.
func PlayerIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(playerIDContextKey).(int64)
	return id, ok
}

// RequireMCPAuth wraps next with bearer-token authentication for the MCP resource. It
// verifies the RS256 signature, issuer, audience, expiry, and token_use claim, then does
// a live site-admin check before invoking next.
func (s *Server) RequireMCPAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		playerID, ok := s.authenticateBearer(r)
		if !ok {
			s.writeInvalidToken(w)
			return
		}

		ctx := context.WithValue(r.Context(), playerIDContextKey, playerID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// authenticateBearer validates the Authorization bearer token and returns the player ID.
func (s *Server) authenticateBearer(r *http.Request) (int64, bool) {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return 0, false
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
		return 0, false
	}

	claims, ok := token.Claims.(jwtgo.MapClaims)
	if !ok {
		return 0, false
	}

	if tokenUse, _ := claims["token_use"].(string); tokenUse != tokenUseMCPAccess {
		return 0, false
	}

	sub, _ := claims["sub"].(string)
	playerID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return 0, false
	}

	// Live authorization check: the player must still be a site admin.
	player, err := s.repos.Players.GetPlayerByID(r.Context(), playerID)
	if err != nil || !player.IsSiteAdmin {
		return 0, false
	}

	return playerID, true
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
