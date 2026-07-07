package jwt

import (
	"path/filepath"
	"testing"
	"time"

	"mondaynightpoker-server/internal/config"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testJWTConfig() config.JWT {
	return config.JWT{
		PublicKey:  filepath.Join("testdata", "public.pem"),
		PrivateKey: filepath.Join("testdata", "private.key"),
	}
}

func newTestSigner(t *testing.T, ttl time.Duration) *Signer {
	t.Helper()

	signer, err := NewSigner(testJWTConfig(), ttl)
	require.NoError(t, err)
	return signer
}

func TestNewSigner_badKeys(t *testing.T) {
	cfg := testJWTConfig()
	cfg.PublicKey = filepath.Join("testdata", "does-not-exist.pem")
	signer, err := NewSigner(cfg, 0)
	assert.Error(t, err)
	assert.Nil(t, signer)

	cfg = testJWTConfig()
	cfg.PrivateKey = filepath.Join("testdata", "does-not-exist.key")
	signer, err = NewSigner(cfg, 0)
	assert.Error(t, err)
	assert.Nil(t, signer)
}

func TestSignAndValidateUserID(t *testing.T) {
	signer := newTestSigner(t, 0)

	sign, err := signer.Sign(18)
	assert.NoError(t, err)

	id, err := signer.ValidUserID(sign)
	assert.NoError(t, err)
	assert.Equal(t, int64(18), id)
}

func TestSigner_ttl(t *testing.T) {
	signer := newTestSigner(t, time.Hour)

	sign, err := signer.Sign(21)
	require.NoError(t, err)

	// the token validates OK
	id, err := signer.ValidUserID(sign)
	assert.NoError(t, err)
	assert.Equal(t, int64(21), id)

	// and expires one hour after it was issued
	var claims jwtgo.RegisteredClaims
	_, _, err = jwtgo.NewParser().ParseUnverified(sign, &claims)
	require.NoError(t, err)
	require.NotNil(t, claims.IssuedAt)
	require.NotNil(t, claims.ExpiresAt)
	assert.Equal(t, time.Hour, claims.ExpiresAt.Sub(claims.IssuedAt.Time))
}

func TestValidUserID_InvalidAudience(t *testing.T) {
	signer := newTestSigner(t, 0)

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.RegisteredClaims{
		Audience: jwtgo.ClaimStrings{"different-audience"},
		ID:       uuid.New().String(),
		IssuedAt: jwtgo.NewNumericDate(time.Now()),
		Issuer:   Issuer,
		Subject:  "15",
	})

	signedToken, err := token.SignedString(signer.privateKey)
	require.NoError(t, err)

	id, err := signer.ValidUserID(signedToken)
	assert.EqualError(t, err, "invalid audience")
	assert.Equal(t, int64(0), id)
}

func TestValidUserID_InvalidIssuer(t *testing.T) {
	signer := newTestSigner(t, 0)

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.RegisteredClaims{
		Audience: jwtgo.ClaimStrings{Audience},
		ID:       uuid.New().String(),
		IssuedAt: jwtgo.NewNumericDate(time.Now()),
		Issuer:   "invalid-issuer",
		Subject:  "15",
	})

	signedToken, err := token.SignedString(signer.privateKey)
	require.NoError(t, err)

	id, err := signer.ValidUserID(signedToken)
	assert.EqualError(t, err, "invalid issuer")
	assert.Equal(t, int64(0), id)
}

func TestValidUserID_Expired(t *testing.T) {
	signer := newTestSigner(t, 0)

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.RegisteredClaims{
		Audience:  jwtgo.ClaimStrings{Audience},
		ID:        uuid.New().String(),
		IssuedAt:  jwtgo.NewNumericDate(time.Now()),
		Issuer:    Issuer,
		ExpiresAt: jwtgo.NewNumericDate(time.Now().Add(time.Hour * -1)),
		Subject:   "15",
	})

	signedToken, err := token.SignedString(signer.privateKey)
	require.NoError(t, err)

	id, err := signer.ValidUserID(signedToken)
	if err != nil {
		assert.Contains(t, err.Error(), "token is expired")
	} else {
		t.Error("expected an error")
	}
	assert.Equal(t, int64(0), id)
}
