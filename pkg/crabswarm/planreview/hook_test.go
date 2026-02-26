package planreview

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"gotest.tools/v3/assert"
)

func makePreToolUseInput(toolName, sessionID, transcriptPath string) string {
	input := map[string]any{
		"session_id":      sessionID,
		"transcript_path": transcriptPath,
		"cwd":             "/test",
		"hook_event_name": "PreToolUse",
		"tool_name":       toolName,
		"tool_input":      map[string]any{},
		"tool_use_id":     "test-123",
	}
	data, _ := json.Marshal(input)
	return string(data)
}

func TestHookPlanCallback_NonExitPlanMode(t *testing.T) {
	input := makePreToolUseInput("Read", "sess-1", "/tmp/transcript.jsonl")
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{})

	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he), "expected HandlerError, got %T: %v", err, err)
	assert.Assert(t, he.Output == nil, "expected nil output for pass-through")
}

func TestHookPlanCallback_EmptySessionID(t *testing.T) {
	input := makePreToolUseInput("ExitPlanMode", "", "/tmp/transcript.jsonl")
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{})

	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))
	assert.Assert(t, he.Output == nil)
}

func TestHookPlanCallback_NoPlanInTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	// Transcript with no Write events to plans dir.
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(`{"tool_name":"Read","tool_input":{"file_path":"/foo.go"}}`+"\n"), 0o644))

	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{
		PlansDir: plansDir,
	})

	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))
	assert.Assert(t, he.Output == nil)
}

func TestHookPlanCallback_CreatesIterationDir(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	outputDir := filepath.Join(tmpDir, "output")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	// Create plan file.
	planFile := filepath.Join(plansDir, "test-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Test Plan\nDetails here."), 0o644))

	// Create transcript pointing to plan file.
	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Test Plan\nDetails here."}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{
		PlansDir:  plansDir,
		OutputDir: outputDir,
	})

	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))

	// Verify iteration directory was created.
	entries, err := os.ReadDir(outputDir)
	assert.NilError(t, err)
	assert.Assert(t, len(entries) == 1, "expected 1 plan dir, got %d", len(entries))

	planDirPath := filepath.Join(outputDir, entries[0].Name())

	// Verify PLAN.md exists.
	planContent, err := os.ReadFile(filepath.Join(planDirPath, "PLAN.md"))
	assert.NilError(t, err)
	assert.Equal(t, string(planContent), "# Test Plan\nDetails here.")

	// Verify intermediate snapshot exists.
	intermediateDir := filepath.Join(planDirPath, "_intermediate")
	snapshot, err := os.ReadFile(filepath.Join(intermediateDir, "001_00_PLAN.md"))
	assert.NilError(t, err)
	assert.Equal(t, string(snapshot), "# Test Plan\nDetails here.")
}

func TestHookPlanCallback_WithCallback(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	outputDir := filepath.Join(tmpDir, "output")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	planFile := filepath.Join(plansDir, "reviewed-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Plan"), 0o644))

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Plan"}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{
		PlansDir:        plansDir,
		OutputDir:       outputDir,
		CallbackCmd:     "echo",
		CallbackArgs:    []string{"LGTM"},
		CallbackTimeout: 10 * time.Second,
	})

	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))

	// Verify review file was saved.
	entries, err := os.ReadDir(outputDir)
	assert.NilError(t, err)
	assert.Assert(t, len(entries) == 1)

	planDirPath := filepath.Join(outputDir, entries[0].Name())
	intermediateDir := filepath.Join(planDirPath, "_intermediate")

	review, err := os.ReadFile(filepath.Join(intermediateDir, "001_01_REVIEW.md"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(review), "LGTM"))
}

func TestHookPlanCallback_CallbackFailure(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	outputDir := filepath.Join(tmpDir, "output")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	planFile := filepath.Join(plansDir, "fail-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Plan"), 0o644))

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Plan"}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{
		PlansDir:        plansDir,
		OutputDir:       outputDir,
		CallbackCmd:     "sh",
		CallbackArgs:    []string{"-c", "echo error-output >&2; exit 1"},
		CallbackTimeout: 10 * time.Second,
	})

	// Should return regular error (not HandlerError) on callback failure.
	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he), "expected regular error, got HandlerError")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "callback failed"))

	// Review file should still be written (stderr captured).
	entries, _ := os.ReadDir(outputDir)
	assert.Assert(t, len(entries) == 1)
	planDirPath := filepath.Join(outputDir, entries[0].Name())
	review, err := os.ReadFile(filepath.Join(planDirPath, "_intermediate", "001_01_REVIEW.md"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(review), "error-output"))
}

func TestHookPlanCallback_EmptyPlanFile(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	planFile := filepath.Join(plansDir, "empty.md")
	assert.NilError(t, os.WriteFile(planFile, []byte(""), 0o644))

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":""}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{
		PlansDir: plansDir,
	})

	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he), "expected regular error, got HandlerError")
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "empty"))
}

func TestHookPlanCallback_InvalidJSON(t *testing.T) {
	err := HookPlanCallback(context.Background(), strings.NewReader("not json"), HookCallbackConfig{})

	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he))
	assert.Assert(t, err != nil)
}
