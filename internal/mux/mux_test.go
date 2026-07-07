package mux

import (
	"context"
	"mondaynightpoker-server/internal/util"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_authRouter(t *testing.T) {
	m := NewMux(testDeps())

	m.authRouter.Path("/test").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, "OK")
	})

	ts := httptest.NewServer(m)
	defer ts.Close()

	var errObj errorResponse
	assertGet(t, ts, "/test", &errObj, 401)
	assert.Equal(t, "Unauthorized", errObj.Message)

	// test bad user ID
	token, _ := testSigner.Sign(0)
	errObj = errorResponse{}
	assertGet(t, ts, "/test", &errObj, 401, token)
	assert.Equal(t, "Unauthorized", errObj.Message)

	// test bad JWT
	errObj = errorResponse{}
	assertGet(t, ts, "/test", &errObj, 401, "foobar")
	assert.Equal(t, "Unauthorized", errObj.Message)

	// test using auth header
	player, _ := testRepos.Players.CreatePlayer(context.Background(), util.RandomEmail(), "x", "", "")
	token, _ = testSigner.Sign(player.ID)
	var str string
	resp := assertGetWithResp(t, ts, "/test", &str, 200, token)
	assert.Equal(t, "OK", str)
	assert.Equal(t, strconv.FormatInt(player.ID, 10), resp.Header.Get("MondayNightPoker-UserID"))
	resp.Body.Close()

	// test using query parameter
	resp = assertGetWithResp(t, ts, "/test?access_token="+url.QueryEscape(token), &str, 200)
	assert.Equal(t, "OK", str)
	assert.Equal(t, strconv.FormatInt(player.ID, 10), resp.Header.Get("MondayNightPoker-UserID"))
	resp.Body.Close()
}

func Test_adminRouter(t *testing.T) {
	m := NewMux(testDeps())

	m.adminRouter.Path("/test").HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, 200, "OK")
	})

	ts := httptest.NewServer(m)
	defer ts.Close()

	player, _ := testRepos.Players.CreatePlayer(context.Background(), util.RandomEmail(), "x", "", "")
	token, _ := testSigner.Sign(player.ID)

	var errObj errorResponse
	assertGet(t, ts, "/test", &errObj, 403, token)
	assert.Equal(t, "Forbidden", errObj.Message)

	_ = testRepos.Players.SetIsSiteAdmin(context.Background(), player, true)

	var str string
	assertGet(t, ts, "/test", &str, 200, token)
	assert.Equal(t, "OK", str)
}
