package oauth

import (
	"net/http"
	"net/url"
	"time"

	"mondaynightpoker-server/pkg/model"

	"github.com/sirupsen/logrus"
)

// genericLoginError is shown for any failed sign-in (unknown email, wrong password, or
// an unverified account) so the form never leaks which condition failed.
const genericLoginError = "Invalid email address or password."

// Authorize returns the GET handler for the authorization endpoint. It validates the
// client and OAuth parameters and renders the login form.
func (s *Server) Authorize() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		clientID := q.Get("client_id")
		redirectURI := q.Get("redirect_uri")

		client, err := s.repos.OAuth.GetClient(r.Context(), clientID)
		if err != nil {
			s.renderErrorPage(w, http.StatusBadRequest, "Unknown or invalid client.")
			return
		}

		if redirectURI == "" || !redirectURIMatches(client.RedirectURIs, redirectURI) {
			s.renderErrorPage(w, http.StatusBadRequest, "The redirect URI is not registered for this client.")
			return
		}

		// From here the redirect_uri is trusted, so protocol errors bounce back to the client.
		state := q.Get("state")
		if errCode := validateAuthParams(q.Get); errCode != "" {
			s.redirectError(w, r, redirectURI, errCode, state)
			return
		}

		params := collectParams(q.Get)
		nonce := s.signNonce(params, time.Now().Add(nonceTTL))

		s.renderLogin(w, http.StatusOK, loginPageDataFrom(q.Get, nonce, ""))
	}
}

// AuthorizePost returns the POST handler for the authorization endpoint. It verifies the
// anti-CSRF nonce, re-validates the request, authenticates the player, and issues an
// authorization code.
func (s *Server) AuthorizePost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			s.renderErrorPage(w, http.StatusBadRequest, "The request could not be processed.")
			return
		}

		if !s.verifyNonce(collectParams(r.FormValue), r.FormValue("nonce")) {
			s.renderErrorPage(w, http.StatusBadRequest, "Your session has expired. Please start again.")
			return
		}

		clientID := r.FormValue("client_id")
		redirectURI := r.FormValue("redirect_uri")

		// Never trust the hidden fields for redirect_uri validation: re-check the client.
		client, err := s.repos.OAuth.GetClient(r.Context(), clientID)
		if err != nil {
			s.renderErrorPage(w, http.StatusBadRequest, "Unknown or invalid client.")
			return
		}

		if redirectURI == "" || !redirectURIMatches(client.RedirectURIs, redirectURI) {
			s.renderErrorPage(w, http.StatusBadRequest, "The redirect URI is not registered for this client.")
			return
		}

		state := r.FormValue("state")
		if errCode := validateAuthParams(r.FormValue); errCode != "" {
			s.redirectError(w, r, redirectURI, errCode, state)
			return
		}

		// Any player with valid credentials for a verified account may authorize;
		// GetPlayerByEmailAndPassword already rejects unverified/blocked/deleted accounts.
		player, err := s.repos.Players.GetPlayerByEmailAndPassword(r.Context(), r.FormValue("email"), r.FormValue("password"))
		if err != nil {
			// Generic message so the form never reveals whether the email exists.
			nonce := s.signNonce(collectParams(r.FormValue), time.Now().Add(nonceTTL))
			s.renderLogin(w, http.StatusOK, loginPageDataFrom(r.FormValue, nonce, genericLoginError))
			return
		}

		rawCode, err := randomToken(32)
		if err != nil {
			logrus.WithError(err).Error("oauth: could not generate authorization code")
			s.renderErrorPage(w, http.StatusInternalServerError, "An internal error occurred.")
			return
		}

		code := &model.OAuthAuthorizationCode{
			CodeHash:      sha256Hex(rawCode),
			ClientID:      clientID,
			PlayerID:      player.ID,
			RedirectURI:   redirectURI,
			CodeChallenge: r.FormValue("code_challenge"),
			Scope:         nullableString(r.FormValue("scope")),
			Resource:      nullableString(r.FormValue("resource")),
			Expires:       time.Now().Add(s.cfg.AuthCodeTTL),
		}

		if err := s.repos.OAuth.CreateAuthCode(r.Context(), code); err != nil {
			logrus.WithError(err).Error("oauth: could not persist authorization code")
			s.renderErrorPage(w, http.StatusInternalServerError, "An internal error occurred.")
			return
		}

		s.redirectSuccess(w, r, redirectURI, rawCode, state)
	}
}

// validateAuthParams returns an OAuth error code if the required authorization parameters
// are missing or unsupported, or "" when they are valid.
func validateAuthParams(get func(string) string) string {
	if get("response_type") != "code" {
		return "unsupported_response_type"
	}

	if get("code_challenge") == "" {
		return "invalid_request"
	}

	if get("code_challenge_method") != "S256" {
		return "invalid_request"
	}

	return ""
}

// loginPageDataFrom builds the login form model from a parameter getter.
func loginPageDataFrom(get func(string) string, nonce, errMsg string) loginPageData {
	return loginPageData{
		Error:               errMsg,
		Nonce:               nonce,
		ClientID:            get("client_id"),
		RedirectURI:         get("redirect_uri"),
		ResponseType:        "code",
		CodeChallenge:       get("code_challenge"),
		CodeChallengeMethod: "S256",
		Scope:               get("scope"),
		State:               get("state"),
		Resource:            get("resource"),
	}
}

// redirectError bounces an OAuth error back to the client's redirect URI.
func (s *Server) redirectError(w http.ResponseWriter, r *http.Request, redirectURI, errCode, state string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		s.renderErrorPage(w, http.StatusBadRequest, "The redirect URI is invalid.")
		return
	}

	q := u.Query()
	q.Set("error", errCode)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

// redirectSuccess bounces the issued authorization code back to the client's redirect URI.
func (s *Server) redirectSuccess(w http.ResponseWriter, r *http.Request, redirectURI, code, state string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		s.renderErrorPage(w, http.StatusBadRequest, "The redirect URI is invalid.")
		return
	}

	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}
