---
tags: codex mcp apm-package env
---

# Codex strips the env, so `crabswarm chat mcp` dies before `initialize` (2026-09-02)

Codex (v0.152.1) fails the crabswarm-chat MCP server at startup with
`handshaking with MCP server failed: connection closed: initialize
response`. Reproduced: codex spawns stdio MCP servers with a fixed env
whitelist (`HOME LANG LOGNAME PATH PWD SHELL SHLVL SSL_CERT_FILE TERM
USER`), dropping `CMDMAN_CMD_ID`, `CRABSWARM_CHAT_TOKEN` and
`XDG_RUNTIME_DIR` even when codex itself has them. `mcpserver.New`
(`crabswarm/chat/mcpserver/mcpserver.go`) calls `cli.ResolveToken`
before serving anything; with no token it returns
`no chat identity token: ...` on stderr and exits 1 with nothing on
stdout, and codex — which does not surface MCP stderr — reports the
closed pipe. Claude Code passes its full env, which is the whole
asymmetry. Verified with `env -i HOME=$HOME PATH=$PATH crabswarm chat
mcp < initialize.json` (exit 1) versus the same with `CMDMAN_CMD_ID`
set (answers `initialize`, `serverInfo.name=crabswarm-chat`). This
answers the earlier "Codex runtime never verified to start the chat
MCP bridge" entry.

Follow-up, two parts:
- Config: codex forwards named variables per server via an `env_vars`
  list (undocumented in `codex mcp add --help`, verified empirically):
  `[mcp_servers.crabswarm-chat] env_vars = ["CMDMAN_CMD_ID",
  "XDG_RUNTIME_DIR"]`. Ship it from
  `apm-package/crabswarm-chat/apm.yml` (apm-cli passes unknown keys
  through `extra:` into the rendered `.codex/config.toml`; whether the
  same key is harmless in the rendered `.mcp.json` for claude is
  unverified — check after `apm install`). Forwarding `CMDMAN_CMD_ID`
  only yields an agent identity when codex itself runs under cmdman;
  otherwise a registered token via `CRABSWARM_CHAT_TOKEN` is the
  fallback.
- Code: move `cli.ResolveToken` out of `New` into `ensureJoined` next
  to the join. `New`'s own doc comment ("Join happens in Run so
  failures surface as MCP tool errors, not a dead harness") and
  `joinWithRetry`'s rationale already forbid dying at startup; the
  token check one line earlier does exactly that. After the change a
  tokenless bridge stays up and its tools say "no identity token".
