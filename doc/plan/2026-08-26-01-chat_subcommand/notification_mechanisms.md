# Research notes — message-arrival notification per harness (Q9)

Verified 2026-08-26 against official docs/repos. Feeds the Q9/D9 decision.

## Codex CLI (github.com/openai/codex)

- `notify` config (`~/.codex/config.toml`) is **outbound-only** — Codex
  invokes an external program on events (`agent-turn-complete`); nothing is
  fed back. Not an inbound channel.
- **Hooks** (stable since ~v0.124.0, Apr 2026; `~/.codex/hooks.json` or
  `<project>/.codex/hooks.json`): JSON stdin→stdout; events include
  `SessionStart`, `UserPromptSubmit`, `PostToolUse`, `Stop`, ...
  - `additionalContext` injection works for `SessionStart`,
    `UserPromptSubmit`, `PostToolUse` (not `PreToolUse`).
  - `Stop` hook returning `{"decision":"block","reason":"..."}` prevents
    turn end and injects the reason as a continuation prompt
    (`stop_hook_active` guards loops). **Turn-boundary only — an idle
    session runs no hooks.**
- `codex app-server` (JSON-RPC over stdio/socket): `turn/start`,
  `turn/steer`, `thread/inject_items` — true injection, **but only into
  sessions the app-server process itself owns**; no cross-process attach to
  a foreign TUI session (openai/codex#25914).
- **Mid-idle push into a live TUI: only tmux/PTY keystroke injection.**

## OpenCode (github.com/anomalyco/opencode, opencode.ai)

- Every TUI run starts an HTTP server (`opencode --hostname 127.0.0.1
  --port 4096` to fix the address; OpenAPI at `/doc`).
  - `POST /session/:id/message` (send + wait), `prompt_async`,
    `POST /tui/append-prompt` / `submit-prompt` — inject into the running
    TUI's prompt box and submit. **Genuine mid-idle push works.**
  - `GET /event` — SSE bus; `session.idle` fires at turn end.
- **Plugins** receive the SDK `client` bound to the running server — a
  plugin can listen externally (socket/watch/timer) and deliver via
  `client.session.prompt(...)` / `client.tui.appendPrompt(...)` at any
  time, including mid-idle.
- No "block turn end" semantic — unnecessary, since idle injection works.

Sources: Codex config reference (learn.chatgpt.com/docs/config-file/
config-reference), codex docs/config.md, app-server README,
openai/codex#25914; OpenCode /docs/server, /docs/sdk, /docs/plugins.

## Claude Code

- **Cross-session messaging socket — true mid-idle push.** An external
  process (same OS user) posts JSON to `$CLAUDE_CODE_MESSAGING_SOCKET`
  (UDS on Linux/WSL2; named pipe on Windows; v2.1.224+). The path is
  **per-session and dynamic**, not a fixed location: discover it via the
  env var (exported to hooks and Bash commands, incl. SessionStart) or
  the `/status` "Peer address" row. Observed pattern on WSL2:
  `$XDG_RUNTIME_DIR/cc-socks/<n>.sock` — an implementation detail; the
  env var is the contract. A per-session `$CLAUDE_CODE_MESSAGING_TOKEN`
  exists too (auth line optional on Linux/WSL2, required on native
  Windows). **Design consequence for step 6 (D13):** the SessionStart
  hook that runs `chat join` sees the env var, so JoinRequest should
  carry the member's notify endpoint (container-side socket path, which
  devenv maps to a host path) instead of the daemon guessing paths from
  config.
  Ref: https://code.claude.com/docs/en/cross-session-messaging
  ("The session's inbox socket"),
  https://code.claude.com/docs/en/env-vars. An idle session
  starts a new turn with the message. Caveats: optional auth token line;
  ~50-message queue; `--bare` sessions don't bind the socket; disabled
  when `DISABLE_TELEMETRY`/`DO_NOT_TRACK` set; not on Bedrock/Vertex;
  message is marked "from another session"; inbound rules can hold/refuse.
- **Stop hook** returning `{"decision":"block","reason":"..."}` forces the
  turn to continue with the reason injected (`stop_hook_active` guard,
  ~8-block cap). **Turn-boundary only** — never fires while idle.
- **`additionalContext` hooks** (`UserPromptSubmit`/`PreToolUse`/
  `PostToolUse`) inject context into an already-running turn; cannot wake
  an idle session.
- **Async hooks (`asyncRewake: true`)** — a background hook completing
  with exit 2 wakes the model into a continuation.
- **Channels** (research preview) — push via MCP channel plugins, needs
  `--channels` at launch and Anthropic auth.
- **Keystroke injection (tmux send-keys)** still works as fallback;
  brittle, obsoleted by the socket.
- `claude -p --resume <id>` does NOT inject into a running session (new
  process).

Sources: code.claude.com/docs/en/cross-session-messaging.md, channels.md,
hooks-guide.md, hooks.md, scheduled-tasks.md.

## Go client availability (verified 2026-08-27, for step 6)

- **Claude messaging socket:** no official client in any language; the
  message frame is undocumented (only the transport and the optional
  `{"type":"auth","token":...}` first line are official). Sole community
  Go client: `github.com/PeterSR/claude-code-socket-transport` (ccsock)
  v0.1.1, single-author/0-star, reverse-engineered — use its README as
  protocol reference (NDJSON frames:
  `{"msgV":1,"msg_id":...,"type":"user","message":{...},"priority":...}`
  + `peer_message_status` receipts), not as a dependency.
- **OpenCode:** official Go SDK `github.com/sst/opencode-sdk-go`
  (Stainless-generated, repo now anomalyco/opencode-sdk-go) exists but is
  dormant — v0.19.2, no commits since 2025-12-18 while the server API
  evolved. Official docs point non-TS languages at the OpenAPI 3.1 spec
  the server serves at `/doc`. (opencode.ai/docs/go/ is an unrelated
  subscription product.)
- **Step 6 consequence:** hand-roll both adapters (a `net.Dial("unix")`
  + NDJSON write; a couple of `net/http` calls). The unofficial Claude
  frame reinforces `send-keys` as the degradation path (D13 risk note).
- **No SDK anywhere for the Claude socket (verified 2026-08-27):** the
  Claude Agent SDK (`@anthropic-ai/claude-agent-sdk`) has no
  peer-messaging/inbox API, and there is no `claude peers send`-style
  CLI. Only the auth line is officially specified.
- **Channels — corrected understanding (verified 2026-08-27):** a channel
  is an MCP server Claude Code spawns as a **local subprocess over
  stdio**; events never route through Anthropic infra (the webhook
  example is localhost-only). The claude.ai/Console-auth requirement is
  **feature-flag gating of the research preview**, not transport — same
  reason it's off on Bedrock/Vertex. The protocol is **openly specified**
  (code.claude.com/docs/en/channels-reference): declare
  `capabilities.experimental['claude/channel']`, emit
  `notifications/claude/channel` over MCP stdio; reply tools and
  permission relay are also specced. Any language works — a Go binary
  qualifies.
- **Future Claude adapter direction:** `crabswarm` itself can be the
  channel server (e.g. `crabswarm chat channel-serve` speaking MCP
  stdio): launched by Claude Code inside the container, it subscribes to
  the daemon over the already-mounted UDS and pushes arrival events into
  the session, optionally with a reply tool. This inverts D13's boundary
  problem for Claude (channel dials out; no socket to mount out) and
  rests on a documented contract, unlike the messaging-socket frame.
  **Preview blockers today:** custom channels need
  `--dangerously-load-development-channels` at every launch plus an
  interactive full-screen confirmation dialog (bad for automated agent
  spawning), and the flag/contract may change. Revisit when channels
  exit preview or the crabswarm channel can ship via an
  org-allowlisted plugin (`allowedChannelPlugins`).

## Codex state tracking for the send-keys guard (verified 2026-08-27, D20)

From the Codex hooks reference (learn.chatgpt.com/docs/hooks):
- `UserPromptSubmit` — turn start. `Stop` — turn complete (can also
  block-to-continue). `notify` fires `agent-turn-complete` only.
- `PermissionRequest` — fires "when Codex is about to ask for approval";
  a hook may return `behavior: "allow"`/`"deny"` to settle it without a
  dialog. `PostToolUse` after it implies the dialog resolved.
- **Gaps:** no event for a dialog being answered/dismissed (denial is
  silent until `Stop`), and no generic "TUI waiting for input" event —
  hence the capture-pane + dialog-marker text-scan guard before any
  injection (D20c).
- Claude Code is hook-complete for this: `Notification` fires on
  `permission_prompt` / `idle_prompt`.

## Mid-task delivery, and rejected MCP-notification trick (2026-08-27)

- Mid-task ("you have messages" while a turn is running): PostToolUse
  `additionalContext` hooks on Claude Code AND Codex inject an unread
  hint after any tool call; Stop-hook drain covers turn end. OpenCode
  needs neither (HTTP injection works any time). Wired in plan step 8.
- Codex caveat: its hook system is recent; `additionalContext` is
  official for SessionStart/UserPromptSubmit/PostToolUse, but Stop-block
  is third-party-documented only — verify each behavior empirically at
  step 8.
- **Rejected:** abusing standard MCP notifications
  (`notifications/tools/list_changed` → smuggle text into refreshed tool
  descriptions) as an injection channel — host support is uneven, model
  attention to tool descriptions is weak, and it is prompt-injection-
  shaped (hosts sanitize it). Standard MCP notifications reach the host
  app only; no turn starts, so they never help an idle agent either.
- Idle remains: Stop-hook drain means agents don't go idle with unread
  messages; post-idle arrivals need a wake — send-keys (D19/D20) today,
  spike-plan natives later.

## Synthesis for Q9

| Harness | Mid-idle push | Turn-boundary injection | Fallback |
| --- | --- | --- | --- |
| Claude Code | messaging socket (`$CLAUDE_CODE_MESSAGING_SOCKET`) | Stop hook block / `additionalContext` | tmux send-keys |
| Codex | **none** (app-server can't attach to a foreign TUI) | hooks: Stop block, `additionalContext` (UserPromptSubmit/PostToolUse) | tmux send-keys (only mid-idle option) |
| OpenCode | HTTP server: `POST /session/:id/message`, `/tui/*`; plugin w/ SDK client | `session.idle` event → inject via API | (not needed) |

Implication: a crabswarm notifier needs a per-harness adapter layer; the
one mechanism that covers all three uniformly (incl. idle Codex) remains
terminal keystroke injection, with native adapters (Claude socket,
OpenCode HTTP) as better-behaved options where available.
