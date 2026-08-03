package hook

import (
	"encoding/json"
	"testing"

	"github.com/ngicks/crabswarm/pkg/claudehook/types"
	"gotest.tools/v3/assert"
)

func marshalHookInput(t *testing.T, in types.HookInput_Value) []byte {
	t.Helper()
	data, err := json.Marshal(types.NewHookInput(in))
	assert.NilError(t, err)
	return data
}

func TestParse_EditEnvelope(t *testing.T) {
	raw := marshalHookInput(t, &types.PreToolUseHookInput{
		BaseHookInput: types.BaseHookInput{SessionID: "sess-1", Cwd: "/work"},
		HookEventName: types.HookEventPreToolUse,
		ToolName:      types.ToolNameWrite,
		ToolInput: types.NewToolInputSchemas(
			&types.FileWriteInput{FilePath: "/work/src/main.go", Content: "x"},
		),
		ToolUseID: "t1",
	})

	got, err := Parse(raw)
	assert.NilError(t, err)
	assert.Equal(t, got.Event, types.HookEventPreToolUse)
	assert.Equal(t, got.ToolName, types.ToolNameWrite)
	assert.Equal(t, got.SessionID, "sess-1")
	assert.Equal(t, got.Cwd, "/work")
	assert.DeepEqual(t, got.Files, []string{"/work/src/main.go"})
	_, ok := got.Input.(*types.PreToolUseHookInput)
	assert.Assert(t, ok, "Input should be *PreToolUseHookInput, got %T", got.Input)
}

func TestParse_NonEditEnvelopeHasNoFiles(t *testing.T) {
	raw := marshalHookInput(t, &types.PreToolUseHookInput{
		BaseHookInput: types.BaseHookInput{Cwd: "/work"},
		HookEventName: types.HookEventPreToolUse,
		ToolName:      types.ToolNameBash,
		ToolInput:     types.NewToolInputSchemas(&types.BashInput{Command: "ls"}),
		ToolUseID:     "t1",
	})

	got, err := Parse(raw)
	assert.NilError(t, err)
	assert.Equal(t, len(got.Files), 0)
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := Parse([]byte("{not json"))
	assert.Assert(t, err != nil)
}
