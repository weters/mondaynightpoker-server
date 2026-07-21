package mcpserver

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/internal/oauth"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/db"
	"mondaynightpoker-server/pkg/model"
)

var cbg = context.Background()

var (
	testDB    *sql.DB
	testRepos *model.Repositories
)

func TestMain(m *testing.M) {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	testDB, err = db.Connect(cfg.Database.DSN)
	if err != nil {
		panic(err)
	}

	testRepos = model.NewRepositories(testDB)

	os.Exit(m.Run())
}

// newServer returns a server wired to the test repositories, with an initialized tool
// registry so registerTool/registerTools can record policies.
func newServer() *server {
	return &server{repos: testRepos, registry: make(map[string]accessPolicy)}
}

// createPlayer creates a new verified player for testing.
func createPlayer(t *testing.T) *model.Player {
	t.Helper()

	p, err := testRepos.Players.CreatePlayer(cbg, util.RandomEmail(), "test-player", "", "127.0.0.1")
	if err != nil {
		t.Fatalf("could not create player: %v", err)
	}

	p.Status = model.PlayerStatusVerified
	if err := testRepos.Players.Save(cbg, p); err != nil {
		t.Fatalf("could not save player: %v", err)
	}

	return p
}

// createSiteAdmin creates a player who is a site admin (so tables can be
// created without hitting the cool-down).
func createSiteAdmin(t *testing.T) *model.Player {
	t.Helper()

	p := createPlayer(t)
	p.IsSiteAdmin = true
	if err := testRepos.Players.Save(cbg, p); err != nil {
		t.Fatalf("could not save site admin: %v", err)
	}

	return p
}

// adminCaller returns a site-admin Caller for the given player id.
func adminCaller(id int64) oauth.Caller {
	return oauth.Caller{PlayerID: id, IsSiteAdmin: true}
}

// playerCaller returns a non-admin Caller for the given player id.
func playerCaller(id int64) oauth.Caller {
	return oauth.Caller{PlayerID: id, IsSiteAdmin: false}
}

// ctxForAdmin returns a context carrying a site-admin Caller for the given player id.
func ctxForAdmin(id int64) context.Context {
	return oauth.ContextWithCaller(cbg, adminCaller(id))
}

// ctxForPlayer returns a context carrying a non-admin Caller for the given player id.
func ctxForPlayer(id int64) context.Context {
	return oauth.ContextWithCaller(cbg, playerCaller(id))
}
