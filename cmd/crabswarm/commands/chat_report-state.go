package commands

import (
	"strings"

	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatReportStateCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "report-state <" + strings.Join(chatcli.HarnessStateNames(), "|") + ">",
		Short: "Record the state of the harness this member runs under",
		Long: `report-state tells the daemon what the caller's harness is doing. It is
driven by harness hooks rather than typed by hand, and prints nothing so its
output never reaches the agent reading the hook's stdout.

The state gates keystroke-injection nudges, which are only safe to deliver
while the harness is idle: running means it is working through a turn, and
waiting_input means a dialog is open that would read a nudge as its answer.`,
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: chatcli.HarnessStateNames(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatReportState(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatReportState(cmd *cobra.Command, args []string, flags *chatFlags) error {
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.ReportState(cmd.Context(), token, args[0])
}
