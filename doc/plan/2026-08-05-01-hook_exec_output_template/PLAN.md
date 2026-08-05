# hook exec output template

Add a second positional argument to `crabswarm hook exec` — a Go text/template
that shapes the hook's JSON output — with a template function for each hook
output field/variant.

## Goal / success criteria

- `crabswarm hook exec <command-template> [output-template]` accepts an
  optional second positional.
- The output template renders against the command's execution result and can
  produce any `types.SyncHookJSONOutput` shape via dedicated template
  functions (`blockDecision`, `context`, `permission`, `updatedToolOutput`, …).
- Omitting the output template preserves today's behavior exactly
  (block-on-nonzero-exit with captured output as reason).
- The built-in behavior is expressible as an output template, proving the
  machinery is complete. Guarded by
  `TestRun_OutputTemplate_RestatesBuiltinBlockReason`
  (`crabswarm/hook/exec/exec_test.go`), which compares a restating template's
  block reason against `blockReason`'s byte-for-byte, with and without
  captured output. `.Error` exists so that restatement can reproduce the
  built-in's `exit:` line.
- Unit tests in `crabswarm/hook/exec` cover the new funcs; e2e covers the CLI
  path.

## Scope / non-goals

- Scope: `crabswarm hook exec` only. `hook audit` and the status-line
  renderer keep using plain `templateutil.FuncMap`.
- Non-goal: async hook outputs (`AsyncHookJSONOutput`) — sync only.
- Non-goal: reading the output template from a file (inline string only, same
  as the command template).

## Context

- `cmd/crabswarm/commands/hook_exec.go` — cobra wiring; currently
  `cobra.MaximumNArgs(1)`, positional = command template.
- `crabswarm/hook/exec/exec.go` — `Option{Template, Filter}`, `Run`/`Render`,
  `prepare`; on non-zero exit builds `handler.Block(blockReason(...))` from
  `CombinedOutput`.
- `crabswarm/hook/exec/render.go` — command-template rendering via
  `internal/templateutil.FuncMap` (`missingkey=zero`).
- `pkg/claudehook/handler/handler.go` — `HandlerError{Output
  *types.SyncHookJSONOutput}`; `Handle` exits 2 + stderr for plain
  decision=block, else marshals JSON to stdout and exits 0.
- `pkg/claudehook/types/types.go:4457` — `SyncHookJSONOutput` (continue,
  suppressOutput, stopReason, decision, systemMessage, terminalSequence,
  reason, hookSpecificOutput) and the ~19 `HookSpecificOutput*` variants.
- `internal/templateutil` — generic string helpers + `FuncDocs`/`FuncHelp`
  self-documentation pattern (guarded by `TestFuncDocs_MatchesFuncMap`).

## Approach

The output template is a small DSL over a mutable output builder: each hook
output function is a **side-effect function** that records a field on an
internal `*types.SyncHookJSONOutput` and renders as the empty string. After
`Execute`, exec marshals whatever accumulated into a `handler.HandlerError`.
Control flow stays plain `text/template` (`if`/`printf`/pipelines), and the
generic `templateutil.FuncMap` helpers remain available. *(Decided — Q1.)*

Coverage is **full**: a function for every `SyncHookJSONOutput` field and
every `HookSpecificOutput*` variant, per the tables below. *(Decided — Q5.)*

Event-scoped functions **error at call time** when the incoming event doesn't
support them; the render fails and exec exits with a plain (non-hook) error
so misconfiguration surfaces instead of being silently ignored.
*(Decided — Q3.)*

Rejected alternatives:

- *Functions return JSON text; rendered output = stdout verbatim.* Puts JSON
  assembly on the user (escaping, one-object-only), loses the exit-2 vs
  exit-0 protocol handling that `handler.Handle` already owns.
- *Flag-based output shaping (`--on-fail-decision=block` …).* Doesn't scale
  to ~19 hookSpecificOutput variants; template functions were requested.

### CLI surface

```
crabswarm hook exec <command-template> [output-template]

# a close analogue of the built-in failure handling — it always emits the
# output: section, which the built-in omits when nothing was captured (the
# exact restatement needs a conditional section; see the test named above):
crabswarm hook exec 'golangci-lint run {{quote .File}}' \
  '{{if not .Success}}{{blockDecision (printf "command failed: %s\nexit: %s\noutput:\n%s" .Command .Error .Output)}}{{end}}'

# success-path context injection:
crabswarm hook exec --ft go 'go vet ./...' \
  '{{if .Success}}{{context "go vet passed"}}{{else}}{{blockDecision .Output}}{{end}}'
```

- `cobra.MaximumNArgs(2)`; `args[1]` → `Option.OutputTemplate`.
- Help text embeds `exec.OutputFuncHelp()` alongside the existing
  `exec.TemplateFuncHelp()`.

### Go API surface (draft — to discuss)

`crabswarm/hook/exec`, new file `output.go` (+ `output_test.go`):

```go
// Option gains one field.
type Option struct {
	Template       string
	OutputTemplate string // optional; empty = built-in block-on-failure
	Filter         []string
}

// OutputData is the output template's data context. (Q4: all three
// capture fields; combined interleaving via io.MultiWriter is best-effort.)
type OutputData struct {
	Data            // everything the command template saw (.File, .Event, …)
	Command  string // rendered command line that ran
	ExitCode int    // 0 on success
	Success  bool   // ExitCode == 0
	Output   string // combined stdout+stderr, trailing newline trimmed
	Stdout   string
	Stderr   string
}

// OutputFuncHelp mirrors TemplateFuncHelp for the output-side functions.
func OutputFuncHelp() string

// internal:
// outputFuncMap(b *outputBuilder, event types.HookEvent) template.FuncMap
// renderOutput(src string, data OutputData, event types.HookEvent) (*types.SyncHookJSONOutput, error)
```

`Run` changes: when `OutputTemplate` is non-empty, capture the result, render
the output template, and return `&handler.HandlerError{Output: built}`
(nil-field builder → plain allow). The existing block-on-failure path becomes
the `OutputTemplate == ""` branch.

Render/run semantics (decided):

- The output template runs only after a command actually executed;
  filter-gate pass-through and an empty rendered command skip it entirely
  and stay plain allow, as today. *(Q7.)*
- Non-whitespace rendered text from the output template is a hard error —
  the builder funcs are the only sanctioned output path, so stray text or a
  typo'd func reference fails the invocation instead of silently degrading.
  *(Q2.)*
- `--dry-run` renders the command line as today AND renders the output
  template against a synthetic success result (`ExitCode: 0`,
  `Success: true`, empty `Output`/`Stdout`/`Stderr`, `Command` = the
  rendered command), printing the resulting JSON so users can preview the
  output shape. *(Q8 — user chose render-both over parse-check-only.)*

### Output template functions (draft — to discuss)

All functions record onto the builder and render as `""`. Event-scoped
functions error at call time when the input event doesn't support them
(decided — Q3).

Top-level `SyncHookJSONOutput` fields (any event):

| Func | Signature | Effect |
| --- | --- | --- |
| `blockDecision` | `blockDecision REASON` | `decision: "block"`, `reason: REASON` |
| `stop` | `stop REASON` | `continue: false`, `stopReason: REASON` |
| `systemMessage` | `systemMessage MSG` | `systemMessage: MSG` |
| `suppressOutput` | `suppressOutput` | `suppressOutput: true` |
| `terminalSequence` | `terminalSequence SEQ` | `terminalSequence: SEQ` |

Event-aware (picks the right `HookSpecificOutput*` variant from `.Event`):

| Func | Signature | Events |
| --- | --- | --- |
| `context` | `context TEXT` | every variant with `additionalContext` (PreToolUse, PostToolUse, PostToolUseFailure, PostToolBatch, UserPromptSubmit, UserPromptExpansion, SessionStart, Setup, SubagentStart, Stop, SubagentStop, Notification) |

Event-specific:

| Func | Signature | Event |
| --- | --- | --- |
| `permission` | `permission DECISION [REASON]` (`allow`/`deny`/`ask`) | PreToolUse |
| `updatedInput` | `updatedInput JSON_STRING` | PreToolUse |
| `updatedToolOutput` | `updatedToolOutput JSON_STRING` | PostToolUse |
| `permissionAllow` | `permissionAllow` | PermissionRequest |
| `permissionDeny` | `permissionDeny MSG [INTERRUPT]` | PermissionRequest |
| `retry` | `retry` | PermissionDenied |
| `sessionTitle` | `sessionTitle TITLE` | SessionStart, UserPromptSubmit |
| `initialUserMessage` | `initialUserMessage MSG` | SessionStart |
| `watchPaths` | `watchPaths PATH...` | SessionStart, CwdChanged, FileChanged |
| `reloadSkills` | `reloadSkills` | SessionStart |
| `suppressOriginalPrompt` | `suppressOriginalPrompt` | UserPromptSubmit |
| `displayContent` | `displayContent TEXT` | MessageDisplay |
| `elicitation` | `elicitation ACTION [JSON_STRING]` | Elicitation, ElicitationResult |
| `worktreePath` | `worktreePath PATH` | WorktreeCreate |

Coverage is full (decided — Q5). ⚠ rough: `updatedInput`/`updatedToolOutput`
JSON-string ergonomics unvalidated.

Docs follow the `templateutil.FuncDoc` pattern: an `OutputFuncDocs()` slice
guarded by a docs↔funcmap sync test.

### Exit-code / output protocol

`handler.HandlerError.Handle` before this plan: plain `decision=block` → exit 2
+ reason on stderr; anything else → JSON on stdout, exit 0. A template that set
both `block` and a hookSpecificOutput field would get the JSON dropped.

Decided rule *(Q6, as revised)*: **always** marshal a non-nil output to stdout
and exit 0 — block included, since block-via-JSON is valid protocol. The exit-2
+ reason-on-stderr path is removed, and so is the `isPlainBlock` heuristic that
picked between the two forms: keying the protocol off "only decision+reason are
set" couples it to `types.SyncHookJSONOutput`'s field list, so any field
upstream adds would silently reclassify outputs. A nil output stays a plain
allow: nothing written, exit 0.

## Implementation steps

1. **`crabswarm/hook/exec/output.go`** — `OutputData`, `outputBuilder`,
   `outputFuncMap`, `renderOutput`, `OutputFuncDocs`/`OutputFuncHelp`.
   Verify: new `output_test.go` unit tests (per-func effect, event
   mismatch errors, blank-render enforcement, docs sync test).
2. **`crabswarm/hook/exec/exec.go`** — add `Option.OutputTemplate`; split
   capture into stdout/stderr/combined (`io.MultiWriter`); route `Run`'s
   post-exec path through `renderOutput` when set; keep legacy path
   byte-identical when unset. Extend `Render` (dry-run) to also render the
   output template against a synthetic success `OutputData` and print the
   JSON.
   Verify: existing `exec_test.go` untouched-green + new `TestRun_OutputTemplate_*`
   and `TestRender_OutputTemplate_SyntheticSuccess`.
3. **`pkg/claudehook/handler/handler.go`** — drop the exit-2 + stderr branch
   and the `isPlainBlock` heuristic; `Handle` always writes the crafted JSON
   to stdout and exits 0 (Q6 as revised).
   Verify: `handler_test.go` pins `Block`'s marshaled bytes.
4. **`cmd/crabswarm/commands/hook_exec.go`** — `MaximumNArgs(2)`, pass
   `args[1]`, extend Long help with `OutputFuncHelp()` + examples.
   Verify: `go run ./cmd/crabswarm hook exec --help`.
5. **e2e** — `e2e/crabswarm/main_test.go`: exec with output template on
   success (JSON on stdout) and failure (block), and event-mismatch error.
   Verify: `go test ./e2e/...`.

## Testing / verification

- `go test ./crabswarm/hook/exec/... ./pkg/claudehook/handler/...`
- `go test ./e2e/...`
- Manual: `echo '<PostToolUse envelope>' | go run ./cmd/crabswarm hook exec 'false' '{{if not .Success}}{{blockDecision "no"}}{{end}}'` → exit 0, `{"decision":"block","reason":"no"}` on stdout.

## Risks

- `text/template` funcs with side effects are order-sensitive within one
  render; conflicting calls (e.g. `permission "allow"` then `block`) need a
  defined last-wins-or-error rule (folded into open question 1's mechanism).
- Claude Code may ignore hookSpecificOutput whose `hookEventName` mismatches
  the actual event — call-time validation (Q3) guards this, but the event
  list must track upstream types as they resync.
- Splitting stdout/stderr capture changes buffering vs today's
  `CombinedOutput`; combined interleaving via `io.MultiWriter` is
  best-effort.

## Open questions

None — all resolved 2026-08-05: Q1 side-effect builder; Q2 stray text is a
hard error; Q3 call-time render error on event mismatch; Q4 expose
stdout/stderr/combined; Q5 full coverage; Q6 always emit the crafted JSON on
stdout with exit 0 (revised from "JSON wins when mixed"; `isPlainBlock`
removed); Q7 output template runs only after actual execution; Q8 dry-run
renders both, output against a synthetic success result. See DECISION.md.
