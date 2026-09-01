package commands

import "github.com/spf13/cobra"

func chatAdminRegisterCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "register <room> <team> <name>",
		Short: "Register a member the team-info provider cannot vouch for (admin)",
		Long: `register mints an identity token for a member no provider knows — a human on
the host, who runs under no cmdman command to be recognized by.

The token is printed once and stored nowhere else in readable form: pass it to
the member verbs as --token or $CRABSWARM_CHAT_TOKEN. Being an admin verb, this
is authenticated by the age identity file named by --identity, not by a token.`,
		Example: `  crabswarm chat admin register /work/proj humans yuki \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.ExactArgs(3),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminRegister(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatAdminRegister(cmd *cobra.Command, args []string, flags *chatFlags) error {
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.RegisterMember(
		cmd.Context(), cmd.OutOrStdout(), identity, args[0], args[1], args[2])
}
