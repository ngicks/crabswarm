# Agent Swarming: teams, identity, presence, and the crabswarm messenger

**Summary:** Let a *team* of `cmdman`-managed, podman-isolated LLM agents
(Claude Code / Codex) talk to each other through the crabswarm server: each
agent carries an injected identity token in gRPC metadata, a claude-code plugin
tracks each agent's idle/running state via lifecycle hooks, and the server
delivers "you have message(s)" notifications by translating a teammate *name*
into a `cmdman` target and typing the prompt into the recipient's pane.

> **Status: FINALIZED.** All 11 open questions resolved across 3 rounds and folded
> in (see [Open questions](#open-questions) for the summary and DECISION.md for the
> full log). Section headings tag the decisions that shaped them. Ready to
> implement; a few implementation-time confirmations are noted inline.

---

## Goal / success criteria

- A human (or orchestrator) can define a **team** (a cmdman compose project) and
  launch members; each member runs as a Claude Code (or Codex) agent inside its
  own podman container, managed by `cmdman`.
- Each agent has an **identity** (team, name, id) and a **token** injected at
  launch. The agent attaches the token as gRPC metadata on every call. The server
  authenticates the token and resolves it to a member.
- A claude-code **plugin** ships hooks that report agent lifecycle state to the
  server: `idle` on session start, `running` when work begins, `idle` again on
  stop.
- The server tracks per-member **state** and an **inbox**. When a teammate sends a
  message, the server delivers a `cmdman send-keys` notification —
  `"you have N message(s). read inbox use skill <name>"` + `Enter` — to the
  recipient's pane, *only* when it is safe (recipient idle), flipping the member
  to `running` **atomically** so concurrent deliveries don't double-inject.
- The plugin bundles a **skill** that teaches the agent how to use the messenger
  (`send`, `inbox`, `members`).
- The end-to-end flow works across podman containers, with a documented contract
  for how **container id** and **agent id** are coordinated between `cmdman`, the
  container, the agent, and the server.

Success = e2e test (per repo convention) that registers two same-team members,
has A send to B, and asserts B's inbox + a recorded `cmdman send-keys` invocation
+ the optimistic state flip.

## Scope

- New gRPC service(s) for membership, presence/state, and messaging.
- Token-based auth (gRPC metadata) with a host-side mint / container-side use
  asymmetry.
- Server-side in-memory state store (members, state machine, inboxes,
  name→cmdman-target map) with atomic optimistic transitions.
- A `cmdman` notifier abstraction (shells out to `cmdman send-keys`).
- New cobra subcommands: agent-side messenger (`swarm send|inbox|members`),
  lifecycle hook (`hook presence`).
- A claude-code plugin scaffold under `plugin/crabswarm/` wiring the hooks and
  bundling the messenger skill.
- A documented identity/transport contract (env vars, socket mounting).

## Non-goals

- Cryptographically hardened auth. Tokens are deliberately host-forgeable; the
  only property we want is "hard to guess from inside another agent's container."
- Building/forking `cmdman` itself — it is an external binary we shell out to
  (`send-keys`, `query`). If `cmdman query` doesn't expose name→target yet, adding
  it is cmdman-side work, out of scope here.
- Cross-team messaging. A member can only `send` within its own team
  (= cmdman compose project). Cross-team is explicitly disabled for now (Q11).
- Scheduling / load-balancing work across the swarm (this is plumbing, not a
  scheduler).

## Context (current state of the repo)

- **Server**: `pkg/crabswarm/server/server.go` — `Server.Serve` listens on a Unix
  domain socket (UDS), holds a flock to prevent duplicate servers, registers a
  single gRPC service (`AuditService`). `auditServiceServer.ReportHookInputEvent`
  just logs. This is the surface we extend.
- **Proto**: `pkg/api/schema/proto/crabhook/v1/audit_service.proto` +
  `sdktypes/v1`. Generated Go lands in `pkg/api/gen/proto/go/...` via buf
  (`pkg/api/buf.gen.yaml`, `buf.yaml`; module `schema/proto`, go_package_prefix
  `github.com/ngicks/crabswarm/pkg/api/gen/proto/go`).
- **Hook client**: `pkg/crabswarm/hook/audit.go` + `cmd/.../hook_audit.go` —
  `crabswarm hook audit` dials the UDS with `insecure` creds and forwards stdin
  to the server. Today there is **no metadata/auth** on the call.
- **Hook plumbing**: `cmd/.../hook.go` groups `audit`, `exec`, `path`.
  `pkg/crabswarm/hook/exec/exec.go` is a declarative text/template→exec hook;
  `pkg/claudehook/handler/handler.go` standardizes hook stdout/exit semantics.
- **Hook events available** (`pkg/api/types/sdktypes/v1/types.go:210+`):
  `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `Stop`,
  `SessionEnd`, `Notification`, `SubagentStart/Stop`, etc. These are the events
  the lifecycle hooks subscribe to.
- **Config**: `pkg/crabswarm/config.go` — layered defaults<file<env<flags;
  `Sock` (UDS path, default `$XDG_RUNTIME_DIR/crabswarm/default.sock`),
  `ProjectDir` (`CLAUDE_PROJECT_DIR`), `HookExec`. New swarm config fields slot in
  here following the `PartialConfig`/`Apply` pattern.
- **Plugin**: `build-plugin-crabswarm.sh` compiles the binary into
  `plugin/crabswarm/bin/crabswarm-<os>-<arch>` (bin gitignored). **The plugin
  source tree does not exist yet** — `plugin/crabswarm/` (plugin.json, hooks,
  skills) must be created.
- **`cmdman`**: not in this repo and not referenced anywhere; it is an external,
  tmux-like command manager (`send-keys <target> "text" "Enter"`) that owns the
  containers. crabswarm shells out to it. The crabswarm **server must run on the
  host** (where `cmdman` and podman live), not inside a container.
- **Conventions** (`.claude/rules/*`): thin `./cmd` (flags only, logic in
  `pkg/<name>` and `pkg/<name>/cli`), context-first, DI over globals, small
  consumer-side interfaces, errgroup/semaphore/singleflight for concurrency,
  errors-as-values with `%w`. gRPC + protobuf is the comms stack.

## Approach

### Component map

```
host (trusted)                                   podman container (semi-trusted)
┌──────────────────────────────┐                 ┌──────────────────────────────┐
│ cmdman  (identity source)    │  launch+inject  │ claude code / codex agent    │
│  - launches agents           │ ──────────────► │  env: CMDMAN_COMMAND_ID       │
│  - owns name↔id↔cid↔target   │                 │       (random), team, name,   │
│    metadata                  │                 │       CRABSWARM_SOCK          │
│  - send-keys <target> "..."  │ ◄────────────┐  │  (UDS bind-mounted in)        │
│  - query name→id/target      │  nudge(host) │  │  plugin hooks:               │
├──────────────────────────────┤              │  │   SessionStart→register/idle │
│ crabswarm serve (host)       │              │  │   UserPromptSubmit→running   │
│  SwarmService (gRPC/UDS)     │              │  │   Notification→waiting-input │
│   Register/SetState/Heartbeat│ ◄────────────┼──┤   Stop→idle                  │
│   Send/ReadInbox/ListMembers │ gRPC, header │  │  skill: crabswarm-chat        │
│  ┌────────────────────────┐  │ x-crabswarm- │  │   `crabswarm swarm send/inbox│
│  │ SQLite: members, state,│  │ id = command │  │    /members`                 │
│  │ inbox (durable msgs)   │  │ id           │  └──────────────────────────────┘
│  └────────────────────────┘  │              │
│  notifier ─ exec `cmdman      │              │
│  send-keys`; name→target via  │              │
│  exec `cmdman query` ─────────┘              │
└──────────────────────────────┘
```

### Identity & credential model  **[RESOLVED Q1–Q3, QA, QB → D1–D3, DA, DB]**

- **`cmdman` owns identity.** `cmdman` launches each agent and embeds ids into its
  own metadata; the name↔command-id↔container-id↔target relation is *derived from
  cmdman's metadata*, not from a separate crabswarm enroll step.
- **The credential is `CMDMAN_COMMAND_ID`** — a **random string** cmdman assigns
  per launched command/agent and injects (with team/name) as an env var into the
  container. Because it is random, it doubles as the auth credential directly (no
  separate token minted — DA). The agent attaches it as gRPC metadata
  `x-crabswarm-id` on every call.
- **Auth = metadata id resolution, not assertion.** A server-side unary
  **interceptor** reads `x-crabswarm-id` and resolves it to a member; the resolved
  identity is authoritative. An agent inside container A only knows its *own*
  random `CMDMAN_COMMAND_ID`, so it cannot impersonate B (it can't guess B's
  random id). Host-forgeable by design (the host can read cmdman metadata / set
  any id — host is trusted); the only property needed is cross-container
  unguessability, which the random id provides.
- **Name → target at notify time:** to send a nudge, the server **shells out to
  `cmdman query`** (exact subcommand/flags TBD against the real cmdman CLI) to
  translate a teammate **name → cmdman target**, keeping cmdman the single source
  of truth (DB).

### State machine & optimistic transition  **[RESOLVED Q4, Q6 → D4, D6]**

States: `idle`, `running`, `waiting-input`, `offline`. Liveness via a heartbeat
TTL: a member that goes silent past the TTL is marked `offline` so a missed Stop
hook can't strand it `running` forever. The transitions are defined per *agent
kind* (Claude Code / Codex) but normalize to the same four states (see "Agent
kinds" below).

Claude Code mapping:
- `SessionStart` hook → `Register` (idempotent) → `idle`.
- `UserPromptSubmit` hook → `SetState(running)` (this is "start of work").
- `Notification` hook (Claude needs permission / user input) → `waiting-input`.
- `Stop` hook → `SetState(idle)`; on this transition the server drains the
  recipient's pending inbox and, if non-empty, fires the "check your inbox" nudge.
- `SessionEnd` hook → `offline`. Heartbeat TTL expiry → `offline`.

Rules:
- **Nudge only when `idle`.** `running` and `waiting-input` are non-deliverable
  (don't corrupt an in-progress turn or a y/n permission prompt); their messages
  stay pending until the next `Stop → idle`.
- **Optimistic running on delivery**: when the server fires a `cmdman send-keys`
  nudge, it flips the target `idle→running` **atomically** (one SQLite
  transaction: check-pending + check-state + flip + then spawn send-keys) so two
  concurrent deliveries can't double-inject. Only the transition that wins the
  flip injects.

### Agent kinds (Claude Code + Codex, both this cut)  **[RESOLVED Q9 → D9]**

The store/server/skill are **kind-agnostic**; only the hook→state mapping differs.

- **Claude Code**: the bundled claude-code plugin wires the events above.
- **Codex**: Codex doesn't use the Claude plugin/hook schema, so its lifecycle is
  wired through Codex's own hook/notify mechanism (config in the Codex devenv),
  mapped to the same `crabswarm hook presence` state calls. `pkg/claudehook` is
  already "reusable for codex" and `sdktypes/v1/ext/codex` exists; the mapping
  layer lives behind the same `SetState`/`Register` calls. **Exact Codex
  event→state mapping is an implementation detail to confirm against the Codex
  hook surface.**

### Teams  **[RESOLVED Q11 → D11]**

- **A team = a cmdman compose project.** Membership is implicit: an agent's team
  is the compose project cmdman launched it under, surfaced as an env value at
  launch and recorded on the member row at `Register`. No separately declared
  roster.
- **Same-team only.** `Send` validates `sender.team == recipient.team` and rejects
  cross-team sends. `ListMembers` returns only same-team members. Cross-team
  messaging is deferred.

### Messages & notification delivery  **[RESOLVED Q5+Q7, see DECISION D5/D7]**

- **Messages are durable.** `Send(to, body)` persists the message to the server's
  **SQLite** store (an inbox row per recipient). The message *content* never rides
  the `cmdman` channel.
- **The send-keys payload is just a nudge** — e.g. `you have N message(s). check
  your inbox — use skill crabswarm-chat` (exact text is config) + a separate
  `Enter` key arg, matching `cmdman send-keys <target> "<text>" "Enter"`. The
  agent then pulls the real messages via `crabswarm swarm inbox` (server reads
  SQLite).
- **Delivery policy: queue until Stop.** A message for a `running` recipient is
  stored and left pending; the nudge fires only when the recipient's `Stop` hook
  flips it to `idle` and the server drains a non-empty pending set. Never
  interrupts mid-turn.

### cmdman notifier

- A small consumer-side interface (e.g. `Notifier interface { Notify(ctx, target,
  text string) error }`) with a default impl that `exec`s the configured `cmdman`
  binary. Server-side only. Configurable binary path / arg template via config so
  tests can stub it (DI per repo rules).

### Transport into containers  **[RESOLVED Q1, see DECISION D1]**

`cmdman` **bind-mounts the host UDS** (`CRABSWARM_SOCK`) into each container; the
agent dials it like any local socket. No network listener is added. The metadata
id guards the shared socket against cross-member impersonation.

### Coordinating container id & agent id  **[RESOLVED Q3, see DECISION D3]**

**`cmdman` is the source of truth.** It launches agents, owns the
name↔command-id↔container-id↔target relation in its own metadata, and injects the
random `CMDMAN_COMMAND_ID` (+ team/name) into each container. There is **no
separate crabswarm enroll step**.

1. **Agent → server (`Register`, at SessionStart)**: the agent presents its
   `CMDMAN_COMMAND_ID` (gRPC metadata `x-crabswarm-id`); the server upserts the
   member row (in SQLite) and marks it `idle`/live.
2. **Server → cmdman (translate at notify time)**: to send a nudge, the server
   resolves a teammate **name → cmdman target** by shelling out to `cmdman query`
   (single source of truth), then `cmdman send-keys <target> ...`.

The container only ever needs its own random `CMDMAN_COMMAND_ID` + name; the
authoritative map lives in cmdman, and member/state/inbox rows live in the
server's SQLite.

### Rejected alternatives

- **Agent self-asserts its name in metadata (no id resolution).** Rejected: any
  container could claim any name. Resolving the injected `CMDMAN_COMMAND_ID` to a
  member is the whole point of "hard to guess from inside the container."
- **Server pushes notifications over a gRPC stream the agent holds open.**
  Rejected as the *primary* path: the agent is a CLI turn-based process, often
  blocked/idle with no event loop to receive a push; `cmdman send-keys` into the
  pane is how you actually get a turn-based agent to act. (A heartbeat/notify
  stream may still complement it — see Q4.)
- **Message content over the cmdman/send-keys channel.** Rejected: send-keys is a
  lossy TTY nudge. Messages are durable rows in SQLite; send-keys only says
  "check your inbox."
- **A separate crabswarm enroll step that re-derives the identity map.** Rejected:
  cmdman already owns name↔id↔cid↔target; duplicating it invites drift. The server
  mirrors/queries cmdman instead.

## Implementation steps

> Each step should be independently buildable/verifiable. File/symbol names are
> indicative and may shift once Open questions resolve.

1. **Proto: swarm service.** Add
   `pkg/api/schema/proto/crabswarm/v1/swarm_service.proto` defining `SwarmService`
   with `Register`, `SetState`, `Heartbeat`, `Send`, `ReadInbox`, `ListMembers`.
   (No `Enroll` RPC — cmdman owns identity; see resolved Q3.) Regenerate via buf
   into `pkg/api/gen/proto/go/crabswarm/v1`. Verify: `buf generate` +
   `go build ./...`.
2. **SQLite store package.** New `pkg/crabswarm/swarm` (store): schema for
   `members` (command_id PK, name, team, state ∈ idle/running/waiting-input/
   offline, last_heartbeat) and `inbox` (id, recipient, sender, body, created_at,
   delivered_at); CRUD + an atomic optimistic `idle→running` transition + "drain
   pending for recipient" query, all in one transaction where ordering matters.
   Pure-Go driver **modernc.org/sqlite** (plugin build is `CGO_ENABLED=0`, no
   cgo). DB path from the new `swarm_db` config field (default
   `$XDG_STATE_HOME/crabswarm/swarm.db`). Concurrency test for the double-`Send`
   race. Verify: `go test`.
3. **Auth interceptor.** Unary server interceptor that extracts `x-crabswarm-id`
   (the `CMDMAN_COMMAND_ID`) from metadata, resolves to a member via the store,
   injects member into context, rejects unknown ids. Verify: unit test with
   `metadata.NewIncomingContext`.
4. **cmdman notifier + query reader.** A `Notifier` interface (default impl
   `exec`s `cmdman send-keys <target> <text> Enter`) and a name→target resolver
   that `exec`s `cmdman query` (exact flags TBD vs real cmdman CLI), both
   config-driven (binary path/args) and stubbable. Verify: unit-test arg
   construction + name→target resolution with a fake cmdman.
5. **Server wiring + dispatch policy.** Implement `SwarmService` backed by the
   SQLite store + notifier: `Send`→(validate same-team)→persist→(recipient idle?
   nudge : leave pending), `SetState(idle)` (from Stop)→drain-pending→nudge, with
   the atomic flip; `running`/`waiting-input` recipients are non-deliverable;
   `ListMembers` is same-team-scoped. Register alongside `AuditService` in
   `server.Serve`. Verify: server-level unit test against a temp SQLite db + fake
   cmdman (incl. a cross-team send rejection case).
6. **Agent-side messenger CLI.** `crabswarm swarm send <name> <text>`,
   `crabswarm swarm inbox`, `crabswarm swarm members` — dial the bind-mounted UDS,
   attach `CMDMAN_COMMAND_ID` from env/config as the `x-crabswarm-id` metadata.
   Thin cmd → `pkg/crabswarm/swarm/cli`. Verify: e2e over UDS.
7. **Lifecycle hook subcommands (kind-agnostic).** A `crabswarm hook presence
   --state=...` (or envelope-parsing) command the hooks call. Claude Code mapping:
   SessionStart→register/idle, UserPromptSubmit→running, Notification→
   waiting-input, Stop→idle, SessionEnd→offline. Verify: feed sample hook JSON,
   assert resulting state call.
8. **Claude Code plugin scaffold.** Create `plugin/crabswarm/` — `plugin.json` (or
   `.claude-plugin/`), `hooks/hooks.json` wiring the lifecycle + audit hooks to
   the bundled binary, and `skills/crabswarm-chat/SKILL.md`. Confirm
   `build-plugin-crabswarm.sh` drops the binary where the manifest expects it.
   Verify: load the plugin in Claude Code; hooks fire.
9. **Codex lifecycle wiring.** Map Codex's own hook/notify mechanism to the same
   `crabswarm hook presence` state calls (confirm Codex event→state against its
   hook surface; reuse `pkg/claudehook`/`sdktypes/v1/ext/codex`). Provide the
   Codex devenv config snippet. Verify: feed sample Codex hook payloads, assert
   state calls.
10. **Chat skill content.** Author `skills/crabswarm-chat/SKILL.md`: identity (read
    from env), how to `send` / read `inbox` / list `members` (same-team only),
    etiquette (don't spam running teammates), and the "check your inbox" loop.
    Verify: prose review.
11. **Identity/transport contract doc + config.** Document env-var contract
    (`CMDMAN_COMMAND_ID`, team = compose project, name, `CRABSWARM_SOCK`), UDS
    bind-mounting, and the cmdman launch recipe (both Claude + Codex devenvs); add
    swarm config fields (`swarm_db` path, cmdman binary path; `sock` already
    exists) to `pkg/crabswarm/config.go` following `PartialConfig`/`Apply`.
    Verify: `crabswarm config` round-trips.
12. **e2e test.** `e2e/crabswarm`: register A and B (same team), A `send` B, assert
    B's SQLite inbox row + recorded cmdman send-keys (fake cmdman on PATH) fires
    only after B's Stop→idle + the optimistic flip; plus a cross-team rejection
    case. Verify: `go test ./e2e/...`.

## Testing and verification

- Unit tests next to packages (`_test.go`), per repo rules.
- Concurrency test for the atomic optimistic flip (double-`Send` race).
- A fake `cmdman` (script on PATH that records argv) for notifier + e2e tests.
- e2e under `e2e/crabswarm` exercising register→send→(queue)→Stop→nudge→inbox
  against a temp SQLite db.
- `go build ./...`, `buf generate` clean, golangci-lint per `.golangci.yaml`.

## Risks

- **Turn-based agents can't be "pushed" to.** The whole notify path depends on
  `cmdman send-keys` actually causing the agent to take a turn; if the pane is at
  a prompt awaiting input it works, but mid-tool-call or mid-permission-prompt it
  may not. Optimistic `running` + deliver-on-idle mitigates but does not fully
  remove this.
- **State drift.** If a hook is missed (crash, killed session), an agent can be
  stuck `running` and never receive notifications. Mitigated by the heartbeat TTL
  → `offline` fallback (D4), but the TTL value is a tuning risk.
- **cmdman coupling.** We depend on an external tool's CLI contract — both
  `send-keys <target> "text" "Enter"` and a `cmdman query` that exposes
  name→id/target in a machine-readable form. If `cmdman query` doesn't exist yet,
  it has to be added on the cmdman side (out of this repo's scope).
- **Socket reachability from containers.** Bind-mounting a host UDS into podman
  works but is a deployment detail that can bite (paths, SELinux labels). (D1)
- **`CMDMAN_COMMAND_ID` leakage via env.** The id is visible to all processes in
  the container; acceptable under the threat model (intra-container trusted,
  cross-container not) because the id is a random string — its secrecy across
  containers is the whole security basis, so it must never be logged/echoed by the
  server in a way a teammate could read.

## Open questions

> Resolve top-down; fold each answer into the sections above and append a
> DECISION.md entry. ~4 per round.

**All resolved — none open.** Summary (see DECISION.md for full entries):

Round 1: **Q1** bind-mount UDS (D1) · **Q2** env var `CMDMAN_COMMAND_ID` (D2) ·
**Q3** cmdman owns the identity map, no enroll (D3, folds **Q10**) · **Q5**
queue-until-Stop, SQLite-stored messages (D5) · **Q7** durable SQLite (D7).

Round 2: **QA** the random `CMDMAN_COMMAND_ID` is itself the credential (DA) ·
**QB** server shells out to `cmdman query` to resolve name→target (DB) · **Q4**
states idle/running/waiting-input/offline + heartbeat TTL→offline (D4) · **Q8**
verbs `crabswarm swarm send|inbox|members`, skill `crabswarm-chat`, metadata key
`x-crabswarm-id` (D8).

Round 3: **Q6** start-of-work = `UserPromptSubmit`, nudge only `idle` (D6) ·
**QC** `modernc.org/sqlite`, db path via `swarm_db` config (default
`$XDG_STATE_HOME/crabswarm/swarm.db`) (DC) · **Q9** Claude + Codex both this cut,
kind-agnostic core (D9) · **Q11** team = cmdman compose project, same-team-only
messaging (D11).

> Implementation-time confirmations remain (not blocking the plan): exact
> `cmdman query` CLI surface, the Codex hook event→state mapping, and the env name
> cmdman uses to expose the compose-project/team. These are noted inline in the
> relevant steps.
