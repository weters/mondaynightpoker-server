package main

import (
	"database/sql"
	"flag"
	"time"

	"mondaynightpoker-server/internal/config"
	"mondaynightpoker-server/pkg/db"

	"github.com/sirupsen/logrus"
)

var version = flag.Int("v", -1, "version to migrate to (if not specified, migrate up)")

func main() {
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("could not load configuration")
	}

	database := waitForDB(cfg.Database.DSN)

	if *version >= 0 {
		err = db.MigrateTo(database, cfg.Database.MigrationsPath, uint(*version))
	} else {
		err = db.Migrate(database, cfg.Database.MigrationsPath)
	}

	if err != nil {
		logrus.WithError(err).Fatal("could not run migrations")
	}
}

func waitForDB(dsn string) *sql.DB {
	timeout := time.NewTimer(time.Second * 10)
	for {
		select {
		case <-timeout.C:
			logrus.Fatal("could not connect to database")
			return nil
		default:
			if database, err := db.Connect(dsn); err == nil {
				return database
			}

			time.Sleep(time.Millisecond * 500)
		}
	}
}
