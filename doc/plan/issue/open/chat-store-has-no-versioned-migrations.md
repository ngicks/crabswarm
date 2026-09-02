---
tags: chat store migration
---

# Chat store has no versioned migrations (2026-09-02)

`NewStore` (`crabswarm/chat/store.go`) runs `schema.DDL()` — every
`ddl/*.sql` concatenated, so `room_log` shares the one database — on
every open, and the daemon does call it at start
(`openChatStore` in `crabswarm/server/server.go`). But the DDL is
`CREATE TABLE/INDEX IF NOT EXISTS` only: a new table self-heals on
restart, a new column on an existing table never does, and nothing
records or checks a schema version (`PRAGMA user_version` is 0, never
read). Commit 8a9e288 already hit this when it added
`members.state_reported_at NOT NULL` ("Existing dev DBs ... must be
recreated"), and the next column addition will break every existing
install the same way, undetected. Note this was *not* the cause of the
admin-TUI "missing row" report (next entry): a stale schema errors with
`no such column`, never `no rows`, and `chat admin list` selecting
`state_reported_at` proves the live schema is current.

Follow-up: read `PRAGMA user_version` in `NewStore`, apply ordered
migration steps and write the version back in the same transaction;
refuse to open a DB newer than the binary. Supersedes the "Release note:
old dev chat DBs need recreating" entry once it lands.
