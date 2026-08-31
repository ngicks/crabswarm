# Issue backlog

Durable follow-ups that outlive their originating plan directories.
Append only; never rewrite or reorder existing entries.

## Unify the two buf generate templates (2026-08-29)

`api/buf.gen.yaml` (full: Go + TS plugins, run by `go generate` in
`api/`) and `api/buf.gen.ts.yaml` (TS-only subset, run by the
`web/package.json` "gen" script) duplicate the `managed` block and the
`protoc-gen-es` plugin block, with a comment obliging humans to keep
them in sync by hand. The duplication is meaningless over-engineering —
both sides must be in sync anyway, so a single file should cover both.

Follow-up: collapse to one template covering both toolchains — e.g.
keep only `buf.gen.yaml` and have `web`'s "gen" script invoke it (Go
plugins must then be present for the web build), or generate the TS
solely from the Go-side `go generate` and drop the pnpm-side gen step.
Decide which toolchain owns generation, then delete `buf.gen.ts.yaml`.

## Add `crabswarm chat admin` subcommand (2026-08-31)

Admin works fundamentally differently from members: it authenticates by
age identity + nonce challenge (`crabswarm/chat/admin.go`,
`crabswarm/chat/cli/admin.go`), never holds a member token, never joins
a room as a member, and always names the room it operates on
explicitly. Today the admin verbs (`chat register`, `chat team`) sit
mixed among the member verbs under `chat`, and there is no admin way to
send into or observe a room — participating requires minting a human
member token via `chat register` and hand-exporting it, which is
counter-intuitive.

Follow-up: group the admin plane under `crabswarm chat admin ...`
(register, team, plus new room-scoped verbs such as send/log), all
age-authenticated and all taking an explicit room id. Do not add an
`--admin` flag to `chat join`; admin never becomes a member.

## Transform chat daemon into an MCP server, keeping the CLI (2026-08-31)

Native notification/hook methods vary between harnesses, so the chat
side should become an MCP server while every existing CLI verb stays.
One MCP server instance corresponds to one agent and calls Join
automatically at startup; `Service.Join` is already idempotent for a
known token (`crabswarm/chat/service_member.go`), so multiple MCP
servers in the same container are fine. Join reporting then comes from
the server itself, replacing the SessionStart hook.

Constraints verified against the MCP spec (2026-07-28) and Claude Code
docs: MCP defines no client-to-server busy/idle notification and no
turn-end signal, and Claude Code hooks cannot call MCP tools yet
(anthropics/claude-code#26112), so harness-state push stays hook-driven
(hooks shelling out to the CLI / a thin RPC). Related known defect: an
ESC interrupt fires no Stop hook ("Stop hooks ... don't fire on user
interrupts", code.claude.com/docs/en/hooks-guide), so the member state
sticks at working/waiting and `notify.SendKeys` (gating on StateDone in
`crabswarm/chat/notify/notify.go`) never nudges that member again; the
`Notification` hook's `idle_prompt` matcher (fires ~60s idle) mapped to
`report-state done`, plus a staleness fallback in the nudge gate, is
the available mitigation.

Follow-up: design the MCP surface (tools for send/read/report-state,
resources + `resources/updated` subscriptions for member state and room
history), wire server-startup auto-join, and keep the CLI as the
hook-facing entry.

## Default member name from cmdman compose labels (2026-08-31)

Member names must be aliased from cmdman-compose's command name, not
from the token. Today the default is `agent-<first-8-hex-of-token>`
(`defaultName` in `crabswarm/chat/service.go`), and the resolver
decodes only `dir` and the labels map, using just
`cmdman.compose.project` (`crabswarm/chat/resolver/cmdman.go`).

Follow-up: read `cmdman.compose.command` and
`cmdman.compose.scale-index` from the already-decoded labels (i.e. what
`cmdman inspect $ID --format '{{index .Config.Labels
"cmdman.compose.command"}}'` returns), carry a `Name` on
`resolver.TeamInfo` (`crabswarm/chat/resolver/resolver.go`), and have
`Service.Join` prefer it (e.g. `<command>-<scale-index>`) over the
token-derived fallback when `--name` is absent.

## Nudge opt-in kind for members (deferred) (2026-08-31)

Every cmdman-resolved member is unconditionally `KindAgent`
(`crabswarm/chat/service_member.go`), so a human joining chat from a
plain cmdman-tracked shell gets the keystroke nudge typed into the
shell and executed as a command line — the injection guards
(`crabswarm/chat/internal/cmdman/cmdman.go`,
`crabswarm/chat/notify/notify.go`) cannot tell a harness from a shell.
Deferred for now because the admin plane (see the `chat admin` entry)
covers the human case: admin never joins as a member.

Follow-up (if plain-shell membership returns): make nudging opt-in at
registration/join — e.g. an `--agent` flag meaning "notify by
keystroke injection"; without it, the member is inbox-only and the
notify path never types at its terminal.

## Per-room message history (2026-08-31)

Chat history does not exist: the store is a pure inbox — per-recipient
message rows drained with `DELETE FROM messages WHERE recipient = ?` on
read (`crabswarm/chat/internal/schema/ddl/schema.sql`,
`queries/queries.sql`). Nothing retains what was said, so neither the
admin nor a member can look back at old conversation, which would be a
useful resource for agents catching up on context.

Follow-up: add an append-only per-room log table written on send, with
a retention cap (row limit and/or age-based pruning), plus read access
for both the admin plane and members (CLI verb now; an MCP resource
once the server lands).

## Admin TUI screen (2026-08-31)

The admin needs its own interactive surface: unlike members it always
specifies a room id, and its job is watching over the agents'
conversation rather than participating in an inbox. Today there is no
TUI code anywhere under `crabswarm/chat/` or
`cmd/crabswarm/commands/`, `chat read` is one-shot with no follow mode,
and the `web/` SPA has generated chat proto types
(`web/src/gen/ngicks/crabswarm/chat/v1/`) that nothing consumes.

Follow-up: build an admin TUI (room picker / explicit room id, live
view of the room's conversation via the per-room history plus a
streaming or polling tail, send-as-admin input). Depends on the `chat
admin` subcommand and per-room history entries above.
