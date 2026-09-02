# Decisions — admin TUI panes, textarea, `@` addressing, editor

One entry per material decision: choice, rationale, rejected alternatives.
Stubs are seeded from PLAN.md's open questions and finished as each
resolves. Inherited decisions from `doc/plan/2026-08-31-06-admin_tui` are
quoted verbatim where they still bind.

## Inherited (quoted, still binding)

- Prior D3: "`--room` required; no picker screen" — **no longer binding**
  as of D13: the screen now carries a room list; `--room` stays as an
  optional initial selection (D14).
- Prior D5: "Code lives in `crabswarm/chat/cli/tui`; `./cmd` stays thin".
- Prior D13's rule that the screen never echoes its own send locally:
  "what the room said is the log's to say" (send.go doc comment).

## D1 — Roster selects members and whole teams; a team-wide send target is added

**Decision (user, 2026-09-02)**: the roster cursor stops on team headings
and on members. `enter` on a member pre-fills `@team/name `; `enter` on a
team pre-fills a team-broadcast address (`@team/* `, spelling finalized
in D10). Because `AdminService.Send` knows only `*` and single members, a
team-wide target is added to the daemon.

**Rationale**: the user confirmed "members only plus whole team broadcast
is ok"; addressing a team is the natural unit in a compose-launched swarm
(team = compose project).

**Rejected**: members only (no team send — user wanted it); team as a
completion scope only (does not deliver to the team).

## D2 — `ctrl+enter` sends; `enter` inserts a newline

**Decision (user, 2026-09-02)**: the send key is `ctrl+enter`. `enter` is
always a newline. The user was explicit: "I strongly [am] against enter to
send." Fallback where the terminal cannot report `ctrl+enter` is open
question 13.

**Rejected**: enter sends + alt+enter newline (user refused); ctrl+s.

## D3 — `@` grammar: first bare token is the target, text sent whole, `to: text` removed

**Decision (user, 2026-09-02)**: the first `@token` outside backticks and
not escaped as `\@` names the target; the message text is sent *whole*,
token included, so it doubles as a mention. No bare `@` → broadcast `*`.
`@` is the only addressing form on the screen; `cli.ParseAddressedLine`
loses its only caller and is removed.

**Rejected**: stripping the token (user chose whole line); keeping the
`to: text` form alongside (ambiguous with a leading word and colon).


## D4 — System line under the textarea; the status bar stays

**Decision (user, 2026-09-02)**: keep both. The status bar keeps room /
tailing-or-scrolled / connection / key hints; every notice (send
result, editor problems, parse errors) moves to a one-line system line
directly under the textarea. Replaces the `notice` field's place in
`statusBar()`.

**Rejected**: a single line doing both (loses persistent hints).

## D5 — Layout by `ultraviolet/layout`; ultraviolet becomes a direct dependency

**Decision (2026-09-02)**: pane rectangles come from
`github.com/charmbracelet/ultraviolet/layout` (`Horizontal`/`Vertical`
with `Len`/`Fill`, `Split`), so `go.mod` promotes
`github.com/charmbracelet/ultraviolet` from `// indirect` to direct at
the exact pseudo-version bubbletea v2.0.9 already pins
(`v0.0.0-20260811164956-006e29f97886`). No new module enters the graph
and nothing drifts; the mock already builds against it.

**Dependency justification**: MIT, maintained by the bubbletea authors as
bubbletea v2's own substrate, footprint zero beyond what is linked today.
The risk to record: the module is untagged, so a future bubbletea bump
moves it too; the code path used is four functions.

**Rejected**: keeping the hand arithmetic (`rosterWidth`, `chromeHeight`)
— what the user asked to replace, and it does not compose with a
two-column split; `lipgloss` `JoinHorizontal` measuring — no constraint
solving, overflow on resize; writing a solver in-repo.

## D6 — The `@` parser is tui-internal

**Decision (user, 2026-09-02)**: the parser that finds the first bare
`@token` (skipping backtick spans and `\@`) lives unexported in
`crabswarm/chat/cli/tui` (`address.go`, table-tested). Resolves open
question 8.

**Rationale**: it is this screen's grammar; nothing else reads it today.
`cli.ParseAddressedLine` loses its only caller and is removed.

**Rejected**: exported in `crabswarm/chat/cli` — surface for a consumer
that does not exist yet.

Pending.

## D7 — Keyboard enhancements requested at startup

**Decision (user, 2026-09-02)**: the program requests key disambiguation
(`tea.View.KeyboardEnhancements` with `DisambiguateEscapeCodes`, kitty
flag / modifyOtherKeys) so terminals that can tell `ctrl+h` from
Backspace do, and `ctrl+enter` is reportable. Where the terminal answers
nothing, `ctrl+h` inside the textarea stays delete-backward (the
Backspace of `^H` terminals) and the members pane is reached by `ctrl+k`
then `ctrl+h`. Resolves open question 9.

**Rejected**: never asking (loses `ctrl+h` as a focus key everywhere).

Pending.

## D8 — Editor lookup order: VISUAL, then EDITOR

**Decision (user, 2026-09-02)**: `ctrl+g` runs `$VISUAL`; when that is
unset, `$EDITOR`. Both unset → the system line says
`no VISUAL or EDITOR set`. The value is split with
`github.com/mattn/go-shellwords` so `code -w` works. The read lives in
`crabswarm/chat/cli` (`EditorFromEnv`), not under `./cmd`, per the rule
in `crabswarm/config.go`. Resolves open
question 12 (the earlier tentative `${EDITOR:-$VISUAL}` reading is
reversed).

**Rejected**: EDITOR first (the plan's first reading of the user's shell
snippet; the user chose the conventional order).

Pending; user wrote `${EDITOR:-$VISUAL}`.

## D9 — A preview mock is built before the gate

**Decision (user, 2026-09-02)**: "Make a mock please? So I can see
behavior?" — a runnable bubbletea mock lives under this plan's `mock/`
directory behind the `tuimock` build tag, like the prior plan's, so the
layout, focus keys, `@` completion, `ctrl+enter` send and `ctrl+g` editor
hand-off can be judged at the gate. It is disposable: production code is
written against the plan, and anything lifted from the mock is read
against DECISION.md wording first (visuals.md rule).

**Rejected**: deferring the mock to implementation start (prior D6) — the
request is a readability complaint, which only a rendered screen answers.

## D10 — `AdminSendRequest.target` becomes a `oneof`: everyone, team, member

**Decision (user, 2026-09-02)**: "Expand proto surface. target with
oneOf. everyone, team, member targets?" — the string `target` field is
replaced by a `oneof target` with three cases: `Everyone` (the whole
room), `TeamTarget{team}` (every member of that team) and
`MemberTarget{team, name}` (one member; `team` empty means the bare-name
rule of `resolveFor`). No string grammar on the wire, so `validateName` is
untouched and a member named `*` stays legal — but only the members
pane's `enter` reaches it: `ParseAdminTarget` and the typed `@` grammar
both read `team/*` as the team case, so from `chat admin send` argv and
from a typed address that member is unaddressable. Accepted. `chat admin send` maps
its argv grammar (`*`, `team/*`, `team/name`, `name`) onto the cases
client-side; the screen builds the cases directly from the roster or
from the parsed `@token`. Resolves open question 14, and 15 with it: a
`TeamTarget` delivers to every current member of that team, counted at
send time.

**Rejected**: `team/*` as a string with `*` reserved (a grammar the
daemon would have to parse and a name it would have to forbid); a
separate top-level `team` field beside the string (two ways to say one
thing).

Pending. Tentative `team/*` with `*` reserved as a name.

## D11 — `alt+enter` sends where `ctrl+enter` cannot be reported

**Decision (user, 2026-09-02)**: the screen requests keyboard
enhancements at startup; when `tea.KeyboardEnhancementsMsg` reports no
key disambiguation, `alt+enter` is the send key and the status hint says
`alt+enter sends (terminal cannot report ctrl+enter)`. `alt+enter` is
accepted as a second send key everywhere.

**Rejected**: refusing to open; leaving send unavailable there.

**Amended (user, 2026-09-02)**: "alt+enter is absorbed by wezterm. It's
toggle fullscreen it can't be used!" — the fallback send key is `ctrl+x`
instead, everywhere `alt+enter` was named: the hint reads
`ctrl+x sends (terminal cannot report ctrl+enter)`, and `ctrl+x` is
accepted as a second send key on every terminal. `ctrl+x` is free in
bubbles' textarea `KeyMap` and has no terminal-level binding.
Alternatives offered and not taken: `ctrl+s` (raw mode disables XOFF,
but the user preferred `ctrl+x`), `ctrl+o`, no fallback.

## D12 — the log is focusable and scrolls with vim keys

**Decision (user, 2026-09-02)**: "There's chat log above textarea right?
We should be able to focus on the log and scroll back" — the conversation
pane takes focus like any other, reached by the ordinary directional keys
(`ctrl+k` from the textarea, since the log is the pane above it). In the
conversation: `j`/`k` one line, `gg` top, `G` bottom, `ctrl+d`/`ctrl+u`
half page; `h`/`l` no-ops (soft wrap).

**Amended (user, 2026-09-02)**: an earlier wording made `ctrl+k` a
special "always the log above" rule with a rejected "return to the pane
last left" alternative. Neither exists: `ctrl+h`/`j`/`k`/`l` are plain
directional focus moves and nothing more; where `ctrl+k` lands follows
from the layout, not from a rule about the message pane.

## D13 — Two-column layout: rooms over members on the left, log over textarea on the right

**Decision (user, 2026-09-02)**: "Put members section as left pane. Left
upper pane is room selection (scrollable), left bottom section is members
(also scrollable). It spans from top to bottom. Right section is of chat
log and text input area." The left column is full height and holds two
scrollable lists, rooms above members; the right column holds the
conversation above the message textarea; the system line and status bar
stay full width underneath. The member list is what the earlier drafts
called the roster; the pane title is `members`.

**Consequences**: a room list is now part of the screen, which retires
prior D3's "no picker screen" and reopens what `--room` means (open
question 16). Horizontal focus moves need a rule for which of the two
panes on the other side they land on (open question 17, tentative rule in
IDEA.md). The narrow-terminal rule drops the whole left column, not just
the members pane.

**Rejected**: the earlier draft's layout (conversation top-left, roster
top-right, message full width) — the roster wedged beside the log left
it neither aligned with the textarea nor able to grow, and there was no
room list at all.

## D14 — `--room` is kept, optional; it pre-selects the room on open

**Decision (user, 2026-09-02)**: "`--room` flag is kept but will be
optional because it can select on-screen." With the flag, the screen
opens on that room (an unknown room is still refused before the terminal
is taken over, UC9). Without it, the screen opens on the first room in
the list, and the operator picks from the rooms pane. Resolves open
question 16.

**Rejected**: keeping it required (pointless once the pane can select);
removing it (loses the one-keystroke-free way to land on a known room,
and every existing invocation and e2e).

## D15 — Horizontal focus moves are derived from the layout, not hard-wired

**Decision (user, 2026-09-02)**: asked which pane a `ctrl+h`/`ctrl+l`
lands on when the two columns' rows do not line up, the user answered
"Let layout engine decide?" — the move is computed from the solved
rectangles: the target is the pane adjacent in that direction whose rows
overlap the focused pane's rows the most (tie: the upper one for a
horizontal move, the left one for a vertical move). The same
function serves `ctrl+j`/`ctrl+k`, so the key router carries no
pane-to-pane table at all; a changed split or a resize moves the targets
with the boundaries. Resolves open question 17.

**Rejected**: a fixed pairing (rooms/members → conversation; conversation
→ rooms, message → members) — right for the default proportions but a
second place that encodes the layout; "return to the pane last left" —
history-dependent.

## D16 — Per-room drafts

**Decision (user, 2026-09-02)**: the textarea keeps one draft per room.
Selecting another room shows that room's draft (empty if none); switching
back restores the earlier text, cursor at the end. Drafts live only for
the screen's lifetime. Resolves open question 18.

**Rejected**: one shared draft (an `@team/name` draft is room-specific);
clearing on switch (loses typed text).

## D17 — Textarea height is dynamic, one to six rows

**Decision (user, 2026-09-02)**: the message textarea grows with its
contents from one row up to six, then scrolls inside; the conversation
pane gives up the rows. Resolves open question 11.

**Rejected**: a fixed three rows (wastes two rows for a one-liner and
hides a six-line draft).

## D18 — Explicit width gate; below it the columns swap on focus moves

**Decision (user, 2026-09-02)**: "Explicit width gate. Navigate to pane
to switch when it's lower than the gate." Below `leftMinWidth` (60
columns, the value today's `rosterMinWidth` holds) the screen shows one
column at a time: the right column by default, and a focus move toward
the hidden column (`ctrl+h` from the conversation or the message) brings
the left column on screen in its place, with focus on the pane the
layout-derived rule picks; a move back (`ctrl+l`) restores the right
column. The status bar's room segment stays, so the operator always
knows where they are. Above the gate both columns are shown and sizes
come from `layout.Len`/`Fill`. Resolves open question 10; refines
IDEA.md's "narrow terminals" rule, which said the left column is lost —
it is hidden, not lost.

**Rejected**: `Min`/`Max` constraints and letting the solver shrink the
column (a 12-column room list is useless); dropping the column with no
way back (the old roster rule).

## D19 — The rooms pane rides the roster poll; a switch re-tails the log

**Decision (user, 2026-09-02)**: `AdminService.ListRooms` already returns
every room with its members, and `poll.go`'s roster poll already calls it
every interval; the same reply now fills both left panes. Selecting a
room resets the log cursor to zero (the tail read), clears the entries,
marks the view as following, and the next tail poll fills the
conversation from that room's newest entries. Resolves open question 19.

**Rejected**: a separate slower rooms poll (a second loop for data the
first already carries).

## D20 — `MemberTarget` is structured `{team, name}`

**Decision (user, 2026-09-02)**: the member case of D10's `oneof` carries
`team` and `name` as two fields; an empty `team` means the bare-name rule
in `resolveFor` (caller's own team first, then unique across the room).
`chat admin send`'s argv keeps `team/name` and splits it client-side.
Resolves open question 20.

**Rejected**: one `address` string inside the member case (the daemon
keeps parsing a grammar the proto could carry as fields).

## D21 — Bubbletea example palette, ANSI-256

**Decision (user, 2026-09-02)**: "Use bubbletea-ish purple color scheme."
One palette block in `tui/styles.go`: focused frame and title `62`
(purple), blurred frame `240`, blurred title `245`; cursor and selection
`205` (pink) bold; team headings `99`; status bar `241`, system line
`245`, key-hint accents `62`; textarea prompt `62`. ANSI-256 only.

**Rejected**: truecolor hex values (not every terminal the operator uses
renders them the same); lipgloss adaptive light/dark pairs (two palettes
to keep consistent for a screen this small).

## D22 — Admin mentions are colored

**Decision (user, 2026-09-02)**: "Color mention to admin." A conversation
entry whose text contains a bare `@admin` or `@admin/admin` token is
drawn in the mention color (`205`), the token bold. Bare uses the send
tokenizer's rules (`parseAddress` in `tui/address.go`): outside a
backtick span, not `\@`-escaped. The rule is textual because the admin
holds no member row and cannot be a send target (`validateName`
reserves the name); `AdminHistoryEntry.to` is never the admin.

**Rejected**: coloring only entries addressed *to* a member of the
admin's choosing (no such target exists); coloring every entry that
mentions anyone (noise — the operator cares about themselves).

## Run mode: autonomous [automatic]

**Decision (orchestrator, 2026-09-03)**: the implementation run started
from a `/goal` with the user away; the availability question was not
asked. Every unclear corner below is decided by the orchestrator and
tagged `[automatic]` for the user to skim on return.

## `ParseAddressedLine` is removed in step 5, not step 2 [automatic]

**Decision (orchestrator, 2026-09-03)**: `cli.ParseAddressedLine` is the
TUI's send parser until the `@` parser lands in step 5. Removing it in
step 2, as PLAN.md words it, would leave `crabswarm/chat/cli/tui` without
a send path for two steps. Step 2 keeps it (adapting it to the typed
`AdminTarget`); step 5 deletes it with its tests when `address.go`
replaces it. Step 1 makes the minimal client-side change needed so
`go build ./...` stays green after the proto field is replaced.

## A team send is logged as a whole-room entry [automatic]

**Decision (orchestrator, 2026-09-03)**: `broadcastTeamFrom` records the
message with no recipient, so `admin history` and the conversation pane
show a team send the way they show a `*` send. The history reader only
reconstructs a recipient when a member name is present, and PLAN.md
forbids a persistent-data change in this plan, so a team-only recipient
row is not added. Logged to HANDOFF.md for a later decision.

## Key disambiguation is bubbletea's default; nothing extra is requested [automatic]

**Decision (orchestrator, 2026-09-03)**: bubbletea v2.0.9 has no
`DisambiguateEscapeCodes` option; its renderer always pushes kitty flag 1
(disambiguate escape codes) for every view, which is exactly what D7
asks for. `View()` sets the zero `KeyboardEnhancements` with a comment
saying so. `ReportAllKeysAsEscapeCodes` (the mock's choice) was not
substituted: it changes how every text key is delivered and the
pipe-driven tests could not catch a regression.

## Tab walks the completion list; Tab on the last row accepts [automatic]

**Decision (orchestrator, 2026-09-03)**: PLAN.md's "`Tab`/`j`/`k` move,
`enter`/second `Tab` accept" cannot both hold literally. The mock the
user saw at the gate has `Tab` move down the list and accept when on the
last row, `enter` accept anywhere; that rule is kept.
