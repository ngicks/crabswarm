package handler

import (
	"encoding/json"
	"testing"

	"github.com/ngicks/crabswarm/pkg/claudesdk/models"
	"gotest.tools/v3/assert"
)

func ptr[V any](v V) *V {
	return &v
}

func TestHandlerError_Error_Allow(t *testing.T) {
	he := &HandlerError{}
	assert.Equal(t, he.Error(), "hook: allow")
}

func TestHandlerError_Error_Block(t *testing.T) {
	he := &HandlerError{
		Output: &models.SyncHookJSONOutput{
			Decision: ptr(models.SyncHookJSONOutputDecision(HookDecisionBlock)),
			Reason:   ptr("not allowed"),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: not allowed")
}

func TestHandlerError_Error_BlockNoReason(t *testing.T) {
	he := &HandlerError{
		Output: &models.SyncHookJSONOutput{
			Decision: ptr(models.SyncHookJSONOutputDecision(HookDecisionBlock)),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: ")
}

func TestNewPermissionRequestAllowError(t *testing.T) {
	he := NewPermissionRequestAllowError()

	assert.Assert(t, he.Output != nil)
	assert.Assert(t, he.Output.HookSpecificOutput != nil)
	hso, ok := he.Output.HookSpecificOutput.(models.HookSpecificOutputPermissionRequest)
	assert.Assert(t, ok, "expected HookSpecificOutputPermissionRequest, got %T", he.Output.HookSpecificOutput)
	assert.Equal(t, string(hso.HookEventName), "PermissionRequest")
	assert.Assert(t, hso.Decision != nil)
	dec, ok := hso.Decision.(models.PermissionRequestDecisionAllow)
	assert.Assert(t, ok, "expected PermissionRequestDecisionAllow, got %T", hso.Decision)
	assert.Equal(t, string(dec.Behavior), "allow")
}

func TestNewPermissionRequestAllowError_JSON(t *testing.T) {
	he := NewPermissionRequestAllowError()

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
