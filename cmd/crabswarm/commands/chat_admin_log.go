package commands

import (
	"strings"

	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatAdminLogCmd(parent *cobra.Command, flags *chatFlags) {
	var read chatcli.ReadFlags

	cmd := &cobra.Command{
		Use:   "log <room>",
		Short: "Print a room's conversation without attending it (admin)",
		Long: `log prints a stretch of a named room's conversation, oldest first, and moves
nothing: the members' own read positions are left exactly where they were.

It is what the room said rather than what any one member was shown — every
line names who it was addressed to, and a board post is addressed to "-".
Messages the admin sent are in it as well, under the reserved sender "admin".

--cursor starts the stretch at the room's first or last message, tail by
default, and --range counts from there: forward when positive, backward when
negative, and ten in the cursor's own direction when left out, so an unflagged
run prints the last ten. --to, --since and --until narrow it further. The
unread cursor belongs to a member read and is refused here: unread is measured
from a read position, and an operator attending no room has none.`,
		Example: `  crabswarm chat admin log /work/proj --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat admin log /work/proj --cursor head --range 200 \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminLog(cmd, args, flags, read)
		},
	}

	f := cmd.Flags()
	f.StringVar(&read.Cursor, "cursor", "tail",
		"where to read from: "+strings.Join(chatcli.HistoryCursorNames(), ", "))
	f.Int32Var(&read.Range, "range", 0,
		"how many messages from the cursor, negative to count backward "+
			"(default ten in the cursor's direction)")
	f.StringVar(&read.To, "to", "",
		"only messages naming any of these roles, or everyone (default every target)")
	f.Int64Var(&read.Since, "since", 0,
		"only messages after this sequence number")
	f.Int64Var(&read.Until, "until", 0,
		"only messages before this sequence number")
	// Fails only on a flag name this file does not declare, which the flag
	// above rules out.
	_ = cmd.RegisterFlagCompletionFunc("cursor",
		cobra.FixedCompletions(chatcli.HistoryCursorNames(), cobra.ShellCompDirectiveNoFileComp))

	parent.AddCommand(cmd)
}

func runChatAdminLog(
	cmd *cobra.Command,
	args []string,
	flags *chatFlags,
	read chatcli.ReadFlags,
) error {
	filter, err := read.HistoryFilter()
	if err != nil {
		return err
	}
	identity, err := chatIdentityPath(flags)
	if err != nil {
		return err
	}
	client, err := dialChat(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.AdminLog(cmd.Context(), cmd.OutOrStdout(), identity, args[0], filter)
}
