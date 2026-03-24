package commands

import (
	"fmt"
	"os"
	"syscall"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(rmCmd)
	rmCmd.Flags().StringArrayP("label", "l", nil, "Target commands matching labels")
	rmCmd.Flags().BoolP("force", "f", false, "Force remove running commands (sends SIGKILL)")
}

var rmCmd = &cobra.Command{
	Use:   "rm [flags] [ID|NAME...]",
	Short: "Remove a stopped command",
	RunE:  runRm,
}

func runRm(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	labelSlice, _ := cmd.Flags().GetStringArray("label")

	store, err := cmdman.OpenStore(cmdman.DBPath(), true)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	ids, err := resolveTargets(store, args, labelSlice)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := rmOne(cmd, store, id, force); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "rm %s: %v\n", id, err)
		}
	}
	return nil
}

func rmOne(cmd *cobra.Command, store *cmdman.Store, id string, force bool) error {
	state, _, stateJSON, err := store.GetCommandState(id)
	if err != nil {
		return err
	}

	if state == cmdman.StateRunning || state == cmdman.StateStarting {
		if !force {
			return fmt.Errorf("command is %s, use --force to remove", state)
		}
		// Force kill the monitor.
		if stateJSON.MonitorPID > 0 {
			proc, err := os.FindProcess(stateJSON.MonitorPID)
			if err == nil {
				proc.Signal(syscall.SIGKILL)
			}
		}
	}

	// Get config for command-dir.
	_, _, cfg, err := store.GetCommandConfig(id)
	if err != nil {
		return err
	}

	// Delete DB rows.
	if err := store.DeleteCommand(id); err != nil {
		return fmt.Errorf("delete from db: %w", err)
	}

	// Remove command directory.
	if cfg.CommandDir != "" {
		os.RemoveAll(cfg.CommandDir)
	}

	// Remove runtime directory.
	runtimeDir := cmdman.MonitorRuntimeDir(id)
	os.RemoveAll(runtimeDir)

	fmt.Fprintln(cmd.OutOrStdout(), id)
	return nil
}
