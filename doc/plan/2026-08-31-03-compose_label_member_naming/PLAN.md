# Default member name from cmdman compose labels

Derive the default chat member name from cmdman-compose's
`cmdman.compose.command` + `cmdman.compose.scale-index` labels instead of the
token prefix.

## Goal / success criteria

- A member joining without `--name` from a compose-managed command is named
  `<command>-<scale-index>` (or `<command>` when no scale index is stamped).
- Explicit `--name` still overrides; label-less commands still fall back to
  `agent-<hex8>`.
- Covered by resolver unit tests and an e2e join test asserting the derived
  name.

## Scope

- `crabswarm/chat/resolver` — surface the name material.
- `crabswarm/chat` — prefer the resolver-provided name in `Service.Join`.

## Non-goals

- No change to explicit `--name` handling or collision semantics (clear
  rejection stays; see IDEA.md).
- No change to the admin `RegisterMember` path (human names are always
  explicit there).
- No new cmdman invocation — the labels are already in the JSON the resolver
  decodes.

## Context

- `crabswarm/chat/service.go:270-277` — `defaultName(token)` returns
  `"agent-" + token[:8]`; called from `Service.Join`
  (`crabswarm/chat/service_member.go:69-72`) when `req.GetName()` is empty.
- `crabswarm/chat/resolver/cmdman.go:89-109` — `CmdmanCompose.Resolve` runs
  `cmdman inspect <token> --format '{{json .Config}}'` and decodes only
  `dir` + `labels`, reading `cmdman.compose.project` for the team. The two
  naming labels sit in the same map, so no extra process run is needed.
- `crabswarm/chat/resolver/resolver.go:25-33` — `TeamInfo{Room, Team}` is the
  resolver's whole output; it has no name field today.
- Verified live: `cmdman inspect $ID --format
  '{{index .Config.Labels "cmdman.compose.command"}}'` and
  `'{{index .Config.Labels "cmdman.compose.scale-index"}}'` return the
  command name and replica index (cmdman v0.0.24).

## Approach

Carry the derived name through the existing resolver seam: `TeamInfo` gains a
`Name` field that `CmdmanCompose.Resolve` fills from the two labels, and
`Service.Join` prefers it over `defaultName(token)`. Rejected alternative: a
second cmdman shell-out from the chat service (extra process run, and it
would put cmdman knowledge outside the resolver package, which the package
doc explicitly forbids).

Name derivation inside `Resolve`:

- `cmdman.compose.command` empty or absent → `TeamInfo.Name` stays `""`
  (join falls back to the token-derived default, not an error).
- `cmdman.compose.scale-index` empty or absent → `Name = <command>`.
- Both present → `Name = <command>-<scale-index>`.

The label values are used verbatim; a bad name fails at the store's existing
join-time rejection rather than being silently sanitized.

## Public surface delta

```go
// crabswarm/chat/resolver/resolver.go
type TeamInfo struct {
	Room string
	Team string
	// Name is the display name the provider derives for the token's holder
	// (compose: "<command>-<scale-index>"). Empty when the provider has no
	// naming information; the chat service then falls back to its own default.
	Name string // added
}
```

No CLI flag, config key, RPC message, or persisted-schema change: the derived
name lands in the existing `members.name` column and `Member.name` proto
field. `chat join --name` semantics are unchanged.

## Implementation steps

1. **resolver: carry the name.** Add `Name` to `TeamInfo`
   (`crabswarm/chat/resolver/resolver.go`); add label constants
   `composeCommandLabel = "cmdman.compose.command"` and
   `composeScaleIndexLabel = "cmdman.compose.scale-index"` beside
   `composeProjectLabel` and derive `Name` in `CmdmanCompose.Resolve`
   (`crabswarm/chat/resolver/cmdman.go:100-109`) per the rules above.
   Verify: new cases in `crabswarm/chat/resolver/cmdman_test.go` (stub JSON
   with both labels, command-only, neither).
2. **chat: prefer the resolver name.** In `Service.Join`
   (`crabswarm/chat/service_member.go:69-72`), resolve the default as
   `req.Name` → `info.Name` → `defaultName(token)`. Update the doc comment on
   `defaultName` (`crabswarm/chat/service.go:270`) — it is no longer "the
   only thing about it the daemon knows".
   Verify: `crabswarm/chat/service_member_test.go` — adjust the
   `agent-01234567` expectation (line 73) per what the stub provider returns,
   and add a case where the provider supplies a name.
3. **e2e coverage.** Extend the join flow in `e2e/crabswarm/chat_test.go` (or
   a sibling) so a member joining without `--name` under a compose-labeled
   stub is listed as `<command>-<scale-index>`.

## Testing / verification

- `go test ./crabswarm/chat/... ./e2e/...` — resolver unit tests, service
  join tests, e2e.
- Manual: `cmdman compose` a two-replica command, hook-join both, check
  `crabswarm chat members` shows `<command>-1` / `<command>-2`.

## Risks

- Label value vocabulary: if compose ever stamps a scale index on single
  commands as `0` vs omitting it, names shift between `<command>` and
  `<command>-0`. Cosmetic, and pinned down by the resolver tests against the
  actual stub shapes.
- Stored names persist: members joined before the change keep their
  `agent-<hex8>` name until they leave and rejoin (store keeps the first
  join). Accepted; no migration.

## Open questions

None — resolved automatically per user directive; see DECISION.md.
