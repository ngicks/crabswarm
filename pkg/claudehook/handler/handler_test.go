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
			Decision: ptr(string(HookDecisionBlock)),
			Reason:   ptr("not allowed"),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: not allowed")
}

func TestHandlerError_Error_BlockNoReason(t *testing.T) {
	he := &HandlerError{
		Output: &models.SyncHookJSONOutput{
			Decision: ptr(string(HookDecisionBlock)),
		},
	}
	assert.Equal(t, he.Error(), "hook: block: ")
}

func TestNewPermissionRequestAllowError(t *testing.T) {
	he := NewPermissionRequestAllowError()

	assert.Assert(t, he.Output != nil)
	assert.Assert(t, he.Output.HookSpecificOutput != nil)
	assert.Assert(t, he.Output.HookSpecificOutput.HookEventName != nil)
	assert.Equal(t, *he.Output.HookSpecificOutput.HookEventName, "PermissionRequest")
	assert.Assert(t, he.Output.HookSpecificOutput.Decision != nil)
	assert.Equal(t, he.Output.HookSpecificOutput.Decision.Behavior, "allow")
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
