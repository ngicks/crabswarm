package commands

import "github.com/spf13/cobra"

func chatAdminListCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List every room, its teams and their members (admin)",
		Long: `list prints the whole topology as a room → team → member tree, each member
followed by its kind — agent for a harness an arriving message is typed into,
human for someone who reads an inbox.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminList(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatAdminList(cmd *cobra.Command, _ []string, flags *chatFlags) error {
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.ListRooms(cmd.Context(), cmd.OutOrStdout(), identity)
}
