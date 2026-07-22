package model

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// randomHash generates a unique, crypto-random hash-like string suitable for testing
func randomHash() string {
	return uuid.New().String()
}

func oauthClient(t *testing.T) *OAuthClient {
	t.Helper()

	secretHash := randomHash()
	client := &OAuthClient{
		ClientID:                uuid.New().String(),
		ClientSecretHash:        &secretHash,
		ClientName:              "test-client",
		RedirectURIs:            []string{"https://example.com/callback", "https://example.com/callback2"},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: "client_secret_basic",
	}

	if err := testRepos.OAuth.CreateClient(cbg, client); err != nil {
		panic(err)
	}

	return client
}

func TestOAuthRepo_CreateClient_GetClient(t *testing.T) {
	secretHash := randomHash()
	client := &OAuthClient{
		ClientID:                uuid.New().String(),
		ClientSecretHash:        &secretHash,
		ClientName:              "my-test-client",
		RedirectURIs:            []string{"https://example.com/cb1", "https://example.com/cb2"},
		GrantTypes:              []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethod: "none",
	}

	err := testRepos.OAuth.CreateClient(cbg, client)
	assert.NoError(t, err)
	assert.False(t, client.Created.IsZero())

	fetched, err := testRepos.OAuth.GetClient(cbg, client.ClientID)
	assert.NoError(t, err)
	assert.Equal(t, client.ClientID, fetched.ClientID)
	assert.Equal(t, client.ClientName, fetched.ClientName)
	assert.Equal(t, &secretHash, fetched.ClientSecretHash)
	assert.Equal(t, client.TokenEndpointAuthMethod, fetched.TokenEndpointAuthMethod)

	// array round-trip
	assert.Equal(t, []string{"https://example.com/cb1", "https://example.com/cb2"}, fetched.RedirectURIs)
	assert.Equal(t, []string{"authorization_code", "refresh_token"}, fetched.GrantTypes)

	// nil client secret hash (public client) round-trips as nil
	publicClient := &OAuthClient{
		ClientID:                uuid.New().String(),
		ClientName:              "public-client",
		RedirectURIs:            []string{"https://example.com/cb"},
		GrantTypes:              []string{"authorization_code"},
		TokenEndpointAuthMethod: "none",
	}
	assert.NoError(t, testRepos.OAuth.CreateClient(cbg, publicClient))
	fetchedPublic, err := testRepos.OAuth.GetClient(cbg, publicClient.ClientID)
	assert.NoError(t, err)
	assert.Nil(t, fetchedPublic.ClientSecretHash)
}

func TestOAuthRepo_GetClient_NotFound(t *testing.T) {
	client, err := testRepos.OAuth.GetClient(cbg, "does-not-exist-"+uuid.New().String())
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, client)
}

func TestOAuthRepo_AuthCode_ConsumeHappyPath(t *testing.T) {
	client := oauthClient(t)
	p := player()

	scope := "read write"
	resource := "https://api.example.com"
	code := &OAuthAuthorizationCode{
		CodeHash:      randomHash(),
		ClientID:      client.ClientID,
		PlayerID:      p.ID,
		RedirectURI:   "https://example.com/callback",
		CodeChallenge: randomHash(),
		Scope:         &scope,
		Resource:      &resource,
		Expires:       time.Now().Add(time.Minute),
	}

	err := testRepos.OAuth.CreateAuthCode(cbg, code)
	assert.NoError(t, err)
	assert.False(t, code.Consumed)

	consumed, err := testRepos.OAuth.ConsumeAuthCode(cbg, code.CodeHash)
	assert.NoError(t, err)
	assert.True(t, consumed.Consumed)
	assert.Equal(t, code.ClientID, consumed.ClientID)
	assert.Equal(t, code.PlayerID, consumed.PlayerID)
	assert.Equal(t, code.RedirectURI, consumed.RedirectURI)
	assert.Equal(t, code.CodeChallenge, consumed.CodeChallenge)
	assert.Equal(t, &scope, consumed.Scope)
	assert.Equal(t, &resource, consumed.Resource)
}

func TestOAuthRepo_AuthCode_ConsumeTwiceFails(t *testing.T) {
	client := oauthClient(t)
	p := player()

	code := &OAuthAuthorizationCode{
		CodeHash:      randomHash(),
		ClientID:      client.ClientID,
		PlayerID:      p.ID,
		RedirectURI:   "https://example.com/callback",
		CodeChallenge: randomHash(),
		Expires:       time.Now().Add(time.Minute),
	}
	assert.NoError(t, testRepos.OAuth.CreateAuthCode(cbg, code))

	_, err := testRepos.OAuth.ConsumeAuthCode(cbg, code.CodeHash)
	assert.NoError(t, err)

	_, err = testRepos.OAuth.ConsumeAuthCode(cbg, code.CodeHash)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestOAuthRepo_AuthCode_ConsumeExpiredFails(t *testing.T) {
	client := oauthClient(t)
	p := player()

	code := &OAuthAuthorizationCode{
		CodeHash:      randomHash(),
		ClientID:      client.ClientID,
		PlayerID:      p.ID,
		RedirectURI:   "https://example.com/callback",
		CodeChallenge: randomHash(),
		Expires:       time.Now().Add(-time.Minute),
	}
	assert.NoError(t, testRepos.OAuth.CreateAuthCode(cbg, code))

	_, err := testRepos.OAuth.ConsumeAuthCode(cbg, code.CodeHash)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestOAuthRepo_AuthCode_ConsumeMissingFails(t *testing.T) {
	_, err := testRepos.OAuth.ConsumeAuthCode(cbg, randomHash())
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestOAuthRepo_RefreshToken_CreateGet(t *testing.T) {
	client := oauthClient(t)
	p := player()

	scope := "read"
	token := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Scope:     &scope,
		Expires:   time.Now().Add(time.Hour),
	}

	err := testRepos.OAuth.CreateRefreshToken(cbg, token)
	assert.NoError(t, err)
	assert.False(t, token.Revoked)
	assert.Nil(t, token.RotatedTo)

	fetched, err := testRepos.OAuth.GetRefreshToken(cbg, token.TokenHash)
	assert.NoError(t, err)
	assert.Equal(t, token.ClientID, fetched.ClientID)
	assert.Equal(t, token.PlayerID, fetched.PlayerID)
	assert.Equal(t, &scope, fetched.Scope)
	assert.False(t, fetched.Revoked)
}

func TestOAuthRepo_GetRefreshToken_NotFound(t *testing.T) {
	token, err := testRepos.OAuth.GetRefreshToken(cbg, randomHash())
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Nil(t, token)
}

func TestOAuthRepo_RotateRefreshToken_HappyPath(t *testing.T) {
	client := oauthClient(t)
	p := player()

	oldToken := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}
	assert.NoError(t, testRepos.OAuth.CreateRefreshToken(cbg, oldToken))

	newToken := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}

	err := testRepos.OAuth.RotateRefreshToken(cbg, oldToken.TokenHash, newToken)
	assert.NoError(t, err)
	assert.False(t, newToken.Revoked)

	oldFetched, err := testRepos.OAuth.GetRefreshToken(cbg, oldToken.TokenHash)
	assert.NoError(t, err)
	assert.True(t, oldFetched.Revoked)
	assert.NotNil(t, oldFetched.RotatedTo)
	assert.Equal(t, newToken.TokenHash, *oldFetched.RotatedTo)

	newFetched, err := testRepos.OAuth.GetRefreshToken(cbg, newToken.TokenHash)
	assert.NoError(t, err)
	assert.False(t, newFetched.Revoked)
}

func TestOAuthRepo_RotateRefreshToken_AlreadyRevokedDetection(t *testing.T) {
	client := oauthClient(t)
	p := player()

	oldToken := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}
	assert.NoError(t, testRepos.OAuth.CreateRefreshToken(cbg, oldToken))

	firstNewToken := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}
	assert.NoError(t, testRepos.OAuth.RotateRefreshToken(cbg, oldToken.TokenHash, firstNewToken))

	// attempting to rotate the already-revoked old token again should be detected as reuse
	secondNewToken := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}
	err := testRepos.OAuth.RotateRefreshToken(cbg, oldToken.TokenHash, secondNewToken)
	assert.ErrorIs(t, err, ErrRefreshTokenAlreadyRevoked)

	// the second new token should not have been created
	_, err = testRepos.OAuth.GetRefreshToken(cbg, secondNewToken.TokenHash)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestOAuthRepo_RotateRefreshToken_MissingToken(t *testing.T) {
	client := oauthClient(t)
	p := player()

	newToken := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}

	err := testRepos.OAuth.RotateRefreshToken(cbg, randomHash(), newToken)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestOAuthRepo_RevokeRefreshTokenFamily(t *testing.T) {
	client := oauthClient(t)
	p := player()

	token1 := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}
	token2 := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  client.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}
	assert.NoError(t, testRepos.OAuth.CreateRefreshToken(cbg, token1))
	assert.NoError(t, testRepos.OAuth.CreateRefreshToken(cbg, token2))

	// a token belonging to a different client should not be revoked
	otherClient := oauthClient(t)
	otherToken := &OAuthRefreshToken{
		TokenHash: randomHash(),
		ClientID:  otherClient.ClientID,
		PlayerID:  p.ID,
		Expires:   time.Now().Add(time.Hour),
	}
	assert.NoError(t, testRepos.OAuth.CreateRefreshToken(cbg, otherToken))

	err := testRepos.OAuth.RevokeRefreshTokenFamily(cbg, p.ID, client.ClientID)
	assert.NoError(t, err)

	fetched1, err := testRepos.OAuth.GetRefreshToken(cbg, token1.TokenHash)
	assert.NoError(t, err)
	assert.True(t, fetched1.Revoked)

	fetched2, err := testRepos.OAuth.GetRefreshToken(cbg, token2.TokenHash)
	assert.NoError(t, err)
	assert.True(t, fetched2.Revoked)

	fetchedOther, err := testRepos.OAuth.GetRefreshToken(cbg, otherToken.TokenHash)
	assert.NoError(t, err)
	assert.False(t, fetchedOther.Revoked)
}
