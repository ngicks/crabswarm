package commands

import "github.com/spf13/cobra"

func chatAdminLogCmd(parent *cobra.Command, flags *chatFlags) {
	var flagLimit int32

	cmd := &cobra.Command{
		Use:   "log <room>",
		Short: "Print a room's conversation without attending it (admin)",
		Long: `log prints the tail of a named room's conversation, oldest first, and consumes
nothing: the members' own reads are left with everything they had pending.

It is what the room said, not what any one member received — messages addressed
to a single member appear too, spelled with the addressee after the arrow, and a
broadcast is addressed to "*". Messages the admin sent into the room are in it
as well, under the reserved sender "admin". An unflagged run prints the 50 most
recent entries; --limit asks for a window of another size, and the daemon's
retention cap bounds how far back either of them reaches.`,
		Example: `  crabswarm chat admin log /work/proj --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat admin log /work/proj --limit 200 \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminLog(cmd, args, flags, flagLimit)
		},
	}

	cmd.Flags().Int32Var(&flagLimit, "limit", 0,
		"how many of the most recent entries to print (default the daemon's own window)")

	parent.AddCommand(cmd)
}

func runChatAdminLog(cmd *cobra.Command, args []string, flags *chatFlags, limit int32) error {
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.AdminLog(cmd.Context(), cmd.OutOrStdout(), identity, args[0], limit)
}
