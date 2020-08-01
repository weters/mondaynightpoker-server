package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_passThePoopFactory_Name(t *testing.T) {
	a := assert.New(t)
	name, err := factories["pass-the-poop"].Name(playable.AdditionalData{
		"edition": "standard",
		"ante":    float64(25),
	})
	a.NoError(err)
	a.Equal("Pass the Poop, Standard Edition", name)

	name, err = factories["pass-the-poop"].Name(playable.AdditionalData{
		"edition": "diarrhea",
		"ante":    float64(25),
	})
	a.NoError(err)
	a.Equal("Pass the Poop, Diarrhea Edition", name)

	name, err = factories["pass-the-poop"].Name(playable.AdditionalData{
		"edition": "pairs",
		"ante":    float64(25),
	})
	a.NoError(err)
	a.Equal("Pass the Poop, Pairs Edition", name)
}
