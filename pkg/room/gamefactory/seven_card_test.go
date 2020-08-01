package gamefactory

import (
	"github.com/stretchr/testify/assert"
	"mondaynightpoker-server/pkg/playable"
	"testing"
)

func Test_sevenCardFactory_Name(t *testing.T) {
	a := assert.New(t)

	name, err := factories["seven-card"].Name(playable.AdditionalData{
		"variant": "stud",
	})
	a.NoError(err)
	a.Equal("Seven-Card Stud", name)

	name, err = factories["seven-card"].Name(playable.AdditionalData{
		"variant": "baseball",
	})
	a.NoError(err)
	a.Equal("Baseball", name)

	name, err = factories["seven-card"].Name(playable.AdditionalData{
		"variant": "low-card-wild",
	})
	a.NoError(err)
	a.Equal("Low Card Wild", name)
}
