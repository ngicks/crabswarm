# How a Claude Code session registers the fakechat plugin as a channel

Recorded 2026-09-22 on Claude Code v2.1.278 (Linux, WSL2 host) with the
official plugin `fakechat@claude-plugins-official` version 0.0.1, installed in
`$CLAUDE_CONFIG_DIR/plugins/cache/`. Every launch set `FAKECHAT_PORT`
explicitly, because a plugin server that inherits no port binds 8787 and
collides with any fakechat already running on the host.

`claude --help` does not list `--channels`. The flag works, and the plugin's own
README documents it.

## Interactive session (works)

    FAKECHAT_PORT=18788 claude --model haiku --channels plugin:fakechat@claude-plugins-official

- No confirmation dialog appears. The session opens straight onto its prompt.
- The startup banner carries one channel line, which stays on screen for the
  whole session:

      ▎ Channels (experimental) messages from plugin:fakechat@claude-plugins-official
      ▎ inject directly in this session · restart without --channels to stop

- The plugin server took the port from the launch environment. `127.0.0.1:18788`
  answered `GET /` with 200 and a page titled `fakechat`, and nothing new bound
  8787.
- One `POST /upload` to that port arrived as a new turn, rendered in the
  transcript as

      ← fakechat · web: [crabswarm chat] new message from team/bob — reply with exa…

  The notice reached the model. The model then declined to obey the instruction
  inside it and called it a prompt injection, so a delivered notice is not an
  obeyed notice.

## Background session (`claude --bg`) also works

    FAKECHAT_PORT=18789 claude --bg --name x --model haiku "<prompt>" --channels plugin:fakechat@claude-plugins-official

- `--channels` is variadic. A prompt written after the flag is read as another
  channel entry, and the session starts with no prompt at all and reports
  `backgrounded · <id> · <name> (idle — send a prompt to start)`. The prompt has
  to come before the flag.
- That first session never appeared in `claude agents --json`, bound nothing on
  its port, and answered `claude logs <id>` with
  `connect ENOENT /tmp/cc-daemon-0/<hash>/control.sock`.
- With the prompt in front, the session answered it, reached `state: done` and
  `status: idle` in `claude agents --json`, and stayed alive.
- The plugin server bound 18789 there too, and one `POST /upload` to it started a
  new turn. The session log holds the same `← fakechat · web: …` line the
  interactive screen showed.

## How the session ran

    cmdman run -t --rm -n fakechat-launch -w <scratch dir> -- \
      env -u CLAUDE_CODE_CHILD_SESSION -u CLAUDE_CODE_SESSION_ID \
          -u CLAUDE_JOB_DIR -u CLAUDECODE -u CLAUDE_CODE_ENTRYPOINT \
          FAKECHAT_PORT=18788 claude --model haiku \
          --channels plugin:fakechat@claude-plugins-official
    cmdman capture-screen fakechat-launch
    cmdman stop fakechat-launch

The scratch directory sits outside any repository. The cleared variables keep
the launching session out of the recording. `CLAUDE_CONFIG_DIR` stayed set,
since the plugin lives under it.

## Consequence

The plugin channel needs no dialog and no development-channel flag, and it
reaches a background session as well as an interactive one. A sender only needs
the port the plugin server listens on.
