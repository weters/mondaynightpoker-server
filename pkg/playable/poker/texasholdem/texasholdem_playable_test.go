package texasholdem

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGame_Name(t *testing.T) {
	g, err := NewGame(logrus.StandardLogger(), []int64{1, 2}, DefaultOptions())
	assert.NoError(t, err)
	assert.Equal(t, "Limit Texas Hold'em (${100}/${200})", g.Name())

	g, err = NewGame(logrus.StandardLogger(), []int64{1, 2}, Options{
		LowerLimit: 200,
		UpperLimit: 400,
	})
	assert.NoError(t, err)
	assert.Equal(t, "Limit Texas Hold'em (${200}/${400})", g.Name())
}
