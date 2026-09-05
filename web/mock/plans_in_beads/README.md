# Issues tab — presentation mock

A runnable mock of the **Issues** tab planned in
`doc/plan/2026-09-04-plans_in_beads/PLAN.md`. It renders the real crabswarm
backlog plus the plan itself, as the plan would be stored in beads (D1), so the
reading experience can be judged before steps 4 and 6 are implemented.

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
- `/issues/{sourceId}` — issue list for a source
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
| `pages/issues/index.tsx` | the issues screen — sources and list on the left, detail on the right — plus the `/` source picker |
| `pages/issues/SourceSwitcher.tsx` | registered issue sources (D13) |
| `pages/issues/IssueList.tsx` | status chips, label multi-select, plans-only toggle, search |
| `pages/issues/IssueView.tsx` | header, rendered fields, children, dependencies, comments, TOC |
| `pages/issues/MarkdownField.tsx` | one `.markdown-body` card, with the client-side mermaid pass |
| `pages/issues/useIssues.ts` | the page's state over the api layer: filters, and the open issue |
| `pages/roots/index.tsx` | `/roots`: a placeholder for the unchanged file browser |
| `pages/not-found.tsx` | fallback route |
| `api/fixtures.json` | generated; do not edit by hand |
| `api/client.ts` | fixture types (the proto messages) and the stand-in service (`listSources`, `listIssues`, `getIssue`) |
| `api/issues.ts` | the request's filters and the decodes the views need, over that service |
| `api/events.ts` | the simulated `WatchIssues` push |
| `signals/issues.ts` | client state: which issue the reader has open |
| `lib/paths.ts`, `lib/format.ts` | `/issues/…` URLs; timestamp and status spelling |
| `public/assets/css/` | generated; alert + chroma stylesheets the rendered HTML expects |

Nothing under `web/src` is modified: the mock imports `src/index.css`,
`src/signals/ui.ts` and `src/components/ThemeToggle.tsx`, and copies the
patterns it needs from the app's `Layout.tsx`, `RootSwitcher.tsx`,
`FileTree.tsx` and `DocView.tsx`.
