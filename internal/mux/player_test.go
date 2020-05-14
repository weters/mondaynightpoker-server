package mux

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"os"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/table"
	"testing"
	"time"
)

func Test_postPlayer(t *testing.T) {
	m := NewMux("")
	m.config.playerCreateDelay = time.Second * -1

	ts := httptest.NewServer(m)
	defer ts.Close()

	var obj errorResponse
	assertPost(t, ts, "/player", "{}", &obj, 400)
	assert.Equal(t, "missing or invalid email address", obj.Message)

	obj = errorResponse{}
	assertPost(t, ts, "/player", playerPayload{
		DisplayName: "&",
		Email:    "",
		Password: "",
	}, &obj, 400)
	assert.Equal(t, "display name must only contain letters, numbers, and spaces", obj.Message)

	email := util.RandomEmail()
	obj = errorResponse{}
	assertPost(t, ts, "/player", playerPayload{
		Email:    email,
		Password: "",
	}, &obj, 400)
	assert.Equal(t, "password must be 6 or more characters", obj.Message)

	var pObj *table.Player
	assertPost(t, ts, "/player", playerPayload{
		Email:    email,
		Password: "123456",
	}, &pObj, 201)
	assert.Greater(t, pObj.ID, int64(0))
	assert.Equal(t, email, pObj.Email)
	assert.Equal(t, email, pObj.DisplayName)

	obj = errorResponse{}
	assertPost(t, ts, "/player", &playerPayload{
		Email:    email,
		Password: "123456",
	}, &obj, 400)
	assert.Equal(t, "email address is already taken", obj.Message)

	// test display name
	email = util.RandomEmail()
	assertPost(t, ts, "/player", playerPayload{
		Email:    email,
		Password: "123456",
		DisplayName: "Tommy",
	}, &pObj, 201)
	assert.Greater(t, pObj.ID, int64(0))
	assert.Equal(t, email, pObj.Email)
	assert.Equal(t, "Tommy", pObj.DisplayName)

	m.config.playerCreateDelay = time.Hour
	obj = errorResponse{}
	assertPost(t, ts, "/player", playerPayload{
		Email: util.RandomEmail(),
		Password: "123456",
	}, &obj, 400)
	assert.Equal(t, "please wait before creating another player", obj.Message)
}

func Test_postPlayerAuth(t *testing.T) {
	os.Setenv("JWT_PUBLIC_KEY", "testdata/public.pem")
	os.Setenv("JWT_PRIVATE_KEY", "testdata/private.key")
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

	var resp postPlayerAuthResponse
	assertPost(t, ts, "/player/auth", playerPayload{
		Email:    email,
		Password: pw,
	}, &resp, 200)
	id, err := jwt.ValidUserID(resp.JWT)
	assert.NoError(t, err)
	assert.Equal(t, player.ID, id)
	assert.Equal(t, email, player.Email)

	var playerObj *table.Player
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
	assertPost(t, ts, "/player/auth", playerPayload{
		Email:    email,
		Password: "bad-password",
	}, &errObj, 401)
	assert.Equal(t, "invalid email address and/or password", errObj.Message)
}
