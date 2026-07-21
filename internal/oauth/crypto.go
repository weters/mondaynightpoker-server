package oauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// nonceParamKeys are the OAuth parameters bound into the anti-CSRF login nonce, in a
// fixed order so the signature is reproducible between the GET and POST handlers.
var nonceParamKeys = []string{
	"client_id",
	"redirect_uri",
	"response_type",
	"code_challenge",
	"code_challenge_method",
	"scope",
	"state",
	"resource",
}

// randomToken returns a base64url (no padding) encoding of n cryptographically random bytes.
func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// sha256Hex returns the lowercase hex-encoded SHA-256 digest of s.
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// verifyPKCE reports whether the S256 transform of verifier equals challenge.
func verifyPKCE(verifier, challenge string) bool {
	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

// collectParams extracts the nonce-bound OAuth parameters using the supplied getter
// (e.g. url.Values.Get or http.Request.FormValue).
func collectParams(get func(string) string) url.Values {
	v := url.Values{}
	for _, k := range nonceParamKeys {
		v.Set(k, get(k))
	}

	return v
}

// nonceMessage builds the canonical string signed by the anti-CSRF nonce.
func nonceMessage(params url.Values, expiry time.Time) string {
	var b strings.Builder
	b.WriteString(strconv.FormatInt(expiry.Unix(), 10))
	for _, k := range nonceParamKeys {
		b.WriteByte('\n')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params.Get(k))
	}

	return b.String()
}

// signNonce returns a short-lived signed anti-CSRF nonce binding the given params.
func (s *Server) signNonce(params url.Values, expiry time.Time) string {
	mac := hmac.New(sha256.New, s.nonceKey)
	mac.Write([]byte(nonceMessage(params, expiry)))
	sig := hex.EncodeToString(mac.Sum(nil))
	return strconv.FormatInt(expiry.Unix(), 10) + "." + sig
}

// verifyNonce reports whether nonce is a valid, unexpired signature over params.
func (s *Server) verifyNonce(params url.Values, nonce string) bool {
	parts := strings.SplitN(nonce, ".", 2)
	if len(parts) != 2 {
		return false
	}

	expUnix, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}

	expiry := time.Unix(expUnix, 0)
	if time.Now().After(expiry) {
		return false
	}

	expected := s.signNonce(params, expiry)
	return hmac.Equal([]byte(expected), []byte(nonce))
}

// nullableString returns nil for an empty string, otherwise a pointer to a copy of s.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
