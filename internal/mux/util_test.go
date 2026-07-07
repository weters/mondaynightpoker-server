package mux

import (
	"context"
	"database/sql"
	"fmt"
	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/jwt"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/db"
	"mondaynightpoker-server/pkg/model"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
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

	// test maxRowsOverride
	start, rows, err = parsePaginationOptions(req("?rows=500"), 1000)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), start)
	assert.Equal(t, 500, rows)

	start, rows, err = parsePaginationOptions(req("?rows=1001"), 1000)
	assert.EqualError(t, err, "rows cannot be greater than 1000")
	assert.Equal(t, int64(0), start)
	assert.Equal(t, 0, rows)
}

var (
	testCfg    config.Config
	testDB     *sql.DB
	testRepos  *model.Repositories
	testSigner *jwt.Signer
)

func TestMain(m *testing.M) {
	os.Setenv("MNP_JWT_PUBLIC_KEY", "testdata/public.pem")
	os.Setenv("MNP_JWT_PRIVATE_KEY", "testdata/private.key")

	var err error
	if testCfg, err = config.Load(); err != nil {
		panic(err)
	}

	testDB, err = db.Connect(testCfg.Database.DSN)
	if err != nil {
		panic(err)
	}

	testRepos = model.NewRepositories(testDB)

	if testSigner, err = jwt.NewSigner(testCfg.JWT, 0); err != nil {
		panic(err)
	}

	os.Exit(m.Run())
}

// testDeps returns Deps wired to the shared test database and signer
func testDeps() Deps {
	return Deps{
		Version: "",
		Config:  testCfg,
		Repos:   testRepos,
		Tokens:  testSigner,
	}
}

func player() (*model.Player, string) {
	player, _ := testRepos.Players.CreatePlayer(context.Background(), util.RandomEmail(), "Player", "password", "")
	j, _ := testSigner.Sign(player.ID)
	return player, j
}
