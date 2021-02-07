package mux

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/table"
	"net/http"
	"os"
	"testing"
)

var cbg = context.Background()

func Test_remoteAddr(t *testing.T) {
	r := &http.Request{RemoteAddr: "127.0.0.1:5000"}
	assert.Equal(t, "127.0.0.1", remoteAddr(r))

	r.RemoteAddr = "[::1]:5000"
	assert.Equal(t, "[::1]", remoteAddr(r))
}

func Test_parsePaginationOptions(t *testing.T) {
	req := func(queryString string) *http.Request {
		req, _ := http.NewRequest(http.MethodGet, "https://example.domain/"+queryString, nil)
		return req
	}

	start, rows, err := parsePaginationOptions(req(""))
	assert.NoError(t, err)
	assert.Equal(t, int64(0), start)
	assert.Equal(t, defaultRows, rows)

	start, rows, err = parsePaginationOptions(req("?start=10&rows=25"))
	assert.NoError(t, err)
	assert.Equal(t, int64(10), start)
	assert.Equal(t, 25, rows)

	start, rows, err = parsePaginationOptions(req("?start=-1&rows=25"))
	assert.EqualError(t, err, "start cannot be less than zero")
	assert.Equal(t, int64(0), start)
	assert.Equal(t, 0, rows)

	start, rows, err = parsePaginationOptions(req("?start=0&rows=0"))
	assert.EqualError(t, err, "rows must be greater than zero")
	assert.Equal(t, int64(0), start)
	assert.Equal(t, 0, rows)

	start, rows, err = parsePaginationOptions(req(fmt.Sprintf("?start=0&rows=%d", maxRows+1)))
	assert.EqualError(t, err, fmt.Sprintf("rows cannot be greater than %d", maxRows))
	assert.Equal(t, int64(0), start)
	assert.Equal(t, 0, rows)
}

func player() (*table.Player, string) {
	player, _ := table.CreatePlayer(context.Background(), util.RandomEmail(), "Player", "password", "")
	j, _ := jwt.Sign(player.ID)
	return player, j
}

func setupJWT() {
	os.Setenv("MNP_JWT_PUBLIC_KEY", "testdata/public.pem")
	os.Setenv("MNP_JWT_PRIVATE_KEY", "testdata/private.key")
	if err := config.Load(); err != nil {
		panic(err)
	}

	jwt.LoadKeys()
}
