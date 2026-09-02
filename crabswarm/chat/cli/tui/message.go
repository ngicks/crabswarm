package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// The message pane is a textarea rather than a line: a message to a harness is
// often a paragraph, and enter is not a send key here — the operator asked for
// it to be a newline, always.
const (
	// sendKey is what sends, and sendFallbackKey what sends where the terminal
	// cannot report the first. ctrl+enter is only distinguishable from enter in
	// terminals that answer the kitty keyboard protocol; the ones that do not
	// answer nothing at all, which is why there is a second key rather than a
	// hint.
	sendKey         = "ctrl+enter"
	sendFallbackKey = "ctrl+x"
	// editorKey opens the message in $VISUAL/$EDITOR. The hand-off itself is
	// not written yet; the key is claimed here so the textarea's select-all
	// cannot take it in the meantime.
	editorKey = "ctrl+g"
	// messageMaxContentRows is how many visual rows a message may grow to.
	// [textarea.Model.MaxHeight] alone doubles as the content guard, which
	// would stop the operator typing at six rows; MaxContentHeight moves the
	// guard off the pane's height so MaxHeight only caps how much of the
	// message is on screen.
	messageMaxContentRows = 500
)

// newTextarea is the message pane's editor: as tall as what is written in it
// within [messageMinRows] and [messageMaxRows], drawing neither line numbers
// nor colours of its own, since the frame around it and [styles.go] are what
// this screen is coloured by.
func newTextarea() textarea.Model {
	text := textarea.New()
	text.Prompt = "> "
	text.ShowLineNumbers = false
	text.Placeholder = "@team/name addresses one; no @ writes to the room"
	text.DynamicHeight = true
	text.MinHeight = messageMinRows
	text.MaxHeight = messageMaxRows
	text.MaxContentHeight = messageMaxContentRows
	text.SetHeight(messageMinRows)

	// ctrl+h and ctrl+k move between panes and ctrl+g opens the editor, so the
	// textarea's bindings for them are unbound rather than shadowed: the router
	// intercepts all three first, and a binding still holding the key would
	// only fire when that interception is wrong. Backspace keeps its own key —
	// what it loses is the ^H that terminals without keyboard enhancements
	// send for it. Nothing is unbound for enter: the textarea has no send of
	// its own, and its enter is already the newline this screen wants.
	text.KeyMap.DeleteCharacterBackward = key.NewBinding(key.WithKeys("backspace"))
	text.KeyMap.DeleteAfterCursor = key.NewBinding()
	text.KeyMap.SelectAll = key.NewBinding()

	styles := text.Styles()
	for _, state := range []*textarea.StyleState{&styles.Focused, &styles.Blurred} {
		state.Prompt = promptStyle
		state.Placeholder = placeholderStyle
		// The defaults carry a background on the cursor line and greys on the
		// rest, which are colours from outside the palette; the message is the
		// terminal's own foreground and the frame says whether it is focused.
		state.CursorLine = plainStyle
		state.Text = plainStyle
		state.EndOfBuffer = plainStyle
	}
	// The caret is a cursor like the ones the two list panes draw, and wears
	// the colour they do rather than the textarea's own.
	styles.Cursor.Color = caretColor
	text.SetStyles(styles)
	return text
}

// textRows is how many rows the message pane writes into, inside its frame:
// what is written in it, from one row up to six. The conversation pane gives
// the rows up, which is what makes a six-line draft readable while it is being
// written and costs nothing while it is one line.
func (m *model) textRows() int {
	return min(max(m.text.Height(), messageMinRows), messageMaxRows)
}

// messageKey writes the message, sends it, and completes an `@token` in it.
// Every other key is the textarea's, q included: letters are text here.
func (m *model) messageKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	// While the dropdown is open it answers the keys that drive it, and only
	// those: everything else closes it and is the message's again.
	if m.completion.open && m.completionKey(s) {
		m.layout()
		return m, nil
	}
	switch s {
	case sendKey, sendFallbackKey:
		return m, m.submit()
	case "tab":
		m.openCompletion()
		m.layout()
		return m, nil
	case editorKey:
		return m, nil
	}
	// Typing answers whatever the system line last said, so the report goes
	// with it.
	m.notice = ""
	var cmd tea.Cmd
	m.text, cmd = m.text.Update(msg)
	// The pane may have just grown or shrunk a row, which is a row the
	// conversation gives up or takes back.
	m.layout()
	return m, cmd
}
