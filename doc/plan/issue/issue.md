Issue backlog: durable follow-ups that outlive their originating plan
directories. One `#` heading per issue item. Append only; never rewrite
or reorder existing entries.

# Unify the two buf generate templates (2026-08-29)

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

# Add `crabswarm chat admin` subcommand (2026-08-31)

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

# Transform chat daemon into an MCP server, keeping the CLI (2026-08-31)

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

# Nudge opt-in kind for members (deferred) (2026-08-31)

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

# Per-room message history (2026-08-31)

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

# Admin TUI screen (2026-08-31)

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

# Reclaim a gone member's name on duplicate-name join (2026-08-31)

A recreated compose replica cannot rejoin the chat room: the identity
token is `$CMDMAN_CMD_ID`, the per-instance cmdman command ID
(`crabswarm/chat/cli/token.go`), and cmdman compose recreate (also
`down`+`up` and `scale` down/up) is remove-then-create — the replica
comes back with a new command ID but the same `cmdman.compose.command` /
`cmdman.compose.scale-index` labels, so it derives the same default name
(e.g. `worker-1`). The new token is unknown to the store, `Service.Join`
takes the fresh-join path, and `Store.Join` rejects the taken name
(`crabswarm/chat/member.go` → `codes.AlreadyExists`). The stale member
holding the name is never freed: reaping is lazy and fires only when the
stale member itself makes an RPC (`crabswarm/chat/service.go`), its
command is gone so it never will, there is no sweep or admin
member-removal RPC, and the store persists across daemon restarts
(`$XDG_STATE_HOME/crabswarm/chat.db`). The only workaround is an
explicit `--name`, which defeats the label-derived-naming feature.

Decided policy: on a duplicate-name join, the server checks whether the
existing holder of that name is still around — resolve its stored token
through the team-info provider — and when the previous member is gone
(token no longer resolves), frees the name and admits the joiner;
a collision with a live member stays a clear `AlreadyExists` rejection.

Regression test to add with the fix: `Service.Join` with a
provider-derived name already taken in the same team by a token the
provider no longer knows.

# Untested guarantee: `/` in a provider-derived member name (2026-08-31)

The comment in `crabswarm/chat/resolver/cmdman.go` (label values used
verbatim) leans on join-time rejection of `/` in names, but only
`req.Name` is test-driven through that path
(`crabswarm/chat/service_member_test.go`); no test sends a
`resolver.TeamInfo.Name` containing `/` through `Service.Join`. A future
"cmdman is trusted, skip validation" refactor could silently break the
documented guarantee.

Follow-up: add a service test driving a `/`-carrying provider-derived
name through `Service.Join` and asserting the `InvalidArgument`
rejection.

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

# Drop the unused protoc plugin binaries from the dev environment (2026-09-02)

The repo no longer invokes any PATH-installed protoc plugin:
`api/buf.gen.yaml` runs the Go plugins through `go tool` (pinned by
`go.mod` tool directives) and `protoc-gen-es` through
`web/node_modules/.bin/protoc-gen-es` (pinned by `web/package.json`).
The nix profile still provides `protoc-gen-go`, `protoc-gen-go-grpc`,
`protoc-gen-connect-go`, and `protoc-gen-es` on PATH — and the PATH
`protoc-gen-es` is 2.12.0 while the pinned one is 2.12.1, a real drift
trap if anything falls back to PATH. The dev-environment definition
lives outside this repository.

Follow-up: remove the four plugin packages from the dev-environment
definition.

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

# Modernize lint findings in crabswarm/hook tests (2026-09-02)

A repo-wide `golangci-lint run` surfaces pre-existing `modernize`
findings in untouched files: `crabswarm/hook/audit_test.go:90` and
`:103` want `errors.As` replaced with `errors.AsType`.

Follow-up: apply the two mechanical replacements.

# Document the chat surface the crabswarm-chat apm package now ships (2026-09-02)

`apm-package/crabswarm-chat/README.md` and
`apm-package/crabswarm-chat/.apm/skills/crabswarm-chat/SKILL.md` teach
only the member CLI verbs. They do not mention the non-destructive
`crabswarm chat history [--limit N]` transcript verb, the bridge's four
MCP tools, or the `crabswarm://chat/members` and
`crabswarm://chat/history` resources the package now ships.

Follow-up: bring both docs up to the shipped surface so agents wired
through the package can discover it.

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

# e2e read test for the MCP resources (2026-09-02)

Neither `crabswarm://chat/members` nor `crabswarm://chat/history` has a
real-process read test; both are covered by in-process unit tests only
(`crabswarm/chat/mcpserver/resources_test.go`).

Follow-up: one e2e that starts `crabswarm chat mcp` over stdio against a
live daemon and reads both resources would close the gap for both.

# Confirm intent: hook path violations now ride the always-JSON output (2026-09-02)

`crabswarm/hook/path/windows.go` calls `handler.Block`, so its violation
report moved with the hook-exec output-template change from exit 2 +
stderr to exit 0 + JSON on stdout, although that feature was scoped to
`hook exec` only. No test, doc, or consumer depends on exit 2 and there
is no deployed consumer, so nothing is broken — but the semantic shift
for `hook path` was never explicitly decided, and the package has no
process-level test at all.

Follow-up: confirm the always-JSON wire form is wanted for `hook path`
too, and pin it with a process-level test either way.

# Admin TUI refuses a room that has history but no members (2026-09-02)

`openRoom` (`crabswarm/chat/cli/tui/tui.go`) decides room existence from
the roster listing, so once every member leaves,
`chat admin tui --room R` errors while `chat admin log R` still serves
the retained transcript.

Follow-up: decide existence from the log as well — needs a read the
admin History RPC does not offer today (an "any rows for this room?"
probe or listing rooms present in `room_log`).

# Admin verb spelling: tui takes --room, the others take a positional (2026-09-02)

Every other room-scoped admin verb (`log`, `send`, ...) takes the room
as its first positional argument; `chat admin tui` requires a `--room`
flag because the plan fixed that spelling. The group is internally
inconsistent.

Follow-up: pick one convention and align the verbs.

# Shell completion for the admin TUI's room argument (2026-09-02)

The admin can already enumerate rooms and the completion precedent
exists (`completeChatMembers`, `cmd/crabswarm/commands/zz_chat.go`), but
`chat admin tui --room` completes nothing.

Follow-up: wire room-name completion for the flag (and for the other
admin verbs' room positional while at it).

# Swap the hand-rolled TUI e2e scraping for teatest once it tags a release (2026-09-02)

`charm.land/x/exp/teatest/v2` resolves as a module path but has no
tagged version, so `e2e/crabswarm/chat_tui_test.go` strips ANSI from
accumulated program output itself.

Follow-up: adopt teatest when it tags a release.

# WatchRoom upgrade path for the admin TUI tail poll (2026-09-02)

The TUI tails the room log by cursor poll (~1s) against
`ChatAdminService.History`. The daemon already serves server-streaming
`ChatService.WatchRoom` on the member plane; once an admin-plane stream
(or a message-appended event feeding one) exists, the poll can be
replaced without changing the `tui` package's `LogReader` consumers.

Follow-up: revisit after the MessageAppended producer decision above.

# Admin TUI conversation re-render is whole-string per poll (2026-09-02)

`layout()` (`crabswarm/chat/cli/tui/model.go`) re-renders the entire
conversation string (bounded at 2000 entries) on every poll that brings
entries. Fine at terminal scale.

Follow-up: make it incremental only if the screen ever feels heavy.
Related: `model.go` is ~395 LoC against the repo's 300-LoC preference;
splitting it is a natural companion cleanup.

# Stale claim: an apm bundle carries the same wiring as a source install (2026-09-02)

`apm-package/crabswarm-chat`'s README claims a bundle install carries
the same wiring as an install from source, but the MCP-server
registration renders into `.mcp.json` / `.codex/config.toml` via
apm.yml, and transitive installs need `--trust-transitive-mcp`; the
claim looks stale (pre-existing, noted during the MCP-server run).

Follow-up: re-verify a bundle install end to end and fix the README.

# Codex runtime never verified to start the chat MCP bridge (2026-09-02)

apm writes `[mcp_servers.crabswarm-chat]` into `.codex/config.toml`,
but nobody has confirmed codex actually starts the bridge.

Follow-up: verify once against a real codex runtime.

# Release note: old dev chat DBs need recreating (2026-09-02)

Existing dev chat databases lack the `members.state_reported_at` NOT
NULL column (no migration by design; the repo has no deployment
back-compat obligation). They must be deleted/recreated.

Follow-up: a release-note line when anything ships.

# godoc nit: [nudgeable] link in SendKeys doc points at an unexported func (2026-09-02)

The `[nudgeable]` doc link in the exported `SendKeys` comment
(`crabswarm/chat/notify`) points at an unexported func and renders as
plain text.

Follow-up: reword or export what the link needs.

# Review-noted chat daemon test gaps (2026-09-02)

Deferred, not defects, from the MCP-server run's review: nothing pins
the magnitude of the 10-minute staleness threshold (fixtures are
relative to the const); the daemon's `ChainStreamInterceptor` wiring in
`crabswarm/server/server.go` is never exercised by a test; the bounded
`GracefulStop` path has no test; no test asserts the timestamp
`ReportState` writes (a zero-time regression would make every busy
member instantly stale); the hook e2e exhaustiveness guard keys on
event name only, so re-adding a matcher-less catch-all `Notification`
group would fail nothing.

Follow-up: pin whichever of these bite first.

# README notification-type list is incomplete (2026-09-02)

`apm-package/crabswarm-chat`'s README enumerates three values for
"every other notification type"; the official hooks docs list ~11
(e.g. `agent_needs_input`, `agent_completed`).

Follow-up: sync the list or reword to avoid enumerating.

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

# Flagless chat members are never reaped and hold their names forever (2026-09-02)

`checkLiveness` (`crabswarm/chat/service.go`) is deliberately
agent-only, so a `KindHuman` member is never reaped even when the
provider forgets its token — it drops out of the cmdman status display
but holds its name against colliding joiners indefinitely. Intended for
admin-registered humans; an explicit `chat leave` is the current answer.

Follow-up: revisit if plain-shell membership becomes common.

# Unnamed flagless joiners land in the roster as agent-<token8> (2026-09-02)

`defaultName` (`crabswarm/chat/service.go`) always derives
`agent-<first 8 of token>` for an unnamed joiner, so a member that
declared it is not an agent still carries an `agent-` name. Cosmetic,
but the name lies about the kind. Changing it touches the e2e pin of
`alpha/agent-tok-bare` in `e2e/crabswarm/chat_test.go`.

Follow-up: derive a kind-neutral default (or a kind-matched prefix).
