---
description: "Basic instructions for the project"
applyTo: "*"
---

### General

Tools to swarm claude(, codex and others!)

### Tech stack

- Go
  - `github.com/spf13/cobra` for subcommands
  - Using gRPC and protobuf for communications
- TypeScript (`web/` frontend: Preact SPA for `crabswarm preview`)
  - connect-web + buf-generated types from the proto schema

### Implementing functionality

- Implement e2e tests if any existing test is not covering the case.
- Don't think too much about backward compatibility, since the app was never actually deployed
- Harness hook wiring (Claude Code / Codex hook config, in-repo or in an
  apm package) is built on `crabswarm hook exec` — a command template plus
  an output template (`blockDecision`, `context`, ...; a template that
  records nothing means plain allow, even on failure). Never standalone
  shell scripts or jq unless a template genuinely cannot express it; if a
  two-step behavior doesn't fit, extend the CLI with a flag instead. Model:
  the hooks packages in apm_modules/ngicks/agents-package/hooks/ and this
  repo's own apm-package/.
- Mermaid in the backlog is guarded at Stop. `crabswarm issues lint` runs
  mermaid-lint over every `` ```mermaid `` fence in bead text — description,
  design, acceptance criteria, notes and comments — and reports one line per
  refused diagram (`<issue-id> <field>[#<comment-n>]:<line>:<col>: <message>`),
  exiting 1 when anything was refused. `apm-package/crabswarm-issues-lint` (a
  package this repo publishes and consumes back as a git dependency in
  apm.yml) wires it as the Stop hook
  `crabswarm hook exec 'crabswarm issues lint'`, so a broken diagram in a bead
  blocks the turn until the bead text is fixed. Flags: `--all` (closed issues
  too), `--limit N` (most recently updated first), `--json`, `-C/--dir` — bd
  and mermaid-lint both run in that directory, so the repository's own
  mermaid-lint configuration governs its issue text.
- Before reporting "tool X cannot do Y" (apm, buf, cmdman, ...), read the
  tool's --help or source and try a second layout/config; one failing
  attempt is an observation, not a limitation.

```
.
├── AGENTS.md
├── api             API definition and generated code / related type definitions. proto schema basically sit here.
│   ├── schema/proto/ngicks/crabswarm/{chat,hook,issues,preview,sdktypes}/v1   *.proto (edit these)
│   ├── gen/proto                                                              buf output (Go + connect); TS lands in web/src/api/gen
│   └── buf.gen.yaml, generate.go                                              `go generate ./api/...` regenerates both sides
├── apm-package     APM packages this repo publishes, and consumes back through git dependencies in its own apm.yml. Each ships its Claude Code
│   │               wiring as a skills-directory plugin (`.apm/skills/<name>/` carrying `.claude-plugin/plugin.json`, `hooks/hooks.json`, `.mcp.json`),
│   │               which apm copies to `~/.claude/skills/<name>/` and Claude Code loads without touching settings.json; Codex gets `.apm/hooks/codex-hooks.json` merged.
│   ├── crabswarm-mcp          wires a harness (claude / codex / opencode) into its chat room — hooks + skill + the `crabswarm mcp` server; `opencode.ts` in the skill dir is the OpenCode plugin.
│   │                          The hooks only deliver messages now (the plugin's `hooks/hooks.json` also reports Claude Code's state; `codex-hooks.json` does not), and env_vars/`.mcp.json` env forward
│   │                          CRABSWARM_CLAUDE_CHANNEL and CRABSWARM_CODEX_APP_SERVER, the launch-time variables that let the server hand a mention to its harness itself.
│   └── crabswarm-issues-lint  the Stop hook above running `crabswarm hook exec 'crabswarm issues lint'`, plus a skill on fixing a finding.
├── bin             git-ignore'd bin dir.
├── cmd
│   └── crabswarm
│       └── commands  One file per cobra subcommand, named by path: chat_admin_tui.go = `crabswarm chat admin tui`. zz_*.go hold a command family's shared helpers (config resolution, dialers, shell completions); every other name is reserved for a subcommand. Wiring only — logic lives under crabswarm/.
├── crabswarm       crabswarm implementation (package per top-level subcommand).
│   ├── config.go   Layered config: DefaultConfig < file < env; Config / PartialConfig; sub-configs are chat.Config, preview.Config, exec.Config.
│   ├── server      The daemon behind `crabswarm serve`: one gRPC server on Config.Sock hosting hook audit + chat services.
│   ├── chat        Chat broker. Attendance is one open Attend stream, messages are the room's persistent log, delivery is a per-role read position.
│   │   │           service.go + service_member.go (Attend/ListMembers/ReportState) + service_inbox.go (Send/Read) = member plane RPCs;
│   │   │           admin.go (age challenge) + admin_rooms.go (list/register/delete-room)/admin_send.go/admin_history.go = admin plane; convert.go = proto <-> domain;
│   │   │           store.go + attendance.go/conversation.go/messages.go/read.go/window.go/rooms.go = SQLite store; delivery.go + notify/ = fan-out, and the keystroke nudge for members whose server cannot deliver one itself;
│   │   │           resolver/ = cmdman team-info provider (token -> room/team/name); interceptor.go = per-RPC token auth.
│   │   ├── cli       Client side: token resolution (token.go), member verbs, admin verbs, and cli/tui = the admin TUI (bubbletea).
│   │   ├── nudge     The words a notice is worded with (`[crabswarm chat] new message from ...`, naming the chat_read tool), shared by the daemon's keystroke nudge and every harness channel so the room speaks one voice.
│   │   ├── notify    The daemon's side of waking a member: SendKeys types a notice into a cmdman-tracked terminal, gated on the member's reported state; ScreenPoller reads every attending Claude Code screen and records the state it shows, since an interrupted turn fires no hook.
│   │   └── internal  cmdman client, sqlc-generated db/, schema/ddl (schema.sql) and schema/queries (rooms.sql, messages.sql, read_positions.sql).
│   ├── mcp         `crabswarm mcp`: the stdio MCP server, one instance per agent; holds the Attend stream for the whole session, retrying until the daemon answers and reopening it after a restart; serves even with no identity token. Tool families register onto it before it runs.
│   │   │           deliver.go = the other half: it watches the room's feed and hands a mention to its own harness, so the daemon stops typing at that member; channel.go wraps the stdio transport so the server can push a notification down the session the SDK is already serving.
│   │   ├── chat      The first tool family: chat_send/chat_read/chat_members plus the `crabswarm://chat/members` resource, all over the member plane.
│   │   └── harness   One file per CLI, chosen by the name the client gives itself in the MCP handshake: claude.go (a channel notification on the session the harness already spawned), codex*.go (the app server the session is hosted by — delivery and the state feed both), opencode.go (a POST to the plugin's loopback relay).
│   │                 A harness launched without its variable has no channel and attends as a terminal member; a harness that also implements StateSource is watched for the whole session and reports its member's state.
│   ├── hook        Claude Code / Codex hook handlers: exec/ (`hook exec` template runner + its Config), path/, audit.go.
│   ├── issues      Beads backlog reader: every call shells out to `bd ... --json`, nothing opens the database.
│   │   │           client.go/types.go/convert.go = the bd client; it collapses identical concurrent invocations into one run and
│   │   │           queues every run of a client behind a one-slot semaphore (embedded dolt admits one process per database).
│   │   │           sources.go = SourceStore (ID hashes the .beads path, so every worktree of a repo is one source; display name
│   │   │           is the issue prefix); service.go = IssuesService, a Connect service the preview daemon mounts beside
│   │   │           PreviewService; poller.go = Poller.Refresh, the shared all-status `bd list` every RPC reads through, diffed
│   │   │           on each run for change events (bd has no change feed) and only scheduled by the 10s ticker; filter.go =
│   │   │           status/label/parent filters, sort, child counts, children and dependency edges, derived from that listing in Go.
│   │   ├── mermaidlint  Sweep/Lint: one temp file per text field or comment, one mermaid-lint run, report mapped back to issue+field+line.
│   │   └── cli          What `issues lint` and the preview registration commands print (RenderFindings, RenderRegistrations, ResolveRegistration).
│   ├── preview     `crabswarm preview` daemon + HTTP API + renderers. `preview DIR` registers DIR as a file root and, when a
│   │               beads database governs it, as an issues source (`--root` / `--issue` register one only; a directory with no
│   │               beads database registers no source silently). `preview list` prints KIND ID NAME PATH (name = root name or
│   │               issue prefix, path = root dir or .beads dir); `preview remove NAME|ID` takes either kind and errors on ambiguity.
│   ├── git         `crabswarm git` helpers (clone, worktree, root, list).
│   ├── statusline  `crabswarm statusline` template rendering.
│   └── cli         Presentation helpers shared by ./cmd (config render, template funcs).
├── doc
│   ├── rules       Some rule guidance sits here.
│   └── plan        The file-authored ngplan directories (<date>-<NN>-<slug>/), kept as history; new plans are beads epics (see below).
├── e2e
│   └── crabswarm   Process-level tests: TestMain builds the binary once; chat_*.go and mcp_package_test.go cover daemon + CLI + hooks + MCP + TUI end to end.
│                   testdata/harness/ holds read-only recordings of what the real harnesses do (each one's MCP `initialize` frame, the Codex app-server session, Claude Code screens and its channel launch notes); re-capture rather than hand-edit them.
├── internal        internal helper packages (libver, loggerfactory, templateutil, stdiopipe, cmdsignals, versioninfo).
├── pkg
│   ├── claudehook  helper for claude code hook. Same code can be resued for codex.
│   └── filetype    filetype detection config consumed by hook exec.
└── web             Preact SPA for `crabswarm preview`, embedded via go:embed as seekable-zstd tar (dist.tar.zst committed).
    └── src
        ├── app.tsx        The shell and the routes: /roots/{rootId}/{path...} is the file browser, /issues/{sourceId} the Issues tab.
        ├── pages          preview/ = the file browser and its own components; issues/ = the Issues tab; not-found.tsx.
        ├── api            client.ts (Connect transport), events.ts (startWatchEvents and startWatchIssues, one stream each), preview.ts + issues.ts (query options and hooks), query.ts (the search bar's query language), gen/ (buf output).
        ├── components     Shared chrome (Layout, Header, Lightbox, ThemeToggle) and ui/ primitives.
        ├── signals        Client state (navigation, preferences).
        └── lib            Focused non-UI helpers (paths.ts).
```

### Navigating quickly

- Checkout layout: `.bare` is the bare repo; `main/` is the main-branch worktree and every path above is relative to it. Other worktrees (e.g. `web/`) sit beside `main/`. Run `git`, `go`, and `apm` from inside a worktree.
- Command -> code: `crabswarm a b c` is `cmd/crabswarm/commands/a_b_c.go`, whose RunE calls into `crabswarm/<a>/`. Read the command file first for flags, then follow the call.
- Runtime state on a dev host: daemon socket `$XDG_RUNTIME_DIR/crabswarm/default.sock`, and where that variable is unset `/run/user/<uid>/crabswarm/default.sock` when the directory is there, else `/tmp/crabswarm/default.sock`; chat DB `~/.local/state/crabswarm/chat.db` (`crabswarm config` prints the resolved paths). The daemon applies the DDL itself on open (`crabswarm/chat/store.go`), creating only the tables that are missing — there is no migration, so a chat.db written before the log-and-read-position layout still starts a daemon and then fails every send and read; remove it before the first start. `chat.history_limit` caps each room's log (0 = the default 1000, negative = no cap).
- The chat member identity is the token: `--token`, else `$CRABSWARM_CHAT_TOKEN`, else `$CMDMAN_CMD_ID` (`crabswarm/chat/cli/token.go`). Anything running outside a cmdman-tracked command — a bare shell, an MCP server launched by a harness that strips env — resolves no token and is refused by every member verb. Attending is not a verb either: an agent's attendance is the open stream `crabswarm mcp` holds, a person's comes from `crabswarm chat admin register`, which mints a token that works from a plain shell and keeps them in attendance until the daemon restarts.
- How a mention wakes an agent is declared at attendance, and `crabswarm chat members` prints it: `<address> <kind> <state> <harness> <nudge>`. A `native` member is one whose own `crabswarm mcp` recognised its harness and found the launch-time variable that harness's channel needs — `CRABSWARM_CLAUDE_CHANNEL=1` for a Claude Code session registered with `--dangerously-load-development-channels`, `CRABSWARM_CODEX_APP_SERVER=unix:///path.sock` for a Codex hosted by an app server, `CRABSWARM_OPENCODE_RELAY` which the OpenCode plugin sets for itself — and that server hands the notice to the harness. Everything else is `terminal`, which the daemon types into through cmdman, and a typed nudge cannot see an unsent composer draft.
- Where a member's state comes from differs by harness: Codex reports on its app-server feed and wires no report-state hooks, OpenCode reports from the plugin's in-process events, and Claude Code reports through hooks with the daemon's screen poller correcting them (`chat.screen_poll_interval` / `CRABSWARM_CHAT_SCREEN_POLL_INTERVAL`, 0 = 3s, negative = off), because an interrupted Claude Code turn fires no Stop hook.
- `AGENTS.md` / `CLAUDE.md` are generated (git-ignored) from `.apm/instructions/*.md`: edit the source, never the generated files. Never run `apm` (compile, install, ...) inside a worktree such as `main/`; the user regenerates them.
- Backlog / plans: the issue backlog is the beads database (`bd`) under the repo root's `.beads/`, shared by every worktree — one `task` bead per item, labels as tags, `Discussion:`/`Decision:` comments, close reason as conclusion (`bd list`, `bd search <text> --status all`, `bd show <id>`; see the ngplan skill's `reference/beads.md`). Never `bd dolt push` or `bd hooks install`.
- Plans live in beads too, one epic labelled `plan` per plan: `description` carries the idea, `design` the plan, `acceptance_criteria` the success criteria, `notes` (`bd update --append-notes`) the status narrative. Steps are child `task`s labelled `step`, ordered by `blocks` so `bd ready` drives execution; a sub-plan is a child epic labelled `plan`; a handoff item is a `task` with `discovered-from:<id>`. Decisions and open questions are `Decision:` / `Discussion:` comments, the idea gate is metadata `idea_gate_passed=YYYY-MM-DD` (absent means the gate was never confirmed), and a finished plan is `bd close --reason`. `doc/plan/<date>-<NN>-<slug>/` holds the file-authored plans (IDEA/PLAN/STATUS/DECISION, + HANDOFF) as history.

## Implementing functionality

- Edit most relavant files to implement said functionality.
- You may ask back the user to resolve unclear corners.
- Implement e2e tests if any existing test is not covering the case.
