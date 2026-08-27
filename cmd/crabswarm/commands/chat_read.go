package commands

import "github.com/spf13/cobra"

func chatReadCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "read",
		Short: "Print the pending messages and consume them",
		Long: `read prints the caller's pending messages, oldest first, and consumes them:
a message is handed out exactly once, so a second read shows only what arrived
in between. An empty inbox says so instead of printing nothing.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatRead(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatRead(cmd *cobra.Command, _ []string, flags *chatFlags) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Read(cmd.Context(), cmd.OutOrStdout(), token)
}
