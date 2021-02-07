package token

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerate(t *testing.T) {
	token, err := Generate(8)
	assert.NoError(t, err)
	assert.Equal(t, 8, len(token))

	token2, err := Generate(8)
	assert.NoError(t, err)
	assert.NotEqual(t, token, token2)
}
