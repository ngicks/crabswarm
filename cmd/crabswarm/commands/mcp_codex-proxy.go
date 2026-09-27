package commands

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/mcp/codexproxy"
)

func mcpCodexProxyCmd(parent *cobra.Command) {
	var (
		flagToken      string
		flagServerName string
	)

	cmd := &cobra.Command{
		Use:   "codex-proxy [flags] -- codex --remote unix://SOCKET [ARGS...]",
		Short: "Run a remote Codex TUI that names its chat member on every thread it starts",
		Long: `codex-proxy runs a Codex TUI attached to a shared app server and relays its
connection, so each thread the TUI starts carries this TUI's chat identity.

Several TUIs can attach to one "codex app-server", and every thread's MCP
servers run under the app server, so none of them can tell which TUI a thread
belongs to. The proxy serves a private unix socket, hands it to the TUI in place
of the --remote address, and forwards every message between the two unchanged,
with two exceptions. A thread/start, thread/resume or thread/fork request gets
the identity token as the per-thread config key
mcp_servers.<server-name>.http_headers.X-Crabswarm-Token, which is the header
the thread's "crabswarm mcp --transport http" session names its member with.
After a thread/resume the proxy also calls the server's crabswarm_focus tool on
the resumed thread, since resuming a thread that is still loaded opens no new
session, and it keeps the answer to that call from the TUI.

The app server has to declare --server-name as a streamable-HTTP server (a
"url" entry under mcp_servers). Codex refuses a thread whose config sets
http_headers on a server it does not declare, or on a stdio one, so every thread
the TUI starts fails otherwise.

The identity token is resolved exactly as in every chat member verb. With none,
the proxy forwards everything unchanged and warns once, and Codex still runs.

The command must follow "--" and name its app server as --remote unix://PATH or
--remote=unix://PATH. The proxy's socket sits under
$XDG_RUNTIME_DIR/crabswarm-codex-proxy/, or under the temporary directory where
that is unset, and is removed when the command exits. The command owns the
terminal: SIGINT and SIGTERM are passed on to it, and the proxy exits with its
status. Logging goes to stderr beneath the TUI, warnings and above by default.`,
		Example: `  crabswarm mcp codex-proxy -- codex --remote unix:///tmp/codex.sock
  crabswarm mcp codex-proxy --token "$TOKEN" -- codex --remote=unix:///tmp/codex.sock -m gpt-6-sol`,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: mcpCodexProxyCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMCPCodexProxy(cmd, args, flagToken, flagServerName)
		},
	}

	cmd.Flags().StringVar(&flagToken, "token", "",
		"identity token stamped on every thread (default $CRABSWARM_CHAT_TOKEN, "+
			"else $CMDMAN_CMD_ID)")
	cmd.Flags().StringVar(&flagServerName, "server-name", codexproxy.DefaultServer,
		"name of the crabswarm MCP server in the app server's mcp_servers config")

	parent.AddCommand(cmd)
}

// mcpCodexProxyCompletion leaves every positional to the shell: the first one
// is the TUI program and the rest are its own arguments.
func mcpCodexProxyCompletion(
	_ *cobra.Command,
	_ []string,
	_ string,
) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func runMCPCodexProxy(cmd *cobra.Command, args []string, token, serverName string) error {
	// A dash anywhere else means the command was written without "--" or mixed
	// with this command's flags, and pflag may have taken a Codex flag as ours.
	if cmd.ArgsLenAtDash() != 0 {
		return errors.New(`the codex command must follow "--", ` +
			`e.g. crabswarm mcp codex-proxy -- codex --remote unix:///tmp/codex.sock`)
	}
	return codexproxy.Run(cmd.Context(), commandLogger(cmd), codexproxy.Config{
		Token:  token,
		Argv:   args,
		Server: serverName,
	})
}
