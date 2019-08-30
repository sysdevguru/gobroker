package templates

import (
	"bytes"
	"html/template"

	"github.com/alpacahq/gobroker/mailer/templates/layouts"
	"github.com/alpacahq/gobroker/mailer/templates/partials"
)

func ExecuteTemplate(layout layouts.Layout, content partials.Partial, data interface{}) (string, error) {
	tmpl, err := template.New("").Parse(string(layout) + string(content))

	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)

	if err := tmpl.ExecuteTemplate(buf, "layout", data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
