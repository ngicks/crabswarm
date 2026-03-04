package models

import (
	"encoding/json"
	"fmt"
	"reflect"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// As per https://platform.claude.com/docs/en/agent-sdk/typescript#hook-input
/*
type HookInput =
  | PreToolUseHookInput
  | PostToolUseHookInput
  | PostToolUseFailureHookInput
  | NotificationHookInput
  | UserPromptSubmitHookInput
  | SessionStartHookInput
  | SessionEndHookInput
  | StopHookInput
  | SubagentStartHookInput
  | SubagentStopHookInput
  | PreCompactHookInput
  | PermissionRequestHookInput
  | SetupHookInput
  | TeammateIdleHookInput
  | TaskCompletedHookInput
  | ConfigChangeHookInput
  | WorktreeCreateHookInput
  | WorktreeRemoveHookInput;
*/

type HookInput interface {
	hookInput()
}

func (PreToolUseHookInput) hookInput()         {}
func (PostToolUseHookInput) hookInput()        {}
func (PostToolUseFailureHookInput) hookInput() {}
func (NotificationHookInput) hookInput()       {}
func (UserPromptSubmitHookInput) hookInput()   {}
func (SessionStartHookInput) hookInput()       {}
func (SessionEndHookInput) hookInput()         {}
func (StopHookInput) hookInput()               {}
func (SubagentStartHookInput) hookInput()      {}
func (SubagentStopHookInput) hookInput()       {}
func (PreCompactHookInput) hookInput()         {}
func (PermissionRequestHookInput) hookInput()  {}
func (SetupHookInput) hookInput()              {}
func (TeammateIdleHookInput) hookInput()       {}
func (TaskCompletedHookInput) hookInput()      {}
func (ConfigChangeHookInput) hookInput()       {}
func (WorktreeCreateHookInput) hookInput()     {}
func (WorktreeRemoveHookInput) hookInput()     {}

// BaseHookInput is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#base-hook-input
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitempty"`
	// Not documented but actually there's
	ToolUseID string `json:"tool_use_id,omitempty"`
}

type hookInputBase struct {
	HookEventName string `json:"hook_event_name"`
}

type PreToolUseHookInput struct{
  BaseHookInput
  hook_event_name: "PreToolUse";
  tool_name: string;
  tool_input: unknown;
  tool_use_id: string;
}

type PostToolUseHookInput struct{
BaseHookInput
  hook_event_name: "PostToolUse";
  tool_name: string;
  tool_input: unknown;
  tool_response: unknown;
  tool_use_id: string;
}
type PostToolUseFailureHookInput struct{
BaseHookInput
 hook_event_name: "PostToolUseFailure";
  tool_name: string;
  tool_input: unknown;
  tool_use_id: string;
  error: string;
  is_interrupt?: boolean;
}
type NotificationHookInput struct{
BaseHookInput
  hook_event_name: "Notification";
  message: string;
  title?: string;
  notification_type: string;
}
type UserPromptSubmitHookInput struct{
BaseHookInput
  hook_event_name: "UserPromptSubmit";
  prompt: string;
}
type SessionStartHookInput struct{
BaseHookInput
 hook_event_name: "SessionStart";
  source: "startup" | "resume" | "clear" | "compact";
  agent_type?: string;
  model?: string;
}
type SessionEndHookInput struct{
BaseHookInput
  hook_event_name: "SessionEnd";
  reason: ExitReason; // String from EXIT_REASONS array
}
type StopHookInput struct{
BaseHookInput
  hook_event_name: "Stop";
  stop_hook_active: boolean;
  last_assistant_message?: string;
}
type SubagentStartHookInput struct{
BaseHookInput
  hook_event_name: "SubagentStart";
  agent_id: string;
  agent_type: string;
}
type SubagentStopHookInput struct{
BaseHookInput
  hook_event_name: "SubagentStop";
  stop_hook_active: boolean;
  agent_id: string;
  agent_transcript_path: string;
  agent_type: string;
  last_assistant_message?: string;
}
type PreCompactHookInput struct{
BaseHookInput
  hook_event_name: "PreCompact";
  trigger: "manual" | "auto";
  custom_instructions: string | null;
}
type PermissionRequestHookInput struct{
BaseHookInput
  hook_event_name: "PermissionRequest";
  tool_name: string;
  tool_input: unknown;
  permission_suggestions?: PermissionUpdate[];
}

type PermissionUpdate struct{}

type SetupHookInput struct{
BaseHookInput
  hook_event_name: "Setup";
  trigger: "init" | "maintenance";
}
type TeammateIdleHookInput struct{
BaseHookInput
 hook_event_name: "TeammateIdle";
  teammate_name: string;
  team_name: string;
}
type TaskCompletedHookInput struct{
BaseHookInput
  hook_event_name: "TaskCompleted";
  task_id: string;
  task_subject: string;
  task_description?: string;
  teammate_name?: string;
  team_name?: string;
}
type ConfigChangeHookInput struct{
BaseHookInput
  hook_event_name: "ConfigChange";
  source:
    | "user_settings"
    | "project_settings"
    | "local_settings"
    | "policy_settings"
    | "skills";
  file_path?: string;
}
type WorktreeCreateHookInput struct{
BaseHookInput
  hook_event_name: "WorktreeCreate";
  name: string;
}
type WorktreeRemoveHookInput struct{
BaseHookInput
  hook_event_name: "WorktreeRemove";
  worktree_path: string;
}
