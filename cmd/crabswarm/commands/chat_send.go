package commands

import (
	"strings"

	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatSendCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   `send <everyone|role[,role...]|""> <text>`,
		Short: "Send a message into the room, to somebody or to nobody",
		Long: `send appends a message to the caller's room, addressed by its first argument.

"everyone" is the whole room. A comma-separated list of roles mentions each of
them, written as "team/name" or as a bare "name" that resolves within the
caller's team first, then room-wide when it is unique there; a name several
teams use is rejected, and the error names the "team/name" form to retry with.
An empty target — the literal "" — is a board post: it is in the room and
mentions nobody, so it interrupts no one.

A send prints one line per role it mentioned, and warns on stderr about any of
them nobody is attending under; that mention waits at the role's read position
until somebody does. Everyone and a board post name no role in particular, so
they print nothing at all.`,
		Example: `  crabswarm chat send everyone "main is red"
  crabswarm chat send devenv/claude-1 "please review"
  crabswarm chat send claude-1,devenv/claude-2 "please review"
  crabswarm chat send "" "fyi: rebased main"`,
		Args: cobra.MinimumNArgs(2),
		ValidArgsFunction: func(
			cmd *cobra.Command,
			args []string,
			_ string,
		) ([]string, cobra.ShellCompDirective) {
			return completeChatTargets(cmd, args, flags)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatSend(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatSend(cmd *cobra.Command, args []string, flags *chatFlags) error {
	target, err := chatcli.ParseTarget(args[0])
	if err != nil {
		return err
	}
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	resp, err := client.Send(cmd.Context(), token, target, strings.Join(args[1:], " "))
	if err != nil {
		return err
	}
	return chatcli.RenderSent(cmd.OutOrStdout(), cmd.ErrOrStderr(), resp)
}
