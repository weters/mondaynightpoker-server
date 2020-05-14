package util

import (
	"github.com/bmizerany/assert"
	"os"
	"testing"
)

func TestGetenv(t *testing.T) {
	os.Setenv("ENV_TEST_KEY", "my value")
	assert.Equal(t, "my value", Getenv("ENV_TEST_KEY", "other value"))
	assert.Equal(t, "other value", Getenv("ENV_TEST_KEY NOT FOUND", "other value"))
}
