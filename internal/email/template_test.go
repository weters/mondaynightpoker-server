package email

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRenderTemplate(t *testing.T) {
	a := assert.New(t)
	tpl, err := NewTemplate("testdata")
	a.NoError(err)
	out, err := tpl.RenderTemplate("file1.html", map[string]string{"Var": "My Variable"})
	a.NoError(err)
	a.Equal("<p>File 1 My Variable</p>", out)

	out, err = tpl.RenderTemplate("file2.html", map[string]string{"Var": "Another Variable"})
	a.NoError(err)
	a.Equal("<p>File 2 Another Variable</p>", out)
}
