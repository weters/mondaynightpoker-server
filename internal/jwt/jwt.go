package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"mondaynightpoker-server/internal/config"

	jwtgo "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Issuer issues the JWT
const Issuer = "us.taproom.mondaynightpoker"

// Audience is the intended JWT audience
const Audience = "mondaynightpoker.taproom.us"

// Signer signs and validates player JWTs
type Signer struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey

	// ttl is how long issued tokens are valid; zero issues tokens without expiry
	ttl time.Duration
}

// NewSigner loads the configured key pair and returns a Signer whose tokens
// expire after ttl (zero means no expiry claim)
func NewSigner(cfg config.JWT, ttl time.Duration) (*Signer, error) {
	privateKey, err := loadPrivateKey(cfg.PrivateKey)
	if err != nil {
		return nil, err
	}

	publicKey, err := loadPublicKey(cfg.PublicKey)
	if err != nil {
		return nil, err
	}

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey,
		ttl:        ttl,
	}, nil
}

// Sign will sign a JWT for the user ID
func (s *Signer) Sign(userID int64) (string, error) {
	now := time.Now()
	claims := jwtgo.RegisteredClaims{
		Audience: jwtgo.ClaimStrings{Audience},
		ID:       uuid.New().String(),
		IssuedAt: jwtgo.NewNumericDate(now),
		Issuer:   Issuer,
		Subject:  strconv.FormatInt(userID, 10),
	}

	if s.ttl > 0 {
		claims.ExpiresAt = jwtgo.NewNumericDate(now.Add(s.ttl))
	}

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

// ValidUserID will validate a signed JWT
// Tokens without an expiry claim remain valid: tokens issued before expiry was
// introduced rotate onto expiring ones when the client next refreshes. Once
// legacy tokens have aged out, enforce jwtgo.WithExpirationRequired() here.
func (s *Signer) ValidUserID(signedString string) (int64, error) {
	token, err := jwtgo.ParseWithClaims(signedString, &jwtgo.RegisteredClaims{}, func(token *jwtgo.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtgo.SigningMethodRSA); !ok {
			return nil, errors.New("expected RS256 signing method")
		}

		return s.publicKey, nil
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

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read public key: %w", err)
	}

	pem, err := jwtgo.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("could not parse RSA public key: %w", err)
	}

	return pem, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read private key: %w", err)
	}

	pem, err := jwtgo.ParseRSAPrivateKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("could not parse RSA private key: %w", err)
	}

	return pem, nil
}

func containsAudience(audiences jwtgo.ClaimStrings, target string) bool {
	for _, aud := range audiences {
		if aud == target {
			return true
		}
	}
	return false
}
