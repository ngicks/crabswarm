package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// The terminal hand-off itself — [tea.ExecProcess] releasing the terminal,
// running the editor on the program's own input and output, and resuming the
// screen — needs a running program and is exercised by the e2e suite. What is
// tested here is everything on either side of it: the temp file the editor is
// handed, the argv the command line is split into, and what the screen does
// with the callback's message either way.

// scriptedEditor writes a shell script that behaves like an editor and hands
// back the command line [Deps.Editor] would hold. body is run with the file to
// edit as "$1".
func scriptedEditor(t *testing.T, name, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the scripted editor is a shell script")
	}
	path := filepath.Join(t.TempDir(), name)
	assert.NilError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o700))
	return path
}

// appendingEditor is an editor that adds a line and exits 0, which is the
// whole of what the screen asks of one. The newline in front of it is the one
// the draft does not carry: the pane's text is written to the file exactly as
// it stands, with no line ending after the last line.
func appendingEditor(t *testing.T) string {
	t.Helper()
	return scriptedEditor(t, "append-editor", `printf '\nfrom the editor\n' >> "$1"`)
}

// failingEditor is an editor that exits non-zero, which is how an operator
// abandons the message — `:cq` in vim, a crash, a command that is not there.
func failingEditor(t *testing.T) string {
	t.Helper()
	return scriptedEditor(t, "failing-editor", `exit 1`)
}

// runEditor is the hand-off with the terminal part left out: the command the
// screen would have handed to bubbletea is run here instead, and the callback
// bubbletea would have called is called with its result.
func runEditor(t *testing.T, m *model) *model {
	t.Helper()
	cmd, path, err := editorCommand(m.deps.Editor, m.text.Value())
	assert.NilError(t, err)

	// What the editor is opened on is the draft exactly as it stands, in a
	// file an editor will read as markdown.
	handed, err := os.ReadFile(path)
	assert.NilError(t, err)
	assert.Equal(t, string(handed), m.text.Value())
	assert.Equal(t, filepath.Ext(path), ".md")

	m = update(t, m, editorDone(path)(cmd.Run()))
	// The file is the hand-off and not a place a message is kept, so it is
	// gone whichever way the editor went.
	_, statErr := os.Stat(path)
	assert.Assert(t, os.IsNotExist(statErr), "the temp file is still there: %v", statErr)
	return m
}

// ctrl+g with neither variable set says so and does nothing else: the draft is
// where it was and the operator can carry on typing.
func TestTheEditorKeyWithNoEditorSetSaysSo(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m = typeLine(t, m, "@backend/alice half a thought")

	next, cmd := m.Update(ctrlPress('g'))
	m = next.(*model)

	assert.Assert(t, cmd == nil)
	assert.Equal(t, m.notice, "no VISUAL or EDITOR set")
	assert.Equal(t, m.text.Value(), "@backend/alice half a thought")
	assert.Equal(t, m.focus, focusMessage)
}

// An editor that exits 0 hands the message back, with the cursor after it so
// the operator can keep writing. Nothing is sent.
func TestTheEditorHandsTheMessageBack(t *testing.T) {
	// The key's own path leaves a temp file behind that only bubbletea's
	// callback would have removed, so it is made somewhere the test collects.
	t.Setenv("TMPDIR", t.TempDir())
	sender := &fakeSender{}
	m := fixtureModel(t, Deps{Sender: sender, Editor: appendingEditor(t)})
	m = typeLine(t, m, "@backend/alice a draft")

	// ctrl+g reaches the hand-off: bubbletea is handed a command to run and
	// the system line has nothing to report. What that command does to the
	// terminal is the e2e suite's to check.
	next, cmd := m.Update(ctrlPress('g'))
	m = next.(*model)
	assert.Assert(t, cmd != nil)
	assert.Equal(t, m.notice, "")

	m = runEditor(t, m)

	assert.Equal(t, m.text.Value(), "@backend/alice a draft\nfrom the editor")
	assert.Equal(t, m.notice, "")
	assert.Equal(t, m.focus, focusMessage)
	// The pane grew a row, which the conversation gave up.
	assert.Equal(t, m.textRows(), 2)
	assert.Equal(t, len(sender.calls), 0)
}

// An editor that exits non-zero leaves the draft exactly as it was and says
// what happened, since the operator's alternative is to type it again.
func TestAnEditorThatFailsLeavesTheDraftAlone(t *testing.T) {
	m := fixtureModel(t, Deps{Editor: failingEditor(t)})
	m = typeLine(t, m, "@backend/alice a draft")

	m = runEditor(t, m)

	assert.Equal(t, m.text.Value(), "@backend/alice a draft")
	assert.Equal(t, m.notice, "editor exited: exit status 1")
	assert.Equal(t, m.focus, focusMessage)
}

// An editor that cannot be started at all is the same report as one that ran
// and failed: the hand-off did not happen and the draft is untouched.
func TestAnEditorThatCannotStartIsReportedTheSameWay(t *testing.T) {
	m := fixtureModel(t, Deps{
		Editor: filepath.Join(t.TempDir(), "no-such-editor"),
	})
	m = typeLine(t, m, "@backend/alice a draft")

	m = runEditor(t, m)

	assert.Equal(t, m.text.Value(), "@backend/alice a draft")
	assert.Assert(t, strings.HasPrefix(m.notice, "editor exited: "), m.notice)
}

// Editors end a file with a newline the operator did not type, and a trailing
// blank line in a chat message is a row of the pane spent on nothing.
func TestTheTrailingNewlineTheEditorAddsIsTakenOff(t *testing.T) {
	m := fixtureModel(t, Deps{})
	m.applyEdited(editedMsg{text: "one\ntwo\n\n", ok: true})
	assert.Equal(t, m.text.Value(), "one\ntwo")

	// An editor may legitimately leave the file empty; that is an emptied
	// draft, not a failure.
	m.applyEdited(editedMsg{text: "\n", ok: true})
	assert.Equal(t, m.text.Value(), "")
}

// The variables carry a command line, not a program: `code -w` is an ordinary
// thing to find in one.
func TestTheEditorCommandLineIsSplitLikeAShellWould(t *testing.T) {
	for _, tc := range []struct {
		name   string
		editor string
		want   []string
		fails  bool
	}{
		{name: "a bare program", editor: "nvim", want: []string{"nvim"}},
		{
			name:   "a program with arguments",
			editor: "code -w",
			want:   []string{"code", "-w"},
		},
		{
			name:   "a quoted path with a space in it",
			editor: `"/opt/my editor/bin/ed" -f`,
			want:   []string{"/opt/my editor/bin/ed", "-f"},
		},
		{name: "an unbalanced quote", editor: `"nvim`, fails: true},
		{name: "nothing at all", editor: "   ", fails: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			argv, err := editorArgs(tc.editor)
			if tc.fails {
				assert.Assert(t, err != nil)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, argv, tc.want)
		})
	}
}

// A split that fails is reported on the system line and nothing is run — not
// as "editor exited", which would say a program ran that never did.
func TestAnUnreadableEditorVariableIsReportedWithoutRunningAnything(t *testing.T) {
	m := fixtureModel(t, Deps{Editor: `"nvim`})
	m = typeLine(t, m, "a draft")

	next, cmd := m.Update(ctrlPress('g'))
	m = next.(*model)

	assert.Assert(t, cmd == nil)
	assert.Assert(t, m.notice != "")
	assert.Equal(t, m.text.Value(), "a draft")
}
