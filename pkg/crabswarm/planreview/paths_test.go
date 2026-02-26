package planreview

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

func TestPlanDirName(t *testing.T) {
	ts := time.Date(2026, 2, 24, 20, 11, 16, 0, time.FixedZone("JST", 9*3600))
	got := PlanDirName(ts, "my-plan")
	assert.Equal(t, got, "2026-02-24T20:11:16+09:00-my-plan")
}

func TestIntermediateFileName(t *testing.T) {
	tests := []struct {
		name      string
		iteration int
		step      int
		suffix    string
		want      string
	}{
		{"first plan", 1, 0, "PLAN", "001_00_PLAN.md"},
		{"first review", 1, 1, "REVIEW", "001_01_REVIEW.md"},
		{"tenth plan", 10, 0, "PLAN", "010_00_PLAN.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntermediateFileName(tt.iteration, tt.step, tt.suffix)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestDerivePlanName(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "markdown heading",
			content: "# Add Plan Iteration Hook\n\nSome details.",
			want:    "add-plan-iteration-hook",
		},
		{
			name:    "heading with multiple spaces",
			content: "##  My  Great   Plan\n",
			want:    "my-great-plan",
		},
		{
			name:    "plain text first line",
			content: "Implement Auth System\nMore details.",
			want:    "implement-auth-system",
		},
		{
			name:    "leading blank lines",
			content: "\n\n\n# Real Title\n",
			want:    "real-title",
		},
		{
			name:    "tabs in title",
			content: "# Word1\tWord2\n",
			want:    "word1-word2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "plan.md")
			assert.NilError(t, os.WriteFile(tmpFile, []byte(tt.content), 0o644))

			got, err := DerivePlanName(tmpFile)
			assert.NilError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestDerivePlanName_EmptyFile(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "empty.md")
	assert.NilError(t, os.WriteFile(tmpFile, []byte(""), 0o644))

	_, err := DerivePlanName(tmpFile)
	assert.Assert(t, err != nil)
}

func TestDerivePlanName_OnlyHashLines(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "hashes.md")
	assert.NilError(t, os.WriteFile(tmpFile, []byte("##\n###\n"), 0o644))

	_, err := DerivePlanName(tmpFile)
	assert.Assert(t, err != nil)
}

func TestDerivePlanName_NonexistentFile(t *testing.T) {
	_, err := DerivePlanName("/nonexistent/plan.md")
	assert.Assert(t, err != nil)
}

func TestPathWithinDir(t *testing.T) {
	// Create a temp dir structure for testing.
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "plans")
	assert.NilError(t, os.MkdirAll(subDir, 0o755))

	// Create a file inside the dir.
	filePath := filepath.Join(subDir, "test.md")
	assert.NilError(t, os.WriteFile(filePath, []byte("test"), 0o644))

	tests := []struct {
		name     string
		file     string
		dir      string
		want     bool
		wantErr  bool
	}{
		{
			name: "file under dir",
			file: filePath,
			dir:  subDir,
			want: true,
		},
		{
			name: "file under parent dir",
			file: filePath,
			dir:  tmpDir,
			want: true,
		},
		{
			name: "file not under dir",
			file: filepath.Join(tmpDir, "other", "test.md"),
			dir:  subDir,
			want: false,
		},
		{
			name: "nonexistent file under existing dir",
			file: filepath.Join(subDir, "nonexistent.md"),
			dir:  subDir,
			want: true,
		},
		{
			name: "dir equals file",
			file: subDir,
			dir:  subDir,
			want: true,
		},
		{
			name:    "nonexistent dir",
			file:    filePath,
			dir:     filepath.Join(tmpDir, "nosuchdir"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := PathWithinDir(tt.file, tt.dir)
			if tt.wantErr {
				assert.Assert(t, err != nil, "expected error")
				return
			}
			assert.NilError(t, err)
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestPathWithinDir_PreventsPrefixFalsePositive(t *testing.T) {
	// /tmp/plans2/file.md should NOT match /tmp/plans
	tmpDir := t.TempDir()
	plans := filepath.Join(tmpDir, "plans")
	plans2 := filepath.Join(tmpDir, "plans2")
	assert.NilError(t, os.MkdirAll(plans, 0o755))
	assert.NilError(t, os.MkdirAll(plans2, 0o755))

	filePath := filepath.Join(plans2, "file.md")
	assert.NilError(t, os.WriteFile(filePath, []byte("x"), 0o644))

	got, err := PathWithinDir(filePath, plans)
	assert.NilError(t, err)
	assert.Equal(t, got, false)
}

func TestCountIterations(t *testing.T) {
	tmpDir := t.TempDir()

	// Empty dir.
	count, err := CountIterations(tmpDir)
	assert.NilError(t, err)
	assert.Equal(t, count, 0)

	// Add some iteration files.
	assert.NilError(t, os.WriteFile(filepath.Join(tmpDir, "001_00_PLAN.md"), []byte("plan1"), 0o644))
	assert.NilError(t, os.WriteFile(filepath.Join(tmpDir, "001_01_REVIEW.md"), []byte("review1"), 0o644))
	assert.NilError(t, os.WriteFile(filepath.Join(tmpDir, "002_00_PLAN.md"), []byte("plan2"), 0o644))

	count, err = CountIterations(tmpDir)
	assert.NilError(t, err)
	assert.Equal(t, count, 2)
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "a", "b", "c")

	assert.NilError(t, EnsureDir(newDir))

	info, err := os.Stat(newDir)
	assert.NilError(t, err)
	assert.Assert(t, info.IsDir())

	// Calling again is idempotent.
	assert.NilError(t, EnsureDir(newDir))
}
