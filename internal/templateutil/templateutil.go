// Package templateutil centralizes the [text/template] helpers shared by the
// crabswarm template-rendering call sites (hook exec command rendering and the
// status line renderer). Keeping a single func map means every template,
// regardless of where it is rendered, sees the same set of functions.
package templateutil

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mattn/go-shellwords"
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
//	commandArgs CMD  → CommandArgs (shellwords split of CMD into its words)
//	commandName CMD  → CommandName (first shellwords word of CMD)
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"env":         os.Getenv,
		"basename":    filepath.Base,
		"dirname":     filepath.Dir,
		"ext":         filepath.Ext,
		"trim":        strings.TrimSpace,
		"quote":       ShellQuote,
		"quoteJoin":   QuoteJoin,
		"which":       Which,
		"commandArgs": CommandArgs,
		"commandName": CommandName,
	}
}

// FuncDoc documents a single helper exposed by FuncMap.
type FuncDoc struct {
	// Name is the bare function name as registered in FuncMap.
	Name string
	// Usage is the function name together with its argument placeholders,
	// e.g. "quote STRING".
	Usage string
	// Desc is a one-line human description of the helper.
	Desc string
}

// FuncDocs returns documentation for every helper in FuncMap, in a stable
// display order. It is the single source of truth behind FuncHelp and the
// command help text; it is kept in sync with FuncMap (guarded by a test).
func FuncDocs() []FuncDoc {
	return []FuncDoc{
		{
			Name:  "env",
			Usage: "env NAME",
			Desc:  "value of environment variable NAME (empty when unset)",
		},
		{Name: "basename", Usage: "basename PATH", Desc: "final element of PATH"},
		{Name: "dirname", Usage: "dirname PATH", Desc: "all but the final element of PATH"},
		{
			Name:  "ext",
			Usage: "ext PATH",
			Desc:  "file-name extension of PATH, including the leading dot",
		},
		{
			Name:  "trim",
			Usage: "trim STRING",
			Desc:  "STRING with leading and trailing whitespace removed",
		},
		{
			Name:  "quote",
			Usage: "quote STRING",
			Desc:  "STRING wrapped in POSIX shell single-quotes (embedded quotes escaped)",
		},
		{
			Name:  "quoteJoin",
			Usage: "quoteJoin LIST",
			Desc:  "each string in LIST double-quoted and joined with single spaces",
		},
		{
			Name:  "which",
			Usage: "which NAME",
			Desc:  "absolute path of command NAME resolved via $PATH (errors when missing)",
		},
		{
			Name:  "commandArgs",
			Usage: "commandArgs CMD",
			Desc:  "words of CMD's first simple command, shellwords-split (stops at unquoted ;, & or |)",
		},
		{
			Name:  "commandName",
			Usage: "commandName CMD",
			Desc:  "first shellwords word of CMD, i.e. the invoked command (empty when CMD is blank)",
		},
	}
}

// FuncHelp renders FuncDocs as an aligned, indented block suitable for
// embedding in command help text. Each line is "  <usage>  <desc>" with the
// usage column padded to a common width; the block ends with a trailing
// newline.
func FuncHelp() string {
	docs := FuncDocs()
	width := 0
	for _, d := range docs {
		width = max(width, len(d.Usage))
	}
	var b strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, d.Usage, d.Desc)
	}
	return b.String()
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

// CommandArgs splits command with [shellwords.Parse] — the same splitting
// the hook exec renderer applies to its own output — and returns the words
// of the first simple command: quoting and escaping are honored, nothing is
// expanded or evaluated, and parsing stops at the first unquoted shell
// operator (;, & or |), dropping everything after it. Intended for
// templating over a Bash tool_input.command, e.g.
//
//	{{ commandArgs .Input.ToolInput.GetValue.Command }}
//
// It returns an error when command cannot be parsed (e.g. an unclosed
// quote), so the template fails to render instead of acting on a bogus
// split.
func CommandArgs(command string) ([]string, error) {
	return shellwords.Parse(command)
}

// CommandName returns the first word of command as split by [CommandArgs],
// i.e. the name of the command being invoked, or "" when command is empty
// or whitespace-only. Like CommandArgs it errors when command cannot be
// parsed.
func CommandName(command string) (string, error) {
	args, err := CommandArgs(command)
	if err != nil || len(args) == 0 {
		return "", err
	}
	return args[0], nil
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
