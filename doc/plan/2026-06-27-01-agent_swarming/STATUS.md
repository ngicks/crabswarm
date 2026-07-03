# STATUS — Agent Swarming

**Current state:** Plan FINALIZED — all 11 open questions resolved across 3 rounds
and folded into PLAN.md / DECISION.md. Ready to implement. No code written yet.

## Step checklist (mirrors PLAN.md → Implementation steps)

- [ ] 1. Proto: `SwarmService` under `crabswarm/v1` (no Enroll RPC)
- [ ] 2. SQLite store `pkg/crabswarm/swarm` (modernc; members/state/inbox, atomic flip)
- [ ] 3. Auth interceptor (`x-crabswarm-id` = CMDMAN_COMMAND_ID → member)
- [ ] 4. cmdman notifier (`send-keys`) + `cmdman query` name→target reader
- [ ] 5. Server wiring + dispatch policy (same-team / persist / queue / drain-on-Stop)
- [ ] 6. Agent-side messenger CLI (`swarm send` / `inbox` / `members`)
- [ ] 7. Lifecycle hook subcommand (`hook presence`, kind-agnostic)
- [ ] 8. Claude Code plugin scaffold under `plugin/crabswarm/`
- [ ] 9. Codex lifecycle wiring (own hook/notify → same state calls)
- [ ] 10. Chat skill content (`skills/crabswarm-chat/SKILL.md`)
- [ ] 11. Identity/transport contract doc + `swarm_db`/cmdman config fields
- [ ] 12. e2e test (`e2e/crabswarm`, incl. cross-team rejection)

## Done

- Repo grounding: server (UDS+flock+AuditService), proto/buf layout, hook client
  (no auth today), hook events, layered config, plugin build script (plugin
  *source tree does not exist yet*), confirmed `cmdman` is external (not in repo)
  and the server must run host-side.

## In progress

- Nothing. Planning complete (rounds 1–3 all folded in).

## Blocked / needs decision

- None blocking. Implementation-time confirmations (not blockers): exact
  `cmdman query` CLI surface, Codex event→state mapping, the env var cmdman uses
  to expose the compose-project/team name.

## Next action

- Begin Step 1 (proto `SwarmService`) when the user is ready to implement.
