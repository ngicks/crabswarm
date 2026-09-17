# How a Claude Code session registers the MCP server as a channel

Recorded 2026-09-17 on Claude Code v2.1.274 (Linux, WSL2 host), with a
minimal stdio MCP server that declares `capabilities.experimental["claude/channel"] = {}`
and emits one `notifications/claude/channel` fifteen seconds after `notifications/initialized`.

## Interactive session (works)

    claude --mcp-config <file> --strict-mcp-config --dangerously-load-development-channels server:<name>

- The flag is variadic: every argument after it is read as a channel entry, so a
  positional prompt must come before it, or the session exits 1 with
  `--dangerously-load-development-channels entries must be tagged: <prompt> ...`.
- At startup a full-screen dialog appears:

      WARNING: Loading development channels
      ...
      Channels: server:probe
      ❯ 1. I am using this for local development
        2. Exit
      Enter to confirm · Esc to cancel

  One Enter (sent through `cmdman send-keys <id> Enter`) confirms it.
- The startup screen then shows
  `Channels (experimental) messages from server:probe inject directly in this session · restart without --dangerously-load-development-channels to stop`.
- The emitted event arrived as a new turn, rendered in the transcript as
  `← probe: Reply with exactly the word BANANA and stop.` and the model answered it.

## Background session (`claude --bg`) does not work

    claude --bg --name x --mcp-config <file> --strict-mcp-config "<prompt>" --dangerously-load-development-channels server:<name>

- Starts, spawns the MCP server (the handshake is identical), answers the prompt
  and reaches `state: done`, `status: idle` in `claude agents --json`.
- No confirmation dialog appears anywhere: not in `claude logs`, not when the
  session is opened with `claude attach` under a terminal.
- The event the server emitted was never delivered: the transcript holds the one
  user turn and nothing else. Silently dropped, as the channels docs say for an
  unregistered channel.

## Consequence

Claude Code runs as an interactive session under cmdman; the launcher sends one
Enter after start to confirm the dialog. A background session cannot be a
channel host, so it is not used, and the agents listing is not a state source.
