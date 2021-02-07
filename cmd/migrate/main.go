package main

import (
	"database/sql"
	"flag"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/db"
	"time"
)

var version = flag.Int("v", -1, "version to migrate to (if not specified, migrate up)")

func main() {
	flag.Parse()

	waitForDB()

	if *version >= 0 {
		db.MigrateTo(uint(*version))
	} else {
		db.Migrate()
	}
}

func waitForDB() {
	timeout := time.NewTimer(time.Second * 10)
	for {
		select {
		case <-timeout.C:
			logrus.Fatal("could not connect to database")
		default:
			dbh := func() *sql.DB {
				defer func() { _ = recover() }()
				return db.Instance()
			}()

			if dbh != nil {
				return
			}

			time.Sleep(time.Millisecond * 500)
		}
	}
}
