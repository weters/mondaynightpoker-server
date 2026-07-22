package model

import (
	"context"
	"database/sql"
	"errors"
	"mondaynightpoker-server/pkg/db"
	"time"

	"github.com/lib/pq"
)

// ErrRefreshTokenAlreadyRevoked is returned when attempting to rotate a refresh token that has
// already been revoked (e.g., reused after rotation), so callers can detect token reuse
var ErrRefreshTokenAlreadyRevoked = errors.New("refresh token has already been revoked")

const oauthClientColumns = `
oauth_clients.client_id,
oauth_clients.client_secret_hash,
oauth_clients.client_name,
oauth_clients.redirect_uris,
oauth_clients.grant_types,
oauth_clients.token_endpoint_auth_method,
oauth_clients.created`

const oauthAuthorizationCodeColumns = `
oauth_authorization_codes.code_hash,
oauth_authorization_codes.client_id,
oauth_authorization_codes.player_id,
oauth_authorization_codes.redirect_uri,
oauth_authorization_codes.code_challenge,
oauth_authorization_codes.scope,
oauth_authorization_codes.resource,
oauth_authorization_codes.expires,
oauth_authorization_codes.consumed,
oauth_authorization_codes.created`

const oauthRefreshColumns = `
oauth_refresh_tokens.token_hash,
oauth_refresh_tokens.client_id,
oauth_refresh_tokens.player_id,
oauth_refresh_tokens.scope,
oauth_refresh_tokens.resource,
oauth_refresh_tokens.expires,
oauth_refresh_tokens.revoked,
oauth_refresh_tokens.rotated_to,
oauth_refresh_tokens.created`

// OAuthClient is a record in the `oauth_clients` table
// ClientSecretHash is the hash of the client secret; hashing happens in a higher service layer
type OAuthClient struct {
	ClientID                string    `json:"clientId"`
	ClientSecretHash        *string   `json:"-"`
	ClientName              string    `json:"clientName"`
	RedirectURIs            []string  `json:"redirectUris"`
	GrantTypes              []string  `json:"grantTypes"`
	TokenEndpointAuthMethod string    `json:"tokenEndpointAuthMethod"`
	Created                 time.Time `json:"created"`
}

// OAuthAuthorizationCode is a record in the `oauth_authorization_codes` table
// CodeHash is the hash of the authorization code; hashing happens in a higher service layer
type OAuthAuthorizationCode struct {
	CodeHash      string    `json:"-"`
	ClientID      string    `json:"clientId"`
	PlayerID      int64     `json:"playerId"`
	RedirectURI   string    `json:"redirectUri"`
	CodeChallenge string    `json:"codeChallenge"`
	Scope         *string   `json:"scope"`
	Resource      *string   `json:"resource"`
	Expires       time.Time `json:"expires"`
	Consumed      bool      `json:"consumed"`
	Created       time.Time `json:"created"`
}

// OAuthRefreshToken is a record in the `oauth_refresh_tokens` table
// TokenHash is the hash of the refresh token; hashing happens in a higher service layer
type OAuthRefreshToken struct {
	TokenHash string    `json:"-"`
	ClientID  string    `json:"clientId"`
	PlayerID  int64     `json:"playerId"`
	Scope     *string   `json:"scope"`
	Resource  *string   `json:"resource"`
	Expires   time.Time `json:"expires"`
	Revoked   bool      `json:"revoked"`
	RotatedTo *string   `json:"-"`
	Created   time.Time `json:"created"`
}

func getOAuthClientByRow(row db.Scanner) (*OAuthClient, error) {
	var c OAuthClient
	if err := row.Scan(
		&c.ClientID,
		&c.ClientSecretHash,
		&c.ClientName,
		pq.Array(&c.RedirectURIs),
		pq.Array(&c.GrantTypes),
		&c.TokenEndpointAuthMethod,
		&c.Created,
	); err != nil {
		return nil, err
	}

	return &c, nil
}

func getOAuthAuthorizationCodeByRow(row db.Scanner) (*OAuthAuthorizationCode, error) {
	var c OAuthAuthorizationCode
	if err := row.Scan(
		&c.CodeHash,
		&c.ClientID,
		&c.PlayerID,
		&c.RedirectURI,
		&c.CodeChallenge,
		&c.Scope,
		&c.Resource,
		&c.Expires,
		&c.Consumed,
		&c.Created,
	); err != nil {
		return nil, err
	}

	return &c, nil
}

func getOAuthRefreshTokenByRow(row db.Scanner) (*OAuthRefreshToken, error) {
	var t OAuthRefreshToken
	if err := row.Scan(
		&t.TokenHash,
		&t.ClientID,
		&t.PlayerID,
		&t.Scope,
		&t.Resource,
		&t.Expires,
		&t.Revoked,
		&t.RotatedTo,
		&t.Created,
	); err != nil {
		return nil, err
	}

	return &t, nil
}

// CreateClient creates a new OAuth client
func (r *OAuthRepo) CreateClient(ctx context.Context, client *OAuthClient) error {
	const query = `
INSERT INTO oauth_clients (client_id, client_secret_hash, client_name, redirect_uris, grant_types, token_endpoint_auth_method)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + oauthClientColumns

	row := r.db.QueryRowContext(ctx, query,
		client.ClientID,
		client.ClientSecretHash,
		client.ClientName,
		pq.Array(client.RedirectURIs),
		pq.Array(client.GrantTypes),
		client.TokenEndpointAuthMethod,
	)

	created, err := getOAuthClientByRow(row)
	if err != nil {
		return err
	}

	*client = *created
	return nil
}

// GetClient returns an OAuth client by its client ID
func (r *OAuthRepo) GetClient(ctx context.Context, clientID string) (*OAuthClient, error) {
	const query = `
SELECT ` + oauthClientColumns + `
FROM oauth_clients
WHERE client_id = $1`

	row := r.db.QueryRowContext(ctx, query, clientID)
	return getOAuthClientByRow(row)
}

// CreateAuthCode creates a new OAuth authorization code
func (r *OAuthRepo) CreateAuthCode(ctx context.Context, code *OAuthAuthorizationCode) error {
	const query = `
INSERT INTO oauth_authorization_codes (code_hash, client_id, player_id, redirect_uri, code_challenge, scope, resource, expires)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING ` + oauthAuthorizationCodeColumns

	row := r.db.QueryRowContext(ctx, query,
		code.CodeHash,
		code.ClientID,
		code.PlayerID,
		code.RedirectURI,
		code.CodeChallenge,
		code.Scope,
		code.Resource,
		code.Expires.In(time.UTC),
	)

	created, err := getOAuthAuthorizationCodeByRow(row)
	if err != nil {
		return err
	}

	*code = *created
	return nil
}

// ConsumeAuthCode atomically marks an authorization code as consumed and returns it
// Returns sql.ErrNoRows if the code does not exist, has already been consumed, or has expired
func (r *OAuthRepo) ConsumeAuthCode(ctx context.Context, codeHash string) (*OAuthAuthorizationCode, error) {
	const query = `
UPDATE oauth_authorization_codes
SET consumed = true
WHERE code_hash = $1
  AND consumed = false
  AND expires > (NOW() AT TIME ZONE 'UTC')
RETURNING ` + oauthAuthorizationCodeColumns

	row := r.db.QueryRowContext(ctx, query, codeHash)
	return getOAuthAuthorizationCodeByRow(row)
}

// CreateRefreshToken creates a new OAuth refresh token
func (r *OAuthRepo) CreateRefreshToken(ctx context.Context, token *OAuthRefreshToken) error {
	const query = `
INSERT INTO oauth_refresh_tokens (token_hash, client_id, player_id, scope, resource, expires)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + oauthRefreshColumns

	row := r.db.QueryRowContext(ctx, query,
		token.TokenHash,
		token.ClientID,
		token.PlayerID,
		token.Scope,
		token.Resource,
		token.Expires.In(time.UTC),
	)

	created, err := getOAuthRefreshTokenByRow(row)
	if err != nil {
		return err
	}

	*token = *created
	return nil
}

// GetRefreshToken returns an OAuth refresh token by its hash
func (r *OAuthRepo) GetRefreshToken(ctx context.Context, tokenHash string) (*OAuthRefreshToken, error) {
	const query = `
SELECT ` + oauthRefreshColumns + `
FROM oauth_refresh_tokens
WHERE token_hash = $1`

	row := r.db.QueryRowContext(ctx, query, tokenHash)
	return getOAuthRefreshTokenByRow(row)
}

// RotateRefreshToken revokes the old refresh token (recording the new token's hash) and inserts
// the new refresh token, all within a single transaction.
// If the old refresh token was already revoked, ErrRefreshTokenAlreadyRevoked is returned so
// callers can detect refresh token reuse.
func (r *OAuthRepo) RotateRefreshToken(ctx context.Context, oldHash string, newToken *OAuthRefreshToken) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	const revokeQuery = `
UPDATE oauth_refresh_tokens
SET revoked = true, rotated_to = $1
WHERE token_hash = $2
  AND revoked = false`

	result, err := tx.ExecContext(ctx, revokeQuery, newToken.TokenHash, oldHash)
	if err != nil {
		rollback(tx)
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		rollback(tx)
		return err
	}

	if affected == 0 {
		rollback(tx)

		// distinguish "already revoked" from "does not exist"
		existing, getErr := r.GetRefreshToken(ctx, oldHash)
		if getErr != nil {
			return getErr
		}
		if existing.Revoked {
			return ErrRefreshTokenAlreadyRevoked
		}

		return sql.ErrNoRows
	}

	const insertQuery = `
INSERT INTO oauth_refresh_tokens (token_hash, client_id, player_id, scope, resource, expires)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING ` + oauthRefreshColumns

	row := tx.QueryRowContext(ctx, insertQuery,
		newToken.TokenHash,
		newToken.ClientID,
		newToken.PlayerID,
		newToken.Scope,
		newToken.Resource,
		newToken.Expires.In(time.UTC),
	)

	created, err := getOAuthRefreshTokenByRow(row)
	if err != nil {
		rollback(tx)
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	*newToken = *created
	return nil
}

// RevokeRefreshTokenFamily revokes all refresh tokens for the given player and client
func (r *OAuthRepo) RevokeRefreshTokenFamily(ctx context.Context, playerID int64, clientID string) error {
	const query = `
UPDATE oauth_refresh_tokens
SET revoked = true
WHERE player_id = $1
  AND client_id = $2
  AND revoked = false`

	_, err := r.db.ExecContext(ctx, query, playerID, clientID)
	return err
}
