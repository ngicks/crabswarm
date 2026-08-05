package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm"
	"github.com/ngicks/crabswarm/crabswarm/hook/exec"
	"github.com/ngicks/crabswarm/internal/stdiopipe"
)

// hookExecLong is the `hook exec` Long help. It sits at package level, not
// inline, so the //nolint:lll its copy-pasteable examples need — one runs past
// 100 columns and wrapping it would break the copy-paste — covers the help
// text alone and leaves the command wiring below linted.
//
// Concatenated, not Sprintf'd: the output-template examples contain literal
// printf verbs.
//
//nolint:lll // help-text examples must stay on one line
var hookExecLong = `executes specified external commands in claude hook manner.

It executes commands expressed in Go text/template format.
The template receives the claude hook input as its data context, plus the
following helper functions:

` + exec.TemplateFuncHelp() + `
The final output of the template is treated as a shell string.
For interpolation behavior see github.com/mattn/go-shellwords

It also detects the "module root" and sets it to command's cwd.
exec ships with a built-in module-detection configuration that is
layered underneath any user config as the lowest-priority overlay —
your entries override defaults on key conflicts. See the output of
--dump-default-config to inspect the built-ins or use it as a
starting point for your own config.

The first positional argument is the Go template to render and execute, e.g.:

  crabswarm hook exec 'golangci-lint run --fix {{ quote .File }}'

Use --ft to gate the invocation on the detected filetype. Pass the flag
multiple times (or comma-separated) to allow several. Invocations whose
detected filetype is not in the allow-list pass through without rendering
or executing anything, e.g.:

  crabswarm hook exec --ft go --ft rust 'echo {{ .File }}'

The optional second positional argument is the output template: a Go
text/template shaping the hook's JSON output. It renders after the command
ran — a gated or empty invocation skips it — and receives everything the
command template saw plus .Command, .ExitCode, .Success, .Error (the run
error's message, e.g. "exit status 1"; empty on success), .Stdout, .Stderr
and .Output (combined). Omit it to keep the built-in behavior: block on a
non-zero exit with the captured output as the reason.

The output template speaks only through the functions below. Each records
one field and renders as the empty string, so any other text it emits is an
error:

` + exec.OutputFuncHelp() + `
A close analogue of the built-in failure handling — it differs in always
emitting the output: section, which the built-in omits when the command
captured nothing — and success-path context injection:

  crabswarm hook exec 'golangci-lint run {{quote .File}}' \
    '{{if not .Success}}{{blockDecision (printf "command failed: %s\nexit: %s\noutput:\n%s" .Command .Error .Output)}}{{end}}'

  crabswarm hook exec --ft go 'go vet ./...' \
    '{{if .Success}}{{context "go vet passed"}}{{else}}{{blockDecision .Output}}{{end}}'
`

func hookExecCmd(parent *cobra.Command, flagConfig *string) {
	var (
		flagDryRun      bool
		flagDumpDefault bool
		flagFt          []string
	)

	cmd := &cobra.Command{
		Use:   "exec <template> [output-template]",
		Short: "executes given commands in claude hook manner.",
		Long:  hookExecLong,
		// The command template, then the optional output template;
		// --dump-default-config needs neither. Both positionals are Go
		// text/template strings, not file paths.
		Args:              cobra.MaximumNArgs(2),
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
	if len(args) > 1 {
		opt.OutputTemplate = args[1]
	}

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	if dryRun {
		return exec.Render(ctx, reader, os.Stdout, cfg.HookExec, opt)
	}
	return exec.Run(ctx, reader, cfg.HookExec, opt)
}
