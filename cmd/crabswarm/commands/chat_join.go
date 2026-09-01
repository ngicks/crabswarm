package commands

import "github.com/spf13/cobra"

func chatJoinCmd(parent *cobra.Command, flags *chatFlags) {
	var (
		flagName  string
		flagAgent bool
	)

	cmd := &cobra.Command{
		Use:   "join",
		Short: "Attend the room this token belongs to",
		Long: `join declares attendance and prints the identity the daemon settled on.

The room and the team are derived from the identity token, not chosen here;
only the name within the team is, and leaving it out takes the default the
daemon derives — from the cmdman-compose command and scale-index labels when
present, otherwise from the token. Joining again with the same token is a
no-op.

Without --agent the member is inbox-only: messages wait in the inbox until
` + "`crabswarm chat read`" + ` asks for them, and nothing is ever typed into this
terminal. Pass --agent from an agent harness that should be woken by a line
typed at its prompt when a message arrives.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatJoin(cmd, args, flags, flagName, flagAgent)
		},
	}

	cmd.Flags().StringVar(&flagName, "name", "",
		"name to attend under, unique within the team "+
			"(default derived from compose labels, else the token)")
	cmd.Flags().BoolVar(&flagAgent, "agent", false,
		"attend as an agent harness, whose terminal an arriving message "+
			"is typed into")

	parent.AddCommand(cmd)
}

func runChatJoin(
	cmd *cobra.Command,
	_ []string,
	flags *chatFlags,
	flagName string,
	flagAgent bool,
) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Join(cmd.Context(), cmd.OutOrStdout(), token, flagName, flagAgent)
}
