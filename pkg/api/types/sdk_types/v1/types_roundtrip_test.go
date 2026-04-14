package sdktypesv1

import (
	"encoding/json"
	"testing"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func mustProto[T any](t *testing.T, v T, err error) T {
	t.Helper()
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	return v
}

func compareJSON(t *testing.T, want, got []byte) {
	t.Helper()
	if string(want) != string(got) {
		t.Fatalf("json mismatch\nwant: %s\ngot:  %s", string(want), string(got))
	}
}

func TestJSONProtoRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		goType    any
		toProto   func(v any) any
		fromProto func(v any) any
	}{
		{
			name:   "PermissionUpdate addRules",
			input:  `{"type":"addRules","rules":[{"toolName":"Bash","ruleContent":"ls"}],"behavior":"allow","destination":"session"}`,
			goType: &PermissionUpdateAddRules{},
			toProto: func(v any) any {
				got, err := PermissionUpdateToProto(v.(PermissionUpdate))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := PermissionUpdateFromProto(v.(*pb.PermissionUpdate))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "PermissionUpdate unknown",
			input:  `{"type":"futureRule","payload":{"x":1}}`,
			goType: &PermissionUpdateUnknown{},
			toProto: func(v any) any {
				got, err := PermissionUpdateToProto(v.(PermissionUpdate))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := PermissionUpdateFromProto(v.(*pb.PermissionUpdate))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "PermissionResult allow",
			input:  `{"behavior":"allow","updatedInput":{"command":"ls"},"updatedPermissions":[{"type":"setMode","mode":"plan","destination":"session"}],"toolUseID":"tool-1"}`,
			goType: &PermissionResultAllow{},
			toProto: func(v any) any {
				got, err := PermissionResultToProto(v.(PermissionResult))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := PermissionResultFromProto(v.(*pb.PermissionResult))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "PermissionResult unknown",
			input:  `{"behavior":"future","note":"opaque"}`,
			goType: &PermissionResultUnknown{},
			toProto: func(v any) any {
				got, err := PermissionResultToProto(v.(PermissionResult))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := PermissionResultFromProto(v.(*pb.PermissionResult))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "PermissionRequestDecision allow",
			input:  `{"behavior":"allow","updatedInput":{"path":"a.txt"},"updatedPermissions":[{"type":"addDirectories","directories":["/tmp"],"destination":"session"}]}`,
			goType: &PermissionRequestDecisionAllow{},
			toProto: func(v any) any {
				got, err := PermissionRequestDecisionToProto(v.(PermissionRequestDecision))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := PermissionRequestDecisionFromProto(v.(*pb.PermissionRequestDecision))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "PermissionRequestDecision unknown",
			input:  `{"behavior":"maybe","message":"future"}`,
			goType: &PermissionRequestDecisionUnknown{},
			toProto: func(v any) any {
				got, err := PermissionRequestDecisionToProto(v.(PermissionRequestDecision))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := PermissionRequestDecisionFromProto(v.(*pb.PermissionRequestDecision))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookSpecificOutput preToolUse",
			input:  `{"hookEventName":"PreToolUse","permissionDecision":"allow","permissionDecisionReason":"ok","updatedInput":{"command":"pwd"},"additionalContext":"ctx"}`,
			goType: &HookSpecificOutputPreToolUse{},
			toProto: func(v any) any {
				got, err := HookSpecificOutputToProto(v.(HookSpecificOutput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookSpecificOutputFromProto(v.(*pb.HookSpecificOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookSpecificOutput permissionRequest",
			input:  `{"hookEventName":"PermissionRequest","decision":{"behavior":"deny","message":"no","interrupt":true}}`,
			goType: &HookSpecificOutputPermissionRequest{},
			toProto: func(v any) any {
				got, err := HookSpecificOutputToProto(v.(HookSpecificOutput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookSpecificOutputFromProto(v.(*pb.HookSpecificOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookSpecificOutput unknown",
			input:  `{"hookEventName":"FutureHook","extra":true}`,
			goType: &HookSpecificOutputUnknown{},
			toProto: func(v any) any {
				got, err := HookSpecificOutputToProto(v.(HookSpecificOutput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookSpecificOutputFromProto(v.(*pb.HookSpecificOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookJSONOutput async",
			input:  `{"async":true,"asyncTimeout":42}`,
			goType: &AsyncHookJSONOutput{},
			toProto: func(v any) any {
				got, err := HookJSONOutputToProto(v.(HookJSONOutput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookJSONOutputFromProto(v.(*pb.HookJSONOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookJSONOutput sync",
			input:  `{"continue":true,"suppressOutput":false,"stopReason":"x","decision":"approve","systemMessage":"m","reason":"r","hookSpecificOutput":{"hookEventName":"Notification","additionalContext":"ctx"}}`,
			goType: &SyncHookJSONOutput{},
			toProto: func(v any) any {
				got, err := HookJSONOutputToProto(v.(HookJSONOutput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookJSONOutputFromProto(v.(*pb.HookJSONOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookJSONOutput unknown",
			input:  `{"future":"value"}`,
			goType: &HookJSONOutputUnknown{},
			toProto: func(v any) any {
				got, err := HookJSONOutputToProto(v.(HookJSONOutput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookJSONOutputFromProto(v.(*pb.HookJSONOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput preToolUse",
			input:  `{"session_id":"sess-1","transcript_path":"/tmp/transcript.jsonl","cwd":"/repo","permission_mode":"default","agent_id":"agent-1","agent_type":"worker","hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"pwd","timeout":10},"tool_use_id":"tool-1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(v.(HookInput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput permissionRequest",
			input:  `{"session_id":"sess-2","transcript_path":"/tmp/transcript.jsonl","cwd":"/repo","hook_event_name":"PermissionRequest","tool_name":"Read","tool_input":{"file_path":"a.txt","offset":0},"tool_use_id":"tool-2","permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Read","ruleContent":"a.txt"}],"behavior":"allow","destination":"session"}]}`,
			goType: &PermissionRequestHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(v.(HookInput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput unknown",
			input:  `{"hook_event_name":"FutureHook","opaque":{"v":1}}`,
			goType: &HookInputUnknown{},
			toProto: func(v any) any {
				got, err := HookInputToProto(v.(HookInput))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := json.Unmarshal([]byte(tc.input), tc.goType); err != nil {
				t.Fatalf("unmarshal input into go type: %v", err)
			}

			protoValue := tc.toProto(tc.goType)
			roundTripped := tc.fromProto(protoValue)

			got, err := json.Marshal(roundTripped)
			if err != nil {
				t.Fatalf("marshal round-tripped value: %v", err)
			}

			compareJSON(t, []byte(tc.input), got)
		})
	}
}
