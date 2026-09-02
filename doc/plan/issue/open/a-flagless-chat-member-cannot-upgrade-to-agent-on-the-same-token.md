---
tags: chat join notify
---

# A flagless chat member cannot upgrade to agent on the same token (2026-09-02)

`Store.Join` is first-join-wins for the member kind: a member that
joined without `--agent` stays `KindHuman` (inbox-only) until it leaves
and re-joins. Harmless for the MCP bridge (it always declares agent on
its first join), but a person who joins by hand and then starts a
harness on the same token stays inbox-only, with no verb to flip the
kind in place. Documented in `Service.Join`'s doc comment and pinned by
the join-idempotency tests.

Follow-up: decide whether a re-join (or an admin verb) may upgrade the
kind before this bites a real workflow.
