package commands

import (
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli/tui"
)

func chatAdminTUICmd(parent *cobra.Command, flags *chatFlags) {
	var flagRoom string

	cmd := &cobra.Command{
		Use:   "tui --room <room>",
		Short: "Watch a room's conversation on a live screen (admin)",
		Long: `tui opens a screen on one room and keeps it current in four framed panes:
the rooms the daemon knows and the room's members on the left, the conversation
as the members have it and a line to send into the room from on the right.

Watching needs no keypresses. ctrl+h, ctrl+j, ctrl+k and ctrl+l move between the
panes, and every other key goes to the pane that has focus: j/k, gg/G and
ctrl+d/ctrl+u scroll the conversation, and the message line takes the addressing
` + "`chat admin send`" + ` takes — "name: text", "team/name: text", or "*: text"
for everyone — with enter sending it. q leaves the screen from the three panes
that are lists, and ctrl-c leaves it from anywhere, the message line included.
Scrolling up holds the view still while the room talks on, and scrolling back to
the bottom follows it again.

The room is named explicitly and has no default: the admin attends none, and a
name the daemon does not know is refused before the screen opens.`,
		Example: `  crabswarm chat admin tui --room /work/proj \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminTUI(cmd, args, flags, flagRoom)
		},
	}

	cmd.Flags().StringVar(&flagRoom, "room", "", "the room to watch")
	_ = cmd.MarkFlagRequired("room")

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
