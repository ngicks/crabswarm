package commands

import "github.com/spf13/cobra"

func chatTeamCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Inspect and edit team formation across every room (admin)",
		Long: `team is the host's view of the chat topology: which rooms exist, who attends
them, and which team each member belongs to.

These are admin verbs. They are authenticated by the age identity file named by
--identity rather than by a member token, so a participant cannot reshape the
rooms it lives in.`,
	}

	chatTeamListCmd(cmd, flags)
	chatTeamMoveCmd(cmd, flags)

	parent.AddCommand(cmd)
}
