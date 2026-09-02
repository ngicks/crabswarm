---
tags: chat mcp proto history
---

# Publish MessageAppended so the history resource can announce updates (2026-09-02)

The proto's `MessageAppended` room-event kind exists
(`api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto`) but no
code path publishes it — every publish site builds joined/left/
state-changed events only. Because of that the
`crabswarm://chat/history` MCP resource is readable-only: its
subscribe/unsubscribe are refused via `announceable` in
`crabswarm/chat/mcpserver/resources.go`.

Follow-up: decide whether `Service.Send`/`Broadcast`
(`crabswarm/chat/service_inbox.go`, where the room-log write already
happens) should publish it; then make the history resource subscribable
by adding the URI to `announceable` and mirroring the members-resource
announcement path.
