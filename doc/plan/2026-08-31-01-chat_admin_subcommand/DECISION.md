# Decisions

## AD1: admin verbs move under `chat admin`; old spellings removed (automatic decision)

Chosen: `crabswarm chat admin {list,register,move,send,log}`; delete
`chat register` and `chat team` outright.
Rationale: the admin plane authenticates, addresses, and behaves
differently from members (user directive 2026-08-31: "admin works
differently from other members", "admin will never join as member");
duplicate spellings would have to be kept in sync for no deployed user.
Rejected: `--admin` flag on `chat join` (user rejected); keeping
`chat team` as alias.

## AD2: `admin send` is a new AdminService RPC, not token impersonation (automatic decision)

Chosen: `AdminService.Send(room, target, text)` behind the existing
nonce auth; admin never holds a member token.
Rationale: keeps the two credential planes separate (upstream D7 in
`doc/plan/2026-08-26-01-chat_subcommand/DECISION.md`: admin proves
possession of the age identity "rather than by carrying a token") and
avoids minting throwaway member rows for the operator.
Rejected: auto-registering a hidden admin member; reusing
`RegisterMember` + member `Send` client-side (leaks a member row,
inbox, and nudge target for the admin).

## AD3: admin messages carry the reserved sender name `admin` (automatic decision)

Chosen: from_name `admin`, from_team `admin`, from_room = target room;
`RegisterMember`/join reject the member name `admin`.
Rationale: recipients must be able to tell operator instructions from
peer chatter; reserving the name keeps attribution unforgeable within
a room.
Rejected: free-form `--as NAME` (spoofable, no gain); empty team
rendered specially (needs rendering special-cases everywhere).

## AD4: target grammar = member send grammar plus `*` (automatic decision)

Chosen: `team/name`, bare `name`, bare `team`, and `*` for the whole
room, resolved by the same helper member `Send` uses.
Rationale: one grammar to learn and one resolver to maintain; `*`
covers the broadcast case without a separate admin broadcast verb.
Rejected: separate `admin broadcast` verb.

## AD5: `admin log` verb is owned here, storage owned by plan 05 (automatic decision)

Chosen: this plan owns the `admin log` verb and the admin-authenticated
`AdminService.History` RPC (room-keyed); both land only once
`2026-08-31-05-per_room_message_history` delivers the
`Store.History(ctx, room, limit)` store read they consume — no stub
before that. (Reconciled 2026-08-31 with plan 05's boundary ledger,
which assigns "verb + admin RPC" here; the earlier `ReadLog` name in
this draft was replaced by plan 05's actual `History` naming.)
Rationale: a stub that prints nothing is a worse failure experience
than an absent verb; the boundary ledger keeps ownership visible.
Rejected: implementing history storage inside this plan (duplicates
plan 05's scope).
