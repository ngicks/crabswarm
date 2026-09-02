---
tags: chat admin join
---

# RegisterMember still refuses a name a gone agent holds (2026-09-02)

`AdminService.RegisterMember` hands the requested name to `Store.Join`
and returns `AlreadyExists` on collision, even when the holder is a gone
agent — the same ghost-holder situation now reclaimed on the
`Service.Join` and `AdminService.MoveMember` collision paths (both go
through the shared `checkLiveness` helper in
`crabswarm/chat/service.go`).

Follow-up: decide whether a human registration may evict a gone agent's
name; if yes, route the register collision through the same
holder-check-and-reap helper.
