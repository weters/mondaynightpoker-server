package oauth

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/sirupsen/logrus"
)

//go:embed login.gohtml error.gohtml
var templateFS embed.FS

var (
	loginTemplate = template.Must(template.ParseFS(templateFS, "login.gohtml"))
	errorTemplate = template.Must(template.ParseFS(templateFS, "error.gohtml"))
)

// loginPageData is the data model rendered into the login form.
type loginPageData struct {
	Error               string
	Nonce               string
	ClientID            string
	RedirectURI         string
	ResponseType        string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	State               string
	Resource            string
}

// renderLogin writes the login form with the given status and data.
func (s *Server) renderLogin(w http.ResponseWriter, statusCode int, data loginPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := loginTemplate.Execute(w, data); err != nil {
		logrus.WithError(err).Error("oauth: could not render login template")
	}
}

// renderErrorPage writes a standalone HTML error page (used when there is no safe
// redirect_uri to bounce the error back to).
func (s *Server) renderErrorPage(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := errorTemplate.Execute(w, message); err != nil {
		logrus.WithError(err).Error("oauth: could not render error template")
	}
}
