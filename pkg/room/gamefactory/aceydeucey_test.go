package gamefactory

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"mondaynightpoker-server/pkg/playable/aceydeucey"
	"testing"
)

func Test_aceyDeuceyFactory_CreateGame(t *testing.T) {
	a := assert.New(t)
	game, err := aceyDeuceyFactory{}.CreateGame(logrus.StandardLogger(), []int64{1, 2}, playable.AdditionalData{})
	a.IsType(&aceydeucey.Game{}, game)
	a.NoError(err)
}

func Test_aceyDeuceyFactory_Details(t *testing.T) {
	a := assert.New(t)

	name, ante, err := aceyDeuceyFactory{}.Details(playable.AdditionalData{
		"ante": float64(50),
	})

	a.Equal("Acey Deucey", name)
	a.Equal(50, ante)
	a.NoError(err)

	name, ante, err = aceyDeuceyFactory{}.Details(playable.AdditionalData{
		"ante":           float64(100),
		"continuousShoe": true,
		"allowPass":      true,
	})

	a.Equal("Acey Deucey (Continuous Shoe and With Passing)", name)
	a.Equal(100, ante)
	a.NoError(err)
}

func Test_getAceyDeuceyOptions(t *testing.T) {
	a := assert.New(t)
	opts := getAceyDeuceyOptions(playable.AdditionalData{})

	a.Equal(25, opts.Ante)
	a.False(opts.ContinuousShoe)
	a.False(opts.AllowPass)

	opts = getAceyDeuceyOptions(playable.AdditionalData{
		"ante":           float64(100),
		"continuousShoe": true,
		"allowPass":      true,
	})
	a.Equal(100, opts.Ante)
	a.True(opts.ContinuousShoe)
	a.True(opts.AllowPass)
}
