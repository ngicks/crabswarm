# Status

Current state: **implementing** (started 2026-09-06, on `main`, one
commit per step, D26) — idea gate confirmed 2026-09-04; 27
decisions recorded (D14 generic viewer and D15 mermaid neighbourhood
2026-09-05; D16–D24 from the mock 2026-09-06, D20 dropping the board and
graph views for GitHub's Open / Closed / Plans buttons, D21 zoom and pan
on the neighbourhood, D22 keeping the GUI read-only, D23 the detail
page's layout, D24 the labels page; D25 the Go layout per the updated
design preference); contracts, nine
steps and the traceability table are in PLAN.md; the presentation mock at
`web/mock/plans_in_beads/` covers the GitHub-style list (search bar
parsed by liqe, state buttons with counts) and the detail page with its
neighbourhood (see PLAN.md "Presentation preview" for the findings it
folded in). One open question, Q14, where the query is evaluated
(decides step 4's `ListIssuesRequest`). Ready to implement, starting at
step 1.

This is intended to be the last file-authored plan. Once the field
convention (D1) is decided, re-authoring this plan as the first plan bead is
a dogfood step; until then it stays here.

## Checklist

- [x] Idea gate confirmed (IDEA.md `Gate:` line, 2026-09-04)
- [x] D1 "steps are child beads" decided
- [x] D2 "directories stay as history" decided
- [x] D3 "end-of-turn mermaid check over updated beads" decided
- [x] D4 "GUI first-class and generic over beads" decided
- [x] D5 "bd CLI is the data path" decided
- [x] D6 "issues surface beside roots, /roots and /issues" decided
- [x] D9 "reuse mermaid-lint" decided
- [x] D11 "Stop hook from this repo's apm project" decided
- [x] D7 "idea_gate_passed metadata" decided
- [x] D10 "sweep all open issues" decided
- [x] D12 "issues everywhere" decided
- [x] D13 "sources via bd where, --root/--issue" decided
- [x] D8 "daemon poll → WatchIssues; bd events --follow later" decided
- [x] D14 "generic viewer; plans = a filter plus affordances, no plan view" decided (board and graph views later left with D20)
- [x] D15 "graph via bundled mermaid, size cap" decided (agent default, overturnable)
- [x] D16 "board columns follow the filter; unconnected hidden; query string travels; `outgoing` = from side" decided in the mock (agent default, overturnable)
- [x] D17 "mock pass stayed inline" recorded; HANDOFF.md opened for the Playwright / nix browser mismatch
- [x] D18 "GitHub-style search bar, liqe, widgets edit the query" decided in the mock
- [x] D19 "type scale moves into web/src/index.css in step 6" decided
- [x] D20 "board and graph views dropped; Open / Closed / Plans buttons; neighbourhood stays" decided (closes Q15)
- [x] D21 "neighbourhood zoom and pan, hand-rolled, legible opening view" decided in the mock
- [x] D22 "GUI stays read-only; writable board is a later plan" decided
- [x] D23 "detail page: title first, section cards with header strips" decided in the mock
- [x] D24 "labels page with Active / Archived, Labels button in the sidebar" decided
- [x] D25 "Go layout per the updated design preference: thin cmd, issues/cli, ctx first, injection" decided
- [ ] Q16 "what makes a label archived" open for step 6 (default: only closed issues carry it)
- [ ] Q14 "where the query is evaluated" open for step 4
- [x] Contracts finalized (surface delta reflects D5–D15)
- [x] Traceability gate passed (PLAN.md "Traceability")
- [x] Step 1 `crabswarm/issues` client + `Where` (D5, D13, D27) — `Where`, `Client.List/Children/Get`, fake-bd tests; commit d7dcc71
- [x] Step 4a `issues/v1` proto and generated bindings — commit dd6de5b
- [x] Step 2 `mermaidlint` over `mermaid-lint` (D9, D28) — one run over temp files, findings mapped back, real-binary test; commit d6176c1
- [ ] Step 3 `crabswarm issues lint` + Stop hook + apm wiring (D3, D10, D11, D12)
- [x] Step 4 `issues/v1` proto, `SourceStore`, `IssuesService`, `Poller` → `WatchIssues` (D6, D8, D13, D29) — mounted beside PreviewService; `ListDependencies` still Unimplemented (step 7); commit 4f09d5f
- [ ] Step 5 `crabswarm preview --root / --issue`, list, remove (D13)
- [ ] Step 6 SPA tabs, `/roots` move, sources, search bar, state buttons, labels page, list + detail (D4, D6, D8, D14, D18, D19, D20, D23, D24)
- [ ] Step 7 `ListDependencies` + neighbourhood graph with zoom and pan (D15, D21)
- [ ] Step 8 instructions + agents-package boundary issue + bd events follow-up
- [ ] Step 9 dogfood: this plan as the first plan issue (D1, D7)

Next action: the user reviews the plan (D15 in particular); implementation
starts with step 1 (`crabswarm/issues` client) in a fresh worktree.
