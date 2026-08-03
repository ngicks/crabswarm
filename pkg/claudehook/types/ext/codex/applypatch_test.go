package codex

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestParsePatch_EditedFiles(t *testing.T) {
	tests := []struct {
		name  string
		patch string
		want  []string
	}{
		{
			name:  "add single",
			patch: "*** Begin Patch\n*** Add File: pkg/a.go\n+package a\n*** End Patch\n",
			want:  []string{"pkg/a.go"},
		},
		{
			name:  "update single",
			patch: "*** Begin Patch\n*** Update File: pkg/a.go\n@@\n-x\n+y\n*** End Patch\n",
			want:  []string{"pkg/a.go"},
		},
		{
			name: "update with rename uses move-to target",
			patch: "*** Begin Patch\n*** Update File: pkg/old.go\n" +
				"*** Move to: pkg/new.go\n@@\n-x\n+y\n*** End Patch\n",
			want: []string{"pkg/new.go"},
		},
		{
			name:  "delete excluded",
			patch: "*** Begin Patch\n*** Delete File: pkg/gone.go\n*** End Patch\n",
			want:  nil,
		},
		{
			name: "multiple ops in order, delete dropped",
			patch: "*** Begin Patch\n" +
				"*** Update File: pkg/old.go\n*** Move to: pkg/new.go\n@@\n-x\n+y\n" +
				"*** Add File: cmd/added.go\n+package cmd\n" +
				"*** Delete File: pkg/gone.go\n" +
				"*** End Patch\n",
			want: []string{"pkg/new.go", "cmd/added.go"},
		},
		{
			name:  "absolute path preserved",
			patch: "*** Begin Patch\n*** Add File: /abs/a.go\n+x\n*** End Patch\n",
			want:  []string{"/abs/a.go"},
		},
		{
			name:  "path with spaces is trimmed of surrounding whitespace only",
			patch: "*** Begin Patch\n*** Add File: dir/a b.go\n+x\n*** End Patch\n",
			want:  []string{"dir/a b.go"},
		},
		{
			name: "patched content that looks like a marker is ignored",
			patch: "*** Begin Patch\n*** Add File: pkg/a.go\n" +
				"+*** Add File: not/a/real/path.go\n+ *** Update File: nope.go\n" +
				"*** End Patch\n",
			want: []string{"pkg/a.go"},
		},
		{
			name:  "move-to without preceding update is ignored",
			patch: "*** Begin Patch\n*** Move to: stray.go\n*** End Patch\n",
			want:  nil,
		},
		{
			name:  "not a patch",
			patch: "Exit code: 0\nWall time: 0.5 seconds\n",
			want:  nil,
		},
		{
			name:  "empty command",
			patch: "",
			want:  nil,
		},
		{
			name:  "crlf line endings",
			patch: "*** Begin Patch\r\n*** Add File: pkg/a.go\r\n+package a\r\n*** End Patch\r\n",
			want:  []string{"pkg/a.go"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParsePatch(tc.patch).EditedFiles()
			assert.DeepEqual(t, got, tc.want)
		})
	}
}

func TestParsePatch_Operations(t *testing.T) {
	patch := "*** Begin Patch\n" +
		"*** Update File: pkg/old.go\n*** Move to: pkg/new.go\n@@\n-x\n+y\n" +
		"*** Add File: cmd/added.go\n+package cmd\n" +
		"*** Delete File: pkg/gone.go\n" +
		"*** End Patch\n"
	want := []PatchOperation{
		{Kind: PatchKindUpdate, Path: "pkg/old.go", MoveTo: "pkg/new.go"},
		{Kind: PatchKindAdd, Path: "cmd/added.go"},
		{Kind: PatchKindDelete, Path: "pkg/gone.go"},
	}
	assert.DeepEqual(t, ParsePatch(patch).Operations, want)
}

func TestParsePatch_EnvironmentId(t *testing.T) {
	patch := "*** Begin Patch\n" +
		"*** Environment ID: remote\n" +
		"*** Add File: pkg/a.go\n+x\n" +
		"*** End Patch\n"

	got := ParsePatch(patch)
	assert.Assert(t, got.EnvironmentId != nil)
	assert.Equal(t, *got.EnvironmentId, "remote")
	assert.DeepEqual(t, got.Operations, []PatchOperation{
		{Kind: PatchKindAdd, Path: "pkg/a.go"},
	})
}

func TestApplyPatchInput_RoundTrip(t *testing.T) {
	// Decoded values re-emit their preserved raw bytes verbatim.
	const raw = `{"command":"*** Begin Patch\n*** Update File: a.go\n` +
		`@@\n-x\n+y\n*** End Patch\n"}`

	var in ApplyPatchInput
	assert.NilError(t, json.Unmarshal([]byte(raw), &in))
	assert.Equal(t, in.Patch.Operations[0].Kind, PatchKindUpdate)
	assert.Equal(t, in.Patch.Operations[0].Path, "a.go")

	got, err := json.Marshal(in)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}

func TestApplyPatchInput_HTMLCharsSemantic(t *testing.T) {
	// '<' is unescaped on the wire (Codex uses serde_json). json.Marshal
	// re-escapes it to <, so bytes differ but the value round-trips.
	const raw = `{"command":"if a < b {}"}`

	var in ApplyPatchInput
	assert.NilError(t, json.Unmarshal([]byte(raw), &in))
	got, err := json.Marshal(in)
	assert.NilError(t, err)

	var back ApplyPatchInput
	assert.NilError(t, json.Unmarshal(got, &back))
	assert.Equal(t, back.Command, in.Command)
}

func TestApplyPatchInput_MarshalFromFields(t *testing.T) {
	// A value built programmatically (no preserved raw) marshals from Command.
	in := ApplyPatchInput{Command: "echo hi"}
	got, err := json.Marshal(in)
	assert.NilError(t, err)
	assert.Equal(t, string(got), `{"command":"echo hi"}`)
}

func TestApplyPatchInput_NotAnObject(t *testing.T) {
	var in ApplyPatchInput
	assert.Assert(t, in.UnmarshalJSON(json.RawMessage(`"just a string"`)) != nil)
}
