package mux

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/db"
	"mondaynightpoker-server/pkg/table"
	"mondaynightpoker-server/pkg/token"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
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
	unset := util.SetEnv("MNP_PLAYER_CREATE_DELAY", "-1")
	defer unset()

	assert.NoError(t, config.Load())

	m := NewMux("")
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
	rand.Seed(0)
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

	unset2 := util.SetEnv("MNP_PLAYER_CREATE_DELAY", "3600")
	defer unset2()

	assert.NoError(t, config.Load())

	obj = errorResponse{}
	assertPost(t, ts, "/player", postPlayerPayload{
		Email:    util.RandomEmail(),
		Password: "123456",
	}, &obj, 400)
	assert.Equal(t, "please wait before creating another player", obj.Message)
}

func Test_postPlayerID(t *testing.T) {
	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	player, j := player()
	player.Status = table.PlayerStatusVerified
	assert.NoError(t, player.Save(cbg))

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

	p, _ := table.GetPlayerByID(context.Background(), player.ID)
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
	newPlayer, err := table.GetPlayerByEmailAndPassword(context.Background(), newEmail, "good-password")
	assert.NoError(t, err)
	assert.NotNil(t, newPlayer)
}

func Test_postPlayerAuth(t *testing.T) {
	os.Setenv("MNP_JWT_PUBLIC_KEY", "testdata/public.pem")
	os.Setenv("MNP_JWT_PRIVATE_KEY", "testdata/private.key")
	jwt.LoadKeys()

	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	email := util.RandomEmail()
	pw := "my-password"

	player, err := table.CreatePlayer(context.Background(), email, email, pw, "")
	if err != nil {
		t.Error(err)
		return
	}

	player.Status = table.PlayerStatusVerified
	_ = player.Save(cbg)

	var resp postPlayerAuthResponse
	assertPost(t, ts, "/player/auth", postPlayerPayload{
		Email:    email,
		Password: pw,
	}, &resp, 200)
	id, err := jwt.ValidUserID(resp.JWT)
	assert.NoError(t, err)
	assert.Equal(t, player.ID, id)
	assert.Equal(t, email, player.Email)

	var playerObj *playerWithEmail
	assertGet(t, ts, fmt.Sprintf("/player/auth/%s", resp.JWT), &playerObj, 200)
	assert.Equal(t, email, playerObj.Email)
}

func Test_getPlayerAuthJWT_BadRequests(t *testing.T) {
	os.Setenv("JWT_PUBLIC_KEY", "testdata/public.pem")
	os.Setenv("JWT_PRIVATE_KEY", "testdata/private.key")
	jwt.LoadKeys()

	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	var errObj errorResponse
	assertGet(t, ts, "/player/auth/bad", &errObj, 401)
	assert.Equal(t, "token contains an invalid number of segments", errObj.Message)

	// this should only happen if user is deleted from database
	signedToken, _ := jwt.Sign(-1)
	errObj = errorResponse{}
	assertGet(t, ts, fmt.Sprintf("/player/auth/%s", signedToken), &errObj, 404)
	assert.Equal(t, "player does not exist", errObj.Message)
}

func Test_postPlayerAuth_BadCreds(t *testing.T) {
	ts := httptest.NewServer(NewMux(""))

	email := util.RandomEmail()
	_, err := table.CreatePlayer(context.Background(), email, email, "my-password", "")
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
	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	p1, j1 := player()
	_ = p1.SetIsSiteAdmin(context.Background(), true)

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

func TestMux_getPlayerIDTable(t *testing.T) {
	a := assert.New(t)

	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	p, _ := player()
	p2, _ := player()

	_ = p.SetIsSiteAdmin(context.Background(), true)

	for i := 1; i <= 3; i++ {
		tbl, _ := p.CreateTable(context.Background(), fmt.Sprintf("Test %d", i))
		_, _ = p2.Join(context.Background(), tbl)

		game, _ := tbl.CreateGame(context.Background(), fmt.Sprintf("test-%d", i))
		_ = game.EndGame(context.Background(), nil, map[int64]int{
			p.ID:  i,
			p2.ID: -1 * i,
		})
	}

	j, _ := jwt.Sign(p.ID)

	path := fmt.Sprintf("/player/%d/table", p.ID)
	var respObj []*table.WithBalance
	assertGet(t, ts, path, &respObj, http.StatusOK, j)

	a.Equal(3, len(respObj))
	a.Equal("Test 3", respObj[0].Name)
	a.Equal(3, respObj[0].Balance)

	j2, _ := jwt.Sign(p2.ID)
	assertGet(t, ts, path, nil, http.StatusForbidden, j2)

	path = "/player/0/table"
	assertGet(t, ts, path, nil, http.StatusNotFound, j)

	path = fmt.Sprintf("/player/%d/table", p.ID)
	assertGet(t, ts, path+"?rows=0", nil, http.StatusBadRequest, j)
}

func TestMux_postAdminPlayerID(t *testing.T) {
	a := assert.New(t)

	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	p1, j1 := player()
	p2, j2 := player()

	_ = p1.SetIsSiteAdmin(context.Background(), true)

	var respObj map[string]string
	assertPost(t, ts, fmt.Sprintf("/admin/player/%d", p1.ID), adminPostPlayerIDRequest{
		Key:   "password",
		Value: "new-pw",
	}, &respObj, http.StatusOK, j1)
	a.Equal("OK", respObj["status"])

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

	setupJWT()
	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	var er errorResponse
	assertPost(t, ts, "/player/reset-password-request", postPlayerResetPasswordRequestPayload{}, &er, http.StatusBadRequest)
	a.Equal("missing email", er.Message)

	p, _ := player()
	assertPost(t, ts, "/player/reset-password-request", postPlayerResetPasswordRequestPayload{Email: p.Email}, nil, http.StatusOK)

	p.Status = table.PlayerStatusVerified
	_ = p.Save(cbg)

	row := db.Instance().QueryRow("SELECT token FROM player_tokens WHERE player_id = $1 ORDER BY created DESC LIMIT 1", p.ID)
	var resetToken string
	a.NoError(row.Scan(&resetToken))

	diffToken, err := token.Generate(20)
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

	m := NewMux("")
	m.recaptcha = newMockRecaptcha(true)

	setupJWT()
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

	player, err := table.GetPlayerByEmail(context.Background(), email)
	a.NoError(err)

	row := db.Instance().QueryRow("SELECT token FROM player_tokens WHERE player_id = $1 AND type = 'account_verification'", player.ID)
	var verifyToken string
	a.NoError(row.Scan(&verifyToken))

	badToken, _ := token.Generate(20)
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
	setupJWT()

	p1, j1 := player()
	_, j2 := player()

	ts := httptest.NewServer(NewMux(""))
	defer ts.Close()

	assertDelete(t, ts, fmt.Sprintf("/player/%d", p1.ID), nil, http.StatusForbidden, j2)
	assertDelete(t, ts, fmt.Sprintf("/player/%d", p1.ID), nil, http.StatusOK, j1)

	p, err := table.GetPlayerByID(cbg, p1.ID)
	a := assert.New(t)
	a.NoError(err)
	a.NotEqual(p1.Email, p.Email)
	a.NotEqual(p1.DisplayName, p.DisplayName)
}
