package tui

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	shellwords "github.com/mattn/go-shellwords"
)

// ctrl+g hands the draft to the operator's own editor and takes back whatever
// it left. Nothing is sent by it: the draft comes back into the message pane
// and the operator sends it themselves, so an editor is a way of writing a
// long message rather than a second way of sending one.
//
// bubbletea runs it through [tea.ExecProcess], which releases the terminal for
// the duration and resumes the screen afterwards — a full-screen editor is
// exactly what the two variables are conventionally set to, and it needs the
// terminal to itself.

// editedMsg is the editor having exited, with whatever it left in the file.
// ok distinguishes a file that was read back from one that never was, since an
// editor may legitimately leave it empty.
type editedMsg struct {
	text string
	ok   bool
	err  error
}

// openEditor writes the draft to a temp markdown file and hands it to
// [Deps.Editor].
//
// The file is markdown because that is what a message to a harness is written
// in, and it is what an editor reads the suffix for when it picks a mode.
func (m *model) openEditor() tea.Cmd {
	if m.deps.Editor == "" {
		m.notice = "no VISUAL or EDITOR set"
		return nil
	}
	cmd, path, err := editorCommand(m.deps.Editor, m.text.Value())
	if err != nil {
		m.notice = err.Error()
		return nil
	}
	m.notice = ""
	return tea.ExecProcess(cmd, editorDone(path))
}

// editorCommand is the editor as an [exec.Cmd] over a temp file holding text,
// and the path of that file so the caller can read it back and remove it. The
// file is removed here only when the command never gets built.
func editorCommand(editor, text string) (*exec.Cmd, string, error) {
	argv, err := editorArgs(editor)
	if err != nil {
		return nil, "", err
	}
	f, err := os.CreateTemp("", "crabswarm-chat-*.md")
	if err != nil {
		return nil, "", editorError(err)
	}
	path := f.Name()
	_, werr := f.WriteString(text)
	cerr := f.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(path)
		return nil, "", editorError(err)
	}
	return exec.Command(argv[0], append(argv[1:], path)...), path, nil
}

// editorArgs splits the editor command line the way a shell would: both
// variables conventionally carry arguments — `code -w`, `emacsclient -nw` —
// and running the whole string as one program name would look for a file with
// a space in it.
func editorArgs(editor string) ([]string, error) {
	argv, err := shellwords.Parse(editor)
	if err != nil {
		return nil, fmt.Errorf("editor: cannot read $VISUAL/$EDITOR: %w", err)
	}
	if len(argv) == 0 {
		return nil, errors.New("editor: $VISUAL/$EDITOR names no command")
	}
	return argv, nil
}

// editorDone reads the file back once the editor has exited, and removes it
// either way — the draft is in the pane by then and the file is the hand-off,
// not a place a message is kept.
func editorDone(path string) tea.ExecCallback {
	return func(err error) tea.Msg {
		defer func() { _ = os.Remove(path) }()
		if err != nil {
			return editedMsg{err: err}
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return editedMsg{err: rerr}
		}
		return editedMsg{text: string(b), ok: true}
	}
}

// editorError is how a failure to reach the editor at all is spelled, which is
// the same sentence as an editor that ran and exited non-zero: from the
// operator's side both are "the hand-off did not happen", and the draft is
// untouched in either case.
func editorError(err error) error {
	return fmt.Errorf("editor exited: %w", err)
}

// applyEdited takes the file back into the message pane, or says why it did
// not. A failed hand-off leaves the draft exactly as it was: the operator's
// alternative is to type it again.
func (m *model) applyEdited(msg editedMsg) {
	if msg.err != nil {
		m.notice = editorError(msg.err).Error()
		return
	}
	if !msg.ok {
		return
	}
	// An editor ends a file with a newline the operator did not type, and a
	// trailing blank line in a chat message is a row of the pane spent on
	// nothing.
	m.text.SetValue(strings.TrimRight(msg.text, "\n"))
	m.text.MoveToEnd()
	m.notice = ""
}
