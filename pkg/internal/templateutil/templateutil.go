// Package templateutil centralizes the [text/template] helpers shared by the
// crabswarm template-rendering call sites (hook exec command rendering and the
// status line renderer). Keeping a single func map means every template,
// regardless of where it is rendered, sees the same set of functions.
package templateutil

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// FuncMap returns the template function map shared across crabswarm's template
// renderers. A fresh map is returned on each call so callers may mutate it
// without affecting one another.
//
//	env STRING       → os.Getenv lookup
//	basename PATH    → filepath.Base
//	dirname PATH     → filepath.Dir
//	ext PATH         → filepath.Ext
//	trim STRING      → strings.TrimSpace
//	quote STRING     → ShellQuote (POSIX shell single-quoting)
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"env":      os.Getenv,
		"basename": filepath.Base,
		"dirname":  filepath.Dir,
		"ext":      filepath.Ext,
		"trim":     strings.TrimSpace,
		"quote":    ShellQuote,
	}
}

// ShellQuote returns s wrapped in POSIX shell single-quotes. Embedded single
// quotes are escaped using the standard close-quote / backslash-quote /
// open-quote sequence. Useful in templates when a string contains whitespace
// or shell-special characters that must survive [shellwords.Parse] as a single
// argument.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
