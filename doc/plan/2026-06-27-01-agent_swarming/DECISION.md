# DECISION LOG — Agent Swarming

One entry per material decision: the choice, the rationale, the rejected
alternatives. Stubs below are seeded from the Open questions and resolve as the
user answers.

## Settled (from grounding, not in question)

- **gRPC + protobuf over the existing UDS server.** Matches the repo's comms
  stack (`.claude/rules/base.md`) and reuses `server.Serve` /
  `pkg/api/...`. New service registered alongside `AuditService`.
- **Server runs host-side.** It must shell out to `cmdman` (which owns the
  podman containers); `cmdman` is external and not in this repo.
- **Token resolution, not token assertion.** The server resolves a metadata
  token to a member; agents never self-assert identity. This is what makes
  identity "hard to forge from inside a container."
- **Random per-member tokens held host-side** (not derived/HMAC), so nothing
  inside a container lets it guess another member's token. Host-forgeable by
  design (host is trusted).

## Resolved — round 1 (2026-06-29)

### D1. Container transport (Q1) — **bind-mount the host UDS**
- **Choice:** cmdman bind-mounts `CRABSWARM_SOCK` (the host UDS) into each
  container; the agent dials it locally. No TCP/vsock listener added.
- **Rationale:** matches the current UDS-only server with zero new listener
  surface; the metadata id already guards the shared socket.
- **Rejected:** TCP listener (extra bind/firewall surface), vsock (exotic, podman
  support varies), cmdman stdin proxy (couples every call to cmdman).

### D2. Identity injection (Q2) — **env var (named `CMDMAN_COMMAND_ID`, see DA)**
- **Choice:** cmdman injects the id (+ team/name) as env vars at launch; the
  agent attaches it as gRPC metadata. (Round 2 fixed the exact name to
  `CMDMAN_COMMAND_ID`.)
- **Rationale:** cmdman already manages launch; env is the cheapest injection and
  intra-container exposure is acceptable under the threat model.
- **Follow-up (QA, resolved round 2):** the id is random, so it is itself the
  credential — no separate token.
- **Rejected:** mounted secret file (extra mount), bootstrap handshake (complexity
  not yet warranted).

### D3. Identity-map ownership (Q3, folds Q10) — **cmdman owns it; no enroll RPC**
- **Choice:** cmdman embeds the name↔id↔container-id↔target relation in its own
  metadata; crabswarm derives it from there (server-side) rather than running a
  separate enroll. No `Enroll`/`AdminService` RPC.
- **Rationale:** cmdman already holds the authoritative mapping; a parallel
  crabswarm enroll would duplicate it and invite drift.
- **Rejected:** host-side `crabswarm swarm enroll`, agent self-register of the
  full map, standalone roster file as the *primary* source.

### D5. Busy-recipient delivery (Q5) — **queue until Stop; SQLite-stored messages**
- **Choice:** message bodies persist in the server's SQLite store; a `running`
  recipient's message stays pending and the `cmdman send-keys` nudge ("check your
  inbox") fires only when its Stop hook flips it to `idle`. send-keys never
  carries message content.
- **Rationale:** never interrupts an in-progress turn; durable messages survive
  restarts; send-keys is a lossy TTY channel unfit for payloads.
- **Rejected:** immediate send-keys regardless of state, idle-for-N-seconds timer,
  content-over-send-keys.

### D7. Persistence (Q7) — **durable, server-side SQLite** (decided with D5)
- **Choice:** members/state/inbox live in a server-side SQLite database.
- **Rationale:** the user specified SQLite; gives durable inboxes + a natural home
  for the atomic optimistic transition (single transaction).

## Resolved — round 2 (2026-06-29)

### DA. Credential (QA) — **the random `CMDMAN_COMMAND_ID` is the credential**
- **Choice:** cmdman assigns a random per-command id, injected as env
  `CMDMAN_COMMAND_ID`; the agent sends it as gRPC metadata `x-crabswarm-id` and it
  is resolved to a member directly. No separate token minted.
- **Rationale:** it's already a random string, so it satisfies "unguessable from
  inside another container" without a second secret to inject/rotate.
- **Consequence:** the server must treat the id as a secret — never echo it where
  a teammate could read it.
- **Rejected:** minting a separate crabswarm token; CMDMAN_ID-plus-cmdman-verify
  (unnecessary once the id is random).

### DB. Reading cmdman metadata (QB) — **shell out to `cmdman query`**
- **Choice:** the server `exec`s `cmdman query` on demand to resolve name→target;
  cmdman stays the single source of truth.
- **Rationale:** no duplicated/drifting map in crabswarm; matches the existing
  "shell out to cmdman" pattern (send-keys).
- **Open dependency:** exact `cmdman query` subcommand/flags + output format must
  be confirmed against the real cmdman CLI (may need adding on cmdman's side).
- **Rejected:** mirror-at-Register (drift risk), roster file (extra file contract).

### D4. State model + heartbeat (Q4) — **idle/running/waiting-input/offline + TTL**
- **Choice:** four states. Hooks drive idle↔running and Notification→waiting-input;
  a heartbeat TTL marks silent members offline. Only `idle` is deliverable.
- **Rationale:** waiting-input (blocked on a permission/user prompt) must not be
  injected into; offline+TTL recovers from missed Stop hooks.
- **Rejected:** idle/running-only (strands on missed Stop), server-ping liveness.

### D8. Naming (Q8) — **`swarm` verbs / skill `crabswarm-chat` / `x-crabswarm-id`**
- **Choice:** `crabswarm swarm send|inbox|members`; bundled skill `crabswarm-chat`;
  gRPC metadata key `x-crabswarm-id` carrying the `CMDMAN_COMMAND_ID`; nudge text
  names the `crabswarm-chat` skill.
- **Rejected:** `crabswarm msg ...`; `authorization: Bearer`.

## Resolved — round 3 (2026-06-29)

### D6. "Start of work" event (Q6) — **UserPromptSubmit; nudge only idle**
- **Choice:** Claude Code `UserPromptSubmit` flips idle→running. Only `idle`
  members are nudged; `running`/`waiting-input` stay pending until `Stop→idle`.
- **Rationale:** UserPromptSubmit fires at turn start (before any tool), the
  earliest reliable "work began" signal; injecting into non-idle panes risks
  corrupting a turn or a permission prompt.
- **Rejected:** first PreToolUse (misses pure-answer turns), either-first (no gain).

### DC. SQLite driver + db path (QC) — **modernc.org/sqlite; `swarm_db` config**
- **Choice:** pure-Go `modernc.org/sqlite`; db path from a new `swarm_db` config
  field defaulting to `$XDG_STATE_HOME/crabswarm/swarm.db`.
- **Rationale:** plugin build is `CGO_ENABLED=0`, ruling out cgo `mattn/go-sqlite3`;
  state dir keeps inboxes across reboots; config field allows override.
- **Rejected:** mattn/go-sqlite3 (cgo), hardcoded $XDG_RUNTIME_DIR (ephemeral).

### D9. Codex scope (Q9) — **Claude + Codex both this cut, kind-agnostic core**
- **Choice:** store/server/skill are kind-agnostic; wire both Claude Code (plugin
  hooks) and Codex (its own hook/notify mechanism) to the same state calls now.
- **Rationale:** the swarm is mixed Claude/Codex; the only kind-specific surface is
  the hook→state mapping, and `pkg/claudehook` is already reusable for Codex.
- **Open detail:** exact Codex event→state mapping confirmed at implementation.
- **Rejected:** Claude-first with Codex as a follow-up.

### D11. Team source (Q11) — **team = cmdman compose project; same-team only**
- **Choice:** a team is the cmdman compose project an agent launched under
  (implicit membership, recorded at Register from a launch env value). `Send` /
  `ListMembers` are restricted to the same team; cross-team is disabled.
- **Rationale:** the compose project is a natural, already-existing team boundary;
  no separate roster to maintain. Cross-team can be added later if needed.
- **Rejected:** declared config roster, implicit-but-cross-team-allowed.

## Open

_None. All questions resolved; the plan is ready to implement._
