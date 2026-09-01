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
to type. It attends the room as it starts, so a member has an inbox before its
first turn rather than from whenever it first says something.

It is configured, not typed: the harness spawns it and speaks MCP to it, so
stdout carries the protocol and nothing else. Logging is opt-in and goes to
stderr, which leaves the stream intact whether or not --log is given.

The identity token is resolved exactly as in every other member verb, so a
bridge configured with no token at all still inherits the one cmdman gave the
agent.`,
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

	// The token goes through raw: New resolves it through the same
	// cli.ResolveToken every member verb uses, and resolving it here first would
	// only move the "no identity token" error earlier for no gain.
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
