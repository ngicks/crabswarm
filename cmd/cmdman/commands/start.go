package commands

import (
	"fmt"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(startCmd)
}

var startCmd = &cobra.Command{
	Use:   "start ID_OR_NAME",
	Short: "Start a created command",
	Args:  cobra.ExactArgs(1),
	RunE:  runStart,
}

func runStart(cmd *cobra.Command, args []string) error {
	return doStart(cmd, args[0])
}

// doStart spawns the monitor for an existing command in "created" state.
func doStart(cmd *cobra.Command, idOrName string) error {
	dbPath := cmdman.DBPath()
	store, err := cmdman.OpenStore(dbPath, true)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	id, _, cfg, err := store.GetCommandConfig(idOrName)
	if err != nil {
		return fmt.Errorf("get command config: %w", err)
	}

	state, _, _, err := store.GetCommandState(id)
	if err != nil {
		return fmt.Errorf("get command state: %w", err)
	}
	switch state {
	case cmdman.StateCreated, cmdman.StateExited:
		// OK to start.
	default:
		return fmt.Errorf("command %s is in state %q, must be %q or %q", idOrName, state, cmdman.StateCreated, cmdman.StateExited)
	}

	_, err = cmdman.SpawnMonitor(id, cfg.CommandDir, dbPath)
	if err != nil {
		return fmt.Errorf("spawn monitor: %w", err)
	}

	finalState, err := cmdman.WaitForState(store, id, cmdman.StateRunning, 100)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v (state: %s)\n", err, finalState)
	}

	return nil
}
