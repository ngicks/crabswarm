package commands

import (
	"fmt"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(migrateCmd)
}

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Run database schema migrations",
	RunE:  runMigrate,
}

func runMigrate(cmd *cobra.Command, args []string) error {
	store, err := cmdman.OpenStore(cmdman.DBPath(), false)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "migrations complete")
	return nil
}
