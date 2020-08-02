package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_bourreFactory_Details(t *testing.T) {
	name, ante, err := factories["bourre"].Details(playable.AdditionalData{
		"ante": float64(25),
	})
	assert.NoError(t, err)
	assert.Equal(t, "Bourré", name)
	assert.Equal(t, 25, ante)
}
