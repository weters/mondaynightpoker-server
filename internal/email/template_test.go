package email

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestRenderTemplate_passwordReset(t *testing.T) {
	a := assert.New(t)
	tpl, err := NewTemplate(filepath.Join("..", "..", "templates"))
	a.NoError(err)

	out, err := tpl.RenderTemplate("password_reset.html", map[string]string{
		"url":   "https://example.com/reset-password/token123",
		"email": "player@example.com",
		"host":  "https://example.com",
	})
	a.NoError(err)
	a.Contains(out, `href="https://example.com/reset-password/token123"`)
	a.Contains(out, "Reset Password</a>")
	a.Contains(out, "This email was intended for player@example.com")
	a.Contains(out, `src="https://example.com/monday-night-poker@2x.png"`)
}

func TestRenderTemplate_verifyAccount(t *testing.T) {
	a := assert.New(t)
	tpl, err := NewTemplate(filepath.Join("..", "..", "templates"))
	a.NoError(err)

	out, err := tpl.RenderTemplate("verify_account.html", map[string]string{
		"url":   "https://example.com/verify-account/token123",
		"email": "player@example.com",
		"host":  "https://example.com",
	})
	a.NoError(err)
	a.Contains(out, `href="https://example.com/verify-account/token123"`)
	a.Contains(out, "Verify Account</a>")
	a.Contains(out, "You have registered player@example.com for an account")
	a.Contains(out, "This email was intended for player@example.com")
	a.Contains(out, `src="https://example.com/monday-night-poker@2x.png"`)
}
