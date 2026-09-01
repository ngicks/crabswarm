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
running under cmdman in one directory and the people working alongside them.

Room and team are never chosen: they follow from the identity token every
member verb carries, which is taken from --token, else $CRABSWARM_CHAT_TOKEN,
else $CMDMAN_CMD_ID. Anything cmdman runs — a harness or the shell someone
types in — inherits the last one and needs no setup; whoever attends from
outside passes the token that ` + "`chat admin register`" + ` printed.

Being typed into is a separate choice from the token: ` + "`join --agent`" + ` asks for
it, and a join without the flag is inbox-only.

The ` + "`admin`" + ` group is host-only and proves it by decrypting a challenge
with the age identity file named by --identity, rather than by carrying a
token. Attending no room, its verbs name the room they act on.`,
		Example: `  # attending by hand: messages wait in the inbox
  crabswarm chat join --name reviewer
  # attending from a harness: an arriving message is typed at its prompt
  crabswarm chat join --agent
  crabswarm chat send backend/alice "PR is ready"
  crabswarm chat read`,
		// Runnable with NoArgs rather than a bare group: cobra returns help for a
		// command it cannot run before it ever validates the arguments, so a group
		// parent left non-runnable answers `chat register ...` — a verb that has
		// moved under `admin` — with help and a success exit instead of saying the
		// spelling is gone.
		Args: cobra.NoArgs,
		RunE: runChat,
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
	chatHistoryCmd(cmd, flags)
	chatMembersCmd(cmd, flags)
	chatLeaveCmd(cmd, flags)
	chatReportStateCmd(cmd, flags)
	chatMCPCmd(cmd, flags)
	chatAdminCmd(cmd, flags)

	parent.AddCommand(cmd)
}

func runChat(cmd *cobra.Command, _ []string) error {
	return cmd.Help()
}
