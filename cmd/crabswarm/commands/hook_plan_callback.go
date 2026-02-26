package commands

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ngicks/crabswarm/cmd/internal/stdiopipe"
	"github.com/ngicks/crabswarm/pkg/crabswarm"
	"github.com/spf13/cobra"
)

func init() {
	hookCmd.AddCommand(hookPlanCallbackCmd)

	hookPlanCallbackCmd.Flags().String("plans-dir", "", "Directory where Claude writes plan files (default: $CLAUDE_CONFIG_DIR/plans/ or $HOME/.claude/plans/)")
	hookPlanCallbackCmd.Flags().String("output-dir", "", "Relative path for plan output directory, joined with $CLAUDE_PROJECT_DIR at runtime (env: CRABSWARM_PLAN_OUTPUT_DIR, default: doc/plans). Must be a relative path.")
	hookPlanCallbackCmd.Flags().String("callback-cmd", "", "Command to run for plan review (env: CRABSWARM_PLAN_CALLBACK_CMD)")
	hookPlanCallbackCmd.Flags().StringSlice("callback-arg", nil, "Arguments for the callback command (repeatable)")
	hookPlanCallbackCmd.Flags().Duration("callback-timeout", 5*time.Minute, "Timeout for callback execution")
}

var hookPlanCallbackCmd = &cobra.Command{
	Use:   "plan-callback",
	Short: "A hook for claude code's PreToolUse ExitPlanMode event. Archives plan iterations and runs review callback.",
	RunE:  runHookPlanCallbackCmd,
}

func runHookPlanCallbackCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	plansDir, _ := cmd.Flags().GetString("plans-dir")
	if plansDir == "" {
		plansDir = resolvePlansDir()
	}

	outputDir, _ := cmd.Flags().GetString("output-dir")
	if outputDir == "" {
		outputDir = os.Getenv("CRABSWARM_PLAN_OUTPUT_DIR")
	}
	if outputDir == "" {
		outputDir = "doc/plans"
	}

	// output-dir is relative — join with CLAUDE_PROJECT_DIR.
	projectDir := os.Getenv("CLAUDE_PROJECT_DIR")
	if projectDir != "" {
		outputDir = filepath.Join(projectDir, outputDir)
	}

	callbackCmd, _ := cmd.Flags().GetString("callback-cmd")
	if callbackCmd == "" {
		callbackCmd = os.Getenv("CRABSWARM_PLAN_CALLBACK_CMD")
	}

	callbackArgs, _ := cmd.Flags().GetStringSlice("callback-arg")
	callbackTimeout, _ := cmd.Flags().GetDuration("callback-timeout")

	cfg := crabswarm.HookCallbackConfig{
		PlansDir:        plansDir,
		OutputDir:       outputDir,
		CallbackCmd:     callbackCmd,
		CallbackArgs:    callbackArgs,
		CallbackTimeout: callbackTimeout,
	}

	return crabswarm.HookPlanCallback(ctx, reader, cfg)
}

// resolvePlansDir returns the Claude plans directory.
// Priority: $CLAUDE_CONFIG_DIR/plans/ > $HOME/.claude/plans/
func resolvePlansDir() string {
	if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
		return filepath.Join(configDir, "plans")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "plans")
}
