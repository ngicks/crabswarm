package tui

import "charm.land/lipgloss/v2"

// The screen wears bubbletea's own colours, ANSI-256 throughout so a terminal
// with a themed palette recolours it rather than fighting it: purple on the
// pane the keys reach, pink on whatever a cursor sits on, grey on everything
// the operator is not being asked to look at.
//
// Every colour the screen draws is named here and nowhere else, so the palette
// is one block to read and one block to change.
var (
	// purple marks what has focus — the frame, its title, the keys the status
	// bar names, and the prompt of the line being written into.
	purple = lipgloss.Color("62")
	// violet heads a team in the members pane.
	violet = lipgloss.Color("99")
	// pink is a selection: the marked room, and — once the rooms and members
	// panes carry one — the row a cursor is on, bold.
	pink = lipgloss.Color("205")
	// dim, muted and faint are the three greys: a blurred frame, a blurred
	// title beside the system line, and the status bar.
	dim   = lipgloss.Color("240")
	muted = lipgloss.Color("245")
	faint = lipgloss.Color("241")
)

var (
	// focusedEdge and blurredEdge are the two frame colours, which is the
	// screen's only answer to "which pane do the keys reach".
	focusedEdge = purple
	blurredEdge = dim
	// focusedTitleStyle and blurredTitleStyle name a pane on its top edge; a
	// blurred pane is still named, a shade lighter than its own frame, so it
	// can be read without being focused first.
	focusedTitleStyle = lipgloss.NewStyle().Foreground(purple).Bold(true)
	blurredTitleStyle = lipgloss.NewStyle().Foreground(muted)
	// systemStyle and statusStyle are the two lines under the columns.
	systemStyle = lipgloss.NewStyle().Foreground(muted)
	statusStyle = lipgloss.NewStyle().Foreground(faint)
	// keyStyle lifts the key out of the hint it heads, so the bar reads as
	// keys with words after them rather than one grey sentence.
	keyStyle = lipgloss.NewStyle().Foreground(purple)
	// markStyle colours the ▸ of the selected room while the cursor is
	// somewhere else, so the mark alone carries the colour and the path stays
	// the room's name.
	markStyle = lipgloss.NewStyle().Foreground(pink)
	// teamStyle heads a team in the members pane.
	teamStyle = lipgloss.NewStyle().Foreground(violet)
	// promptStyle is the "> " the message pane writes in front of the line.
	promptStyle = lipgloss.NewStyle().Foreground(purple)
)
