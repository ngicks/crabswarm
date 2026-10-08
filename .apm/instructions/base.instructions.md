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
├── apm-package     APM packages this repo publishes, and consumes back through git dependencies in its own apm.yml. A package that targets Claude Code ships its
│   │               wiring as a skills-directory plugin (`.apm/skills/<name>/` carrying `.claude-plugin/plugin.json` and whichever of `hooks/hooks.json` and `.mcp.json` the package needs),
│   │               which apm copies to `~/.claude/skills/<name>/` and Claude Code loads without touching settings.json. A package that hooks Codex ships `.apm/hooks/codex-hooks.json`, which apm merges.
│   ├── crabswarm-mcp          wires Claude Code alone into its chat room: skill + the stdio `crabswarm mcp` server, with no hooks and no `.apm/hooks/`.
│   │                          A mention reaches the member through the fakechat channel or the daemon's typed nudge, and its state comes off the agents listing its own server polls.
│   │                          env_vars/`.mcp.json` env forward CMDMAN_CMD_ID, CRABSWARM_CHAT_TOKEN, XDG_RUNTIME_DIR and the channel variables CRABSWARM_CLAUDE_CHANNEL and FAKECHAT_PORT.
│   ├── crabswarm-mcp-shared   wires Codex and OpenCode: skill + a remote `crabswarm-mcp` entry at `http://127.0.0.1:47300/mcp`, served by the one `crabswarm mcp --transport http` a compose session runs; no hooks.
│   │                          The skill dir also holds the OpenCode plugins: `opencode.ts` runs in `opencode serve` (names the calling session on every tool call, appends mid-turn messages to tool results),
│   │                          `opencode-tui.ts` runs in each `opencode attach` (holds the notice stream, registers the session the TUI shows, and in a session the TUI owns reports state and prompts notices and what arrived when a turn ends).
│   └── crabswarm-issues-lint  the Stop hook above running `crabswarm hook exec 'crabswarm issues lint'`, plus a skill on fixing a finding.
├── bin             git-ignore'd bin dir.
├── cmd
│   └── crabswarm
│       └── commands  One file per cobra subcommand, named by path: chat_admin_tui.go = `crabswarm chat admin tui`. zz_*.go hold a command family's shared helpers (config resolution, dialers, shell completions); every other name is reserved for a subcommand. Wiring only — logic lives under crabswarm/.
├── crabswarm       crabswarm implementation (package per top-level subcommand).
│   ├── config.go   Layered config: DefaultConfig < file < env; Config / PartialConfig; sub-configs are chat.Config, preview.Config, exec.Config, mcp.Config.
│   ├── server      The daemon behind `crabswarm serve`: one gRPC server on Config.Sock hosting hook audit + chat services.
│   ├── chat        Chat broker. Attendance is one open Attend stream, messages are the room's persistent log, delivery is a per-role read position.
│   │   │           service.go + service_member.go (Attend/ListMembers/ReportState) + service_inbox.go (Send/Read, a skip read included) + service_follow.go (Follow) = member plane RPCs;
│   │   │           admin.go (age challenge) + admin_rooms.go (list/register/delete-room)/admin_send.go/admin_history.go/admin_follow.go = admin plane; convert.go = proto <-> domain;
│   │   │           follow.go = the Follow stream body both planes serve: Followed first, then the stored messages past since when the client sets it, then each appended message, every one once and in seq order. Following attends nothing and moves no read position; the admin stream follows as nobody, so its mentioned_you is always false.
│   │   │           store.go + attendance.go/conversation.go/messages.go/read.go (Read, and Skip behind `chat read --skip`)/window.go/rooms.go = SQLite store; delivery.go + notify/ = fan-out, and the keystroke nudge for members whose server cannot deliver one itself;
│   │   │           resolver/ = cmdman team-info provider (token -> room/team/name); interceptor.go = per-RPC token auth.
│   │   ├── cli       Client side: token resolution (token.go), member verbs, admin verbs, and cli/tui = the admin TUI (bubbletea).
│   │   │             follow.go = FollowInto, which writes a Follow stream as the JSON Lines of `chat follow` and reopens a lost stream resuming after the last seq it wrote.
│   │   │             room.go = the cwd-to-room rule: RoomForDir picks the nearest of a directory and its ancestors among the listed rooms, ResolveRoom lists the rooms once and applies it, OpeningRoom serves `chat admin tui` and RoomToFollow serves `chat follow --admin`.
│   │   ├── nudge     The words a notice is worded with (`[crabswarm chat] new message from ...`, naming chat_read, chat_send and the server both tools come from), shared by the daemon's keystroke nudge and every harness channel so the room speaks one voice; naming the answer keeps a channel plugin's own reply tool from carrying it away.
│   │   ├── notify    The daemon's side of waking a member: SendKeys types a notice into a cmdman-tracked terminal, gated on the member's reported state; ScreenPoller reads the screen of an attending agent whose harness the daemon does not recognise and records the state it shows, since that member has no feed of its own to report on.
│   │   └── internal  cmdman client, sqlc-generated db/, schema/ddl (schema.sql) and schema/queries (rooms.sql, messages.sql, read_positions.sql).
│   ├── mcp         `crabswarm mcp --transport stdio|http`: the MCP server that attends the room for its agents. Each member (member.go) holds its Attend stream, retrying until the daemon answers and reopening it after a restart; a server with no identity token still serves. Tool families register onto it before it runs.
│   │   │           mcp.go = Server, the stdio transport: one process and one member per session (Claude Code). http.go = HTTPServer on --listen at /mcp: one process per compose session and one member per X-Crabswarm-Token header, for Codex and OpenCode. host.go = what both transports share.
│   │   │           threads.go = which of a member's Codex thread sessions a mention goes to (the one last seen), answered to harnessctl as AgentThreads; calls.go = the reserved crabswarm_focus/crabswarm_ping tool calls the server answers itself, and the thread id each Codex tool call carries in its metadata.
│   │   │           opencode.go = the routes an OpenCode TUI plugin registers over (GET /opencode/notices, PUT /opencode/session, POST /opencode/sessions/{id}/read), and the `_crabswarm_session` argument the OpenCode server plugin adds to every tool call: it names the calling OpenCode session, and the call acts as the member that owns that session, the one whose TUI registered it first among the TUIs showing it now.
│   │   │           deliver.go = the other half: it watches the room's feed and hands a mention to its own harness, so the daemon stops typing at that member; a Claude Code session also gets server instructions naming the fakechat event a notice arrives as and forbidding an answer through fakechat's reply tool.
│   │   ├── chat        The first tool family: chat_send/chat_read/chat_members plus the `crabswarm://chat/members` resource, all over the member plane.
│   │   ├── codexproxy  `crabswarm mcp codex-proxy -- codex --remote unix://...`: runs one Codex TUI behind a private WebSocket relay to the shared app server, stamps the replica's token as the X-Crabswarm-Token http_headers entry on thread/start, thread/resume and thread/fork, and calls crabswarm_focus after a resume. ready.go holds the TUI back until the app server answers account/read (--ready-timeout, default 1m).
│   │   └── (the channels live in pkg/harnessctl; deliver.go asks it which harness the client is and hands it each mention)
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
│   └── crabswarm   Process-level tests: TestMain builds the binary once; chat_*.go cover daemon + CLI + MCP + TUI end to end, the shared layouts included (chat_codex_shared_test.go against codexfake_shared_test.go's fake app server; chat_opencode*_test.go skip without `opencode` on PATH).
│                   mcp_package_test.go and apm_package_test.go pin the apm packages; hook_exec_test.go covers `hook exec` and the hook files the packages ship.
│                   testdata/harness/ holds read-only recordings of what the real harnesses do (each one's MCP `initialize` frame, Codex app-server, TUI, exec-server and codex-proxy sessions, an `opencode serve` with attached TUIs and its plugins, Claude Code screens and its channel launch notes); re-capture rather than hand-edit them.
├── internal        internal helper packages (libver, loggerfactory, templateutil, stdiopipe, cmdsignals, versioninfo,
│                   supervisor = the Supervisor interface + the Start flow behind `util supervised [--name N] [--poll URI] -- CMD...`, cmdman/ implements it).
├── mods            Claude Code mods: each folder is a skills-directory plugin whose `hooks/hooks.json` loads a hooks module (`hooks/register.tsx`) into the session; `apm install -g ngicks/crabswarm/mods/<name>` installs one.
│   │               A mod's code is generic: crabswarm appears only in the `userConfig` defaults of its `.claude-plugin/plugin.json` and in its README and SKILL docs. An override goes under `pluginConfigs["<name>@skills-dir"].options` in the user settings file;
│   │               Claude Code does not read `pluginConfigs` from project settings.
│   ├── report-agent-state  reports the session's state from `claude agents --json` every 2s by running `reportCommand` with `working`, `waiting` or `done` appended (default `crabswarm chat report-state`).
│   └── handle-events       runs a JSON Lines source for the whole session (`source`, default `crabswarm chat follow`), shows its status records on the mod's status line, and submits the `inject: true` message records as one prompt,
│                           one `<prefix> <message>` line each, when the main-loop turn ends. It then acks each record with `ack` (default `crabswarm chat read --skip {seq}`).
├── pkg
│   ├── harnessctl  Reach a coding-agent harness from beside it: Detect picks the CLI off the MCP clientInfo name, and a Detector does so for a server with many sessions; claude.go + claudeagents.go (a POST to the fakechat plugin's loopback server, probed with GET / first, and the state feed built on `claude agents --json`),
│   │               codex*.go (the app server the session is hosted by — delivery and the state feed both; codexhub.go holds one connection per app server for every agent on it, and threads.go's ThreadScoper/AgentThreads scope a shared app server to each agent's threads),
│   │               opencode.go (OpenCodeNotices, the notice streams a TUI plugin opens, which crabswarm/mcp attaches to that TUI's member; Detect hands out none, so an OpenCode MCP session alone is a terminal one).
│   │               A harness launched without its variable has no channel and is a terminal one; a Prober's channel is asked whether it is there before each attendance; a harness that also implements StateSource is watched for the whole session. Knows nothing about chat; crabswarm/mcp composes it with the room.
│   ├── claudehook  helper for claude code hook. Same code can be resued for codex.
│   ├── filetype    filetype detection config consumed by hook exec.
│   └── util        Behind `crabswarm util`, independent of the rest of crabswarm: poll/ (`util poll wait URI` probes a unix socket,
│                   file:// path or http(s) URL; --start-period delays the first probe, --interval and --retries carry their
│                   Docker-healthcheck meaning; --method (default GET), --header, --payload/--file and --status (default
│                   2xx) shape an HTTP probe, and `util supervised` takes them as --poll-*).
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
- Runtime state on a dev host: daemon socket `$XDG_RUNTIME_DIR/crabswarm/host/default.sock`, and where that variable is unset `/run/user/<uid>/crabswarm/host/default.sock` when the directory is there, else `/tmp/crabswarm/host/default.sock`; the sockets one compose session shares (such as the Codex app server's) go under `<runtime dir>/crabswarm/session/`, which a container mounts writable beside a read-only `crabswarm/host/` (apm-package/crabswarm-mcp-shared/README.md); chat DB `~/.local/state/crabswarm/chat.db` (`crabswarm config` prints the resolved paths). The daemon applies the DDL itself on open (`crabswarm/chat/store.go`), creating only the tables that are missing — there is no migration, so a chat.db written before the log-and-read-position layout still starts a daemon and then fails every send and read; remove it before the first start. `chat.history_limit` caps each room's log (0 = the default 1000, negative = no cap).
- The chat member identity is the token: `--token`, else `$CRABSWARM_CHAT_TOKEN`, else `$CMDMAN_CMD_ID` (`crabswarm/chat/cli/token.go`). Anything running outside a cmdman-tracked command — a bare shell, an MCP server launched by a harness that strips env — resolves no token and is refused by every member verb. Attending is not a verb either: an agent's attendance is the open stream `crabswarm mcp` holds, a person's comes from `crabswarm chat admin register`, which mints a token that works from a plain shell and keeps them in attendance until the daemon restarts.
- How a mention wakes an agent is declared at attendance, and `crabswarm chat members` prints it: `<address> <kind> <state> <harness> <nudge>`. A `native` member is one its own `crabswarm mcp` hands the notice to, and that server delivers only while the member reports done. A Claude Code session is native when its stdio server finds `CRABSWARM_CLAUDE_CHANNEL=1` plus `FAKECHAT_PORT`, for a session launched as `FAKECHAT_PORT=<port> CRABSWARM_CLAUDE_CHANNEL=1 claude --channels plugin:fakechat@claude-plugins-official` (the official plugin's loopback server takes the notice on `POST /upload`; unset and empty both mean its default 8787, so a launcher sets a real port or nothing, one per session, and never the empty string). A Codex replica is native when the shared `crabswarm mcp --transport http` runs with `CRABSWARM_CODEX_APP_SERVER=unix:///path.sock` naming the app server that hosts it: the server starts a turn on the thread the replica's TUI (behind `crabswarm mcp codex-proxy`) was last seen in, and nothing reaches a Codex turn already running. An OpenCode replica is native while its TUI plugin holds `GET /opencode/notices` open on the shared server; the plugin needs `CRABSWARM_OPENCODE_SERVER` and the replica's token in the TUI's environment, and prompts each notice into the session the TUI shows when the TUI owns it (registered it first among the TUIs showing it now); a notice arriving while the TUI shows another replica's session stays unread. Everything else is `terminal`, which the daemon types into through cmdman, and a typed nudge cannot see an unsent composer draft.
- A declared channel gates the attendance it belongs to: `crabswarm mcp` probes the fakechat port with `GET /` before every attempt, and while nothing answers (plugin dead, port taken, flag forgotten) the member does not attend at all — it logs why, retries on the attendance backoff, and attends as `native` once fakechat answers, so a launch gone wrong shows up as a member missing from `crabswarm chat members`. A Claude Code session launched without `CRABSWARM_CLAUDE_CHANNEL=1` is probed for nothing and attends as a terminal member.
- Where a member's state comes from differs by harness, and the Claude Code plugin reports none through hooks: Claude Code reports from the agents listing (`claude agents --json`) that its own `crabswarm mcp` polls every 2s, matching its session by the pid of its parent Claude Code (then by `CLAUDE_CODE_SESSION_ID`, which `/clear` and `/resume` leave stale), skipping pid-less job records, and reporting on change. The report-agent-state mod reads the same listing from inside the Claude Code process, so it reads the session's own PID namespace and config directory wherever the MCP server runs, and reports through `crabswarm chat report-state`. Codex reports on its app-server feed, which the shared server follows per replica on the threads that replica's sessions serve; OpenCode's TUI plugin runs `crabswarm chat report-state` off the session and permission events of the session its TUI shows and owns, its end-of-turn `crabswarm chat read --quiet --done-when-empty` reports done when it hands nothing over, and it reports done when its TUI moves away from the session it owned. A bridge watching a feed also re-sends the last state that feed reported every 10s, so a restarted daemon hears it again. The daemon's screen poller is the fallback for a harness with no feed of its own, so it polls only members whose harness is `other` (`chat.screen_poll_interval` / `CRABSWARM_CHAT_SCREEN_POLL_INTERVAL`, 0 = 3s, negative = off).
- A launcher that sandboxes Claude Code sessions must give each one a private `sessions/` and `cc-socks/` directory. Claude Code keeps its session registry at `<config home>/sessions/<pid>.json`, keyed by the pid inside the session's own PID namespace, and its messaging socket at `<runtime dir>/cc-socks/<pid>.sock`. Sessions in different PID namespaces sharing one config home overwrite each other's record, and the listing then omits them. A Claude Code member whose session the listing omits keeps the state it attended with.
- `crabswarm chat follow` streams the token's room to stdout as JSON Lines for another program, such as the handle-events mod, to read. Each line is one object with a `type` and a `message`: `{"type":"status","message":"following <room>"}` opens every stream, and `reconnecting to the daemon` is a status written once per outage. A message line carries `"type":"message"`, `message` (`<sender>: <text>`, where `<sender>` is the sender's address: `<team>/<name>`, or the name alone for a sender with no team, such as the operator's `admin`), `inject` (the message names the follower or everyone, and the follower did not send it) and every field of the Message proto by its proto name, with `seq` as a JSON number and `mentioned_you` repeating `inject`. The stream starts live. A lost stream reopens after a pause that starts at 1s and doubles up to 30s, resuming after the last seq it printed, so each message is printed once. A room deleted under the follow, or a replaced chat.db, numbers from one again; the follow prints `following <room>` again and goes on with the restarted room's messages. Following is not attendance: it puts nobody in the room and moves no read position. `--admin` follows as the host operator, and its `inject` is always false.
- `chat follow --admin` without `--room` and `chat admin tui` without `--room` both pick the room of the current directory: the nearest of that directory and its ancestors the daemon lists (`crabswarm/chat/cli/room.go`). They differ when no room matches. `chat follow --admin` follows the directory itself, since agents may start there later, and picks the room only once the daemon answers, waiting for it as for a lost stream; `chat admin tui` opens the first room the daemon lists and says on its status bar that the directory matched no room.
- `crabswarm chat read --skip SEQ` marks messages read without printing them, for a client that already showed them some other way; handle-events acks with it by default. It moves the read position through SEQ inclusive and prints how many unread mentions are left (`--quiet` drops that line). The read position is one seq per role in a room, so a skip also marks every earlier unread mention read, shown or not. A SEQ at or before the position moves nothing, a SEQ past the room's newest message stops there, and `--skip` takes none of the filter flags or `--done-when-empty`.
- `AGENTS.md` / `CLAUDE.md` are generated (git-ignored) from `.apm/instructions/*.md`: edit the source, never the generated files. Never run `apm` (compile, install, ...) inside a worktree such as `main/`; the user regenerates them.
- Backlog / plans: the issue backlog is the beads database (`bd`) under the repo root's `.beads/`, shared by every worktree — one `task` bead per item, labels as tags, `Discussion:`/`Decision:` comments, close reason as conclusion (`bd list`, `bd search <text> --status all`, `bd show <id>`; see the ngplan skill's `reference/beads.md`). Never `bd dolt push` or `bd hooks install`.
- Plans live in beads too, one epic labelled `plan` per plan: `description` carries the idea, `design` the plan, `acceptance_criteria` the success criteria, `notes` (`bd update --append-notes`) the status narrative. Steps are child `task`s labelled `step`, ordered by `blocks` so `bd ready` drives execution; a sub-plan is a child epic labelled `plan`; a handoff item is a `task` with `discovered-from:<id>`. Decisions and open questions are `Decision:` / `Discussion:` comments, the idea gate is metadata `idea_gate_passed=YYYY-MM-DD` (absent means the gate was never confirmed), and a finished plan is `bd close --reason`. `doc/plan/<date>-<NN>-<slug>/` holds the file-authored plans (IDEA/PLAN/STATUS/DECISION, + HANDOFF) as history.
- Commit messages follow `doc/COMMIT_CONVENTION.md`: one emoji prefix, a scope from its Scopes table, and a short imperative subject.

## Implementing functionality

- Edit most relavant files to implement said functionality.
- You may ask back the user to resolve unclear corners.
- Implement e2e tests if any existing test is not covering the case.
