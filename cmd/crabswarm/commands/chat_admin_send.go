package commands

import (
	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatAdminSendCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "send <room> <name|team/name|team/*|*> <text>",
		Short: "Send a message into a room without attending it (admin)",
		Long: `send delivers a message into a room the admin does not attend, addressed the
way ` + "`chat send`" + ` addresses one — a bare name, a "team/name" pair — or,
as "team/*", to every member of that team and, as "*", to everyone in the room.
Quote the star so the shell hands it over instead of expanding it against the
working directory.

A team is counted when the message is sent: whoever attends it then receives it.

The message arrives attributed to the reserved sender "admin" and creates no
member, so there is nobody there to address back.`,
		Example: `  crabswarm chat admin send /work/proj backend/alice "rebased onto main" \
    --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat admin send /work/proj 'backend/*' "who is on the rebase" \
    --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat admin send /work/proj '*' "standup in five" \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.ExactArgs(3),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminSend(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatAdminSend(cmd *cobra.Command, args []string, flags *chatFlags) error {
	target, err := chatcli.ParseAdminTarget(args[1])
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

	return client.AdminSend(
		cmd.Context(), cmd.OutOrStdout(), identity, args[0], target, args[2])
}
