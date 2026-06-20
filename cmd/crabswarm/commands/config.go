package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/pkg/crabswarm"
	"github.com/ngicks/crabswarm/pkg/crabswarm/cli"
)

// configLongFmt documents the resolved-config object so users can write --format
// templates without reading the source. The template runs against the Go
// crabswarm.Config value, so field paths use the Go field names ({{.Sock}});
// the default JSON output uses the lower-case keys shown in parentheses. The %s
// is filled with the shared template-helper docs (see cli.TemplateFuncHelp).
const configLongFmt = `config loads every configuration layer (defaults < file < environment),
applies any explicitly-set flags on top, and prints the fully-resolved
configuration to stdout. With no flags it prints indented JSON; with --format
it renders a Go text/template against the config value instead.

The value passed to --format has this shape (Go field name -> JSON key):

  {
    .Sock           string  // unix socket path                 (sock)
    .ProjectDir     string  // $CLAUDE_PROJECT_DIR              (project_dir)
    .GitRepoBaseDir string  // git clone base directory         (git_repo_base_dir)
    .HookExec               // hook exec config                 (hook_exec)
      .Filetypes []  // one entry per filetype rule set         (filetypes)
        .Ext         map[string]string         // ext -> name   (ext)
        .Filename    map[string]string         // name -> name  (filename)
        .RootMarkers map[string][]MarkerGroup  // root markers  (root_markers)
  }

Use the Go field names in --format (e.g. {{.Sock}}); the default JSON output
uses the lower-case keys shown in parentheses. The template also sees the
shared helper functions:

%s`

const configExample = `  crabswarm config
  crabswarm config --format '{{.Sock}}'
  crabswarm config --format '{{.GitRepoBaseDir}} {{.ProjectDir}}'`

func configCmd(parent *cobra.Command, flagConfig *string) {
	var flagFormat string

	cmd := &cobra.Command{
		Use:               "config",
		Short:             "Print the resolved configuration",
		Long:              fmt.Sprintf(configLongFmt, cli.TemplateFuncHelp()),
		Example:           configExample,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(cmd, args, *flagConfig, flagFormat)
		},
	}

	cmd.Flags().StringVarP(
		&flagFormat,
		"format",
		"f",
		"",
		"Go text/template rendered against the resolved config instead of JSON",
	)

	parent.AddCommand(cmd)
}

func runConfig(cmd *cobra.Command, _ []string, flagConfig, flagFormat string) error {
	cfg, err := crabswarm.LoadConfig(flagConfig)
	if err != nil {
		return err
	}
	// Presentation (JSON / template rendering) lives in pkg/crabswarm/cli; ./cmd
	// only wires it to stdout. cmd.Println would route to stderr.
	return cli.RenderConfig(cmd.OutOrStdout(), cfg, flagFormat)
}
