package commands

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/mcp"
	mcpchat "github.com/ngicks/crabswarm/crabswarm/mcp/chat"
)

// The transports `crabswarm mcp` serves over.
const (
	mcpTransportStdio = "stdio"
	mcpTransportHTTP  = "http"
)

func mcpCmd(parent *cobra.Command, flagSock, flagConfig *string) {
	var (
		flagToken     string
		flagTransport string
		flagListen    string
	)

	// The admin identity is left out: nothing this command does authenticates
	// as the host, so the helpers it shares with `chat` never read that field.
	flags := &chatFlags{
		sock:   flagSock,
		config: flagConfig,
		token:  &flagToken,
	}

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve crabswarm's verbs as MCP tools over stdio or HTTP",
		Long: `mcp runs crabswarm's MCP server, which offers crabswarm's verbs to an agent
as tools instead of commands it has to remember to type. Over stdio, the
default, it serves one agent as the subprocess that agent's harness starts for
it. With --transport http it serves many agents at once on --listen, for a
harness server that hosts several of them, such as a shared "codex app-server"
or "opencode serve".

Chat is the first family of them. An agent attends the room as soon as its
session opens and holds that attendance for as long as the session runs, so
the room has the member before its first turn rather than from whenever it
first says something — and has it again once a daemon that went away comes
back.

It is configured, not typed: the harness starts it or connects to it and
speaks MCP. Over stdio, stdout carries the protocol and nothing else. Logging
goes to stderr, warnings and above by default and whatever --log asks for
otherwise, which leaves the stream intact either way.

Over stdio the identity token is resolved exactly as in every chat member verb,
so a server configured with no token at all still inherits the one cmdman gave
the agent. One that resolves none still serves, and every tool answers with
what is missing.

Over http the server answers at the /mcp path. Each session names the member it
acts as in its X-Crabswarm-Token request header, and a token attends the room
for as long as it has a session open. A session without the header acts as
nobody, and its tools say so. --token has no meaning there.`,
		Example: `  crabswarm mcp
  crabswarm mcp --sock /run/user/1000/crabswarm/daemon.sock
  crabswarm mcp --transport http --listen 127.0.0.1:7801`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCP(cmd, args, flags, flagTransport, flagListen)
		},
	}

	cmd.Flags().StringVar(&flagToken, "token", "",
		"identity token to act as (default $CRABSWARM_CHAT_TOKEN, else $CMDMAN_CMD_ID); "+
			"stdio only")
	cmd.Flags().StringVar(&flagTransport, "transport", mcpTransportStdio,
		"how harnesses reach the server: stdio (one session) or http (many sessions)")
	cmd.Flags().StringVar(&flagListen, "listen", "",
		"host:port the http transport listens on; required with --transport http")
	// Fails only on a flag name this file does not declare, which the flag
	// above rules out.
	_ = cmd.RegisterFlagCompletionFunc("transport",
		cobra.FixedCompletions([]string{mcpTransportStdio, mcpTransportHTTP},
			cobra.ShellCompDirectiveNoFileComp))

	mcpCodexProxyCmd(cmd)

	parent.AddCommand(cmd)
}

func runMCP(
	cmd *cobra.Command, _ []string, flags *chatFlags, transport, listen string,
) error {
	if err := checkMCPTransport(transport, listen, *flags.token); err != nil {
		return err
	}

	ctx := cmd.Context()

	sock, err := chatSockPath(cmd, flags)
	if err != nil {
		return err
	}

	if transport == mcpTransportHTTP {
		err = serveMCPHTTP(ctx, commandLogger(cmd), sock, listen)
	} else {
		err = serveMCPStdio(ctx, commandLogger(cmd), sock, *flags.token)
	}
	// Serving reports a signalled shutdown as ctx.Err(), which is a clean exit
	// here the way it is for `serve` — those services swallow it and return nil,
	// and a server torn down with its harness has not failed. The guard is
	// against this ctx rather than the bare sentinel, so a context.Canceled
	// raised anywhere else still surfaces as the error it is.
	if err != nil && !errors.Is(err, ctx.Err()) {
		return err
	}
	return nil
}

// checkMCPTransport refuses a flag combination that names no way to serve, or
// one whose other flags the transport would ignore.
func checkMCPTransport(transport, listen, token string) error {
	switch transport {
	case mcpTransportStdio:
		if listen != "" {
			return errors.New("--listen needs --transport http; stdio serves on stdin/stdout")
		}
	case mcpTransportHTTP:
		if listen == "" {
			return errors.New("--transport http needs --listen host:port")
		}
		if token != "" {
			return errors.New("--token needs --transport stdio; an http session names " +
				"its identity in the " + mcp.TokenHeader + " header")
		}
	default:
		return fmt.Errorf("unknown --transport %q: use %s or %s",
			transport, mcpTransportStdio, mcpTransportHTTP)
	}
	return nil
}

func serveMCPStdio(ctx context.Context, logger *slog.Logger, sock, token string) error {
	// The token goes through raw: the server resolves it through the same
	// cli.ResolveToken every member verb uses, where it uses it. Resolving it
	// here would end the process before the harness got its handshake, which is
	// the one outcome nobody can read a reason out of.
	srv, err := mcp.New(logger, sock, token)
	if err != nil {
		return err
	}
	// Every family registers before the server runs, which is what puts its
	// tools in the handshake rather than in a change the harness has to notice.
	mcpchat.Register(srv)
	return srv.Run(ctx)
}

func serveMCPHTTP(ctx context.Context, logger *slog.Logger, sock, listen string) error {
	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("listening for MCP over HTTP on %s: %w", listen, err)
	}
	srv, err := mcp.NewHTTP(logger, sock, nil)
	if err != nil {
		_ = ln.Close()
		return err
	}
	mcpchat.Register(srv)
	return srv.Serve(ctx, ln)
}
