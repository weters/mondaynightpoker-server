package model

import (
	"database/sql"
)

// Repositories bundles the data-access repositories sharing one database handle
type Repositories struct {
	Players *PlayerRepo
	Tables  *TableRepo
	Games   *GameRepo
	OAuth   *OAuthRepo
}

// NewRepositories creates a new Repositories backed by the provided database handle
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{Players: &PlayerRepo{db: db}, Tables: &TableRepo{db: db}, Games: &GameRepo{db: db}, OAuth: &OAuthRepo{db: db}}
}

// PlayerRepo provides data access for players, player tokens, and player profiles
type PlayerRepo struct{ db *sql.DB }

// TableRepo provides data access for tables and players at tables
type TableRepo struct{ db *sql.DB }

// GameRepo provides data access for games
type GameRepo struct{ db *sql.DB }

// OAuthRepo provides data access for OAuth clients, authorization codes, and refresh tokens
type OAuthRepo struct{ db *sql.DB }
