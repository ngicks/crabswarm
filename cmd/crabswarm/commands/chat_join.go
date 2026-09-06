package commands

import (
	"strings"

	"github.com/spf13/cobra"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

func chatJoinCmd(parent *cobra.Command, flags *chatFlags) {
	var (
		flagName string
		flagKind string
	)

	cmd := &cobra.Command{
		Use:   "join --kind <" + strings.Join(chatcli.MemberKindNames(), "|") + ">",
		Short: "Attend the room this token belongs to",
		Long: `join declares attendance and prints the identity the daemon settled on.

The room and the team are derived from the identity token, not chosen here;
only the name within the team is, and leaving it out takes the default the
daemon derives — from the cmdman-compose command and scale-index labels when
present, otherwise from the token. Joining again with the same token is a
no-op.

--kind says what attends, and the daemon refuses a join that declares nothing.
An agent is woken by a line typed at its prompt when a message arrives, so pass
--kind agent from an agent harness. A human is inbox-only: messages wait until
` + "`crabswarm chat read`" + ` asks for them, and nothing is ever typed into the
terminal.`,
		Example: `  # from an agent harness, whose terminal an arriving message is typed into
  crabswarm chat join --kind agent
  # by hand: messages wait in the inbox
  crabswarm chat join --kind human --name reviewer`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatJoin(cmd, args, flags, flagName, flagKind)
		},
	}

	cmd.Flags().StringVar(&flagName, "name", "",
		"name to attend under, unique within the team "+
			"(default derived from compose labels, else the token)")
	cmd.Flags().StringVar(&flagKind, "kind", "",
		"required: what attends, "+
			strings.Join(chatcli.MemberKindNames(), " or ")+
			" — an agent is typed into, a human is inbox-only")
	// Both only fail on a flag name this file does not declare, which the flag
	// above rules out.
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.RegisterFlagCompletionFunc("kind",
		cobra.FixedCompletions(chatcli.MemberKindNames(), cobra.ShellCompDirectiveNoFileComp))

	parent.AddCommand(cmd)
}

func runChatJoin(
	cmd *cobra.Command,
	_ []string,
	flags *chatFlags,
	flagName string,
	flagKind string,
) error {
	kind, err := chatcli.ParseMemberKind(flagKind)
	if err != nil {
		return err
	}
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return err
	}
	defer client.Close()

	return client.Join(cmd.Context(), cmd.OutOrStdout(), token, flagName, kind)
}
