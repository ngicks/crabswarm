package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/cmd/internal/stdiopipe"
	"github.com/ngicks/crabswarm/pkg/crabswarm/statusline"
)

func statuslineRenderCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "render <template>",
		Short: "Render a Go text/template against the status line JSON from stdin",
		Long: `render reads Claude Code's status line JSON from stdin and renders the
positional argument as a Go text/template against it. The rendered output is
written to stdout verbatim (no trailing newline is added), which Claude Code
shows as the status line.

The template data is the parsed payload. Field paths are the PascalCase of the
documented JSON keys, e.g. context_window.used_percentage becomes
{{ .ContextWindow.UsedPercentage }} (abbreviations are mixed case: .Model.Id,
.Pr.Url). Objects Claude Code omits in some sessions (effort, vim, agent, pr,
worktree, workspace.repo) are nil when absent, so guard them with {{ with }} or
{{ if }}.

Template functions: env, basename, dirname, ext, trim.

Configure it in Claude Code settings.json, for example:

  {
    "statusLine": {
      "type": "command",
      "command": "crabswarm statusline render '{{ .Model.DisplayName }} | {{ basename .Workspace.CurrentDir }}{{ with .Workspace.GitWorktree }} ⑂{{ . }}{{ end }} | {{ .ContextWindow.UsedPercentage }}%'"
    }
  }
`,
		Example: `  # "model@effort | context usage in percent | cwd"
  crabswarm statusline render '{{ .Model.DisplayName }}{{ with .Effort }}@{{ .Level }}{{ end }} | {{ .ContextWindow.UsedPercentage }}% | {{ .Workspace.CurrentDir }}'`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              runStatuslineRender,
	}

	parent.AddCommand(cmd)
}

func runStatuslineRender(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	return statusline.Render(reader, os.Stdout, args[0])
}
