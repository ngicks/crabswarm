---
tags: chat admin cli
---

# Add `crabswarm chat admin` subcommand (2026-08-31)

Admin works fundamentally differently from members: it authenticates by
age identity + nonce challenge (`crabswarm/chat/admin.go`,
`crabswarm/chat/cli/admin.go`), never holds a member token, never joins
a room as a member, and always names the room it operates on
explicitly. Today the admin verbs (`chat register`, `chat team`) sit
mixed among the member verbs under `chat`, and there is no admin way to
send into or observe a room — participating requires minting a human
member token via `chat register` and hand-exporting it, which is
counter-intuitive.

Follow-up: group the admin plane under `crabswarm chat admin ...`
(register, team, plus new room-scoped verbs such as send/log), all
age-authenticated and all taking an explicit room id. Do not add an
`--admin` flag to `chat join`; admin never becomes a member.
