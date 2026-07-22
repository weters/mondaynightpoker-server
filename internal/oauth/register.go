package oauth

import (
	"encoding/json"
	"net/http"

	"mondaynightpoker-server/pkg/model"

	"github.com/sirupsen/logrus"
)

// registrationRequest is the subset of RFC 7591 client metadata accepted for registration.
type registrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// Register returns the RFC 7591 dynamic client registration handler. It only creates
// public clients (token_endpoint_auth_method "none").
func (s *Server) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req registrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "request body must be valid JSON")
			return
		}

		if len(req.RedirectURIs) == 0 {
			writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "redirect_uris is required and must not be empty")
			return
		}

		for _, uri := range req.RedirectURIs {
			if !isValidRegistrationURI(uri) {
				writeOAuthError(w, http.StatusBadRequest, "invalid_client_metadata", "each redirect_uri must be https or a loopback http URI")
				return
			}
		}

		grantTypes := req.GrantTypes
		if len(grantTypes) == 0 {
			grantTypes = []string{"authorization_code", "refresh_token"}
		}

		clientID, err := randomToken(32)
		if err != nil {
			logrus.WithError(err).Error("oauth: could not generate client_id")
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
			return
		}

		client := &model.OAuthClient{
			ClientID:                clientID,
			ClientName:              req.ClientName,
			RedirectURIs:            req.RedirectURIs,
			GrantTypes:              grantTypes,
			TokenEndpointAuthMethod: "none",
		}

		if err := s.repos.OAuth.CreateClient(r.Context(), client); err != nil {
			logrus.WithError(err).Error("oauth: could not create client")
			writeOAuthError(w, http.StatusInternalServerError, "server_error", "")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"client_id":                  client.ClientID,
			"client_name":                client.ClientName,
			"redirect_uris":              client.RedirectURIs,
			"grant_types":                client.GrantTypes,
			"token_endpoint_auth_method": client.TokenEndpointAuthMethod,
			"client_id_issued_at":        client.Created.Unix(),
		})
	}
}
