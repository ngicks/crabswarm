package commands

import (
	"strings"

	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatReadCmd(parent *cobra.Command, flags *chatFlags) {
	var (
		read chatcli.ReadFlags
		opts chatcli.ReadOptions
	)

	cmd := &cobra.Command{
		Use:   "read",
		Short: "Print messages of the room and move the read position",
		Long: `read prints messages of the caller's room, oldest first, and moves the
caller's read position to the newest one it showed — whichever cursor was used,
the tail included. A read of old history therefore never un-reads what came
after it. A read that found nothing says so instead of printing nothing.

--cursor says where to start. unread, the default, is what the caller has not
been shown: messages naming it or everyone, that it did not send. head is the
room's first message, tail its last. --range counts from there, forward when
positive and backward when negative; left out it is ten in the cursor's own
direction, so an unflagged read is the first ten unread and --cursor tail alone
is the last ten of the room. --to keeps only the messages naming any of the
given roles, written the way ` + "`chat send`" + ` writes a target, and --since
and --until bound the stretch by the sequence number each line begins with.

The two remaining flags are for harness hooks rather than for typing. --quiet
drops the line an empty read prints, so a hook can tell messages from none by
whether the output is empty at all. --done-when-empty additionally reports this
member done when the read handed nothing over, which is what re-arms the
daemon's terminal nudge for a member whose turn is ending. They belong to the
same process as the read on purpose: hooks wired to one event run concurrently,
so a separate report-state entry would race the delivering path and mark a
continuing turn done.`,
		Example: `  crabswarm chat read
  crabswarm chat read --range 5
  crabswarm chat read --cursor tail
  crabswarm chat read --cursor head --range 20 --to everyone
  crabswarm chat read --cursor head --since 120 --until 180 --range 100`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatRead(cmd, args, flags, read, opts)
		},
	}

	f := cmd.Flags()
	f.StringVar(&read.Cursor, "cursor", "unread",
		"where to read from: "+strings.Join(chatcli.ReadCursorNames(), ", "))
	f.Int32Var(&read.Range, "range", 0,
		"how many messages from the cursor, negative to count backward "+
			"(default ten in the cursor's direction)")
	f.StringVar(&read.To, "to", "",
		"only messages naming any of these roles, or everyone (default every target)")
	f.Int64Var(&read.Since, "since", 0,
		"only messages after this sequence number")
	f.Int64Var(&read.Until, "until", 0,
		"only messages before this sequence number")
	f.BoolVar(&opts.Quiet, "quiet", false,
		"print nothing at all when the read hands nothing over")
	f.BoolVar(&opts.DoneWhenEmpty, "done-when-empty", false,
		"report this member done when the read handed nothing over")
	// Fails only on a flag name this file does not declare, which the flag
	// above rules out.
	_ = cmd.RegisterFlagCompletionFunc("cursor",
		cobra.FixedCompletions(chatcli.ReadCursorNames(), cobra.ShellCompDirectiveNoFileComp))

	parent.AddCommand(cmd)
}

func runChatRead(
	cmd *cobra.Command,
	_ []string,
	flags *chatFlags,
	read chatcli.ReadFlags,
	opts chatcli.ReadOptions,
) error {
	filter, err := read.Filter()
	if err != nil {
		return err
	}
	opts.Filter = filter
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.ReadInto(cmd.Context(), cmd.OutOrStdout(), token, opts)
}
