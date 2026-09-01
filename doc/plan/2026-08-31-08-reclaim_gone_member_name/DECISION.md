# Decisions

## D1 — Free a gone holder's name at join time (user decision)

**Choice**: on a duplicate-name join, the server checks whether the
existing holder of that name is still around, and when the previous member
is gone (its token no longer resolves through the team-info provider),
frees the name and admits the joiner. A collision with a live member stays
a clear `AlreadyExists` rejection.

**Rationale**: decided by the user (2026-08-31) after the compose-label
naming work surfaced the recreate scenario: a recreated replica derives
its predecessor's exact name, and nothing else ever frees it. This keeps
the inherited "duplicate participant name → clear rejection, not silent
aliasing" rule intact by fixing who counts as a participant, not the
rejection itself.

**Rejected**: keeping the hard rejection (breaks the label-derived-naming
feature in its own target scenario); automatic suffixing (explicitly ruled
out by the inherited rule); requiring the operator to remove the stale
member by hand (no admin verb exists, and the flow must be automatic).

## D2 — Reuse the existing reap semantics for the liveness check

**Choice**: the collision paths follow the semantics of `stillKnown`
(`crabswarm/chat/service.go:184`) — agent-only checking, a verdict-less
provider failure keeps the holder (and rejects the newcomer), a reaped
member is removed with its inbox and logged. Join uses `stillKnown`
itself; the admin move path shares the holder-check-and-reap logic via an
in-package helper (AdminService has no TTL cache; a direct provider check
is fine for a rare admin operation).

**Rationale**: one definition of "gone" across the daemon; a flaky cmdman
must not free names, exactly as it must not empty rooms.

**Rejected**: a separate, looser staleness heuristic for collisions;
duplicating the reap logic per path.

## D3 — Reclaim covers admin MoveMember too (user decision)

**Choice**: the gone-holder check applies to `AdminService.MoveMember`
name collisions as well as `Service.Join` — a gone holder does not block
an admin move. Chosen by the user (2026-08-31) over the join-only
default.

**Consequence**: `NewAdminService` gains a `TeamInfoProvider` parameter
(see PLAN.md Public surface delta); the sole caller is
`crabswarm/server/server.go:152`.

**Rejected**: join-only reclaim (leaves the admin hitting a wall a plain
member would pass through, with no removal verb to clear it).

## D4 — collision liveness check bypasses the join-time TTL cache (automatic decision)

**Choice**: the collision paths do not run `s.stillKnown(ctx, holder)` as
the plan said. The reap semantics D2 asks for are extracted into a shared
`checkLiveness` helper (`crabswarm/chat/service.go:220`) that asks the
team-info provider directly, and both collision paths go through that.
`stillKnown` keeps its `providerCheckTTL` cache layered on top of the same
helper and behaves exactly as it did before.

**Rationale**: `stillKnown` answers from a 30s cache of successful provider
lookups, and the predecessor's own join is what stamps its token as
verified — so the cache would vouch for precisely the holder a recreated
replica replaces, and D1's target scenario would be refused until the entry
expired. A collision is rare enough to be worth one direct lookup, and the
answer decides whether the caller gets in at all. The cache-bypass is
pinned by `TestService_JoinReclaimLooksPastTheCachedVerdict`
(`crabswarm/chat/service_member_test.go:152`).

**Rejected**: sleeping the cache out in tests (hides a real 30s window in
which a user is refused, rather than removing it); invalidating the
holder's cache entry on collision (more machinery for the same result as a
direct check).
