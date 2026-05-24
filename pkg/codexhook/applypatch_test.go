package codexhook

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseApplyPatchInput(t *testing.T) {
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
			toolInput, err := json.Marshal(map[string]string{"command": tc.patch})
			assert.NilError(t, err)

			got, err := ParseApplyPatchInput(toolInput)
			assert.NilError(t, err)
			assert.Equal(t, got.Command, tc.patch)
			assert.DeepEqual(t, got.Files, tc.want)
		})
	}
}

// Codex always sends apply_patch's tool_input as a {"command": ...} object;
// a non-object value is a decode error rather than a silently empty result.
func TestParseApplyPatchInput_NotAnObject(t *testing.T) {
	_, err := ParseApplyPatchInput(json.RawMessage(`"just a string"`))
	assert.Assert(t, err != nil)
}
