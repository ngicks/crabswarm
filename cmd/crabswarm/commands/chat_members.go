package commands

import "github.com/spf13/cobra"

func chatMembersCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:     "members",
		Aliases: []string{"ls"},
		Short:   "List everyone attending the room",
		Long: `members lists the caller's room, one team-qualified member per line. Each
line is exactly the address ` + "`chat send`" + ` takes.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatMembers(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatMembers(cmd *cobra.Command, _ []string, flags *chatFlags) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.ListMembers(cmd.Context(), cmd.OutOrStdout(), token)
}
