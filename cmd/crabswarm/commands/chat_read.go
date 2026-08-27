package commands

import (
	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatReadCmd(parent *cobra.Command, flags *chatFlags) {
	var opts chatcli.ReadOptions

	cmd := &cobra.Command{
		Use:   "read",
		Short: "Print the pending messages and consume them",
		Long: `read prints the caller's pending messages, oldest first, and consumes them:
a message is handed out exactly once, so a second read shows only what arrived
in between. An empty inbox says so instead of printing nothing.

The two flags are for harness hooks rather than for typing. --quiet drops the
empty-inbox line, so a hook can tell mail from no mail by whether the output is
empty at all. --idle-when-empty additionally reports this member idle when the
read handed nothing over, which is what re-arms the daemon's terminal nudge for
a member whose turn is ending. They belong to the same process as the read on
purpose: hooks wired to one event run concurrently, so a separate report-state
entry would race the delivering path and mark a continuing turn idle.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatRead(cmd, args, flags, opts)
		},
	}

	f := cmd.Flags()
	f.BoolVar(&opts.Quiet, "quiet", false,
		"print nothing at all when the inbox is empty")
	f.BoolVar(&opts.IdleWhenEmpty, "idle-when-empty", false,
		"report this member idle when the read handed nothing over")

	parent.AddCommand(cmd)
}

func runChatRead(
	cmd *cobra.Command,
	_ []string,
	flags *chatFlags,
	opts chatcli.ReadOptions,
) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Read(cmd.Context(), cmd.OutOrStdout(), token, opts)
}
