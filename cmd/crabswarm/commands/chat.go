package commands

import "github.com/spf13/cobra"

// chatFlags bundles what the chat subcommands share: the root's --sock and
// --config, which are only final once the root has parsed them (hence the
// pointers), and chat's own two credentials. A struct rather than four
// arguments — they are all *string, and a swapped pair would compile.
type chatFlags struct {
	sock     *string
	config   *string
	token    *string
	identity *string
}

func chatCmd(parent *cobra.Command, flagSock, flagConfig *string) {
	var (
		flagToken    string
		flagIdentity string
	)

	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Talk to the other agents and humans attending this room",
		Long: `chat brokers messages between the participants of a room — the agents
running under cmdman in one directory and the humans registered into it.

Room and team are never chosen: they follow from the identity token every
member verb carries, which is taken from --token, else $CRABSWARM_CHAT_TOKEN,
else $CMDMAN_CMD_ID. An agent inherits the last one from cmdman and needs no
setup; a human passes the token that ` + "`chat register`" + ` printed.

The admin verbs (register, team) are host-only and prove it by decrypting a
challenge with the age identity file named by --identity, rather than by
carrying a token.`,
		Example: `  crabswarm chat join --name reviewer
  crabswarm chat send backend/alice "PR is ready"
  crabswarm chat read`,
	}

	cmd.PersistentFlags().StringVar(&flagToken, "token", "",
		"identity token to act as (default $CRABSWARM_CHAT_TOKEN, else $CMDMAN_CMD_ID)")
	cmd.PersistentFlags().StringVar(&flagIdentity, "identity", "",
		"age identity file authenticating the admin verbs")

	flags := &chatFlags{
		sock:     flagSock,
		config:   flagConfig,
		token:    &flagToken,
		identity: &flagIdentity,
	}

	chatJoinCmd(cmd, flags)
	chatSendCmd(cmd, flags)
	chatBroadcastCmd(cmd, flags)
	chatReadCmd(cmd, flags)
	chatMembersCmd(cmd, flags)
	chatLeaveCmd(cmd, flags)
	chatReportStateCmd(cmd, flags)
	chatMCPCmd(cmd, flags)
	chatRegisterCmd(cmd, flags)
	chatTeamCmd(cmd, flags)

	parent.AddCommand(cmd)
}
