# Decisions — admin TUI

All entries below are tagged **(automatic decision)**: drafted without
user confirmation per the user's 2026-08-31 directive ("Skip all
questions; I'll use your recommendation for now, so tag it automatic
decision"). Each is revisitable at plan review.

## D1 — TUI framework: charm.land/bubbletea/v2 (automatic decision)

Chosen: bubbletea v2 (with its viewport component for the conversation
pane). Rationale: the repository has no TUI stack yet, and the ngplan
skill's own guidance names `charm.land/bubbletea/v2` as the Go TUI
default; it is the de-facto standard and its Elm-style model suits a
poll-driven watch screen. Rejected: tview (heavier widget model, less
idiomatic here), raw termbox/ansi (hand-rolling layout and input for no
benefit).

## D2 — Live tail by cursor polling, no streaming RPC in v1 (automatic decision)

Chosen: the TUI polls the room-log read RPC with a since-id cursor
(~1s), plus a slower roster poll (~5s), in v1; it switches to
`ChatService.WatchRoom` once plan `2026-08-31-02-chat_mcp_server`
delivers it. Rationale: polling needs no new RPC surface and keeps
this plan decoupled from plans 02/05 landing order. (Reconciled
2026-08-31: this entry originally claimed a streaming RPC "would poll
internally anyway" — that is wrong; the daemon owns every write, and
plan 02's D4 builds `WatchRoom` on an in-daemon per-room fan-out with
no polling. `WatchRoom` is therefore the committed upgrade path, not a
hypothetical.) Rejected: blocking this plan on `WatchRoom` (couples
TUI delivery to the MCP plan), tailing the DB file directly (bypasses
the auth plane).

## D3 — `--room` required; no picker screen (automatic decision)

Chosen: `chat admin tui --room R` errors without `--room`; unknown
rooms error listing existing rooms. Rationale: the user stated the
admin "always needs to specify room id"; a picker would contradict
that and add a screen with no requested use case. Rejected: interactive
room picker on launch, defaulting to the sole room when only one
exists (implicit magic).

## D4 — Single-screen layout: conversation viewport + roster sidebar + input/status (automatic decision)

Chosen: one screen, conversation dominant, roster collapsible on
narrow terminals, one input line reusing `chat send`'s `to: text`
addressing. Rationale: watching is the primary mode (IDEA.md); tabs or
multi-screen navigation add chrome without an oversight use case.
Rejected: multi-room tab view (admin can run one TUI per pane under
cmdman), separate send dialog.

## D5 — Code lives in `crabswarm/chat/cli/tui`; `./cmd` stays thin (automatic decision)

Chosen: new package `crabswarm/chat/cli/tui` holding all presentation;
`cmd/crabswarm/commands/chat_admin_tui.go` only parses flags, dials as
admin, and calls `tui.Run`. Rationale: repo design rule — no
CLI-presentation logic under `./cmd`; terminal control belongs in a
`cli/` package. Rejected: package directly under `crabswarm/chat/tui`
(splits presentation across two roots).

## D6 — No presentation preview in the drafting pass (automatic decision)

Chosen: the ngplan preview mock (bubbletea layout demo with fixture
data + MOCK_LIMITS) is planned as implementation step 1, not created
while drafting. Rationale: the user asked for plan drafts only; the
layout has an IDEA.md flowchart for review, and a runnable mock is
cheap to produce when implementation starts. Rejected: drafting the
mock now (premature while sibling contracts may still move).

## D7 — The preview mock is excluded by a build tag (automatic decision)

Chosen: the mock lives at `mock/main.go` under the plan directory
behind `//go:build tuimock`, run with
`go run -tags tuimock ./doc/plan/2026-08-31-06-admin_tui/mock`.
Rationale: `go build ./...` skips a package whose files are all
excluded, while `go mod tidy` still sees the imports — a directory
named to be ignored by the go tool instead (a leading `_`) would have
had tidy drop the dependency. Rejected: a `_`-prefixed directory (tidy
strips the dependency), no exclusion at all (a throwaway in the
module's build).

## D8 — Run takes bubbletea program options (automatic decision)

Chosen: `Run(ctx context.Context, deps Deps, opts ...tea.ProgramOption)
error`, the caller's options applied after Run's own. Rationale: the
e2e drives the screen against a live daemon, and a terminal program
under bare pipes cannot be handed an input and an output from outside
its own process — bubbletea opens /dev/tty when its input is not a
terminal. The variadic tail leaves the signature the plan states intact
for every other caller. Rejected: input/output fields on `Deps` (mixes
what the screen needs from the daemon with how it is driven), a
separate exported test entry point (a second Run to keep in step).

## D9 — cli gains an identity-bound admin client (automatic decision)

Chosen: `Client.Admin(identityPath)` returns an `AdminClient` with
`Rooms`, `RoomLog` and `Send`, which answer with what the daemon said
rather than rendering it; the existing `ListRooms`, `AdminLog` and
`AdminSend` render on top of those. It is what satisfies the screen's
three interfaces. Rationale: the existing verbs write to an
`io.Writer`, and a screen that had to parse rendered lines would make
that wording an interface. Binding the identity once also spares every
caller from carrying it. Rejected: structured methods taking the
identity per call (the screen would hold a path it has no use for),
parsing the rendered output (an unwritten contract nobody could
change).

## D10 — The "to: text" line is parsed by cli.ParseAddressedLine (automatic decision)

Chosen: splitting the input line into an addressee and a message is a
new exported function in `crabswarm/chat/cli`, beside the
`ParseQualifiedName` the admin verbs already address members with. It
cuts at the first colon, trims both halves, requires both, and hands the
addressee on untouched. Rationale: the address grammar belongs where the
rest of the CLI's addressing lives, and passing the addressee through is
what keeps "name", "team/name" and "*" meaning exactly what they mean to
`chat send` — the daemon resolves it and its refusal names the form to
retry with. Rejected: parsing in the tui package (a second home for the
CLI's addressing), validating the addressee's shape locally (would
reject an address the daemon accepts, or drift from it).

## D11 — The room is looked up inside Run, not in ./cmd (automatic decision)

Chosen: `Run` reads the roster before it constructs the bubbletea
program, and reports an unknown room, an unreachable daemon and a
refused identity as plain errors from there. Rationale: D5 keeps ./cmd
free of logic, and the check needs the room listing `Run` already holds
a reader for; doing it before the program starts is also what makes
every IDEA.md UC4 path exit without the terminal ever being taken over.
Rejected: checking in the command wiring (logic under ./cmd, and a
second listing call), checking inside the model (the screen would be
open and would have to close itself).

## D12 — The scrollback is one tail read; there is no paging backwards (automatic decision)

Chosen: the screen opens with a single read of the room log's tail (500
entries, which the daemon clamps to its retention cap) and every read
after that is forward from the cursor. Scrolling up reaches as far as
that opening read and no further. Rationale: `ChatAdminService.History`
reads either the tail or forward from a `since_id` — there is no
before-id read to page backwards with, so "as far as retention keeps
it" (IDEA.md UC2) is bought up front or not at all. Rejected: adding a
backwards read to the RPC (a schema change this plan does not own),
re-reading the tail with a growing limit on every scroll (re-fetches
the whole room to walk back one screen).

## D13 — Watch mode by default; the input line is focused deliberately (automatic decision)

Chosen: the screen opens unfocused, where `q` quits, `i` (or enter)
moves to the input line, and the arrows/page keys scroll; on the input
line every key is text, enter sends and esc leaves it. `ctrl-c` quits
from either. The status bar reads
`room R · tailing|scrolled back · connected|… · <last report> · <keys>`,
and entries are stamped with the UTC time of day. Rationale: IDEA.md
UC1 wants a screen useful without keypresses and UC3 has the operator
"focus the input line" — with one always-focused input, `q` could not
quit and every navigation key would be text. UTC because the CLI
transcript is UTC for the same reason (a room spans containers), and
time-of-day alone because the pane is narrow and the reader is
comparing messages minutes apart. Rejected: an always-focused input
(costs the navigation keys), a full date per line (costs width for
nothing).

## D14 — The model holds the context Run was given (automatic decision)

Chosen: the model keeps `ctx` as a field, against the repo's "do not
stash a context in a struct" preference. Rationale: a bubbletea command
is a `func() tea.Msg` with nowhere to take a context, and the poll and
send commands are built inside `Update`, which is handed only a message
— there is no other way for them to reach the one `Run` was given. The
field is documented at its declaration for that reason. Rejected:
`context.Background()` per call (the screen's work would outlive the
caller's cancellation), a package-level context (worse in every way).

