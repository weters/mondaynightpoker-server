package model

import (
	"database/sql"
	"mondaynightpoker-server/pkg/db"
	"sync"
)

// Repositories bundles the data-access repositories sharing one database handle
type Repositories struct {
	Players *PlayerRepo
	Tables  *TableRepo
	Games   *GameRepo
}

// NewRepositories creates a new Repositories backed by the provided database handle
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{Players: &PlayerRepo{db: db}, Tables: &TableRepo{db: db}, Games: &GameRepo{db: db}}
}

// PlayerRepo provides data access for players, player tokens, and player profiles
type PlayerRepo struct{ db *sql.DB }

// TableRepo provides data access for tables and players at tables
type TableRepo struct{ db *sql.DB }

// GameRepo provides data access for games
type GameRepo struct{ db *sql.DB }

var (
	defaultRepos     *Repositories
	defaultReposOnce sync.Once
)

// deprecatedRepos backs the deprecated package-level functions and entity methods
// until all callers are migrated to injected Repositories. It will be deleted.
func deprecatedRepos() *Repositories {
	defaultReposOnce.Do(func() { defaultRepos = NewRepositories(db.Instance()) })
	return defaultRepos
}
