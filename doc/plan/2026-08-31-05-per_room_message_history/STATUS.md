# Status — per-room message history

**State**: implemented (2026-09-01). All seven steps are in the tree:
`room_log` records one row per send/broadcast inside the delivering
transaction, pruned per room to the configured cap; `Store.History`
reads the tail by room id; `ChatService.History` and
`crabswarm chat history [--limit N]` expose it to members.

**Next action**: review the change. Sibling plans can now consume
`Store.History` (admin verb, MCP resource, TUI transcript view).

## Checklist

- [x] Step 1 — `ddl/room_log.sql` + `queries/room_log.sql` + sqlc
      regen (D-H1: "separate append-only `room_log` table with no
      foreign key to `members`")
- [x] Step 2 — log writes in `Store.Send`/`Store.Broadcast` + inline
      prune (D-H2: "a broadcast logs exactly one row"; D-H3: "delete
      rows beyond the newest `history_limit` ... same transaction")
- [x] Step 3 — `Store.History(ctx, room, limit)` oldest-first (D-H4:
      "keyed by room id so the admin plan and MCP plan consume it")
- [x] Step 4 — `ChatService.History` RPC + `Service.History` handler
      (D-H4: "non-destructive member RPC ... default window 50";
      D-H5: room-visible)
- [x] Step 5 — `HistoryLimit` config plumbing (D-H3: "0 means
      default, negative disables")
- [x] Step 6 — `chat history` CLI verb, presentation in `chat/cli`
      (D-H4)
- [x] Step 7 — store/service/e2e tests incl. drain-then-history case
      (D-H6: messages only)

## Deviations from the plan

- The log write sits in the shared `sendFrom`/`broadcastFrom` helpers
  rather than only in `Store.Send`/`Store.Broadcast`, so an admin send
  into a room is recorded too: it lands in members' inboxes, and a
  transcript that omitted it would misreport what the room received.
- `NewStore` resolves a zero `historyLimit` to the default instead of
  taking an already-resolved cap: a literal zero cap would prune every
  row it wrote, so the encoding lives in one place.
- The proto entry carries `Member from` / `Member to`; the plan's
  `Sender` is the Go-side name, and the schema has no such message.
- `HistoryEntry` and `Store.History` live in a new
  `crabswarm/chat/history.go` rather than in `inbox.go`.
- Old databases need no deletion: the table is created with
  `CREATE TABLE IF NOT EXISTS`, so an existing store gains it on the
  next open.

## Out of scope, left for triage

- The `crabswarm-chat` apm package (README + skill) documents the
  member verbs and does not mention `history`.

## Sibling dependencies (boundary ledger consumers)

- `2026-08-31-01-chat_admin_subcommand`: admin history verb over
  `Store.History` — not this plan's work.
- `2026-08-31-02-chat_mcp_server`: history MCP resource — not this
  plan's work.
- `2026-08-31-06-admin_tui`: live transcript view — not this plan's
  work.
