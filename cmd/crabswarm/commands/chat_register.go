package commands

import "github.com/spf13/cobra"

func chatRegisterCmd(parent *cobra.Command, flags *chatFlags) {
	var (
		flagRoom string
		flagTeam string
		flagName string
	)

	cmd := &cobra.Command{
		Use:   "register --room ROOM --team TEAM --name NAME",
		Short: "Register a member the team-info provider cannot vouch for (admin)",
		Long: `register mints an identity token for a member no provider knows — a human on
the host, who runs under no cmdman command to be recognized by.

The token is printed once and stored nowhere else in readable form: pass it to
the member verbs as --token or $CRABSWARM_CHAT_TOKEN. Being an admin verb, this
is authenticated by the age identity file named by --identity, not by a token.`,
		Example: `  crabswarm chat register --room /work/proj --team humans --name yuki \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatRegister(cmd, args, flags, flagRoom, flagTeam, flagName)
		},
	}

	cmd.Flags().StringVar(&flagRoom, "room", "", "room to register the member into")
	cmd.Flags().StringVar(&flagTeam, "team", "", "team to register the member under")
	cmd.Flags().StringVar(&flagName, "name", "", "name to register, unique within the team")
	_ = cmd.MarkFlagRequired("room")
	_ = cmd.MarkFlagRequired("team")
	_ = cmd.MarkFlagRequired("name")

	parent.AddCommand(cmd)
}

func runChatRegister(
	cmd *cobra.Command,
	_ []string,
	flags *chatFlags,
	flagRoom, flagTeam, flagName string,
) error {
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
		cmd.Context(), cmd.OutOrStdout(), identity, flagRoom, flagTeam, flagName)
}
