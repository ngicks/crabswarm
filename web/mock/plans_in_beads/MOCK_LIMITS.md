# What this mock fakes, and what it therefore cannot validate

This directory is a **presentation mock** of the Issues tab planned in
`doc/plan/2026-09-04-plans_in_beads/PLAN.md` (sections "Preview integration",
"SPA routes", "Proto"; decisions D1, D6, D7, D8, D12, D13). It exists to make
the reading experience judgeable before step 4 and step 6 are built. It is not
a prototype of the implementation and shares no code with it.

## What is faked

- **No daemon, no API.** There is no `IssuesService`, no Connect client, no
  HTTP call of any kind. `ListSources`, `AddSource`, `RemoveSource`,
  `ListIssues` and `GetIssue` do not exist here; `data.ts` imports one
  `fixtures.json` and filters it in the browser.
- **No `bd`.** Nothing shells out. The issue data is a frozen
  `bd export --all` of the crabswarm backlog taken 2026-09-04
  (`doc/plan/2026-09-04-plans_in_beads/mock/issues-export.jsonl`, with the
  throwaway `sample-title` probe dropped).
- **The plan issue does not exist in beads.** `crabswarm-plan1` and its eight
  `crabswarm-plan1.N` step children are synthesized by `gen.go` out of this
  plan directory's markdown, following D1's field convention (idea →
  description, plan → design, criteria → acceptance, status → notes,
  DECISION.md entries → `Decision:` comments, steps → child tasks ordered by
  `blocks`). Step 8 of the plan — writing them for real — has not happened.
- **The second source is invented.** `agents-package` and its three issues are
  made up so the source switcher has somewhere to switch. No `bd where` ever
  ran: `gen.go` computes both source ids as a sha256 prefix of the `.beads`
  path, imitating what the daemon's registry would key on (D13).
- **Markdown is rendered ahead of time.** `gen.go` calls the previewer's own
  renderer (`crabswarm/preview/render`) once per field at generation time, so
  the HTML in the fixture is exactly what `GetIssue` would return — but there
  is no per-request render, no cache, no render error path, and editing a plan
  file does not change the mock until `gen.go` is run again.
- **No `WatchIssues` stream and no poll.** D8's daemon-side 10 s poll is
  replaced by the header's "simulate change" button, which bumps one issue's
  title and `updated_at` in memory. It is labelled *simulated* on purpose.
- **No Roots tab.** The file browser is not re-mocked; `/roots` shows a
  placeholder saying so, and no `/roots/{rootId}/{path...}` document resolves.
- **Routing is mock-local.** `App.tsx` has its own `preact-iso` router with the
  planned URL shapes. `web/src/routes.tsx` is untouched, and the app still
  serves the file browser at `/r/{rootId}/…`.
- **Rendering support files are served locally.** The alert and chroma
  stylesheets that `signals/ui.ts` links from `/assets/css/…` are written into
  `public/assets/css/` by `gen.go` instead of being served by the daemon.
  MathJax is not loaded (no `/vendor/mathjax` here), and image sources are not
  rewritten to `/raw` (no raw endpoint), so a math formula or a relative image
  in an issue will not render as it would in the app.
- **Read-only.** Nothing writes to beads, which matches the plan (editing in
  the GUI is a non-goal) but also means no failure path is exercised.

## Shape deviations from PLAN.md's proto sketch

`fixtures.json` is protobuf-JSON spelling of `ngicks.crabswarm.issues.v1`
(camelCase fields, `ISSUE_STATUS_*` enum names, RFC 3339 timestamps), with
these differences — each worth a decision when step 4 writes the real proto:

- **`Issue.sourceId` added.** The fixture is one flat list across both sources;
  in the RPC the source comes from `ListIssuesRequest` / `GetIssueRequest`.
- **`close_reason` is rendered.** The sketch has `string close_reason`, but
  every close reason in the real backlog is markdown (paragraphs, inline code),
  so the mock renders it to a `RenderedField` and shows it as a card. Either
  the proto carries a `RenderedField` here too, or the SPA renders it itself.
- **Heading anchors are namespaced per field** (`description--<id>`) by
  `MarkdownField`, because description and design are two whole markdown
  documents on one page and goldmark's `AutoHeadingID` can produce the same id
  in both. The wire `RenderedField.toc` ids are unprefixed, as the proto says;
  the namespacing is a client concern the real view will also have.
- **Counts are only as good as the generator.** `commentCount` comes from bd's
  export; `childCount` is set on the plan issue only. The real `ListIssues`
  would fill both for every issue.

## What this mock therefore cannot validate

- `bd` latency (~1.5 s per invocation) — whether opening an issue, listing a
  source, or switching sources feels acceptable when every read is a subprocess.
- The cost and correctness of D8's poll: one `bd list` per source per 10 s,
  diffing ids and `updated_at`, and what a push does to a view being read
  (the button only shows the intended effect on one issue).
- Source discovery: `BD_JSON_ENVELOPE=1 bd where --json`, the `no_beads_directory`
  error path, deduplicating worktrees by `.beads` path, and the
  `crabswarm preview --root / --issue`, `preview list`, `preview remove` surface (D13).
- The mermaid guard: `crabswarm issues lint`, the `mermaid-lint` mapping back to
  issue/field/comment/line, and the Stop hook blocking a turn (D3, D9, D10, D11).
- The `/r/…` → `/roots/…` move (D6): nothing here proves the file browser and
  its Playwright specs survive the rename.
- Live updates end to end: stream reconnection, query invalidation, and what a
  reorder does under the reader's cursor.
- Decoding real `bd` JSON, where empty fields are omitted, and statuses or
  types this export happens not to contain.
- Anything about the shipped artifact: `web/dist`, `dist.tar.zst`, `embed.go`.
  The mock builds to its own git-ignored `dist/` and is never embedded.
