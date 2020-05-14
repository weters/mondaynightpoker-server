package mux

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/table"
	"strconv"
	"testing"
)

func Test_authRouter(t *testing.T) {
	setupJWT()
	m := NewMux("")

	m.authRouter.Path("/test").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, "OK")
	})

	ts := httptest.NewServer(m)
	defer ts.Close()

	var errObj errorResponse
	assertGet(t, ts, "/test", &errObj, 401)
	assert.Equal(t, "Unauthorized", errObj.Message)

	player, _ := table.CreatePlayer(context.Background(), util.RandomEmail(), "x", "", "")
	token, _ := jwt.Sign(player.ID)

	// test using auth header
	var str string
	resp := assertGet(t, ts, "/test", &str, 200, token)
	assert.Equal(t, "OK", str)
	assert.Equal(t, strconv.FormatInt(player.ID, 10), resp.Header.Get("MondayNightPoker-UserID"))

	// test using query parameter
	resp = assertGet(t, ts, "/test?access_token=" + url.QueryEscape(token), &str, 200)
	assert.Equal(t, "OK", str)
	assert.Equal(t, strconv.FormatInt(player.ID, 10), resp.Header.Get("MondayNightPoker-UserID"))
}

func Test_adminRouter(t *testing.T) {
	setupJWT()
	m := NewMux("")

	m.adminRouter.Path("/test").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, "OK")
	})

	ts := httptest.NewServer(m)
	defer ts.Close()

	player, _ := table.CreatePlayer(context.Background(), util.RandomEmail(), "x", "", "")
	token, _ := jwt.Sign(player.ID)

	var errObj errorResponse
	assertGet(t, ts, "/test", &errObj, 403, token)
	assert.Equal(t, "Forbidden", errObj.Message)

	_ = player.SetIsSiteAdmin(context.Background(), true)

	var str string
	assertGet(t, ts, "/test", &str, 200, token)
	assert.Equal(t, "OK", str)
}
