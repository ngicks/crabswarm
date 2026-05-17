package exec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func funcMap() template.FuncMap {
	return template.FuncMap{
		"env":      os.Getenv,
		"basename": filepath.Base,
		"dirname":  filepath.Dir,
		"ext":      filepath.Ext,
		"trim":     strings.TrimSpace,
		"quote":    shellQuote,
	}
}

// shellQuote returns s wrapped in POSIX shell single-quotes. Embedded
// single quotes are escaped using the standard close-quote / backslash-
// quote / open-quote sequence. Useful in templates when a string contains
// whitespace or shell-special characters that must survive
// [shellwords.Parse] as a single argument.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func renderTemplate(src string, data Data) (string, error) {
	tmpl, err := template.New("template").
		Option("missingkey=zero").
		Funcs(funcMap()).
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
