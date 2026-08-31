# Nudge opt-in kind for members — implementation plan

Make keystroke-injection nudges opt-in via `chat join --agent`; without the
flag a member is inbox-only and never typed at.

Status: deferred by user (DECISION.md D1) — do not implement until the user
re-schedules it.

## Goal / success criteria

- A member joined without `--agent` never receives keystroke injection: the
  notify path declines before `cmdman send-keys` runs.
- A member joined with `--agent` behaves exactly as agents do today.
- Hook-wired harnesses (apm-package/crabswarm-chat) keep working unchanged
  from the agent's point of view.

## Non-goals

- Admin participation (covered by the `chat admin` plan family; per the user,
  admin never joins as a member).
- Detecting harness-vs-shell automatically (heuristics on the process tree
  were rejected; the joiner declares it).

## Context

- `crabswarm/chat/service_member.go:78` — `Service.Join` stores every
  provider-resolved member as `Kind: KindAgent` unconditionally.
- `crabswarm/chat/internal/cmdman/cmdman.go:103` — `Terminal.SendCommand`
  refuses to type unless `member.Kind == chat.KindAgent` (deliberately not
  `!= KindHuman`; unknown kinds are also not typed at).
- `crabswarm/chat/notify/notify.go:58-81` — `SendKeys.Notify` gates on
  `State == StateDone` and dialog markers; kind gating happens in cmdman.go.
- `crabswarm/chat/store.go:42-45` — `KindAgent` ("provider-originated agent
  session") / `KindHuman` ("registered through an admin RPC").
- `api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto:112-118` —
  `JoinRequest` has `name = 1`, `reserved 2, 3`.
- `apm-package/crabswarm-chat/.apm/hooks/report-state.json` — SessionStart
  runs `chat join`; this is where `--agent` gets added for harnesses.

## Approach

Reuse `MemberKind` as the nudge gate instead of adding a parallel flag or
column: `--agent` → `KindAgent`, absent → `KindHuman`. Every existing guard
(`cmdman.go:103`, `status.go:88`, `service.go:185`) already branches on
exactly this distinction, so the change concentrates in `Service.Join` plus
the CLI/proto surface. See DECISION.md D2 for rejected alternatives.

## Public surface delta

```proto
// api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto
message JoinRequest {
  string name = 1;
  reserved 2, 3;
  // Agent declares the caller is an agent harness whose terminal may be
  // nudged by keystroke injection while idle. Without it the member is
  // inbox-only and is never typed at.
  bool agent = 4;
}
```

```console
# cmd/crabswarm/commands/chat_join.go
crabswarm chat join [--name NAME] [--agent]
```

```go
// crabswarm/chat/cli/member.go
func (c *Client) Join(ctx context.Context, w io.Writer, token, name string, agent bool) error

// crabswarm/chat/store.go — doc change only, values unchanged:
// KindAgent: an agent harness; nudgeable by keystroke injection.
// KindHuman: any other member (plain shell, admin-registered); inbox-only.
```

```json
// apm-package/crabswarm-chat/.apm/hooks/report-state.json (SessionStart)
"crabswarm chat join --agent"
```

## Implementation steps

1. Proto: add `bool agent = 4` to `JoinRequest` in
   `api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto`; run the
   repo's `go generate` in `api/` to refresh Go and TS stubs.
2. `Service.Join` (`crabswarm/chat/service_member.go`): choose
   `KindAgent`/`KindHuman` from `req.GetAgent()`; update the doc comments on
   the kind constants in `crabswarm/chat/store.go:42-45`.
3. CLI: `--agent` flag in `cmd/crabswarm/commands/chat_join.go`, threaded
   through `chatcli.Client.Join` in `crabswarm/chat/cli/member.go`.
4. Hook wiring: append `--agent` to the SessionStart join command in
   `apm-package/crabswarm-chat/.apm/hooks/report-state.json` (both claude and
   codex targets if split).
5. Tests: unit in `crabswarm/chat/service_member_test.go` (kind chosen per
   flag); e2e in `e2e/crabswarm/chat_test.go` asserting a no-flag member
   receives no send-keys while an `--agent` member does (extend the existing
   notify stubs).

## Testing and verification

- `go test ./crabswarm/chat/...` and `go test ./e2e/...`.
- Manual: join from a plain shell without the flag, have another member send
  to it, verify nothing is typed into the shell and `chat read` shows the
  message.

## Risks

- Existing deployments' hooks that join without `--agent` silently lose
  nudges. Acceptable per repo rule "don't think too much about backward
  compatibility"; the apm package update ships in the same change.
- `CmdmanStatusMirror` (`crabswarm/chat/status.go:88`) also gates on
  `KindAgent`, so no-flag members drop out of cmdman status display — that is
  correct (a plain shell has no harness state) but worth noting in review.

## Open questions

(none — resolved automatically; see DECISION.md)
