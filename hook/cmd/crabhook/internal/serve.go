package internal

import (
	"fmt"

	"github.com/ngicks/crabswarm/hook/internal/server"
	"github.com/ngicks/crabswarm/hook/internal/tui"
	"github.com/spf13/cobra"
)

var (
	listenAddr string
	plainMode  bool
)

// serveCmd is the serve subcommand for running the interactive permission server.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the interactive permission server",
	Long: `Start the interactive permission server that handles permission requests
from crabhook clients.

The server displays permission requests in a TUI and prompts for user input.
Run this in a separate terminal before using Claude Code with hooks configured.

Use --plain for a non-interactive plain text mode (no TUI).`,
	RunE: runServer,
}

func init() {
	serveCmd.Flags().StringVarP(&listenAddr, "listen", "l", "localhost:50051", "Address to listen on")
	serveCmd.Flags().BoolVar(&plainMode, "plain", false, "Use plain text prompts instead of TUI")
	rootCmd.AddCommand(serveCmd)
}

// runServer starts the interactive permission server.
func runServer(cmd *cobra.Command, args []string) error {
	cfg := server.Config{
		Address: listenAddr,
	}

	if plainMode {
		cfg.Reader = cmd.InOrStdin()
		cfg.Writer = cmd.OutOrStdout()
	} else {
		prompter, program := tui.New()
		cfg.Prompter = prompter
		cfg.Program = program
	}

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if plainMode {
		fmt.Fprintf(cmd.OutOrStdout(), "Permission server listening on %s\n", srv.Address())
		fmt.Fprintln(cmd.OutOrStdout(), "Waiting for permission requests...")
		fmt.Fprintln(cmd.OutOrStdout(), "Press Ctrl+C to stop.")
	}

	if err := srv.Serve(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
