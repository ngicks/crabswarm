package commands

import (
	"strings"

	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatAdminSendCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   `send <room> <everyone|role[,role...]|""> <text>`,
		Short: "Send a message into a room without attending it (admin)",
		Long: `send delivers a message into a room the admin does not attend, addressed the
way ` + "`chat send`" + ` addresses one: "everyone" for the whole room, a
comma-separated list of roles as "team/name" or "name", or an empty "" for a
board post that mentions nobody.

The admin is in no team, so a bare name is looked for across the whole room; a
name two teams carry is refused as ambiguous, and the error names them.

The message arrives attributed to the reserved sender "admin" and creates no
member, so there is nobody there to address back. It prints one line per role
it mentioned and warns on stderr about any of them nobody is attending under.`,
		Example: `  crabswarm chat admin send /work/proj backend/alice "rebased onto main" \
    --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat admin send /work/proj everyone "standup in five" \
    --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat admin send /work/proj "" "deploy is frozen" \
    --identity ~/.config/crabswarm/chat_admin.key`,
		Args:              cobra.MinimumNArgs(3),
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatAdminSend(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatAdminSend(cmd *cobra.Command, args []string, flags *chatFlags) error {
	target, err := chatcli.ParseTarget(args[1])
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

	return client.AdminSend(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr(),
		identity, args[0], target, strings.Join(args[2:], " "))
}
