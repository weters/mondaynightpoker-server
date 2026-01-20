package token

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate returns a crypto-secure random string of length n
// The random string is contains the following characters:
// ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_
func Generate(n int) (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// base64 increases size by ~33%
	return base64.RawURLEncoding.EncodeToString(b)[0:n], nil
}
