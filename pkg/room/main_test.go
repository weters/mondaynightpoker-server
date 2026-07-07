package room

import (
	"os"
	"testing"

	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/pkg/db"
	"mondaynightpoker-server/pkg/model"
)

// testStore is the shared repository set backed by the test database
var testStore *model.Repositories

func TestMain(m *testing.M) {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	database, err := db.Connect(cfg.Database.DSN)
	if err != nil {
		panic(err)
	}

	testStore = model.NewRepositories(database)

	os.Exit(m.Run())
}
