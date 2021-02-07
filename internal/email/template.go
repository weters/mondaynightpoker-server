package email

import (
	"bytes"
	"html/template"
	"path/filepath"
)

// Template is a light-weight wrapper around html/template.Template
type Template struct {
	*template.Template
}

// NewTemplate loads templates in a directory and returns a new Template object
func NewTemplate(dir string) (*Template, error) {
	tpl, err := template.ParseGlob(filepath.Join(dir, "*.html"))
	if err != nil {
		return nil, err
	}

	return &Template{tpl}, nil
}

// RenderTemplate will render the template and return a string
func (t *Template) RenderTemplate(name string, data interface{}) (string, error) {
	buf := bytes.Buffer{}
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
