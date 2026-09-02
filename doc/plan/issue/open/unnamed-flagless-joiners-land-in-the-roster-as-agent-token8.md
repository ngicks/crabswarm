---
tags: chat join naming
---

# Unnamed flagless joiners land in the roster as agent-<token8> (2026-09-02)

`defaultName` (`crabswarm/chat/service.go`) always derives
`agent-<first 8 of token>` for an unnamed joiner, so a member that
declared it is not an agent still carries an `agent-` name. Cosmetic,
but the name lies about the kind. Changing it touches the e2e pin of
`alpha/agent-tok-bare` in `e2e/crabswarm/chat_test.go`.

Follow-up: derive a kind-neutral default (or a kind-matched prefix).
