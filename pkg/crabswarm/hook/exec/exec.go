// Package exec implements `crabswarm hook exec`: a declarative way to wire
// Claude Code and Codex hooks. The template is expressed in Go
// text/template form; the rendered output is split with [shellwords] and
// executed directly (no shell). The child process's working directory is
// set to the detected module root.
//
// [shellwords]: https://github.com/mattn/go-shellwords
package exec

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	osexec "os/exec"
	"slices"
	"strings"

	"github.com/mattn/go-shellwords"

	sdktypesv1 "github.com/ngicks/crabswarm/pkg/api/types/sdktypes/v1"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/ngicks/crabswarm/pkg/filetype"
)

//go:embed default_config.json
var defaultConfigJSON []byte

// Config is the persistent configuration for the exec hook — it's the
// `hook_exec` block of the crabswarm global config. Per-invocation knobs
// live on [Option], passed alongside Config to [Render] / [Run].
//
// JSON shape:
//
//	{
//	  "filetypes": [
//	    {
//	      "ext":          { "<ext>": "<filetype>", ... },
//	      "filename":     { "<file>": "<filetype>", ... },
//	      "root_markers": { "<filetype>": [["<marker>", ...], ...], ... }
//	    },
//	    ...
//	  ]
//	}
//
// Each filetypes element is a self-contained leaf with the same shape as
// the merged table (see [filetype.Config]). The runtime folds the slice
// via [filetype.MergeAll]; later leaves win on key conflicts. The
// built-in [Default] leaves are folded underneath at the lowest priority
// so they apply out of the box; caller-supplied entries override them.
type Config struct {
	Filetypes []filetype.Config `json:"filetypes,omitzero"`
}

// Option holds the per-invocation knobs (template + filetype gate) that
// the caller supplies on each [Render] / [Run] call.
type Option struct {
	// Template is the Go text/template to render against [Data]. Required.
	Template string
	// Filter, when non-empty, gates the invocation on the detected
	// filetype: when the detected name is not in this list, the call
	// passes through without rendering or executing.
	Filter []string
}

// Data is the value passed to the template at render time.
type Data struct {
	// Input is the parsed Claude / Codex hook envelope. Concrete type
	// depends on the event, e.g. [*sdktypesv1.PreToolUseHookInput].
	Input sdktypesv1.HookInput
	// Event mirrors hook_event_name from the envelope.
	Event sdktypesv1.HookEvent
	// ToolName mirrors tool_name for tool events; empty otherwise.
	ToolName string
	// File is a best-effort path of the edited file; empty when not
	// applicable (e.g. Bash tool, non-tool events). For Claude it's
	// tool_input.file_path; for a Codex apply_patch it's the first path
	// in Files. Filetype and Root are detected from this path.
	File string
	// Files holds every edited file for tools that touch more than one in
	// a single call — notably Codex's apply_patch, where the paths are
	// resolved to absolute against Cwd. For single-file Claude edits it's
	// a one-element slice; empty when File is empty.
	Files []string
	// Filetype is the detected filetype Name; empty when nothing matches.
	Filetype string
	// Root is the detected module root. At Run time, the child process's
	// working directory is set to Root when non-empty.
	Root string
	// Cwd mirrors the envelope's cwd; empty for events without one.
	Cwd string
	// SessionID mirrors session_id; empty for events without one.
	SessionID string
}

// Default returns a fresh copy of the built-in configuration: a curated
// set of common filetypes with conventional module-root markers, embedded
// from default_config.json at build time.
//
// [Run] and [Render] layer Default underneath the caller-supplied
// [Config.Filetypes] as the lowest-priority overlay, so the built-ins
// apply out of the box and the caller's leaves win on key conflicts. The
// CLI also exposes the full snapshot via --dump-default-config for
// inspection or as a starting point for a user config.
func Default() Config {
	var c Config
	if err := json.Unmarshal(defaultConfigJSON, &c); err != nil {
		panic(fmt.Errorf("exec: invalid embedded default_config.json: %w", err))
	}
	return c
}

// Render reads a hook envelope from r, renders opt.Template against the
// parsed input, and writes the rendered text to w. Honors opt.Filter.
// Useful for inspecting what Run would execute.
func Render(ctx context.Context, r io.Reader, w io.Writer, cfg Config, opt Option) error {
	rendered, _, err := prepare(r, cfg, opt)
	if err != nil {
		return err
	}
	if rendered == "" {
		return &handler.HandlerError{}
	}
	if _, err := fmt.Fprintln(w, rendered); err != nil {
		return fmt.Errorf("writing rendered template: %w", err)
	}
	return &handler.HandlerError{}
}

// Run reads a hook envelope from r, renders opt.Template, splits the
// result with [shellwords.Parse], and executes it directly (no shell).
// The child process's working directory is set to the detected module
// root when available. Captures combined stdout+stderr; on non-zero exit
// returns a block-decision [handler.HandlerError] with the captured
// output as the reason so the agent sees the failure.
// Honors opt.Filter.
func Run(ctx context.Context, r io.Reader, cfg Config, opt Option) error {
	rendered, data, err := prepare(r, cfg, opt)
	if err != nil {
		return err
	}
	if rendered == "" {
		return &handler.HandlerError{}
	}

	args, err := shellwords.Parse(rendered)
	if err != nil {
		return fmt.Errorf("parsing rendered command %q: %w", rendered, err)
	}
	if len(args) == 0 {
		return &handler.HandlerError{}
	}

	cmd := osexec.CommandContext(ctx, args[0], args[1:]...)
	if data.Root != "" {
		cmd.Dir = data.Root
	}

	bin, err := cmd.CombinedOutput()
	if err == nil {
		return &handler.HandlerError{}
	}

	// Only a non-zero exit from a process that actually ran (ExitError)
	// represents a lint-style "tool said no" outcome we should surface
	// back to the agent as a block. Anything else — command not found,
	// permission denied, context cancellation, I/O on the pipes — is an
	// infrastructure problem that should propagate as a plain error.
	//
	// TODO: allow narrowing further by exit-code allow-list once we have
	// a concrete need (e.g. Option.BlockExitCodes []int).
	exitErr, ok := errors.AsType[*osexec.ExitError](err)
	if !ok {
		return fmt.Errorf("running %q: %w", rendered, err)
	}

	return handler.Block(blockReason(rendered, exitErr, string(bin)))
}

// prepare reads the envelope, applies the filter gate, and renders the
// template. Returns ("", data, nil) when the gate blocks or the template
// is empty.
func prepare(r io.Reader, cfg Config, opt Option) (string, Data, error) {
	if opt.Template == "" {
		return "", Data{}, fmt.Errorf("exec: opt.Template is required")
	}

	bin, err := io.ReadAll(r)
	if err != nil {
		return "", Data{}, fmt.Errorf("reading stdin: %w", err)
	}

	// Default leaves go first so caller-supplied leaves take precedence
	// on key conflicts (later leaves win in MergeAll).
	ft := filetype.MergeAll(slices.Concat(Default().Filetypes, cfg.Filetypes))
	data, err := parseInput(bin, ft)
	if err != nil {
		return "", Data{}, err
	}

	if !opt.allowsFiletype(data.Filetype) {
		return "", data, nil
	}

	rendered, err := renderTemplate(opt.Template, data)
	if err != nil {
		return "", data, err
	}
	return rendered, data, nil
}

// allowsFiletype reports whether name passes the filter. An empty Filter
// disables the gate.
func (o Option) allowsFiletype(name string) bool {
	if len(o.Filter) == 0 {
		return true
	}
	return slices.Contains(o.Filter, name)
}

// blockReason formats a captured command failure for the LLM. The shape
// is informational; the agent reads it as the hook's block reason.
func blockReason(command string, runErr error, output string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "command failed: %s\n", command)
	fmt.Fprintf(&sb, "exit: %v\n", runErr)
	output = strings.TrimRight(output, "\n")
	if output != "" {
		fmt.Fprintf(&sb, "output:\n%s\n", output)
	}
	return sb.String()
}
