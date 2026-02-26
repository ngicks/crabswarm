package models_test

import (
	"encoding/json"
	"testing"

	"github.com/ngicks/crabswarm/pkg/claudesdk/models"
	"gotest.tools/v3/assert"
)

func ptr[V any](v V) *V {
	return &v
}

func TestSyncHookJSONOutput_ToProto_NilOutput(t *testing.T) {
	m := &models.SyncHookJSONOutput{}
	p, err := m.ToProto()
	assert.NilError(t, err)
	assert.Assert(t, p != nil)
	assert.Assert(t, p.HookSpecificOutput == nil)
}

func TestSyncHookJSONOutput_ToProto_Decision(t *testing.T) {
	m := &models.SyncHookJSONOutput{
		Decision: ptr("block"),
		Reason:   ptr("not allowed"),
	}
	p, err := m.ToProto()
	assert.NilError(t, err)
	assert.Equal(t, *p.Decision, "block")
	assert.Equal(t, *p.Reason, "not allowed")
}

func TestSyncHookJSONOutput_ToProto_PreToolUse(t *testing.T) {
	m := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			PermissionDecision:       ptr("allow"),
			PermissionDecisionReason: ptr("whitelisted"),
		},
	}
	p, err := m.ToProto()
	assert.NilError(t, err)
	assert.Assert(t, p.HookSpecificOutput != nil)
	ptu := p.HookSpecificOutput.GetPreToolUse()
	assert.Assert(t, ptu != nil)
	assert.Equal(t, *ptu.PermissionDecision, "allow")
	assert.Equal(t, *ptu.PermissionDecisionReason, "whitelisted")
}

func TestSyncHookJSONOutput_ToProto_PermissionRequestAllow(t *testing.T) {
	m := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			Decision: &models.PermissionRequestDecision{
				Behavior: "allow",
			},
		},
	}
	p, err := m.ToProto()
	assert.NilError(t, err)
	assert.Assert(t, p.HookSpecificOutput != nil)
	pr := p.HookSpecificOutput.GetPermissionRequest()
	assert.Assert(t, pr != nil)
	assert.Assert(t, pr.Decision != nil)
	assert.Assert(t, pr.Decision.GetAllow() != nil)
}

func TestSyncHookJSONOutput_ToProto_PermissionRequestDeny(t *testing.T) {
	m := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			Decision: &models.PermissionRequestDecision{
				Behavior: "deny",
				Message:  ptr("forbidden"),
			},
		},
	}
	p, err := m.ToProto()
	assert.NilError(t, err)
	pr := p.HookSpecificOutput.GetPermissionRequest()
	assert.Assert(t, pr != nil)
	deny := pr.Decision.GetDeny()
	assert.Assert(t, deny != nil)
	assert.Equal(t, deny.Message, "forbidden")
}

func TestSyncHookJSONOutput_FromProto_Nil(t *testing.T) {
	m := &models.SyncHookJSONOutput{}
	// Build a proto with basic fields only.
	p := &models.SyncHookJSONOutput{
		Continue:   ptr(true),
		StopReason: ptr("done"),
	}
	proto, err := p.ToProto()
	assert.NilError(t, err)

	err = m.FromProto(proto)
	assert.NilError(t, err)
	assert.Assert(t, m.Continue != nil)
	assert.Equal(t, *m.Continue, true)
	assert.Assert(t, m.StopReason != nil)
	assert.Equal(t, *m.StopReason, "done")
}

func TestSyncHookJSONOutput_RoundTrip_PermissionRequestAllow(t *testing.T) {
	original := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			Decision: &models.PermissionRequestDecision{
				Behavior: "allow",
			},
		},
	}

	// model -> proto
	p, err := original.ToProto()
	assert.NilError(t, err)

	// proto -> model
	result := &models.SyncHookJSONOutput{}
	err = result.FromProto(p)
	assert.NilError(t, err)

	assert.Assert(t, result.HookSpecificOutput != nil)
	assert.Assert(t, result.HookSpecificOutput.Decision != nil)
	assert.Equal(t, result.HookSpecificOutput.Decision.Behavior, "allow")
}

func TestSyncHookJSONOutput_RoundTrip_PermissionRequestDeny(t *testing.T) {
	original := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			Decision: &models.PermissionRequestDecision{
				Behavior:  "deny",
				Message:   ptr("not allowed"),
				Interrupt: ptr(true),
			},
		},
	}

	p, err := original.ToProto()
	assert.NilError(t, err)

	result := &models.SyncHookJSONOutput{}
	err = result.FromProto(p)
	assert.NilError(t, err)

	assert.Assert(t, result.HookSpecificOutput != nil)
	assert.Assert(t, result.HookSpecificOutput.Decision != nil)
	assert.Equal(t, result.HookSpecificOutput.Decision.Behavior, "deny")
	assert.Assert(t, result.HookSpecificOutput.Decision.Message != nil)
	assert.Equal(t, *result.HookSpecificOutput.Decision.Message, "not allowed")
	assert.Assert(t, result.HookSpecificOutput.Decision.Interrupt != nil)
	assert.Equal(t, *result.HookSpecificOutput.Decision.Interrupt, true)
}

func TestSyncHookJSONOutput_RoundTrip_AdditionalContext(t *testing.T) {
	original := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			AdditionalContext: ptr("extra context"),
		},
	}

	p, err := original.ToProto()
	assert.NilError(t, err)

	result := &models.SyncHookJSONOutput{}
	err = result.FromProto(p)
	assert.NilError(t, err)

	assert.Assert(t, result.HookSpecificOutput != nil)
	assert.Assert(t, result.HookSpecificOutput.AdditionalContext != nil)
	assert.Equal(t, *result.HookSpecificOutput.AdditionalContext, "extra context")
}

func TestSyncHookJSONOutput_JSON_PermissionRequestAllow(t *testing.T) {
	m := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			HookEventName: ptr("PermissionRequest"),
			Decision: &models.PermissionRequestDecision{
				Behavior: "allow",
			},
		},
	}

	data, err := json.Marshal(m)
	assert.NilError(t, err)

	var raw map[string]any
	assert.NilError(t, json.Unmarshal(data, &raw))

	hso := raw["hookSpecificOutput"].(map[string]any)
	assert.Equal(t, hso["hookEventName"], "PermissionRequest")

	decision := hso["decision"].(map[string]any)
	assert.Equal(t, decision["behavior"], "allow")
}

func TestSyncHookJSONOutput_ToProto_UnknownBehavior(t *testing.T) {
	m := &models.SyncHookJSONOutput{
		HookSpecificOutput: &models.HookSpecificOutput{
			Decision: &models.PermissionRequestDecision{
				Behavior: "unknown",
			},
		},
	}
	_, err := m.ToProto()
	assert.Assert(t, err != nil, "expected error for unknown behavior")
}
