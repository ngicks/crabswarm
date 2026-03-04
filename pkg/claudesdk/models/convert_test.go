package models_test

import (
	"encoding/json"
	"testing"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"github.com/ngicks/crabswarm/pkg/claudesdk/models"
	"google.golang.org/protobuf/proto"
	"gotest.tools/v3/assert"
)

func ptr[V any](v V) *V {
	return &v
}

func TestSyncHookJSONOutput_ProtoModelRoundTrip_Table(t *testing.T) {
	tests := []struct {
		name  string
		model models.SyncHookJSONOutput
		proto *pb.SyncHookJSONOutput
	}{
		{
			name:  "empty",
			model: models.SyncHookJSONOutput{},
			proto: &pb.SyncHookJSONOutput{},
		},
		{
			name: "top-level decision",
			model: models.SyncHookJSONOutput{
				Decision: ptr("block"),
				Reason:   ptr("not allowed"),
			},
			proto: &pb.SyncHookJSONOutput{
				Decision: ptr("block"),
				Reason:   ptr("not allowed"),
			},
		},
		{
			name: "pre-tool-use permission fields",
			model: models.SyncHookJSONOutput{
				HookSpecificOutput: &models.HookSpecificOutput{
					PermissionDecision:       ptr("allow"),
					PermissionDecisionReason: ptr("whitelisted"),
				},
			},
			proto: &pb.SyncHookJSONOutput{
				HookSpecificOutput: &pb.HookSpecificOutput{
					Output: &pb.HookSpecificOutput_PreToolUse{PreToolUse: &pb.PreToolUseHookSpecificOutput{
						PermissionDecision:       ptr("allow"),
						PermissionDecisionReason: ptr("whitelisted"),
					}},
				},
			},
		},
		{
			name: "permission request allow",
			model: models.SyncHookJSONOutput{
				HookSpecificOutput: &models.HookSpecificOutput{
					Decision: &models.PermissionRequestDecision{Behavior: "allow"},
				},
			},
			proto: &pb.SyncHookJSONOutput{
				HookSpecificOutput: &pb.HookSpecificOutput{
					Output: &pb.HookSpecificOutput_PermissionRequest{PermissionRequest: &pb.PermissionRequestHookSpecificOutput{
						Decision: &pb.PermissionResult{Result: &pb.PermissionResult_Allow{Allow: &pb.AllowResult{}}},
					}},
				},
			},
		},
		{
			name: "permission request deny",
			model: models.SyncHookJSONOutput{
				HookSpecificOutput: &models.HookSpecificOutput{
					Decision: &models.PermissionRequestDecision{
						Behavior:  "deny",
						Message:   ptr("not allowed"),
						Interrupt: ptr(true),
					},
				},
			},
			proto: &pb.SyncHookJSONOutput{
				HookSpecificOutput: &pb.HookSpecificOutput{
					Output: &pb.HookSpecificOutput_PermissionRequest{PermissionRequest: &pb.PermissionRequestHookSpecificOutput{
						Decision: &pb.PermissionResult{Result: &pb.PermissionResult_Deny{Deny: &pb.DenyResult{
							Message:   "not allowed",
							Interrupt: ptr(true),
						}}},
					}},
				},
			},
		},
		{
			name: "additional context",
			model: models.SyncHookJSONOutput{
				HookSpecificOutput: &models.HookSpecificOutput{
					AdditionalContext: ptr("extra context"),
				},
			},
			proto: &pb.SyncHookJSONOutput{
				HookSpecificOutput: &pb.HookSpecificOutput{
					Output: &pb.HookSpecificOutput_PostToolUse{PostToolUse: &pb.PostToolUseHookSpecificOutput{
						AdditionalContext: ptr("extra context"),
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotProto, err := tt.model.ToProto()
			assert.NilError(t, err)
			assert.Assert(t, proto.Equal(gotProto, tt.proto))

			var gotModel models.SyncHookJSONOutput
			err = gotModel.FromProto(tt.proto)
			assert.NilError(t, err)
			assert.DeepEqual(t, gotModel, tt.model)

			roundTripProto, err := gotModel.ToProto()
			assert.NilError(t, err)
			assert.Assert(t, proto.Equal(roundTripProto, tt.proto))
		})
	}
}

func TestSyncHookJSONOutput_ToProto_UnknownBehavior_Table(t *testing.T) {
	tests := []struct {
		name  string
		model models.SyncHookJSONOutput
	}{
		{
			name: "permission request unknown behavior",
			model: models.SyncHookJSONOutput{
				HookSpecificOutput: &models.HookSpecificOutput{
					Decision: &models.PermissionRequestDecision{Behavior: "unknown"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.model.ToProto()
			assert.Assert(t, err != nil)
		})
	}
}

func TestSyncHookJSONOutput_JSON_Table(t *testing.T) {
	tests := []struct {
		name string
		in   models.SyncHookJSONOutput
		want map[string]any
	}{
		{
			name: "permission request allow",
			in: models.SyncHookJSONOutput{
				HookSpecificOutput: &models.HookSpecificOutput{
					HookEventName: ptr("PermissionRequest"),
					Decision:      &models.PermissionRequestDecision{Behavior: "allow"},
				},
			},
			want: map[string]any{
				"hookSpecificOutput": map[string]any{
					"hookEventName": "PermissionRequest",
					"decision": map[string]any{
						"behavior": "allow",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.in)
			assert.NilError(t, err)

			var got map[string]any
			assert.NilError(t, json.Unmarshal(data, &got))
			assert.DeepEqual(t, got, tt.want)
		})
	}
}
