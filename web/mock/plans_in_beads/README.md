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

| File | Role |
|---|---|
| `index.html`, `main.tsx` | mount point; imports the app's stylesheet and theme effect |
| `mock.css` | imports `../../src/index.css` and declares this directory as a tailwind source |
| `App.tsx` | shell and routes (`/`, `/roots…`, `/issues/{sourceId}[/{issueId}]`) |
| `data.ts` | fixture types (the proto messages), filters, and the simulated push |
| `components/TabHeader.tsx` | Roots \| Issues tabs, "simulate change", theme toggle |
| `components/SourceSwitcher.tsx` | registered issue sources (D13) |
| `components/IssueList.tsx` | status chips, label multi-select, plans-only toggle, search |
| `components/IssueView.tsx` | header, rendered fields, children, dependencies, comments, TOC |
| `components/MarkdownField.tsx` | one `.markdown-body` card, with the client-side mermaid pass |
| `fixtures.json` | generated; do not edit by hand |
| `public/assets/css/` | generated; alert + chroma stylesheets the rendered HTML expects |

Nothing under `web/src` is modified: the mock imports `src/index.css`,
`src/signals/ui.ts` and `src/components/ThemeToggle.tsx`, and copies the
patterns it needs from `Layout.tsx`, `RootSwitcher.tsx`, `FileTree.tsx` and
`DocView.tsx`.
