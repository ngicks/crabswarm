package commands

import (
	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli/tui"
)

func chatAdminTUICmd(parent *cobra.Command, flags *chatFlags) {
	var flagRoom string

	cmd := &cobra.Command{
		Use:   "tui [--room <room>]",
		Short: "Watch a room's conversation on a live screen (admin)",
		Long: `tui opens a screen on one room and keeps it current. Four framed panes fill
it — the rooms the daemon knows over that room's members on the left, the
conversation over the message to send into it on the right — with two lines
under them: a system line that reports how the last send went, an editor that
would not run or a message addressed to nobody, and a status bar naming the
room, whether the view is still following it, how the daemon is answering, and
the keys the screen cannot show.

--room says which room the screen opens on; without it the screen opens on the
first room the daemon lists, and the rooms pane switches between them from
there. A room that is named but not known is refused before the screen opens.

Watching needs no keypresses. ctrl+h, ctrl+j, ctrl+k and ctrl+l move between the
panes, and every other key belongs to the pane that has focus. In the three that
are lists — rooms, members and the conversation — j and k move a line, gg and G
go to the ends, and ctrl+d and ctrl+u move half a page. enter in the rooms pane
switches to the room under the cursor; enter in the members pane writes
` + "`@team/name `" + ` in front of the message — ` + "`@team/* `" + ` on a team
heading — and moves focus there. Scrolling the conversation up holds the view
still while the room talks on, and scrolling back to the bottom follows it
again.

Below 60 columns the two columns no longer fit side by side. The screen then
shows one of them at a time, and ctrl+h or ctrl+l brings the other on screen.

In the message pane enter is a newline — the completion list described below is
the one thing that takes it: ctrl+enter sends, and so does ctrl+x, which is the
key for terminals that cannot report the first. ctrl+g opens the draft in
$VISUAL, or $EDITOR when VISUAL is unset, and takes back whatever the editor
leaves; nothing is sent by it. q leaves the screen from the three panes that are
lists — in the message pane it is a letter — and ctrl+c leaves it from anywhere.

Tab after an @ completes the token against the room's members. One match is
applied outright; two or more open a list above the message, where tab and j
move the highlight down, shift+tab and k move it up, enter accepts the row it is
on, and esc closes the list and leaves the token as typed. Each team is offered
as a ` + "`team/*`" + ` row above its own members, so a whole team is one completion away.

The message is addressed with an @: the first bare @token in it — ` + "`@name`" + `,
` + "`@team/name`" + ` or ` + "`@team/*`" + ` — says who it is for, and a message with no @ at all
goes to everyone in the room. Bare means the @ starts a word — it is the first
character of the message, or a space or a newline is in front of it — outside a
backtick span and not written as \@. Anything else is text and addresses nobody,
so ops@corp.example goes to the whole room. The text is sent whole, that token
included, so the room reads who was asked.`,
		Example: `  crabswarm chat admin tui
  crabswarm chat admin tui --room /work/proj \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminTUI(cmd, args, flags, flagRoom)
		},
	}

	cmd.Flags().StringVar(&flagRoom, "room", "",
		"the room to watch; the first room listed when unset")

	parent.AddCommand(cmd)
}

func runChatAdminTUI(
	cmd *cobra.Command,
	_ []string,
	flags *chatFlags,
	room string,
) error {
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	admin := client.Admin(identity)
	return tui.Run(cmd.Context(), tui.Deps{
		Room:   room,
		Log:    admin,
		Roster: admin,
		Sender: admin,
		// The screen never reads the environment itself: the editor ctrl+g
		// runs is resolved here, out of the one place $VISUAL and $EDITOR are
		// read.
		Editor: chatcli.EditorFromEnv(),
	})
}
