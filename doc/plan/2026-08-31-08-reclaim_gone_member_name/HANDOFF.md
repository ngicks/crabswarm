# Handoff

## `AdminService.RegisterMember` still refuses a name a gone agent holds

Registering a human under a name that an agent already holds returns
`AlreadyExists` even when that agent has vanished
(`crabswarm/chat/admin_rooms.go:113`, which hands the name straight to
`Store.Join`). It is the same ghost-holder situation `Service.Join` and
`AdminService.MoveMember` now clear on their name-collision paths — both
reclaim through the shared `checkLiveness` helper in
`crabswarm/chat/service.go` — but the register path sat outside this
change's scope and was left as it was.

Extending reclaim there needs a call nobody has made yet: whether a human
registration may evict the name of a gone agent, or whether an operator
should be told the name is taken and pick another.

## Pre-existing `modernize` lint findings in untouched files

A repo-wide `golangci-lint` run surfaces two `modernize` findings in files
this change never touched: `crabswarm/hook/audit_test.go:90` and
`crabswarm/hook/audit_test.go:103` both want their `errors.As` call
replaced with `errors.AsType`. Unrelated to name reclaim, and a separate
pass to fix.
