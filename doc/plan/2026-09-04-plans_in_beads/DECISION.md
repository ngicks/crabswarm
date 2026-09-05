# Decisions

## D1 — Steps are child beads (decided 2026-09-04)

Choice: one plan bead (`description` = idea, `design` = plan body,
`acceptance` = success criteria, `notes` = short narrative status,
comments = decisions and discussion, metadata = gate); implementation
steps are child `task` beads ordered by `blocks` dependencies; sub-plans
and handoff items are children too.
Rationale: status becomes derivable from child state, `bd ready` drives
execution, and a handoff item is born linked to the step that found it.
Rejected: steps as a numbered prose section in `design` — self-contained
but status would be a hand-maintained checklist again.

## D2 — Existing plan directories stay as history (decided 2026-09-04)

Choice: no migration in this plan; the 14 `doc/plan/` directories remain
readable in git. The user will order a migration once plans in beads are
stable enough.
Rejected: migrate now (premature before the convention is proven);
delete (loses nothing in git but gains nothing either).

## D3 — End-of-turn mermaid check over updated beads (decided 2026-09-04)

Choice: implement a `crabswarm` subcommand that sweeps beads — all open
ones, or the N most recently modified by `updated_at` — and validates
every mermaid fence in their description, design, notes, acceptance and
comments; install it as a Stop hook so the agent's turn is blocked with
the bead ID, field and parser message until fixed.
Rationale: one mechanism covers every write path (`bd create`, `update`,
`comment`, inline `-d`, stdin, `bd edit` in an editor) without parsing
`bd` command lines; the check runs once per turn instead of once per
command. The user rejected "beads changed during this turn" as the
selection: nothing reliable tracks that, and a stateless sweep by status
or modified date is cheap at this backlog's size.
Rejected: a PreToolUse hook on `bd` invocations (misses inline and editor
writes, must parse argv); a git hook managed by hk (never sees bead
bodies, which do not pass through git); both combined (two definitions to
keep in sync). Whether the same Stop hook also lints modified `.md`
files is open as Q10.

## D4 — GUI is first-class and generic over beads (decided 2026-09-04)

Choice: the preview GUI is in scope for this plan, built here, and renders
beads the way `bd` models them rather than a plan-specific view. The
ngplan skill and the plan convention feed back into `ngicks/agents-package`
after they have been tried against that GUI.
Rejected: convention-first with the GUI later or in a separate plan (the
user wants the reading experience tried alongside the convention);
plan-specific GUI (would not serve the backlog).

## D5 — Data path is the `bd` CLI (decided 2026-09-04)

Choice: `crabswarm/beads` shells out to `bd list --json` / `bd show --json
--include-comments` / `bd info -q` from the root's directory.
Rationale: no new Go dependency, bd's JSON carries every field, ~1.5 s per
call is acceptable for a GUI and a per-turn sweep, and concurrent calls
work. Rejected: a Dolt Go driver (huge, duplicates beads' storage layer,
second-opener risk in embedded mode); `bd export` per request (whole
database per read); `bd sql` (unsupported in embedded mode).

## D6 — Issues are a second surface beside roots (decided 2026-09-04)

Choice: the SPA gets a top-of-page tab header switching Roots | Issues;
file-browser URLs move from `/r/{rootId}/…` to `/roots/{rootId}/…` (a
breaking change the user accepted) and issues live at `/issues/…`; the API
is a new `IssuesService` in proto package `ngicks.crabswarm.issues.v1`.
Rationale: the user wants the two surfaces visibly separate, with a
matching URL scheme, rather than beads squeezed into the file browser.
Rejected: RPCs on `PreviewService`; a synthetic root over
`GetTree`/`GetDocument`.

## D7 — Gate is metadata `idea_gate` = date (decided 2026-09-04)

Choice: `bd update <id> --set-metadata idea_gate=<YYYY-MM-DD>` on
confirmation; absent means not confirmed; `--unset-metadata idea_gate`
resets it after a substantive idea edit.
Rationale: one flat key, visible in `bd show --json` and the GUI, set and
cleared with existing flags. Rejected: a nested `ngplan` object (needs
full-JSON `--metadata` on every change); a `gated` label (no date).

## D8 — Daemon-side poll feeding `WatchIssues`, for now (decided 2026-09-04)

Choice: the daemon polls each registered source every 10 s with one
`bd list` call, diffs IDs and `updated_at`, and pushes `IssuesChanged`
on `IssuesService.WatchIssues`; the SPA invalidates queries as it does
for file changes. Memo from the user: once `bd` ships `bd events
--follow` (expected around 1.2.3; 1.2.2 has no `events` command), replace
the poller with a follower — filed as a backlog issue in step 7.
Rationale: the same live-reload experience as files with no client-side
timers; the cost (one `bd` call per source per 10 s) is fine on a dev
machine. Rejected: client-side poll while a view is open (per-tab timers,
no shared refresh); manual refresh only.

## D9 — Reuse the `mermaid-lint` CLI (decided 2026-09-04)

Choice: `crabswarm beads lint` runs the same `mermaid-lint` CLI
(`mermaid-lint-cli` 0.53.1 via mise) that agents-package's
`hooks/markdown-mermaid-lint` already runs on edited markdown, writing
bead text to temp `.md` files and reading `--format json`.
Rationale: one engine for files and beads, already in the dev
environment, no browser, JSON output with line numbers. Rejected: a node
script over the bundled `mermaid` (second engine); `@mermaid-js/mermaid-cli`
(Chromium per developer).

## D10 — Sweep all open issues (decided 2026-09-04)

Choice: `crabswarm issues lint` sweeps every open issue by default (one
`bd list --json`, then `bd show` only for issues whose text has a mermaid
fence); `--limit N` by `updated_at` and `--all` are opt-in. Markdown
files in the worktree stay with the file hook.
Rationale: cheap at today's size, stateless, never skips an older broken
issue. Rejected: newest-N default (skips old breakage); sweeping modified
`.md` files too (duplicates `hooks/markdown-mermaid-lint`).

## D11 — Stop hook ships from this repository's apm project (decided 2026-09-04)

Choice: a local hook package `hooks/beads-mermaid-lint` in this repo,
listed in `apm.yml` beside a new dependency on agents-package's
`hooks/markdown-mermaid-lint`; step 3 verifies apm accepts a local path
and falls back to a documented `settings.local.json` entry if not.
Rationale: deploys with the other hooks, stays in this repo until D4's
feedback to agents-package. Rejected: hand-maintained
`settings.local.json` (easy to lose); authoring in agents-package now
(blocks on a second repository).

## D12 — "issues" everywhere (decided 2026-09-04)

Choice: the word is "issue" across the CLI (`crabswarm issues lint`), Go
packages (`crabswarm/issues`, `crabswarm/issues/mermaidlint`), the hook
package (`hooks/issues-mermaid-lint`), the proto (`IssuesService`,
`ngicks.crabswarm.issues.v1`), the GUI and URLs (`/issues/…`). "beads"
names only the store and its `bd` tool.
Rationale: one word for the user-facing concept. Rejected: "beads" for
the store-side packages and CLI (two words for one thing).

## D13 — Issue sources discovered with `bd where`; `--root` / `--issue` on preview (decided 2026-09-04)

Choice: the daemon keeps an issue-source registry beside the root
registry. `crabswarm preview [DIR]` registers DIR as a file root and, by
running `BD_JSON_ENVELOPE=1 bd where --json` in DIR, its beads database
as an issues source keyed by the reported `.beads` path (one source per
repository however many worktrees). `--root` registers only the
directory, `--issue` only the source; no flag means both. `preview list`
and `preview remove` cover both kinds.
Rationale: the user's spelling of the command; a source is a repository,
not a directory, so keying by `.beads` path deduplicates worktrees.
Rejected: keying issues by preview root (one database shown once per
worktree); daemon-wide discovery from the daemon's own cwd (cannot show
two repositories).

## D14 — Generic issue viewer with list, board, graph and detail; no plan view (decided 2026-09-05)

Choice: the Issues surface is a generic beads viewer with four views per
source — list (default), kanban board by status with parent-epic
swimlanes, dependency graph, and the detail page. Plans get no view of
their own: a **Plans** saved filter (`label=plan`) plus affordances that
apply to any issue with the data — an epic progress bar from child
status, `Decision:` / `Discussion:` badges on comments by prefix,
metadata chips (so `idea_gate` shows), `discovered-from` edges labelled
in the graph. No new dependency type; bd's four kinds are drawn.
Rationale: under D1 a plan is ordinary bead data, so everything a plan
view would show is derivable from generic data; a second view would
re-derive it and drift. The community GUIs (bd-board, Scotty, beads-ui,
BeadSpec, Lista) converge on the same four views. Read-only stays
(editing is a non-goal), so the board has no drag-and-drop.
Rejected: a dedicated plan view or plan tab (drift, duplicate code); a
crabswarm-side dependency type (belongs to beads).

## D15 — Graph drawn with the bundled mermaid, filtered set, size cap (decided 2026-09-05, default chosen by the agent)

Choice: the graph view and the detail page's neighbourhood emit a
mermaid `flowchart LR` from `ListDependencies` edges and render it
through the SPA's existing `mermaid.run` path; click-through via the
rendered SVG node ids under `securityLevel: "strict"`; the view draws
the filtered set and warns above 150 nodes.
Rationale: zero new dependencies (mermaid 11.16 is already bundled for
documents), consistent look with rendered diagrams, enough for a
backlog of this size. Rejected: elkjs / dagre / cytoscape / react-flow
(a layout or graph library is a new manifest dependency with a real
footprint; revisit only if the cap bites in practice — that becomes the
fallback). The user asked for the graph, not for the engine; overturn
this entry if an interactive graph is wanted from the start.

## D16 — View behaviours settled in the mock (decided 2026-09-06, agent default, overturnable) [automatic]

Choice, validated in `web/mock/plans_in_beads` (see its MOCK_LIMITS.md):
the board draws a column per status the filter lists, so bd's default
listing (no closed issues) shows no empty closed column; the graph view
hides issues no edge touches unless the URL says `isolated=show`, and
says how many it hid; view options travel in the query string beside the
filters (`lanes=none`, `isolated=show`), and the query string is carried
onto the detail URL and back so a view survives opening an issue;
`IssueDependency.outgoing` is true when the issue is the edge's `from`
side (dependent, child, discoverer), parent-child rows are left out of
the dependencies table, and the graph draws every arrow from the side
that comes first (blocker, parent, origin).
Rationale: a permanent four-column board sat empty on the right on every
visit; 40 of 51 open issues have no edge and mermaid stacks them in one
tall column that buries the graph; the wording in the dependencies table
and the arrows in the graph must read the same edge the same way.
Rejected: a fixed set of columns; drawing every filtered node; keeping
`outgoing` as "this bead -> other" without saying which end that is.

## D17 — Mock passes stay inline for this plan (decided 2026-09-06, agent default) [automatic]

Choice: the mock's D14/D15 pass was written inline rather than delegated
to a subagent as the ngplan visuals reference asks; the result was
reviewed, type-checked, built and driven headless, so it is not redone.
The next mock pass, if any, is delegated.
Rationale: the guidance exists to keep bulk output out of the planning
context, and that cost was already paid when the user pointed it out.
Rejected: regenerating the same views through a subagent for the sake of
the rule.
