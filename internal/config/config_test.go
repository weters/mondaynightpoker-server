package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestInstance(t *testing.T) {
	clear1 := setEnv("MNP_CONFIG_FILE", "testdata/config.yaml")
	defer clear1()
	clear2 := setEnv("MNP_JWT_PRIVATE_KEY", "private2.key")
	defer clear2()

	a := assert.New(t)
	cfg := Instance()
	a.Equal("user@mondaynight.bid", cfg.Email.Username)
	a.Equal("public.pem", cfg.JWT.PublicKey)
	a.Equal("private2.key", cfg.JWT.PrivateKey)

	// ensure that it's only loaded once
	_ = os.Setenv("MNP_JWT_PRIVATE_KEY", "private3.key")
	// ensure we aren't using a pointer
	cfg.JWT.PrivateKey = "bad"
	cfg = Instance()
	a.Equal("private2.key", cfg.JWT.PrivateKey)
}

func TestDefaults(t *testing.T) {
	assert.NoError(t, Load())
	cfg := Instance()
	assert.Equal(t, "no-reply@mondaynight.bid", cfg.Email.Sender)
}

func setEnv(key, val string) func() {
	orig := os.Getenv(key)
	_ = os.Setenv(key, val)
	return func() {
		if orig == "" {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, orig)
		}
	}
}
