# STATUS

**Current state:** implemented + reviewed — steps 1-5 landed; re-review
2026-09-01 returned approve-with-nits and all four nits are fixed.

## Checklist (mirrors PLAN.md steps)

- [x] 1. `crabswarm/hook/exec/output.go` — builder, funcmap, docs, unit tests
- [x] 2. `crabswarm/hook/exec/exec.go` — `Option.OutputTemplate`, capture split, Run routing, dry-run synthetic render
- [x] 3. `pkg/claudehook/handler/handler.go` — always emit the crafted JSON on stdout, exit 0 (Q6 as revised; exit-2 path and `isPlainBlock` removed)
- [x] 4. `cmd/crabswarm/commands/hook_exec.go` — second positional + help
- [x] 5. e2e coverage in `e2e/crabswarm/main_test.go`

## Notes discovered during implementation

- `block` is a reserved text/template keyword and cannot be a func name;
  registered as `blockDecision` instead. PLAN.md tables/examples updated.
- Pre-existing bug: `types.SyncHookJSONOutput` silently dropped
  `TerminalSequence` — `MarshalJSON` declared it but never assigned it, and
  `UnmarshalJSON` lacked the field entirely. Both fixed (types.go:8800,
  8813, 8826); `TestSyncHookJSONOutputJSONRoundTrip` guards it.
- The proto schema had no `terminal_sequence` field, so the value was still
  lost across `HookJSONOutputToProto`/`FromProto`, and the `HookInput` oneof
  had no `PermissionDenied` variant at all (`HookInputToProto` returned
  "unsupported HookInput"). Both roundtrip tests were JSON-only for that
  reason. Closed as a follow-up: `SyncHookJSONOutput.terminal_sequence`
  (field 8) and `PermissionDeniedHookInput` (`HookInput` oneof field 21) are
  now in the schema and threaded through both conversion directions;
  `TestJSONProtoRoundTrip` pins them ("HookJSONOutput sync" carries
  `terminalSequence`, plus a new "HookInput permissionDenied" case, which
  replaces `TestPermissionDeniedHookInputJSONRoundTrip`).
  `TestSyncHookJSONOutputJSONRoundTrip` stays: it drives the JSON marshalers
  with no proto conversion in between.
- Splitting the capture needs a mutex-guarded combined buffer
  (`syncBuffer`, exec.go:299): `cmd.Stdout` and `cmd.Stderr` are distinct
  writers, so os/exec runs two copy goroutines into it.
- Dry-run prints `null` when the output template records nothing — the JSON
  spelling of "no hook output", i.e. plain allow.
- Repeated builder calls are last-wins (documented in output.go; tested).
- `permission` rejects `defer` (plan lists allow/deny/ask only).
- e2e cases live in `main_test.go` as the plan specifies, next to `TestMain`;
  the sibling `preview_test.go` suggests a `hook_exec_test.go` split would
  read better, but placement is cosmetic within one package.

## Review fixes applied (round 1)

- The help's "built-in behavior spelled out explicitly" claim was false:
  the built-in prints an `exit:` line the template data could not supply, and
  it omits the `output:` section when nothing was captured. `OutputData` gains
  `Error` (the run error's message), the CLI/PLAN examples are relabeled a
  *close analogue*, and `TestRun_OutputTemplate_RestatesBuiltinBlockReason`
  now proves an exact restatement is expressible — byte-for-byte against
  `blockReason`, with and without captured output.
- `Render` mirrors `Run`'s argv gate: a rendered line that splits into no
  arguments prints nothing and skips the preview, instead of previewing a
  command that would never run. A dry run now also reports an unbalanced
  quote that only `Run` used to catch.
- Optional trailing args are cleared on repeat: `permission` without a REASON
  and `elicitation` without CONTENT no longer inherit the previous call's
  value, which contradicted the documented last-wins contract.
- `permissionAllow` reached parity with `handler.PermissionAllow`:
  `permissionAllow [UPDATED_INPUT_JSON [UPDATED_PERMISSIONS_JSON]]`, following
  `updatedInput`'s JSON-string pattern. This is the widest usage string, so
  the help's usage column reflows.
- The `//nolint:lll` no longer covers the whole `hookExecCmd` function: the
  Long help is a package-level `hookExecLong` var carrying the directive
  itself. It is a `var`, not a `const` — the text concatenates
  `exec.TemplateFuncHelp()` / `exec.OutputFuncHelp()` calls.
- Corrected the doc comments claiming context cancellation is a plain-error
  infra case: killing a *started* child surfaces as an `*exec.ExitError`, so
  it reaches the template. Pinned by
  `TestRun_OutputTemplate_CancelledContextReachesTemplate`: `Success` false,
  `ExitCode` -1, `Error` "signal: killed". Only a context already cancelled
  before the child starts is a plain error.
- `syncBuffer`'s race coverage was too small to catch a removed mutex;
  `TestRun_OutputTemplate_ConcurrentStreamsStayIntact` pushes 2000 lines down
  each stream and counts every one back out.
- `handler.Error()` documents itself as a display-string approximation:
  `Handle`/`isPlainBlock` own the protocol decision. No behavior change.
- The e2e no-output-template case asserts the whole stderr
  (`"command failed: false\nexit: exit status 1\n"`) so a stray `output:`
  section fails it.

## Review fixes applied (round 2)

- Superseding user decision (DECISION.md "Q6 revision"): the exit-2 +
  reason-on-stderr path is gone. `HandlerError.Handle` now always marshals a
  non-nil `Output` to stdout and exits 0; a nil `Output` stays exit 0 with
  nothing written. `isPlainBlock` and its tests are deleted — the heuristic
  enumerated `SyncHookJSONOutput`'s fields, so an upstream field addition would
  have silently reclassified outputs between the two protocol forms.
- `handler_test.go` swaps the `isPlainBlock` table for `TestBlock_JSON`, which
  pins `Block`'s marshaled bytes whole
  (`{"decision":"block","reason":"lint failed"}`).
- The two e2e block cases now assert exit 0 plus one JSON line on stdout. The
  no-output-template case keeps its whole-value rigor by comparing the raw
  line, which also pins that `blockReason`'s trailing newline now lives inside
  the JSON `reason` string instead of on stderr.
- `crabswarm/hook/exec` needed no behavior change: `Run` and `blockReason` are
  protocol-agnostic. Only `output.go`'s "owns the JSON assembly and the
  exit-code protocol" comment was trimmed.

## Re-review 2026-09-01 (approve-with-nits) — fixes applied

- `OutputFuncDoc.Usage`'s example said `block REASON`, a function that cannot
  exist; it names `blockDecision` now, matching the note above.
- PLAN.md's early "omitting the output template preserves today's behavior
  exactly" contradicted the protocol section that supersedes it. The decision
  is still unchanged; only the wire form is, and the bullet now says so.
- `permission`, `permissionDeny` and `elicitation` read index 0 of their
  variadic and ignored the rest, while `permissionAllow` errored — a
  silent-degradation asymmetry against the fail-loud design. All four now
  share `errSurplusArgs`; `TestRenderOutput_ArgumentValidation` covers each.
- e2e had no case for the nil-output plain-allow path, the idiom the shipped
  chat hooks ride on. `TestHookExec_OutputTemplateRecordingNothingIsPlainAllow`
  pins empty stdout + exit 0 for a template that records nothing, with a
  failing command so it also proves the built-in block is out of the picture.

## Blocked / waiting

- Nothing.

## Next action

None — the feature is landed and reviewed.
