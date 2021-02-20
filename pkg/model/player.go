package model

import (
	"context"
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"mondaynightpoker-server/internal/util"
	"mondaynightpoker-server/pkg/db"
	"mondaynightpoker-server/pkg/token"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/synacor/argon2id"
)

const passwordResetRequestTTL = time.Hour

const (
	tokenTypePasswordReset       = "password_reset"
	tokenTypeAccountVerification = "account_verification" // nolint
)

const playerColumns = `
players.id,
players.email,
players.display_name,
players.is_site_admin,
players.status,
players.password_hash,
players.created,
players.updated`

const pqDuplicateKeyErrorCode pq.ErrorCode = "23505"

// ErrInvalidEmailOrPassword is an error for an invalid email or password
var ErrInvalidEmailOrPassword = UserError("invalid email address and/or password")

// ErrDuplicateKey happens if a user tries to create a player with a taken email
var ErrDuplicateKey = errors.New("duplicate key constraint violation")

// ErrTokenExpired is an error if the password reset request is no longer valid
var ErrTokenExpired = errors.New("token is expired")

// ErrAccountNotVerified is an error if the user tries to log in without being verified
var ErrAccountNotVerified = UserError("account not verified")

// PlayerStatus is the status of a player
type PlayerStatus string

// PlayerStatus constants
const (
	PlayerStatusCreated  PlayerStatus = "created"
	PlayerStatusVerified PlayerStatus = "verified"
	PlayerStatusBlocked  PlayerStatus = "blocked"
	PlayerStatusDeleted  PlayerStatus = "deleted"
)

// Player is a record in the `players` table
type Player struct {
	ID           int64        `json:"id"`
	Email        string       `json:"-"`
	DisplayName  string       `json:"displayName"`
	IsSiteAdmin  bool         `json:"isSiteAdmin"`
	Status       PlayerStatus `json:"status"`
	passwordHash string
	Created      time.Time `json:"created"`
	Updated      time.Time `json:"updated"`
}

// TableWithBalance extends the Table object to include the player's balance
type TableWithBalance struct {
	*Table
	Balance int `json:"balance"`
}

func getPlayerByRow(row db.Scanner) (*Player, error) {
	var player Player
	if err := row.Scan(&player.ID, &player.Email, &player.DisplayName, &player.IsSiteAdmin, &player.Status, &player.passwordHash, &player.Created, &player.Updated); err != nil {
		return nil, err
	}

	return &player, nil
}

// GetPlayerByID returns player based on the ID
func GetPlayerByID(ctx context.Context, id int64) (*Player, error) {
	const query = `
SELECT ` + playerColumns + `
FROM players
WHERE id = $1`

	row := db.Instance().QueryRowContext(ctx, query, id)
	return getPlayerByRow(row)
}

// Save will persist any changes made to the user to the database
func (p *Player) Save(ctx context.Context) error {
	const query = `
UPDATE players
SET email = $1,
    password_hash = $2,
    display_name = $3,
    is_site_admin = $4,
    status = $5,
    updated = (NOW() AT TIME ZONE 'utc')
WHERE id = $6`

	_, err := db.Instance().ExecContext(ctx, query, p.Email, p.passwordHash, p.DisplayName, p.IsSiteAdmin, p.Status, p.ID)
	return err
}

// GetPlayerByEmail will return a user by the email address
func GetPlayerByEmail(ctx context.Context, email string) (*Player, error) {
	const query = `
SELECT ` + playerColumns + `
FROM players
WHERE lower(email) = Lower($1)`

	row := db.Instance().QueryRowContext(ctx, query, email)
	return getPlayerByRow(row)
}

// GetPlayerByEmailAndPassword will return a user if the email and password are valid
func GetPlayerByEmailAndPassword(ctx context.Context, email, password string) (*Player, error) {
	player, err := GetPlayerByEmail(ctx, email)
	if err != nil {
		if err == sql.ErrNoRows {
			// prevent timing attacks
			_ = argon2id.Compare("", "")
			return nil, ErrInvalidEmailOrPassword
		}

		return nil, err
	}

	if err := player.ValidatePassword(password); err != nil {
		return nil, err
	}

	if player.Status != PlayerStatusVerified {
		return nil, ErrAccountNotVerified
	}

	return player, nil
}

// ValidatePassword will validate a user's password
// Returns nil if the password is valid
func (p *Player) ValidatePassword(password string) error {
	if err := argon2id.Compare(p.passwordHash, password); err != nil {
		return ErrInvalidEmailOrPassword
	}

	return nil
}

// LastPlayerCreatedAt returns the last time a player was created by the remote address
// If a player hasn't been created yet, this will return a nil error and a time.Time{} object (i.e., zero)
func LastPlayerCreatedAt(ctx context.Context, remoteAddr string) (time.Time, error) {
	const query = `
SELECT MAX(created)
FROM players
WHERE remote_addr = $1`

	var created sql.NullTime
	if err := db.Instance().QueryRowContext(ctx, query, remoteAddr).Scan(&created); err != nil {
		return time.Time{}, err
	}

	return created.Time, nil
}

// CreatePlayer creates a new player
func CreatePlayer(ctx context.Context, email, displayName, password, remoteAddr string) (*Player, error) {
	hashPassword, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return nil, err
	}

	const query = `
INSERT INTO players (email, display_name, password_hash, remote_addr)
VALUES ($1, $2, $3, $4)
RETURNING ` + playerColumns

	row := db.Instance().QueryRowContext(ctx, query, email, displayName, hashPassword, remoteAddr)
	player, err := getPlayerByRow(row)
	if err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code == pqDuplicateKeyErrorCode {
			return nil, ErrDuplicateKey
		}

		return nil, err
	}

	return player, nil
}

// SetPassword will set a new password on the player instance
// Important: you must call Save() to persist this change
func (p *Player) SetPassword(password string) error {
	newHash, err := argon2id.DefaultHashPassword(password)
	if err != nil {
		return err
	}

	p.passwordHash = newHash
	return nil
}

// GetPlayerTable gets the PlayerTable record from for the associated table
func (p *Player) GetPlayerTable(ctx context.Context, table *Table) (*PlayerTable, error) {
	const query = `
SELECT ` + playerColumns + `, ` + playerTableColumns + `
FROM players_tables
INNER JOIN players ON players_tables.player_id = players.id
WHERE players_tables.player_id = $1 AND players_tables.table_uuid = $2`

	row := db.Instance().QueryRowContext(ctx, query, p.ID, table.UUID)
	pt, err := getPlayerTableByRow(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPlayerNotAtTable
		}
		return nil, err
	}

	return pt, nil
}

// Join joins the table
func (p *Player) Join(ctx context.Context, table *Table) (*PlayerTable, error) {
	const query = `
WITH pt AS (
	INSERT INTO players_tables (player_id, table_uuid)
	VALUES ($1, $2)
	RETURNING *
)
SELECT ` + playerColumns + `, ` + playerTableColumns + `
FROM pt AS players_tables
INNER JOIN players ON players_tables.player_id = players.id
`
	row := db.Instance().QueryRowContext(ctx, query, p.ID, table.UUID)

	pt, err := getPlayerTableByRow(row)
	if err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code == pqDuplicateKeyErrorCode {
			return nil, ErrDuplicateKey
		}

		return nil, err
	}

	return pt, nil
}

// SetIsSiteAdmin sets whether the player is a site admin
func (p *Player) SetIsSiteAdmin(ctx context.Context, isSiteAdmin bool) error {
	if p.IsSiteAdmin == isSiteAdmin {
		return nil
	}

	const query = `
UPDATE players
SET is_site_admin = $1, updated = (NOW() AT TIME ZONE 'UTC')
WHERE id = $2
RETURNING updated`

	var updated sql.NullTime
	if err := db.Instance().QueryRowContext(ctx, query, isSiteAdmin, p.ID).Scan(&updated); err != nil {
		return err
	}

	p.IsSiteAdmin = isSiteAdmin
	p.Updated = updated.Time
	return nil
}

// GetTables returns a list of tables the player belongs to
func (p *Player) GetTables(ctx context.Context, offset int64, limit int) ([]*TableWithBalance, error) {
	const query = `
SELECT ` + tableColumns + `, players_tables.balance
FROM tables
INNER JOIN players_tables ON tables.uuid = players_tables.table_uuid
WHERE players_tables.player_id = $1
ORDER BY players_tables.id DESC
OFFSET $2
LIMIT $3`

	rows, err := db.Instance().QueryContext(ctx, query, p.ID, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := make([]*TableWithBalance, 0)
	for rows.Next() {
		var balance int
		tbl, err := getTableByRow(rows, &balance)
		if err != nil {
			return nil, err
		}

		records = append(records, &TableWithBalance{
			Table:   tbl,
			Balance: balance,
		})
	}

	return records, nil
}

func getPlayers(rows *sql.Rows, err error) ([]*Player, error) {
	if err != nil {
		return nil, err
	}

	players := make([]*Player, 0)
	for rows.Next() {
		player, err := getPlayerByRow(rows)
		if err != nil {
			return nil, err
		}

		players = append(players, player)
	}

	return players, nil
}

// GetPlayersWithSearch will return a list of players match the specified search string
func GetPlayersWithSearch(ctx context.Context, search string, offset int64, limit int) ([]*Player, error) {
	if search == "" {
		return GetPlayers(ctx, offset, limit)
	}

	if searchInt, _ := strconv.ParseInt(search, 10, 64); searchInt > 0 {
		const query = `
SELECT ` + playerColumns + `
FROM players
WHERE id = $1`

		return getPlayers(db.Instance().QueryContext(ctx, query, searchInt))
	}

	const query = `
SELECT ` + playerColumns + `
FROM players
WHERE display_name LIKE $1 || '%' OR email LIKE $1 || '%'
ORDER BY id ASC
OFFSET $2
LIMIT $3`

	return getPlayers(db.Instance().QueryContext(ctx, query, search, offset, limit))
}

// GetPlayers returns a list of players
func GetPlayers(ctx context.Context, offset int64, limit int) ([]*Player, error) {
	const query = `
SELECT ` + playerColumns + `
FROM players
ORDER BY id ASC
OFFSET $1
LIMIT $2`

	return getPlayers(db.Instance().QueryContext(ctx, query, offset, limit))
}

// CreatePasswordResetRequest generates a new password request and returns the token
func (p *Player) CreatePasswordResetRequest(ctx context.Context) (string, error) {
	if err := p.expirePlayerTokens(ctx, tokenTypePasswordReset); err != nil {
		return "", err
	}

	return p.createPlayerToken(ctx, tokenTypePasswordReset)
}

// CreateAccountVerificationToken generates a new account verification token
func (p *Player) CreateAccountVerificationToken(ctx context.Context) (string, error) {
	if err := p.expirePlayerTokens(ctx, tokenTypeAccountVerification); err != nil {
		return "", err
	}

	return p.createPlayerToken(ctx, tokenTypeAccountVerification)
}

// createPlayerToken creates a new player token
func (p *Player) createPlayerToken(ctx context.Context, tokenType string) (string, error) {
	const query = `
INSERT INTO player_tokens (token, player_id, type)
VALUES ($1, $2, $3)`

	resetToken, err := token.Generate(20)
	if err != nil {
		return "", err
	}

	if _, err := db.Instance().ExecContext(ctx, query, resetToken, p.ID, tokenType); err != nil {
		return "", err
	}

	return resetToken, nil
}

// expirePlayerTokens ensures all existing password requests are disabled
func (p *Player) expirePlayerTokens(ctx context.Context, tokenType string) error {
	const query = `
UPDATE player_tokens
SET active = 'f'
WHERE player_id = $1 AND type = $2`

	_, err := db.Instance().ExecContext(ctx, query, p.ID, tokenType)
	return err
}

// ResetPassword will attempt to reset the player's password
func (p *Player) ResetPassword(ctx context.Context, newPassword, resetToken string) error {
	newPasswordHash, err := argon2id.DefaultHashPassword(newPassword)
	if err != nil {
		return err
	}

	const query = `
SELECT reset_password
FROM reset_password($1, $2, $3, $4)`

	row := db.Instance().QueryRowContext(ctx, query, p.ID, newPasswordHash, resetToken, time.Now().In(time.UTC).Add(-1*passwordResetRequestTTL))

	var ok bool
	if err := row.Scan(&ok); err != nil {
		return err
	}

	if !ok {
		return errors.New("could not reset the password")
	}

	return nil
}

// Delete will mark a player as deleted
// The player isn't actually deleted from the database, but their email is destroyed and their password is changed
func (p *Player) Delete(ctx context.Context) error {
	newDisplayName := util.GetRandomName()
	newEmail := uuid.New().String() + "@deleted.monday-night.poker"

	p.DisplayName = newDisplayName
	p.Email = newEmail
	p.Status = PlayerStatusDeleted
	if err := p.Save(ctx); err != nil {
		return err
	}

	return p.SetPassword(uuid.New().String())
}

// IsPasswordResetTokenValid will return an error if the token is not valid
func IsPasswordResetTokenValid(ctx context.Context, t string) error {
	_, err := isPlayerTokenValid(ctx, t, tokenTypePasswordReset, time.Now().In(time.UTC).Add(-1*passwordResetRequestTTL))
	return err
}

// isPlayerTokenValid checks if the token is still valid
func isPlayerTokenValid(ctx context.Context, playerToken, expectedType string, createdAfter time.Time) (int64, error) {
	const query = `
SELECT player_id, type, created
FROM player_tokens
WHERE token = $1
  AND active`

	row := db.Instance().QueryRowContext(ctx, query, playerToken)

	var playerID int64
	var tokenType string
	var created time.Time
	if err := row.Scan(&playerID, &tokenType, &created); err != nil {
		return 0, ErrTokenExpired
	}

	if tokenType != expectedType || created.Before(createdAfter) {
		return 0, ErrTokenExpired
	}

	return playerID, nil
}

// VerifyAccount will verify the account if the token is valid
func VerifyAccount(ctx context.Context, verifyToken string) error {
	playerID, err := isPlayerTokenValid(ctx, verifyToken, tokenTypeAccountVerification, time.Time{})
	if err != nil {
		return err
	}

	player, err := GetPlayerByID(ctx, playerID)
	if err != nil {
		return err
	}

	if player.Status != PlayerStatusCreated {
		return errors.New("player cannot be verified")
	}
	if err := expireToken(ctx, verifyToken); err != nil {
		return err
	}

	player.Status = PlayerStatusVerified
	return player.Save(ctx)
}

func expireToken(ctx context.Context, t string) error {
	const query = `
UPDATE player_tokens
SET active = 'f'
WHERE token = $1`

	_, err := db.Instance().ExecContext(ctx, query, t)
	return err
}
