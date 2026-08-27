package commands

import "github.com/spf13/cobra"

func chatBroadcastCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "broadcast <text>",
		Short: "Send a message to everyone in the room",
		Long: `broadcast delivers a message to every member of the caller's room, across
teams, and reports how many inboxes it reached. The caller is not counted.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatBroadcast(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatBroadcast(cmd *cobra.Command, args []string, flags *chatFlags) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Broadcast(cmd.Context(), cmd.OutOrStdout(), token, args[0])
}
