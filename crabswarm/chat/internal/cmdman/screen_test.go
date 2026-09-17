package cmdman

import (
	"slices"
	"strings"
	"testing"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"gotest.tools/v3/assert"
)

func TestDialogMarker(t *testing.T) {
	// Every literal marker must match whatever casing the harness prints it in,
	// and an idle prompt must match none of them — the guard declining forever
	// would be as silent a failure as it never declining.
	for _, marker := range DialogMarkers {
		t.Run(marker, func(t *testing.T) {
			for _, snapshot := range []string{
				"noise\n" + marker + "\nnoise",
				"noise\n" + strings.ToUpper(marker) + "\nnoise",
				"noise\n" + strings.ToLower(marker) + "\nnoise",
			} {
				got, found := dialogMarker(snapshot)
				assert.Assert(t, found, "no marker found in %q", snapshot)
				assert.Equal(t, got, marker)
			}
		})
	}

	_, found := dialogMarker("$ echo hello\nhello\n" + idlePrompt)
	assert.Assert(t, !found, "an idle prompt must not read as a dialog")
}

func TestDialogMarker_ReadsTheWholeTable(t *testing.T) {
	// A turn still running is as bad a moment to type into as a dialog: the
	// spinner is a pattern, so it reaches the guard through the table rather
	// than through [DialogMarkers].
	got, found := dialogMarker("* Generating… (19s · ↓ 193 tokens)\n")
	assert.Assert(t, found, "a running turn must stop the send")
	assert.Equal(t, got, `…[ \t]*\(\d+s`)

	// A composer standing empty settles the screen, so nothing painted above it
	// is read as a reason not to type. The padding is the no-break space Claude
	// Code fills the empty prompt with.
	_, found = dialogMarker("✻ Worked for 1s · done 11:08 PM\n────\n❯ \n────\n")
	assert.Assert(t, !found, "an empty composer must not read as busy")
}

func TestDialogMarkers_AreTheBusyLiterals(t *testing.T) {
	// The guard and the classifier read one table. DialogMarkers is the literal
	// half of what that table calls busy, so a marker added for the classifier
	// tightens the guard with it rather than drifting away from it.
	for _, m := range ScreenMarkers {
		listed := slices.Contains(DialogMarkers, m.Text)
		busyLiteral := m.Pattern == nil && m.State != chat.StateDone
		assert.Equal(t, listed, busyLiteral, "marker %q", m)
	}
}
