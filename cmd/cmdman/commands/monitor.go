package commands

import (
	"log/slog"
	"os"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(monitorCmd)
	monitorCmd.Flags().String("id", "", "Command ID")
	monitorCmd.Flags().String("command-dir", "", "Command directory path")
	monitorCmd.Flags().String("db", "", "Database path")
	monitorCmd.MarkFlagRequired("id")
	monitorCmd.MarkFlagRequired("command-dir")
	monitorCmd.MarkFlagRequired("db")
}

var monitorCmd = &cobra.Command{
	Use:    "__monitor",
	Short:  "Internal monitor process (do not call directly)",
	Hidden: true,
	RunE:   runMonitor,
}

func runMonitor(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	commandDir, _ := cmd.Flags().GetString("command-dir")
	dbPath, _ := cmd.Flags().GetString("db")

	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	}))

	return cmdman.RunMonitor(cmd.Context(), id, commandDir, dbPath, logger)
}
