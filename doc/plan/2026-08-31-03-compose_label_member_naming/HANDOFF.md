# Handoff

Deferred discoveries from the implementation run; needs user triage.

## Recreated compose replica cannot rejoin — stale member holds its derived name

Found by the final review pass; verified causal chain, blocks merge until
the user picks a policy:

- The identity token is `$CMDMAN_CMD_ID`, the per-instance cmdman command
  ID (`crabswarm/chat/cli/token.go:14-24`).
- cmdman compose recreate is remove-then-create: a recreated replica gets a
  **new** command ID but keeps the **same** `cmdman.compose.command` /
  `cmdman.compose.scale-index` labels. Same for `compose down`+`up` and
  `compose scale` down/up.
- The new token is unknown to the store, so `Service.Join` takes the
  fresh-join path and derives the identical name (e.g. `worker-1`), which
  `Store.Join` rejects (`crabswarm/chat/member.go:51-53` →
  `codes.AlreadyExists`). The join fails outright.
- The stale member is never freed: reaping is lazy and only fires when the
  stale member itself makes an RPC (`crabswarm/chat/service.go:169,
  184-208`), and its command is gone; there is no sweep and no admin
  member-removal RPC. The store persists across daemon restarts
  (`$XDG_STATE_HOME/crabswarm/chat.db`).

Before this change the fresh token produced a unique `agent-<hex8>`, so
recreation only left cosmetic clutter; now it is a hard, non-self-healing
join failure in the feature's own target scenario (workaround: explicit
`--name`).

Not fixed in-run because every fix is new collision policy — e.g. evicting
a name-colliding member whose token the provider no longer resolves, or
falling back to the token-derived name (silent aliasing in spirit) — and
the plan explicitly declared collision semantics a non-goal ("clear
rejection stays"). The shipped failure mode is a loud `AlreadyExists` at
join time, per that inherited rule.

Suggested regression test once decided: `Service.Join` with a
provider-derived name already taken in the same team by a token the
provider no longer knows.

## Untested guarantee: `/` in a provider-derived name (minor)

The comment at `crabswarm/chat/resolver/cmdman.go:120-125` leans on
join-time rejection of `/` in names, but only `req.Name` is test-driven
through that path (`crabswarm/chat/service_member_test.go:119`); no test
sends an `info.Name` containing `/` through `Service.Join`. A future
"cmdman is trusted, skip validation" refactor could silently break the
documented guarantee.
