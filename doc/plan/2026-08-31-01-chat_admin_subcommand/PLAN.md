# Plan: group the admin plane under `crabswarm chat admin`

Move the age-authenticated admin verbs into their own `chat admin`
subcommand group and give the admin a way to speak into a room and read
its history without ever being a member.

## Goal / success criteria

- `crabswarm chat admin {list,register,move,send,log}` exists; every
  room-scoped verb takes the room as an explicit positional argument.
- `chat register` and `chat team` are gone from the tree.
- `admin send` delivers into member inboxes attributed to the reserved
  sender `admin`, triggers the normal notify path, and creates no
  member row.
- All admin verbs authenticate by the existing age challenge-response;
  none accept or use a token.

## Scope

- CLI tree under `cmd/crabswarm/commands/` and presentation in
  `crabswarm/chat/cli/`.
- `AdminService` (proto + `crabswarm/chat/admin_rooms.go`): one new
  `Send` RPC. `Log`/history read is surfaced here but its storage and
  RPC are owned by sibling plan `2026-08-31-05-per_room_message_history`.

## Non-goals

- No admin TUI (sibling plan `2026-08-31-06-admin_tui` builds on this).
- No MCP surface (sibling plan `2026-08-31-02-chat_mcp_server`).
- No change to member auth, join, or nudge gating.
- No backward-compatible aliases for the removed spellings.

## Context

- Command tree: `cmd/crabswarm/commands/chat.go:53-61` registers member
  and admin verbs flat; `chat_register.go`, `chat_team.go`,
  `chat_team_list.go`, `chat_team_move.go` are the admin ones.
- Admin auth: nonce challenge in `crabswarm/chat/admin.go`
  (`authenticate`, `GetNonce`) and client side in
  `crabswarm/chat/cli/admin.go` (`nonce`, `ResolveIdentityPath`).
- Admin RPCs: `ListRooms`, `MoveMember`, `RegisterMember` in
  `crabswarm/chat/admin_rooms.go`; proto `service AdminService` in
  `api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto:56-63`.
- Member delivery + notify: `crabswarm/chat/service_inbox.go` (member
  `Send`/`Broadcast`), notify gating in `crabswarm/chat/notify/notify.go`.
- Design record being amended: D7/D11 in
  `doc/plan/2026-08-26-01-chat_subcommand/DECISION.md` (admin auth is
  age challenge; non-cmdman participant = registered human). This plan
  keeps both; it adds "admin can speak without membership".

## Approach

Cobra-side this is a re-grouping: a new `chatAdminCmd(parent, flags)`
registers the five verbs on an `admin` subcommand, reusing the existing
run functions for list/move/register. Service-side the only new logic
is `AdminService.Send`: authenticate, resolve the target inside the
named room with the same address grammar member send uses, insert into
each recipient's inbox with the reserved sender identity, and fire the
existing notifier. Delivery is shared with member send by extracting
the deliver-and-notify step out of `Service` into a helper both
services hold, rather than AdminService importing Service.

Rejected alternatives:
- `--admin` flag on `chat join` — rejected by the user: admin never
  joins as a member.
- Keeping `chat team` alongside `chat admin` — duplicate spellings for
  one plane; repo has no deployed users to keep compatibility for.

## Public surface delta

```console
# added
crabswarm chat admin list                              # was: chat team list
crabswarm chat admin register ROOM TEAM NAME           # was: chat register --room --team --name
crabswarm chat admin move ROOM TEAM/NAME TO_TEAM       # was: chat team move
crabswarm chat admin send ROOM TARGET TEXT             # new; TARGET: team/name | name | team | '*'
crabswarm chat admin log ROOM [--limit N]              # new; delivered by plan 05's storage

# removed
crabswarm chat register
crabswarm chat team list
crabswarm chat team move
```

```proto
// api/schema/proto/ngicks/crabswarm/chat/v1/chat_service.proto
service AdminService {
  // existing: GetNonce, ListRooms, MoveMember, RegisterMember
  rpc Send(AdminSendRequest) returns (AdminSendResponse);
  // History reads a named room's log, admin-authenticated. Owned here
  // (step 6); the storage + room-keyed Store.History read it consumes
  // are delivered by plan 2026-08-31-05-per_room_message_history.
  rpc History(AdminHistoryRequest) returns (AdminHistoryResponse);
}

message AdminSendRequest {
  string room = 1;
  string target = 2; // same grammar as member SendRequest recipient, plus "*"
  string text = 3;
}
message AdminSendResponse {
  int32 delivered = 1; // recipients reached
}
```

```go
// crabswarm/chat/admin_rooms.go
func (a *AdminService) Send(ctx context.Context, req *chatv1.AdminSendRequest) (*chatv1.AdminSendResponse, error)

// crabswarm/chat/cli (client + rendering)
func (c *Client) AdminSend(ctx context.Context, w io.Writer, identityPath, room, target, text string) error
```

Durable vocabulary: sender identity of admin messages is the reserved
name `admin` (from_name `admin`, from_team `admin`, from_room = target
room); member registration rejects the name `admin` so the attribution
stays unambiguous.

## Implementation steps

1. **Proto**: add `Send` + messages to `AdminService` in
   `chat_service.proto`; regenerate (`go generate ./api/...`).
2. **Delivery helper**: extract the insert-inbox-and-notify step used by
   member `Send`/`Broadcast` in `crabswarm/chat/service_inbox.go` into a
   shared helper; verify member tests still pass.
3. **`AdminService.Send`** in `crabswarm/chat/admin_rooms.go`:
   authenticate, validate non-empty room, resolve `target` (reuse the
   member resolution; `*` = every member of the room), deliver with the
   reserved `admin` sender, return delivered count. Reject registration
   of the member name `admin` in `RegisterMember`/store join.
4. **CLI client**: `AdminSend` + rendering in `crabswarm/chat/cli/`.
5. **Cobra re-group**: new `chat_admin.go` (+ per-verb files) under
   `cmd/crabswarm/commands/` registering `admin
   {list,register,move,send}`; delete `chat_register.go`,
   `chat_team*.go`; update `chat.go`'s long help text (its admin-verbs
   paragraph names register/team).
6. **`admin log`**: implement `AdminService.History` (room-keyed, over
   plan 05's `Store.History(ctx, room, limit)`) and register the verb —
   blocked on plan 05 step 3; no stub before that.
7. **e2e**: extend `e2e/crabswarm/chat_test.go` — admin send reaches a
   member inbox with `admin` attribution and no new member row; removed
   spellings return unknown-command.

## Boundary ledger

| Deliverable | Owner |
| --- | --- |
| `chat admin` group, list/register/move/send verbs | this plan |
| Reserved `admin` sender identity + name rejection | this plan, step 3 |
| Room history storage + `Store.History` room-keyed read | plan 2026-08-31-05-per_room_message_history |
| `chat admin log` verb + `AdminService.History` RPC | this plan, step 6 (blocked on plan 05 step 3) |
| Live watch / TUI over rooms | plan 2026-08-31-06-admin_tui |
| MCP exposure of admin state/resources | plan 2026-08-31-02-chat_mcp_server |

## Testing and verification

- Unit: `AdminService.Send` (auth required, unknown room/target,
  `*` fan-out, delivered count) beside `admin_rooms_test.go`.
- e2e per step 7; `go test ./...` for the regrouping fallout
  (`chat_test.go` exercises the old spellings today).

## Risks

- Target-grammar drift between member send and admin send — mitigated
  by reusing one resolution helper (step 3).
- The delivery-helper extraction (step 2) touches the member hot path;
  keep it a mechanical move guarded by existing inbox tests.

## Open questions

None — resolved automatically; see DECISION.md (all entries tagged
"automatic decision").
