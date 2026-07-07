package db

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file" // needed
	"github.com/sirupsen/logrus"
)

// Connect opens and pings a database connection
func Connect(dsn string) (*sql.DB, error) {
	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return database, nil
}

// Migrate runs all pending migrations
func Migrate(database *sql.DB, migrationsPath string) error {
	m, err := getMigrate(database, migrationsPath)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// MigrateTo migrates to the specified version
func MigrateTo(database *sql.DB, migrationsPath string, version uint) error {
	m, err := getMigrate(database, migrationsPath)
	if err != nil {
		return err
	}

	return m.Migrate(version)
}

func getMigrate(database *sql.DB, migrationsPath string) (*migrate.Migrate, error) {
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	logrus.WithField("migrationsPath", sourceURL).Info("running migrations")

	driver, err := postgres.WithInstance(database, &postgres.Config{})
	if err != nil {
		return nil, err
	}

	return migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
}

// Scanner is an interface that sql should've provided
// No snark here...
type Scanner interface {
	Scan(...interface{}) error
}
