package oauth

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"mondaynightpoker-server/pkg/model"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Token returns the handler for the token endpoint, dispatching on grant_type.
func (s *Server) Token() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Token responses must never be cached (RFC 6749 section 5.1).
		w.Header().Set("Cache-Control", "no-store")

		if err := r.ParseForm(); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_request", "malformed request body")
			return
		}

		switch r.FormValue("grant_type") {
		case "authorization_code":
			s.handleAuthCodeGrant(w, r)
		case "refresh_token":
			s.handleRefreshGrant(w, r)
		default:
			writeOAuthError(w, http.StatusBadRequest, "unsupported_grant_type", "")
		}
	}
}

// handleAuthCodeGrant exchanges an authorization code (with PKCE) for tokens.
func (s *Server) handleAuthCodeGrant(w http.ResponseWriter, r *http.Request) {
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	codeVerifier := r.FormValue("code_verifier")

	if code == "" || redirectURI == "" || clientID == "" || codeVerifier == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "missing required parameter")
		return
	}

	stored, err := s.repos.OAuth.ConsumeAuthCode(r.Context(), sha256Hex(code))
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "invalid or expired authorization code")
		return
	}

	if stored.ClientID != clientID || stored.RedirectURI != redirectURI {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "client or redirect URI mismatch")
		return
	}

	if !verifyPKCE(codeVerifier, stored.CodeChallenge) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}

	accessToken, err := s.newAccessToken(stored.PlayerID)
	if err != nil {
		logrus.WithError(err).Error("oauth: could not sign access token")
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	rawRefresh, err := randomToken(32)
	if err != nil {
		logrus.WithError(err).Error("oauth: could not generate refresh token")
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	refresh := &model.OAuthRefreshToken{
		TokenHash: sha256Hex(rawRefresh),
		ClientID:  stored.ClientID,
		PlayerID:  stored.PlayerID,
		Scope:     stored.Scope,
		Resource:  stored.Resource,
		Expires:   time.Now().Add(s.cfg.RefreshTokenTTL),
	}

	if err := s.repos.OAuth.CreateRefreshToken(r.Context(), refresh); err != nil {
		logrus.WithError(err).Error("oauth: could not persist refresh token")
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	s.writeTokenResponse(w, accessToken, rawRefresh, scopeOrDefault(stored.Scope))
}

// handleRefreshGrant rotates a refresh token and issues a fresh access token.
func (s *Server) handleRefreshGrant(w http.ResponseWriter, r *http.Request) {
	rawRefresh := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")

	if rawRefresh == "" || clientID == "" {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "missing required parameter")
		return
	}

	oldHash := sha256Hex(rawRefresh)
	stored, err := s.repos.OAuth.GetRefreshToken(r.Context(), oldHash)
	if err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "invalid refresh token")
		return
	}

	if stored.ClientID != clientID {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "client mismatch")
		return
	}

	// Presenting an already-revoked token indicates reuse: revoke the whole family.
	if stored.Revoked {
		s.revokeFamily(r.Context(), stored.PlayerID, stored.ClientID)
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token has been revoked")
		return
	}

	if time.Now().After(stored.Expires) {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "refresh token has expired")
		return
	}

	// Live authorization re-check: a demoted admin loses access immediately.
	player, err := s.repos.Players.GetPlayerByID(r.Context(), stored.PlayerID)
	if err != nil || !player.IsSiteAdmin {
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "player is no longer authorized")
		return
	}

	newRaw, err := randomToken(32)
	if err != nil {
		logrus.WithError(err).Error("oauth: could not generate refresh token")
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	newToken := &model.OAuthRefreshToken{
		TokenHash: sha256Hex(newRaw),
		ClientID:  stored.ClientID,
		PlayerID:  stored.PlayerID,
		Scope:     stored.Scope,
		Resource:  stored.Resource,
		Expires:   time.Now().Add(s.cfg.RefreshTokenTTL),
	}

	if err := s.repos.OAuth.RotateRefreshToken(r.Context(), oldHash, newToken); err != nil {
		if errors.Is(err, model.ErrRefreshTokenAlreadyRevoked) {
			// Lost the race against a concurrent reuse: revoke the family.
			s.revokeFamily(r.Context(), stored.PlayerID, stored.ClientID)
		} else {
			logrus.WithError(err).Error("oauth: could not rotate refresh token")
		}
		writeOAuthError(w, http.StatusBadRequest, "invalid_grant", "could not rotate refresh token")
		return
	}

	accessToken, err := s.newAccessToken(stored.PlayerID)
	if err != nil {
		logrus.WithError(err).Error("oauth: could not sign access token")
		writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
		return
	}

	s.writeTokenResponse(w, accessToken, newRaw, scopeOrDefault(stored.Scope))
}

// revokeFamily revokes every refresh token in a player/client family, logging on failure.
func (s *Server) revokeFamily(ctx context.Context, playerID int64, clientID string) {
	if err := s.repos.OAuth.RevokeRefreshTokenFamily(ctx, playerID, clientID); err != nil {
		logrus.WithError(err).Error("oauth: could not revoke refresh token family")
	}
}

// newAccessToken mints an RS256-signed MCP access token for the player.
func (s *Server) newAccessToken(playerID int64) (string, error) {
	now := time.Now()
	jti, err := randomToken(16)
	if err != nil {
		return "", err
	}

	claims := jwtgo.MapClaims{
		"iss":       s.cfg.Issuer,
		"sub":       strconv.FormatInt(playerID, 10),
		"aud":       s.cfg.Resource,
		"exp":       now.Add(s.cfg.AccessTokenTTL).Unix(),
		"iat":       now.Unix(),
		"jti":       jti,
		"scope":     scopeMCP,
		"token_use": tokenUseMCPAccess,
	}

	return jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims).SignedString(s.priv)
}

// writeTokenResponse writes a successful RFC 6749 token response.
func (s *Server) writeTokenResponse(w http.ResponseWriter, accessToken, refreshToken, scope string) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    int(s.cfg.AccessTokenTTL.Seconds()),
		"refresh_token": refreshToken,
		"scope":         scope,
	})
}

// scopeOrDefault returns the stored scope, or the default MCP scope when unset.
func scopeOrDefault(scope *string) string {
	if scope != nil && *scope != "" {
		return *scope
	}

	return scopeMCP
}
