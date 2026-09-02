package commands

import (
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli/tui"
)

func chatAdminTUICmd(parent *cobra.Command, flags *chatFlags) {
	var flagRoom string

	cmd := &cobra.Command{
		Use:   "tui [--room <room>]",
		Short: "Watch a room's conversation on a live screen (admin)",
		Long: `tui opens a screen on one room and keeps it current in four framed panes:
the rooms the daemon knows and the room's members on the left, the conversation
as the members have it and the message to send into the room on the right.

Watching needs no keypresses. ctrl+h, ctrl+j, ctrl+k and ctrl+l move between the
panes, and every other key goes to the pane that has focus: j/k, gg/G and
ctrl+d/ctrl+u scroll the conversation and move a cursor in the rooms and members
panes, where enter switches to the room under it or writes the member under it
in front of the message.

The message is addressed with an @: the first ` + "`@name`" + `, ` + "`@team/name`" + ` or
` + "`@team/*`" + ` in it says who it is for, and a message with no @ at all goes to
everyone in the room. The text is sent whole, that token included, so the room
reads who was asked; a backticked ` + "`@`" + ` and a \@ are text and address nobody.
tab after an @ completes it against the room's members. enter is always a
newline — ctrl+enter sends, and so does ctrl+x, which is the key for terminals
that cannot report the first. q leaves the screen from the three panes that are
lists, and ctrl-c leaves it from anywhere, the message included. Scrolling up
holds the view still while the room talks on, and scrolling back to the bottom
follows it again.

--room says which room the screen opens on; without it the screen opens on the
first room the daemon lists and the rooms pane switches between them. A room
that is named but not known is refused before the screen opens.`,
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
	})
}
