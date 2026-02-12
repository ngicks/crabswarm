package tool_iov1

import (
	"encoding"
	"fmt"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// enumTestCase describes a single enum type to test.
type enumTestCase struct {
	name string
	// all non-UNSPECIFIED values
	values []enumValue
	// marshal function wrapping MarshalText
	marshal func(int32) ([]byte, error)
	// unmarshal function wrapping UnmarshalText
	unmarshal func([]byte) (int32, error)
	// proto enum descriptor for completeness check
	descriptor func() protoreflect.EnumDescriptor
}

type enumValue struct {
	num  int32
	text string
}

var allEnumTests = []enumTestCase{
	{
		name: "PermissionMode",
		values: []enumValue{
			{int32(PermissionMode_PERMISSION_MODE_DEFAULT), "default"},
			{int32(PermissionMode_PERMISSION_MODE_ACCEPT_EDITS), "acceptEdits"},
			{int32(PermissionMode_PERMISSION_MODE_BYPASS_PERMISSIONS), "bypassPermissions"},
			{int32(PermissionMode_PERMISSION_MODE_PLAN), "plan"},
		},
		marshal: func(v int32) ([]byte, error) { return PermissionMode(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x PermissionMode
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return PermissionMode(0).Descriptor() },
	},
	{
		name: "PermissionBehavior",
		values: []enumValue{
			{int32(PermissionBehavior_PERMISSION_BEHAVIOR_ALLOW), "allow"},
			{int32(PermissionBehavior_PERMISSION_BEHAVIOR_DENY), "deny"},
			{int32(PermissionBehavior_PERMISSION_BEHAVIOR_ASK), "ask"},
		},
		marshal: func(v int32) ([]byte, error) { return PermissionBehavior(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x PermissionBehavior
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return PermissionBehavior(0).Descriptor() },
	},
	{
		name: "PermissionUpdateDestination",
		values: []enumValue{
			{int32(PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_USER_SETTINGS), "userSettings"},
			{int32(PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_PROJECT_SETTINGS), "projectSettings"},
			{int32(PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_LOCAL_SETTINGS), "localSettings"},
			{int32(PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_SESSION), "session"},
		},
		marshal: func(v int32) ([]byte, error) { return PermissionUpdateDestination(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x PermissionUpdateDestination
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return PermissionUpdateDestination(0).Descriptor() },
	},
	{
		name: "SettingSource",
		values: []enumValue{
			{int32(SettingSource_SETTING_SOURCE_USER), "user"},
			{int32(SettingSource_SETTING_SOURCE_PROJECT), "project"},
			{int32(SettingSource_SETTING_SOURCE_LOCAL), "local"},
		},
		marshal: func(v int32) ([]byte, error) { return SettingSource(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x SettingSource
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return SettingSource(0).Descriptor() },
	},
	{
		name: "AgentModel",
		values: []enumValue{
			{int32(AgentModel_AGENT_MODEL_SONNET), "sonnet"},
			{int32(AgentModel_AGENT_MODEL_OPUS), "opus"},
			{int32(AgentModel_AGENT_MODEL_HAIKU), "haiku"},
			{int32(AgentModel_AGENT_MODEL_INHERIT), "inherit"},
		},
		marshal: func(v int32) ([]byte, error) { return AgentModel(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x AgentModel
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return AgentModel(0).Descriptor() },
	},
	{
		name: "HookEvent",
		values: []enumValue{
			{int32(HookEvent_HOOK_EVENT_PRE_TOOL_USE), "PreToolUse"},
			{int32(HookEvent_HOOK_EVENT_POST_TOOL_USE), "PostToolUse"},
			{int32(HookEvent_HOOK_EVENT_POST_TOOL_USE_FAILURE), "PostToolUseFailure"},
			{int32(HookEvent_HOOK_EVENT_NOTIFICATION), "Notification"},
			{int32(HookEvent_HOOK_EVENT_USER_PROMPT_SUBMIT), "UserPromptSubmit"},
			{int32(HookEvent_HOOK_EVENT_SESSION_START), "SessionStart"},
			{int32(HookEvent_HOOK_EVENT_SESSION_END), "SessionEnd"},
			{int32(HookEvent_HOOK_EVENT_STOP), "Stop"},
			{int32(HookEvent_HOOK_EVENT_SUBAGENT_START), "SubagentStart"},
			{int32(HookEvent_HOOK_EVENT_SUBAGENT_STOP), "SubagentStop"},
			{int32(HookEvent_HOOK_EVENT_PRE_COMPACT), "PreCompact"},
			{int32(HookEvent_HOOK_EVENT_PERMISSION_REQUEST), "PermissionRequest"},
		},
		marshal: func(v int32) ([]byte, error) { return HookEvent(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x HookEvent
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return HookEvent(0).Descriptor() },
	},
	{
		name: "SessionStartSource",
		values: []enumValue{
			{int32(SessionStartSource_SESSION_START_SOURCE_STARTUP), "startup"},
			{int32(SessionStartSource_SESSION_START_SOURCE_RESUME), "resume"},
			{int32(SessionStartSource_SESSION_START_SOURCE_CLEAR), "clear"},
			{int32(SessionStartSource_SESSION_START_SOURCE_COMPACT), "compact"},
		},
		marshal: func(v int32) ([]byte, error) { return SessionStartSource(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x SessionStartSource
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return SessionStartSource(0).Descriptor() },
	},
	{
		name: "PreCompactTrigger",
		values: []enumValue{
			{int32(PreCompactTrigger_PRE_COMPACT_TRIGGER_MANUAL), "manual"},
			{int32(PreCompactTrigger_PRE_COMPACT_TRIGGER_AUTO), "auto"},
		},
		marshal: func(v int32) ([]byte, error) { return PreCompactTrigger(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x PreCompactTrigger
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return PreCompactTrigger(0).Descriptor() },
	},
	{
		name: "SDKResultSubtype",
		values: []enumValue{
			{int32(SDKResultSubtype_SDK_RESULT_SUBTYPE_SUCCESS), "success"},
			{int32(SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_TURNS), "error_max_turns"},
			{int32(SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_DURING_EXECUTION), "error_during_execution"},
			{int32(SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_BUDGET_USD), "error_max_budget_usd"},
			{int32(SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_STRUCTURED_OUTPUT_RETRIES), "error_max_structured_output_retries"},
		},
		marshal: func(v int32) ([]byte, error) { return SDKResultSubtype(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x SDKResultSubtype
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return SDKResultSubtype(0).Descriptor() },
	},
	{
		name: "ApiKeySource",
		values: []enumValue{
			{int32(ApiKeySource_API_KEY_SOURCE_USER), "user"},
			{int32(ApiKeySource_API_KEY_SOURCE_PROJECT), "project"},
			{int32(ApiKeySource_API_KEY_SOURCE_ORG), "org"},
			{int32(ApiKeySource_API_KEY_SOURCE_TEMPORARY), "temporary"},
		},
		marshal: func(v int32) ([]byte, error) { return ApiKeySource(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x ApiKeySource
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return ApiKeySource(0).Descriptor() },
	},
	{
		name: "SdkBeta",
		values: []enumValue{
			{int32(SdkBeta_SDK_BETA_CONTEXT_1M_2025_08_07), "context-1m-2025-08-07"},
		},
		marshal: func(v int32) ([]byte, error) { return SdkBeta(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x SdkBeta
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return SdkBeta(0).Descriptor() },
	},
	{
		name: "ConfigScope",
		values: []enumValue{
			{int32(ConfigScope_CONFIG_SCOPE_LOCAL), "local"},
			{int32(ConfigScope_CONFIG_SCOPE_USER), "user"},
			{int32(ConfigScope_CONFIG_SCOPE_PROJECT), "project"},
		},
		marshal: func(v int32) ([]byte, error) { return ConfigScope(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x ConfigScope
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return ConfigScope(0).Descriptor() },
	},
	{
		name: "McpServerConnectionStatus",
		values: []enumValue{
			{int32(McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_CONNECTED), "connected"},
			{int32(McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_FAILED), "failed"},
			{int32(McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_NEEDS_AUTH), "needs-auth"},
			{int32(McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_PENDING), "pending"},
		},
		marshal: func(v int32) ([]byte, error) { return McpServerConnectionStatus(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x McpServerConnectionStatus
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return McpServerConnectionStatus(0).Descriptor() },
	},
	{
		name: "CallToolResultContentType",
		values: []enumValue{
			{int32(CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_TEXT), "text"},
			{int32(CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_IMAGE), "image"},
			{int32(CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_RESOURCE), "resource"},
		},
		marshal: func(v int32) ([]byte, error) { return CallToolResultContentType(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x CallToolResultContentType
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return CallToolResultContentType(0).Descriptor() },
	},
	{
		name: "GrepOutputMode",
		values: []enumValue{
			{int32(GrepOutputMode_GREP_OUTPUT_MODE_CONTENT), "content"},
			{int32(GrepOutputMode_GREP_OUTPUT_MODE_FILES_WITH_MATCHES), "files_with_matches"},
			{int32(GrepOutputMode_GREP_OUTPUT_MODE_COUNT), "count"},
		},
		marshal: func(v int32) ([]byte, error) { return GrepOutputMode(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x GrepOutputMode
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return GrepOutputMode(0).Descriptor() },
	},
	{
		name: "NotebookCellType",
		values: []enumValue{
			{int32(NotebookCellType_NOTEBOOK_CELL_TYPE_CODE), "code"},
			{int32(NotebookCellType_NOTEBOOK_CELL_TYPE_MARKDOWN), "markdown"},
		},
		marshal: func(v int32) ([]byte, error) { return NotebookCellType(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x NotebookCellType
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return NotebookCellType(0).Descriptor() },
	},
	{
		name: "NotebookEditMode",
		values: []enumValue{
			{int32(NotebookEditMode_NOTEBOOK_EDIT_MODE_REPLACE), "replace"},
			{int32(NotebookEditMode_NOTEBOOK_EDIT_MODE_INSERT), "insert"},
			{int32(NotebookEditMode_NOTEBOOK_EDIT_MODE_DELETE), "delete"},
		},
		marshal: func(v int32) ([]byte, error) { return NotebookEditMode(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x NotebookEditMode
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return NotebookEditMode(0).Descriptor() },
	},
	{
		name: "TodoStatus",
		values: []enumValue{
			{int32(TodoStatus_TODO_STATUS_PENDING), "pending"},
			{int32(TodoStatus_TODO_STATUS_IN_PROGRESS), "in_progress"},
			{int32(TodoStatus_TODO_STATUS_COMPLETED), "completed"},
		},
		marshal: func(v int32) ([]byte, error) { return TodoStatus(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x TodoStatus
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return TodoStatus(0).Descriptor() },
	},
	{
		name: "BashOutputStatus",
		values: []enumValue{
			{int32(BashOutputStatus_BASH_OUTPUT_STATUS_RUNNING), "running"},
			{int32(BashOutputStatus_BASH_OUTPUT_STATUS_COMPLETED), "completed"},
			{int32(BashOutputStatus_BASH_OUTPUT_STATUS_FAILED), "failed"},
		},
		marshal: func(v int32) ([]byte, error) { return BashOutputStatus(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x BashOutputStatus
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return BashOutputStatus(0).Descriptor() },
	},
	{
		name: "NotebookEditType",
		values: []enumValue{
			{int32(NotebookEditType_NOTEBOOK_EDIT_TYPE_REPLACED), "replaced"},
			{int32(NotebookEditType_NOTEBOOK_EDIT_TYPE_INSERTED), "inserted"},
			{int32(NotebookEditType_NOTEBOOK_EDIT_TYPE_DELETED), "deleted"},
		},
		marshal: func(v int32) ([]byte, error) { return NotebookEditType(v).MarshalText() },
		unmarshal: func(b []byte) (int32, error) {
			var x NotebookEditType
			err := x.UnmarshalText(b)
			return int32(x), err
		},
		descriptor: func() protoreflect.EnumDescriptor { return NotebookEditType(0).Descriptor() },
	},
}

// Compile-time interface checks.
var (
	_ encoding.TextMarshaler   = PermissionMode(0)
	_ encoding.TextUnmarshaler = (*PermissionMode)(nil)
	_ encoding.TextMarshaler   = PermissionBehavior(0)
	_ encoding.TextUnmarshaler = (*PermissionBehavior)(nil)
	_ encoding.TextMarshaler   = PermissionUpdateDestination(0)
	_ encoding.TextUnmarshaler = (*PermissionUpdateDestination)(nil)
	_ encoding.TextMarshaler   = SettingSource(0)
	_ encoding.TextUnmarshaler = (*SettingSource)(nil)
	_ encoding.TextMarshaler   = AgentModel(0)
	_ encoding.TextUnmarshaler = (*AgentModel)(nil)
	_ encoding.TextMarshaler   = HookEvent(0)
	_ encoding.TextUnmarshaler = (*HookEvent)(nil)
	_ encoding.TextMarshaler   = SessionStartSource(0)
	_ encoding.TextUnmarshaler = (*SessionStartSource)(nil)
	_ encoding.TextMarshaler   = PreCompactTrigger(0)
	_ encoding.TextUnmarshaler = (*PreCompactTrigger)(nil)
	_ encoding.TextMarshaler   = SDKResultSubtype(0)
	_ encoding.TextUnmarshaler = (*SDKResultSubtype)(nil)
	_ encoding.TextMarshaler   = ApiKeySource(0)
	_ encoding.TextUnmarshaler = (*ApiKeySource)(nil)
	_ encoding.TextMarshaler   = SdkBeta(0)
	_ encoding.TextUnmarshaler = (*SdkBeta)(nil)
	_ encoding.TextMarshaler   = ConfigScope(0)
	_ encoding.TextUnmarshaler = (*ConfigScope)(nil)
	_ encoding.TextMarshaler   = McpServerConnectionStatus(0)
	_ encoding.TextUnmarshaler = (*McpServerConnectionStatus)(nil)
	_ encoding.TextMarshaler   = CallToolResultContentType(0)
	_ encoding.TextUnmarshaler = (*CallToolResultContentType)(nil)
	_ encoding.TextMarshaler   = GrepOutputMode(0)
	_ encoding.TextUnmarshaler = (*GrepOutputMode)(nil)
	_ encoding.TextMarshaler   = NotebookCellType(0)
	_ encoding.TextUnmarshaler = (*NotebookCellType)(nil)
	_ encoding.TextMarshaler   = NotebookEditMode(0)
	_ encoding.TextUnmarshaler = (*NotebookEditMode)(nil)
	_ encoding.TextMarshaler   = TodoStatus(0)
	_ encoding.TextUnmarshaler = (*TodoStatus)(nil)
	_ encoding.TextMarshaler   = BashOutputStatus(0)
	_ encoding.TextUnmarshaler = (*BashOutputStatus)(nil)
	_ encoding.TextMarshaler   = NotebookEditType(0)
	_ encoding.TextUnmarshaler = (*NotebookEditType)(nil)
)

func TestEnumTextRoundTrip(t *testing.T) {
	for _, tc := range allEnumTests {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range tc.values {
				t.Run(v.text, func(t *testing.T) {
					// Marshal
					b, err := tc.marshal(v.num)
					if err != nil {
						t.Fatalf("MarshalText(%d) error: %v", v.num, err)
					}
					if string(b) != v.text {
						t.Fatalf("MarshalText(%d) = %q, want %q", v.num, string(b), v.text)
					}
					// Unmarshal
					got, err := tc.unmarshal(b)
					if err != nil {
						t.Fatalf("UnmarshalText(%q) error: %v", v.text, err)
					}
					if got != v.num {
						t.Fatalf("UnmarshalText(%q) = %d, want %d", v.text, got, v.num)
					}
				})
			}
		})
	}
}

func TestEnumTextUnspecified(t *testing.T) {
	for _, tc := range allEnumTests {
		t.Run(tc.name, func(t *testing.T) {
			// UNSPECIFIED (0) marshals to nil/empty
			b, err := tc.marshal(0)
			if err != nil {
				t.Fatalf("MarshalText(0) error: %v", err)
			}
			if len(b) != 0 {
				t.Fatalf("MarshalText(0) = %q, want empty", string(b))
			}
			// Empty bytes unmarshal to 0
			got, err := tc.unmarshal(nil)
			if err != nil {
				t.Fatalf("UnmarshalText(nil) error: %v", err)
			}
			if got != 0 {
				t.Fatalf("UnmarshalText(nil) = %d, want 0", got)
			}
			got, err = tc.unmarshal([]byte(""))
			if err != nil {
				t.Fatalf("UnmarshalText(\"\") error: %v", err)
			}
			if got != 0 {
				t.Fatalf("UnmarshalText(\"\") = %d, want 0", got)
			}
		})
	}
}

func TestEnumTextUnknown(t *testing.T) {
	for _, tc := range allEnumTests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.unmarshal([]byte("__BOGUS__"))
			if err == nil {
				t.Fatal("UnmarshalText(\"__BOGUS__\") expected error, got nil")
			}
		})
	}
}

func TestEnumTextCompleteness(t *testing.T) {
	for _, tc := range allEnumTests {
		t.Run(tc.name, func(t *testing.T) {
			desc := tc.descriptor()
			vals := desc.Values()
			covered := make(map[int32]bool)
			for _, v := range tc.values {
				covered[v.num] = true
			}
			for i := 0; i < vals.Len(); i++ {
				vd := vals.Get(i)
				num := int32(vd.Number())
				if num == 0 {
					continue // skip UNSPECIFIED
				}
				if !covered[num] {
					t.Errorf("enum value %s (%d) not covered by text marshaling", vd.Name(), num)
				}
				b, err := tc.marshal(num)
				if err != nil {
					t.Errorf("MarshalText(%d) error: %v", num, err)
					continue
				}
				if len(b) == 0 {
					t.Errorf("MarshalText(%d) returned empty text for non-UNSPECIFIED value %s", num, vd.Name())
				}
			}
			// Verify test covers exactly the right number of values
			want := vals.Len() - 1 // minus UNSPECIFIED
			if len(tc.values) != want {
				t.Errorf("test covers %d values, but proto descriptor has %d non-UNSPECIFIED values",
					len(tc.values), want)
			}
		})
	}
}

func TestEnumTextDescriptorCount(t *testing.T) {
	// Verify we test all 20 enum types.
	if got := len(allEnumTests); got != 20 {
		names := make([]string, len(allEnumTests))
		for i, tc := range allEnumTests {
			names[i] = tc.name
		}
		t.Errorf("expected 20 enum types, got %d: %v", got, names)
	}
}

func TestEnumTextEmptySlice(t *testing.T) {
	// Ensure empty byte slice (not nil) also unmarshals to UNSPECIFIED.
	for _, tc := range allEnumTests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.unmarshal([]byte{})
			if err != nil {
				t.Fatalf("UnmarshalText([]byte{}) error: %v", err)
			}
			if got != 0 {
				t.Fatalf("UnmarshalText([]byte{}) = %d, want 0", got)
			}
		})
	}
}

// Suppress unused import warning for fmt.
var _ = fmt.Sprintf
