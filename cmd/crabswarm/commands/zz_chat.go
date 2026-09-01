package commands

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm"
	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// chatSockPath resolves the daemon socket the way every client command does —
// layered config with an explicitly-set --sock on top. It is separate from
// dialChat because the MCP bridge dials for itself and needs only the path.
func chatSockPath(cmd *cobra.Command, flags *chatFlags) (string, error) {
	cfg, err := crabswarm.LoadConfig(*flags.config)
	if err != nil {
		return "", err
	}
	if cmd.Flags().Changed("sock") {
		cfg.Sock = *flags.sock
	}
	return cfg.Sock, nil
}

// dialChat returns a client for the resolved daemon socket. The caller closes
// it.
func dialChat(cmd *cobra.Command, flags *chatFlags) (*chatcli.Client, error) {
	sock, err := chatSockPath(cmd, flags)
	if err != nil {
		return nil, err
	}
	return chatcli.Dial(sock)
}

// dialChatAsMember resolves the caller's identity token and dials the daemon,
// which is what every member verb needs before it can do anything.
func dialChatAsMember(
	cmd *cobra.Command,
	flags *chatFlags,
) (client *chatcli.Client, token string, err error) {
	token, err = chatcli.ResolveToken(*flags.token)
	if err != nil {
		return nil, "", err
	}
	client, err = dialChat(cmd, flags)
	if err != nil {
		return nil, "", err
	}
	return client, token, nil
}

// chatIdentityPath resolves the age identity file the admin verbs authenticate
// with: --identity first, then the chat config's admin_identity_file.
func chatIdentityPath(flags *chatFlags) (string, error) {
	cfg, err := crabswarm.LoadConfig(*flags.config)
	if err != nil {
		return "", err
	}
	return chatcli.ResolveIdentityPath(*flags.identity, cfg.Chat.AdminIdentityFile)
}

// chatCompletionTimeout bounds the lookup behind a completion so a stopped
// daemon costs the shell a pause, not a hang.
const chatCompletionTimeout = time.Second

// completeChatMembers completes the address argument of `chat send` with the
// members the daemon reports for the caller's room. It is best-effort: a
// missing token, an unreachable daemon or a caller that has not joined yet all
// degrade to no suggestions rather than surfacing an error into the shell.
func completeChatMembers(
	cmd *cobra.Command,
	args []string,
	flags *chatFlags,
) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	client, token, err := dialChatAsMember(cmd, flags)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(cmd.Context(), chatCompletionTimeout)
	defer cancel()

	members, err := client.MemberAddresses(ctx, token)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return members, cobra.ShellCompDirectiveNoFileComp
}
