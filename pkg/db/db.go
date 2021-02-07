package db

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // needed
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/internal/config"
)

var instance *sql.DB

// Instance returns a database instance
func Instance() *sql.DB {
	if instance == nil {
		LoadInstance()
	}

	return instance
}

// LoadInstance will load the database instance
func LoadInstance() {
	db, err := sql.Open("postgres", config.Instance().Database.DSN)
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	instance = db
}

// Migrate runs the migrations
func Migrate() {
	migrationsPath := config.Instance().Database.MigrationsPath
	db := Instance()

	logrus.WithField("migrationsPath", migrationsPath).Info("running migrations")
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file://%s", migrationsPath), "postgres", driver)
	if err != nil {
		panic(err)
	}

	if err := m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			panic(err)
		}
	}
}

// Scanner is an interface that sql should've provided
// No snark here...
type Scanner interface {
	Scan(...interface{}) error
}
