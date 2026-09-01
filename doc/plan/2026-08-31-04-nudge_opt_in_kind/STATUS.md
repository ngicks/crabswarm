# Status — nudge opt-in kind for members

Current state: **implemented 2026-09-02** on branch `worktree-nudge-opt-in`.
The user's implement-all-plans directive superseded the D1 deferral
(DECISION.md D1 Amendment); the automatic decisions D2 and D3 were carried out
as drafted, D3 with the retarget D5 records.

Next action: re-review and merge. The review of 2026-09-02 returned
needs-changes on the doc sweep above and on a test that declared no agent (so
the TTL cache it names went unexercised); both are fixed. `go test ./...`
passes.

## Checklist

- [x] Step 1 — proto `JoinRequest.agent` field + regenerate (D2: kind chosen
      from the join request)
- [x] Step 2 — `Service.Join` maps flag to `KindAgent`/`KindHuman`; kind doc
      comments updated (D2: "represent opt-in via the existing MemberKind").
      The same change falsified the provenance rationale wherever it was
      written down. The first pass caught `internal/cmdman/cmdman.go`,
      `status.go`'s guard and the two liveness doc comments in `service.go`;
      the review of 2026-09-02 found four more — `status.go`'s struct doc,
      `Member.Token` and `Member.Kind` in `store.go`, and `TokenMetadataKey`
      in `interceptor.go` — plus the parent `chat` command's help and five
      test comments. All of them now read off capability.
- [x] Step 3 — `chat join --agent` CLI flag threaded through
      `chatcli.Client.Join` (D1 quote: "`--agent` means notification. But
      without it, there shouldn't be notification."), and the MCP bridge's
      auto-join declares `agent=true` (D5)
- [x] Step 4 — **retargeted**: the plan said to append `--agent` to the
      apm-package SessionStart join hook, but that hook no longer exists — the
      chat MCP-server work removed it, leaving the stdio bridge as the only
      automatic join. Verified by grep: no `chat join` remains in
      `.apm/hooks/report-state.json`, the skill, or `apm.yml`. The bridge
      carries the declaration instead (step 3), and the README's two
      manual-join sentences now spell `--agent`. See D5.
- [x] Step 5 — unit + e2e tests: no-flag member never receives send-keys;
      `--agent` member does

## Known consequences (intended)

- A no-flag member drops out of the cmdman status display: `CmdmanStatusMirror`
  gates on `KindAgent`, and a member that reports no harness state has none to
  show.
- A no-flag member is never reaped by the provider-liveness check and keeps its
  name against a colliding joiner, since `checkLiveness` is agent-only.
- Re-joining does not change an established kind: the store keeps the first
  join, so a member that joined flagless stays inbox-only until it leaves.
