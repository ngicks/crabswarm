---
tags: apm-package docs chat mcp
---

# Document the chat surface the crabswarm-chat apm package now ships (2026-09-02)

`apm-package/crabswarm-chat/README.md` and
`apm-package/crabswarm-chat/.apm/skills/crabswarm-chat/SKILL.md` teach
only the member CLI verbs. They do not mention the non-destructive
`crabswarm chat history [--limit N]` transcript verb, the bridge's four
MCP tools, or the `crabswarm://chat/members` and
`crabswarm://chat/history` resources the package now ships.

Follow-up: bring both docs up to the shipped surface so agents wired
through the package can discover it.
