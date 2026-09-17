package commands

import "github.com/spf13/cobra"

func chatMembersCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:     "members",
		Aliases: []string{"ls"},
		Short:   "List everyone attending the room",
		Long: `members lists the caller's room, one member per line: the team-qualified
address, the kind, the harness state, the harness and how a mention is
delivered. The first column is exactly the address ` + "`chat send`" + ` takes.

The kind says whether a message reaches that member on its own — an agent is
woken by an arriving message, a human reads the room when it asks — so it is
what tells the sender who to expect an answer from.

The harness names the CLI the member runs: claude-code, codex, opencode or
other. The delivery says by what route a mention gets there: terminal, where
the daemon types it into the member's terminal, or native, where the member's
own server pushes it through the harness. Either reads as "-" where nobody
declared one, which is what a human carries.`,
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
