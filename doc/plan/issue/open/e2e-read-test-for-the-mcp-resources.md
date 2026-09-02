---
tags: chat mcp e2e test
---

# e2e read test for the MCP resources (2026-09-02)

Neither `crabswarm://chat/members` nor `crabswarm://chat/history` has a
real-process read test; both are covered by in-process unit tests only
(`crabswarm/chat/mcpserver/resources_test.go`).

Follow-up: one e2e that starts `crabswarm chat mcp` over stdio against a
live daemon and reads both resources would close the gap for both.
