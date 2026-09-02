---
tags: codex mcp apm-package
---

# Codex runtime never verified to start the chat MCP bridge (2026-09-02)

apm writes `[mcp_servers.crabswarm-chat]` into `.codex/config.toml`,
but nobody has confirmed codex actually starts the bridge.

Follow-up: verify once against a real codex runtime.
