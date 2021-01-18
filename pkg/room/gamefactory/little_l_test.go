package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_littleLFactory_Name(t *testing.T) {
	name, ante, err := factories["little-l"].Details(playable.AdditionalData{
		"tradeIns": []float64{0, 2},
		"ante":     float64(25),
	})

	assert.NoError(t, err)
	assert.Equal(t, "4-Card Little L (trade: 0, 2)", name)
	assert.Equal(t, 25, ante)

	name, ante, err = factories["little-l"].Details(playable.AdditionalData{
		"tradeIns":    []float64{0, 1, 2},
		"ante":        float64(50),
		"initialDeal": float64(3),
	})

	assert.NoError(t, err)
	assert.Equal(t, "3-Card Little L (trade: 0, 1, 2)", name)
	assert.Equal(t, 50, ante)

	name, ante, err = factories["little-l"].Details(playable.AdditionalData{
		"tradeIns":    []float64{0, 4},
		"initialDeal": float64(3),
	})

	assert.EqualError(t, err, "invalid trade-in option: 4")
	assert.Empty(t, name)
	assert.Empty(t, ante)
}
