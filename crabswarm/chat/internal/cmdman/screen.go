// This file holds what a terminal snapshot is read against: the table of
// markers, and the guard [Terminal.SendCommand] runs it through. The screen
// poller in ../../notify classifies a snapshot off the same table, so what
// stops a nudge and what a member's state is read as cannot drift apart.

package cmdman

import (
	"regexp"
	"strings"

	"github.com/ngicks/crabswarm/crabswarm/chat"
)

// ScreenMarker is one thing a harness paints on its terminal, together with the
// state a screen painting it is in. A marker is either a literal the harness
// prints or, where the text changes from frame to frame, a pattern.
type ScreenMarker struct {
	// Text is the literal the harness prints, matched case-insensitively as a
	// substring. Empty when the marker carries a Pattern instead.
	Text string
	// Pattern matches what no fixed string can — a spinner whose glyph and
	// elapsed count change every frame, a prompt line that has to be empty.
	// Nil when the marker carries a Text instead.
	Pattern *regexp.Regexp
	// State is what a terminal showing the marker is doing.
	State chat.MemberState
}

// Matches reports whether snapshot shows the marker.
func (m ScreenMarker) Matches(snapshot string) bool {
	if m.Pattern != nil {
		return m.Pattern.MatchString(snapshot)
	}
	// Both sides are lowered here rather than keeping the table lowered, so
	// adding a marker stays a copy of what the harness actually prints.
	return strings.Contains(strings.ToLower(snapshot), strings.ToLower(m.Text))
}

// String names the marker for a log line: the literal itself, or the pattern's
// source.
func (m ScreenMarker) String() string {
	if m.Pattern != nil {
		return m.Pattern.String()
	}
	return m.Text
}

// ScreenMarkers is what a terminal snapshot is read against, in the order the
// markers are tested. The first that matches settles the screen.
//
// A dialog comes first because it overlays everything else — the status line
// included — so a screen showing one is waiting for an answer whatever is still
// painted behind it. A running turn comes next. The empty prompt comes last:
// the composer sits under a running turn too, so on its own it only means that
// nothing is running above it.
//
// Each group names the recorded screen it was read off, under
// e2e/crabswarm/testdata/harness/. They are heuristics: what a harness paints is
// whatever it happens to paint today, so this table is revised as those UIs
// change and nothing in it is a guarantee about the terminal.
var ScreenMarkers = []ScreenMarker{
	// claude-screen-dialog.txt: the permission dialog's heading and question,
	// its first numbered choice, and the hint on the last row.
	{Text: "Do you want", State: chat.StateWaiting},
	{Text: "This command requires approval", State: chat.StateWaiting},
	{Text: "❯ 1. Yes", State: chat.StateWaiting},
	{Text: "Esc to cancel", State: chat.StateWaiting},
	// The classic shell yes/no prompt. No fixture holds it: it belongs to
	// whatever the harness happens to be running, not to the harness.
	{Text: "(y/n)", State: chat.StateWaiting},
	// claude-screen-working.txt: the hint under a tool running in the
	// foreground, and the spinner line's elapsed counter. Claude Code's own
	// "(esc to interrupt)" hint is part of the footer a configured status line
	// replaces, so the spinner is what is left to read a running turn off; it is
	// a pattern because its glyph and its counter change every frame. The
	// ellipsis and the counter must sit on one line — the truncation marker that
	// ends an over-long status line is the same character.
	{Text: "(ctrl+b to run in background)", State: chat.StateWorking},
	{Pattern: regexp.MustCompile(`…[ \t]*\(\d+s`), State: chat.StateWorking},
	{Text: "esc to interrupt)", State: chat.StateWorking},
	// claude-screen-idle.txt: the composer's prompt line, standing empty between
	// its two rule lines. The whole line, not a "❯" found anywhere: the dialog's
	// first choice carries one, and so does every prompt the session has already
	// answered. The padding is \p{Zs} rather than a plain space because Claude
	// Code pads the empty composer with a no-break space. What is printed below
	// it is not a marker — an operator configures what the status line says, so
	// its words are not the harness's to keep.
	{Pattern: regexp.MustCompile(`(?m)^❯[\p{Zs}\t]*\r?$`), State: chat.StateDone},
}

// DialogMarkers are the literal strings of [ScreenMarkers] that mean the
// recipient's terminal is in no state to be typed into: a dialog waiting for an
// answer, which an injected line would answer instead of being typed, or a turn
// still running.
//
// Exported only so sibling packages' tests can build a snapshot that trips the
// guard without copying a marker; this is an internal package, so the set is
// not public API and stays free to change.
var DialogMarkers = busyLiterals()

// busyLiterals collects the literal markers of a state that must not be typed
// into. The patterns are left out: a caller of [DialogMarkers] wants a string it
// can put in a snapshot, and a regular expression is not one.
func busyLiterals() []string {
	var out []string
	for _, m := range ScreenMarkers {
		if m.Pattern == nil && m.State != chat.StateDone {
			out = append(out, m.Text)
		}
	}
	return out
}

// dialogMarker names the marker that stops the send, when the screen shows one.
//
// It reads the first of [ScreenMarkers] that snapshot matches, so a screen the
// table settles as done stops the scan: whatever a later marker would also have
// matched was painted under a prompt that is standing empty.
func dialogMarker(snapshot string) (string, bool) {
	for _, m := range ScreenMarkers {
		if !m.Matches(snapshot) {
			continue
		}
		if m.State == chat.StateDone {
			return "", false
		}
		return m.String(), true
	}
	return "", false
}
