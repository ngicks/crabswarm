package planreview

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

// CallbackConfig holds configuration for running the review callback.
type CallbackConfig struct {
	Command         string
	Args            []string
	PlanWorkingFile string // Abs path to PLAN.md in iteration dir
	PlanSourceFile  string // Abs path to original plan in Claude plans dir
	PlanDir         string // Abs path to iteration plan directory
	PlanName        string
	Iteration       int
	IntermediateDir string // Abs path to _intermediate/
}

// RunCallback runs the review callback command with environment variables set.
// Returns captured stdout, stderr, and any error.
func RunCallback(ctx context.Context, cfg CallbackConfig) (stdout, stderr string, err error) {
	if cfg.Command == "" {
		return "", "", fmt.Errorf("callback command is empty")
	}

	cmd := exec.CommandContext(ctx, cfg.Command, cfg.Args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	cmd.Env = append(cmd.Environ(),
		"PLAN_WORKING_FILE="+cfg.PlanWorkingFile,
		"PLAN_SOURCE_FILE="+cfg.PlanSourceFile,
		"PLAN_DIR="+cfg.PlanDir,
		"PLAN_NAME="+cfg.PlanName,
		"PLAN_ITERATION="+strconv.Itoa(cfg.Iteration),
		"PLAN_INTERMEDIATE_DIR="+cfg.IntermediateDir,
		"PLAN_REVIEW_FILE="+fmt.Sprintf("%03d_REVIEW.md", cfg.Iteration),
	)

	if err := cmd.Run(); err != nil {
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("callback command failed: %w", err)
	}

	return stdoutBuf.String(), stderrBuf.String(), nil
}
