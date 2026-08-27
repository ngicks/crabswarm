package commands

import "github.com/spf13/cobra"

func chatTeamMoveCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "move <room> <team>/<name> <new-team>",
		Short: "Move a member to another team within its room (admin)",
		Long: `move reassigns one member to another team of the same room.

The member is addressed the way every other chat command spells one, as
"team/name", so a line of ` + "`chat team list`" + ` can be pasted in. The room
is named separately because a name may repeat across rooms.`,
		Example: `  crabswarm chat team move /work/proj backend/alice frontend \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.ExactArgs(3),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatTeamMove(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatTeamMove(cmd *cobra.Command, args []string, flags *chatFlags) error {
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.MoveMember(
		cmd.Context(), cmd.OutOrStdout(), identity, args[0], args[1], args[2])
}
