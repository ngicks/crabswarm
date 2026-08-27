package commands

import "github.com/spf13/cobra"

func chatLeaveCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "leave",
		Short: "Withdraw from the room",
		Long: `leave withdraws the caller's attendance. Any message still pending for it
is dropped with it.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatLeave(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatLeave(cmd *cobra.Command, _ []string, flags *chatFlags) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Leave(cmd.Context(), cmd.OutOrStdout(), token)
}
