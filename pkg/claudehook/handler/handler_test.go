package handler

import (
	"encoding/json"
	"testing"

	"github.com/ngicks/crabswarm/pkg/claudehook/types"
	"gotest.tools/v3/assert"
)

func TestHandlerError_Error_Allow(t *testing.T) {
	he := &HandlerError{}
	assert.Equal(t, he.Error(), "hook: allow")
}

func TestHandlerError_Error_Block(t *testing.T) {
	he := &HandlerError{
		Output: &types.SyncHookJSONOutput{
			Decision: new(types.HookDecisionBlock),
			Reason:   new("not allowed"),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: not allowed")
}

func TestHandlerError_Error_BlockNoReason(t *testing.T) {
	he := &HandlerError{
		Output: &types.SyncHookJSONOutput{
			Decision: new(types.HookDecisionBlock),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: ")
}

// A block is conveyed as JSON on stdout — there is no exit-2 + stderr form
// anymore — so the exact bytes Block produces are what a hook consumer reads.
// Matched whole rather than by key lookup: a field leaking into the object
// would change the hook's meaning, and a per-key check cannot see a stray one.
func TestBlock_JSON(t *testing.T) {
	data, err := json.Marshal(Block("lint failed").Output)
	assert.NilError(t, err)
	assert.Equal(t, string(data), `{"decision":"block","reason":"lint failed"}`)
}

func TestNewPermissionRequestAllowError(t *testing.T) {
	he := PermissionAllow(nil, nil)

	assert.Assert(t, he.Output != nil)
	assert.Assert(t, he.Output.HookSpecificOutput.GetValue() != nil)
	hso, ok := he.Output.HookSpecificOutput.GetPermissionRequest()
	assert.Assert(
		t,
		ok,
		"expected HookSpecificOutputPermissionRequest, got %T",
		he.Output.HookSpecificOutput.GetValue(),
	)
	assert.Equal(t, string(hso.HookEventName), "PermissionRequest")
	assert.Assert(t, hso.Decision.GetValue() != nil)
	dec, ok := hso.Decision.GetAllow()
	assert.Assert(t, ok, "expected PermissionRequestDecisionAllow, got %T", hso.Decision.GetValue())
	assert.Equal(t, dec.Behavior, types.PermissionRequestDecisionBehaviorAllow)
}

func TestNewPermissionRequestAllowError_JSON(t *testing.T) {
	he := PermissionAllow(nil, nil)

	data, err := json.Marshal(he.Output)
	assert.NilError(t, err)

	// Verify the JSON structure matches what Claude Code expects.
	var result map[string]any
	assert.NilError(t, json.Unmarshal(data, &result))

	hso, ok := result["hookSpecificOutput"].(map[string]any)
	assert.Assert(t, ok, "expected hookSpecificOutput in output")

	assert.Equal(t, hso["hookEventName"], "PermissionRequest")

	decision, ok := hso["decision"].(map[string]any)
	assert.Assert(t, ok, "expected decision in hookSpecificOutput")
	assert.Equal(t, decision["behavior"], string(types.PermissionRequestDecisionBehaviorAllow))
}
