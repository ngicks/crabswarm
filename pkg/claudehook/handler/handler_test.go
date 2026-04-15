package handler

import (
	"encoding/json"
	"testing"

	sdktypesv1 "github.com/ngicks/crabswarm/pkg/api/types/sdktypes/v1"
	"gotest.tools/v3/assert"
)

func TestHandlerError_Error_Allow(t *testing.T) {
	he := &HandlerError{}
	assert.Equal(t, he.Error(), "hook: allow")
}

func TestHandlerError_Error_Block(t *testing.T) {
	he := &HandlerError{
		Output: &sdktypesv1.SyncHookJSONOutput{
			Decision: new(sdktypesv1.HookDecisionBlock),
			Reason:   new("not allowed"),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: not allowed")
}

func TestHandlerError_Error_BlockNoReason(t *testing.T) {
	he := &HandlerError{
		Output: &sdktypesv1.SyncHookJSONOutput{
			Decision: new(sdktypesv1.HookDecisionBlock),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: ")
}

func TestNewPermissionRequestAllowError(t *testing.T) {
	he := Allow(nil, nil)

	assert.Assert(t, he.Output != nil)
	assert.Assert(t, he.Output.HookSpecificOutput != nil)
	hso, ok := he.Output.HookSpecificOutput.(*sdktypesv1.HookSpecificOutputPermissionRequest)
	assert.Assert(t, ok, "expected HookSpecificOutputPermissionRequest, got %T", he.Output.HookSpecificOutput)
	assert.Equal(t, string(hso.HookEventName), "PermissionRequest")
	assert.Assert(t, hso.Decision != nil)
	dec, ok := hso.Decision.(*sdktypesv1.PermissionRequestDecisionAllow)
	assert.Assert(t, ok, "expected PermissionRequestDecisionAllow, got %T", hso.Decision)
	assert.Equal(t, string(dec.Behavior), "allow")
}

func TestNewPermissionRequestAllowError_JSON(t *testing.T) {
	he := Allow(nil, nil)

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
	assert.Equal(t, decision["behavior"], "allow")
}
