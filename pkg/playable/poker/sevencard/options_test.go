package sevencard

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOptions_Validate(t *testing.T) {
	a := assert.New(t)

	o := Options{}
	a.EqualError(o.Validate(), "ante must be greater than zero")

	o.Ante = -25
	a.EqualError(o.Validate(), "ante must be greater than zero")

	o.Ante = 26
	a.EqualError(o.Validate(), "ante must be divisible by 25")

	o.Ante = 25
	a.EqualError(o.Validate(), "seven-card variant must be specified")

	o.Variant = &Stud{}
	a.NoError(o.Validate())
}
