package model

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/pkg/db"
)

var cbg = context.Background()

var (
	testDB    *sql.DB
	testRepos *Repositories
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

	testRepos = NewRepositories(testDB)

	os.Exit(m.Run())
}
