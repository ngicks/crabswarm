package internal

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/ngicks/crabswarm/hook/internal/server"
	"github.com/ngicks/crabswarm/hook/internal/tui"
	"github.com/spf13/cobra"
)

var (
	listenAddr  string
	plainMode   bool
	auditEnable bool
	auditOutput string
	auditFormat string
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
	serveCmd.Flags().BoolVar(&auditEnable, "audit-enable", false, "Enable audit logging of hook events")
	serveCmd.Flags().StringVar(&auditOutput, "audit-output", "stderr", "Audit output destination: \"stderr\" or a file path")
	serveCmd.Flags().StringVar(&auditFormat, "audit-format", "text", "Audit output format: \"text\" or \"json\"")
	rootCmd.AddCommand(serveCmd)
}

// runServer starts the interactive permission server.
func runServer(cmd *cobra.Command, args []string) error {
	cfg := server.Config{
		Address: listenAddr,
	}

	// Create base audit handler (file/slog) if enabled.
	var closer io.Closer
	if auditEnable {
		handler, c, err := createAuditHandler()
		if err != nil {
			return fmt.Errorf("failed to create audit handler: %w", err)
		}
		cfg.AuditHandler = handler
		closer = c
	}
	if closer != nil {
		defer closer.Close()
	}

	if plainMode {
		cfg.Reader = cmd.InOrStdin()
		cfg.Writer = cmd.OutOrStdout()
	} else {
		prompter, program := tui.New()
		cfg.Prompter = prompter
		cfg.Program = program
		// Always wrap with TUIAuditHandler in TUI mode so audit events
		// appear in the log panel regardless of --audit-enable.
		cfg.AuditHandler = tui.NewTUIAuditHandler(program, cfg.AuditHandler)
	}

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if plainMode {
		slog.Info("permission server started", "address", srv.Address())
		slog.Info("waiting for permission requests")
		slog.Info("press Ctrl+C to stop")
	}

	if err := srv.Serve(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// createAuditHandler creates a LogAuditHandler based on the audit flags.
// It returns the handler, an optional io.Closer for the underlying file (nil for stderr), and an error.
func createAuditHandler() (*server.LogAuditHandler, io.Closer, error) {
	var writer io.Writer
	var closer io.Closer

	if auditOutput == "stderr" {
		writer = os.Stderr
	} else {
		f, err := os.OpenFile(auditOutput, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open audit file %q: %w", auditOutput, err)
		}
		writer = f
		closer = f
	}

	var handler slog.Handler
	switch auditFormat {
	case "json":
		handler = slog.NewJSONHandler(writer, nil)
	case "text":
		handler = slog.NewTextHandler(writer, nil)
	default:
		if closer != nil {
			closer.Close()
		}
		return nil, nil, fmt.Errorf("invalid audit format %q: must be \"text\" or \"json\"", auditFormat)
	}

	logger := slog.New(handler)
	return server.NewLogAuditHandler(logger), closer, nil
}
