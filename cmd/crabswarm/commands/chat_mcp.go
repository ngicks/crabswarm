package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/chat/mcpserver"
)

func chatMCPCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve the chat verbs as MCP tools over stdio",
		Long: `mcp runs the per-agent bridge a harness starts as its own stdio subprocess,
offering the chat verbs as tools instead of commands an agent has to remember
to type. It attends the room as it starts and holds that attendance for the
whole session, so the room has the member before its first turn rather than
from whenever it first says something — and has it again once a daemon that
went away comes back.

It is configured, not typed: the harness spawns it and speaks MCP to it, so
stdout carries the protocol and nothing else. Logging goes to stderr, warnings
and above by default and whatever --log asks for otherwise, which leaves the
stream intact either way.

The identity token is resolved exactly as in every other member verb, so a
bridge configured with no token at all still inherits the one cmdman gave the
agent. A bridge that resolves none still serves, and every tool answers with
what is missing.`,
		Example: `  crabswarm chat mcp
  crabswarm chat mcp --sock /run/user/1000/crabswarm/daemon.sock`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChatMCP(cmd, args, flags)
		},
	}

	parent.AddCommand(cmd)
}

func runChatMCP(cmd *cobra.Command, _ []string, flags *chatFlags) error {
	ctx := cmd.Context()

	sock, err := chatSockPath(cmd, flags)
	if err != nil {
		return err
	}

	// The token goes through raw: the bridge resolves it through the same
	// cli.ResolveToken every member verb uses, where it uses it. Resolving it
	// here would end the process before the harness got its handshake, which is
	// the one outcome nobody can read a reason out of.
	srv, err := mcpserver.New(commandLogger(cmd), sock, *flags.token)
	if err != nil {
		return err
	}

	// Run reports a signalled shutdown as ctx.Err(), which is a clean exit here
	// the way it is for `serve` — those services swallow it and return nil, and
	// a bridge torn down with its harness has not failed. The guard is against
	// this ctx rather than the bare sentinel, so a context.Canceled raised
	// anywhere else still surfaces as the error it is.
	if err := srv.Run(ctx); err != nil && !errors.Is(err, ctx.Err()) {
		return err
	}
	return nil
}
