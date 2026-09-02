package commands

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/statusline"
	"github.com/ngicks/crabswarm/internal/stdiopipe"
)

//nolint:lll // Long/Example embed one-line settings.json / shell examples that must stay unwrapped to be copy-pasteable.
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

Template functions:

` + statusline.TemplateFuncHelp() + `
Narrow terminals: when Claude Code exports $COLUMNS, {{ columns }} returns
it as an integer (0 when unset), so a template can pick a shorter layout,
e.g. keep the last runes of the directory with truncRuneLeft. When $COLUMNS
is missing, fall back to the last few path components with
{{ splitPath .Workspace.CurrentDir | lastN 3 | join "/" }}. The pad/trunc
helpers measure terminal cells (a CJK character or emoji is two), so padded
columns line up and truncated text fits the width it was given.

Configure it in Claude Code settings.json, for example:

  {
    "statusLine": {
      "type": "command",
      "command": "crabswarm statusline render '{{ .Model.DisplayName }} | {{ basename .Workspace.CurrentDir }}{{ with .Workspace.GitWorktree }} ⑂{{ . }}{{ end }} | {{ .ContextWindow.UsedPercentage }}%'"
    }
  }
`,
		Example: `  # "model @ effort | context usage in percent | cwd"
  crabswarm statusline render '{{.Model.DisplayName}}{{ with .Effort }} @ {{ .Level }}{{ end }} | {{.ContextWindow.UsedPercentage}}% used | {{.Workspace.CurrentDir}}'

  # Fixed-width model column; cwd trimmed to fit a narrow terminal, or its
  # last three components when $COLUMNS is not exported
  crabswarm statusline render '{{ padRuneRight 8 .Model.DisplayName }}| {{ if and columns (lt columns 80) }}…{{ truncRuneLeft 24 .Workspace.CurrentDir }}{{ else if columns }}{{ .Workspace.CurrentDir }}{{ else }}{{ splitPath .Workspace.CurrentDir | lastN 3 | join "/" }}{{ end }}'`,
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
