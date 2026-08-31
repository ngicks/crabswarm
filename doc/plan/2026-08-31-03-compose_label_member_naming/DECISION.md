# Decisions

## D1 — Name comes through the resolver seam (automatic decision)

**Choice**: `TeamInfo` gains a `Name` field filled by `CmdmanCompose.Resolve`
from the already-decoded labels map; `Service.Join` prefers it over
`defaultName(token)`.

**Rationale**: the labels are in the JSON the resolver already fetches, so no
extra cmdman invocation; and the resolver package doc declares it the one
place crabswarm keeps cmdman knowledge.

**Rejected**: a second shell-out from the chat service (extra process, leaks
cmdman surface knowledge across the package boundary); changing the CLI/hook
to pass `--name` (would put derivation in every harness wiring instead of one
daemon-side spot).

## D2 — Alias format `<command>-<scale-index>`, degrading gracefully (automatic decision)

**Choice**: both labels present → `<command>-<scale-index>`; scale index
absent → `<command>`; command label absent → empty `Name`, and the join falls
back to the existing `agent-<hex8>` default rather than failing.

**Rationale**: the user mandated the two labels
(`cmdman.compose.command`, `cmdman.compose.scale-index`) as the source; a
missing label must not turn a valid join into an error since naming is a
convenience, not placement.

**Rejected**: erroring on missing labels (breaks joins for older cmdman or
non-scaled commands); including team/project in the name (team is already a
separate namespace column).

## D3 — Collisions stay a clear rejection (automatic decision, inherited)

**Choice**: no collision handling change. Quoting the operative rule from
`doc/plan/2026-08-26-01-chat_subcommand/IDEA.md:143-145`: "duplicate
participant name in a team → clear rejection at join time, not silent
aliasing".

**Rationale**: the scale index makes compose replicas of one command unique
by construction; anything else colliding is a real configuration problem the
operator should see.

**Rejected**: suffixing duplicates automatically (explicitly ruled out by the
inherited decision).

## D4 — Label values used verbatim, no sanitizing (automatic decision)

**Choice**: the derived name is the raw label content joined with `-`; an
unusable name surfaces as the store's existing join-time rejection.

**Rationale**: compose file authors control these values; silent rewriting
would make the displayed name diverge from the compose file, which defeats
the recognizability goal.

**Rejected**: character filtering / truncation in the resolver.

## D5 — No migration for already-stored names (automatic decision)

**Choice**: members joined before the change keep their stored name until
they leave and rejoin (`Service.Join` returns the existing membership
unchanged by design).

**Rationale**: the store deliberately keeps the first join
(`crabswarm/chat/service_member.go:31-33`), the app has never been deployed
(repo rule: don't over-weigh backward compatibility), and a room restart
naturally refreshes everything.

**Rejected**: renaming existing members on rejoin (would break in-flight
addressing and violate the idempotent-rejoin decision D12 of the chat
subcommand plan).

## D6 — Goal directive taken as the IDEA.md gate confirmation [automatic]

**Choice**: proceed with implementation; the user's /goal instruction to
implement this plan is treated as approval of the IDEA.md gate and the
automatic decisions D1-D5.

**Rationale**: the user set the goal and declared themselves away for the
run; stalling on the gate would contradict the directive.

## D7 — Plan-file updates land in the branch worktree copy [automatic]

**Choice**: STATUS.md / DECISION.md updates during this run are made in the
`compose_label_member_naming` worktree's copy of the plan directory and
committed with the branch, not in main's copy.

**Rationale**: the branch copy travels with the implementation commits and
merges back; editing both copies would drift.

## D8 — Recreate-collision fix deferred to user decision [automatic]

**Choice**: the review-found hard failure (a recreated compose replica
cannot rejoin because the stale member permanently holds its derived name;
full chain in HANDOFF.md) is deferred, not fixed in this run.

**Rationale**: every fix is new collision policy (stale-member eviction, or
fallback naming that is silent aliasing in spirit), and the plan explicitly
declared collision semantics a non-goal — "clear rejection stays". With the
user away, overriding a written non-goal is a scope change reserved for
them; the shipped failure mode is a loud AlreadyExists at join time, and
the app has never been deployed.

**Rejected (for this run)**: evict-on-unresolvable-token; fallback to the
token-derived name on derived-name collision.
