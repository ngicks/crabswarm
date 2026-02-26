package crabswarm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/ngicks/crabswarm/pkg/claudesdk/models"
	"github.com/ngicks/crabswarm/pkg/crabswarm/planreview"
)

// HookCallbackConfig holds configuration for the plan-callback hook.
type HookCallbackConfig struct {
	PlansDir        string        // Where Claude writes plan files (e.g. ~/.claude/plans/)
	OutputDir       string        // Where to create iteration directories (abs path)
	CallbackCmd     string        // Command to run for review
	CallbackArgs    []string      // Arguments for the callback command
	CallbackTimeout time.Duration // Timeout for callback execution
}

// HookPlanCallback implements the plan-callback hook logic.
// It reads a PreToolUseHookInput from r, checks for ExitPlanMode,
// resolves the last plan file from the transcript, creates iteration snapshots,
// and runs the review callback.
func HookPlanCallback(ctx context.Context, r io.Reader, cfg HookCallbackConfig) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input models.PreToolUseHookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("parsing PreToolUseHookInput: %w", err)
	}

	// Only process ExitPlanMode events.
	if input.ToolName != "ExitPlanMode" {
		return &handler.HandlerError{}
	}

	// If session_id is empty, pass through.
	if input.SessionID == "" {
		return &handler.HandlerError{}
	}

	// Resolve the last plan file from the transcript.
	planPath, err := planreview.ResolveLastPlanPath(input.TranscriptPath, cfg.PlansDir)
	if err != nil {
		return fmt.Errorf("resolving last plan path: %w", err)
	}
	if planPath == "" {
		// No plan file found in transcript — nothing to review.
		return &handler.HandlerError{}
	}

	// Read plan file content.
	planContent, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("reading plan file %s: %w", planPath, err)
	}
	if len(planContent) == 0 {
		return fmt.Errorf("plan file is empty: %s", planPath)
	}

	// Derive plan name and set up iteration directory.
	planName, err := planreview.DerivePlanName(planPath)
	if err != nil {
		return fmt.Errorf("deriving plan name: %w", err)
	}
	planDirName := planreview.PlanDirName(time.Now(), planName)
	planDir := filepath.Join(cfg.OutputDir, planDirName)
	intermediateDir := filepath.Join(planDir, "_intermediate")

	if err := os.MkdirAll(intermediateDir, fs.ModePerm); err != nil {
		return fmt.Errorf("creating intermediate dir: %w", err)
	}

	// Count existing iterations.
	iteration, err := planreview.CountIterations(intermediateDir)
	if err != nil {
		return fmt.Errorf("counting iterations: %w", err)
	}
	iteration++ // Next iteration number.

	// Write plan snapshot to intermediate directory.
	planSnapshotPath := filepath.Join(intermediateDir, planreview.IntermediateFileName(iteration, "PLAN"))
	if err := os.WriteFile(planSnapshotPath, planContent, 0o644); err != nil {
		return fmt.Errorf("writing plan snapshot: %w", err)
	}

	// Write/overwrite PLAN.md in plan directory (latest version).
	planWorkingFile := filepath.Join(planDir, "PLAN.md")
	if err := os.WriteFile(planWorkingFile, planContent, 0o644); err != nil {
		return fmt.Errorf("writing working plan file: %w", err)
	}

	// Run callback if configured.
	if cfg.CallbackCmd != "" {
		callbackCtx := ctx
		if cfg.CallbackTimeout > 0 {
			var cancel context.CancelFunc
			callbackCtx, cancel = context.WithTimeout(ctx, cfg.CallbackTimeout)
			defer cancel()
		}

		absPlanDir, _ := filepath.Abs(planDir)
		absIntermediateDir, _ := filepath.Abs(intermediateDir)
		absWorkingFile, _ := filepath.Abs(planWorkingFile)
		absSourceFile, _ := filepath.Abs(planPath)

		stdout, stderr, err := planreview.RunCallback(callbackCtx, planreview.CallbackConfig{
			Command:         cfg.CallbackCmd,
			Args:            cfg.CallbackArgs,
			PlanWorkingFile: absWorkingFile,
			PlanSourceFile:  absSourceFile,
			PlanDir:         absPlanDir,
			PlanName:        planName,
			Iteration:       iteration,
			IntermediateDir: absIntermediateDir,
		})

		// Save review output regardless of error.
		reviewPath := filepath.Join(intermediateDir, planreview.IntermediateFileName(iteration, "REVIEW"))
		reviewContent := stdout
		if reviewContent == "" && stderr != "" {
			reviewContent = stderr
		}
		if writeErr := os.WriteFile(reviewPath, []byte(reviewContent), 0o644); writeErr != nil {
			log.Printf("warning: failed to write review file: %v", writeErr)
		}

		if err != nil {
			log.Printf("callback error: %v", err)
			if stderr != "" {
				log.Printf("callback stderr: %s", stderr)
			}
			return fmt.Errorf("callback failed: %w", err)
		}
	}

	// Exit 0 — user decides whether to iterate.
	return &handler.HandlerError{}
}
