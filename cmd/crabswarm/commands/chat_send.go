package commands

import "github.com/spf13/cobra"

func chatSendCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "send <name|team/name> <text>",
		Short: "Send a message to one member of the room",
		Long: `send delivers a message to a single member of the caller's room.

A bare name resolves within the caller's team first, then room-wide when it is
unique there. A name that several teams use is rejected, and the error names
the "team/name" form to retry with.`,
		Example: `  crabswarm chat send alice "rebased onto main"
  crabswarm chat send backend/alice "rebased onto main"`,
		Args: cobra.ExactArgs(2),
		ValidArgsFunction: func(
			cmd *cobra.Command,
			args []string,
			_ string,
		) ([]string, cobra.ShellCompDirective) {
			return completeChatMembers(cmd, args, flags)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatSend(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatSend(cmd *cobra.Command, args []string, flags *chatFlags) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Send(cmd.Context(), cmd.OutOrStdout(), token, args[0], args[1])
}
