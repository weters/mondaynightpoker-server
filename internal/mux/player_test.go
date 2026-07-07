package mux

import (
	"context"
	"errors"
	"fmt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/mnptoken"
	"mondaynightpoker-server/pkg/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockRecaptcha struct {
	valid bool
	token string
}

func newMockRecaptcha(valid bool) *mockRecaptcha { return &mockRecaptcha{valid: valid} }

func (m *mockRecaptcha) Verify(token string) error {
	m.token = token

	if m.valid {
		return nil
	}

	return errors.New("token is not valid")
}

func Test_postPlayer(t *testing.T) {
	deps := testDeps()
	deps.Config.PlayerCreateDelay = 0

	m := NewMux(deps)
	mr := newMockRecaptcha(false)
	m.recaptcha = mr

	ts := httptest.NewServer(m)
	defer ts.Close()

	var obj errorResponse
	assertPost(t, ts, "/player", "{}", &obj, 400)
	assert.Equal(t, "token is not valid", obj.Message)

	obj = errorResponse{}
	assertPost(t, ts, "/player", `{"token":"bad"}`, &obj, 400)
	assert.Equal(t, "token is not valid", obj.Message)
	assert.Equal(t, "bad", mr.token)

	mr.valid = true

	obj = errorResponse{}
	assertPost(t, ts, "/player", `{"token":"good"}`, &obj, 400)
	assert.Equal(t, "missing or invalid email address", obj.Message)
	assert.Equal(t, "good", mr.token)

	obj = errorResponse{}
	assertPost(t, ts, "/player", postPlayerPayload{
		DisplayName: "&",
		Email:       "",
		Password:    "",
	}, &obj, 400)
	assert.Equal(t, "display name must only contain letters, numbers, and spaces, and be 40 characters or less", obj.Message)

	obj = errorResponse{}
	assertPost(t, ts, "/player", postPlayerPayload{
		DisplayName: strings.Repeat("A", 41),
		Email:       "",
		Password:    "",
	}, &obj, 400)
	assert.Equal(t, "display name must only contain letters, numbers, and spaces, and be 40 characters or less", obj.Message)

	email := util.RandomEmail()
	obj = errorResponse{}
	assertPost(t, ts, "/player", postPlayerPayload{
		Email:    email,
		Password: "",
	}, &obj, 400)
	assert.Equal(t, "password must be at least six characters", obj.Message)

	// test random name
	var pObj *playerWithEmail
	assertPost(t, ts, "/player", postPlayerPayload{
		Email:    email,
		Password: "123456",
	}, &pObj, 201)
	assert.Greater(t, pObj.ID, int64(0))
	assert.Equal(t, email, pObj.Email)
	assert.Equal(t, 2, len(strings.Split(pObj.DisplayName, " ")))

	obj = errorResponse{}
	assertPost(t, ts, "/player", &postPlayerPayload{
		Email:    email,
		Password: "123456",
	}, &obj, 400)
	assert.Equal(t, "email address is already taken", obj.Message)

	// test display name
	email = util.RandomEmail()
	assertPost(t, ts, "/player", postPlayerPayload{
		Email:       email,
		Password:    "123456",
		DisplayName: "Tommy",
	}, &pObj, 201)
	assert.Greater(t, pObj.ID, int64(0))
	assert.Equal(t, email, pObj.Email)
	assert.Equal(t, "Tommy", pObj.DisplayName)

	// a second server with a long create delay must reject the request because a
	// player was just created above from the same remote address
	delayDeps := testDeps()
	delayDeps.Config.PlayerCreateDelay = 3600
	delayDeps.Recaptcha = newMockRecaptcha(true)

	ts2 := httptest.NewServer(NewMux(delayDeps))
	defer ts2.Close()

	obj = errorResponse{}
	assertPost(t, ts2, "/player", postPlayerPayload{
		Email:    util.RandomEmail(),
		Password: "123456",
	}, &obj, 400)
	assert.Equal(t, "please wait before creating another player", obj.Message)
}

func Test_postPlayerID(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	player, j := player()
	player.Status = model.PlayerStatusVerified
	assert.NoError(t, testRepos.Players.Save(cbg, player))

	// playerID must match
	var errResp errorResponse
	assertPost(t, ts, "/player/0", postPlayerIDPayload{}, &errResp, http.StatusForbidden, j)

	newEmail := util.RandomEmail()
	payload := postPlayerIDPayload{
		DisplayName: "TEST",
		Email:       newEmail,
	}

	var resp map[string]interface{}
	assertPost(t, ts, fmt.Sprintf("/player/%d", player.ID), payload, &resp, http.StatusOK, j)
	assert.Equal(t, "OK", resp["status"])

	p, _ := testRepos.Players.GetPlayerByID(context.Background(), player.ID)
	assert.Equal(t, "TEST", p.DisplayName)
	assert.Equal(t, newEmail, p.Email)

	// no change OK
	resp = make(map[string]interface{})
	assertPost(t, ts, fmt.Sprintf("/player/%d", player.ID), postPlayerIDPayload{}, &resp, http.StatusOK, j)
	assert.Equal(t, "OK", resp["status"])

	// bad email
	errResp = errorResponse{}
	assertPost(t, ts, fmt.Sprintf("/player/%d", player.ID), postPlayerIDPayload{Email: "invalid"}, &errResp, http.StatusBadRequest, j)
	assert.Equal(t, "invalid email address", errResp.Message)

	// bad username
	errResp = errorResponse{}
	assertPost(t, ts, fmt.Sprintf("/player/%d", player.ID), postPlayerIDPayload{DisplayName: "!"}, &errResp, http.StatusBadRequest, j)
	assert.Equal(t, "display name must only contain letters, numbers, and spaces", errResp.Message)

	// bad password
	assertPost(t, ts, fmt.Sprintf("/player/%d", player.ID), postPlayerIDPayload{NewPassword: "bad"}, &errResp, http.StatusBadRequest, j)
	assert.Equal(t, "password must be at least six characters", errResp.Message)

	assertPost(t, ts, fmt.Sprintf("/player/%d", player.ID), postPlayerIDPayload{NewPassword: "good-password"}, &errResp, http.StatusBadRequest, j)
	assert.Equal(t, "old password does not match", errResp.Message)

	assertPost(t, ts, fmt.Sprintf("/player/%d", player.ID), postPlayerIDPayload{NewPassword: "good-password", OldPassword: "password"}, nil, http.StatusOK, j)
	newPlayer, err := testRepos.Players.GetPlayerByEmailAndPassword(context.Background(), newEmail, "good-password")
	assert.NoError(t, err)
	assert.NotNil(t, newPlayer)
}

func Test_postPlayerAuth(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	email := util.RandomEmail()
	pw := "my-password"

	player, err := testRepos.Players.CreatePlayer(context.Background(), email, email, pw, "")
	if err != nil {
		t.Error(err)
		return
	}

	player.Status = model.PlayerStatusVerified
	_ = testRepos.Players.Save(cbg, player)

	var resp postPlayerAuthResponse
	assertPost(t, ts, "/player/auth", postPlayerPayload{
		Email:    email,
		Password: pw,
	}, &resp, 200)
	id, err := testSigner.ValidUserID(resp.JWT)
	assert.NoError(t, err)
	assert.Equal(t, player.ID, id)
	assert.Equal(t, email, player.Email)

	var playerObj *playerWithEmail
	assertGet(t, ts, fmt.Sprintf("/player/auth/%s", resp.JWT), &playerObj, 200)
	assert.Equal(t, email, playerObj.Email)
}

func Test_postPlayerAuthRefresh(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p, j := player()

	var resp postPlayerAuthResponse
	assertPost(t, ts, "/player/auth/refresh", nil, &resp, http.StatusOK, j)
	assert.NotEmpty(t, resp.JWT)
	assert.NotEqual(t, j, resp.JWT, "refresh must issue a new token")

	id, err := testSigner.ValidUserID(resp.JWT)
	assert.NoError(t, err)
	assert.Equal(t, p.ID, id)
	assert.Equal(t, p.Email, resp.Player.Email)

	// requires authentication
	var errResp errorResponse
	assertPost(t, ts, "/player/auth/refresh", nil, &errResp, http.StatusUnauthorized)
}

func Test_getPlayerAuthJWT_BadRequests(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	var errObj errorResponse
	assertGet(t, ts, "/player/auth/bad", &errObj, 401)
	assert.Equal(t, "token is malformed: token contains an invalid number of segments", errObj.Message)

	// this should only happen if user is deleted from database
	signedToken, _ := testSigner.Sign(-1)
	errObj = errorResponse{}
	assertGet(t, ts, fmt.Sprintf("/player/auth/%s", signedToken), &errObj, 404)
	assert.Equal(t, "player does not exist", errObj.Message)
}

func Test_postPlayerAuth_BadCreds(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))

	email := util.RandomEmail()
	_, err := testRepos.Players.CreatePlayer(context.Background(), email, email, "my-password", "")
	if err != nil {
		t.Error(err)
		return
	}

	var errObj errorResponse
	assertPost(t, ts, "/player/auth", postPlayerPayload{
		Email:    email,
		Password: "bad-password",
	}, &errObj, 401)
	assert.Equal(t, "invalid email address and/or password", errObj.Message)
}

func Test_getPlayers(t *testing.T) {
	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p1, j1 := player()
	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), p1, true)

	_, j2 := player()
	_, _ = player()
	_, _ = player()

	assertGet(t, ts, "/player", nil, 403, j2)

	var players []*playerWithEmail
	assertGet(t, ts, "/player?start=0&rows=4", &players, 200, j1)
	assert.Equal(t, 4, len(players))
	assert.NotEmpty(t, players[0].Email)

	players = make([]*playerWithEmail, 0)
	partialEmail := p1.Email
	partialEmail = partialEmail[0 : len(partialEmail)-3]
	assertGet(t, ts, "/player?start=0&rows=4&search="+partialEmail, &players, 200, j1)
	assert.Equal(t, 1, len(players))
	assert.Equal(t, p1.Email, players[0].Email)

	var err errorResponse
	assertGet(t, ts, "/player?start=-1", &err, 400, j1)
	assert.Equal(t, "start cannot be less than zero", err.Message)
}

func TestMux_getPlayerProfile(t *testing.T) {
	a := assert.New(t)

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p, _ := player()
	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), p, true)

	for i := 1; i <= 3; i++ {
		tbl, _ := testRepos.Tables.CreateTable(context.Background(), p, fmt.Sprintf("Profile Test %d", i))

		game, _ := testRepos.Games.CreateGame(context.Background(), tbl, "Bourré")
		_ = testRepos.Games.EndGame(context.Background(), game, nil, map[int64]int{
			p.ID: i * 100,
		})
	}

	// remove site admin so we test as a regular user
	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), p, false)

	j, _ := testSigner.Sign(p.ID)

	// player can view own profile
	var profile model.PlayerProfile
	assertGet(t, ts, "/player/profile", &profile, http.StatusOK, j)
	a.Equal(p.ID, profile.Player.ID)
	a.Equal(3, profile.Stats.TablesJoined)
	a.Equal(3, profile.Stats.GamesPlayed)
	a.Equal(600, profile.Stats.TotalWinnings)
	a.Equal(600, profile.Stats.WinningsByGame["Bourre"])
	a.Equal(3, len(profile.Tables))

	// test pagination
	profile = model.PlayerProfile{}
	assertGet(t, ts, "/player/profile?start=0&rows=2", &profile, http.StatusOK, j)
	a.Equal(2, len(profile.Tables))

	// test bad pagination
	var errResp errorResponse
	assertGet(t, ts, "/player/profile?rows=0", &errResp, http.StatusBadRequest, j)

	// test bad date format
	errResp = errorResponse{}
	assertGet(t, ts, "/player/profile?from=bad-date", &errResp, http.StatusBadRequest, j)
	a.Equal("invalid 'from' date format, use ISO 8601", errResp.Message)

	// test unauthenticated
	assertGet(t, ts, "/player/profile", nil, http.StatusUnauthorized)
}

func TestMux_getPlayerIDProfile(t *testing.T) {
	a := assert.New(t)

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p, _ := player()
	p2, _ := player()

	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), p, true)

	for i := 1; i <= 3; i++ {
		tbl, _ := testRepos.Tables.CreateTable(context.Background(), p, fmt.Sprintf("Profile Test %d", i))
		_, _ = testRepos.Tables.Join(context.Background(), p2, tbl)

		game, _ := testRepos.Games.CreateGame(context.Background(), tbl, "Bourré")
		_ = testRepos.Games.EndGame(context.Background(), game, nil, map[int64]int{
			p.ID:  i * 100,
			p2.ID: -1 * i * 100,
		})
	}

	j, _ := testSigner.Sign(p.ID)
	j2, _ := testSigner.Sign(p2.ID)

	// admin can view any profile
	path := fmt.Sprintf("/player/%d/profile", p.ID)
	var profile model.PlayerProfile
	assertGet(t, ts, path, &profile, http.StatusOK, j)
	a.Equal(p.ID, profile.Player.ID)
	a.Equal(3, profile.Stats.TablesJoined)
	a.Equal(3, profile.Stats.GamesPlayed)
	a.Equal(600, profile.Stats.TotalWinnings)
	a.Equal(600, profile.Stats.WinningsByGame["Bourre"])
	a.Equal(3, len(profile.Tables))

	// admin can view another player's profile
	path = fmt.Sprintf("/player/%d/profile", p2.ID)
	profile = model.PlayerProfile{}
	assertGet(t, ts, path, &profile, http.StatusOK, j)
	a.Equal(p2.ID, profile.Player.ID)

	// non-admin cannot view any profile
	assertGet(t, ts, path, nil, http.StatusForbidden, j2)
	assertGet(t, ts, fmt.Sprintf("/player/%d/profile", p2.ID), nil, http.StatusForbidden, j2)

	// test pagination
	path = fmt.Sprintf("/player/%d/profile", p.ID)
	profile = model.PlayerProfile{}
	assertGet(t, ts, path+"?start=0&rows=2", &profile, http.StatusOK, j)
	a.Equal(2, len(profile.Tables))

	// test non-existent player
	assertGet(t, ts, "/player/0/profile", nil, http.StatusNotFound, j)

	// test bad pagination
	var errResp errorResponse
	assertGet(t, ts, path+"?rows=0", &errResp, http.StatusBadRequest, j)

	// test bad date format
	errResp = errorResponse{}
	assertGet(t, ts, path+"?from=bad-date", &errResp, http.StatusBadRequest, j)
	a.Equal("invalid 'from' date format, use ISO 8601", errResp.Message)

	// test unauthenticated
	assertGet(t, ts, path, nil, http.StatusUnauthorized)
}

func TestMux_postAdminTestPlayer(t *testing.T) {
	a := assert.New(t)

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	admin, adminJWT := player()
	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), admin, true)

	_, nonAdminJWT := player()

	// non-admin gets 403
	assertPost(t, ts, "/admin/test-player", postAdminTestPlayerPayload{
		Email:    "test@example.com",
		Password: "123456",
	}, nil, http.StatusForbidden, nonAdminJWT)

	// missing email returns 400
	var errResp errorResponse
	assertPost(t, ts, "/admin/test-player", postAdminTestPlayerPayload{
		Password: "123456",
	}, &errResp, http.StatusBadRequest, adminJWT)
	a.Equal("email is required", errResp.Message)

	// missing password returns 400
	errResp = errorResponse{}
	assertPost(t, ts, "/admin/test-player", postAdminTestPlayerPayload{
		Email: util.RandomEmail(),
	}, &errResp, http.StatusBadRequest, adminJWT)
	a.Equal("password must be at least six characters", errResp.Message)

	// admin can create test player (returns verified status)
	email := util.RandomEmail()
	var resp postAdminTestPlayerResponse
	assertPost(t, ts, "/admin/test-player", postAdminTestPlayerPayload{
		DisplayName: "TestBot",
		Email:       email,
		Password:    "123456",
	}, &resp, http.StatusCreated, adminJWT)
	a.Greater(resp.PlayerID, int64(0))
	a.Equal(email, resp.Email)

	// verify the player is actually verified (can log in)
	p, err := testRepos.Players.GetPlayerByEmailAndPassword(context.Background(), email, "123456")
	a.NoError(err)
	a.NotNil(p)
	a.Equal(model.PlayerStatusVerified, p.Status)
	a.Equal("TestBot", p.DisplayName)

	// test with auto-generated display name
	email2 := util.RandomEmail()
	var resp2 postAdminTestPlayerResponse
	assertPost(t, ts, "/admin/test-player", postAdminTestPlayerPayload{
		Email:    email2,
		Password: "123456",
	}, &resp2, http.StatusCreated, adminJWT)
	a.Greater(resp2.PlayerID, int64(0))

	p2, err := testRepos.Players.GetPlayerByID(context.Background(), resp2.PlayerID)
	a.NoError(err)
	a.NotEmpty(p2.DisplayName)

	// duplicate email returns 400
	errResp = errorResponse{}
	assertPost(t, ts, "/admin/test-player", postAdminTestPlayerPayload{
		Email:    email,
		Password: "123456",
	}, &errResp, http.StatusBadRequest, adminJWT)
	a.Equal("email address is already taken", errResp.Message)
}

func TestMux_postAdminPlayerID(t *testing.T) {
	a := assert.New(t)

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	p1, j1 := player()
	p2, j2 := player()

	p1.Status = model.PlayerStatusVerified
	a.NoError(testRepos.Players.Save(cbg, p1))

	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), p1, true)

	_, err := testRepos.Players.GetPlayerByEmailAndPassword(cbg, p1.Email, "new-pw")
	a.EqualError(err, "invalid email address and/or password")

	var respObj map[string]string
	assertPost(t, ts, fmt.Sprintf("/admin/player/%d", p1.ID), adminPostPlayerIDRequest{
		Key:   "password",
		Value: "new-pw",
	}, &respObj, http.StatusOK, j1)
	a.Equal("OK", respObj["status"])

	// verify password is changed
	_, err = testRepos.Players.GetPlayerByEmailAndPassword(cbg, p1.Email, "new-pw")
	a.NoError(err)

	respObj = map[string]string{}
	assertPost(t, ts, fmt.Sprintf("/admin/player/%d", p2.ID), adminPostPlayerIDRequest{
		Key:   "password",
		Value: "new-pw",
	}, &respObj, http.StatusOK, j1)
	a.Equal("OK", respObj["status"])

	var errResp errorResponse
	assertPost(t, ts, fmt.Sprintf("/admin/player/%d", p1.ID), map[string]string{}, &errResp, http.StatusBadRequest, j1)
	a.Equal(errorResponse{
		Message:    "bad payload",
		StatusCode: http.StatusBadRequest,
	}, errResp)

	assertPost(t, ts, fmt.Sprintf("/admin/player/%d", p1.ID), adminPostPlayerIDRequest{
		Key:   "password",
		Value: "new-pw",
	}, nil, http.StatusForbidden, j2)
}

func TestMux_postPlayerResetPasswordRequest(t *testing.T) {
	a := assert.New(t)

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	var er errorResponse
	assertPost(t, ts, "/player/reset-password-request", postPlayerResetPasswordRequestPayload{}, &er, http.StatusBadRequest)
	a.Equal("missing email", er.Message)

	p, _ := player()
	assertPost(t, ts, "/player/reset-password-request", postPlayerResetPasswordRequestPayload{Email: p.Email}, nil, http.StatusOK)

	p.Status = model.PlayerStatusVerified
	_ = testRepos.Players.Save(cbg, p)

	row := testDB.QueryRow("SELECT token FROM player_tokens WHERE player_id = $1 ORDER BY created DESC LIMIT 1", p.ID)
	var resetToken string
	a.NoError(row.Scan(&resetToken))

	diffToken, err := mnptoken.Generate(20)
	a.NoError(err)

	assertGet(t, ts, "/player/reset-password/"+resetToken, nil, http.StatusOK)
	assertGet(t, ts, "/player/reset-password/"+diffToken, nil, http.StatusNotFound)

	assertPost(t, ts, "/player/reset-password/"+resetToken, postPlayerResetPasswordPayload{
		Email:    "",
		Password: "",
	}, &er, http.StatusBadRequest)
	a.Equal("email is required", er.Message)

	assertPost(t, ts, "/player/reset-password/"+resetToken, postPlayerResetPasswordPayload{
		Email:    p.Email,
		Password: "12345",
	}, &er, http.StatusBadRequest)
	a.Equal("password must be at least six characters", er.Message)

	diffPlayer, _ := player()
	assertPost(t, ts, "/player/reset-password/"+resetToken, postPlayerResetPasswordPayload{
		Email:    diffPlayer.Email,
		Password: "123456",
	}, nil, http.StatusBadRequest)

	assertPost(t, ts, "/player/reset-password/"+resetToken, postPlayerResetPasswordPayload{
		Email:    p.Email + "unknown",
		Password: "123456",
	}, nil, http.StatusBadRequest)

	assertPost(t, ts, "/player/reset-password/"+diffToken, postPlayerResetPasswordPayload{
		Email:    p.Email,
		Password: "123456",
	}, nil, http.StatusNotFound)

	assertPost(t, ts, "/player/reset-password/"+resetToken, postPlayerResetPasswordPayload{
		Email:    p.Email,
		Password: "123456",
	}, nil, http.StatusOK)

	assertPost(t, ts, "/player/auth", map[string]string{
		"email":    p.Email,
		"password": "123456",
	}, nil, http.StatusOK)
}

func TestMux_accountVerification(t *testing.T) {
	a := assert.New(t)

	m := NewMux(testDeps())
	m.recaptcha = newMockRecaptcha(true)

	ts := httptest.NewServer(m)
	defer ts.Close()

	email := util.RandomEmail()
	password := "my-password"
	assertPost(t, ts, "/player", postPlayerPayload{
		DisplayName: "Test Name",
		Email:       email,
		Password:    password,
	}, nil, http.StatusCreated)

	var er errorResponse
	assertPost(t, ts, "/player/auth", map[string]string{
		"email":    email,
		"password": password,
	}, &er, http.StatusUnauthorized)
	a.Equal("account not verified", er.Message)

	player, err := testRepos.Players.GetPlayerByEmail(context.Background(), email)
	a.NoError(err)

	row := testDB.QueryRow("SELECT token FROM player_tokens WHERE player_id = $1 AND type = 'account_verification'", player.ID)
	var verifyToken string
	a.NoError(row.Scan(&verifyToken))

	badToken, _ := mnptoken.Generate(20)
	assertPost(t, ts, "/player/verify/"+badToken, nil, nil, http.StatusBadRequest)
	assertPost(t, ts, "/player/verify/"+verifyToken, nil, nil, http.StatusOK)

	assertPost(t, ts, "/player/auth", map[string]string{
		"email":    email,
		"password": password,
	}, &er, http.StatusOK)

	// can't re-use
	assertPost(t, ts, "/player/verify/"+verifyToken, nil, nil, http.StatusBadRequest)
}

func TestMux_deletePlayerID(t *testing.T) {
	p1, j1 := player()
	_, j2 := player()

	ts := httptest.NewServer(NewMux(testDeps()))
	defer ts.Close()

	assertDelete(t, ts, fmt.Sprintf("/player/%d", p1.ID), nil, http.StatusForbidden, j2)
	assertDelete(t, ts, fmt.Sprintf("/player/%d", p1.ID), nil, http.StatusOK, j1)

	p, err := testRepos.Players.GetPlayerByID(cbg, p1.ID)
	a := assert.New(t)
	a.NoError(err)
	a.NotEqual(p1.Email, p.Email)
	a.NotEqual(p1.DisplayName, p.DisplayName)
}
