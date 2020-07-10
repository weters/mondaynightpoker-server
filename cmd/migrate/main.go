package main

import (
	"database/sql"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/pkg/db"
	"time"
)

func main() {
	waitForDB()
	db.Migrate()
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
