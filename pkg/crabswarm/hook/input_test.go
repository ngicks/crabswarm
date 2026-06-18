package hook

import (
	"encoding/json"
	"testing"

	sdktypesv1 "github.com/ngicks/crabswarm/pkg/api/types/sdktypes/v1"
	"gotest.tools/v3/assert"
)

func marshalHookInput(t *testing.T, in sdktypesv1.HookInput_Value) []byte {
	t.Helper()
	data, err := json.Marshal(sdktypesv1.NewHookInput(in))
	assert.NilError(t, err)
	return data
}

func TestParse_EditEnvelope(t *testing.T) {
	raw := marshalHookInput(t, &sdktypesv1.PreToolUseHookInput{
		BaseHookInput: sdktypesv1.BaseHookInput{SessionID: "sess-1", Cwd: "/work"},
		HookEventName: sdktypesv1.HookEventPreToolUse,
		ToolName:      sdktypesv1.ToolNameWrite,
		ToolInput: sdktypesv1.NewToolInputSchemas(
			&sdktypesv1.FileWriteInput{FilePath: "/work/src/main.go", Content: "x"},
		),
		ToolUseID: "t1",
	})

	got, err := Parse(raw)
	assert.NilError(t, err)
	assert.Equal(t, got.Event, sdktypesv1.HookEventPreToolUse)
	assert.Equal(t, got.ToolName, sdktypesv1.ToolNameWrite)
	assert.Equal(t, got.SessionID, "sess-1")
	assert.Equal(t, got.Cwd, "/work")
	assert.DeepEqual(t, got.Files, []string{"/work/src/main.go"})
	_, ok := got.Input.(*sdktypesv1.PreToolUseHookInput)
	assert.Assert(t, ok, "Input should be *PreToolUseHookInput, got %T", got.Input)
}

func TestParse_NonEditEnvelopeHasNoFiles(t *testing.T) {
	raw := marshalHookInput(t, &sdktypesv1.PreToolUseHookInput{
		BaseHookInput: sdktypesv1.BaseHookInput{Cwd: "/work"},
		HookEventName: sdktypesv1.HookEventPreToolUse,
		ToolName:      sdktypesv1.ToolNameBash,
		ToolInput:     sdktypesv1.NewToolInputSchemas(&sdktypesv1.BashInput{Command: "ls"}),
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
