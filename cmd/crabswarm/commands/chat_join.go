package commands

import "github.com/spf13/cobra"

func chatJoinCmd(parent *cobra.Command, flags *chatFlags) {
	var flagName string

	cmd := &cobra.Command{
		Use:   "join",
		Short: "Attend the room this token belongs to",
		Long: `join declares attendance and prints the identity the daemon settled on.

The room and the team are derived from the identity token, not chosen here;
only the name within the team is, and leaving it out takes the name the daemon
derives from the token. Joining again with the same token is a no-op.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatJoin(cmd, args, flags, flagName)
		},
	}

	cmd.Flags().StringVar(&flagName, "name", "",
		"name to attend under, unique within the team (default derived from the token)")

	parent.AddCommand(cmd)
}

func runChatJoin(cmd *cobra.Command, _ []string, flags *chatFlags, flagName string) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Join(cmd.Context(), cmd.OutOrStdout(), token, flagName)
}
