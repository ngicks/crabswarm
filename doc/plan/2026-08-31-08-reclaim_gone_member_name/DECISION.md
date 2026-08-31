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

## D2 — Reuse the existing reap semantics for the liveness check (stub)

Tentative: the collision path calls the existing `stillKnown`
(`crabswarm/chat/service.go:184`) so agent-only checking, the
verdict-less-failure-keeps-the-member rule, TTL caching, and reap logging
stay single-sourced. To be confirmed with the idea gate / question round.

## D3 — Join-only; MoveMember and admin paths unchanged (stub)

Tentative default for open question 1: reclaim happens only in
`Service.Join`; an admin moving a member onto a stale name keeps being
rejected until the admin plane grows explicit member removal.
