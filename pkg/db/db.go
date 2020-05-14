package db

import (
	"database/sql"
	"fmt"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/sirupsen/logrus"
	"mondaynightpoker-server/internal/util"

	_ "github.com/golang-migrate/migrate/v4/source/file" // needed
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
	dsn := util.Getenv("PG_DSN", "postgres://postgres@localhost:5432/postgres?sslmode=disable")

	db, err := sql.Open("postgres", dsn)
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
	migrationsPath := util.Getenv("MIGRATIONS_PATH", "./sql")
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
