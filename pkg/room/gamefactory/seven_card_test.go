package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_sevenCardFactory_Name(t *testing.T) {
	a := assert.New(t)

	name, ante, err := factories["seven-card"].Details(playable.AdditionalData{
		"variant": "stud",
		"ante":    float64(25),
	})
	a.NoError(err)
	a.Equal(25, ante)
	a.Equal("Seven-Card Stud", name)

	name, ante, err = factories["seven-card"].Details(playable.AdditionalData{
		"variant": "baseball",
		"ante":    float64(50),
	})
	a.NoError(err)
	a.Equal(50, ante)
	a.Equal("Baseball", name)

	name, ante, err = factories["seven-card"].Details(playable.AdditionalData{
		"variant": "low-card-wild",
		"ante":    float64(75),
	})
	a.NoError(err)
	a.Equal(75, ante)
	a.Equal("Low Card Wild", name)
}
