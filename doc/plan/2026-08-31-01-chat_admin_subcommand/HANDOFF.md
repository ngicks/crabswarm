# Handoff

## Bare-team target fan-out for send

`chat admin send ROOM TEAM TEXT` (whole-team addressing) was planned but
not implemented: the shared resolver (crabswarm/chat/member.go,
resolveFor) has no bare-team form on the member path either, and adding
it needs a name-vs-team collision rule. If wanted, land it on both
member and admin send plus the proto comment for AdminSendRequest.target.

## `admin log` verb + AdminService.History RPC

Deliberately absent, blocked on plan 2026-08-31-05-per_room_message_history
delivering room-keyed history storage (`Store.History`). No stub was
added, per the failure-experience decision.
