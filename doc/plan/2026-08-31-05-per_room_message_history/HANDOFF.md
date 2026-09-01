# Handoff — per-room message history

Deferred/out-of-scope discoveries from the implementation run
(2026-09-02). Entries are candidates for `doc/plan/issue/issue.md`.

# Document the `chat history` verb in the crabswarm-chat apm package

`apm-package/crabswarm-chat/README.md` and
`apm-package/crabswarm-chat/.apm/skills/crabswarm-chat/SKILL.md` describe
the member chat verbs but do not mention the new non-destructive
`crabswarm chat history [--limit N]` transcript verb. Agents wired
through the package will not discover it until those docs are updated.
