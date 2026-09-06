# Issues tab — presentation mock

A runnable mock of the **Issues** tab planned in
`doc/plan/2026-09-04-plans_in_beads/PLAN.md`. It renders the real crabswarm
backlog plus the plan itself, as the plan would be stored in beads (D1), so the
reading experience — list, board, graph and detail (D14, D15) — can be judged
before steps 4 and 6–8 are implemented.

Read `MOCK_LIMITS.md` first: no daemon, no `bd`, no `IssuesService`, no
`WatchIssues` — and the list of plan requirements this mock cannot validate.

## Run

```sh
# 1. regenerate the fixtures (from the module root, main/)
go run doc/plan/2026-09-04-plans_in_beads/mock/gen.go

# 2. serve the mock (from web/)
cd web && pnpm exec vite --config mock/plans_in_beads/vite.config.ts
```

Then open the printed URL. Useful entry points:

- `/` — source picker
- `/issues/{sourceId}` — list view for a source, query `is:open`
- `/issues/{sourceId}?view=board` — board; `?view=graph` — dependency graph
- `/issues/{sourceId}?q=is:plan` — the Plans saved filter
- `/issues/{sourceId}?q=is:open label:chat -label:tui type:task priority:<2 admin`
  — the search bar's query, GitHub style, parsed by liqe; `q` is carried
  between views and onto the detail page, and the sidebar widgets edit it
- `/issues/{sourceId}/{issueId}` — issue detail
- `/roots` — placeholder for the unchanged file browser

Type-check and build (from `web/`):

```sh
pnpm exec tsc --noEmit -p mock/plans_in_beads/tsconfig.json
pnpm exec vite build --config mock/plans_in_beads/vite.config.ts   # output: mock/plans_in_beads/dist (git-ignored)
```

## What is here

The tree is the page-based layout PLAN.md fixes for `web/src` ("Repository
layout"), rooted here: pages own their own components, `api/` stands in for the
wire, `signals/` is client state, `lib/` is helpers.

| File | Role |
|---|---|
| `index.html`, `main.tsx` | mount point; installs the providers, the stylesheet and the theme effect |
| `index.css` | imports `../../src/index.css` and declares this directory as a tailwind source |
| `app.tsx` | routes (`/`, `/roots…`, `/issues/{sourceId}[/{issueId}]`) inside the shell |
| `components/Layout.tsx` | the shell: tab header above the routed page |
| `components/Header.tsx` | Roots \| Issues tabs, "simulate change", theme toggle |
| `pages/issues/index.tsx` | the issues screen — sources and filters on the left, the view strip and the active view or the open issue on the right — plus the `/` source picker |
| `pages/issues/SourceSwitcher.tsx` | registered issue sources (D13) |
| `pages/issues/QueryBar.tsx` | the search bar: one query text, suggestions for the token under the caret (Ark combobox), Enter applies |
| `pages/issues/IssueFilters.tsx` | quick filters — status toggle group (Ark), the Plans chip, label combobox (Ark) — that add and remove tokens in the bar's query |
| `pages/issues/ViewTabs.tsx` | List \| Board \| Graph strip (Ark tabs), `?view=` |
| `pages/issues/IssueList.tsx` | list view: table rows with labels, metadata chips and the epic progress bar |
| `pages/issues/IssueBoard.tsx` | board view: status columns, swimlanes by parent epic with progress |
| `pages/issues/IssueGraph.tsx` | graph view and the detail page's neighbourhood: mermaid flowchart from edges, click-through, size cap |
| `pages/issues/IssueView.tsx` | detail: header with progress, rendered fields, children, dependencies, neighbourhood, comments, TOC |
| `pages/issues/MarkdownField.tsx` | one `.markdown-body` card, with the client-side mermaid pass |
| `pages/issues/useIssues.ts` | the page's state over the api layer: the query-string filters, the rows, the open issue |
| `pages/roots/index.tsx` | `/roots`: a placeholder for the unchanged file browser |
| `pages/not-found.tsx` | fallback route |
| `api/fixtures.json` | generated; do not edit by hand |
| `api/client.ts` | fixture types (the proto messages) and the stand-in service (`listSources`, `listIssues`, `getIssue`, `listDependencies`) |
| `api/query.ts` | the query language: liqe parses, this gives `is:` `status:` `label:` `type:` `parent:` `priority:` and free text their meaning, plus token edits and suggestions |
| `api/issues.ts` | the URL's `q` / `view` spelling and the decodes the views need, over that service |
| `api/events.ts` | the simulated `WatchIssues` push |
| `signals/issues.ts` | client state: which issue the reader has open |
| `lib/paths.ts`, `lib/format.ts` | `/issues/…` URLs; timestamp and status spelling |
| `lib/graph.ts`, `lib/mermaid.ts` | edges → mermaid flowchart text; the shared mermaid run |
| `public/assets/css/` | generated; alert + chroma stylesheets the rendered HTML expects |

The query language, in the bar's own hint line: `is:open` (not closed, the
default), `is:closed`, `is:plan`; `status:<bd status>`; `label:x` and
`-label:x`; `type:epic`; `parent:<id>`; `priority:1`, `priority:<2`; bare
words match title and id; `AND`, `OR`, parentheses and quotes as in
Lucene. An unknown qualifier matches nothing and is named under the bar; a
query that does not parse shows the parser's message and lists nothing.

Nothing under `web/src` is modified: the mock imports `src/index.css`,
`src/signals/ui.ts` and `src/components/ThemeToggle.tsx`, and copies the
patterns it needs from the app's `Layout.tsx`, `RootSwitcher.tsx`,
`FileTree.tsx` and `DocView.tsx`.

Import aliases follow the web preference rule: `@/…` is this directory
(declared in `tsconfig.json` `paths` and `vite.config.ts` `resolve.alias`,
kept in sync), and `#src/…` is the app's source through the `imports` field
in `web/package.json` (`"#*": "./*"`), which TypeScript and Vite both read.
Sibling files import with `./`.

## Driving it headless

The nix chromium needs a fontconfig file, or it aborts in Skia's font code the
moment an Ark combobox renders (`SkFontMgr_FontConfigInterface.cpp: Not
implemented`, SIGTRAP). Point `FONTCONFIG_FILE` at a `fonts.conf` whose
`<dir>` names a font directory from the nix store before launching a browser
against the mock.
