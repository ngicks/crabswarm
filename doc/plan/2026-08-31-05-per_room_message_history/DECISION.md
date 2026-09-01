# Decisions — per-room message history

## D-H1: New `room_log` table, not a flag on `messages` (automatic decision)

**Choice**: a separate append-only `room_log` table with no foreign key
to `members`.
**Rationale**: the `messages` table is delivery state — per-recipient,
drained on read, `ON DELETE CASCADE` from `members`. Reusing it with a
"read" marker would change `chat read` semantics and erase a room's
transcript whenever a member leaves. History must outlive both
delivery and membership.
**Rejected**: soft-delete flag on `messages`; FK to `members` with
cascade.

## D-H2: One row per utterance (automatic decision)

**Choice**: a broadcast logs exactly one row (`to_name`/`to_team`
empty); a directed send logs one row naming the addressee.
**Rationale**: history records what was said in the room; delivery
records who received it. Per-recipient history rows would show one
broadcast N times and grow with room size, not conversation length.
**Rejected**: per-recipient log rows mirroring the inbox writes.

## D-H3: Inline prune to a per-room row cap, default 1000 (automatic decision)

**Choice**: after each log insert, delete rows beyond the newest
`history_limit` for that room, in the same transaction. Default 1000;
`chat.history_limit` config key / `CRABSWARM_CHAT_HISTORY_LIMIT` env;
0 means default, negative disables logging entirely.
**Rationale**: self-maintaining, no background job, bounded DB growth;
the 0/negative encoding keeps `chat.Config`'s "meaningful when empty"
property (`config.go` doc comment) without inventing a package default
struct. Row-count caps are deterministic where age-based pruning
depends on traffic.
**Rejected**: age-based pruning; a periodic cleanup goroutine; making
0 mean "disabled" (would turn the zero value into silent data loss).

## D-H4: Member surface is `chat history` + `ChatService.History` (automatic decision)

**Choice**: a non-destructive member RPC `History(limit)` scoped to the
caller's room (default window 50, oldest-first), CLI verb
`crabswarm chat history [--limit N]`. `Store.History(ctx, room, limit)`
is keyed by room id so the admin plan and MCP plan consume it without
member auth.
**Rationale**: members already carry room identity in their token, so
no room argument; naming it `history` contrasts cleanly with the
consuming `read`.
**Rejected**: `chat read --history` flag (overloads a destructive verb
with a non-destructive mode); pagination cursors now (cap is 1000 rows,
a limit knob suffices until someone needs paging).

## D-H5: Room-visible transcript (automatic decision)

**Choice**: every room member can read the full room history, directed
messages included.
**Rationale**: the user framed history as a shared "useful resource"
for members; directed sends in a coordination room are work traffic,
not private mail. Restricting per-recipient would reduce history back
to an inbox copy.
**Rejected**: filtering directed messages to sender/recipient only.

## D-H6: Messages only, no system events (automatic decision)

**Choice**: the log records send/broadcast text only; joins, leaves,
team moves, and state changes are not logged.
**Rationale**: keeps schema and scope minimal; membership churn is
queryable live via `ListMembers`, and nothing in the stated use cases
needs historical membership. Revisit if the admin TUI plan wants
event annotations.
**Rejected**: a polymorphic event log (kind column + nullable fields).

## D-H7: Log at the delivery helpers, so host/admin sends are recorded (automatic decision)

**Choice**: the room-log write lives in the store's shared delivery paths
(`sendFrom`/`broadcastFrom` in `crabswarm/chat/inbox.go`), not only in the
member-facing `Store.Send`/`Store.Broadcast` wrappers.
**Rationale**: an admin/host send lands in member inboxes; a transcript
that omitted it would misreport what the room received. Pinned by
`TestStore_HistoryRecordsWhatTheHostSaid` (crabswarm/chat/history_test.go).
**Rejected**: logging only member utterances (the plan's literal wording,
which predates the admin send plane).

## D-H8: The 0→1000 cap resolution happens in NewStore (automatic decision)

**Choice**: `NewStore` resolves a zero cap to the default; the server
wiring passes the config value through unresolved.
**Rationale**: a literal 0 cap would prune every row it just wrote, and
several existing call sites construct stores without going through server
wiring — resolving at the single constructor keeps them all safe.
**Rejected**: resolving at the server wiring (the plan's wording).

## D-H9: Proto entries reuse `Member`, not a new `Sender` message (automatic decision)

**Choice**: `HistoryEntry.from`/`to` are `Member`, matching how
`Message.from` is already typed in the schema.
**Rationale**: the plan's sketch named a `Sender` message that does not
exist in the proto; inventing it would duplicate `Member` for no gain.
