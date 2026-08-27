# DECISION — `crabswarm chat` subcommand

One entry per material decision: choice, rationale, rejected alternatives.
Stubs seeded from IDEA.md open questions; each becomes a full entry when the
user resolves it.

### D1. Relation to agent_swarming plan (Q1) — **supersede** [2026-08-26]
- **Choice:** this chat plan supersedes the unimplemented
  `doc/plan/2026-06-27-01-agent_swarming` messenger design; that plan is
  retired.
- **Rationale:** nothing was implemented, and the repo explicitly disclaims
  backward-compatibility concerns; one messaging design should exist, not
  two.
- **Rejected:** live beside it; fold rooms/join into that plan as a
  revision.

### D2. Participants (Q2) — **both, agents primary** [2026-08-26]
- **Choice:** both LLM agent sessions and humans in terminals join rooms;
  agents are the primary audience.
- **Rationale:** the repo exists to swarm agents, but a human joining to
  observe/steer is valuable and cheap if the client works over
  stdin/stdout.
- **Rejected:** agents-only; humans-only.

### D3. Serve placement (Q3) — **extend `crabswarm serve`** [2026-08-26]
- **Choice:** register the chat service on the existing `crabswarm serve`
  daemon (`crabswarm/server/server.go`, same socket, same flock).
- **Rationale:** one process to manage; matches the current single-daemon
  architecture; the user's phrase "serve serves daemon" reads as the
  existing verb.
- **Rejected:** a dedicated `crabswarm chat serve` with its own socket.

### D4. Team semantics & formation (Q4) — **strict scoping; daemon-controlled formation** [2026-08-26]
- **Choice (user's words, near verbatim):** "Strictly team only for now.
  Team formation will only occur outside devenv containers. Only the daemon
  itself can edit team formation, otherwise through a management socket.
  Agents through cmdman compose projects automatically will be put into the
  same team. Same work_dir ⇒ same team."
- **Operative clauses:** (a) message visibility is strictly own-team, no
  room-wide channel for now; (b) participants cannot choose or change their
  team — no `--team` flag; (c) automatic assignment: same cmdman compose
  project / same work_dir ⇒ same team; (d) manual edits to formation happen
  only via a management socket unreachable from inside devenv containers.
- **Rejected:** `--team` chosen at join; room-wide broadcast channel;
  room-wide-by-default with optional teams.

### D5. Interaction model of `join` (Q5) — **attendance + one-shot verbs** [2026-08-26]
- **Choice (user's words, near verbatim):** "Join declares its attendance.
  Messages should be shown through one-shot command. Agents use it through
  just a CLI command with skill. The agents can fetch it proactively but
  often messages are sent to them without notification."
- **Operative clauses:** (a) `chat join` is a one-shot attendance
  declaration, not a long-lived session; (b) reading messages is a one-shot
  command (and sending, symmetrically); (c) agents use the plain CLI guided
  by a bundled skill; (d) because messages arrive while the agent is not
  looking, an arrival-notification mechanism is required — that mechanism
  is its own question (Q9/D9).
- **Rejected:** long-lived streaming session as the primary model; hybrid
  stream-for-humans model.

### D13. Notifier container boundary (C1) — **mount endpoints out** [2026-08-27]
- **Choice:** devenv bind-mounts the Claude Code messaging-socket
  directory to a host path and publishes the OpenCode HTTP endpoint to the
  host, so the daemon's native adapters (claude-socket, opencode-http)
  work in v1; cmdman `send-keys` stays the fallback for Codex/unknown.
- **Rationale:** native push (mid-idle wake) beats keystroke injection in
  reliability and transcript cleanliness; the mounts are devenv's to
  provide, symmetric with how the crabswarm socket is mounted in.
- **Rejected:** keystroke-only v1; nudging via cmdman exec inside the
  container.
- **Note:** exact devenv mount/publish wiring is external to this repo —
  implementation-time confirmation alongside C3.

### D14. Inbox store (C2) — **SQLite-backed** [2026-08-27]
- **Choice:** pending messages are persisted in SQLite (modernc driver,
  as the retired swarm plan chose), surviving daemon restarts.
- **Rationale:** join-then-one-shot-verbs means messages must wait
  server-side; losing them on a daemon restart silently drops
  coordination between agents that don't know to resend.
- **Rejected:** in-memory drain-on-read; in-memory cursor.

### D15. Send semantics (C4) — **directed + room broadcast** [2026-08-27]
- **Choice:** `send <name|team/name> <text>` targets one member;
  additionally a room-wide broadcast verb delivers to every member of the
  caller's room.
- **Rationale:** directed is the D10 baseline; room broadcast covers
  announcements without reintroducing team walls.
- **Rejected:** directed-only; directed + team-only broadcast.

### D16. Human join path (C5) — **admin-registered** [2026-08-27]
- **Choice:** a host-side human (no provider entry) is registered via
  `ChatAdminService.RegisterMember` (age nonce challenge, D7); the daemon
  issues them a token, after which the normal verbs work unchanged.
- **Rationale:** keeps D11's provider gate absolute for unauthenticated
  joins while honoring D2 (humans participate).
- **Rejected:** humans out of scope for v1.

### D17. Proto package naming (C6) — **`ngicks.crabswarm.chat.v1`, rename existing services too** [2026-08-27]
- **Choice (user's words, near verbatim):** "`ngicks.crabswarm.chat.v1`.
  Rename other proto service names to this convention." So: chat lands as
  `ngicks.crabswarm.chat.v1`, and the existing packages are renamed —
  `crabhook.v1` → `ngicks.crabswarm.hook.v1`, `crabpreview.v1` →
  `ngicks.crabswarm.preview.v1`, `sdktypes.v1` (+ `ext/codex`) →
  `ngicks.crabswarm.sdktypes.v1` — with Go and TS codegen, imports, and
  the web frontend updated to match.
- **Rationale:** one vendor-prefixed convention across the schema; the
  repo explicitly disclaims backward compatibility.
- **Rejected:** `crabchat.v1` (perpetuates the old convention);
  `crabswarm.v1`.

### D18. Departure detection (C7) — **Leave + lazy reap** [2026-08-27]
- **Choice:** explicit `chat leave`, plus the daemon drops a member when a
  provider lookup for its token fails during normal operations (send
  resolution, member listing). No timers.
- **Reconciliation with D16:** lazy reap applies only to
  provider-originated members. Admin-registered members (D16) carry
  daemon-issued tokens that never resolve via the cmdman-compose provider;
  token resolution therefore consults the provider *or* the daemon's
  registered-member table, and registered members are removed only by
  explicit Leave or an admin RPC.
- **Rationale:** covers crashed/vanished agents without background
  machinery.
- **Rejected:** explicit-only (stale entries); periodic sweep (timer
  complexity).

### D19. Notifier v1 scope narrowed (user follow-up) — **send-keys only; native adapters spun off** [2026-08-27]
- **Supersedes D13 and narrows D9 for v1.** User's words: "For now we'll
  implement send-keys provider only. Spawn another plan about Channels so
  another session can spike on it."
- **Choice:** v1 ships only the cmdman `send-keys` notifier adapter. The
  `Notifier` interface still lands (D9's pluggability survives), but the
  claude and opencode native adapters, the D13 endpoint mounts, and the
  crabswarm-as-channel-server direction move to a dedicated spike plan:
  `doc/plan/2026-08-27-01-chat_channels_spike`, to be worked by another
  session. Recorded in HANDOFF.md.
- **Rationale:** the Claude messaging-socket frame is unofficial, Channels
  is the documented path but preview-gated, and Codex app-server requires
  owning the session lifecycle — all better explored in a spike than
  blocking chat v1.
- **Rejected (for v1):** mounting endpoints out now (D13's original
  choice); hand-rolled socket/HTTP adapters now.

### D20. send-keys gating: hook-fed member state + snapshot guard [2026-08-27]
- **Problem (user):** send-keys "changes its meaning if the CLI surface is
  displaying AskUserQuestion or a permission request" — blind injection
  can answer a dialog instead of typing a nudge.
- **Verified facts:** Codex hooks give partial state coverage —
  `UserPromptSubmit` (turn start), `PermissionRequest` (fires *before*
  an approval dialog; a hook may even auto-allow/deny), `PostToolUse`
  (dialog resolved), `Stop` + notify `agent-turn-complete` (idle). **No
  event exists for a dialog being answered/dismissed or a generic
  "waiting for input" state.** Claude Code is hook-complete for this:
  its `Notification` hook fires on `permission_prompt`/`idle_prompt`.
- **Choice:** (a) harness hooks report member state to the daemon
  (idle / running / waiting-input) via a `ReportState` RPC; (b) the
  send-keys notifier injects only into **idle** members and drops nudges
  otherwise (turn-end delivery is covered by the Stop-hook inbox
  reinforcement, step 8); (c) immediately before injecting, the notifier
  snapshots the CLI surface via cmdman capture-pane and text-scans for
  dialog markers as the final guard (the user's proposed heuristic),
  covering Codex's event gap and races.
- **Rationale:** hooks are cheap and authoritative where they exist; the
  snapshot guard closes exactly the states hooks cannot see. Revives the
  retired swarm plan's member-state design (its step 7).
- **Rejected:** blind injection; snapshot-only polling (heavier, laggier
  than hook events).

### D12. Duplicate join idempotency (idea-gate feedback) — **same token re-join is a no-op success** [2026-08-27]
- **User's words (near verbatim):** "The current agents system (pressing
  left arrow twice in Claude Code) may trigger 2 or more Start hooks. Join
  likely happens in a Start hook, so a duplicate join with an identical
  token can either silently swallow the error, or clearly reject while
  agents ignore it."
- **Choice:** join is **idempotent per token** — a repeated join with the
  same `$CMDMAN_CMD_ID` returns success without side effects (attendance
  already declared), so hook-triggered duplicates need no special
  handling client-side.
- **Rationale:** of the two behaviors the user allows, a no-op success
  keeps the skill/hook script trivial and produces no scary error output
  in agent transcripts.
- **Rejected:** clear rejection that clients are taught to ignore.

### D11. Provider-gated join (idea-gate feedback) — **no team info, no join** [2026-08-27]
- **User's words (near verbatim):** "Agents inside devenv container can
  join only because the daemon is cmdman-compose-aware; without it, join
  is rejected because there is no prior team coordination information. We
  may add more of these team-info provider types, but that is not
  planned."
- **Operative clauses:** (a) the daemon validates every join against a
  **team-info provider**; a joiner the provider does not know is rejected
  with a clear error; (b) the only provider is the **cmdman-compose
  provider**, which resolves the reported `$CMDMAN_CMD_ID` (D8) to
  work_dir ⇒ room and compose project ⇒ team (D10); (c) the provider
  concept is deliberately extensible, but no additional provider type is
  planned.
- **Consequence for humans (D2):** a host-side human has no provider
  entry; tentative default — they enter through the management-
  authenticated path (age challenge, D7) as an explicitly registered
  member, choosing a room. To revisit at the gate if wrong.

### D10. Room/team semantics revised (idea-gate feedback) — **teams are name namespaces, not walls** [2026-08-26]
- **Supersedes D4's clause (a).** D4's formation clauses (b)–(d) stand.
- **User's words (near verbatim):** "A room is where teams can talk. Teams
  can separate names; if [members] with same name exist between 2 [teams],
  team-prefix must be added to talk to other team's. Nothing other than
  that really separates members in a room. work_dir ⇒ room for cmdman
  compose projects auto team coordination."
- **Operative clauses:** (a) any member of a room can talk to any other
  member of the room — teams do not gate visibility; (b) a team is a name
  namespace: names are unique within a team, may collide across teams, and
  a colliding name is addressed cross-team as `<team>/<name>`; (c) nothing
  besides naming separates members within a room; (d) for cmdman compose
  projects, **work_dir determines the room** automatically, with team
  coordination (compose project ⇒ team) inside it.
- **Rejected (was D4a):** strict own-team-only visibility.

### D6. Transport scope (Q6) — **UDS only** [2026-08-26]
- **Choice:** unix-domain socket only, single host; devenv containers
  reach it via a mounted socket.
- **Rationale:** matches everything else in the repo.
- **Rejected:** additional TCP listener for multi-machine rooms.

### D7. Management access (Q7) — **age identity + nonce challenge** [2026-08-26]
- **User's direction (verbatim):** "Can we use age for pubkey? A cert file
  on disk. The devenv mount does not inherit so separation can be achieved
  by that."
- **Choice:** management RPCs live on the main socket, gated by proof of
  possession of an **age identity file** kept on host disk outside the
  devenv mounts. Since age (X25519) encrypts but does not sign, the proof
  is a challenge–response: the daemon's config names the admin age
  *recipient* (public key); a management call receives a nonce encrypted
  to that recipient; the CLI decrypts it with the identity file and echoes
  it back. Implemented with `filippo.io/age`.
- **Rationale:** mount separation makes the key file naturally
  unreachable from containers; no second socket to configure/mount-guard;
  age is a single small dependency.
- **Rejected:** ssh ed25519 signature scheme (extra key format for one
  fewer round trip); second management-only UDS (no crypto, but another
  path to configure and keep out of mounts).

### D8. Team identity of joiners (Q8) — **$CMDMAN_CMD_ID as token** [2026-08-26]
- **Choice (user's words, near verbatim):** "The devenv forwards
  $CMDMAN_CMD_ID to containers. Agents report CMDMAN_CMD_ID as token. The
  daemon can use it to identify incoming request."
- **Operative clauses:** (a) devenv forwards `$CMDMAN_CMD_ID` into each
  container; (b) the chat client sends it as its identity token on every
  request; (c) the daemon resolves the token to the member — and thus to
  the compose project / work_dir that determines the team (D4).
- **Rejected:** bare client-reported work_dir trusted as-is; per-team
  socket paths.

### D9. Message-arrival notification (Q9) — **adapters + keystroke fallback** [2026-08-26]
- **Choice:** pluggable notifier with per-harness adapters — Claude Code
  via its cross-session messaging socket (`$CLAUDE_CODE_MESSAGING_SOCKET`,
  real idle push), OpenCode via its HTTP server API (real idle push),
  Codex and anything unknown via cmdman `send-keys` keystroke injection
  (the only idle wake for Codex). Turn-boundary hooks (Stop block /
  `additionalContext`) reinforce where supported.
- **Rationale:** research (see notification_mechanisms.md) shows no single
  native mechanism covers all three harnesses; adapters use the
  best-behaved channel each harness offers, with injection as universal
  fallback.
- **Rejected:** keystroke-injection-only (brittle, visible); hooks+polling
  only (idle agents stay unaware).

### C3 resolution — cmdman query surface confirmed from source [automatic] [2026-08-27]
- **Context:** C3 was reserved as an implementation-time confirmation with
  the user; the user is away for this run, so the surface was confirmed
  empirically against the local cmdman source
  (~/gitrepo/github.com/ngicks/cmdman/main) and the installed CLI (v0.0.23).
- **Confirmed facts:**
  - `$CMDMAN_CMD_ID` is set per spawned command to the command's own ID
    (cmdman/config/env.go WithCommandContextEnv).
  - Token → info: `cmdman inspect <ID> --format '{{json .Config}}'`;
    `.dir` is the working directory (⇒ room) and
    `.labels["cmdman.compose.project"]` is the compose project (⇒ team;
    absent for non-compose commands ⇒ provider rejects the token).
  - Nudge: `cmdman send-keys <ID> '<text>' Enter` (default mode; a
    non-key-name token is sent as literal bytes, `Enter` translates to CR).
- **Provider consequence:** a token that `cmdman inspect` cannot resolve,
  or that resolves without a compose-project label, is unknown ⇒ join
  rejected.

### Capture-pane guard fallback — logs --tail text scan [automatic] [2026-08-27]
- **Problem:** the snapshot guard before keystroke injection assumed a
  capture-pane-like CLI. cmdman has an internal VT screen snapshot
  (monitor/terminal_screen.go) but exposes it only through the streaming
  `attach`; there is NO one-shot screen-dump command.
- **Choice:** the send-keys notifier guards with
  `cmdman logs --tail <N>` (recent PTY output playback) and text-scans it
  for dialog markers (permission prompt / question UI strings) before
  injecting; combined with the idle-only state gate this covers the race
  window. The guard sits behind the notifier interface so a true
  screen-snapshot surface (if cmdman grows one) can replace it without
  touching callers.
- **Rejected:** opening an attach stream just to grab the repaint frame
  (long-lived interactive protocol, heavy for a pre-send check); skipping
  the guard entirely (blind injection was already rejected).

### D21. Harness-state vocabulary aligned to cmdman [user] [2026-08-28]
- **Choice:** rename the harness-state vocabulary everywhere to mirror
  `cmdman status set working|waiting|done`: `running`→`working`,
  `waiting_input`→`waiting`, `idle`→`done`. Applies to the proto enum
  (`HARNESS_STATE_WORKING=1|WAITING=2|DONE=3`, renumbered to cmdman's
  order), the store's `MemberState` strings, the CLI words for
  `chat report-state`, the hook definitions, and docs. The derived flag
  `chat read --idle-when-empty` became `--done-when-empty` (judgment
  call: the flag name is part of the same vocabulary).
- **Rationale:** user directive 2026-08-28; one shared vocabulary across
  cmdman and crabswarm chat. Renumbering is safe per the repo's
  no-backward-compatibility rule (never deployed).
- **Rejected:** keeping the old wire values while renaming only the CLI
  words (two vocabularies, one per layer); migrating stored rows (no
  deployment exists to migrate).

### D22. Chat store queries generated with sqlc [user] [2026-08-28]
- **Choice:** the chat store's SQL is generated by sqlc (engine `sqlite`,
  `database/sql` codegen, modernc driver unchanged): `crabswarm/chat/
  schema.sql` is the single source of truth (runtime DDL via `go:embed`
  and query typing), `queries.sql` holds the 12 named queries, and the
  generated package lives at `crabswarm/chat/internal/chatdb` (regen via
  `go generate`). All queries proved static; no hand-written SQL remains.
- **Rationale:** user directive 2026-08-28 — the user intended sqlc from
  the start, but D14 never captured it; recorded now and enacted
  post-implementation.
- **Rejected:** keeping hand-written `database/sql` queries (typo-prone
  string SQL, schema/query drift unchecked); leaving dynamic queries
  hand-written (none turned out dynamic).

### D22 amendment — sqlc layout gathered under internal/schema [user] [2026-08-28]
- **Choice:** the flat crabswarm/chat/{schema,queries}.sql + sqlc.yaml
  layout from D22's first cut is replaced by directory-style config:
  `crabswarm/chat/internal/schema/` holds `sqlc.yaml`, `ddl/*.sql`, and
  `queries/*.sql` (sqlc reads whole directories in filename order, so
  new .sql files are drop-ins needing no config edit); generated code
  moved to `crabswarm/chat/internal/db` (package `db`), replacing
  `internal/chatdb`. The schema package owns the `go:embed` (go:embed
  cannot cross package dirs) and exposes `DDL()` as the runtime source
  of truth; the `go:generate` directive lives beside sqlc.yaml.
- **Rationale:** user directive — "all things under this dir"-styled
  config tolerates forward changes better than enumerating files.
- **Rejected:** flat files in the package dir (the first cut);
  `internal/schema/schema/` nesting (reads badly; `ddl/` chosen).

### D23. Token resolver and nonce auth split into subpackages [user] [2026-08-28]
- **Choice:** two concerns moved out of the flat chat package so the root
  can be skimmed: the token resolver (`crabswarm/chat/resolver`:
  `TeamInfo`, `ErrUnknownToken`, `CmdmanCompose` — formerly provider.go)
  and the age-nonce challenge (`crabswarm/chat/auth`: `Nonces`,
  `EncryptNonce`/`DecryptNonce` — formerly inside admin.go, whose
  `AdminService` keeps only RPC glue). Both plain subpackages, not
  internal/ (server wiring constructs the resolver; chat/cli decrypts
  nonces). `TeamInfoProvider` stays declared at its consumer in
  service.go per the interfaces-at-the-consumer rule, keeping resolver a
  stdlib-only leaf. Renames: `CmdmanComposeProvider`→`resolver.CmdmanCompose`,
  `chat.DecryptNonce`→`auth.DecryptNonce`; `resolver.ValidateToken` and
  `resolver.DefaultCmdmanBin` newly exported for the notifier's shared
  cmdman surface.
- **Rationale:** user directive 2026-08-28 — package-root skimmability.
- **Rejected:** `internal/` placement (outside importers exist; auth kept
  plain for symmetry); naming both packages "provider"; moving the
  consumer interface down into resolver (would invert the dependency).

### D24. Member state mirrored onto cmdman status [user] [2026-08-28]
- **Choice:** a `StatusMirror` interface consumed by the Service (Notifier
  pattern; `NopStatusMirror` default) publishes member state to cmdman:
  `cmdman status set <state> <token> --detail "crabswarm chat"` on Join
  and ReportState, `cmdman status delete <token>` on Leave. The D21
  vocabulary crosses unmapped. Adapter `CmdmanStatusMirror` lives in
  crabswarm/chat/status.go beside SendKeysNotifier (write-side cmdman
  adapters sit with their consumer; resolver stays read-only) and guards
  itself: non-agent members skipped, `resolver.ValidateToken` before any
  argv, best-effort with detached 3s-timeout writes (Warn on failed Set,
  Debug on failed Clear — the command is ordinarily gone by Leave).
  Reap publishes nothing (the cmdman command is already gone).
  Idempotent re-Join republishes the *stored* state so a restarted
  cmdman regains the display. `NewService` gained a `mirror` parameter
  (breaking; all callers updated).
- **Rationale:** user directive 2026-08-28 — operators watching cmdman
  should read the same word the room holds.
- **Rejected:** mirroring from the resolver package (second
  responsibility in a read-only package); failing RPCs on mirror errors
  (best-effort like the notifier); mapping table between vocabularies
  (D21 made them identical).
