# Status

Current state: **planned, not started** — IDEA.md gate confirmed by the
user 2026-08-31; open questions drained (reclaim also covers admin
MoveMember, D3).

## Checklist

- [ ] Step 1 — store lookup by (room, team, name) for the collision path
      (supports D1 "checks whether the existing holder ... is still
      around")
- [ ] Step 2 — `Service.Join` reclaim path: holder lookup → `stillKnown`
      → one retry; service tests for gone/live/human/verdict-less holder
      (D1 "frees the name and admits the joiner"; D1 "collision with a
      live member stays a clear AlreadyExists rejection"; D2 reuse of
      reap semantics)
- [ ] Step 3 — admin move reclaim: `NewAdminService` provider parameter +
      `AdminService.MoveMember` holder check via shared helper; admin
      service tests (D3 "applies to AdminService.MoveMember name
      collisions as well"; D2 shared helper)
- [ ] Step 4 — e2e recreate-shaped rejoin under the same derived name
      (goal criterion "recreated compose replica ... rejoins
      successfully")

Next action: implement step 1.
