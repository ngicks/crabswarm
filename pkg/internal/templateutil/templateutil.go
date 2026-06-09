// Package templateutil centralizes the [text/template] helpers shared by the
// crabswarm template-rendering call sites (hook exec command rendering and the
// status line renderer). Keeping a single func map means every template,
// regardless of where it is rendered, sees the same set of functions.
package templateutil

import (
	"os"
	"os/exec"
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
//	quoteJoin LIST   → QuoteJoin (double-quote each element, join with spaces)
//	which NAME       → Which (resolve a command to its absolute path)
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"env":       os.Getenv,
		"basename":  filepath.Base,
		"dirname":   filepath.Dir,
		"ext":       filepath.Ext,
		"trim":      strings.TrimSpace,
		"quote":     ShellQuote,
		"quoteJoin": QuoteJoin,
		"which":     Which,
	}
}

// Which resolves command name to its absolute path, like the which(1) shell
// command: it searches the directories in $PATH (via [exec.LookPath]) and
// returns the absolute path of the first match. A name containing a path
// separator is resolved directly relative to the working directory rather than
// searched in $PATH. It returns an error when name cannot be found or is not
// executable, so a template referencing a missing command fails to render
// instead of emitting a broken command line.
func Which(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	return filepath.Abs(path)
}

// ShellQuote returns s wrapped in POSIX shell single-quotes. Embedded single
// quotes are escaped using the standard close-quote / backslash-quote /
// open-quote sequence. Useful in templates when a string contains whitespace
// or shell-special characters that must survive [shellwords.Parse] as a single
// argument.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// QuoteJoin wraps each element of ss in double quotes and joins the results
// with a single space, e.g. {"a", "b c"} → `"a" "b c"`. Embedded backslashes
// and double quotes are escaped (\\ and \") so every element survives
// [shellwords.Parse] as a single argument. It is the []string counterpart to
// ShellQuote, handy for templating a whole argument list into a command line.
func QuoteJoin(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		quoted[i] = `"` + s + `"`
	}
	return strings.Join(quoted, " ")
}
