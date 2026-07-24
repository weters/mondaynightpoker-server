package mux

import (
	"crypto/rand"
	"crypto/rsa"
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

	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/mcpserver"
	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/model"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	mcpIssuer      = "https://mcp.mux.test"
	mcpPassword    = "sup3r-s3cret-pw"
	mcpRedirectURI = "http://127.0.0.1:9999/callback"

	ctJSON = "application/json"
	ctForm = "application/x-www-form-urlencoded"
	// mcpAccept is the Accept header the MCP streamable HTTP handler requires.
	mcpAccept = "application/json, text/event-stream"

	grantAuthCode = "authorization_code"
	grantRefresh  = "refresh_token"

	pathAuthorize = "/oauth/authorize"
	pathToken     = "/oauth/token" //nolint:gosec // G101 false positive: this is a URL path, not a credential
	pathRegister  = "/oauth/register"
	pathMCP       = "/mcp"

	tokenUseMCP = "mcp_access"

	// JSON-RPC request bodies. In stateless mode each POST stands alone and the
	// handler seeds a default initialized session, so no initialize-first is
	// required before tools/list or tools/call.
	rpcToolsList     = `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	rpcListGameTypes = `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_game_types","arguments":{}}}`
	rpcListPlayers   = `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_players","arguments":{}}}`
	rpcGetPlayerFmt  = `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_player","arguments":{"id":%d}}}`
	rpcWhoami        = `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"whoami","arguments":{}}}`

	errTextRequiresAdmin    = "requires site admin"
	errTextPermissionDenied = "permission denied"
)

// mcpResource is the protected-resource URL (aud claim on access tokens).
var mcpResource = mcpIssuer + "/mcp"

// mcpNonceRe extracts the anti-CSRF nonce hidden field from the login form.
var mcpNonceRe = regexp.MustCompile(`name="nonce" value="([^"]+)"`)

// allToolNames are the tools the MCP server must advertise via tools/list.
var allToolNames = []string{
	"whoami", "list_players", "get_player", "get_player_by_email", "get_player_stats",
	"get_player_profile", "list_player_tables", "list_tables", "get_table",
	"get_table_roster", "list_table_games", "get_game", "get_table_stats",
	"list_player_transactions", "leaderboard", "list_game_types",
}

// allGameSlugs are the game-type identifiers list_game_types must return.
var allGameSlugs = []string{
	"acey-deucey", "bourre", "guts", "little-l", "pass-the-poop",
	"seven-card", "texas-hold-em",
}

// newMCPMux builds a Mux wired with the OAuth 2.1 server and MCP handler. The RSA
// key pair (shared with the token signer) is returned for hand-signing test tokens
// and verifying issued ones.
func newMCPMux(t *testing.T) (*Mux, *rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()

	priv, pub, err := jwt.LoadKeyPair(testCfg.JWT)
	require.NoError(t, err)

	deps := testDeps()
	deps.OAuth = oauth.New(testRepos, priv, pub, oauth.Config{Issuer: mcpIssuer, Resource: mcpResource})
	deps.MCPHandler = mcpserver.New(testRepos, "test")

	return NewMux(deps), priv, pub
}

// seedMCPPlayer creates a verified player (optionally a site admin) whose password
// is mcpPassword, so it can authenticate via the login form.
func seedMCPPlayer(t *testing.T, admin bool) *model.Player {
	t.Helper()

	p, err := testRepos.Players.CreatePlayer(cbg, util.RandomEmail(), "MCP Player", mcpPassword, "127.0.0.1")
	require.NoError(t, err)

	p.Status = model.PlayerStatusVerified
	require.NoError(t, testRepos.Players.Save(cbg, p))

	if admin {
		require.NoError(t, testRepos.Players.SetIsSiteAdmin(cbg, p, true))
	}

	return p
}

// mcpPKCE returns a random PKCE verifier and its S256 challenge.
func mcpPKCE(t *testing.T) (verifier, challenge string) {
	t.Helper()

	b := make([]byte, 32)
	_, err := rand.Read(b)
	require.NoError(t, err)

	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

// serve runs req through the mux and returns the recorded response.
func serve(m *Mux, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	m.ServeHTTP(rr, req)
	return rr
}

// decodeMap decodes a JSON object body into a map.
func decodeMap(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(body, &m))
	return m
}

// registerClient performs dynamic client registration and returns the client_id.
func registerClient(t *testing.T, m *Mux) string {
	t.Helper()

	body := fmt.Sprintf(`{"client_name":"MCP Test Client","redirect_uris":[%q]}`, mcpRedirectURI)
	req := httptest.NewRequest(http.MethodPost, pathRegister, strings.NewReader(body))
	req.Header.Set("Content-Type", ctJSON)

	rr := serve(m, req)
	require.Equal(t, http.StatusCreated, rr.Code)

	clientID, _ := decodeMap(t, rr.Body.Bytes())["client_id"].(string)
	require.NotEmpty(t, clientID)
	return clientID
}

// authorizeQuery builds the base /oauth/authorize parameters.
func authorizeQuery(clientID, redirectURI, challenge, challengeMethod string) url.Values {
	return url.Values{
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"code_challenge":        {challenge},
		"code_challenge_method": {challengeMethod},
	}
}

// authorizeGet issues the GET /oauth/authorize request.
func authorizeGet(m *Mux, q url.Values) *httptest.ResponseRecorder {
	return serve(m, httptest.NewRequest(http.MethodGet, pathAuthorize+"?"+q.Encode(), nil))
}

// authorizePost issues the POST /oauth/authorize (login) request.
func authorizePost(m *Mux, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, pathAuthorize, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", ctForm)
	return serve(m, req)
}

// runAuthorize drives the GET (to obtain the nonce) then POST login flow, returning
// the POST recorder. Callers assert on the outcome (302 with code, or 200 re-render).
func runAuthorize(t *testing.T, m *Mux, clientID, challenge, email, password string, extra url.Values) *httptest.ResponseRecorder {
	t.Helper()

	q := authorizeQuery(clientID, mcpRedirectURI, challenge, "S256")
	for k, vs := range extra {
		q[k] = vs
	}

	getRR := authorizeGet(m, q)
	require.Equal(t, http.StatusOK, getRR.Code)

	match := mcpNonceRe.FindStringSubmatch(getRR.Body.String())
	require.Len(t, match, 2, "nonce hidden field not found in login form")

	form := q
	form.Set("nonce", match[1])
	form.Set("email", email)
	form.Set("password", password)
	return authorizePost(m, form)
}

// obtainCode drives the full authorize flow for a player and returns the issued code.
func obtainCode(t *testing.T, m *Mux, clientID, challenge, email string) string {
	t.Helper()

	rr := runAuthorize(t, m, clientID, challenge, email, mcpPassword, nil)
	require.Equal(t, http.StatusFound, rr.Code)

	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)

	code := loc.Query().Get("code")
	require.NotEmpty(t, code)
	return code
}

// tokenRequest posts a form-encoded request to the token endpoint.
func tokenRequest(m *Mux, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, pathToken, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", ctForm)
	return serve(m, req)
}

// authCodeForm builds the authorization_code grant form.
func authCodeForm(code, clientID, verifier string) url.Values {
	return url.Values{
		"grant_type":    {grantAuthCode},
		"code":          {code},
		"redirect_uri":  {mcpRedirectURI},
		"client_id":     {clientID},
		"code_verifier": {verifier},
	}
}

// mcpPost sends a JSON-RPC request to /mcp with the given bearer token (empty to omit).
func mcpPost(m *Mux, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, pathMCP, strings.NewReader(body))
	req.Header.Set("Content-Type", ctJSON)
	req.Header.Set("Accept", mcpAccept)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return serve(m, req)
}

// mcpPayload extracts the JSON-RPC payload from an MCP response body. The stateless
// streamable handler frames replies as text/event-stream ("event: message\ndata: {...}"),
// so the data lines are unwrapped before decoding.
func mcpPayload(t *testing.T, body string) string {
	t.Helper()

	body = strings.TrimSpace(body)
	if !strings.Contains(body, "data:") {
		return body
	}

	var b strings.Builder
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data:") {
			b.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	return b.String()
}

// parseAccessClaims verifies and returns the claims of an RS256 access token.
func parseAccessClaims(t *testing.T, pub *rsa.PublicKey, raw string) jwtgo.MapClaims {
	t.Helper()

	claims := jwtgo.MapClaims{}
	_, err := jwtgo.ParseWithClaims(raw, claims, func(*jwtgo.Token) (interface{}, error) {
		return pub, nil
	})
	require.NoError(t, err)
	return claims
}

// signMCPToken hand-signs an RS256 MCP access token with the shared key, allowing
// tests to control the audience and expiry.
func signMCPToken(t *testing.T, priv *rsa.PrivateKey, playerID int64, aud string, exp time.Time) string {
	t.Helper()

	claims := jwtgo.MapClaims{
		"iss":       mcpIssuer,
		"sub":       strconv.FormatInt(playerID, 10),
		"aud":       aud,
		"exp":       exp.Unix(),
		"iat":       time.Now().Unix(),
		"token_use": tokenUseMCP,
	}
	raw, err := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims).SignedString(priv)
	require.NoError(t, err)
	return raw
}

// assertContainsAll asserts that body contains each of subs.
func assertContainsAll(t *testing.T, body string, subs []string) {
	t.Helper()
	for _, s := range subs {
		assert.Contains(t, body, s)
	}
}

// TestMCP_EndToEndHappyPath walks the entire OAuth 2.1 + MCP flow through the mux:
// register -> authorize (PKCE) -> token -> MCP tools -> refresh with rotation.
func TestMCP_EndToEndHappyPath(t *testing.T) {
	m, _, pub := newMCPMux(t)
	admin := seedMCPPlayer(t, true)

	// 1. dynamic client registration
	clientID := registerClient(t, m)

	// 2. PKCE + authorize (GET renders login form, POST issues the code)
	verifier, challenge := mcpPKCE(t)
	const state = "opaque-state-value"
	rr := runAuthorize(t, m, clientID, challenge, admin.Email, mcpPassword, url.Values{
		"state": {state},
		"scope": {"mcp"},
	})

	// 3. the login POST redirects to the client with the code + exact state echo
	require.Equal(t, http.StatusFound, rr.Code)
	loc, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9999", loc.Host)
	code := loc.Query().Get("code")
	require.NotEmpty(t, code)
	assert.Equal(t, state, loc.Query().Get("state"))

	// 4. token exchange (authorization_code grant + PKCE verifier)
	tokRR := tokenRequest(m, authCodeForm(code, clientID, verifier))
	require.Equal(t, http.StatusOK, tokRR.Code)
	tok := decodeMap(t, tokRR.Body.Bytes())
	assert.Equal(t, "Bearer", tok["token_type"])
	assert.EqualValues(t, 3600, tok["expires_in"])
	accessToken, _ := tok["access_token"].(string)
	refreshToken, _ := tok["refresh_token"].(string)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// verify the access-token claims
	claims := parseAccessClaims(t, pub, accessToken)
	assert.Equal(t, mcpIssuer, claims["iss"])
	assert.Equal(t, mcpResource, claims["aud"])
	assert.Equal(t, tokenUseMCP, claims["token_use"])
	assert.Equal(t, strconv.FormatInt(admin.ID, 10), claims["sub"])

	// 5. exercise the MCP endpoint with the bearer token
	listRR := mcpPost(m, accessToken, rpcToolsList)
	require.Equal(t, http.StatusOK, listRR.Code)
	assertContainsAll(t, mcpPayload(t, listRR.Body.String()), allToolNames)

	callRR := mcpPost(m, accessToken, rpcListGameTypes)
	require.Equal(t, http.StatusOK, callRR.Code)
	assertContainsAll(t, mcpPayload(t, callRR.Body.String()), allGameSlugs)

	// 6. refresh grant rotates the refresh token; the old one is then rejected
	refreshForm := url.Values{
		"grant_type":    {grantRefresh},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	refRR := tokenRequest(m, refreshForm)
	require.Equal(t, http.StatusOK, refRR.Code)
	refBody := decodeMap(t, refRR.Body.Bytes())
	newRefresh, _ := refBody["refresh_token"].(string)
	assert.NotEmpty(t, refBody["access_token"])
	assert.NotEmpty(t, newRefresh)
	assert.NotEqual(t, refreshToken, newRefresh)

	// reusing the original (now-rotated) refresh token fails
	reuseRR := tokenRequest(m, refreshForm)
	require.Equal(t, http.StatusBadRequest, reuseRR.Code)
	assert.Equal(t, "invalid_grant", decodeMap(t, reuseRR.Body.Bytes())["error"])
}

// TestMCP_WellKnownMetadata verifies both discovery endpoints are served
// unauthenticated with correct JSON.
func TestMCP_WellKnownMetadata(t *testing.T) {
	m, _, _ := newMCPMux(t)

	prRR := serve(m, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource", nil))
	require.Equal(t, http.StatusOK, prRR.Code)
	pr := decodeMap(t, prRR.Body.Bytes())
	assert.Equal(t, mcpResource, pr["resource"])
	assert.Equal(t, []interface{}{mcpIssuer}, pr["authorization_servers"])

	asRR := serve(m, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil))
	require.Equal(t, http.StatusOK, asRR.Code)
	as := decodeMap(t, asRR.Body.Bytes())
	assert.Equal(t, mcpIssuer, as["issuer"])
	assert.Equal(t, mcpIssuer+pathAuthorize, as["authorization_endpoint"])
	assert.Equal(t, mcpIssuer+pathToken, as["token_endpoint"])
	assert.Equal(t, mcpIssuer+pathRegister, as["registration_endpoint"])
}

// TestMCP_Unauthorized covers the bearer-token rejection paths on /mcp.
func TestMCP_Unauthorized(t *testing.T) {
	m, priv, _ := newMCPMux(t)
	admin := seedMCPPlayer(t, true)
	playerSessionJWT, err := testSigner.Sign(admin.ID)
	require.NoError(t, err)

	tests := []struct {
		name  string
		token string
	}{
		{"no authorization header", ""},
		{"garbage token", "not-a-real-jwt"},
		{"player-session jwt wrong audience", playerSessionJWT},
		{"expired access token", signMCPToken(t, priv, admin.ID, mcpResource, time.Now().Add(-time.Hour))},
		{"wrong audience access token", signMCPToken(t, priv, admin.ID, mcpIssuer, time.Now().Add(time.Hour))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := mcpPost(m, tc.token, rpcToolsList)
			assert.Equal(t, http.StatusUnauthorized, rr.Code)
			assert.Contains(t, rr.Header().Get("WWW-Authenticate"), "resource_metadata=")
		})
	}
}

// TestMCP_DemotedAdminBecomesScoped verifies the live admin re-check: a valid token
// keeps working once the player is demoted (401s are reserved for invalid/missing
// tokens), but admin-only tools now reject it with a tool error instead of succeeding.
func TestMCP_DemotedAdminBecomesScoped(t *testing.T) {
	m, _, _ := newMCPMux(t)
	admin := seedMCPPlayer(t, true)
	clientID := registerClient(t, m)

	verifier, challenge := mcpPKCE(t)
	code := obtainCode(t, m, clientID, challenge, admin.Email)
	tokRR := tokenRequest(m, authCodeForm(code, clientID, verifier))
	require.Equal(t, http.StatusOK, tokRR.Code)
	accessToken, _ := decodeMap(t, tokRR.Body.Bytes())["access_token"].(string)
	require.NotEmpty(t, accessToken)

	// admin-only tool works while admin
	okRR := mcpPost(m, accessToken, rpcListPlayers)
	require.Equal(t, http.StatusOK, okRR.Code)
	assert.NotContains(t, mcpPayload(t, okRR.Body.String()), errTextRequiresAdmin)

	// demote and retry: the token itself still works (live check reads the DB,
	// not the token claims), but the admin-only tool now returns a tool error.
	require.NoError(t, testRepos.Players.SetIsSiteAdmin(cbg, admin, false))

	stillAuthedRR := mcpPost(m, accessToken, rpcToolsList)
	require.Equal(t, http.StatusOK, stillAuthedRR.Code)

	scopedRR := mcpPost(m, accessToken, rpcListPlayers)
	require.Equal(t, http.StatusOK, scopedRR.Code)
	scopedPayload := mcpPayload(t, scopedRR.Body.String())
	assert.Contains(t, scopedPayload, `"isError":true`)
	assert.Contains(t, scopedPayload, errTextRequiresAdmin)
}

// TestMCP_TokenExchangeFailures covers PKCE and code-reuse failures at the token endpoint.
func TestMCP_TokenExchangeFailures(t *testing.T) {
	m, _, _ := newMCPMux(t)
	admin := seedMCPPlayer(t, true)
	clientID := registerClient(t, m)

	t.Run("wrong code_verifier", func(t *testing.T) {
		_, challenge := mcpPKCE(t)
		code := obtainCode(t, m, clientID, challenge, admin.Email)

		rr := tokenRequest(m, authCodeForm(code, clientID, "totally-wrong-verifier"))
		require.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "invalid_grant", decodeMap(t, rr.Body.Bytes())["error"])
	})

	t.Run("reused authorization code", func(t *testing.T) {
		verifier, challenge := mcpPKCE(t)
		code := obtainCode(t, m, clientID, challenge, admin.Email)
		form := authCodeForm(code, clientID, verifier)

		require.Equal(t, http.StatusOK, tokenRequest(m, form).Code)

		rr := tokenRequest(m, form)
		require.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "invalid_grant", decodeMap(t, rr.Body.Bytes())["error"])
	})
}

// TestMCP_AuthorizeRejections covers authorize-endpoint rejections that must never
// issue a code.
func TestMCP_AuthorizeRejections(t *testing.T) {
	m, _, _ := newMCPMux(t)
	clientID := registerClient(t, m)

	t.Run("plain challenge method bounces error to client", func(t *testing.T) {
		_, challenge := mcpPKCE(t)
		q := authorizeQuery(clientID, mcpRedirectURI, challenge, "plain")
		q.Set("state", "st")

		rr := authorizeGet(m, q)
		// redirect_uri is registered, so the protocol error bounces back as a redirect
		require.Equal(t, http.StatusFound, rr.Code)
		loc, err := url.Parse(rr.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "invalid_request", loc.Query().Get("error"))
		assert.Empty(t, loc.Query().Get("code"))
	})

	t.Run("unregistered redirect_uri renders error page", func(t *testing.T) {
		_, challenge := mcpPKCE(t)
		q := authorizeQuery(clientID, "https://attacker.example.com/cb", challenge, "S256")

		rr := authorizeGet(m, q)
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Empty(t, rr.Header().Get("Location"))
	})
}

// TestMCP_NonAdminEndToEnd walks the full OAuth 2.1 + MCP flow for a verified
// non-admin player and asserts the per-tool scoping: self access and list_game_types
// succeed, cross-player and admin-only tools are rejected with tool errors, and the
// refresh grant still works for a non-admin.
func TestMCP_NonAdminEndToEnd(t *testing.T) {
	m, _, _ := newMCPMux(t)
	nonAdmin := seedMCPPlayer(t, false)
	other := seedMCPPlayer(t, false)
	clientID := registerClient(t, m)

	// full OAuth flow as the non-admin
	verifier, challenge := mcpPKCE(t)
	code := obtainCode(t, m, clientID, challenge, nonAdmin.Email)
	tokRR := tokenRequest(m, authCodeForm(code, clientID, verifier))
	require.Equal(t, http.StatusOK, tokRR.Code)
	tok := decodeMap(t, tokRR.Body.Bytes())
	accessToken, _ := tok["access_token"].(string)
	refreshToken, _ := tok["refresh_token"].(string)
	require.NotEmpty(t, accessToken)
	require.NotEmpty(t, refreshToken)

	// get_player with own id succeeds and their own email is visible to self
	selfRR := mcpPost(m, accessToken, fmt.Sprintf(rpcGetPlayerFmt, nonAdmin.ID))
	require.Equal(t, http.StatusOK, selfRR.Code)
	selfPayload := mcpPayload(t, selfRR.Body.String())
	assert.NotContains(t, selfPayload, `"isError":true`)
	assert.Contains(t, selfPayload, nonAdmin.Email)

	// get_player with the other player's id is denied
	otherRR := mcpPost(m, accessToken, fmt.Sprintf(rpcGetPlayerFmt, other.ID))
	require.Equal(t, http.StatusOK, otherRR.Code)
	otherPayload := mcpPayload(t, otherRR.Body.String())
	assert.Contains(t, otherPayload, `"isError":true`)
	assert.Contains(t, otherPayload, errTextPermissionDenied)

	// list_players is admin only
	listRR := mcpPost(m, accessToken, rpcListPlayers)
	require.Equal(t, http.StatusOK, listRR.Code)
	listPayload := mcpPayload(t, listRR.Body.String())
	assert.Contains(t, listPayload, `"isError":true`)
	assert.Contains(t, listPayload, errTextRequiresAdmin)

	// whoami identifies the token's own player, email included, without an argument
	whoRR := mcpPost(m, accessToken, rpcWhoami)
	require.Equal(t, http.StatusOK, whoRR.Code)
	whoPayload := mcpPayload(t, whoRR.Body.String())
	assert.NotContains(t, whoPayload, `"isError":true`)
	assert.Contains(t, whoPayload, nonAdmin.Email)
	assert.NotContains(t, whoPayload, other.Email)

	// list_game_types is open to everyone
	gtRR := mcpPost(m, accessToken, rpcListGameTypes)
	require.Equal(t, http.StatusOK, gtRR.Code)
	assertContainsAll(t, mcpPayload(t, gtRR.Body.String()), allGameSlugs)

	// refresh grant is allowed for non-admins and the new tokens work
	refreshForm := url.Values{
		"grant_type":    {grantRefresh},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}
	refRR := tokenRequest(m, refreshForm)
	require.Equal(t, http.StatusOK, refRR.Code)
	refBody := decodeMap(t, refRR.Body.Bytes())
	newAccessToken, _ := refBody["access_token"].(string)
	require.NotEmpty(t, newAccessToken)

	newTokRR := mcpPost(m, newAccessToken, rpcListGameTypes)
	require.Equal(t, http.StatusOK, newTokRR.Code)
	assertContainsAll(t, mcpPayload(t, newTokRR.Body.String()), allGameSlugs)
}
