package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestInstance(t *testing.T) {
	_ = os.Setenv("MNP_CONFIG_FILE", "testdata/config.yaml")
	_ = os.Setenv("MNP_JWT_PRIVATE_KEY", "private2.key")
	a := assert.New(t)
	cfg := Instance()
	a.Equal("postgres://localhost", cfg.PGDSN)
	a.Equal("public.pem", cfg.JWT.PublicKey)
	a.Equal("private2.key", cfg.JWT.PrivateKey)

	// ensure that it's only loaded once
	_ = os.Setenv("MNP_JWT_PRIVATE_KEY", "private3.key")
	// ensure we aren't using a pointer
	cfg.JWT.PrivateKey = "bad"
	cfg = Instance()
	a.Equal("private2.key", cfg.JWT.PrivateKey)

}
