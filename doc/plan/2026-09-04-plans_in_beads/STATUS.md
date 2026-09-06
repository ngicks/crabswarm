# Status

Current state: **planned** — idea gate confirmed 2026-09-04; 20
decisions recorded (D14 generic viewer and D15 mermaid neighbourhood
2026-09-05; D16–D20 from the mock 2026-09-06, D20 dropping the board and
graph views for GitHub's Open / Closed / Plans buttons); contracts, nine
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
- [ ] Q14 "where the query is evaluated" open for step 4
- [x] Contracts finalized (surface delta reflects D5–D15)
- [x] Traceability gate passed (PLAN.md "Traceability")
- [ ] Step 1 `crabswarm/issues` client + `Where` (D5, D13)
- [ ] Step 2 `mermaidlint` over `mermaid-lint` (D9)
- [ ] Step 3 `crabswarm issues lint` + Stop hook + apm wiring (D3, D10, D11, D12)
- [ ] Step 4 `issues/v1` proto, `SourceStore`, `IssuesService` (D6, D13)
- [ ] Step 5 `crabswarm preview --root / --issue`, list, remove (D13)
- [ ] Step 6 SPA tabs, `/roots` move, sources, search bar, state buttons, list + detail (D4, D6, D8, D14, D18, D19, D20)
- [ ] Step 7 `ListDependencies` + neighbourhood graph (D15)
- [ ] Step 8 instructions + agents-package boundary issue + bd events follow-up
- [ ] Step 9 dogfood: this plan as the first plan issue (D1, D7)

Next action: the user reviews the plan (D15 in particular); implementation
starts with step 1 (`crabswarm/issues` client) in a fresh worktree.
