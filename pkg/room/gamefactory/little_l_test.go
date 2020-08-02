package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_littleLFactory_Name(t *testing.T) {
	name, ante, err := factories["little-l"].Details(playable.AdditionalData{
		"tradeIns": []int{0, 2},
		"ante":     float64(25),
	})

	assert.NoError(t, err)
	assert.Equal(t, "Little L (trade: 0, 2)", name)
	assert.Equal(t, 25, ante)
}
