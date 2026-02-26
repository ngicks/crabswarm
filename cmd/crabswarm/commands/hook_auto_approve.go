package commands

import (
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

	toolPatterns, _ := cmd.Flags().GetStringSlice("tool")
	underDirs, _ := cmd.Flags().GetStringSlice("under")

	cfg := crabswarm.AutoApproveConfig{
		ToolPatterns: toolPatterns,
		UnderDirs:    underDirs,
	}

	return crabswarm.HookAutoApprove(ctx, reader, cfg)
}
