package oauth

import "net/http"

// ProtectedResourceMetadata returns a handler serving RFC 9728 protected-resource
// metadata for the MCP endpoint.
func (s *Server) ProtectedResourceMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"resource":                 s.cfg.Resource,
			"authorization_servers":    []string{s.cfg.Issuer},
			"scopes_supported":         []string{scopeMCP},
			"bearer_methods_supported": []string{"header"},
		})
	}
}

// AuthorizationServerMetadata returns a handler serving RFC 8414 authorization-server
// metadata.
func (s *Server) AuthorizationServerMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"issuer":                                s.cfg.Issuer,
			"authorization_endpoint":                s.cfg.Issuer + "/oauth/authorize",
			"token_endpoint":                        s.cfg.Issuer + "/oauth/token",
			"registration_endpoint":                 s.cfg.Issuer + "/oauth/register",
			"response_types_supported":              []string{"code"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
			"code_challenge_methods_supported":      []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
		})
	}
}
