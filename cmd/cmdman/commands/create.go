package commands

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(createCmd)
	addCreateFlags(createCmd)
}

func addCreateFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringP("name", "n", "", "Human-readable unique name")
	f.StringP("dir", "C", "", "Working directory for the command")
	f.StringArrayP("env", "E", nil, "Environment variable KEY=VALUE (repeatable)")
	f.StringArrayP("label", "l", nil, "Metadata label KEY=VALUE (repeatable)")
	f.String("restart", string(cmdman.RestartPolicyNo), "Restart policy: no, on-failure, always")
	f.Bool("rm", false, "Auto-remove on exit")
	f.Int("scrollback-bytes", cmdman.DefaultScrollbackBytes, "Scrollback buffer size in bytes")
}

var createCmd = &cobra.Command{
	Use:   "create [flags] -- COMMAND [ARGS...]",
	Short: "Create a new command without starting it",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	id, name, err := doCreate(cmd, args)
	if err != nil {
		return err
	}
	displayName := id
	if name != "" {
		displayName = name
	}
	fmt.Fprintln(cmd.OutOrStdout(), displayName)
	return nil
}

// doCreate creates a command entry in the store and returns the generated ID and name.
func doCreate(cmd *cobra.Command, args []string) (id, name string, err error) {
	f := cmd.Flags()
	name, _ = f.GetString("name")
	dir, _ := f.GetString("dir")
	envSlice, _ := f.GetStringArray("env")
	labelSlice, _ := f.GetStringArray("label")
	restartPolicy, _ := f.GetString("restart")
	autoRemove, _ := f.GetBool("rm")
	scrollbackBytes, _ := f.GetInt("scrollback-bytes")

	if !cmdman.IsRestartPolicy(restartPolicy) {
		return "", "", fmt.Errorf("invalid restart policy: %s", restartPolicy)
	}
	rp := cmdman.RestartPolicy(restartPolicy)

	labels, err := parseLabels(labelSlice)
	if err != nil {
		return "", "", err
	}

	annotations := make(map[string]string)
	if autoRemove {
		annotations[cmdman.AnnotationAutoRemove] = "true"
	}

	id = generateID()
	commandDir := cmdman.CommandDir(id)

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

	if cfg.Dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("get working directory: %w", err)
		}
		cfg.Dir = wd
	}

	dbPath := cmdman.DBPath()
	store, err := cmdman.OpenStore(dbPath, true)
	if err != nil {
		return "", "", fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	if err := store.InsertCommandConfig(id, name, cfg); err != nil {
		return "", "", fmt.Errorf("insert config: %w", err)
	}

	if err := cfg.Write(); err != nil {
		return "", "", fmt.Errorf("materialize config: %w", err)
	}

	stateJSON := &cmdman.CommandStateJSON{}
	if err := store.InsertCommandState(id, cmdman.StateCreated, stateJSON); err != nil {
		return "", "", fmt.Errorf("insert state: %w", err)
	}

	return id, name, nil
}

func generateID() string {
	var buf [16]byte
	rand.Read(buf[:])
	return hex.EncodeToString(buf[:])
}
