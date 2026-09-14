package commands

import "github.com/spf13/cobra"

func chatAdminDeleteRoomCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "delete-room <room>",
		Short: "Delete a room and everything said in it (admin)",
		Long: `delete-room removes a room's messages, the mentions in them and every read
position it holds, and reports how many messages went with it.

Nothing else ever removes a room: a room's row is created by the first
attendance or message in it and outlives every session that made it, so this is
the only cleanup there is.

It is refused while anybody attends the room. Ending those sessions is the
operator's move rather than the daemon's, and deleting under a live member
would leave that session addressing a room the broker no longer has.`,
		Example: `  crabswarm chat admin delete-room /work/proj \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminDeleteRoom(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatAdminDeleteRoom(cmd *cobra.Command, args []string, flags *chatFlags) error {
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.DeleteRoom(cmd.Context(), cmd.OutOrStdout(), identity, args[0])
}
