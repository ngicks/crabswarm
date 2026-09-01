# Status

Current state: **implemented 2026-09-01** — all four steps landed in
`crabswarm/chat` plus the e2e case; `go test ./crabswarm/chat/... ./e2e/...`
passes.

## Checklist

- [x] Step 1 — store lookup by (room, team, name) for the collision path
      (supports D1 "checks whether the existing holder ... is still
      around")
- [x] Step 2 — `Service.Join` reclaim path: holder lookup → liveness check
      → one retry; service tests for gone/live/human/verdict-less holder
      (D1 "frees the name and admits the joiner"; D1 "collision with a
      live member stays a clear AlreadyExists rejection"; D2 reuse of
      reap semantics)
- [x] Step 3 — admin move reclaim: `NewAdminService` provider parameter +
      `AdminService.MoveMember` holder check via shared helper; admin
      service tests (D3 "applies to AdminService.MoveMember name
      collisions as well"; D2 shared helper)
- [x] Step 4 — e2e recreate-shaped rejoin under the same derived name
      (goal criterion "recreated compose replica ... rejoins
      successfully")

## Deviation from the plan

Step 2 planned to run the collision check through `Service.stillKnown`,
which answers from the 30s provider-check cache. That cache vouches for
exactly the holder a recreated replica has just replaced — the
predecessor's own join stamped it moments earlier — so the plan's own
step-4 scenario would have been refused. The reap semantics D2 asks for
are instead shared as `checkLiveness` (`crabswarm/chat/service.go`), and
the collision path (`reclaimName`, same file) asks the provider afresh:
`stillKnown` keeps the cache and its behaviour unchanged, a collision
pays one lookup. `TestService_JoinReclaimLooksPastTheCachedVerdict` pins
it.

Next action: none — merge and, if wanted, the manual cmdman-compose
recreate check from the plan's verification notes.

HANDOFF.md folded into doc/plan/issue/issue.md 2026-09-02 (run of the
implement-all-plans goal); entries there are the durable copies.
