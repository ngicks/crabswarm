package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/cmd/internal/stdiopipe"
	"github.com/ngicks/crabswarm/pkg/crabswarm"
	crabswarmhook "github.com/ngicks/crabswarm/pkg/crabswarm/hook"
)

func hookAutoApproveCmd(parent *cobra.Command) {
	var (
		flagTools []string
		flagUnder []string
	)

	cmd := &cobra.Command{
		Use:   "auto-approve",
		Short: "Auto-approve matching tools under specified directories",
		Long: "A hook for claude code's PermissionRequest event. " +
			"Auto-approves matching tools under specified directories.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHookAutoApprove(cmd, args, flagTools, flagUnder)
		},
	}

	f := cmd.Flags()
	f.StringSliceVar(&flagTools, "tool", nil,
		"Regex to match tool name (repeatable; at least one must match)")
	f.StringSliceVar(&flagUnder, "under", nil,
		"Directory to check file_path containment (repeatable)")

	parent.AddCommand(cmd)
}

func runHookAutoApprove(cmd *cobra.Command, _ []string, toolPatterns, underDirs []string) error {
	ctx := cmd.Context()

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	// --under is always relative — join with CLAUDE_PROJECT_DIR.
	projectDir := crabswarm.ProjectDir()
	if projectDir == "" {
		return fmt.Errorf("%s environment variable is not set", crabswarm.EnvProjectDir)
	}
	for i, d := range underDirs {
		underDirs[i] = filepath.Join(projectDir, d)
	}

	return crabswarmhook.AutoApprove(ctx, reader, crabswarmhook.AutoApproveConfig{
		ToolPatterns: toolPatterns,
		UnderDirs:    underDirs,
	})
}
