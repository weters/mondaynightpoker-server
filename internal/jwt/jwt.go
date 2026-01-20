package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"mondaynightpoker-server/internal/config"
	"os"
	"strconv"
	"time"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Issuer issues the JWT
const Issuer = "us.taproom.mondaynightpoker"

// Audience is the intended JWT audience
const Audience = "mondaynightpoker.taproom.us"

var publicKey *rsa.PublicKey
var privateKey *rsa.PrivateKey

// LoadKeys will load the public and private keys
// this method should only be called once.
func LoadKeys() {
	cfg := config.Instance().JWT
	privateKey = loadPrivateKey(cfg.PrivateKey)
	publicKey = loadPublicKey(cfg.PublicKey)
}

// Sign will sign a JWT for the user ID
func Sign(userID int64) (string, error) {
	if privateKey == nil {
		panic("LoadKeys() not called")
	}

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, jwtgo.RegisteredClaims{
		Audience: jwtgo.ClaimStrings{Audience},
		ID:       uuid.New().String(),
		IssuedAt: jwtgo.NewNumericDate(time.Now()),
		Issuer:   Issuer,
		Subject:  strconv.FormatInt(userID, 10),
	})

	return token.SignedString(privateKey)
}

// ValidUserID will validate a signed JWT
func ValidUserID(signedString string) (int64, error) {
	if publicKey == nil {
		panic("LoadKeys() not called")
	}

	token, err := jwtgo.ParseWithClaims(signedString, &jwtgo.RegisteredClaims{}, func(token *jwtgo.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtgo.SigningMethodRSA); !ok {
			return nil, errors.New("expected RS256 signing method")
		}

		return publicKey, nil
	})

	if err != nil {
		return 0, err
	}

	if token.Valid {
		if claims, ok := token.Claims.(*jwtgo.RegisteredClaims); ok {
			if !containsAudience(claims.Audience, Audience) {
				return 0, errors.New("invalid audience")
			}

			if claims.Issuer != Issuer {
				return 0, errors.New("invalid issuer")
			}

			return strconv.ParseInt(claims.Subject, 10, 64)
		}

		return 0, fmt.Errorf("expected jwt.RegisteredClaims, got %T", token.Claims)
	}

	logrus.Warn("token claims were not valid. did not expect to reach this code")
	return 0, errors.New("claims were not valid")
}

func loadPublicKey(path string) *rsa.PublicKey {
	b, err := os.ReadFile(path)
	if err != nil {
		logrus.WithError(err).Fatal("could not read file")
	}

	pem, err := jwtgo.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		logrus.WithError(err).Fatal("could not parse RSA private key")
	}

	return pem
}

func loadPrivateKey(path string) *rsa.PrivateKey {
	b, err := os.ReadFile(path)
	if err != nil {
		logrus.WithError(err).Fatal("could not read file")
	}

	pem, err := jwtgo.ParseRSAPrivateKeyFromPEM(b)
	if err != nil {
		logrus.WithError(err).Fatal("could not parse RSA private key")
	}

	return pem
}

func containsAudience(audiences jwtgo.ClaimStrings, target string) bool {
	for _, aud := range audiences {
		if aud == target {
			return true
		}
	}
	return false
}
