package exec

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/ngicks/crabswarm/pkg/internal/templateutil"
)

func renderTemplate(src string, data Data) (string, error) {
	tmpl, err := template.New("template").
		Option("missingkey=zero").
		Funcs(templateutil.FuncMap()).
		Parse(src)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering template: %w", err)
	}
	return buf.String(), nil
}
