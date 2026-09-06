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
  repo's own hooks/.
- Mermaid in the backlog is guarded at Stop. `crabswarm issues lint` runs
  mermaid-lint over every `` ```mermaid `` fence in bead text — description,
  design, acceptance criteria, notes and comments — and reports one line per
  refused diagram (`<issue-id> <field>[#<comment-n>]:<line>:<col>: <message>`),
  exiting 1 when anything was refused. `hooks/issues-mermaid-lint` (a local
  `path:` dependency in apm.yml) wires it as the Stop hook
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
├── apm-package
│   └── crabswarm-chat  APM package this repo publishes: wires a harness (claude / codex) into its chat room — hooks + skill + `[mcp_servers.crabswarm-chat]` (`crabswarm chat mcp`).
├── bin             git-ignore'd bin dir.
├── cmd
│   └── crabswarm
│       └── commands  One file per cobra subcommand, named by path: chat_admin_tui.go = `crabswarm chat admin tui`. zz_*.go hold a command family's shared helpers (config resolution, dialers, shell completions); every other name is reserved for a subcommand. Wiring only — logic lives under crabswarm/.
├── crabswarm       crabswarm implementation (package per top-level subcommand).
│   ├── config.go   Layered config: DefaultConfig < file < env; Config / PartialConfig; sub-configs are chat.Config, preview.Config, exec.Config.
│   ├── server      The daemon behind `crabswarm serve`: one gRPC server on Config.Sock hosting hook audit + chat services.
│   ├── chat        Chat broker. service*.go = member plane RPCs (Join/Send/Read/WatchRoom/ReportState); admin*.go = age-authenticated admin plane;
│   │   │           store.go/member.go/inbox.go/history.go = SQLite store; delivery.go + notify/ = fan-out and keystroke nudges via cmdman;
│   │   │           resolver/ = cmdman team-info provider (token -> room/team/name); interceptor.go = per-RPC auth + lazy reaping.
│   │   ├── cli       Client side: token resolution (token.go), member verbs, admin verbs, and cli/tui = the admin TUI (bubbletea).
│   │   ├── mcpserver `crabswarm chat mcp`: stdio MCP bridge, one instance per agent; auto-joins on startup; tools + resources over the member plane.
│   │   ├── notify    Nudging a harness by typing into its cmdman-tracked terminal (SendKeys, gated on reported state).
│   │   └── internal  cmdman client, sqlc-generated db/, schema/ddl (schema.sql + room_log.sql) and schema/queries.
│   ├── hook        Claude Code / Codex hook handlers: exec/ (`hook exec` template runner + its Config), path/, audit.go.
│   ├── issues      Beads backlog reader: every call shells out to `bd ... --json`, nothing opens the database.
│   │   │           client.go/types.go/convert.go = the bd client; sources.go = SourceStore (ID hashes the .beads path, so every
│   │   │           worktree of a repo is one source; display name is the issue prefix); service.go = IssuesService, a Connect
│   │   │           service the preview daemon mounts beside PreviewService; poller.go = 10s `bd list` diff (bd has no change feed).
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
│   └── crabswarm   Process-level tests: TestMain builds the binary once; chat_*.go cover daemon + CLI + hooks + MCP + TUI end to end.
├── hooks           This repo's own APM hook packages, listed as local path deps in apm.yml; issues-mermaid-lint/hooks/hook.json is the Stop hook above.
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
- Runtime state on a dev host: daemon socket `$XDG_RUNTIME_DIR/crabswarm/default.sock`; chat DB `~/.local/state/crabswarm/chat.db` (`crabswarm config` prints the resolved paths). The daemon applies the DDL itself on open — check `crabswarm/chat/store.go` before assuming a migration step exists.
- The chat member identity is the token: `--token`, else `$CRABSWARM_CHAT_TOKEN`, else `$CMDMAN_CMD_ID` (`crabswarm/chat/cli/token.go`). Anything running outside a cmdman-tracked command — a bare shell, an MCP server launched by a harness that strips env — has no token and cannot join.
- `AGENTS.md` / `CLAUDE.md` are generated (git-ignored) from `.apm/instructions/*.md`: edit the source, never the generated files. Never run `apm` (compile, install, ...) inside a worktree such as `main/`; the user regenerates them.
- Backlog / plans: the issue backlog is the beads database (`bd`) under the repo root's `.beads/`, shared by every worktree — one `task` bead per item, labels as tags, `Discussion:`/`Decision:` comments, close reason as conclusion (`bd list`, `bd search <text> --status all`, `bd show <id>`; see the ngplan skill's `reference/beads.md`). Never `bd dolt push` or `bd hooks install`.
- Plans live in beads too, one epic labelled `plan` per plan: `description` carries the idea, `design` the plan, `acceptance_criteria` the success criteria, `notes` (`bd update --append-notes`) the status narrative. Steps are child `task`s labelled `step`, ordered by `blocks` so `bd ready` drives execution; a sub-plan is a child epic labelled `plan`; a handoff item is a `task` with `discovered-from:<id>`. Decisions and open questions are `Decision:` / `Discussion:` comments, the idea gate is metadata `idea_gate_passed=YYYY-MM-DD` (absent means the gate was never confirmed), and a finished plan is `bd close --reason`. `doc/plan/<date>-<NN>-<slug>/` holds the file-authored plans (IDEA/PLAN/STATUS/DECISION, + HANDOFF) as history.

## Implementing functionality

- Edit most relavant files to implement said functionality.
- You may ask back the user to resolve unclear corners.
- Implement e2e tests if any existing test is not covering the case.
