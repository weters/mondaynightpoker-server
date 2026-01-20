package jwt

import (
	"path/filepath"
	"testing"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSignAndValidateUserID(t *testing.T) {
	publicKey = loadPublicKey(filepath.Join("testdata", "public.pem"))
	privateKey = loadPrivateKey(filepath.Join("testdata", "private.key"))

	sign, err := Sign(18)
	assert.NoError(t, err)

	id, err := ValidUserID(sign)
	assert.NoError(t, err)
	assert.Equal(t, int64(18), id)
}

func TestValidUserID_InvalidAudience(t *testing.T) {
	publicKey = loadPublicKey(filepath.Join("testdata", "public.pem"))
	privateKey = loadPrivateKey(filepath.Join("testdata", "private.key"))

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.RegisteredClaims{
		Audience: jwtgo.ClaimStrings{"different-audience"},
		ID:       uuid.New().String(),
		IssuedAt: jwtgo.NewNumericDate(time.Now()),
		Issuer:   Issuer,
		Subject:  "15",
	})

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Error(err)
		return
	}

	id, err := ValidUserID(signedToken)
	assert.EqualError(t, err, "invalid audience")
	assert.Equal(t, int64(0), id)
}

func TestValidUserID_InvalidIssuer(t *testing.T) {
	publicKey = loadPublicKey(filepath.Join("testdata", "public.pem"))
	privateKey = loadPrivateKey(filepath.Join("testdata", "private.key"))

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.RegisteredClaims{
		Audience: jwtgo.ClaimStrings{Audience},
		ID:       uuid.New().String(),
		IssuedAt: jwtgo.NewNumericDate(time.Now()),
		Issuer:   "invalid-issuer",
		Subject:  "15",
	})

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Error(err)
		return
	}

	id, err := ValidUserID(signedToken)
	assert.EqualError(t, err, "invalid issuer")
	assert.Equal(t, int64(0), id)
}

func TestValidUserID_Expired(t *testing.T) {
	publicKey = loadPublicKey(filepath.Join("testdata", "public.pem"))
	privateKey = loadPrivateKey(filepath.Join("testdata", "private.key"))

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.RegisteredClaims{
		Audience:  jwtgo.ClaimStrings{Audience},
		ID:        uuid.New().String(),
		IssuedAt:  jwtgo.NewNumericDate(time.Now()),
		Issuer:    Issuer,
		ExpiresAt: jwtgo.NewNumericDate(time.Now().Add(time.Hour * -1)),
		Subject:   "15",
	})

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Error(err)
		return
	}

	id, err := ValidUserID(signedToken)
	if err != nil {
		assert.Contains(t, err.Error(), "token is expired")
	} else {
		t.Error("expected an error")
	}
	assert.Equal(t, int64(0), id)
}
