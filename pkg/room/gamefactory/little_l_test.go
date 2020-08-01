package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_littleLFactory_Name(t *testing.T) {
	name, err := factories["little-l"].Name(playable.AdditionalData{
		"tradeIns": []int{0, 2},
	})

	assert.NoError(t, err)
	assert.Equal(t, "Little L (trade: 0, 2)", name)
}
