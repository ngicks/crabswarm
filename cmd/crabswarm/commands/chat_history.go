package commands

import (
	"github.com/spf13/cobra"
)

func chatHistoryCmd(parent *cobra.Command, flags *chatFlags) {
	var flagLimit int32

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Print the room's conversation without consuming it",
		Long: `history prints the tail of this room's conversation, oldest first, and
consumes nothing: it can be read again, and it shows what was said before this
member joined as readily as what arrived a moment ago.

It is the whole room, not an inbox: messages addressed to other members appear
too, spelled with the addressee after the arrow, and a broadcast is addressed to
"*". An unflagged run prints the 50 most recent entries; --limit asks for a
window of another size, and the daemon's retention cap bounds how far back
either of them reaches.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatHistory(cmd, args, flags, flagLimit)
		},
	}

	cmd.Flags().Int32Var(&flagLimit, "limit", 0,
		"how many of the most recent entries to print (default the daemon's own window)")

	parent.AddCommand(cmd)
}

func runChatHistory(cmd *cobra.Command, _ []string, flags *chatFlags, limit int32) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.History(cmd.Context(), cmd.OutOrStdout(), token, limit)
}
