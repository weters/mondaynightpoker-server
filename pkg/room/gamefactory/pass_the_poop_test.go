package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_passThePoopFactory_Name(t *testing.T) {
	a := assert.New(t)
	name, ante, err := factories["pass-the-poop"].Details(playable.AdditionalData{
		"edition": "standard",
		"ante":    float64(25),
	})
	a.NoError(err)
	a.Equal(25, ante)
	a.Equal("Pass the Poop, Standard Edition", name)

	name, ante, err = factories["pass-the-poop"].Details(playable.AdditionalData{
		"edition": "diarrhea",
		"ante":    float64(25),
	})
	a.NoError(err)
	a.Equal(25, ante)
	a.Equal("Pass the Poop, Diarrhea Edition", name)

	name, ante, err = factories["pass-the-poop"].Details(playable.AdditionalData{
		"edition":     "pairs",
		"ante":        float64(75),
		"allowBlocks": false,
	})
	a.NoError(err)
	a.Equal(75, ante)
	a.Equal("Pass the Poop, Pairs Edition", name)

	name, ante, err = factories["pass-the-poop"].Details(playable.AdditionalData{
		"edition":     "pairs",
		"ante":        float64(75),
		"allowBlocks": true,
	})
	a.NoError(err)
	a.Equal(75, ante)
	a.Equal("Pass the Poop, Pairs Edition (with Blocks)", name)
}
