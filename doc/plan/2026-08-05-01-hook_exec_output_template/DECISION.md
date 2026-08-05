# DECISION LOG

One entry per material decision: choice, rationale, rejected alternatives.

## Q1 — output mechanism: side-effect builder funcs (2026-08-05)

- **Choice:** each output func (`block`, `context`, `permission`, …) records
  a field on an internal `*types.SyncHookJSONOutput` builder and renders as
  `""`; after `Execute`, exec marshals the accumulated output via
  `handler.HandlerError`, which owns the exit-code protocol.
- **Rationale:** keeps JSON assembly and exit-2/exit-0 protocol handling in
  exec/handler instead of user templates; control flow stays plain
  `text/template`.
- **Rejected:** value-returning funcs whose rendered text becomes stdout
  verbatim — pushes escaping and one-object-only discipline onto users and
  special-cases the exit-2 block form.

## Q3 — event-mismatch validation: call-time render error (2026-08-05)

- **Choice:** an event-scoped func called for an unsupported event fails the
  render; exec exits with a plain (non-hook) error.
- **Rationale:** misconfigured hooks surface immediately instead of Claude
  Code silently ignoring a mismatched `hookEventName`.
- **Rejected:** emit anyway — permissive, but hides configuration bugs.

## Q5 — coverage scope: full (2026-08-05)

- **Choice:** implement a template function for every `SyncHookJSONOutput`
  field and every `HookSpecificOutput*` variant now.
- **Rationale:** matches the request ("template function for each hook
  output"); the per-func implementation cost is small once the builder
  machinery exists.
- **Rejected:** curated core with a long tail later.

## Q6 — mixed block+JSON exit semantics: JSON wins (2026-08-05)

- **Choice:** `handler.Handle` emits the exit-2 + stderr form only when
  decision/reason are the *only* fields set; any additional field switches
  to the full JSON-on-stdout, exit-0 form.
- **Rationale:** block-via-JSON is valid hook protocol; the current
  precedence would silently drop hookSpecificOutput data from mixed
  templates.
- **Rejected:** keep exit-2 precedence and document the data loss.
- **SUPERSEDED** by the always-JSON decision below.

## Q6 revision — always emit crafted JSON, drop `isPlainBlock` (2026-08-05)

- **Choice:** `handler.Handle` always marshals a non-nil
  `SyncHookJSONOutput` to stdout and exits 0; the exit-2 + reason-on-stderr
  path and the `isPlainBlock` heuristic are removed entirely.
- **Rationale (user):** the "only decision+reason set" check is fragile —
  upstream may add fields to `SyncHookJSONOutput`, and any new field would
  silently reclassify outputs between the two protocol forms. Block-via-JSON
  is valid protocol, so the exit-2 form buys nothing worth that coupling.
  There is no deployed consumer requiring the old exit codes.
- **Rejected:** the field-enumerating `isPlainBlock` heuristic (implemented
  first, then removed at user direction).

## Q2 — stray rendered text: hard error (2026-08-05)

- **Choice:** non-whitespace text rendered by the output template fails the
  invocation with a plain error; builder funcs are the only sanctioned
  output path.
- **Rationale:** a typo'd func reference or stray literal text should fail
  loudly, not silently degrade to allow-everything.
- **Rejected:** silently ignore (hides bugs); map to `systemMessage`
  (conflates typos with intent).

## Q4 — capture split: stdout, stderr, and combined (2026-08-05)

- **Choice:** `OutputData` exposes `.Stdout`, `.Stderr`, and combined
  `.Output` (via `io.MultiWriter`; interleaving in `.Output` best-effort).
- **Rationale:** templates can route linter stdout vs stderr noise
  differently; combined stays available for the default-equivalent template.
- **Rejected:** combined-only (would need a breaking change later).

## Q7 — output template runs only after actual execution (2026-08-05)

- **Choice:** filter-gate pass-through and empty rendered command skip the
  output template entirely; result stays plain allow, as today.
- **Rationale:** the output template shapes the result of a command that
  ran; gated events keep zero-cost pass-through semantics.
- **Rejected:** always-run with `.Executed=false` — more states to reason
  about with no current use case.

## Q8 — dry-run renders both templates (2026-08-05)

- **Choice:** `--dry-run` prints the rendered command line and additionally
  renders the output template against a synthetic success result
  (`ExitCode: 0`, `Success: true`, empty captures, `Command` = rendered
  command), printing the resulting JSON.
- **Rationale:** user preference — previewing the output JSON shape is worth
  the synthetic-data caveat; also parse-checks the output template before
  it's wired into a live hook.
- **Rejected:** command-only dry-run with parse-check (the recommended
  default; user chose the richer preview instead).
