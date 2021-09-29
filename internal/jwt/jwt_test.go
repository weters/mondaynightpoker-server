package jwt

import (
	"path/filepath"
	"testing"
	"time"

	jwtgo "github.com/golang-jwt/jwt"
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

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.StandardClaims{
		Audience: "different-audience",
		Id:       uuid.New().String(),
		IssuedAt: time.Now().Unix(),
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

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.StandardClaims{
		Audience: Audience,
		Id:       uuid.New().String(),
		IssuedAt: time.Now().Unix(),
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

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.StandardClaims{
		Audience:  Audience,
		Id:        uuid.New().String(),
		IssuedAt:  time.Now().Unix(),
		Issuer:    Issuer,
		ExpiresAt: time.Now().Add(time.Hour * -1).Unix(),
		Subject:   "15",
	})

	signedToken, err := token.SignedString(privateKey)
	if err != nil {
		t.Error(err)
		return
	}

	id, err := ValidUserID(signedToken)
	if err != nil {
		assert.Regexp(t, "^token is expired", err.Error())
	} else {
		t.Error("expected an error")
	}
	assert.Equal(t, int64(0), id)
}
