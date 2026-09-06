# Status

Current state: **planned** — idea gate confirmed 2026-09-04; 15
decisions recorded (D14 generic viewer with board and graph, D15 graph via
bundled mermaid, added 2026-09-05); contracts, ten steps and the
traceability table are in PLAN.md; the presentation mock at
`web/mock/plans_in_beads/` covers list, board, graph and detail with the
Plans saved filter and a GitHub-style search bar parsed by liqe (D14,
D15, D18; brought up to date 2026-09-06, see PLAN.md "Presentation
preview" for the findings it folded in). Two open questions: Q14, where
the query is evaluated (decides step 4's `ListIssuesRequest`), and Q15,
how the board and the graph are scoped (before step 7). Ready to
implement, starting at step 1.

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
- [x] D7 "idea_gate metadata" decided
- [x] D10 "sweep all open issues" decided
- [x] D12 "issues everywhere" decided
- [x] D13 "sources via bd where, --root/--issue" decided
- [x] D8 "daemon poll → WatchIssues; bd events --follow later" decided
- [x] D14 "generic viewer: list, board, graph, detail; plans = saved filter" decided
- [x] D15 "graph via bundled mermaid, size cap" decided (agent default, overturnable)
- [x] D16 "board columns follow the filter; unconnected hidden; query string travels; `outgoing` = from side" decided in the mock (agent default, overturnable)
- [x] D17 "mock pass stayed inline" recorded; HANDOFF.md opened for the Playwright / nix browser mismatch
- [x] D18 "GitHub-style search bar, liqe, widgets edit the query" decided in the mock
- [x] D19 "type scale moves into web/src/index.css in step 6" decided
- [ ] Q14 "where the query is evaluated" open for step 4
- [ ] Q15 "how the board and the graph are scoped" open for step 7
- [x] Contracts finalized (surface delta reflects D5–D15)
- [x] Traceability gate passed (PLAN.md "Traceability")
- [ ] Step 1 `crabswarm/issues` client + `Where` (D5, D13)
- [ ] Step 2 `mermaidlint` over `mermaid-lint` (D9)
- [ ] Step 3 `crabswarm issues lint` + Stop hook + apm wiring (D3, D10, D11, D12)
- [ ] Step 4 `issues/v1` proto, `SourceStore`, `IssuesService` (D6, D13)
- [ ] Step 5 `crabswarm preview --root / --issue`, list, remove (D13)
- [ ] Step 6 SPA tabs, `/roots` move, sources, filters, list + detail (D4, D6, D8, D14)
- [ ] Step 7 board view (D14)
- [ ] Step 8 `ListDependencies` + graph view + local graph (D14, D15)
- [ ] Step 9 instructions + agents-package boundary issue + bd events follow-up
- [ ] Step 10 dogfood: this plan as the first plan issue (D1, D7)

Next action: the user reviews the plan (D15 in particular); implementation
starts with step 1 (`crabswarm/issues` client) in a fresh worktree.
