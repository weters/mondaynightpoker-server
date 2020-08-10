package passthepoop

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	a := assert.New(t)
	a.Equal(75, opts.Ante)
	a.Equal((&StandardEdition{}).Name(), opts.Edition.Name())
	a.Equal(3, opts.Lives)
	a.Equal(false, opts.AllowBlocks)
}
