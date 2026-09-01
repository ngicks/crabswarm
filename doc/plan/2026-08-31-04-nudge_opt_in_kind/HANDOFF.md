# Handoff — nudge opt-in kind

Deferred/out-of-scope discoveries from the implementation run
(2026-09-02). Entries are candidates for `doc/plan/issue/issue.md`.

# A flagless member cannot upgrade to agent on the same token

`Store.Join` is first-join-wins: a member that joined without `--agent`
stays `KindHuman` until it leaves and re-joins. Harmless for the MCP
bridge (it always declares agent on its first join), but a person who
joins by hand and then starts a harness on the same token stays
inbox-only, with no verb to flip the kind in place. Documented in
`Service.Join`'s doc comment; decide whether a re-join (or an admin
verb) may upgrade the kind before this bites a real workflow.

# Flagless members are never reaped and hold their names forever

`checkLiveness` (crabswarm/chat/service.go) is deliberately agent-only,
so a `KindHuman` member is never reaped even when the provider forgets
its token — it drops out of the cmdman status display but holds its
name against colliding joiners indefinitely. Intended for
admin-registered humans; worth revisiting if plain-shell membership
becomes common (an explicit `chat leave` is the current answer).

# An unnamed flagless joiner is named `agent-<token8>`

`defaultName` (crabswarm/chat/service.go) is the last resort when
neither the join request nor the team-info provider supplied a name, and
it still prefixes `agent-`. A person who joins by hand without `--name`
therefore lands in the roster as `agent-01234567` while being
inbox-only, so the name says the opposite of the kind. Cosmetic — no
behavior reads the prefix — but changing it means either passing the
declared kind into `defaultName` or dropping the prefix, and the e2e
suite pins the current spelling (`alpha/agent-tok-bare` in
e2e/crabswarm/chat_test.go), so the pin moves with it.
