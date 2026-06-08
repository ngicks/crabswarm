package codex

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestApplyPatchOutput_Parse(t *testing.T) {
	const text = "Exit code: 0\nWall time: 0.6 seconds\nOutput:\n" +
		"Success. Updated the following files:\n" +
		"A cmd/added.go\nM pkg/changed.go\nD pkg/gone.go\n"
	raw, err := json.Marshal(text)
	assert.NilError(t, err)

	var out ApplyPatchOutput
	assert.NilError(t, json.Unmarshal(raw, &out))

	assert.Equal(t, out.Text, text)
	assert.Assert(t, out.ExitCode != nil)
	assert.Equal(t, *out.ExitCode, int64(0))
	assert.Assert(t, out.WallTimeSeconds != nil)
	assert.Equal(t, *out.WallTimeSeconds, 0.6)
	assert.DeepEqual(t, out.Summary, []SummaryEntry{
		{Kind: PatchKindAdd, Path: "cmd/added.go"},
		{Kind: PatchKindUpdate, Path: "pkg/changed.go"},
		{Kind: PatchKindDelete, Path: "pkg/gone.go"},
	})
}

func TestApplyPatchOutput_TotalOutputLines(t *testing.T) {
	const text = "Exit code: 1\nWall time: 2.3 seconds\nTotal output lines: 5000\nOutput:\nboom\n"
	raw, err := json.Marshal(text)
	assert.NilError(t, err)

	var out ApplyPatchOutput
	assert.NilError(t, json.Unmarshal(raw, &out))
	assert.Equal(t, *out.ExitCode, int64(1))
	assert.Assert(t, out.TotalOutputLines != nil)
	assert.Equal(t, *out.TotalOutputLines, int64(5000))
	assert.Equal(t, out.Output, "boom\n")
	assert.Assert(t, out.Summary == nil)
}

func TestApplyPatchOutput_RoundTrip(t *testing.T) {
	// Decoded values re-emit their preserved raw bytes verbatim.
	const raw = `"Exit code: 0\nWall time: 0.1 seconds\nOutput:\nupdated\n"`

	var out ApplyPatchOutput
	assert.NilError(t, json.Unmarshal([]byte(raw), &out))
	got, err := json.Marshal(out)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}

func TestApplyPatchOutput_NonStringTolerated(t *testing.T) {
	const raw = `{"unexpected":true}`
	var out ApplyPatchOutput
	assert.NilError(t, json.Unmarshal([]byte(raw), &out))
	assert.Equal(t, out.Text, "")
	got, err := json.Marshal(out)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}
