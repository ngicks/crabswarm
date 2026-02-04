package internal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ngicks/crabswarm/hook/internal/server"
	"github.com/spf13/cobra"
)

var (
	listenAddr string
)

// serveCmd is the serve subcommand for running the interactive permission server.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the interactive permission server",
	Long: `Start the interactive permission server that handles permission requests
from crabhook clients.

The server displays permission requests in the terminal and prompts for user input.
Run this in a separate terminal before using Claude Code with hooks configured.`,
	RunE: runServer,
}

func init() {
	serveCmd.Flags().StringVarP(&listenAddr, "listen", "l", "localhost:50051", "Address to listen on")
	rootCmd.AddCommand(serveCmd)
}

// runServer starts the interactive permission server.
func runServer(cmd *cobra.Command, args []string) error {
	cfg := server.Config{
		Address: listenAddr,
		Reader:  os.Stdin,
		Writer:  os.Stdout,
	}

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Fprintln(os.Stderr, "\nShutting down server...")
		srv.Stop()
	}()

	fmt.Printf("Permission server listening on %s\n", srv.Address())
	fmt.Println("Waiting for permission requests...")
	fmt.Println("Press Ctrl+C to stop.")

	if err := srv.Serve(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
