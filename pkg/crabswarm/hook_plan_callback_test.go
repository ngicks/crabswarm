package crabswarm

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
	snapshot, err := os.ReadFile(filepath.Join(intermediateDir, "001_PLAN.md"))
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

	review, err := os.ReadFile(filepath.Join(intermediateDir, "001_REVIEW.md"))
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

	// Review file should NOT be written on failure (enables retry).
	entries, _ := os.ReadDir(outputDir)
	assert.Assert(t, len(entries) == 1)
	planDirPath := filepath.Join(outputDir, entries[0].Name())
	_, err = os.Stat(filepath.Join(planDirPath, "_intermediate", "001_REVIEW.md"))
	assert.Assert(t, os.IsNotExist(err), "review should not be written on callback failure")
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

func TestHookPlanCallback_ReusesExistingDir(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	outputDir := filepath.Join(tmpDir, "output")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	planFile := filepath.Join(plansDir, "reuse-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Reuse Plan\nVersion 1"), 0o644))

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Reuse Plan\nVersion 1"}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	// First call.
	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{
		PlansDir:  plansDir,
		OutputDir: outputDir,
	})
	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))

	// Verify 1 dir exists.
	entries, err := os.ReadDir(outputDir)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 1)
	planDirPath := filepath.Join(outputDir, entries[0].Name())

	// Update plan content and call again.
	assert.NilError(t, os.WriteFile(planFile, []byte("# Reuse Plan\nVersion 2"), 0o644))
	transcript2 := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Reuse Plan\nVersion 2"}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript2), 0o644))

	input2 := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err = HookPlanCallback(context.Background(), strings.NewReader(input2), HookCallbackConfig{
		PlansDir:  plansDir,
		OutputDir: outputDir,
	})
	assert.Assert(t, errors.As(err, &he))

	// Still only 1 directory.
	entries, err = os.ReadDir(outputDir)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 1)

	// Verify both snapshots exist.
	intermediateDir := filepath.Join(planDirPath, "_intermediate")
	_, err = os.Stat(filepath.Join(intermediateDir, "001_PLAN.md"))
	assert.NilError(t, err)
	_, err = os.Stat(filepath.Join(intermediateDir, "002_PLAN.md"))
	assert.NilError(t, err)

	// Verify state.json exists.
	stateData, err := os.ReadFile(filepath.Join(planDirPath, "state.json"))
	assert.NilError(t, err)
	assert.Assert(t, strings.Contains(string(stateData), "source_plan_path"))
}

func TestHookPlanCallback_SkipsDuplicateContent(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	outputDir := filepath.Join(tmpDir, "output")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	// Create a marker file to track callback invocations.
	markerFile := filepath.Join(tmpDir, "callback-count")

	planFile := filepath.Join(plansDir, "dedup-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Dedup Plan\nSame content"), 0o644))

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Dedup Plan\nSame content"}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	cfg := HookCallbackConfig{
		PlansDir:        plansDir,
		OutputDir:       outputDir,
		CallbackCmd:     "sh",
		CallbackArgs:    []string{"-c", "echo LGTM; echo 1 >> " + markerFile},
		CallbackTimeout: 10 * time.Second,
	}

	// First call — should run callback.
	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), cfg)
	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))

	// Second call with identical content — should skip.
	input2 := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err = HookPlanCallback(context.Background(), strings.NewReader(input2), cfg)
	assert.Assert(t, errors.As(err, &he))

	// Only 1 plan directory.
	entries, err := os.ReadDir(outputDir)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 1)

	planDirPath := filepath.Join(outputDir, entries[0].Name())
	intermediateDir := filepath.Join(planDirPath, "_intermediate")

	// Only 001_PLAN.md — no 002_PLAN.md.
	_, err = os.Stat(filepath.Join(intermediateDir, "001_PLAN.md"))
	assert.NilError(t, err)
	_, err = os.Stat(filepath.Join(intermediateDir, "002_PLAN.md"))
	assert.Assert(t, os.IsNotExist(err), "expected no 002_PLAN.md, but it exists")

	// Callback only ran once.
	markerContent, err := os.ReadFile(markerFile)
	assert.NilError(t, err)
	lines := strings.Split(strings.TrimSpace(string(markerContent)), "\n")
	assert.Equal(t, len(lines), 1, "callback should have run exactly once")
}

func TestHookPlanCallback_RetriesOnCallbackFailure(t *testing.T) {
	tmpDir := t.TempDir()
	plansDir := filepath.Join(tmpDir, "plans")
	outputDir := filepath.Join(tmpDir, "output")
	assert.NilError(t, os.MkdirAll(plansDir, 0o755))

	planFile := filepath.Join(plansDir, "retry-plan.md")
	assert.NilError(t, os.WriteFile(planFile, []byte("# Retry Plan\nContent"), 0o644))

	transcriptPath := filepath.Join(tmpDir, "transcript.jsonl")
	transcript := `{"type":"tool_use","tool_name":"Write","tool_input":{"file_path":"` + planFile + `","content":"# Retry Plan\nContent"}}` + "\n"
	assert.NilError(t, os.WriteFile(transcriptPath, []byte(transcript), 0o644))

	// First call — callback fails, no review written.
	input := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err := HookPlanCallback(context.Background(), strings.NewReader(input), HookCallbackConfig{
		PlansDir:        plansDir,
		OutputDir:       outputDir,
		CallbackCmd:     "sh",
		CallbackArgs:    []string{"-c", "echo error >&2; exit 1"},
		CallbackTimeout: 10 * time.Second,
	})
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "callback failed"))

	// Verify 001_PLAN.md exists but 001_REVIEW.md does NOT (failure → no review).
	entries, err := os.ReadDir(outputDir)
	assert.NilError(t, err)
	assert.Equal(t, len(entries), 1)
	planDirPath := filepath.Join(outputDir, entries[0].Name())
	intermediateDir := filepath.Join(planDirPath, "_intermediate")

	_, err = os.Stat(filepath.Join(intermediateDir, "001_PLAN.md"))
	assert.NilError(t, err)
	_, err = os.Stat(filepath.Join(intermediateDir, "001_REVIEW.md"))
	assert.Assert(t, os.IsNotExist(err), "review should NOT be written on failure")

	// Second call with same content — should retry (not skip) since no review exists.
	markerFile := filepath.Join(tmpDir, "retry-marker")
	input2 := makePreToolUseInput("ExitPlanMode", "sess-1", transcriptPath)
	err = HookPlanCallback(context.Background(), strings.NewReader(input2), HookCallbackConfig{
		PlansDir:        plansDir,
		OutputDir:       outputDir,
		CallbackCmd:     "sh",
		CallbackArgs:    []string{"-c", "echo OK; touch " + markerFile},
		CallbackTimeout: 10 * time.Second,
	})
	var he *handler.HandlerError
	assert.Assert(t, errors.As(err, &he))

	// 002_PLAN.md should exist (retry happened).
	_, err = os.Stat(filepath.Join(intermediateDir, "002_PLAN.md"))
	assert.NilError(t, err, "expected 002_PLAN.md from retry")

	// 002_REVIEW.md should exist (success).
	_, err = os.Stat(filepath.Join(intermediateDir, "002_REVIEW.md"))
	assert.NilError(t, err, "expected 002_REVIEW.md from successful retry")

	// Marker file confirms callback ran.
	_, err = os.Stat(markerFile)
	assert.NilError(t, err, "expected retry marker file")
}
