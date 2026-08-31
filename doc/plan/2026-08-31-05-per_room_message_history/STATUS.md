# Status — per-room message history

**State**: not started. Plan drafted with automatic decisions (user
directed skipping question rounds); IDEA.md gate not confirmed —
review IDEA.md and DECISION.md before implementation.

**Next action**: user reviews the automatic decisions (especially
D-H3 retention encoding and D-H5 room-visible transcript), confirms
the IDEA.md gate, then step 1.

## Checklist

- [ ] Step 1 — `ddl/room_log.sql` + `queries/room_log.sql` + sqlc
      regen (D-H1: "separate append-only `room_log` table with no
      foreign key to `members`")
- [ ] Step 2 — log writes in `Store.Send`/`Store.Broadcast` + inline
      prune (D-H2: "a broadcast logs exactly one row"; D-H3: "delete
      rows beyond the newest `history_limit` ... same transaction")
- [ ] Step 3 — `Store.History(ctx, room, limit)` oldest-first (D-H4:
      "keyed by room id so the admin plan and MCP plan consume it")
- [ ] Step 4 — `ChatService.History` RPC + `Service.History` handler
      (D-H4: "non-destructive member RPC ... default window 50";
      D-H5: room-visible)
- [ ] Step 5 — `HistoryLimit` config plumbing (D-H3: "0 means
      default, negative disables")
- [ ] Step 6 — `chat history` CLI verb, presentation in `chat/cli`
      (D-H4)
- [ ] Step 7 — store/service/e2e tests incl. drain-then-history case
      (D-H6: messages only)

## Sibling dependencies (boundary ledger consumers)

- `2026-08-31-01-chat_admin_subcommand`: admin history verb over
  `Store.History` — not this plan's work.
- `2026-08-31-02-chat_mcp_server`: history MCP resource — not this
  plan's work.
- `2026-08-31-06-admin_tui`: live transcript view — not this plan's
  work.
