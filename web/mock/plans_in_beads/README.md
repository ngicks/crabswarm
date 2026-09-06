# Issues tab — presentation mock

A runnable mock of the **Issues** tab planned in
`doc/plan/2026-09-04-plans_in_beads/PLAN.md`. It renders the real crabswarm
backlog plus the plan itself, as the plan would be stored in beads (D1), so the
reading experience — a GitHub-style list and the detail page with its
neighbourhood (D14, D15, D18, D20, D21) — can be judged before steps 4,
6 and 7 are implemented.

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
- `/issues/{sourceId}` — the list, query `is:open`; Open / Closed / Plans
  buttons with counts above the rows, as GitHub draws them
- `/issues/{sourceId}?q=is:closed` — what the Closed button writes;
  `?q=is:open is:plan` what Plans adds
- `/issues/{sourceId}?q=is:open label:chat -label:tui type:task priority:<2 admin`
  — the search bar's query, GitHub style, parsed by liqe; `q` is carried
  onto the detail page and back, and the buttons and the label picker edit it
- `/issues/{sourceId}/{issueId}` — issue detail with the neighbourhood graph
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
| `pages/issues/index.tsx` | the issues screen — sources and the label picker on the left; the search bar over the table, or the open issue, on the right — plus the `/` source picker |
| `pages/issues/SourceSwitcher.tsx` | registered issue sources (D13) |
| `pages/issues/QueryBar.tsx` | the search bar: one query text, suggestions for the token under the caret (Ark combobox), clear and Search inside the box, Enter applies |
| `pages/issues/StateButtons.tsx` | Open N \| Closed N \| Plans N above the rows, GitHub's way: spellings of `is:open`, `is:closed`, `is:plan` with the counts the rest of the query would match |
| `pages/issues/IssueFilters.tsx` | the label combobox (Ark) that adds and removes `label:` tokens in the bar's query |
| `pages/issues/IssueList.tsx` | the table: rows with labels, metadata chips and the epic progress bar, the state buttons in its header |
| `pages/issues/IssueGraph.tsx` | the detail page's neighbourhood: mermaid flowchart from edges in a zoom / pan viewport (wheel, drag, −/+/1:1/Fit, resizable), click-through |
| `pages/issues/IssueView.tsx` | detail: header with progress, rendered fields, children, dependencies, neighbourhood, comments, TOC |
| `pages/issues/MarkdownField.tsx` | one `.markdown-body` card, with the client-side mermaid pass |
| `pages/issues/useIssues.ts` | the page's state over the api layer: the query-string filters, the rows, the open issue |
| `pages/roots/index.tsx` | `/roots`: a placeholder for the unchanged file browser |
| `pages/not-found.tsx` | fallback route |
| `api/fixtures.json` | generated; do not edit by hand |
| `api/client.ts` | fixture types (the proto messages) and the stand-in service (`listSources`, `listIssues`, `getIssue`, `listDependencies`) |
| `api/query.ts` | the query language: liqe parses, this gives `is:` `status:` `label:` `type:` `parent:` `priority:` and free text their meaning, plus token edits and suggestions |
| `api/issues.ts` | the URL's `q` spelling and the decodes the views need, over that service |
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
