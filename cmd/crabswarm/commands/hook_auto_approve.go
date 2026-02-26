package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ngicks/crabswarm/cmd/internal/stdiopipe"
	"github.com/ngicks/crabswarm/pkg/crabswarm"
	"github.com/spf13/cobra"
)

func init() {
	hookCmd.AddCommand(hookAutoApproveCmd)

	hookAutoApproveCmd.Flags().StringSlice("tool", nil, "Regex pattern to match tool name (repeatable, at least one must match)")
	hookAutoApproveCmd.Flags().StringSlice("under", nil, "Directory to check file_path containment (repeatable, file must be under at least one)")
}

var hookAutoApproveCmd = &cobra.Command{
	Use:   "auto-approve",
	Short: "A hook for claude code's PermissionRequest event. Auto-approves matching tools under specified directories.",
	RunE:  runHookAutoApproveCmd,
}

func runHookAutoApproveCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	toolPatterns, err := cmd.Flags().GetStringSlice("tool")
	if err != nil {
		return err
	}
	underDirs, err := cmd.Flags().GetStringSlice("under")
	if err != nil {
		return err
	}

	// --under is always relative — join with CLAUDE_PROJECT_DIR.
	projectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	if projectDir == "" {
		panic(fmt.Errorf("CLAUDE_PROJECT_DIR environment variable is not set"))
	}
	for i, d := range underDirs {
		underDirs[i] = filepath.Join(projectDir, d)
	}

	cfg := crabswarm.AutoApproveConfig{
		ToolPatterns: toolPatterns,
		UnderDirs:    underDirs,
	}

	return crabswarm.HookAutoApprove(ctx, reader, cfg)
}
