package exec

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/ngicks/crabswarm/internal/templateutil"
)

// TemplateFuncHelp returns aligned help text documenting the helper functions
// available to hook exec templates. The functions themselves come from
// [templateutil.FuncMap]; this is exposed so the CLI can embed the same
// documentation in its help message without reaching into the internal
// templateutil package.
func TemplateFuncHelp() string {
	return templateutil.FuncHelp()
}

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
