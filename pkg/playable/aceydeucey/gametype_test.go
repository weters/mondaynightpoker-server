package aceydeucey

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetGameTypes(t *testing.T) {
	gt := GetGameTypes()
	assert.Equal(t, map[GameType]string{
		0: "Standard",
		1: "Continuous Shoe",
		2: "Chaos",
	}, gt)
}

func TestGetGameType(t *testing.T) {
	a := assert.New(t)

	gt, err := GetGameType("foo")
	a.Equal(GameType(-1), gt)
	a.EqualError(err, "unknown game type: foo")

	gt, err = GetGameType("standard")
	a.NoError(err)
	a.Equal(GameType(0), gt)

	gt, err = GetGameType("continuous SHOE")
	a.NoError(err)
	a.Equal(GameType(1), gt)

	gt, err = GetGameType("Chaos")
	a.NoError(err)
	a.Equal(GameType(2), gt)
}
