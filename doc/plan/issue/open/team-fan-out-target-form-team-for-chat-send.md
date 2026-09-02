---
tags: chat send proto
---

# Team fan-out target form `team/*` for chat send (2026-09-01)

Target resolution for both member `chat send` and `chat admin send`
goes through one resolver (`resolveFor` in `crabswarm/chat/member.go`),
which understands `team/name` (exact) and a bare token as a member
name (own team first, then unique across the room). There is no way to
address a whole team, although role/group addressing (`@everyone`,
`@<role>`) is a natural chat-system use case; admin send got `*`
(whole room) but team fan-out was left out because a bare `team` token
would collide with member names.

Follow-up: model resolution as "query members by condition, with a
shortpath for all" and add `team/*` — every member of that team — to
both send paths. This composes with the existing grammar without any
name-vs-team precedence rule: bare token stays a member name,
`team/name` stays exact, `*` stays whole-room. Requires reserving `*`
as a member name in `validateName` (`crabswarm/chat/store.go`) —
today a member literally named `*` is legal and addressed as `team/*`.
Update the proto comments for `SendRequest.to` and
`AdminSendRequest.target`, the deliverer fan-out, and e2e on both
paths.
