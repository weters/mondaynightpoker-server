// Package oauth implements an OAuth 2.1 authorization server and a resource-server
// middleware guarding the MCP endpoint. It issues RS256 access tokens plus rotating
// refresh tokens, performs the authorization-code + PKCE flow with a login form, and
// supports RFC 7591 dynamic client registration.
package oauth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"time"

	"mondaynightpoker-server/pkg/model"

	"github.com/sirupsen/logrus"
)

// Default token/code lifetimes applied when the corresponding Config field is zero.
const (
	defaultAccessTokenTTL  = time.Hour
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
	defaultAuthCodeTTL     = 10 * time.Minute
)

// nonceTTL is how long an authorization login form (and its anti-CSRF nonce) stays valid.
const nonceTTL = 10 * time.Minute

// scopeMCP is the single scope supported by this authorization server.
const scopeMCP = "mcp"

// tokenUseMCPAccess is the value of the token_use claim on issued MCP access tokens.
const tokenUseMCPAccess = "mcp_access"

// Config configures a Server. Issuer is the authorization server base URL and Resource
// is the full MCP resource URL (typically Issuer + "/mcp").
type Config struct {
	Issuer          string
	Resource        string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	AuthCodeTTL     time.Duration
}

// Server is an OAuth 2.1 authorization server and MCP resource-server middleware provider.
type Server struct {
	repos    *model.Repositories
	priv     *rsa.PrivateKey
	pub      *rsa.PublicKey
	cfg      Config
	nonceKey []byte
}

// New creates a Server. Zero-valued Config TTLs fall back to package defaults. The MCP
// access-token TTL may be set negative to mint already-expired tokens (used in tests).
func New(repos *model.Repositories, priv *rsa.PrivateKey, pub *rsa.PublicKey, cfg Config) *Server {
	if cfg.AccessTokenTTL == 0 {
		cfg.AccessTokenTTL = defaultAccessTokenTTL
	}
	if cfg.RefreshTokenTTL == 0 {
		cfg.RefreshTokenTTL = defaultRefreshTokenTTL
	}
	if cfg.AuthCodeTTL == 0 {
		cfg.AuthCodeTTL = defaultAuthCodeTTL
	}

	nonceKey := make([]byte, 32)
	if _, err := rand.Read(nonceKey); err != nil {
		// crypto/rand failure is catastrophic and effectively impossible on supported platforms.
		panic("oauth: could not initialize nonce key: " + err.Error())
	}

	return &Server{
		repos:    repos,
		priv:     priv,
		pub:      pub,
		cfg:      cfg,
		nonceKey: nonceKey,
	}
}

// writeJSON writes body as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, statusCode int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		logrus.WithError(err).Error("oauth: could not write JSON response")
	}
}

// writeOAuthError writes an RFC 6749 style error object with the given HTTP status.
func writeOAuthError(w http.ResponseWriter, statusCode int, code, description string) {
	body := map[string]string{"error": code}
	if description != "" {
		body["error_description"] = description
	}
	writeJSON(w, statusCode, body)
}
