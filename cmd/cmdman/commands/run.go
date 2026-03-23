package commands

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
	f := runCmd.Flags()
	f.StringP("name", "n", "", "Human-readable unique name")
	f.StringP("dir", "C", "", "Working directory for the command")
	f.StringArrayP("env", "E", nil, "Environment variable KEY=VALUE (repeatable)")
	f.StringArrayP("label", "l", nil, "Metadata label KEY=VALUE (repeatable)")
	f.String("restart", string(cmdman.RestartPolicyNo), "Restart policy: no, on-failure, always")
	f.Bool("rm", false, "Auto-remove on exit")
	f.Int("scrollback-bytes", cmdman.DefaultScrollbackBytes, "Scrollback buffer size in bytes")
	f.Bool("attach", false, "Attach after the command reaches running")
}

var runCmd = &cobra.Command{
	Use:   "run [flags] -- COMMAND [ARGS...]",
	Short: "Start a new command",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()
	name, _ := f.GetString("name")
	dir, _ := f.GetString("dir")
	envSlice, _ := f.GetStringArray("env")
	labelSlice, _ := f.GetStringArray("label")
	restartPolicy, _ := f.GetString("restart")
	autoRemove, _ := f.GetBool("rm")
	scrollbackBytes, _ := f.GetInt("scrollback-bytes")
	attach, _ := f.GetBool("attach")

	// Validate restart policy.
	if !cmdman.IsRestartPolicy(restartPolicy) {
		return fmt.Errorf("invalid restart policy: %s", restartPolicy)
	}
	rp := cmdman.RestartPolicy(restartPolicy)

	// Parse labels.
	labels, err := parseLabels(labelSlice)
	if err != nil {
		return err
	}

	// Build annotations.
	annotations := make(map[string]string)
	if autoRemove {
		annotations[cmdman.AnnotationAutoRemove] = "true"
	}

	// Generate command ID.
	id := uuid.New().String()
	commandDir := cmdman.CommandDir(id)

	// Build config.
	cfg := &cmdman.CommandConfigJSON{
		Argv:            args,
		Dir:             dir,
		Env:             envSlice,
		RestartPolicy:   rp,
		ScrollbackBytes: scrollbackBytes,
		Labels:          labels,
		Annotations:     annotations,
		CommandDir:      commandDir,
	}

	// Resolve working directory.
	if cfg.Dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("get working directory: %w", err)
		}
		cfg.Dir = wd
	}

	// Open store.
	dbPath := cmdman.DBPath()
	store, err := cmdman.OpenStore(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	// Insert config.
	if err := store.InsertCommandConfig(id, name, cfg); err != nil {
		return fmt.Errorf("insert config: %w", err)
	}

	// Materialize config.json.
	if err := cfg.Write(); err != nil {
		return fmt.Errorf("materialize config: %w", err)
	}

	// Insert initial state.
	stateJSON := &cmdman.CommandStateJSON{}
	if err := store.InsertCommandState(id, cmdman.StateCreated, stateJSON); err != nil {
		return fmt.Errorf("insert state: %w", err)
	}

	// Spawn monitor.
	_, err = cmdman.SpawnMonitor(id, commandDir, dbPath)
	if err != nil {
		// Clean up on failure.
		store.DeleteCommand(id)
		os.RemoveAll(commandDir)
		return fmt.Errorf("spawn monitor: %w", err)
	}

	// Wait for running state.
	finalState, err := cmdman.WaitForState(store, id, cmdman.StateRunning, 100)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v (state: %s)\n", err, finalState)
	}

	displayName := id
	if name != "" {
		displayName = name
	}
	fmt.Fprintln(cmd.OutOrStdout(), displayName)

	// Attach if requested.
	if attach && finalState == cmdman.StateRunning {
		return runAttach(cmd, id)
	}

	return nil
}
