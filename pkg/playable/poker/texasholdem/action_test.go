package texasholdem

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_newAction(t *testing.T) {
	a := assert.New(t)

	action, err := newAction(betKey, 50)
	a.NoError(err)
	a.Equal(50, action.Amount)
	a.Equal(betKey, action.Name)

	action, err = newAction("bad", 50)
	a.EqualError(err, "bad is not a valid action")
	a.Equal(Action{}, action)
}
