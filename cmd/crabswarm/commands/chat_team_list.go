package commands

import "github.com/spf13/cobra"

func chatTeamListCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls"},
		Short:             "List every room, its teams and their members (admin)",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatTeamList(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatTeamList(cmd *cobra.Command, _ []string, flags *chatFlags) error {
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
