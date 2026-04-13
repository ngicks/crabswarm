package commands

import (
	"fmt"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	addCreateFlags(runCmd)
	runCmd.Flags().Bool("attach", false, "Attach after the command reaches running")
}

var runCmd = &cobra.Command{
	Use:   "run [flags] -- COMMAND [ARGS...]",
	Short: "Create and start a new command",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	attach, _ := cmd.Flags().GetBool("attach")

	id, name, err := doCreate(cmd, args)
	if err != nil {
		return err
	}

	if err := doStart(cmd, id); err != nil {
		return err
	}

	displayName := id
	if name != "" {
		displayName = name
	}
	fmt.Fprintln(cmd.OutOrStdout(), displayName)

	if attach {
		dbPath := cmdman.DBPath()
		store, err := cmdman.OpenStore(dbPath, true)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		state, _, _, err := store.GetCommandState(id)
		store.Close()
		if err != nil {
			return fmt.Errorf("get state: %w", err)
		}
		if state == cmdman.StateRunning {
			return runAttach(cmd, id)
		}
	}

	return nil
}
