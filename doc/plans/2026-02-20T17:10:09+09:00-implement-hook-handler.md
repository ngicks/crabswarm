# Implement Claude Code Hook Handler

## Context

Claude Code hooks are shell commands that run at specific lifecycle points. A hook receives JSON on stdin and communicates results via exit codes and stdout/stderr:

- **Exit 0**: success. Claude Code parses **stdout** for JSON decision fields.
- **Exit 2**: blocking error. Claude Code reads **stderr** as plain text error message. Stdout is **ignored**.
- **Other exit codes**: non-blocking error, stderr shown in verbose mode.

These are two distinct paths:
- JSON decision output (e.g. `{"decision":"block","reason":"..."}`) → **stdout, exit 0**
- Blocking error (plain text) → **stderr, exit 2**

We need a `HandlerError` type that hook subcommands return from cobra `RunE`. The `Execute` wrapper in `root.go` intercepts it and calls `Handle()`, which writes to the correct stream and exits with the correct code.

## Files to modify

- `pkg/claudehook/handler/handler.go` — `HandlerError`, `HookOutput`, typed constants, `Handle()`
- `cmd/crabswarm/commands/root.go` — intercept `HandlerError`
- `cmd/crabswarm/commands/hook.go` — set `SilenceErrors`/`SilenceUsage` on `hookCmd` only
- `cmd/crabswarm/commands/hook_audit.go` — return `HandlerError` on success

## Changes

### 1. `pkg/claudehook/handler/handler.go`

```go
package handler

import (
	"encoding/json"
	"fmt"
	"os"
)

// --- Typed string constants ---

// HookDecision is the top-level "decision" field value.
// Used by: UserPromptSubmit, PostToolUse, PostToolUseFailure, Stop, SubagentStop, ConfigChange.
// See https://code.claude.com/docs/en/hooks#decision-control
type HookDecision string

const (
	HookDecisionBlock HookDecision = "block"
)

// PermissionDecision is the "permissionDecision" field inside hookSpecificOutput for PreToolUse.
// See https://code.claude.com/docs/en/hooks#pretooluse-decision-control
type PermissionDecision string

const (
	PermissionDecisionAllow PermissionDecision = "allow"
	PermissionDecisionDeny  PermissionDecision = "deny"
	PermissionDecisionAsk   PermissionDecision = "ask"
)

// PermissionRequestBehavior is the "behavior" field inside hookSpecificOutput.decision for PermissionRequest.
// See https://code.claude.com/docs/en/hooks#permissionrequest-decision-control
type PermissionRequestBehavior string

const (
	PermissionRequestBehaviorAllow PermissionRequestBehavior = "allow"
	PermissionRequestBehaviorDeny  PermissionRequestBehavior = "deny"
)

// --- Hook-specific output structs ---

// PreToolUseOutput is the hookSpecificOutput for PreToolUse events.
type PreToolUseOutput struct {
	HookEventName          string             `json:"hookEventName"`
	PermissionDecision     PermissionDecision `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string           `json:"permissionDecisionReason,omitempty"`
	UpdatedInput           any                `json:"updatedInput,omitempty"`
	AdditionalContext      string             `json:"additionalContext,omitempty"`
}

// PermissionRequestDecision is the nested decision object for PermissionRequest hookSpecificOutput.
type PermissionRequestDecision struct {
	Behavior         PermissionRequestBehavior `json:"behavior"`
	UpdatedInput     any                       `json:"updatedInput,omitempty"`
	Message          string                    `json:"message,omitempty"`
	Interrupt        *bool                     `json:"interrupt,omitempty"`
}

// PermissionRequestOutput is the hookSpecificOutput for PermissionRequest events.
type PermissionRequestOutput struct {
	HookEventName string                     `json:"hookEventName"`
	Decision      *PermissionRequestDecision `json:"decision,omitempty"`
}

// SessionStartOutput is the hookSpecificOutput for SessionStart events.
type SessionStartOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// UserPromptSubmitOutput is the hookSpecificOutput for UserPromptSubmit events.
type UserPromptSubmitOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// PostToolUseOutput is the hookSpecificOutput for PostToolUse events.
type PostToolUseOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// PostToolUseFailureOutput is the hookSpecificOutput for PostToolUseFailure events.
type PostToolUseFailureOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// SubagentStartOutput is the hookSpecificOutput for SubagentStart events.
type SubagentStartOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}

// --- HookOutput ---

// HookOutput is the JSON written to stdout on exit 0.
// See https://code.claude.com/docs/en/hooks#json-output
type HookOutput struct {
	// Universal fields
	Continue       *bool  `json:"continue,omitempty"`
	SuppressOutput *bool  `json:"suppressOutput,omitempty"`
	StopReason     string `json:"stopReason,omitempty"`
	SystemMessage  string `json:"systemMessage,omitempty"`

	// Top-level decision (for UserPromptSubmit, PostToolUse, PostToolUseFailure, Stop, SubagentStop, ConfigChange)
	Decision HookDecision `json:"decision,omitempty"`
	Reason   string       `json:"reason,omitempty"`

	// Event-specific output
	HookSpecificOutput any `json:"hookSpecificOutput,omitempty"`
}

// --- HandlerError ---

// HandlerError is returned by hook subcommands to signal the hook result.
// It implements error so cobra RunE can return it.
//
// Two modes:
//   - Output != nil: marshal Output as JSON to stdout, exit 0.
//     (Even "block" decisions go to stdout with exit 0 — Claude Code parses the JSON.)
//   - BlockError != "": write BlockError to stderr, exit 2.
//     (This is the "blocking error" path — no JSON, just plain text.)
//   - Both nil/empty: exit 0, no output (allow).
type HandlerError struct {
	// Output is the structured JSON to write to stdout on exit 0.
	Output *HookOutput

	// BlockError is a plain text message to write to stderr on exit 2.
	// Mutually exclusive with Output.
	BlockError string
}

func (e *HandlerError) Error() string {
	if e.BlockError != "" {
		return fmt.Sprintf("hook: block error: %s", e.BlockError)
	}
	if e.Output != nil && e.Output.Decision == HookDecisionBlock {
		return fmt.Sprintf("hook: block: %s", e.Output.Reason)
	}
	return "hook: allow"
}

// Handle writes output and exits the process.
func (e *HandlerError) Handle() {
	// Exit 2 path: plain text to stderr
	if e.BlockError != "" {
		fmt.Fprint(os.Stderr, e.BlockError)
		os.Exit(2)
	}

	// No output: exit 0
	if e.Output == nil {
		os.Exit(0)
	}

	// JSON output to stdout, exit 0
	data, err := json.Marshal(e.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hook: failed to marshal output: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, string(data))
	os.Exit(0)
}
```

### 2. `cmd/crabswarm/commands/root.go`

Add `HandlerError` interception in `Execute`. Non-hook commands (e.g. `serve`) are unaffected — they return regular errors.

```go
package commands

import (
	"context"
	"errors"

	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/spf13/cobra"
)

func Execute(ctx context.Context) error {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		if he, ok := errors.AsType[*handler.HandlerError](err); ok {
			he.Handle() // writes output, calls os.Exit
		}
	}
	return err
}

var rootCmd = &cobra.Command{
	Use:   "crabswarm",
	Short: "crabswarm CLI",
	Long:  `crabswarm is a CLI tool for managing Claude Code hooks.`,
}

func init() {
	rootCmd.PersistentFlags().String("sock", "", "Unix socket path")
}
```

### 2b. `cmd/crabswarm/commands/hook.go`

Silence cobra error/usage output **only for hook subcommands**, so `serve` and other commands retain normal cobra error behavior.

```go
package commands

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(hookCmd)
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Hook management commands",
	// Silence cobra output for hook commands — we control stdout/stderr via HandlerError.Handle().
	SilenceErrors: true,
	SilenceUsage:  true,
}
```

### 3. `cmd/crabswarm/commands/hook_audit.go`

Change the return from `return nil` to:

```go
return &handler.HandlerError{}
```

Add import: `"github.com/ngicks/crabswarm/pkg/claudehook/handler"`.

The audit hook is observation-only — it sends the input to the backend and always allows (exit 0, no JSON output).

## Design notes

- **Success-via-error pattern**: Hook subcommands return `HandlerError` even on success. This is intentional — it ensures `Handle()` always runs for hook commands, giving us full control over exit codes and output streams. Non-hook commands (e.g. `serve`) return regular errors unaffected.
- **`hookEventName` in output structs**: Required by the Claude Code spec. The JSON `hookSpecificOutput` object must include `hookEventName` set to the event name.
- **`any` for `UpdatedInput` and `HookSpecificOutput`**: `UpdatedInput` varies per tool (Bash input vs Edit input vs ...). `HookSpecificOutput` on `HookOutput` uses `any` so callers pass typed structs (e.g. `PreToolUseOutput`, `PermissionRequestOutput`) which are marshaled correctly.
- **Custom types instead of proto types for output**: The proto types (`SyncHookJSONOutput`, `HookSpecificOutput`) use oneof wrappers that serialize to a different JSON shape than Claude Code expects. Proto types produce `{"preToolUse": {...}}` while Claude Code expects `{"hookEventName": "PreToolUse", ...}`. Proto types are used for **input** parsing only.

## Verification

1. `go build ./...` — compiles
2. `go vet ./...` — no issues
3. `go test ./...` — passes
