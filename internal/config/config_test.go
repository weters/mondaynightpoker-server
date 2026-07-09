package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	clear1 := setEnv("MNP_CONFIG_FILE", "testdata/config.yaml")
	defer clear1()
	clear2 := setEnv("MNP_JWT_PRIVATE_KEY", "private2.key")
	defer clear2()

	a := assert.New(t)
	cfg, err := Load()
	a.NoError(err)
	a.Equal("user@mondaynight.bid", cfg.Email.Username)
	a.Equal("public.pem", cfg.JWT.PublicKey)
	a.Equal("private2.key", cfg.JWT.PrivateKey)

	// environment changes only apply on the next Load
	clear3 := setEnv("MNP_JWT_PRIVATE_KEY", "private3.key")
	defer clear3()

	cfg2, err := Load()
	a.NoError(err)
	a.Equal("private3.key", cfg2.JWT.PrivateKey)

	// ensure the previously returned config is a copy
	a.Equal("private2.key", cfg.JWT.PrivateKey)
}

func TestDefaults(t *testing.T) {
	cfg, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, "dealer@mondaynight.bid", cfg.Email.Sender)
}

func TestConfig_WebSocketOrigins(t *testing.T) {
	cfg := Config{Host: "https://mondaynight.bid"}
	assert.Equal(t, []string{"https://mondaynight.bid"}, cfg.WebSocketOrigins())

	cfg.AllowedOrigins = []string{"http://localhost:8080", "https://mondaynight.bid"}
	assert.Equal(t, []string{"http://localhost:8080", "https://mondaynight.bid"}, cfg.WebSocketOrigins())
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
