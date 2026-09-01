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
		Long: `tui opens a screen on one room and keeps it current: the conversation as the
members have it, a sidebar naming everyone attending and the harness state each
last reported, and a line to send into the room from.

Watching needs no keypresses. i or enter moves to the input line, which takes
the addressing ` + "`chat admin send`" + ` takes — "name: text", "team/name: text",
or "*: text" for everyone — and enter sends it; esc leaves the line with what is
written on it still there. q or esc leaves the screen while watching, and ctrl-c
leaves it from anywhere, the input line included. Scrolling up holds the view
still while the room talks on, and scrolling back to the bottom follows it
again.

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
