// Package cli holds crabswarm's CLI-presentation code — rendering resolved
// configuration and other terminal output. The ./cmd wiring layer calls into
// this package and returns its error; presentation logic never lives under
// ./cmd itself.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/ngicks/crabswarm/pkg/crabswarm"
	"github.com/ngicks/crabswarm/pkg/internal/templateutil"
)

// TemplateFuncHelp returns the aligned help block for the helper functions
// available to a --format template, the same set every crabswarm template
// renderer exposes. Command help embeds it so the docs cannot drift from
// [templateutil.FuncMap].
func TemplateFuncHelp() string {
	return templateutil.FuncHelp()
}

// RenderConfig writes the resolved configuration to w.
//
// With format == "" it writes indented JSON. Otherwise format is parsed as a Go
// text/template and executed against cfg (the crabswarm.Config value, so field
// paths use the Go field names, e.g. {{.Sock}}); it sees the shared
// [templateutil.FuncMap] helpers (env, quote, basename, ...). Either form is
// terminated with a trailing newline. A malformed or failing template is
// returned as an error attributed to the --format flag.
func RenderConfig(w io.Writer, cfg crabswarm.Config, format string) error {
	if format != "" {
		tmpl, err := template.New("config").
			Funcs(templateutil.FuncMap()).
			Parse(format)
		if err != nil {
			return fmt.Errorf("--format: %w", err)
		}
		if err := tmpl.Execute(w, cfg); err != nil {
			return fmt.Errorf("--format: %w", err)
		}
		fmt.Fprintln(w)
		return nil
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(b))
	return nil
}
