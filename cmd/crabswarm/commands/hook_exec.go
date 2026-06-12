package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/cmd/internal/stdiopipe"
	"github.com/ngicks/crabswarm/pkg/crabswarm"
	"github.com/ngicks/crabswarm/pkg/crabswarm/hook/exec"
)

func hookExecCmd(parent *cobra.Command, flagConfig *string) {
	var (
		flagDryRun      bool
		flagDumpDefault bool
		flagFt          []string
	)

	cmd := &cobra.Command{
		Use:   "exec <template>",
		Short: "executes given commands in claude hook manner.",
		Long: fmt.Sprintf(`executes specified external commands in claude hook manner.

It executes commands expressed in Go text/template format.
The template receives the claude hook input as its data context, plus the
following helper functions:

%s
The final output of the template is treated as a shell string.
For interpolation behavior see github.com/mattn/go-shellwords

It also detects the "module root" and sets it to command's cwd.
exec ships with a built-in module-detection configuration that is
layered underneath any user config as the lowest-priority overlay —
your entries override defaults on key conflicts. See the output of
--dump-default-config to inspect the built-ins or use it as a
starting point for your own config.

The positional argument is the Go template to render and execute, e.g.:

  crabswarm hook exec 'golangci-lint run --fix {{ quote .File }}'

Use --ft to gate the invocation on the detected filetype. Pass the flag
multiple times (or comma-separated) to allow several. Invocations whose
detected filetype is not in the allow-list pass through without rendering
or executing anything, e.g.:

  crabswarm hook exec --ft go --ft rust 'echo {{ .File }}'
`, exec.TemplateFuncHelp()),
		// At most one template; --dump-default-config doesn't need one.
		// The positional is a Go text/template string, not a file path.
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHookExec(cmd, args, *flagConfig, flagDryRun, flagDumpDefault, flagFt)
		},
	}

	f := cmd.Flags()
	f.BoolVar(
		&flagDryRun,
		"dry-run",
		false,
		"Print rendered commands to stdout instead of executing them",
	)
	f.BoolVar(
		&flagDumpDefault,
		"dump-default-config",
		false,
		"Print the built-in default config as JSON and exit",
	)
	f.StringSliceVar(
		&flagFt,
		"ft",
		nil,
		"Only run when the detected filetype is in this list (repeatable; "+
			"empty means no filter)",
	)

	parent.AddCommand(cmd)
}

func runHookExec(
	cmd *cobra.Command, args []string,
	configOverride string, dryRun, dumpDefault bool, allowedFt []string,
) error {
	if dumpDefault {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(exec.Default()); err != nil {
			return fmt.Errorf("encoding default config: %w", err)
		}
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("a template positional argument is required")
	}

	ctx := cmd.Context()

	cfg, err := crabswarm.LoadConfig(configOverride)
	if err != nil {
		return err
	}

	opt := exec.Option{
		Template: args[0],
		Filter:   allowedFt,
	}

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	if dryRun {
		return exec.Render(ctx, reader, os.Stdout, cfg.HookExec, opt)
	}
	return exec.Run(ctx, reader, cfg.HookExec, opt)
}
