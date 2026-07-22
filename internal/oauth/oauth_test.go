package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/model"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pkcePair returns a random verifier and its S256 challenge.
func pkcePair(t *testing.T) (verifier, challenge string) {
	t.Helper()
	v, err := randomToken(32)
	require.NoError(t, err)
	sum := sha256.Sum256([]byte(v))
	return v, base64.RawURLEncoding.EncodeToString(sum[:])
}

func decodeJSON(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

// -------------------- metadata --------------------

func TestProtectedResourceMetadata(t *testing.T) {
	s := newTestServer(t, nil)
	rr := httptest.NewRecorder()
	s.ProtectedResourceMetadata()(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	m := decodeJSON(t, rr.Body.Bytes())
	assert.Equal(t, testResource, m["resource"])
	assert.Equal(t, []interface{}{testIssuer}, m["authorization_servers"])
	assert.Equal(t, []interface{}{"mcp"}, m["scopes_supported"])
	assert.Equal(t, []interface{}{"header"}, m["bearer_methods_supported"])
}

func TestAuthorizationServerMetadata(t *testing.T) {
	s := newTestServer(t, nil)
	rr := httptest.NewRecorder()
	s.AuthorizationServerMetadata()(rr, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	m := decodeJSON(t, rr.Body.Bytes())
	assert.Equal(t, testIssuer, m["issuer"])
	assert.Equal(t, testIssuer+"/oauth/authorize", m["authorization_endpoint"])
	assert.Equal(t, testIssuer+"/oauth/token", m["token_endpoint"])
	assert.Equal(t, testIssuer+"/oauth/register", m["registration_endpoint"])
	assert.Equal(t, []interface{}{"code"}, m["response_types_supported"])
	assert.Equal(t, []interface{}{"authorization_code", "refresh_token"}, m["grant_types_supported"])
	assert.Equal(t, []interface{}{"S256"}, m["code_challenge_methods_supported"])
	assert.Equal(t, []interface{}{"none"}, m["token_endpoint_auth_methods_supported"])
}

// -------------------- register --------------------

func TestRegister_Happy(t *testing.T) {
	s := newTestServer(t, nil)
	body := `{"client_name":"My App","redirect_uris":["https://app.example.com/cb","http://127.0.0.1:1234/cb"]}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	rr := httptest.NewRecorder()
	s.Register()(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code)
	m := decodeJSON(t, rr.Body.Bytes())
	assert.NotEmpty(t, m["client_id"])
	assert.Equal(t, "My App", m["client_name"])
	assert.Equal(t, "none", m["token_endpoint_auth_method"])
	assert.Equal(t, []interface{}{"authorization_code", "refresh_token"}, m["grant_types"])
	assert.NotZero(t, m["client_id_issued_at"])

	// verify persisted
	client, err := testRepos.OAuth.GetClient(cbg, m["client_id"].(string))
	require.NoError(t, err)
	assert.Equal(t, "none", client.TokenEndpointAuthMethod)
}

func TestRegister_BadRedirectURIs(t *testing.T) {
	s := newTestServer(t, nil)

	cases := map[string]string{
		"empty":          `{"client_name":"x","redirect_uris":[]}`,
		"missing":        `{"client_name":"x"}`,
		"http non-loop":  `{"client_name":"x","redirect_uris":["http://evil.example.com/cb"]}`,
		"custom scheme":  `{"client_name":"x","redirect_uris":["ftp://host/cb"]}`,
		"one bad in set": `{"client_name":"x","redirect_uris":["https://ok.example.com/cb","http://evil.example.com/cb"]}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
			rr := httptest.NewRecorder()
			s.Register()(rr, req)
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			m := decodeJSON(t, rr.Body.Bytes())
			assert.Equal(t, "invalid_client_metadata", m["error"])
		})
	}
}

// -------------------- authorize GET --------------------

func authorizeURL(clientID, redirectURI, challenge string, extra url.Values) string {
	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	return "/oauth/authorize?" + q.Encode()
}

func TestAuthorizeGet_Happy(t *testing.T) {
	s := newTestServer(t, nil)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)

	extra := url.Values{"state": {"xyz-state"}, "scope": {"mcp"}}
	req := httptest.NewRequest(http.MethodGet, authorizeURL(client.ClientID, "https://app.example.com/cb", challenge, extra), nil)
	rr := httptest.NewRecorder()
	s.Authorize()(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	html := rr.Body.String()
	assert.Contains(t, html, `name="client_id" value="`+client.ClientID+`"`)
	assert.Contains(t, html, `name="redirect_uri" value="https://app.example.com/cb"`)
	assert.Contains(t, html, `name="code_challenge" value="`+challenge+`"`)
	assert.Contains(t, html, `name="state" value="xyz-state"`)
	// nonce hidden field present and non-empty
	assert.Regexp(t, `name="nonce" value="[^"]+"`, html)
}

func TestAuthorizeGet_UnknownClient(t *testing.T) {
	s := newTestServer(t, nil)
	_, challenge := pkcePair(t)
	req := httptest.NewRequest(http.MethodGet, authorizeURL("does-not-exist", "https://app.example.com/cb", challenge, nil), nil)
	rr := httptest.NewRecorder()
	s.Authorize()(rr, req)

	// error page, never a redirect
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Empty(t, rr.Header().Get("Location"))
	assert.Contains(t, rr.Body.String(), "Unable to continue")
}

func TestAuthorizeGet_BadRedirectURI(t *testing.T) {
	s := newTestServer(t, nil)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)
	req := httptest.NewRequest(http.MethodGet, authorizeURL(client.ClientID, "https://attacker.example.com/cb", challenge, nil), nil)
	rr := httptest.NewRecorder()
	s.Authorize()(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Empty(t, rr.Header().Get("Location"))
}

func TestAuthorizeGet_PlainChallengeRejected(t *testing.T) {
	s := newTestServer(t, nil)
	client := seedClient(t, "https://app.example.com/cb")

	q := url.Values{}
	q.Set("client_id", client.ClientID)
	q.Set("redirect_uri", "https://app.example.com/cb")
	q.Set("response_type", "code")
	q.Set("code_challenge", "plainchallenge")
	q.Set("code_challenge_method", "plain")
	q.Set("state", "st")
	req := httptest.NewRequest(http.MethodGet, "/oauth/authorize?"+q.Encode(), nil)
	rr := httptest.NewRecorder()
	s.Authorize()(rr, req)

	// valid redirect_uri, so protocol error bounces back
	require.Equal(t, http.StatusFound, rr.Code)
	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", loc.Query().Get("error"))
	assert.Equal(t, "st", loc.Query().Get("state"))
}

// -------------------- authorize POST --------------------

// runAuthorizeGet performs the GET and extracts the nonce from the rendered form.
func runAuthorizeGet(t *testing.T, s *Server, clientID, redirectURI, challenge string, extra url.Values) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, authorizeURL(clientID, redirectURI, challenge, extra), nil)
	rr := httptest.NewRecorder()
	s.Authorize()(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	match := nonceRegexp.FindStringSubmatch(rr.Body.String())
	require.Len(t, match, 2, "nonce hidden field not found in form")
	return match[1]
}

var nonceRegexp = regexp.MustCompile(`name="nonce" value="([^"]+)"`)

func authorizePostForm(clientID, redirectURI, challenge, nonce, email, password string, extra url.Values) url.Values {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("redirect_uri", redirectURI)
	form.Set("response_type", "code")
	form.Set("code_challenge", challenge)
	form.Set("code_challenge_method", "S256")
	form.Set("nonce", nonce)
	form.Set("email", email)
	form.Set("password", password)
	for k, vs := range extra {
		for _, v := range vs {
			form.Set(k, v)
		}
	}
	return form
}

func postAuthorize(t *testing.T, s *Server, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.AuthorizePost()(rr, req)
	return rr
}

func TestAuthorizePost_Happy(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)

	extra := url.Values{"state": {"the-state"}, "scope": {"mcp"}}
	nonce := runAuthorizeGet(t, s, client.ClientID, "https://app.example.com/cb", challenge, extra)

	form := authorizePostForm(client.ClientID, "https://app.example.com/cb", challenge, nonce, admin.Email, testPassword, extra)
	rr := postAuthorize(t, s, form)

	require.Equal(t, http.StatusFound, rr.Code)
	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "app.example.com", loc.Host)
	assert.NotEmpty(t, loc.Query().Get("code"))
	assert.Equal(t, "the-state", loc.Query().Get("state"))
}

func TestAuthorizePost_WrongPassword(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)
	nonce := runAuthorizeGet(t, s, client.ClientID, "https://app.example.com/cb", challenge, nil)

	form := authorizePostForm(client.ClientID, "https://app.example.com/cb", challenge, nonce, admin.Email, "wrong-password", nil)
	rr := postAuthorize(t, s, form)

	// form re-rendered with generic error, 200, no redirect
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Location"))
	assert.Contains(t, rr.Body.String(), genericLoginError)
}

func TestAuthorizePost_NonAdminAllowed(t *testing.T) {
	s := newTestServer(t, nil)
	nonAdmin := seedPlayer(t, false)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)

	extra := url.Values{"state": {"na-state"}, "scope": {"mcp"}}
	nonce := runAuthorizeGet(t, s, client.ClientID, "https://app.example.com/cb", challenge, extra)

	form := authorizePostForm(client.ClientID, "https://app.example.com/cb", challenge, nonce, nonAdmin.Email, testPassword, extra)
	rr := postAuthorize(t, s, form)

	// non-admin players may now authorize: the login POST issues a code and redirects
	require.Equal(t, http.StatusFound, rr.Code)
	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	assert.NotEmpty(t, loc.Query().Get("code"))
	assert.Equal(t, "na-state", loc.Query().Get("state"))
}

func TestAuthorizePost_UnverifiedRejected(t *testing.T) {
	s := newTestServer(t, nil)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)

	// a freshly created (unverified) player cannot authorize
	email := util.RandomEmail()
	player, err := testRepos.Players.CreatePlayer(cbg, email, "Unverified", testPassword, "127.0.0.1")
	require.NoError(t, err)
	require.Equal(t, model.PlayerStatusCreated, player.Status)

	nonce := runAuthorizeGet(t, s, client.ClientID, "https://app.example.com/cb", challenge, nil)
	form := authorizePostForm(client.ClientID, "https://app.example.com/cb", challenge, nonce, email, testPassword, nil)
	rr := postAuthorize(t, s, form)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Header().Get("Location"))
	// identical generic message: does not leak that the account merely needs verifying
	assert.Contains(t, rr.Body.String(), genericLoginError)
}

func TestAuthorizePost_BadNonce(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)

	form := authorizePostForm(client.ClientID, "https://app.example.com/cb", challenge, "tampered.nonce", admin.Email, testPassword, nil)
	rr := postAuthorize(t, s, form)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Empty(t, rr.Header().Get("Location"))
}

// -------------------- token: authorization_code --------------------

// testRedirectURI is the registered redirect URI shared by the token-flow test helpers.
const testRedirectURI = "https://app.example.com/cb"

// obtainCode drives the full GET+POST authorize flow (for testRedirectURI) and returns
// the issued code.
func obtainCode(t *testing.T, s *Server, email, clientID, challenge string, extra url.Values) string {
	t.Helper()
	nonce := runAuthorizeGet(t, s, clientID, testRedirectURI, challenge, extra)
	form := authorizePostForm(clientID, testRedirectURI, challenge, nonce, email, testPassword, extra)
	rr := postAuthorize(t, s, form)
	require.Equal(t, http.StatusFound, rr.Code)
	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)
	return code
}

func tokenForm(values map[string]string) url.Values {
	form := url.Values{}
	for k, v := range values {
		form.Set(k, v)
	}
	return form
}

func postToken(t *testing.T, s *Server, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	s.Token()(rr, req)
	return rr
}

func TestToken_AuthCodeHappy(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	verifier, challenge := pkcePair(t)
	code := obtainCode(t, s, admin.Email, client.ClientID, challenge, url.Values{"scope": {"mcp"}})

	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  "https://app.example.com/cb",
		"client_id":     client.ClientID,
		"code_verifier": verifier,
	}))

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "no-store", rr.Header().Get("Cache-Control"))
	m := decodeJSON(t, rr.Body.Bytes())
	assert.Equal(t, "Bearer", m["token_type"])
	assert.Equal(t, "mcp", m["scope"])
	assert.NotEmpty(t, m["refresh_token"])
	assert.EqualValues(t, 3600, m["expires_in"])

	// verify access token claims
	claims := parseAccessToken(t, s, m["access_token"].(string))
	assert.Equal(t, testIssuer, claims["iss"])
	assert.Equal(t, testResource, claims["aud"])
	assert.Equal(t, strconv.FormatInt(admin.ID, 10), claims["sub"])
	assert.Equal(t, "mcp_access", claims["token_use"])
	assert.Equal(t, "mcp", claims["scope"])
	assert.NotEmpty(t, claims["jti"])
	assert.Greater(t, int64(claims["exp"].(float64)), time.Now().Unix())
}

func parseAccessToken(t *testing.T, s *Server, raw string) jwtgo.MapClaims {
	t.Helper()
	claims := jwtgo.MapClaims{}
	_, err := jwtgo.ParseWithClaims(raw, claims, func(_ *jwtgo.Token) (interface{}, error) {
		return &s.priv.PublicKey, nil
	})
	require.NoError(t, err)
	return claims
}

func TestToken_AuthCodeWrongVerifier(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	_, challenge := pkcePair(t)
	code := obtainCode(t, s, admin.Email, client.ClientID, challenge, nil)

	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  "https://app.example.com/cb",
		"client_id":     client.ClientID,
		"code_verifier": "totally-wrong-verifier",
	}))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "invalid_grant", decodeJSON(t, rr.Body.Bytes())["error"])
}

func TestToken_AuthCodeReused(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	verifier, challenge := pkcePair(t)
	code := obtainCode(t, s, admin.Email, client.ClientID, challenge, nil)

	form := tokenForm(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  "https://app.example.com/cb",
		"client_id":     client.ClientID,
		"code_verifier": verifier,
	})
	require.Equal(t, http.StatusOK, postToken(t, s, form).Code)

	// second use of the same code fails
	rr := postToken(t, s, form)
	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "invalid_grant", decodeJSON(t, rr.Body.Bytes())["error"])
}

func TestToken_AuthCodeRedirectMismatch(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb", "https://app.example.com/other")
	verifier, challenge := pkcePair(t)
	code := obtainCode(t, s, admin.Email, client.ClientID, challenge, nil)

	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  "https://app.example.com/other",
		"client_id":     client.ClientID,
		"code_verifier": verifier,
	}))

	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "invalid_grant", decodeJSON(t, rr.Body.Bytes())["error"])
}

func TestToken_MissingParams(t *testing.T) {
	s := newTestServer(t, nil)
	rr := postToken(t, s, tokenForm(map[string]string{"grant_type": "authorization_code"}))
	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "invalid_request", decodeJSON(t, rr.Body.Bytes())["error"])
}

// -------------------- token: refresh --------------------

// firstRefresh obtains an initial refresh token via the auth-code flow.
func firstRefresh(t *testing.T, s *Server, email, clientID string) string {
	t.Helper()
	verifier, challenge := pkcePair(t)
	code := obtainCode(t, s, email, clientID, challenge, nil)
	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  testRedirectURI,
		"client_id":     clientID,
		"code_verifier": verifier,
	}))
	require.Equal(t, http.StatusOK, rr.Code)
	return decodeJSON(t, rr.Body.Bytes())["refresh_token"].(string)
}

func TestToken_RefreshHappyAndRotation(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	refresh := firstRefresh(t, s, admin.Email, client.ClientID)

	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refresh,
		"client_id":     client.ClientID,
	}))
	require.Equal(t, http.StatusOK, rr.Code)
	m := decodeJSON(t, rr.Body.Bytes())
	newRefresh := m["refresh_token"].(string)
	assert.NotEqual(t, refresh, newRefresh)
	assert.NotEmpty(t, m["access_token"])

	// old refresh token is now invalid
	rr2 := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refresh,
		"client_id":     client.ClientID,
	}))
	require.Equal(t, http.StatusBadRequest, rr2.Code)
	assert.Equal(t, "invalid_grant", decodeJSON(t, rr2.Body.Bytes())["error"])
}

func TestToken_RefreshReuseRevokesFamily(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	refresh := firstRefresh(t, s, admin.Email, client.ClientID)

	// rotate once
	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refresh,
		"client_id":     client.ClientID,
	}))
	require.Equal(t, http.StatusOK, rr.Code)
	newRefresh := decodeJSON(t, rr.Body.Bytes())["refresh_token"].(string)

	// reuse the old (already-rotated) token: triggers family revocation
	rr2 := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refresh,
		"client_id":     client.ClientID,
	}))
	require.Equal(t, http.StatusBadRequest, rr2.Code)
	assert.Equal(t, "invalid_grant", decodeJSON(t, rr2.Body.Bytes())["error"])

	// the freshly-rotated token is now also revoked (family compromise)
	rr3 := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": newRefresh,
		"client_id":     client.ClientID,
	}))
	require.Equal(t, http.StatusBadRequest, rr3.Code)
	assert.Equal(t, "invalid_grant", decodeJSON(t, rr3.Body.Bytes())["error"])
}

func TestToken_RefreshDemotedAdmin(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "https://app.example.com/cb")
	refresh := firstRefresh(t, s, admin.Email, client.ClientID)

	// demote the admin: the MCP resource is open to all players, so refresh still works
	require.NoError(t, testRepos.Players.SetIsSiteAdmin(cbg, admin, false))

	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refresh,
		"client_id":     client.ClientID,
	}))
	require.Equal(t, http.StatusOK, rr.Code)
	assert.NotEmpty(t, decodeJSON(t, rr.Body.Bytes())["access_token"])
}

func TestToken_NonAdminAuthCodeFlow(t *testing.T) {
	s := newTestServer(t, nil)
	player := seedPlayer(t, false)
	client := seedClient(t, "https://app.example.com/cb")
	verifier, challenge := pkcePair(t)
	code := obtainCode(t, s, player.Email, client.ClientID, challenge, url.Values{"scope": {"mcp"}})

	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  "https://app.example.com/cb",
		"client_id":     client.ClientID,
		"code_verifier": verifier,
	}))

	require.Equal(t, http.StatusOK, rr.Code)
	claims := parseAccessToken(t, s, decodeJSON(t, rr.Body.Bytes())["access_token"].(string))
	assert.Equal(t, strconv.FormatInt(player.ID, 10), claims["sub"])
}

func TestToken_RefreshDeletedPlayer(t *testing.T) {
	s := newTestServer(t, nil)
	player := seedPlayer(t, false)
	client := seedClient(t, "https://app.example.com/cb")
	refresh := firstRefresh(t, s, player.Email, client.ClientID)

	// deleting the player revokes access even though the refresh token is otherwise valid
	require.NoError(t, testRepos.Players.Delete(cbg, player))

	rr := postToken(t, s, tokenForm(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refresh,
		"client_id":     client.ClientID,
	}))
	require.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "invalid_grant", decodeJSON(t, rr.Body.Bytes())["error"])
}

// -------------------- middleware --------------------

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, ok := PlayerIDFromContext(r.Context())
		if !ok {
			http.Error(w, "no player id", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.FormatInt(id, 10)))
	})
}

// callerHandler echoes the stashed Caller as "id:isAdmin" so tests can assert scoping.
func callerHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, ok := CallerFromContext(r.Context())
		if !ok {
			http.Error(w, "no caller", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "%d:%t", c.PlayerID, c.IsSiteAdmin)
	})
}

func TestRequireMCPAuth_NoHeader(t *testing.T) {
	s := newTestServer(t, nil)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(okHandler()).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/mcp", nil))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Header().Get("WWW-Authenticate"), `resource_metadata="`+testIssuer+`/.well-known/oauth-protected-resource"`)
	assert.Contains(t, rr.Header().Get("WWW-Authenticate"), `error="invalid_token"`)
}

func TestRequireMCPAuth_GarbageToken(t *testing.T) {
	s := newTestServer(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-jwt")
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireMCPAuth_WrongAudience(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)

	// sign a token with aud = issuer (wrong)
	claims := jwtgo.MapClaims{
		"iss":       testIssuer,
		"sub":       strconv.FormatInt(admin.ID, 10),
		"aud":       testIssuer,
		"exp":       time.Now().Add(time.Hour).Unix(),
		"iat":       time.Now().Unix(),
		"token_use": "mcp_access",
	}
	raw, err := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims).SignedString(testPriv)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireMCPAuth_ValidToken(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	raw, err := s.newAccessToken(admin.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(okHandler()).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, strconv.FormatInt(admin.ID, 10), rr.Body.String())
}

func TestRequireMCPAuth_StashesAdminCaller(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	raw, err := s.newAccessToken(admin.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(callerHandler()).ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, fmt.Sprintf("%d:true", admin.ID), rr.Body.String())
}

func TestRequireMCPAuth_StashesNonAdminCaller(t *testing.T) {
	s := newTestServer(t, nil)
	player := seedPlayer(t, false)
	raw, err := s.newAccessToken(player.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(callerHandler()).ServeHTTP(rr, req)

	// a non-admin now passes the middleware; the Caller reflects the live flag
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, fmt.Sprintf("%d:false", player.ID), rr.Body.String())
}

func TestRequireMCPAuth_DemotedAdmin(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	raw, err := s.newAccessToken(admin.ID)
	require.NoError(t, err)

	require.NoError(t, testRepos.Players.SetIsSiteAdmin(cbg, admin, false))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(callerHandler()).ServeHTTP(rr, req)

	// a demoted admin is still a valid player: 200, but IsSiteAdmin is now false
	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, fmt.Sprintf("%d:false", admin.ID), rr.Body.String())
}

func TestRequireMCPAuth_DeletedPlayer(t *testing.T) {
	s := newTestServer(t, nil)
	player := seedPlayer(t, false)
	raw, err := s.newAccessToken(player.ID)
	require.NoError(t, err)

	require.NoError(t, testRepos.Players.Delete(cbg, player))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(callerHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestRequireMCPAuth_ExpiredToken(t *testing.T) {
	s := newTestServer(t, func(c *Config) { c.AccessTokenTTL = -time.Hour })
	admin := seedAdmin(t)
	raw, err := s.newAccessToken(admin.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+raw)
	rr := httptest.NewRecorder()
	s.RequireMCPAuth(okHandler()).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// -------------------- loopback redirect matching --------------------

func TestAuthorize_LoopbackAnyPort(t *testing.T) {
	s := newTestServer(t, nil)
	admin := seedAdmin(t)
	client := seedClient(t, "http://127.0.0.1:8080/callback")
	_, challenge := pkcePair(t)

	// request uses a different port than registered
	provided := "http://127.0.0.1:54321/callback"
	nonce := runAuthorizeGet(t, s, client.ClientID, provided, challenge, nil)
	form := authorizePostForm(client.ClientID, provided, challenge, nonce, admin.Email, testPassword, nil)
	rr := postAuthorize(t, s, form)

	require.Equal(t, http.StatusFound, rr.Code)
	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:54321", loc.Host)
	assert.NotEmpty(t, loc.Query().Get("code"))
}
