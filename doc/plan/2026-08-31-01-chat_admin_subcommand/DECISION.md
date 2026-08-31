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

## AD6: run executed unattended; idea gate treated as confirmed [automatic]

The /goal directive "implement this plan" is taken as the idea-gate
confirmation; the run proceeds in away mode, deciding unclear corners
autonomously and tagging them [automatic]. The `History` RPC is NOT
added to the proto in this run (step 6 skipped, blocked on the per-room
message-history plan); only `Send` lands.

## AD7: buf lint standard-name exemption for admin Send messages [automatic]

buf's STANDARD lint demands `SendRequest`/`ChatAdminServiceSendRequest`
naming, but the member service already owns `SendRequest`. Kept the
planned `AdminSendRequest`/`AdminSendResponse` names and added
`RPC_REQUEST_STANDARD_NAME`/`RPC_RESPONSE_STANDARD_NAME` exemptions in
api/buf.yaml scoped to chat_service.proto. Inline `// buf:lint:ignore`
was tried and rejected: buf leaks those lines verbatim into generated
TS/Go docs.

## AD8: admin send target grammar excludes bare `team` [automatic]

AD4 listed `team/name | name | team | *`, but the member resolver
(crabswarm/chat/member.go resolveFor) supports only `team/name` and bare
`name`; bare-team fan-out does not exist on the member path. Adding it
only to admin send would fork the grammar AD4 wants shared, and adding
it to both paths needs a name-vs-team collision rule. Shipped
`team/name | name | *`; bare-team fan-out deferred to HANDOFF.md.

## AD9: chat group parents get cobra.NoArgs so removed spellings fail [automatic]

`chat register`/`chat team ...` after the regroup hit cobra's legacy
group-parent behavior: print help, exit 0. The plan's e2e wants an
unknown-command failure. Fix scoped to the chat tree only (`chat` and
`chat admin` parents get `Args: cobra.NoArgs`); making that a repo-wide
convention for other group commands (git/preview) is left to the user.
