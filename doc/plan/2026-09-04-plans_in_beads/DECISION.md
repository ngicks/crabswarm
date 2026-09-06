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

## D7 — Gate is metadata `idea_gate_passed` = date (decided 2026-09-04)

Choice: `bd update <id> --set-metadata idea_gate_passed=<YYYY-MM-DD>` on
confirmation; absent means not confirmed; `--unset-metadata idea_gate_passed`
resets it after a substantive idea edit.
Rationale: one flat key, visible in `bd show --json` and the GUI, set and
cleared with existing flags. Rejected: a nested `ngplan` object (needs
full-JSON `--metadata` on every change); a `gated` label (no date).

Amended by the user, 2026-09-06: the key was `idea_gate` and is now
`idea_gate_passed`. The semantics are unchanged; the old name did not
say at a glance that presence means the gate has passed.

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
metadata chips (so `idea_gate_passed` shows), `discovered-from` edges labelled
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

## D18 — A GitHub-style search bar, parsed by liqe, is the filter (decided 2026-09-06)

Choice: the Issues surface filters through one query text in the URL's
`q`, GitHub style — `is:open` (not closed; the default), `is:closed`,
`is:plan`, `status:<bd status>`, `label:` and `-label:`, `type:`,
`parent:`, `priority:` with comparisons, free text on title and id, AND,
OR, parentheses and quotes. `liqe` (3.8.7, BSD-3, added to
`web/package.json`) parses the text; `api/query.ts` gives the qualifiers
their meaning and produces suggestions for the token under the caret,
shown through the Ark combobox. The sidebar widgets keep no state: they
read their tags out of the query and toggle tokens in it. Unknown
qualifiers match nothing and are named; a parse error shows the message.
Validated in `web/mock/plans_in_beads`.
Rationale: the user asked for the GitHub bar; one query text is a link,
reads like `bd list --status`, and removes the drift between separate
`status` / `label` / `filter` parameters and the widgets. liqe is the one
maintained, typed parser that returns a tree (negation, OR, ranges)
rather than a flat map, and the mock's evaluator is 150 lines over it.
Rejected: GitHub's own QueryBuilder web component (Primer styling, still
needs the grammar); `search-query-parser` (flat key/value, no negation
or OR); a hand-rolled tokenizer (no OR, no quoting rules); keeping
separate URL parameters beside the bar (two sources of truth).
Left open as Q14: where the real feature evaluates the query.

## D19 — The app adopts the mock's type scale (decided 2026-09-06)

Choice: the type scale the mock settled on — body text 15px and
metadata 13px through Tailwind's `--text-sm` / `--text-xs` tokens in
rem, and daisyUI's md tier taking its font size from `text-sm` — moves
from `web/mock/plans_in_beads/index.css` into `web/src/index.css` in
the implementation phase (step 6), so the file browser and the Issues
tab share one scale.
Rationale: the app's chrome sat on Tailwind's 12 / 14px steps and
daisyUI's 12 / 14 / 18px tiers, which read small in a table of titles;
15px is where feeds settle for body text, and there is no such tier to
pick, so the scale has to be declared. Rem keeps the browser's font
setting in force.
Rejected: scaling the root font size (scales spacing and every daisyUI
control, and github-markdown-css is authored in px); one daisyUI tier
up (18px).

## D20 — Board and graph views leave the plan; GitHub's state buttons over the list (decided 2026-09-06)

Choice: the kanban board and the whole-source dependency graph views
are dropped from this plan (they may be reconsidered in a later one);
D14 keeps the generic list and detail pages and their affordances, and
D15 keeps only the detail page's neighbourhood graph, redrawn at
natural size with bold titles and the current issue in the primary
colour. Above the list, as GitHub's issues page has them, **Open N**,
**Closed N** and **Plans N** buttons: Open and Closed pick one state
(`is:open` / `is:closed`, and picking the active one again clears the
state), Plans toggles `is:plan`, and each count is what the rest of the
query would match with that state. D16's board-column and
unconnected-issue rules go with the views; its `outgoing` and
query-string clauses stay. Q15 closes.
Rationale: the user's call after seeing the mock; a board needs its own
scoping and a whole-source graph is mostly unconnected issues, so
neither earned its place, while the state buttons answer the everyday
question (what is open, what did we close, which are plans) without a
query.
Rejected: keeping a placeholder tab strip with one tab; a "saved filter"
chip in the sidebar instead of GitHub's row (the row is where the eye
looks first and carries counts).

## D17 — Mock passes stay inline for this plan (decided 2026-09-06, agent default) [automatic]

Choice: the mock's D14/D15 pass was written inline rather than delegated
to a subagent as the ngplan visuals reference asks; the result was
reviewed, type-checked, built and driven headless, so it is not redone.
The next mock pass, if any, is delegated.
Rationale: the guidance exists to keep bulk output out of the planning
context, and that cost was already paid when the user pointed it out.
Rejected: regenerating the same views through a subagent for the sake of
the rule.

Amended by the user, 2026-09-06: the passes behind D18–D20 were also
written inline. From here on every change to the mock is delegated to a
subagent one model class down, as the ngplan visuals reference asks; the
planning agent reviews the result, keeps `MOCK_LIMITS.md` current and
records the decisions. This is not an agent default any more.

## D21 — The neighbourhood zooms and pans (decided 2026-09-06, from the mock)

Choice: the detail page's neighbourhood graph sits in a viewport of
fixed, user-resizable height. The wheel zooms about the cursor between
a quarter size (or the fit scale, whichever is smaller) and four times;
a drag pans; a toolbar offers −, +, 1:1 and Fit with the current
percentage; double-click on empty space toggles Fit and 1:1. A
neighbourhood whose fit scale is at least 0.6 opens fitted; a wider one
opens at 1:1 centred on the current issue, so a wide epic is readable at
once and the overview is one Fit away. A node click still opens the
issue; a press that moves under 4px counts as a click, more counts as a
drag and never navigates. The transform is written by hand in
`IssueGraph.tsx`, about forty lines, and adds no dependency.
Rationale: the user asked for zoom on the neighbourhood, and the mock
showed that the plan epic fits its box only at a quarter size, where
no label reads, so a fitted opening view defeats the graph's purpose.
Rejected: `d3-zoom`, which mermaid bundles, because importing it means
declaring `d3-zoom` and `d3-selection` as dependencies of the SPA and
the mock, more contract than the arithmetic is worth; `svg-pan-zoom`,
a further dependency; a scrolling box without zoom, the previous
state, which the user judged poor; opening every neighbourhood fitted,
unreadable for a wide one.

## D22 — The GUI stays read-only; a writable board is a later plan (decided 2026-09-06)

Choice: this plan ships no write path. Composing comments on the issue
page, editing issue content on the strength of a comment, and a log of
those edits that the page can show all belong to a separate plan for a
writable issue board.
Rationale: the user's call when the detail view was restyled; the
reading experience is what this plan proves, and the first write path
through the daemon deserves its own idea phase (who writes, as whom,
how an edit is logged, what bd offers for history).
Rejected for now: a comment box writing through the daemon to `bd`;
an edit history taken from bd's version history; logging edits as
`Edit:` comments by convention. All three are candidates for that plan.

## D23 — Detail page: title first, one card per section with a header strip (decided 2026-09-06, from the mock)

Choice: the detail page opens with the title, large, and the metadata
under it: status, id, type, priority and labels on one row; parent,
created, updated and counts on the next; then the metadata chips and
the epic progress bar. Every section after it is a card with a header
strip in the stronger base colour carrying the section name at heading
size: close reason, each rendered field, children, dependencies, the
neighbourhood (its legend and zoom toolbar ride in the strip) and the
comments, where each comment has its own smaller strip with author,
kind and time. The rendered fields stay separate cards, because each
is its own markdown document rendered by the server.
Rationale: the user wanted to skim section borders at a glance, and
the title where GitHub puts it; the small uppercase labels outside the
cards did not mark sections strongly enough.
Rejected: merging every section into one bordered box with markdown
style headings (the user withdrew it once the per-field rendering was
clear); keeping the labels outside the cards at a larger size.

Amended 2026-09-06: the TOC outlines every section in page order, with
each rendered field's own headings under it, instead of the description
and design headings alone, which left acceptance criteria and everything
below unreachable. A bar sticks to the top of the detail column with
the back link, the id, the title and a Jump to menu, an Ark UI
navigation menu, whose items scroll to the sections.

## D24 — A labels page, GitHub's way, instead of a sidebar picker (decided 2026-09-06)

Choice: the sidebar holds the source switcher and, under it, a
**Labels N** button that opens `/issues/{sourceId}/labels`: a
title, a name filter kept in `q`, **Active N** and **Archived N**
buttons kept in `state`, and a table of every label of the source with
its open and closed counts, each a link into the list with the matching
`label:` query, and its last update. The route is matched before
`{issueId}`; no bd id is `labels`. The counts are aggregated from the
listing in the client, as the state buttons are (Q14 decides whether
that moves).
Rationale: the user asked for GitHub's shape; the picker in the
sidebar was carried over from the earlier filter sidebar, not decided,
and a page with counts answers "which labels exist and how used" where
a combobox only completes a name.
Open: bd has no label entity and no archive flag, so "archived" is
defined here as carried only by closed issues (Q16).
Rejected: keeping the picker beside the page; a labels tab in the
header (labels belong to one source, the header tabs do not); the
button beside the search bar, GitHub's spot, which the user moved into
the sidebar so the drawer stays the place for a source's navigation.

## D25 — Go packages follow the design preference rule as updated 2026-09-06 (decided 2026-09-06)

Choice: the plan's Go layout adopts the repository's design preference
rule in the form it took on 2026-09-06. Files under `./cmd` parse flags
and hand off; printing, tables and exit decisions for the issues
commands live in a new `crabswarm/issues/cli` package, as
`crabswarm/chat/cli` already does for chat; `Run` functions return
errors; cancellable work takes a context first; the `bd` runner, the
working directory and the source registry are injected through
constructors; the service declares the client interface it consumes;
tests sit beside their package.
Rationale: the user asked for the layout convention change to be folded
into the plan before implementation starts, so that steps 1, 3, 4 and 5
are written against the current rule and not reviewed against it
afterwards.
Rejected: printing from the command files (the earlier plan text let
the lint command render its own lines); a package-level `bd` path read
from the environment inside the client.

## D26 — Implementation runs in the `main` worktree, one commit per step (decided 2026-09-06) [automatic]

Choice: the steps are implemented on the `main` branch in the existing
`main/` worktree, not in a fresh worktree, with one commit per finished
step in the repository's commit convention. The user was away when the
run started.
Rationale: every mock and plan commit of this plan already went to
`main` directly; the `web/` dependencies are installed only in `main/`,
and steps 4 and 6 need them for code generation and the build; the
user's dev server watches `main/`. A fresh worktree would add a
`pnpm install` and a branch merge for no isolation the user asked for.
Rejected: a `plans-in-beads` branch in a sibling worktree, the plan's
earlier wording; squashing the steps into one commit.

## D27 — The bd client's surface follows bd's JSON, not the sketch (decided 2026-09-06) [automatic]

Choice: the client exposes what the fence sketched plus what bd
actually returns: constructor options `WithBinary`, `WithEnv`,
`WithLogger`; a `Status` string type with the five bd statuses
including `deferred`; `Summary` carrying every `bd list` field, the
text fields included, so a lint sweep can find mermaid fences without a
`bd show` per issue; `Dependency` embedding the other issue's `Summary`
with the edge kind, as `bd show` reports it. `List` always passes
`--limit`, spelling 0 as unlimited, because bd's own default of 50
would silently truncate a source. Identifiers keep the repository's
`ID` spelling. `Where` takes only a context and a directory.
Rationale: the fence was a sketch written before the JSON was decoded;
mirroring bd's record avoids a second decode later, and the option
functions are how the runner is injected without environment reads.
Rejected: a lean `Summary` with the text fields on `Issue` only (one
`bd show` per issue for the lint sweep, ~1.5 s each); the `Id`
spelling a review skill prefers, against eleven existing `ID` uses.

## D28 — mermaidlint's surface and parity with the file hook (decided 2026-09-06) [automatic]

Choice: `Lint` takes options (`WithBinary`, `WithDir`), exports
`HasFence` for pre-filtering a listing, `ErrBinaryNotFound` as the
`errors.Is` target for a missing tool, and `Field*` constants for the
field names. Parity with the file hook: a rule warning that
`mermaid-lint` reports with severity "error" (it exits 1 on those)
becomes a Finding with the rule name as its type, not only a parse
failure; and the run happens in the source's directory when the caller
passes it, so a repository's `mermaid-lint` configuration judges issue
text as it judges files. Close reasons are not linted.
Rationale: the hook's promise is "what the file hook refuses, the issue
hook refuses"; a temp-dir run with defaults and parse errors only would
let a diagram the file hook rejects pass through beads.
Rejected: a `Finding` without a rule id (nowhere to say why a warning
blocked); linting close reasons (closed issues are swept only with
`--all`, and a close reason is prose).

## D29 — The service's constructor takes the store; the poller and renderer are named interfaces (decided 2026-09-06) [automatic]

Choice: `NewService(logger, renderer, store, opts...)` takes the
`SourceStore` it serves; `SourceStore.Add` returns whether the source
was new, which is what starts a poller and publishes `SourcesChanged`;
`NewPoller(logger, sourceID, lister, interval, emit)` consumes an
`IssueLister` interface and `Service` a `Renderer` interface, both
declared beside their use; `Service.Run(ctx)` supervises the pollers on
the previewer's errgroup; the HTTP layer mounts the issues handler only
when the previewer hands one over. Two follow-ups are decided for the
cleanup pass before the final review: `Summary` gains `Metadata` and
`Labels` as `bd list` returns them, replacing the interim
`Client.ListFull`; and the proto enum gains `ISSUE_STATUS_DEFERRED`,
since bd has a fifth status that today maps to unspecified and cannot
be filtered for.
Rationale: the plan's three-argument sketch hid where the store came
from; passing it keeps the service free of package state. The interim
`ListFull` exists only because the store worker could not edit the
types file while the lint worker owned it.
Rejected: the service creating its own store; a poller per
`WatchIssues` subscriber (the poll must run with no viewer so the diff
baseline exists when one connects).

## D30 — Source registration never blocks previewing files (decided 2026-09-06) [automatic]

Choice: in its default mode `crabswarm preview DIR` never fails on the
issues side: a directory with no beads database registers no source
silently (the common markdown-only case), and any other failure — no
`bd` on PATH, a daemon-side error — is one warning line on stderr; the
root is registered and the URL printed either way. Only `--issue`
makes a failure an error.
`preview list` gains a KIND column and shows a source's bd prefix as
its name and its `.beads` directory as its path; `preview remove`
resolves ids and names across both kinds and refuses an ambiguous
name. The unreferenced root-only renderer in `crabswarm/cli` is
removed.
Rationale: the contract said NotFound is tolerated, but a host without
`bd` would have failed a plain markdown preview after printing its
URL; previewing files must not depend on beads being installed.
Rejected: mapping every daemon failure to NotFound (hides real errors
from `--issue`); keeping the old renderer beside the new one.

## D31 — The mock graduates into `web/src` and is removed; header tabs are route links (decided 2026-09-06) [automatic]

Choice: the presentation mock's pages, query language, signals and
helpers move into `web/src` as the real Issues tab over the daemon,
and `web/mock/plans_in_beads/` is deleted in the same step; its
`MOCK_LIMITS.md` moves into this plan directory so the findings
survive, and the fixture generator under `mock/` stays as history.
The Roots | Issues header draws plain anchors with the tab role rather
than Ark UI Tabs: each tab is a route, not a panel switch, so there is
no keyboard-driven panel state for Ark to own. The client-side liqe
evaluation over one full listing per source stays the default (the
open question on where the query is evaluated remains open, with that
as its tentative answer).
Rationale: the reorganisation broke the mock's imports (the theme
signal moved and the `@` alias now means `src/`), and a mock that
imports app files cannot outlive the app's layout; once the app holds
the same views the mock has nothing left to show.
Rejected: keeping the mock alive with its own alias map (churn in a
throwaway); Ark Tabs for route links (behaviour Ark would supply is
not needed).

## D32 — The detail page reads both edge directions from one whole-source dependency listing (decided 2026-09-06) [automatic]

Choice: the SPA calls `ListDependencies(source_id, [])` once per
source — one `bd dep list` over every issue id — caches the edges
under the source's query subtree, invalidates them with the listing
on `IssuesChanged`, and derives from them both the detail page's
dependencies table (outgoing and incoming rows, parent-child still
carried by children and parent) and the neighbourhood's node set.
`GetIssue`'s own `dependencies` field stays on the wire but the page
no longer depends on it. Only the detail page triggers the fetch.
Rationale: `bd show` reports only the edges an issue is the from side
of, so a blocker or an origin issue showed no dependencies and no
neighbourhood; the mock drew both directions from a whole-source edge
list, and one cached `bd dep list` per source is the cost of that.
Rejected: a per-issue incoming lookup RPC (bd offers no reverse
query short of listing everything); dropping incoming rows.
