# Status

Current state: **not started** — scaffold drafted from the user-decided
policy (issue backlog entry "Reclaim a gone member's name on
duplicate-name join"); IDEA.md gate not yet confirmed.

## Checklist

- [ ] Step 1 — store lookup by (room, team, name) for the collision path
      (supports D1 "checks whether the existing holder ... is still
      around")
- [ ] Step 2 — `Service.Join` reclaim path: holder lookup → `stillKnown`
      → one retry; service tests for gone/live/human/verdict-less holder
      (D1 "frees the name and admits the joiner"; D1 "collision with a
      live member stays a clear AlreadyExists rejection"; D2 reuse of
      reap semantics)
- [ ] Step 3 — e2e recreate-shaped rejoin under the same derived name
      (goal criterion "recreated compose replica ... rejoins
      successfully")

Next action: user confirms the IDEA.md gate and answers open question 1
(join-only vs also MoveMember/admin paths).
