package types

import (
	"encoding/json"
	"testing"

	pb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/sdktypes/v1"
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
				got, err := PermissionUpdateToProto(NewPermissionUpdate(v.(PermissionUpdate_Value)))
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
				got, err := PermissionUpdateToProto(NewPermissionUpdate(v.(PermissionUpdate_Value)))
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
				got, err := PermissionResultToProto(NewPermissionResult(v.(PermissionResult_Value)))
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
				got, err := PermissionResultToProto(NewPermissionResult(v.(PermissionResult_Value)))
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
				got, err := PermissionRequestDecisionToProto(
					NewPermissionRequestDecision(v.(PermissionRequestDecision_Value)),
				)
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
				got, err := PermissionRequestDecisionToProto(
					NewPermissionRequestDecision(v.(PermissionRequestDecision_Value)),
				)
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
				got, err := HookSpecificOutputToProto(
					NewHookSpecificOutput(v.(HookSpecificOutput_Value)),
				)
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
				got, err := HookSpecificOutputToProto(
					NewHookSpecificOutput(v.(HookSpecificOutput_Value)),
				)
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
				got, err := HookSpecificOutputToProto(
					NewHookSpecificOutput(v.(HookSpecificOutput_Value)),
				)
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
				got, err := HookJSONOutputToProto(NewHookJSONOutput(v.(HookJSONOutput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookJSONOutputFromProto(v.(*pb.HookJSONOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookJSONOutput sync",
			input:  `{"continue":true,"suppressOutput":false,"stopReason":"x","decision":"approve","systemMessage":"m","terminalSequence":"","reason":"r","hookSpecificOutput":{"hookEventName":"Notification","additionalContext":"ctx"}}`,
			goType: &SyncHookJSONOutput{},
			toProto: func(v any) any {
				got, err := HookJSONOutputToProto(NewHookJSONOutput(v.(HookJSONOutput_Value)))
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
				got, err := HookJSONOutputToProto(NewHookJSONOutput(v.(HookJSONOutput_Value)))
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
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name: "HookInput postToolUse with duration",
			input: `{"session_id":"s","transcript_path":"t","cwd":"c",` +
				`"hook_event_name":"PostToolUse","tool_name":"Bash",` +
				`"tool_input":{"command":"ls"},` +
				`"tool_response":{"stdout":"x","stderr":"","interrupted":false},` +
				`"tool_use_id":"u1","duration_ms":42}`,
			goType: &PostToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name: "HookInput postToolUseFailure with duration",
			input: `{"session_id":"s","transcript_path":"t","cwd":"c",` +
				`"hook_event_name":"PostToolUseFailure","tool_name":"Bash",` +
				`"tool_input":{"command":"ls"},"tool_use_id":"u1",` +
				`"error":"boom","is_interrupt":true,"duration_ms":7}`,
			goType: &PostToolUseFailureHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput permissionRequest",
			input:  `{"session_id":"sess-2","transcript_path":"/tmp/transcript.jsonl","cwd":"/repo","effort":{"level":"high"},"hook_event_name":"PermissionRequest","tool_name":"Read","tool_input":{"file_path":"a.txt","offset":0},"permission_suggestions":[{"type":"addRules","rules":[{"toolName":"Read","ruleContent":"a.txt"}],"behavior":"allow","destination":"session"}]}`,
			goType: &PermissionRequestHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name: "HookInput permissionDenied",
			input: `{"session_id":"s","transcript_path":"t","cwd":"c",` +
				`"hook_event_name":"PermissionDenied","tool_name":"Bash",` +
				`"tool_input":{"command":"ls"},"tool_use_id":"u1","reason":"no"}`,
			goType: &PermissionDeniedHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
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
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput postToolBatch",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PostToolBatch","tool_calls":[{"tool_name":"Bash","tool_input":{"command":"ls"},"tool_use_id":"u1","tool_response":{"stdout":"x"}}]}`,
			goType: &PostToolBatchHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput messageDisplay with effort",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","effort":{"level":"high"},"hook_event_name":"MessageDisplay","turn_id":"turn-1","message_id":"msg-1","index":3,"final":true,"delta":"hello"}`,
			goType: &MessageDisplayHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput stop with background tasks and crons",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"Stop","stop_hook_active":true,"last_assistant_message":"done","background_tasks":[{"id":"b1","type":"shell","status":"running","description":"d"}],"session_crons":[{"id":"c1","schedule":"* * * * *","recurring":true,"prompt":"p"}]}`,
			goType: &StopHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookInput subagentStop reshape",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"SubagentStop","stop_hook_active":true,"agent_id":"a1","agent_transcript_path":"/tmp/a.jsonl","agent_type":"worker"}`,
			goType: &SubagentStopHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "HookSpecificOutput postToolUse updatedToolOutput",
			input:  `{"hookEventName":"PostToolUse","additionalContext":"ctx","updatedToolOutput":{"v":1}}`,
			goType: &HookSpecificOutputPostToolUse{},
			toProto: func(v any) any {
				got, err := HookSpecificOutputToProto(
					NewHookSpecificOutput(v.(HookSpecificOutput_Value)),
				)
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookSpecificOutputFromProto(v.(*pb.HookSpecificOutput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput Monitor via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"Monitor","tool_input":{"command":"tail -f log","description":"watch","timeout_ms":300000,"persistent":true},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput TaskUpdate via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"TaskUpdate","tool_input":{"taskId":"t1","status":"completed","addBlocks":["b1"]},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput TaskList empty via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"TaskList","tool_input":{},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput Agent alias via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"d","prompt":"p","subagent_type":"general"},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput Workflow via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"Workflow","tool_input":{"name":"my-wf","args":{"q":"x"}},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput TaskStop via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"TaskStop","tool_input":{"task_id":"t1"},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput TaskOutput via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"TaskOutput","tool_input":{"task_id":"t1","block":true,"timeout":30},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
				return mustProto(t, got, err)
			},
			fromProto: func(v any) any {
				got, err := HookInputFromProto(v.(*pb.HookInput))
				return mustProto(t, got, err)
			},
		},
		{
			name:   "ToolInput EnterWorktree via PreToolUse",
			input:  `{"session_id":"s","transcript_path":"t","cwd":"c","hook_event_name":"PreToolUse","tool_name":"EnterWorktree","tool_input":{"name":"wt","path":"/repo/wt"},"tool_use_id":"u1"}`,
			goType: &PreToolUseHookInput{},
			toProto: func(v any) any {
				got, err := HookInputToProto(NewHookInput(v.(HookInput_Value)))
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

// TestToolOutputJSONRoundTrip validates the JSON shape of tool output types
// (dispatched by tool name) survives an Unmarshal -> Marshal round trip. This
// directly exercises the rewritten output shapes (WebSearch union, TodoWrite,
// ExitPlanMode, ListMcpResources bare array) and the new Task/Monitor/Workflow
// outputs.
func TestToolOutputJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		input    string
	}{
		{
			name:     "WebSearch object-or-string union",
			toolName: ToolNameWebSearch,
			input:    `{"query":"go","results":[{"tool_use_id":"t1","content":[{"title":"Go","url":"https://go.dev"}]},"bare result"],"durationSeconds":2}`,
		},
		{
			name:     "TodoWrite slices",
			toolName: ToolNameTodoWrite,
			input:    `{"oldTodos":[{"content":"a","status":"completed","activeForm":"b"}],"newTodos":[{"content":"c","status":"in_progress","activeForm":"d"}]}`,
		},
		{
			name:     "ExitPlanMode with plan",
			toolName: ToolNameExitPlanMode,
			input:    `{"plan":"do the thing","isAgent":false}`,
		},
		{
			name:     "ExitPlanMode null plan",
			toolName: ToolNameExitPlanMode,
			input:    `{"plan":null,"isAgent":true,"requestId":"r1"}`,
		},
		{
			name:     "ListMcpResources bare array",
			toolName: ToolNameListMcpResources,
			input:    `[{"uri":"u","name":"n","mimeType":"text/plain","server":"srv"}]`,
		},
		{
			name:     "Monitor",
			toolName: ToolNameMonitor,
			input:    `{"taskId":"t","timeoutMs":5000,"persistent":true}`,
		},
		{
			name:     "Workflow",
			toolName: ToolNameWorkflow,
			input:    `{"status":"async_launched","taskId":"t","error":"syntax"}`,
		},
		{
			name:     "TaskCreate",
			toolName: ToolNameTaskCreate,
			input:    `{"task":{"id":"i","subject":"s"}}`,
		},
		{
			name:     "TaskUpdate",
			toolName: ToolNameTaskUpdate,
			input:    `{"success":true,"taskId":"t","updatedFields":["status"],"statusChange":{"from":"pending","to":"completed"}}`,
		},
		{
			name:     "TaskGet null",
			toolName: ToolNameTaskGet,
			input:    `{"task":null}`,
		},
		{
			name:     "TaskGet present",
			toolName: ToolNameTaskGet,
			input:    `{"task":{"id":"i","subject":"s","description":"d","status":"pending","blocks":[],"blockedBy":["x"]}}`,
		},
		{
			name:     "TaskList",
			toolName: ToolNameTaskList,
			input:    `{"tasks":[{"id":"i","subject":"s","status":"completed","blockedBy":[]}]}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v ToolOutputSchemas
			if err := v.UnmarshalForTool(tc.toolName, []byte(tc.input)); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			compareJSON(t, []byte(tc.input), got)
		})
	}
}

// TestSDKMessageJSONRoundTrip validates the JSON shape of the new and changed
// conversation-message types (which are schema-only: no proto conversion).
func TestSDKMessageJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		newFn func() any
		input string
	}{
		{
			name:  "SDKTaskUpdatedMessage",
			newFn: func() any { return &SDKTaskUpdatedMessage{} },
			input: `{"type":"system","subtype":"task_updated","task_id":"t","patch":{"status":"running","is_backgrounded":true},"uuid":"u","session_id":"s"}`,
		},
		{
			name:  "SDKCommandsChangedMessage",
			newFn: func() any { return &SDKCommandsChangedMessage{} },
			input: `{"type":"system","subtype":"commands_changed","commands":[],"uuid":"u","session_id":"s"}`,
		},
		{
			name:  "SDKPluginInstallMessage",
			newFn: func() any { return &SDKPluginInstallMessage{} },
			input: `{"type":"system","subtype":"plugin_install","status":"installed","name":"mp","uuid":"u","session_id":"s"}`,
		},
		{
			name:  "SDKPermissionDeniedMessage",
			newFn: func() any { return &SDKPermissionDeniedMessage{} },
			input: `{"type":"system","subtype":"permission_denied","tool_name":"Bash","tool_use_id":"tu","decision_reason_type":"rule","message":"denied","uuid":"u","session_id":"s"}`,
		},
		{
			name:  "SDKMessageOrigin peer variant",
			newFn: func() any { return &SDKMessageOrigin{} },
			input: `{"kind":"peer","from":"addr","name":"Alice"}`,
		},
		{
			name:  "SDKTaskProgressMessage subagentType and summary",
			newFn: func() any { return &SDKTaskProgressMessage{} },
			input: `{"type":"system","subtype":"task_progress","task_id":"t","description":"d","subagent_type":"general","usage":{"total_tokens":1,"tool_uses":2,"duration_ms":3},"summary":"sum","uuid":"u","session_id":"s"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.newFn()
			if err := json.Unmarshal([]byte(tc.input), v); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			compareJSON(t, []byte(tc.input), got)
		})
	}
}

// TestConfigTypesJSONRoundTrip validates JSON shapes for the config / other
// schema-only types changed or added in the reconciliation.
func TestConfigTypesJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		newFn func() any
		input string
	}{
		{
			name:  "SlashCommand aliases",
			newFn: func() any { return &SlashCommand{} },
			input: `{"name":"foo","description":"d","argumentHint":"h","aliases":["bar"]}`,
		},
		{
			name:  "McpSetServersResult",
			newFn: func() any { return &McpSetServersResult{} },
			input: `{"added":["a"],"removed":[],"errors":{"x":"boom"}}`,
		},
		{
			name:  "RewindFilesResult",
			newFn: func() any { return &RewindFilesResult{} },
			input: `{"canRewind":true,"filesChanged":["a.go"]}`,
		},
		{
			name:  "AgentDefinition new fields",
			newFn: func() any { return &AgentDefinition{} },
			input: `{"description":"d","prompt":"p","initialPrompt":"go","background":true,"memory":"project","effort":"high","permissionMode":"plan"}`,
		},
		{
			name:  "ThinkingConfigAdaptive display",
			newFn: func() any { return &ThinkingConfigAdaptive{} },
			input: `{"type":"adaptive","display":"summarized"}`,
		},
		{
			name:  "SandboxSettings failIfUnavailable",
			newFn: func() any { return &SandboxSettings{} },
			input: `{"enabled":true,"failIfUnavailable":false}`,
		},
		{
			name:  "Usage full",
			newFn: func() any { return &Usage{} },
			input: `{"input_tokens":1,"output_tokens":2,"cache_creation_input_tokens":null,"cache_read_input_tokens":null,"cache_creation":null,"server_tool_use":null,"service_tier":"standard","speed":"fast","inference_geo":null,"iterations":null}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := tc.newFn()
			if err := json.Unmarshal([]byte(tc.input), v); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			got, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			compareJSON(t, []byte(tc.input), got)
		})
	}
}

// TestSyncHookJSONOutputJSONRoundTrip covers the top-level
// SyncHookJSONOutput fields. terminalSequence is the reason it exists: the
// field was declared in MarshalJSON's raw struct but never assigned, and
// absent from UnmarshalJSON's altogether, so a value set on it silently
// vanished in both directions.
//
// The proto schema carries terminal_sequence too, and
// TestJSONProtoRoundTrip's sync case pins that leg. This test stays separate
// because it drives MarshalJSON/UnmarshalJSON with no conversion in between,
// so a regression in the marshalers alone still fails here.
func TestSyncHookJSONOutputJSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			// OSC 9 desktop notification, the field's documented use.
			name:  "terminalSequence alone",
			input: `{"terminalSequence":"\u001b]9;done\u0007"}`,
		},
		{
			name: "every top-level field",
			input: `{"continue":true,"suppressOutput":false,"stopReason":"x",` +
				`"decision":"block","systemMessage":"m","terminalSequence":"\u0007",` +
				`"reason":"r",` +
				`"hookSpecificOutput":{"hookEventName":"Notification","additionalContext":"ctx"}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var v SyncHookJSONOutput
			if err := json.Unmarshal([]byte(tc.input), &v); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if v.TerminalSequence == nil {
				t.Fatal("terminalSequence dropped on unmarshal")
			}
			got, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			compareJSON(t, []byte(tc.input), got)
		})
	}
}
