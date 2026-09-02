---
tags: apm-package docs mcp
---

# Stale claim: an apm bundle carries the same wiring as a source install (2026-09-02)

`apm-package/crabswarm-chat`'s README claims a bundle install carries
the same wiring as an install from source, but the MCP-server
registration renders into `.mcp.json` / `.codex/config.toml` via
apm.yml, and transitive installs need `--trust-transitive-mcp`; the
claim looks stale (pre-existing, noted during the MCP-server run).

Follow-up: re-verify a bundle install end to end and fix the README.
