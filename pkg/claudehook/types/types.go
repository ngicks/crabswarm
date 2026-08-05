// Package types contains handwritten Claude Agent SDK shapes derived from:
// https://code.claude.com/docs/en/agent-sdk/typescript
package types

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	pb "github.com/ngicks/crabswarm/api/gen/proto/go/sdktypes/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

// UUID is the SDK UUID alias.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type UUID string

// BetaMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkassistantmessage
type BetaMessage json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m BetaMessage) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *BetaMessage) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

// MessageParam is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkusermessage
type MessageParam json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m MessageParam) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *MessageParam) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

// BetaRawMessageStreamEvent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkpartialassistantmessage
type BetaRawMessageStreamEvent json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m BetaRawMessageStreamEvent) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *BetaRawMessageStreamEvent) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func marshalRawPayload[T ~[]byte](m T) ([]byte, error) {
	if len(m) == 0 {
		return []byte("null"), nil
	}
	return m, nil
}

func unmarshalRawPayload(dst *json.RawMessage, data []byte) error {
	*dst = append((*dst)[:0], data...)
	return nil
}

// UnknownUnion preserves unsupported union members.
//
// UnknownUnion is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type UnknownUnion struct {
	Discriminator string          `json:"-"`
	Raw           json.RawMessage `json:"-"`
}

// PermissionMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionmode
type PermissionMode string

const (
	PermissionModeDefault           PermissionMode = "default"
	PermissionModeAcceptEdits       PermissionMode = "acceptEdits"
	PermissionModeBypassPermissions PermissionMode = "bypassPermissions"
	PermissionModePlan              PermissionMode = "plan"
	PermissionModeDontAsk           PermissionMode = "dontAsk"
	PermissionModeAuto              PermissionMode = "auto"
)

// PermissionBehavior is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionbehavior
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdatedestination
type PermissionUpdateDestination string

const (
	PermissionUpdateDestinationUserSettings    PermissionUpdateDestination = "userSettings"
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings"
	PermissionUpdateDestinationLocalSettings   PermissionUpdateDestination = "localSettings"
	PermissionUpdateDestinationSession         PermissionUpdateDestination = "session"
	PermissionUpdateDestinationCliArg          PermissionUpdateDestination = "cliArg"
)

// SettingSource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#settingsource
type SettingSource string

const (
	SettingSourceUser    SettingSource = "user"
	SettingSourceProject SettingSource = "project"
	SettingSourceLocal   SettingSource = "local"
)

// ApiKeySource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#apikeysource
type ApiKeySource string

const (
	ApiKeySourceUser      ApiKeySource = "user"
	ApiKeySourceProject   ApiKeySource = "project"
	ApiKeySourceOrg       ApiKeySource = "org"
	ApiKeySourceTemporary ApiKeySource = "temporary"
	ApiKeySourceOauth     ApiKeySource = "oauth"
	// ApiKeySourceNone sits outside the documented union: the runtime emits
	// it on the init message when no API key is in use, for example when the
	// session authenticates with an OAuth token.
	ApiKeySourceNone ApiKeySource = "none"
)

// SdkBeta is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkbeta
type SdkBeta string

const (
	// Deprecated: retired as of April 30, 2026. Passing it with Claude
	// Sonnet 4.5 or Sonnet 4 has no effect, and requests over the standard
	// 200k-token window return an error.
	SdkBetaContext1M20250807 SdkBeta = "context-1m-2025-08-07"
)

// Effort is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type Effort string

const (
	EffortLow    Effort = "low"
	EffortMedium Effort = "medium"
	EffortHigh   Effort = "high"
	EffortXHigh  Effort = "xhigh"
	EffortMax    Effort = "max"
)

// OptionsExecutable is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type OptionsExecutable string

const (
	OptionsExecutableBun  OptionsExecutable = "bun"
	OptionsExecutableDeno OptionsExecutable = "deno"
	OptionsExecutableNode OptionsExecutable = "node"
)

// AgentModel is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent
type AgentModel string

const (
	AgentModelSonnet AgentModel = "sonnet"
	AgentModelOpus   AgentModel = "opus"
	AgentModelHaiku  AgentModel = "haiku"
	AgentModelFable  AgentModel = "fable"
)

// AgentMode is a handwritten Claude Agent SDK type.
//
// Deprecated: the mode field on AgentInput is documented as ignored.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent
type AgentMode string

const (
	AgentModeAcceptEdits       AgentMode = "acceptEdits"
	AgentModeAuto              AgentMode = "auto"
	AgentModeBypassPermissions AgentMode = "bypassPermissions"
	AgentModeDefault           AgentMode = "default"
	AgentModeDontAsk           AgentMode = "dontAsk"
	AgentModePlan              AgentMode = "plan"
)

// AgentIsolation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent
type AgentIsolation string

const (
	AgentIsolationWorktree AgentIsolation = "worktree"
	AgentIsolationRemote   AgentIsolation = "remote"
)

// ToolPreset is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ToolPreset string

const (
	ToolPresetClaudeCode ToolPreset = "claude_code"
)

// SystemPromptPresetName is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPromptPresetName string

const (
	SystemPromptPresetClaudeCode SystemPromptPresetName = "claude_code"
)

// ConfigScope is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#configscope
type ConfigScope string

const (
	ConfigScopeLocal   ConfigScope = "local"
	ConfigScopeUser    ConfigScope = "user"
	ConfigScopeProject ConfigScope = "project"
)

// HookEvent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookevent
type HookEvent string

const (
	HookEventPreToolUse          HookEvent = "PreToolUse"
	HookEventPostToolUse         HookEvent = "PostToolUse"
	HookEventPostToolUseFailure  HookEvent = "PostToolUseFailure"
	HookEventPostToolBatch       HookEvent = "PostToolBatch"
	HookEventNotification        HookEvent = "Notification"
	HookEventUserPromptSubmit    HookEvent = "UserPromptSubmit"
	HookEventUserPromptExpansion HookEvent = "UserPromptExpansion"
	HookEventSessionStart        HookEvent = "SessionStart"
	HookEventSessionEnd          HookEvent = "SessionEnd"
	HookEventStop                HookEvent = "Stop"
	HookEventStopFailure         HookEvent = "StopFailure"
	HookEventSubagentStart       HookEvent = "SubagentStart"
	HookEventSubagentStop        HookEvent = "SubagentStop"
	HookEventPreCompact          HookEvent = "PreCompact"
	HookEventPostCompact         HookEvent = "PostCompact"
	HookEventPermissionRequest   HookEvent = "PermissionRequest"
	HookEventPermissionDenied    HookEvent = "PermissionDenied"
	HookEventSetup               HookEvent = "Setup"
	HookEventTeammateIdle        HookEvent = "TeammateIdle"
	HookEventTaskCreated         HookEvent = "TaskCreated"
	HookEventTaskCompleted       HookEvent = "TaskCompleted"
	HookEventElicitation         HookEvent = "Elicitation"
	HookEventElicitationResult   HookEvent = "ElicitationResult"
	HookEventConfigChange        HookEvent = "ConfigChange"
	HookEventDirectoryAdded      HookEvent = "DirectoryAdded"
	HookEventWorktreeCreate      HookEvent = "WorktreeCreate"
	HookEventWorktreeRemove      HookEvent = "WorktreeRemove"
	HookEventInstructionsLoaded  HookEvent = "InstructionsLoaded"
	HookEventCwdChanged          HookEvent = "CwdChanged"
	HookEventFileChanged         HookEvent = "FileChanged"
	HookEventMessageDisplay      HookEvent = "MessageDisplay"
)

// SessionStartSource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sessionstarthookinput
type SessionStartSource string

const (
	SessionStartSourceStartup SessionStartSource = "startup"
	SessionStartSourceResume  SessionStartSource = "resume"
	SessionStartSourceClear   SessionStartSource = "clear"
	SessionStartSourceCompact SessionStartSource = "compact"
	SessionStartSourceFork    SessionStartSource = "fork"
)

// UserPromptExpansionType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#userpromptexpansionhookinput
type UserPromptExpansionType string

const (
	UserPromptExpansionTypeSlashCommand UserPromptExpansionType = "slash_command"
	UserPromptExpansionTypeMcpPrompt    UserPromptExpansionType = "mcp_prompt"
)

// ExitReason is a handwritten Claude Agent SDK type. The docs reference the
// named TypeScript type ExitReason ("String from EXIT_REASONS array") without
// defining its literal set, so no constants are declared.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sessionendhookinput
type ExitReason string

// PreCompactTrigger is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#precompacthookinput
type PreCompactTrigger string

const (
	PreCompactTriggerManual PreCompactTrigger = "manual"
	PreCompactTriggerAuto   PreCompactTrigger = "auto"
)

// PostCompactTrigger is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#postcompacthookinput
type PostCompactTrigger string

const (
	PostCompactTriggerManual PostCompactTrigger = "manual"
	PostCompactTriggerAuto   PostCompactTrigger = "auto"
)

// CompactMetadataTrigger is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcompactboundarymessage
type CompactMetadataTrigger string

const (
	CompactMetadataTriggerManual CompactMetadataTrigger = "manual"
	CompactMetadataTriggerAuto   CompactMetadataTrigger = "auto"
)

// ElicitationMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#elicitationhookinput
type ElicitationMode string

const (
	ElicitationModeForm ElicitationMode = "form"
	ElicitationModeURL  ElicitationMode = "url"
)

// ElicitationResultMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#elicitationresulthookinput
type ElicitationResultMode string

const (
	ElicitationResultModeForm ElicitationResultMode = "form"
	ElicitationResultModeURL  ElicitationResultMode = "url"
)

// ElicitationResultAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#elicitationresulthookinput
type ElicitationResultAction string

const (
	ElicitationResultActionAccept  ElicitationResultAction = "accept"
	ElicitationResultActionDecline ElicitationResultAction = "decline"
	ElicitationResultActionCancel  ElicitationResultAction = "cancel"
)

// InstructionsMemoryType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#instructionsloadedhookinput
type InstructionsMemoryType string

const (
	InstructionsMemoryTypeUser    InstructionsMemoryType = "User"
	InstructionsMemoryTypeProject InstructionsMemoryType = "Project"
	InstructionsMemoryTypeLocal   InstructionsMemoryType = "Local"
	InstructionsMemoryTypeManaged InstructionsMemoryType = "Managed"
)

// InstructionsLoadReason is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#instructionsloadedhookinput
type InstructionsLoadReason string

const (
	InstructionsLoadReasonSessionStart    InstructionsLoadReason = "session_start"
	InstructionsLoadReasonNestedTraversal InstructionsLoadReason = "nested_traversal"
	InstructionsLoadReasonPathGlobMatch   InstructionsLoadReason = "path_glob_match"
	InstructionsLoadReasonInclude         InstructionsLoadReason = "include"
	InstructionsLoadReasonCompact         InstructionsLoadReason = "compact"
)

// DirectoryAddedSource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#directoryaddedhookinput
type DirectoryAddedSource string

const (
	DirectoryAddedSourceSlashCommand     DirectoryAddedSource = "slash_command"
	DirectoryAddedSourceRegisterRepoRoot DirectoryAddedSource = "register_repo_root"
)

// FileChangedEvent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#filechangedhookinput
type FileChangedEvent string

const (
	FileChangedEventChange FileChangedEvent = "change"
	FileChangedEventAdd    FileChangedEvent = "add"
	FileChangedEventUnlink FileChangedEvent = "unlink"
)

// SetupTrigger is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#setuphookinput
type SetupTrigger string

const (
	SetupTriggerInit        SetupTrigger = "init"
	SetupTriggerMaintenance SetupTrigger = "maintenance"
)

// ConfigChangeSource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#configchangehookinput
type ConfigChangeSource string

const (
	ConfigChangeSourceUserSettings    ConfigChangeSource = "user_settings"
	ConfigChangeSourceProjectSettings ConfigChangeSource = "project_settings"
	ConfigChangeSourceLocalSettings   ConfigChangeSource = "local_settings"
	ConfigChangeSourcePolicySettings  ConfigChangeSource = "policy_settings"
	ConfigChangeSourceSkills          ConfigChangeSource = "skills"
)

// HookDecision is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookDecision string

const (
	HookDecisionApprove HookDecision = "approve"
	HookDecisionBlock   HookDecision = "block"
)

// HookPermissionDecision is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookPermissionDecision string

const (
	HookPermissionDecisionAllow HookPermissionDecision = "allow"
	HookPermissionDecisionDeny  HookPermissionDecision = "deny"
	HookPermissionDecisionAsk   HookPermissionDecision = "ask"
	HookPermissionDecisionDefer HookPermissionDecision = "defer"
)

// PermissionResultBehavior is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionresult
type PermissionResultBehavior string

const (
	PermissionResultBehaviorAllow PermissionResultBehavior = "allow"
	PermissionResultBehaviorDeny  PermissionResultBehavior = "deny"
)

func permissionResultBehaviorToProto(v PermissionResultBehavior) pb.PermissionResultBehavior {
	switch v {
	case PermissionResultBehaviorAllow:
		return pb.PermissionResultBehavior_PERMISSION_RESULT_BEHAVIOR_ALLOW
	case PermissionResultBehaviorDeny:
		return pb.PermissionResultBehavior_PERMISSION_RESULT_BEHAVIOR_DENY
	default:
		return pb.PermissionResultBehavior_PERMISSION_RESULT_BEHAVIOR_UNSPECIFIED
	}
}

func permissionResultBehaviorFromProto(v pb.PermissionResultBehavior) PermissionResultBehavior {
	switch v {
	case pb.PermissionResultBehavior_PERMISSION_RESULT_BEHAVIOR_ALLOW:
		return PermissionResultBehaviorAllow
	case pb.PermissionResultBehavior_PERMISSION_RESULT_BEHAVIOR_DENY:
		return PermissionResultBehaviorDeny
	default:
		return ""
	}
}

// PermissionRequestDecisionBehavior is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type PermissionRequestDecisionBehavior string

const (
	PermissionRequestDecisionBehaviorAllow PermissionRequestDecisionBehavior = "allow"
	PermissionRequestDecisionBehaviorDeny  PermissionRequestDecisionBehavior = "deny"
)

func permissionRequestDecisionBehaviorToProto(
	v PermissionRequestDecisionBehavior,
) pb.PermissionRequestDecisionBehavior {
	switch v {
	case PermissionRequestDecisionBehaviorAllow:
		return pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_ALLOW
	case PermissionRequestDecisionBehaviorDeny:
		return pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_DENY
	default:
		return pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_UNSPECIFIED
	}
}

func permissionRequestDecisionBehaviorFromProto(
	v pb.PermissionRequestDecisionBehavior,
) PermissionRequestDecisionBehavior {
	switch v {
	case pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_ALLOW:
		return PermissionRequestDecisionBehaviorAllow
	case pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_DENY:
		return PermissionRequestDecisionBehaviorDeny
	default:
		return ""
	}
}

// TodoStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#todowrite
type TodoStatus string

const (
	TodoStatusPending    TodoStatus = "pending"
	TodoStatusInProgress TodoStatus = "in_progress"
	TodoStatusCompleted  TodoStatus = "completed"
)

// NotebookCellType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#notebookedit
type NotebookCellType string

const (
	NotebookCellTypeCode     NotebookCellType = "code"
	NotebookCellTypeMarkdown NotebookCellType = "markdown"
)

// NotebookEditMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#notebookedit
type NotebookEditMode string

const (
	NotebookEditModeReplace NotebookEditMode = "replace"
	NotebookEditModeInsert  NotebookEditMode = "insert"
	NotebookEditModeDelete  NotebookEditMode = "delete"
)

// GrepOutputMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#grep
type GrepOutputMode string

const (
	GrepOutputModeContent          GrepOutputMode = "content"
	GrepOutputModeFilesWithMatches GrepOutputMode = "files_with_matches"
	GrepOutputModeCount            GrepOutputMode = "count"
)

// FileReadImageMimeType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadImageMimeType string

const (
	FileReadImageMimeTypeJPEG FileReadImageMimeType = "image/jpeg"
	FileReadImageMimeTypePNG  FileReadImageMimeType = "image/png"
	FileReadImageMimeTypeGIF  FileReadImageMimeType = "image/gif"
	FileReadImageMimeTypeWEBP FileReadImageMimeType = "image/webp"
)

// McpServerStatusState is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatus
type McpServerStatusState string

const (
	McpServerStatusStateConnected McpServerStatusState = "connected"
	McpServerStatusStateFailed    McpServerStatusState = "failed"
	McpServerStatusStateNeedsAuth McpServerStatusState = "needs-auth"
	McpServerStatusStatePending   McpServerStatusState = "pending"
	McpServerStatusStateDisabled  McpServerStatusState = "disabled"
)

// McpServerStatusServerInfo is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatus
type McpServerStatusServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// McpServerStatusToolAnnotations is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatus
type McpServerStatusToolAnnotations struct {
	ReadOnly    *bool `json:"readOnly,omitzero"`
	Destructive *bool `json:"destructive,omitzero"`
	OpenWorld   *bool `json:"openWorld,omitzero"`
}

// McpServerStatusTool is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatus
type McpServerStatusTool struct {
	Name        string                          `json:"name"`
	Description *string                         `json:"description,omitzero"`
	Annotations *McpServerStatusToolAnnotations `json:"annotations,omitzero"`
}

// McpServerStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatus
type McpServerStatus struct {
	Name       string                     `json:"name"`
	Status     McpServerStatusState       `json:"status"`
	ServerInfo *McpServerStatusServerInfo `json:"serverInfo,omitzero"`
	Error      *string                    `json:"error,omitzero"`
	Config     McpServerStatusConfig      `json:"config,omitzero"`
	Scope      *string                    `json:"scope,omitzero"`
	Tools      []McpServerStatusTool      `json:"tools,omitzero"`
}

// McpSetServersResult is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpsetserversresult
type McpSetServersResult struct {
	Added   []string          `json:"added"`
	Removed []string          `json:"removed"`
	Errors  map[string]string `json:"errors"`
}

// RewindFilesResult is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#rewindfilesresult
type RewindFilesResult struct {
	CanRewind    bool     `json:"canRewind"`
	Error        *string  `json:"error,omitzero"`
	FilesChanged []string `json:"filesChanged,omitzero"`
	Insertions   *int64   `json:"insertions,omitzero"`
	Deletions    *int64   `json:"deletions,omitzero"`
	SkippedLinks *int64   `json:"skippedLinks,omitzero"`
}

// RateLimitStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkratelimitevent
type RateLimitStatus string

const (
	RateLimitStatusAllowed        RateLimitStatus = "allowed"
	RateLimitStatusAllowedWarning RateLimitStatus = "allowed_warning"
	RateLimitStatusRejected       RateLimitStatus = "rejected"
)

// RateLimitErrorCode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkratelimitevent
type RateLimitErrorCode string

const (
	RateLimitErrorCodeCreditsRequired RateLimitErrorCode = "credits_required"
)

// SystemMessageType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SystemMessageType string

const (
	SystemMessageTypeSystem            SystemMessageType = "system"
	SystemMessageTypeStreamEvent       SystemMessageType = "stream_event"
	SystemMessageTypeAuthStatus        SystemMessageType = "auth_status"
	SystemMessageTypeToolProgress      SystemMessageType = "tool_progress"
	SystemMessageTypeRateLimitEvent    SystemMessageType = "rate_limit_event"
	SystemMessageTypeAssistant         SystemMessageType = "assistant"
	SystemMessageTypeUser              SystemMessageType = "user"
	SystemMessageTypeResult            SystemMessageType = "result"
	SystemMessageTypeToolUseSummary    SystemMessageType = "tool_use_summary"
	SystemMessageTypePromptSuggestion  SystemMessageType = "prompt_suggestion"
	SystemMessageTypeConversationReset SystemMessageType = "conversation_reset"
)

// SystemSubtype is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SystemSubtype string

const (
	SystemSubtypeInit                   SystemSubtype = "init"
	SystemSubtypeCompactBoundary        SystemSubtype = "compact_boundary"
	SystemSubtypeStatus                 SystemSubtype = "status"
	SystemSubtypeTaskNotification       SystemSubtype = "task_notification"
	SystemSubtypeHookStarted            SystemSubtype = "hook_started"
	SystemSubtypeHookProgress           SystemSubtype = "hook_progress"
	SystemSubtypeHookResponse           SystemSubtype = "hook_response"
	SystemSubtypeTaskStarted            SystemSubtype = "task_started"
	SystemSubtypeTaskProgress           SystemSubtype = "task_progress"
	SystemSubtypeTaskUpdated            SystemSubtype = "task_updated"
	SystemSubtypeFilesPersisted         SystemSubtype = "files_persisted"
	SystemSubtypeLocalCommandOutput     SystemSubtype = "local_command_output"
	SystemSubtypeCommandsChanged        SystemSubtype = "commands_changed"
	SystemSubtypePluginInstall          SystemSubtype = "plugin_install"
	SystemSubtypePermissionDenied       SystemSubtype = "permission_denied"
	SystemSubtypeInformational          SystemSubtype = "informational"
	SystemSubtypeWorkerShuttingDown     SystemSubtype = "worker_shutting_down"
	SystemSubtypeBackgroundTasksChanged SystemSubtype = "background_tasks_changed"
	SystemSubtypeThinkingTokens         SystemSubtype = "thinking_tokens"
)

// SDKInformationalMessageLevel is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkinformationalmessage
type SDKInformationalMessageLevel string

const (
	SDKInformationalMessageLevelInfo       SDKInformationalMessageLevel = "info"
	SDKInformationalMessageLevelNotice     SDKInformationalMessageLevel = "notice"
	SDKInformationalMessageLevelSuggestion SDKInformationalMessageLevel = "suggestion"
	SDKInformationalMessageLevelWarning    SDKInformationalMessageLevel = "warning"
)

// SDKAssistantMessageError is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkassistantmessage
type SDKAssistantMessageError string

const (
	SDKAssistantMessageErrorAuthenticationFailed SDKAssistantMessageError = "authentication_failed"
	SDKAssistantMessageErrorOauthOrgNotAllowed   SDKAssistantMessageError = "oauth_org_not_allowed"
	SDKAssistantMessageErrorBillingError         SDKAssistantMessageError = "billing_error"
	SDKAssistantMessageErrorRateLimit            SDKAssistantMessageError = "rate_limit"
	SDKAssistantMessageErrorOverloaded           SDKAssistantMessageError = "overloaded"
	SDKAssistantMessageErrorInvalidRequest       SDKAssistantMessageError = "invalid_request"
	SDKAssistantMessageErrorModelNotFound        SDKAssistantMessageError = "model_not_found"
	SDKAssistantMessageErrorServerError          SDKAssistantMessageError = "server_error"
	SDKAssistantMessageErrorMaxOutputTokens      SDKAssistantMessageError = "max_output_tokens"
	SDKAssistantMessageErrorUnknown              SDKAssistantMessageError = "unknown"
)

// TerminalReason is a handwritten Claude Agent SDK type. It reports why the
// agent loop ended, carried on SDKResultMessage.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type TerminalReason string

const (
	TerminalReasonCompleted                      TerminalReason = "completed"
	TerminalReasonMaxTurns                       TerminalReason = "max_turns"
	TerminalReasonToolDeferred                   TerminalReason = "tool_deferred"
	TerminalReasonAbortedStreaming               TerminalReason = "aborted_streaming"
	TerminalReasonAbortedTools                   TerminalReason = "aborted_tools"
	TerminalReasonHookStopped                    TerminalReason = "hook_stopped"
	TerminalReasonStopHookPrevented              TerminalReason = "stop_hook_prevented"
	TerminalReasonBlockingLimit                  TerminalReason = "blocking_limit"
	TerminalReasonRapidRefillBreaker             TerminalReason = "rapid_refill_breaker"
	TerminalReasonPromptTooLong                  TerminalReason = "prompt_too_long"
	TerminalReasonImageError                     TerminalReason = "image_error"
	TerminalReasonModelError                     TerminalReason = "model_error"
	TerminalReasonBackgroundRequested            TerminalReason = "background_requested"
	TerminalReasonAPIError                       TerminalReason = "api_error"
	TerminalReasonMalformedToolUseExhausted      TerminalReason = "malformed_tool_use_exhausted"
	TerminalReasonBudgetExhausted                TerminalReason = "budget_exhausted"
	TerminalReasonStructuredOutputRetryExhausted TerminalReason = "structured_output_retry_exhausted"
	TerminalReasonToolDeferredUnavailable        TerminalReason = "tool_deferred_unavailable"
	TerminalReasonTurnSetupFailed                TerminalReason = "turn_setup_failed"
)

// FastModeState is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type FastModeState string

const (
	FastModeStateOn       FastModeState = "on"
	FastModeStateOff      FastModeState = "off"
	FastModeStateCooldown FastModeState = "cooldown"
)

// FastModeDisabledReason is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type FastModeDisabledReason string

const (
	FastModeDisabledReasonFree               FastModeDisabledReason = "free"
	FastModeDisabledReasonPreference         FastModeDisabledReason = "preference"
	FastModeDisabledReasonExtraUsageDisabled FastModeDisabledReason = "extra_usage_disabled"
	FastModeDisabledReasonNetworkError       FastModeDisabledReason = "network_error"
	FastModeDisabledReasonUnknown            FastModeDisabledReason = "unknown"
	FastModeDisabledReasonNotFirstParty      FastModeDisabledReason = "not_first_party"
	FastModeDisabledReasonDisabledByEnv      FastModeDisabledReason = "disabled_by_env"
	FastModeDisabledReasonModelNotAllowed    FastModeDisabledReason = "model_not_allowed"
	FastModeDisabledReasonSdkOptInRequired   FastModeDisabledReason = "sdk_opt_in_required"
	FastModeDisabledReasonPending            FastModeDisabledReason = "pending"
)

// SDKMessageOriginKind is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessageorigin
type SDKMessageOriginKind string

const (
	SDKMessageOriginKindHuman            SDKMessageOriginKind = "human"
	SDKMessageOriginKindChannel          SDKMessageOriginKind = "channel"
	SDKMessageOriginKindPeer             SDKMessageOriginKind = "peer"
	SDKMessageOriginKindTaskNotification SDKMessageOriginKind = "task-notification"
	SDKMessageOriginKindCoordinator      SDKMessageOriginKind = "coordinator"
	SDKMessageOriginKindAutoContinuation SDKMessageOriginKind = "auto-continuation"
)

// SDKMessageOrigin is a handwritten Claude Agent SDK type. It is the provenance
// of a user-role message, discriminated on kind. The variant-specific fields
// (server for channel; from/name/fromSession/senderTaskId/body/verifiedPeerPid
// for peer) are present only for their kind.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessageorigin
type SDKMessageOrigin struct {
	Kind            SDKMessageOriginKind `json:"kind"`
	Server          *string              `json:"server,omitzero"`
	From            *string              `json:"from,omitzero"`
	Name            *string              `json:"name,omitzero"`
	FromSession     *string              `json:"fromSession,omitzero"`
	SenderTaskID    *string              `json:"senderTaskId,omitzero"`
	Body            *string              `json:"body,omitzero"`
	VerifiedPeerPid *int64               `json:"verifiedPeerPid,omitzero"`
}

// DeferredToolUse is a handwritten Claude Agent SDK type. It carries a tool
// call deferred by a PreToolUse hook returning permissionDecision "defer".
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type DeferredToolUse struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

// SDKResultSubtype is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type SDKResultSubtype string

const (
	SDKResultSubtypeSuccess                         SDKResultSubtype = "success"
	SDKResultSubtypeErrorMaxTurns                   SDKResultSubtype = "error_max_turns"
	SDKResultSubtypeErrorDuringExecution            SDKResultSubtype = "error_during_execution"
	SDKResultSubtypeErrorMaxBudgetUSD               SDKResultSubtype = "error_max_budget_usd"
	SDKResultSubtypeErrorMaxStructuredOutputRetries SDKResultSubtype = "error_max_structured_output_retries"
)

// HookResponseOutcome is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkhookresponsemessage
type HookResponseOutcome string

const (
	HookResponseOutcomeSuccess   HookResponseOutcome = "success"
	HookResponseOutcomeError     HookResponseOutcome = "error"
	HookResponseOutcomeCancelled HookResponseOutcome = "cancelled"
)

// TaskNotificationStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktasknotificationmessage
type TaskNotificationStatus string

const (
	TaskNotificationStatusCompleted TaskNotificationStatus = "completed"
	TaskNotificationStatusFailed    TaskNotificationStatus = "failed"
	TaskNotificationStatusStopped   TaskNotificationStatus = "stopped"
)

// FilesPersistedFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkfilespersistedevent
type FilesPersistedFile struct {
	Filename string `json:"filename"`
	FileID   string `json:"file_id"`
}

// FilesPersistedFailure is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkfilespersistedevent
type FilesPersistedFailure struct {
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

// RateLimitInfo is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkratelimitevent
type RateLimitInfo struct {
	Status                          RateLimitStatus     `json:"status"`
	ResetsAt                        *float64            `json:"resetsAt,omitzero"`
	Utilization                     *float64            `json:"utilization,omitzero"`
	ErrorCode                       *RateLimitErrorCode `json:"errorCode,omitzero"`
	CanUserPurchaseCredits          *bool               `json:"canUserPurchaseCredits,omitzero"`
	HasChargeableSavedPaymentMethod *bool               `json:"hasChargeableSavedPaymentMethod,omitzero"`
}

// ToolAnnotations is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolannotations
type ToolAnnotations struct {
	Title           *string `json:"title,omitzero"`
	ReadOnlyHint    *bool   `json:"readOnlyHint,omitzero"`
	DestructiveHint *bool   `json:"destructiveHint,omitzero"`
	IdempotentHint  *bool   `json:"idempotentHint,omitzero"`
	OpenWorldHint   *bool   `json:"openWorldHint,omitzero"`
}

// AskUserQuestionPreviewFormat is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolconfig
type AskUserQuestionPreviewFormat string

const (
	AskUserQuestionPreviewFormatMarkdown AskUserQuestionPreviewFormat = "markdown"
	AskUserQuestionPreviewFormatHTML     AskUserQuestionPreviewFormat = "html"
)

// ToolConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolconfig
type ToolConfig struct {
	AskUserQuestion *AskUserQuestionToolConfig `json:"askUserQuestion,omitzero"`
}

// AskUserQuestionToolConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolconfig
type AskUserQuestionToolConfig struct {
	PreviewFormat *AskUserQuestionPreviewFormat `json:"previewFormat,omitzero"`
}

// OutputFormat is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type OutputFormat struct {
	Type   string          `json:"type"`
	Schema json.RawMessage `json:"schema"`
}

// SystemPrompt is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPrompt struct {
	value SystemPrompt_Value
}

// SystemPrompt_Value is the variant interface implemented by every [SystemPrompt] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPrompt_Value interface{ systemPrompt() }

// MarshalJSON marshals the active [SystemPrompt] variant.
func (o SystemPrompt) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [SystemPrompt] union value from JSON. The SDK emits a
// bare string or a {"type":"preset",...} object, so dispatch is by JSON token.
func (o *SystemPrompt) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var (
		v   SystemPrompt_Value
		err error
	)
	switch trimmed[0] {
	case '"':
		v, err = decodeUnionVariant[SystemPromptString](data)
	case '{':
		var disc struct {
			Type string `json:"type"`
		}
		if e := json.Unmarshal(data, &disc); e != nil {
			return e
		}
		if disc.Type == "preset" {
			v, err = decodeUnionVariant[SystemPromptPreset](data)
		} else {
			v, err = decodeUnionVariant[SystemPromptUnknown](data)
		}
	default:
		v, err = decodeUnionVariant[SystemPromptUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ SystemPrompt_Value = (*SystemPromptUnknown)(nil)
	_ SystemPrompt_Value = (*SystemPromptString)(nil)
	_ SystemPrompt_Value = (*SystemPromptPreset)(nil)
)

// SystemPromptUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPromptUnknown struct{ UnknownUnion }

// SystemPromptString is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPromptString struct{ Value string }

// MarshalJSON encodes [SystemPromptString] as the bare SDK string form.
func (o SystemPromptString) MarshalJSON() ([]byte, error) { return json.Marshal(o.Value) }

// UnmarshalJSON decodes the bare SDK string form into [SystemPromptString].
func (o *SystemPromptString) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &o.Value)
}

// SystemPromptPreset is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPromptPreset struct {
	Type                   string                 `json:"type"`
	Preset                 SystemPromptPresetName `json:"preset"`
	Append                 *string                `json:"append,omitzero"`
	ExcludeDynamicSections *bool                  `json:"excludeDynamicSections,omitzero"`
}

func (SystemPromptUnknown) systemPrompt() {}
func (SystemPromptString) systemPrompt()  {}
func (SystemPromptPreset) systemPrompt()  {}

// ThinkingConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfig struct {
	value ThinkingConfig_Value
}

// ThinkingConfig_Value is the variant interface implemented by every [ThinkingConfig] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfig_Value interface{ thinkingConfig() }

// MarshalJSON marshals the active [ThinkingConfig] variant.
func (o ThinkingConfig) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [ThinkingConfig] union value from JSON, dispatching on `type`.
func (o *ThinkingConfig) UnmarshalJSON(data []byte) error {
	var disc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	var (
		v   ThinkingConfig_Value
		err error
	)
	switch disc.Type {
	case "adaptive":
		v, err = decodeUnionVariant[ThinkingConfigAdaptive](data)
	case "enabled":
		v, err = decodeUnionVariant[ThinkingConfigEnabled](data)
	case "disabled":
		v, err = decodeUnionVariant[ThinkingConfigDisabled](data)
	default:
		v, err = decodeUnionVariant[ThinkingConfigUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ ThinkingConfig_Value = (*ThinkingConfigUnknown)(nil)
	_ ThinkingConfig_Value = (*ThinkingConfigAdaptive)(nil)
	_ ThinkingConfig_Value = (*ThinkingConfigEnabled)(nil)
	_ ThinkingConfig_Value = (*ThinkingConfigDisabled)(nil)
)

// ThinkingConfigUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfigUnknown struct{ UnknownUnion }

// ThinkingDisplay is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#thinkingconfig
type ThinkingDisplay string

const (
	ThinkingDisplaySummarized ThinkingDisplay = "summarized"
	ThinkingDisplayOmitted    ThinkingDisplay = "omitted"
)

// ThinkingConfigAdaptive is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#thinkingconfig
type ThinkingConfigAdaptive struct {
	Type    string           `json:"type"`
	Display *ThinkingDisplay `json:"display,omitzero"`
}

// ThinkingConfigEnabled is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfigEnabled struct {
	Type         string           `json:"type"`
	BudgetTokens *int64           `json:"budgetTokens,omitzero"`
	Display      *ThinkingDisplay `json:"display,omitzero"`
}

// ThinkingConfigDisabled is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfigDisabled struct {
	Type string `json:"type"`
}

func (ThinkingConfigUnknown) thinkingConfig()  {}
func (ThinkingConfigAdaptive) thinkingConfig() {}
func (ThinkingConfigEnabled) thinkingConfig()  {}
func (ThinkingConfigDisabled) thinkingConfig() {}

// ToolsConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ToolsConfig struct {
	value ToolsConfig_Value
}

// ToolsConfig_Value is the variant interface implemented by every [ToolsConfig] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ToolsConfig_Value interface{ toolsConfig() }

// MarshalJSON marshals the active [ToolsConfig] variant.
func (o ToolsConfig) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [ToolsConfig] union value from JSON. The SDK emits a
// string array or a {"type":"preset",...} object, so dispatch is by JSON token.
func (o *ToolsConfig) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var (
		v   ToolsConfig_Value
		err error
	)
	switch trimmed[0] {
	case '[':
		v, err = decodeUnionVariant[ToolsConfigList](data)
	case '{':
		var disc struct {
			Type string `json:"type"`
		}
		if e := json.Unmarshal(data, &disc); e != nil {
			return e
		}
		if disc.Type == "preset" {
			v, err = decodeUnionVariant[ToolsConfigPreset](data)
		} else {
			v, err = decodeUnionVariant[ToolsConfigUnknown](data)
		}
	default:
		v, err = decodeUnionVariant[ToolsConfigUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ ToolsConfig_Value = (*ToolsConfigUnknown)(nil)
	_ ToolsConfig_Value = (*ToolsConfigList)(nil)
	_ ToolsConfig_Value = (*ToolsConfigPreset)(nil)
)

// ToolsConfigUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ToolsConfigUnknown struct{ UnknownUnion }

// ToolsConfigList is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ToolsConfigList struct {
	Tools []string `json:"-"`
}

// MarshalJSON encodes [ToolsConfigList] as the bare SDK string-array form.
func (o ToolsConfigList) MarshalJSON() ([]byte, error) { return json.Marshal(o.Tools) }

// UnmarshalJSON decodes the bare SDK string-array form into [ToolsConfigList].
func (o *ToolsConfigList) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &o.Tools)
}

// ToolsConfigPreset is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ToolsConfigPreset struct {
	Type   string     `json:"type"`
	Preset ToolPreset `json:"preset"`
}

func (ToolsConfigUnknown) toolsConfig() {}
func (ToolsConfigList) toolsConfig()    {}
func (ToolsConfigPreset) toolsConfig()  {}

// PermissionRuleValue is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionrulevalue
type PermissionRuleValue struct {
	ToolName    string  `json:"toolName"`
	RuleContent *string `json:"ruleContent,omitzero"`
}

// PermissionUpdate is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdate struct {
	value PermissionUpdate_Value
}

// PermissionUpdate_Value is the variant interface implemented by every [PermissionUpdate] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdate_Value interface{ permissionUpdate() }

// MarshalJSON marshals the active [PermissionUpdate] variant.
func (o PermissionUpdate) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [PermissionUpdate] union value from JSON.
func (o *PermissionUpdate) UnmarshalJSON(data []byte) error {
	var disc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	switch disc.Type {
	case "addRules":
		var v PermissionUpdateAddRules
		o.value = &v
		return v.UnmarshalJSON(data)
	case "replaceRules":
		var v PermissionUpdateReplaceRules
		o.value = &v
		return v.UnmarshalJSON(data)
	case "removeRules":
		var v PermissionUpdateRemoveRules
		o.value = &v
		return v.UnmarshalJSON(data)
	case "setMode":
		var v PermissionUpdateSetMode
		o.value = &v
		return v.UnmarshalJSON(data)
	case "addDirectories":
		var v PermissionUpdateAddDirectories
		o.value = &v
		return v.UnmarshalJSON(data)
	case "removeDirectories":
		var v PermissionUpdateRemoveDirectories
		o.value = &v
		return v.UnmarshalJSON(data)
	default:
		var v PermissionUpdateUnknown
		o.value = &v
		return v.UnmarshalJSON(data)
	}
}

var (
	_ PermissionUpdate_Value = (*PermissionUpdateUnknown)(nil)
	_ PermissionUpdate_Value = (*PermissionUpdateAddRules)(nil)
	_ PermissionUpdate_Value = (*PermissionUpdateReplaceRules)(nil)
	_ PermissionUpdate_Value = (*PermissionUpdateRemoveRules)(nil)
	_ PermissionUpdate_Value = (*PermissionUpdateSetMode)(nil)
	_ PermissionUpdate_Value = (*PermissionUpdateAddDirectories)(nil)
	_ PermissionUpdate_Value = (*PermissionUpdateRemoveDirectories)(nil)
)

// PermissionUpdateUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdateUnknown struct{ UnknownUnion }

// PermissionUpdateAddRules is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdateAddRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateReplaceRules is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdateReplaceRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateRemoveRules is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdateRemoveRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateSetMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdateSetMode struct {
	Type        string                      `json:"type"`
	Mode        PermissionMode              `json:"mode"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateAddDirectories is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdateAddDirectories struct {
	Type        string                      `json:"type"`
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateRemoveDirectories is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionupdate
type PermissionUpdateRemoveDirectories struct {
	Type        string                      `json:"type"`
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (PermissionUpdateUnknown) permissionUpdate()           {}
func (PermissionUpdateAddRules) permissionUpdate()          {}
func (PermissionUpdateReplaceRules) permissionUpdate()      {}
func (PermissionUpdateRemoveRules) permissionUpdate()       {}
func (PermissionUpdateSetMode) permissionUpdate()           {}
func (PermissionUpdateAddDirectories) permissionUpdate()    {}
func (PermissionUpdateRemoveDirectories) permissionUpdate() {}

// PermissionResult is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionresult
type PermissionResult struct {
	value PermissionResult_Value
}

// PermissionResult_Value is the variant interface implemented by every [PermissionResult] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionresult
type PermissionResult_Value interface{ permissionResult() }

// MarshalJSON marshals the active [PermissionResult] variant.
func (o PermissionResult) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [PermissionResult] union value from JSON, dispatching on `behavior`.
func (o *PermissionResult) UnmarshalJSON(data []byte) error {
	var disc struct {
		Behavior PermissionResultBehavior `json:"behavior"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	var (
		v   PermissionResult_Value
		err error
	)
	switch disc.Behavior {
	case PermissionResultBehaviorAllow:
		v, err = decodeUnionVariant[PermissionResultAllow](data)
	case PermissionResultBehaviorDeny:
		v, err = decodeUnionVariant[PermissionResultDeny](data)
	default:
		v, err = decodeUnionVariant[PermissionResultUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ PermissionResult_Value = (*PermissionResultUnknown)(nil)
	_ PermissionResult_Value = (*PermissionResultAllow)(nil)
	_ PermissionResult_Value = (*PermissionResultDeny)(nil)
)

// PermissionResultUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionresult
type PermissionResultUnknown struct{ UnknownUnion }

// PermissionResultAllow is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionresult
type PermissionResultAllow struct {
	Behavior           PermissionResultBehavior `json:"behavior"`
	UpdatedInput       map[string]any           `json:"updatedInput,omitzero"`
	UpdatedPermissions []PermissionUpdate       `json:"updatedPermissions,omitzero"`
	ToolUseID          *string                  `json:"toolUseID,omitzero"`
}

// PermissionResultDeny is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionresult
type PermissionResultDeny struct {
	Behavior  PermissionResultBehavior `json:"behavior"`
	Message   string                   `json:"message"`
	Interrupt *bool                    `json:"interrupt,omitzero"`
	ToolUseID *string                  `json:"toolUseID,omitzero"`
}

func (PermissionResultUnknown) permissionResult() {}
func (PermissionResultAllow) permissionResult()   {}
func (PermissionResultDeny) permissionResult()    {}

// McpServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverconfig
type McpServerConfig struct {
	value McpServerConfig_Value
}

// McpServerConfig_Value is the variant interface implemented by every [McpServerConfig] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverconfig
type McpServerConfig_Value interface{ mcpServerConfig() }

// MarshalJSON marshals the active [McpServerConfig] variant.
func (o McpServerConfig) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes an [McpServerConfig] union value from JSON, dispatching
// on `type` (absent or "stdio" selects the stdio variant). The documented union
// has four members; a "claudeai-proxy" payload is preserved as the unknown
// variant here because [McpClaudeAIProxyServerConfig] is not part of this union
// (it is one in [McpServerStatusConfig]).
func (o *McpServerConfig) UnmarshalJSON(data []byte) error {
	var disc struct {
		Type *string `json:"type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	t := ""
	if disc.Type != nil {
		t = *disc.Type
	}
	var (
		v   McpServerConfig_Value
		err error
	)
	switch t {
	case "", "stdio":
		v, err = decodeUnionVariant[McpStdioServerConfig](data)
	case "sse":
		v, err = decodeUnionVariant[McpSSEServerConfig](data)
	case "http":
		v, err = decodeUnionVariant[McpHttpServerConfig](data)
	case "sdk":
		v, err = decodeUnionVariant[McpSdkServerConfigWithInstance](data)
	default:
		v, err = decodeUnionVariant[McpServerConfigUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ McpServerConfig_Value = (*McpServerConfigUnknown)(nil)
	_ McpServerConfig_Value = (*McpStdioServerConfig)(nil)
	_ McpServerConfig_Value = (*McpSSEServerConfig)(nil)
	_ McpServerConfig_Value = (*McpHttpServerConfig)(nil)
	_ McpServerConfig_Value = (*McpSdkServerConfigWithInstance)(nil)
)

// McpServerConfigUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverconfig
type McpServerConfigUnknown struct{ UnknownUnion }

// McpStdioServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpstdioserverconfig
type McpStdioServerConfig struct {
	Type    *string           `json:"type,omitzero"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitzero"`
	Env     map[string]string `json:"env,omitzero"`
}

// McpSSEServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpsseserverconfig
type McpSSEServerConfig struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitzero"`
}

// McpHttpServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcphttpserverconfig
type McpHttpServerConfig struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitzero"`
}

// McpSdkServerConfigWithInstance is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpsdkserverconfigwithinstance
type McpSdkServerConfigWithInstance struct {
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Instance json.RawMessage `json:"instance"`
}

// McpSdkServerConfig is a handwritten Claude Agent SDK type. The docs name it
// as a member of McpServerStatusConfig and AgentMcpServerSpec without defining
// it on the page, so the payload is preserved as raw JSON.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatusconfig
type McpSdkServerConfig json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m McpSdkServerConfig) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *McpSdkServerConfig) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

// McpClaudeAIProxyServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpclaudeaiproxyserverconfig
type McpClaudeAIProxyServerConfig struct {
	Type string `json:"type"`
	URL  string `json:"url"`
	ID   string `json:"id"`
}

func (McpServerConfigUnknown) mcpServerConfig()         {}
func (McpStdioServerConfig) mcpServerConfig()           {}
func (McpSSEServerConfig) mcpServerConfig()             {}
func (McpHttpServerConfig) mcpServerConfig()            {}
func (McpSdkServerConfigWithInstance) mcpServerConfig() {}

// McpServerStatusConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatusconfig
type McpServerStatusConfig struct {
	value McpServerStatusConfig_Value
}

// McpServerStatusConfig_Value is the variant interface implemented by every
// [McpServerStatusConfig] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatusconfig
type McpServerStatusConfig_Value interface{ mcpServerStatusConfig() }

// MarshalJSON marshals the active [McpServerStatusConfig] variant.
func (o McpServerStatusConfig) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes an [McpServerStatusConfig] union value from JSON,
// dispatching on `type` (absent or "stdio" selects the stdio variant). The
// "sdk" member is the docs' McpSdkServerConfig, which is not defined on the
// page, so it decodes into the raw-preserving [McpSdkServerConfig].
func (o *McpServerStatusConfig) UnmarshalJSON(data []byte) error {
	var disc struct {
		Type *string `json:"type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	t := ""
	if disc.Type != nil {
		t = *disc.Type
	}
	var (
		v   McpServerStatusConfig_Value
		err error
	)
	switch t {
	case "", "stdio":
		v, err = decodeUnionVariant[McpStdioServerConfig](data)
	case "sse":
		v, err = decodeUnionVariant[McpSSEServerConfig](data)
	case "http":
		v, err = decodeUnionVariant[McpHttpServerConfig](data)
	case "sdk":
		v, err = decodeUnionVariant[McpSdkServerConfig](data)
	case "claudeai-proxy":
		v, err = decodeUnionVariant[McpClaudeAIProxyServerConfig](data)
	default:
		v, err = decodeUnionVariant[McpServerStatusConfigUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ McpServerStatusConfig_Value = (*McpServerStatusConfigUnknown)(nil)
	_ McpServerStatusConfig_Value = (*McpStdioServerConfig)(nil)
	_ McpServerStatusConfig_Value = (*McpSSEServerConfig)(nil)
	_ McpServerStatusConfig_Value = (*McpHttpServerConfig)(nil)
	_ McpServerStatusConfig_Value = (*McpSdkServerConfig)(nil)
	_ McpServerStatusConfig_Value = (*McpClaudeAIProxyServerConfig)(nil)
)

// NewMcpServerStatusConfig wraps a variant into an [McpServerStatusConfig].
func NewMcpServerStatusConfig(v McpServerStatusConfig_Value) McpServerStatusConfig {
	return McpServerStatusConfig{value: v}
}

// GetValue returns the active [McpServerStatusConfig] variant, or nil when unset.
func (o McpServerStatusConfig) GetValue() McpServerStatusConfig_Value { return o.value }

// McpServerStatusConfigUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpserverstatusconfig
type McpServerStatusConfigUnknown struct{ UnknownUnion }

func (McpServerStatusConfigUnknown) mcpServerStatusConfig() {}
func (McpStdioServerConfig) mcpServerStatusConfig()         {}
func (McpSSEServerConfig) mcpServerStatusConfig()           {}
func (McpHttpServerConfig) mcpServerStatusConfig()          {}
func (McpSdkServerConfig) mcpServerStatusConfig()           {}
func (McpClaudeAIProxyServerConfig) mcpServerStatusConfig() {}

// SdkPluginConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkpluginconfig
type SdkPluginConfig struct {
	Type             string `json:"type"`
	Path             string `json:"path"`
	SkipMcpDiscovery *bool  `json:"skipMcpDiscovery,omitzero"`
}

// SandboxNetworkConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sandboxnetworkconfig
type SandboxNetworkConfig struct {
	AllowedDomains          []string `json:"allowedDomains,omitzero"`
	DeniedDomains           []string `json:"deniedDomains,omitzero"`
	StrictAllowlist         *bool    `json:"strictAllowlist,omitzero"`
	AllowManagedDomainsOnly *bool    `json:"allowManagedDomainsOnly,omitzero"`
	AllowLocalBinding       *bool    `json:"allowLocalBinding,omitzero"`
	AllowUnixSockets        []string `json:"allowUnixSockets,omitzero"`
	AllowAllUnixSockets     *bool    `json:"allowAllUnixSockets,omitzero"`
	HttpProxyPort           *int64   `json:"httpProxyPort,omitzero"`
	SocksProxyPort          *int64   `json:"socksProxyPort,omitzero"`
}

// SandboxFilesystemConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sandboxfilesystemconfig
type SandboxFilesystemConfig struct {
	AllowWrite []string `json:"allowWrite,omitzero"`
	DenyWrite  []string `json:"denyWrite,omitzero"`
	DenyRead   []string `json:"denyRead,omitzero"`
}

// RipgrepConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sandboxsettings
type RipgrepConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitzero"`
}

// SandboxSettings is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sandboxsettings
type SandboxSettings struct {
	Enabled                   *bool                    `json:"enabled,omitzero"`
	FailIfUnavailable         *bool                    `json:"failIfUnavailable,omitzero"`
	AutoAllowBashIfSandboxed  *bool                    `json:"autoAllowBashIfSandboxed,omitzero"`
	ExcludedCommands          []string                 `json:"excludedCommands,omitzero"`
	AllowUnsandboxedCommands  *bool                    `json:"allowUnsandboxedCommands,omitzero"`
	Network                   *SandboxNetworkConfig    `json:"network,omitzero"`
	Filesystem                *SandboxFilesystemConfig `json:"filesystem,omitzero"`
	IgnoreViolations          map[string][]string      `json:"ignoreViolations,omitzero"`
	EnableWeakerNestedSandbox *bool                    `json:"enableWeakerNestedSandbox,omitzero"`
	Ripgrep                   *RipgrepConfig           `json:"ripgrep,omitzero"`
}

// AgentDefinition is a handwritten Claude Agent SDK type. Model is a plain
// string upstream, accepting an alias such as "fable", "opus", "sonnet",
// "haiku", "inherit", or a full model ID.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentdefinition
type AgentDefinition struct {
	Description                        string               `json:"description"`
	Tools                              []string             `json:"tools,omitzero"`
	DisallowedTools                    []string             `json:"disallowedTools,omitzero"`
	Prompt                             string               `json:"prompt"`
	Model                              *string              `json:"model,omitzero"`
	McpServers                         []AgentMcpServerSpec `json:"mcpServers,omitzero"`
	Skills                             []string             `json:"skills,omitzero"`
	InitialPrompt                      *string              `json:"initialPrompt,omitzero"`
	MaxTurns                           *int64               `json:"maxTurns,omitzero"`
	Background                         *bool                `json:"background,omitzero"`
	Memory                             *AgentMemory         `json:"memory,omitzero"`
	Effort                             json.RawMessage      `json:"effort,omitzero"`
	PermissionMode                     *PermissionMode      `json:"permissionMode,omitzero"`
	CriticalSystemReminderExperimental *string              `json:"criticalSystemReminder_EXPERIMENTAL,omitzero"`
}

// AgentMcpServerSpec is a handwritten Claude Agent SDK type. It is either a
// server name string or a map of name to McpServerConfigForProcessTransport
// (McpStdioServerConfig | McpSSEServerConfig | McpHttpServerConfig |
// McpSdkServerConfig).
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentmcpserverspec
type AgentMcpServerSpec struct {
	value AgentMcpServerSpec_Value
}

// AgentMcpServerSpec_Value is the variant interface implemented by every
// [AgentMcpServerSpec] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentmcpserverspec
type AgentMcpServerSpec_Value interface{ agentMcpServerSpec() }

// MarshalJSON marshals the active [AgentMcpServerSpec] variant.
func (o AgentMcpServerSpec) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes an [AgentMcpServerSpec] union value from JSON. The SDK
// emits a bare server-name string or a name-to-config object, so dispatch is by
// JSON token.
func (o *AgentMcpServerSpec) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var (
		v   AgentMcpServerSpec_Value
		err error
	)
	switch trimmed[0] {
	case '"':
		v, err = decodeUnionVariant[AgentMcpServerSpecName](data)
	case '{':
		v, err = decodeUnionVariant[AgentMcpServerSpecConfigs](data)
	default:
		v, err = decodeUnionVariant[AgentMcpServerSpecUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ AgentMcpServerSpec_Value = (*AgentMcpServerSpecUnknown)(nil)
	_ AgentMcpServerSpec_Value = (*AgentMcpServerSpecName)(nil)
	_ AgentMcpServerSpec_Value = (*AgentMcpServerSpecConfigs)(nil)
)

// NewAgentMcpServerSpec wraps a variant into an [AgentMcpServerSpec].
func NewAgentMcpServerSpec(v AgentMcpServerSpec_Value) AgentMcpServerSpec {
	return AgentMcpServerSpec{value: v}
}

// GetValue returns the active [AgentMcpServerSpec] variant, or nil when unset.
func (o AgentMcpServerSpec) GetValue() AgentMcpServerSpec_Value { return o.value }

// GetName reports whether the active variant is [*AgentMcpServerSpecName] and returns it.
func (o AgentMcpServerSpec) GetName() (*AgentMcpServerSpecName, bool) {
	v, ok := o.value.(*AgentMcpServerSpecName)
	return v, ok
}

// GetConfigs reports whether the active variant is [*AgentMcpServerSpecConfigs] and returns it.
func (o AgentMcpServerSpec) GetConfigs() (*AgentMcpServerSpecConfigs, bool) {
	v, ok := o.value.(*AgentMcpServerSpecConfigs)
	return v, ok
}

// GetUnknown reports whether the active variant is [*AgentMcpServerSpecUnknown] and returns it.
func (o AgentMcpServerSpec) GetUnknown() (*AgentMcpServerSpecUnknown, bool) {
	v, ok := o.value.(*AgentMcpServerSpecUnknown)
	return v, ok
}

// AgentMcpServerSpecUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentmcpserverspec
type AgentMcpServerSpecUnknown struct{ UnknownUnion }

// AgentMcpServerSpecName is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentmcpserverspec
type AgentMcpServerSpecName struct{ Value string }

// MarshalJSON encodes [AgentMcpServerSpecName] as the bare SDK string form.
func (o AgentMcpServerSpecName) MarshalJSON() ([]byte, error) { return json.Marshal(o.Value) }

// UnmarshalJSON decodes the bare SDK string form into [AgentMcpServerSpecName].
func (o *AgentMcpServerSpecName) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &o.Value)
}

// AgentMcpServerSpecConfigs is a handwritten Claude Agent SDK type. Values are
// the docs' McpServerConfigForProcessTransport union, which shares members with
// [McpServerStatusConfig] except the Claude-AI proxy variant; that variant
// decodes into the unknown case here.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentmcpserverspec
type AgentMcpServerSpecConfigs map[string]McpServerConfig

func (AgentMcpServerSpecUnknown) agentMcpServerSpec() {}
func (AgentMcpServerSpecName) agentMcpServerSpec()    {}
func (AgentMcpServerSpecConfigs) agentMcpServerSpec() {}

// AgentMemory is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentdefinition
type AgentMemory string

const (
	AgentMemoryUser    AgentMemory = "user"
	AgentMemoryProject AgentMemory = "project"
	AgentMemoryLocal   AgentMemory = "local"
)

// Options is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type Options struct {
	AdditionalDirectories           []string                   `json:"additionalDirectories,omitzero"`
	Agent                           *string                    `json:"agent,omitzero"`
	Agents                          map[string]AgentDefinition `json:"agents,omitzero"`
	AllowDangerouslySkipPermissions *bool                      `json:"allowDangerouslySkipPermissions,omitzero"`
	AllowedTools                    []string                   `json:"allowedTools,omitzero"`
	Betas                           []SdkBeta                  `json:"betas,omitzero"`
	Continue                        *bool                      `json:"continue,omitzero"`
	Cwd                             *string                    `json:"cwd,omitzero"`
	Debug                           *bool                      `json:"debug,omitzero"`
	DebugFile                       *string                    `json:"debugFile,omitzero"`
	DisallowedTools                 []string                   `json:"disallowedTools,omitzero"`
	Effort                          *Effort                    `json:"effort,omitzero"`
	EnableFileCheckpointing         *bool                      `json:"enableFileCheckpointing,omitzero"`
	Env                             map[string]*string         `json:"env,omitzero"`
	Executable                      *OptionsExecutable         `json:"executable,omitzero"`
	ExecutableArgs                  []string                   `json:"executableArgs,omitzero"`
	ExtraArgs                       map[string]*string         `json:"extraArgs,omitzero"`
	FallbackModel                   *string                    `json:"fallbackModel,omitzero"`
	ForkSession                     *bool                      `json:"forkSession,omitzero"`
	IncludePartialMessages          *bool                      `json:"includePartialMessages,omitzero"`
	MaxBudgetUsd                    *float64                   `json:"maxBudgetUsd,omitzero"`
	// Deprecated: use Thinking instead.
	MaxThinkingTokens          *int64                     `json:"maxThinkingTokens,omitzero"`
	MaxTurns                   *int64                     `json:"maxTurns,omitzero"`
	McpServers                 map[string]McpServerConfig `json:"mcpServers,omitzero"`
	Model                      *string                    `json:"model,omitzero"`
	OutputFormat               *OutputFormat              `json:"outputFormat,omitzero"`
	PathToClaudeCodeExecutable *string                    `json:"pathToClaudeCodeExecutable,omitzero"`
	PermissionMode             *PermissionMode            `json:"permissionMode,omitzero"`
	PermissionPromptToolName   *string                    `json:"permissionPromptToolName,omitzero"`
	PersistSession             *bool                      `json:"persistSession,omitzero"`
	Plugins                    []SdkPluginConfig          `json:"plugins,omitzero"`
	PromptSuggestions          *bool                      `json:"promptSuggestions,omitzero"`
	Resume                     *string                    `json:"resume,omitzero"`
	ResumeSessionAt            *string                    `json:"resumeSessionAt,omitzero"`
	Sandbox                    *SandboxSettings           `json:"sandbox,omitzero"`
	SessionID                  *string                    `json:"sessionId,omitzero"`
	SettingSources             []SettingSource            `json:"settingSources,omitzero"`
	StrictMcpConfig            *bool                      `json:"strictMcpConfig,omitzero"`
	SystemPrompt               SystemPrompt               `json:"systemPrompt,omitzero"`
	Thinking                   ThinkingConfig             `json:"thinking,omitzero"`
	ToolConfig                 *ToolConfig                `json:"toolConfig,omitzero"`
	Tools                      ToolsConfig                `json:"tools,omitzero"`
	AgentProgressSummaries     *bool                      `json:"agentProgressSummaries,omitzero"`
	ForwardSubagentText        *bool                      `json:"forwardSubagentText,omitzero"`
	IncludeHookEvents          *bool                      `json:"includeHookEvents,omitzero"`
	LoadTimeoutMs              *int64                     `json:"loadTimeoutMs,omitzero"`
	ManagedSettings            json.RawMessage            `json:"managedSettings,omitzero"`
	PlanModeInstructions       *string                    `json:"planModeInstructions,omitzero"`
	SessionStoreFlush          *SessionStoreFlush         `json:"sessionStoreFlush,omitzero"`
	Settings                   json.RawMessage            `json:"settings,omitzero"`
	Skills                     json.RawMessage            `json:"skills,omitzero"`
	TaskBudget                 *OptionsTaskBudget         `json:"taskBudget,omitzero"`
	Title                      *string                    `json:"title,omitzero"`
	ToolAliases                map[string]string          `json:"toolAliases,omitzero"`
}

// SessionStoreFlush is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SessionStoreFlush string

const (
	SessionStoreFlushBatched SessionStoreFlush = "batched"
	SessionStoreFlushEager   SessionStoreFlush = "eager"
)

// OptionsTaskBudget is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type OptionsTaskBudget struct {
	Total int64 `json:"total"`
}

// SlashCommand is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#slashcommand
type SlashCommand struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	ArgumentHint string   `json:"argumentHint"`
	Aliases      []string `json:"aliases,omitzero"`
}

// ModelInfo is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#modelinfo
type ModelInfo struct {
	Value                    string   `json:"value"`
	ResolvedModel            *string  `json:"resolvedModel,omitzero"`
	DisplayName              string   `json:"displayName"`
	Description              string   `json:"description"`
	SupportsEffort           *bool    `json:"supportsEffort,omitzero"`
	SupportedEffortLevels    []Effort `json:"supportedEffortLevels,omitzero"`
	SupportsAdaptiveThinking *bool    `json:"supportsAdaptiveThinking,omitzero"`
	SupportsFastMode         *bool    `json:"supportsFastMode,omitzero"`
	SupportsAutoMode         *bool    `json:"supportsAutoMode,omitzero"`
}

// AgentInfo is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentinfo
type AgentInfo struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Model       *string `json:"model,omitzero"`
}

// AccountInfo is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#accountinfo
type AccountInfo struct {
	Email            *string `json:"email,omitzero"`
	Organization     *string `json:"organization,omitzero"`
	SubscriptionType *string `json:"subscriptionType,omitzero"`
	TokenSource      *string `json:"tokenSource,omitzero"`
	ApiKeySource     *string `json:"apiKeySource,omitzero"`
}

// SDKControlInitializeResponse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolinitializeresponse
type SDKControlInitializeResponse struct {
	Commands              []SlashCommand `json:"commands"`
	Agents                []AgentInfo    `json:"agents"`
	OutputStyle           string         `json:"output_style"`
	AvailableOutputStyles []string       `json:"available_output_styles"`
	Models                []ModelInfo    `json:"models"`
	Account               AccountInfo    `json:"account"`
	// FastModeState is always reported since v2.1.219; older CLIs omit it
	// when fast mode is unavailable.
	FastModeState          *FastModeState          `json:"fast_mode_state,omitzero"`
	FastModeDisabledReason *FastModeDisabledReason `json:"fast_mode_disabled_reason,omitzero"`
}

// SDKControlInterruptResponse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolinterruptresponse
type SDKControlInterruptResponse struct {
	StillQueued []string `json:"still_queued"`
	// Cancelled is only populated for an interrupt control request that set
	// cancel_queued; StillQueued is then empty.
	Cancelled []string `json:"cancelled,omitzero"`
}

// ContextUsageCategory is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageCategory struct {
	Name       string `json:"name"`
	Tokens     int64  `json:"tokens"`
	Color      string `json:"color"`
	IsDeferred *bool  `json:"isDeferred,omitzero"`
}

// ContextUsageGridSquare is a handwritten Claude Agent SDK type. It is one
// square of the usage grid /context renders; the response carries a slice of
// rows of these.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageGridSquare struct {
	Color          string  `json:"color"`
	IsFilled       bool    `json:"isFilled"`
	CategoryName   string  `json:"categoryName"`
	Tokens         int64   `json:"tokens"`
	Percentage     float64 `json:"percentage"`
	SquareFullness float64 `json:"squareFullness"`
}

// ContextUsageMemoryFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageMemoryFile struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Tokens int64  `json:"tokens"`
}

// ContextUsageMcpTool is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageMcpTool struct {
	Name       string `json:"name"`
	ServerName string `json:"serverName"`
	Tokens     int64  `json:"tokens"`
	IsLoaded   *bool  `json:"isLoaded,omitzero"`
}

// ContextUsageDeferredBuiltinTool is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageDeferredBuiltinTool struct {
	Name     string `json:"name"`
	Tokens   int64  `json:"tokens"`
	IsLoaded bool   `json:"isLoaded"`
}

// ContextUsageSystemTool is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageSystemTool struct {
	Name   string `json:"name"`
	Tokens int64  `json:"tokens"`
}

// ContextUsageSystemPromptSection is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageSystemPromptSection struct {
	Name   string `json:"name"`
	Tokens int64  `json:"tokens"`
}

// ContextUsageAgent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageAgent struct {
	AgentType string `json:"agentType"`
	Source    string `json:"source"`
	Tokens    int64  `json:"tokens"`
}

// ContextUsageSlashCommands is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageSlashCommands struct {
	TotalCommands    int64 `json:"totalCommands"`
	IncludedCommands int64 `json:"includedCommands"`
	Tokens           int64 `json:"tokens"`
}

// ContextUsageSkillFrontmatter is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageSkillFrontmatter struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Tokens int64  `json:"tokens"`
}

// ContextUsageSkills is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageSkills struct {
	TotalSkills      int64                          `json:"totalSkills"`
	IncludedSkills   int64                          `json:"includedSkills"`
	Tokens           int64                          `json:"tokens"`
	SkillFrontmatter []ContextUsageSkillFrontmatter `json:"skillFrontmatter"`
}

// ContextUsageToolCallByType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageToolCallByType struct {
	Name         string `json:"name"`
	CallTokens   int64  `json:"callTokens"`
	ResultTokens int64  `json:"resultTokens"`
}

// ContextUsageAttachmentByType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageAttachmentByType struct {
	Name   string `json:"name"`
	Tokens int64  `json:"tokens"`
}

// ContextUsageMessageBreakdown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageMessageBreakdown struct {
	ToolCallTokens          int64                          `json:"toolCallTokens"`
	ToolResultTokens        int64                          `json:"toolResultTokens"`
	AttachmentTokens        int64                          `json:"attachmentTokens"`
	AssistantMessageTokens  int64                          `json:"assistantMessageTokens"`
	UserMessageTokens       int64                          `json:"userMessageTokens"`
	RedirectedContextTokens int64                          `json:"redirectedContextTokens"`
	UnattributedTokens      int64                          `json:"unattributedTokens"`
	ToolCallsByType         []ContextUsageToolCallByType   `json:"toolCallsByType"`
	AttachmentsByType       []ContextUsageAttachmentByType `json:"attachmentsByType"`
}

// ContextUsageApiUsage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type ContextUsageApiUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// SDKControlGetContextUsageResponse is a handwritten Claude Agent SDK type. It
// is the payload /context renders, so it carries display fields such as Color,
// GridRows and Percentage alongside the token counts.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcontrolgetcontextusageresponse
type SDKControlGetContextUsageResponse struct {
	Categories   []ContextUsageCategory     `json:"categories"`
	TotalTokens  int64                      `json:"totalTokens"`
	MaxTokens    int64                      `json:"maxTokens"`
	RawMaxTokens int64                      `json:"rawMaxTokens"`
	Percentage   float64                    `json:"percentage"`
	GridRows     [][]ContextUsageGridSquare `json:"gridRows"`
	Model        string                     `json:"model"`
	MemoryFiles  []ContextUsageMemoryFile   `json:"memoryFiles"`
	McpTools     []ContextUsageMcpTool      `json:"mcpTools"`
	// DeferredBuiltinTools, SystemTools and SystemPromptSections are
	// diagnostics Claude Code leaves unset; expect them to be absent.
	DeferredBuiltinTools []ContextUsageDeferredBuiltinTool `json:"deferredBuiltinTools,omitzero"`
	SystemTools          []ContextUsageSystemTool          `json:"systemTools,omitzero"`
	SystemPromptSections []ContextUsageSystemPromptSection `json:"systemPromptSections,omitzero"`
	Agents               []ContextUsageAgent               `json:"agents"`
	SlashCommands        *ContextUsageSlashCommands        `json:"slashCommands,omitzero"`
	Skills               *ContextUsageSkills               `json:"skills,omitzero"`
	AutoCompactThreshold *float64                          `json:"autoCompactThreshold,omitzero"`
	IsAutoCompactEnabled bool                              `json:"isAutoCompactEnabled"`
	MessageBreakdown     *ContextUsageMessageBreakdown     `json:"messageBreakdown,omitzero"`
	ApiUsage             *ContextUsageApiUsage             `json:"apiUsage"`
}

// ModelUsage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#modelusage
type ModelUsage struct {
	InputTokens              int64   `json:"inputTokens"`
	OutputTokens             int64   `json:"outputTokens"`
	CacheReadInputTokens     int64   `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int64   `json:"cacheCreationInputTokens"`
	WebSearchRequests        int64   `json:"webSearchRequests"`
	CostUSD                  float64 `json:"costUSD"`
	ContextWindow            int64   `json:"contextWindow"`
	MaxOutputTokens          int64   `json:"maxOutputTokens"`
	CanonicalModel           *string `json:"canonicalModel,omitzero"`
	Provider                 *string `json:"provider,omitzero"`
}

// Usage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#usage
type Usage struct {
	InputTokens              int64               `json:"input_tokens"`
	OutputTokens             int64               `json:"output_tokens"`
	CacheCreationInputTokens *int64              `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64              `json:"cache_read_input_tokens"`
	CacheCreation            *UsageCacheCreation `json:"cache_creation"`
	ServerToolUse            json.RawMessage     `json:"server_tool_use"`
	ServiceTier              *UsageServiceTier   `json:"service_tier"`
	Speed                    *UsageSpeed         `json:"speed"`
	InferenceGeo             *string             `json:"inference_geo"`
	Iterations               json.RawMessage     `json:"iterations"`
}

// UsageCacheCreation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#usage
type UsageCacheCreation struct {
	Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens"`
	Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
}

// UsageServiceTier is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#usage
type UsageServiceTier string

const (
	UsageServiceTierStandard UsageServiceTier = "standard"
	UsageServiceTierPriority UsageServiceTier = "priority"
	UsageServiceTierBatch    UsageServiceTier = "batch"
)

// UsageSpeed is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#usage
type UsageSpeed string

const (
	UsageSpeedStandard UsageSpeed = "standard"
	UsageSpeedFast     UsageSpeed = "fast"
)

// NonNullableUsage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#nonnullableusage
type NonNullableUsage struct {
	InputTokens              int64              `json:"input_tokens"`
	OutputTokens             int64              `json:"output_tokens"`
	CacheCreationInputTokens int64              `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64              `json:"cache_read_input_tokens"`
	CacheCreation            UsageCacheCreation `json:"cache_creation"`
	ServerToolUse            json.RawMessage    `json:"server_tool_use"`
	ServiceTier              UsageServiceTier   `json:"service_tier"`
	Speed                    UsageSpeed         `json:"speed"`
	InferenceGeo             string             `json:"inference_geo"`
	Iterations               json.RawMessage    `json:"iterations"`
}

// CallToolResultContentType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#calltoolresult
type CallToolResultContentType string

const (
	CallToolResultContentTypeText         CallToolResultContentType = "text"
	CallToolResultContentTypeImage        CallToolResultContentType = "image"
	CallToolResultContentTypeAudio        CallToolResultContentType = "audio"
	CallToolResultContentTypeResource     CallToolResultContentType = "resource"
	CallToolResultContentTypeResourceLink CallToolResultContentType = "resource_link"
)

// CallToolResultContent is a handwritten Claude Agent SDK type. The docs type
// the element as { type: ... } with additional type-dependent sibling fields,
// so the whole element object is preserved raw alongside the discriminator.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#calltoolresult
type CallToolResultContent struct {
	Type CallToolResultContentType `json:"type"`
	Raw  json.RawMessage           `json:"-"`
}

// MarshalJSON emits the preserved element object, or a minimal {"type": ...}
// object when no raw payload was captured.
func (c CallToolResultContent) MarshalJSON() ([]byte, error) {
	if len(c.Raw) > 0 {
		return c.Raw, nil
	}
	type plain struct {
		Type CallToolResultContentType `json:"type"`
	}
	return json.Marshal(plain{Type: c.Type})
}

// UnmarshalJSON captures the discriminator and preserves the element verbatim.
func (c *CallToolResultContent) UnmarshalJSON(data []byte) error {
	var disc struct {
		Type CallToolResultContentType `json:"type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	c.Type = disc.Type
	c.Raw = cloneRawMessage(data)
	return nil
}

// CallToolResult is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#calltoolresult
type CallToolResult struct {
	Content           []CallToolResultContent `json:"content"`
	StructuredContent map[string]any          `json:"structuredContent,omitzero"`
	IsError           *bool                   `json:"isError,omitzero"`
}

// SDKPermissionDenial is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkpermissiondenial
type SDKPermissionDenial struct {
	ToolName  string         `json:"tool_name"`
	ToolUseID string         `json:"tool_use_id"`
	ToolInput map[string]any `json:"tool_input"`
}

// CompactMetadata is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcompactboundarymessage
type CompactMetadata struct {
	Trigger   CompactMetadataTrigger `json:"trigger"`
	PreTokens int64                  `json:"pre_tokens"`
}

// TaskUsageSummary is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktaskprogressmessage
type TaskUsageSummary struct {
	TotalTokens int64 `json:"total_tokens"`
	ToolUses    int64 `json:"tool_uses"`
	DurationMs  int64 `json:"duration_ms"`
}

// SDKMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKMessage struct {
	value SDKMessage_Value
}

// SDKMessage_Value is the variant interface implemented by every [SDKMessage] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKMessage_Value interface{ sdkMessage() }

// MarshalJSON marshals the active [SDKMessage] variant.
func (o SDKMessage) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes an [SDKMessage] union value from JSON, dispatching on
// `type` and, for system messages and results, `subtype`. Unrecognized
// type/subtype pairs are preserved through the unknown variant.
func (o *SDKMessage) UnmarshalJSON(data []byte) error {
	var disc struct {
		Type    SystemMessageType `json:"type"`
		Subtype string            `json:"subtype"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	var (
		v   SDKMessage_Value
		err error
	)
	switch disc.Type {
	case SystemMessageTypeAssistant:
		v, err = decodeUnionVariant[SDKAssistantMessage](data)
	case SystemMessageTypeUser:
		var probe struct {
			IsReplay bool `json:"isReplay"`
		}
		if e := json.Unmarshal(data, &probe); e != nil {
			return e
		}
		if probe.IsReplay {
			v, err = decodeUnionVariant[SDKUserMessageReplay](data)
		} else {
			v, err = decodeUnionVariant[SDKUserMessage](data)
		}
	case SystemMessageTypeResult:
		switch SDKResultSubtype(disc.Subtype) {
		case SDKResultSubtypeSuccess:
			v, err = decodeUnionVariant[SDKResultMessageSuccess](data)
		case SDKResultSubtypeErrorMaxTurns,
			SDKResultSubtypeErrorDuringExecution,
			SDKResultSubtypeErrorMaxBudgetUSD,
			SDKResultSubtypeErrorMaxStructuredOutputRetries:
			v, err = decodeUnionVariant[SDKResultMessageError](data)
		default:
			v, err = decodeUnionVariant[SDKResultMessageUnknown](data)
		}
	case SystemMessageTypeStreamEvent:
		v, err = decodeUnionVariant[SDKPartialAssistantMessage](data)
	case SystemMessageTypeAuthStatus:
		v, err = decodeUnionVariant[SDKAuthStatusMessage](data)
	case SystemMessageTypeToolProgress:
		v, err = decodeUnionVariant[SDKToolProgressMessage](data)
	case SystemMessageTypeRateLimitEvent:
		v, err = decodeUnionVariant[SDKRateLimitEvent](data)
	case SystemMessageTypeToolUseSummary:
		v, err = decodeUnionVariant[SDKToolUseSummaryMessage](data)
	case SystemMessageTypePromptSuggestion:
		v, err = decodeUnionVariant[SDKPromptSuggestionMessage](data)
	case SystemMessageTypeConversationReset:
		v, err = decodeUnionVariant[SDKConversationResetMessage](data)
	case SystemMessageTypeSystem:
		v, err = decodeSDKSystemMessage(disc.Subtype, data)
	default:
		v, err = decodeUnionVariant[SDKMessageUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

// decodeSDKSystemMessage decodes a type:"system" [SDKMessage] by its subtype.
func decodeSDKSystemMessage(subtype string, data []byte) (SDKMessage_Value, error) {
	switch SystemSubtype(subtype) {
	case SystemSubtypeInit:
		return decodeUnionVariant[SDKSystemMessage](data)
	case SystemSubtypeCompactBoundary:
		return decodeUnionVariant[SDKCompactBoundaryMessage](data)
	case SystemSubtypeStatus:
		return decodeUnionVariant[SDKStatusMessage](data)
	case SystemSubtypeTaskNotification:
		return decodeUnionVariant[SDKTaskNotificationMessage](data)
	case SystemSubtypeHookStarted:
		return decodeUnionVariant[SDKHookStartedMessage](data)
	case SystemSubtypeHookProgress:
		return decodeUnionVariant[SDKHookProgressMessage](data)
	case SystemSubtypeHookResponse:
		return decodeUnionVariant[SDKHookResponseMessage](data)
	case SystemSubtypeTaskStarted:
		return decodeUnionVariant[SDKTaskStartedMessage](data)
	case SystemSubtypeTaskProgress:
		return decodeUnionVariant[SDKTaskProgressMessage](data)
	case SystemSubtypeTaskUpdated:
		return decodeUnionVariant[SDKTaskUpdatedMessage](data)
	case SystemSubtypeFilesPersisted:
		return decodeUnionVariant[SDKFilesPersistedEvent](data)
	case SystemSubtypeLocalCommandOutput:
		return decodeUnionVariant[SDKLocalCommandOutputMessage](data)
	case SystemSubtypeCommandsChanged:
		return decodeUnionVariant[SDKCommandsChangedMessage](data)
	case SystemSubtypePluginInstall:
		return decodeUnionVariant[SDKPluginInstallMessage](data)
	case SystemSubtypePermissionDenied:
		return decodeUnionVariant[SDKPermissionDeniedMessage](data)
	case SystemSubtypeInformational:
		return decodeUnionVariant[SDKInformationalMessage](data)
	case SystemSubtypeWorkerShuttingDown:
		return decodeUnionVariant[SDKWorkerShuttingDownMessage](data)
	case SystemSubtypeBackgroundTasksChanged:
		return decodeUnionVariant[SDKBackgroundTasksChangedMessage](data)
	case SystemSubtypeThinkingTokens:
		return decodeUnionVariant[SDKThinkingTokensMessage](data)
	default:
		return decodeUnionVariant[SDKMessageUnknown](data)
	}
}

var (
	_ SDKMessage_Value = (*SDKMessageUnknown)(nil)
	_ SDKMessage_Value = (*SDKAssistantMessage)(nil)
	_ SDKMessage_Value = (*SDKUserMessage)(nil)
	_ SDKMessage_Value = (*SDKUserMessageReplay)(nil)
	_ SDKMessage_Value = (*SDKResultMessageUnknown)(nil)
	_ SDKMessage_Value = (*SDKResultMessageSuccess)(nil)
	_ SDKMessage_Value = (*SDKResultMessageError)(nil)
	_ SDKMessage_Value = (*SDKSystemMessage)(nil)
	_ SDKMessage_Value = (*SDKPartialAssistantMessage)(nil)
	_ SDKMessage_Value = (*SDKCompactBoundaryMessage)(nil)
	_ SDKMessage_Value = (*SDKStatusMessage)(nil)
	_ SDKMessage_Value = (*SDKLocalCommandOutputMessage)(nil)
	_ SDKMessage_Value = (*SDKHookStartedMessage)(nil)
	_ SDKMessage_Value = (*SDKHookProgressMessage)(nil)
	_ SDKMessage_Value = (*SDKHookResponseMessage)(nil)
	_ SDKMessage_Value = (*SDKToolProgressMessage)(nil)
	_ SDKMessage_Value = (*SDKAuthStatusMessage)(nil)
	_ SDKMessage_Value = (*SDKTaskNotificationMessage)(nil)
	_ SDKMessage_Value = (*SDKTaskStartedMessage)(nil)
	_ SDKMessage_Value = (*SDKTaskProgressMessage)(nil)
	_ SDKMessage_Value = (*SDKFilesPersistedEvent)(nil)
	_ SDKMessage_Value = (*SDKToolUseSummaryMessage)(nil)
	_ SDKMessage_Value = (*SDKRateLimitEvent)(nil)
	_ SDKMessage_Value = (*SDKPromptSuggestionMessage)(nil)
	_ SDKMessage_Value = (*SDKTaskUpdatedMessage)(nil)
	_ SDKMessage_Value = (*SDKCommandsChangedMessage)(nil)
	_ SDKMessage_Value = (*SDKPluginInstallMessage)(nil)
	_ SDKMessage_Value = (*SDKPermissionDeniedMessage)(nil)
	_ SDKMessage_Value = (*SDKInformationalMessage)(nil)
	_ SDKMessage_Value = (*SDKWorkerShuttingDownMessage)(nil)
	_ SDKMessage_Value = (*SDKBackgroundTasksChangedMessage)(nil)
	_ SDKMessage_Value = (*SDKThinkingTokensMessage)(nil)
	_ SDKMessage_Value = (*SDKConversationResetMessage)(nil)
	_ SDKMessage_Value = (*SDKSessionStateChangedMessage)(nil)
	_ SDKMessage_Value = (*SDKNotificationMessage)(nil)
	_ SDKMessage_Value = (*SDKMemoryRecallMessage)(nil)
	_ SDKMessage_Value = (*SDKElicitationCompleteMessage)(nil)
	_ SDKMessage_Value = (*SDKAPIRetryMessage)(nil)
	_ SDKMessage_Value = (*SDKMirrorErrorMessage)(nil)
)

// SDKMessageUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type SDKMessageUnknown struct{ UnknownUnion }

func (SDKMessageUnknown) sdkMessage() {}

// SDKAssistantMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkassistantmessage
type SDKAssistantMessage struct {
	Type            string                    `json:"type"`
	UUID            UUID                      `json:"uuid"`
	SessionID       string                    `json:"session_id"`
	Message         BetaMessage               `json:"message"`
	ParentToolUseID *string                   `json:"parent_tool_use_id"`
	Error           *SDKAssistantMessageError `json:"error,omitzero"`
	// Aborted is the literal true when the message was cut short by an abort;
	// absent on normally-completed messages.
	Aborted   *bool   `json:"aborted,omitzero"`
	Timestamp *string `json:"timestamp,omitzero"`
}

func (SDKAssistantMessage) sdkMessage() {}

// SDKUserMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkusermessage
type SDKUserMessage struct {
	Type            string            `json:"type"`
	UUID            *UUID             `json:"uuid,omitzero"`
	SessionID       *string           `json:"session_id,omitzero"`
	Message         MessageParam      `json:"message"`
	ParentToolUseID *string           `json:"parent_tool_use_id"`
	IsSynthetic     *bool             `json:"isSynthetic,omitzero"`
	ShouldQuery     *bool             `json:"shouldQuery,omitzero"`
	ToolUseResult   json.RawMessage   `json:"tool_use_result,omitzero"`
	Origin          *SDKMessageOrigin `json:"origin,omitzero"`
}

func (SDKUserMessage) sdkMessage() {}

// SDKUserMessageReplay is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkusermessagereplay
type SDKUserMessageReplay struct {
	Type            string            `json:"type"`
	UUID            UUID              `json:"uuid"`
	SessionID       string            `json:"session_id"`
	Message         MessageParam      `json:"message"`
	ParentToolUseID *string           `json:"parent_tool_use_id"`
	IsSynthetic     *bool             `json:"isSynthetic,omitzero"`
	ToolUseResult   json.RawMessage   `json:"tool_use_result,omitzero"`
	Origin          *SDKMessageOrigin `json:"origin,omitzero"`
	IsReplay        bool              `json:"isReplay"`
}

func (SDKUserMessageReplay) sdkMessage() {}

// SDKResultMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type SDKResultMessage interface {
	sdkMessage()
	sdkResultMessage()
}

// SDKResultMessageUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type SDKResultMessageUnknown struct{ UnknownUnion }

func (SDKResultMessageUnknown) sdkMessage()       {}
func (SDKResultMessageUnknown) sdkResultMessage() {}

// SDKResultMessageSuccess is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type SDKResultMessageSuccess struct {
	Type              string                `json:"type"`
	Subtype           SDKResultSubtype      `json:"subtype"`
	UUID              UUID                  `json:"uuid"`
	SessionID         string                `json:"session_id"`
	DurationMs        int64                 `json:"duration_ms"`
	DurationAPIMs     int64                 `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	APIErrorStatus    *int64                `json:"api_error_status,omitzero"`
	NumTurns          int64                 `json:"num_turns"`
	Result            string                `json:"result"`
	StopReason        *string               `json:"stop_reason"`
	TTFTMs            *int64                `json:"ttft_ms,omitzero"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             NonNullableUsage      `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []SDKPermissionDenial `json:"permission_denials"`
	StructuredOutput  json.RawMessage       `json:"structured_output,omitzero"`
	DeferredToolUse   *DeferredToolUse      `json:"deferred_tool_use,omitzero"`
	TerminalReason    *TerminalReason       `json:"terminal_reason,omitzero"`
	FastModeState     *FastModeState        `json:"fast_mode_state,omitzero"`
	Origin            *SDKMessageOrigin     `json:"origin,omitzero"`
	TTFTStreamMs      *int64                `json:"ttft_stream_ms,omitzero"`
	UserMessageUUID   *string               `json:"user_message_uuid,omitzero"`
	RequestSentWallMs *int64                `json:"request_sent_wall_ms,omitzero"`

	FastModeDisabledReason *FastModeDisabledReason `json:"fast_mode_disabled_reason,omitzero"`
}

func (SDKResultMessageSuccess) sdkMessage()       {}
func (SDKResultMessageSuccess) sdkResultMessage() {}

// SDKResultMessageError is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkresultmessage
type SDKResultMessageError struct {
	Type              string                `json:"type"`
	Subtype           SDKResultSubtype      `json:"subtype"`
	UUID              UUID                  `json:"uuid"`
	SessionID         string                `json:"session_id"`
	DurationMs        int64                 `json:"duration_ms"`
	DurationAPIMs     int64                 `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	NumTurns          int64                 `json:"num_turns"`
	StopReason        *string               `json:"stop_reason"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             NonNullableUsage      `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []SDKPermissionDenial `json:"permission_denials"`
	Errors            []string              `json:"errors"`
	TerminalReason    *TerminalReason       `json:"terminal_reason,omitzero"`
	FastModeState     *FastModeState        `json:"fast_mode_state,omitzero"`
	Origin            *SDKMessageOrigin     `json:"origin,omitzero"`

	FastModeDisabledReason *FastModeDisabledReason `json:"fast_mode_disabled_reason,omitzero"`
}

func (SDKResultMessageError) sdkMessage()       {}
func (SDKResultMessageError) sdkResultMessage() {}

// SDKSystemServer is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdksystemmessage
type SDKSystemServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// SDKSystemPlugin is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdksystemmessage
type SDKSystemPlugin struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// SDKSystemMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdksystemmessage
type SDKSystemMessage struct {
	Type              string            `json:"type"`
	Subtype           SystemSubtype     `json:"subtype"`
	UUID              UUID              `json:"uuid"`
	SessionID         string            `json:"session_id"`
	Agents            []string          `json:"agents,omitzero"`
	ApiKeySource      ApiKeySource      `json:"apiKeySource"`
	Betas             []string          `json:"betas,omitzero"`
	ClaudeCodeVersion string            `json:"claude_code_version"`
	Cwd               string            `json:"cwd"`
	Tools             []string          `json:"tools"`
	McpServers        []SDKSystemServer `json:"mcp_servers"`
	Model             string            `json:"model"`
	PermissionMode    PermissionMode    `json:"permissionMode"`
	SlashCommands     []string          `json:"slash_commands"`
	OutputStyle       string            `json:"output_style"`
	Skills            []string          `json:"skills"`
	Plugins           []SDKSystemPlugin `json:"plugins"`

	FastModeState          *FastModeState          `json:"fast_mode_state,omitzero"`
	FastModeDisabledReason *FastModeDisabledReason `json:"fast_mode_disabled_reason,omitzero"`
	Capabilities           []string                `json:"capabilities,omitzero"`
}

func (SDKSystemMessage) sdkMessage() {}

// SDKPartialAssistantMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkpartialassistantmessage
type SDKPartialAssistantMessage struct {
	Type            string                    `json:"type"`
	Event           BetaRawMessageStreamEvent `json:"event"`
	ParentToolUseID *string                   `json:"parent_tool_use_id"`
	UUID            UUID                      `json:"uuid"`
	SessionID       string                    `json:"session_id"`
	// TTFTMs is present only on message_start events.
	TTFTMs *int64 `json:"ttft_ms,omitzero"`
}

func (SDKPartialAssistantMessage) sdkMessage() {}

// SDKCompactBoundaryMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcompactboundarymessage
type SDKCompactBoundaryMessage struct {
	Type            string          `json:"type"`
	Subtype         SystemSubtype   `json:"subtype"`
	UUID            UUID            `json:"uuid"`
	SessionID       string          `json:"session_id"`
	CompactMetadata CompactMetadata `json:"compact_metadata"`
}

func (SDKCompactBoundaryMessage) sdkMessage() {}

// SDKStatusMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkstatusmessage
type SDKStatusMessage struct {
	Type           string          `json:"type"`
	Subtype        SystemSubtype   `json:"subtype"`
	Status         *string         `json:"status"`
	PermissionMode *PermissionMode `json:"permissionMode,omitzero"`
	UUID           UUID            `json:"uuid"`
	SessionID      string          `json:"session_id"`
}

func (SDKStatusMessage) sdkMessage() {}

// SDKLocalCommandOutputMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdklocalcommandoutputmessage
type SDKLocalCommandOutputMessage struct {
	Type      string        `json:"type"`
	Subtype   SystemSubtype `json:"subtype"`
	Content   string        `json:"content"`
	UUID      UUID          `json:"uuid"`
	SessionID string        `json:"session_id"`
}

func (SDKLocalCommandOutputMessage) sdkMessage() {}

// SDKHookStartedMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkhookstartedmessage
type SDKHookStartedMessage struct {
	Type      string        `json:"type"`
	Subtype   SystemSubtype `json:"subtype"`
	HookID    string        `json:"hook_id"`
	HookName  string        `json:"hook_name"`
	HookEvent string        `json:"hook_event"`
	UUID      UUID          `json:"uuid"`
	SessionID string        `json:"session_id"`
}

func (SDKHookStartedMessage) sdkMessage() {}

// SDKHookProgressMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkhookprogressmessage
type SDKHookProgressMessage struct {
	Type      string        `json:"type"`
	Subtype   SystemSubtype `json:"subtype"`
	HookID    string        `json:"hook_id"`
	HookName  string        `json:"hook_name"`
	HookEvent string        `json:"hook_event"`
	Stdout    string        `json:"stdout"`
	Stderr    string        `json:"stderr"`
	Output    string        `json:"output"`
	UUID      UUID          `json:"uuid"`
	SessionID string        `json:"session_id"`
}

func (SDKHookProgressMessage) sdkMessage() {}

// SDKHookResponseMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkhookresponsemessage
type SDKHookResponseMessage struct {
	Type      string              `json:"type"`
	Subtype   SystemSubtype       `json:"subtype"`
	HookID    string              `json:"hook_id"`
	HookName  string              `json:"hook_name"`
	HookEvent string              `json:"hook_event"`
	Output    string              `json:"output"`
	Stdout    string              `json:"stdout"`
	Stderr    string              `json:"stderr"`
	ExitCode  *int64              `json:"exit_code,omitzero"`
	Outcome   HookResponseOutcome `json:"outcome"`
	UUID      UUID                `json:"uuid"`
	SessionID string              `json:"session_id"`
}

func (SDKHookResponseMessage) sdkMessage() {}

// SDKToolProgressMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktoolprogressmessage
type SDKToolProgressMessage struct {
	Type               string                        `json:"type"`
	ToolUseID          string                        `json:"tool_use_id"`
	ToolName           string                        `json:"tool_name"`
	ParentToolUseID    *string                       `json:"parent_tool_use_id"`
	ElapsedTimeSeconds float64                       `json:"elapsed_time_seconds"`
	TaskID             *string                       `json:"task_id,omitzero"`
	Heartbeat          *bool                         `json:"heartbeat,omitzero"`
	SubagentType       *string                       `json:"subagent_type,omitzero"`
	SubagentRetry      *SDKToolProgressSubagentRetry `json:"subagent_retry,omitzero"`
	UUID               UUID                          `json:"uuid"`
	SessionID          string                        `json:"session_id"`
}

// SDKToolProgressSubagentRetry is a handwritten Claude Agent SDK type.
// ErrorCategory is typed as a plain string upstream; the documented tokens are
// rate_limit, overloaded, authentication_failed, server_error, and unknown.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktoolprogressmessage
type SDKToolProgressSubagentRetry struct {
	AgentID       string `json:"agent_id"`
	Attempt       int64  `json:"attempt"`
	MaxRetries    int64  `json:"max_retries"`
	RetryDelayMs  int64  `json:"retry_delay_ms"`
	ErrorStatus   *int64 `json:"error_status"`
	ErrorCategory string `json:"error_category"`
}

func (SDKToolProgressMessage) sdkMessage() {}

// SDKAuthStatusMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkauthstatusmessage
type SDKAuthStatusMessage struct {
	Type             string   `json:"type"`
	IsAuthenticating bool     `json:"isAuthenticating"`
	Output           []string `json:"output"`
	Error            *string  `json:"error,omitzero"`
	UUID             UUID     `json:"uuid"`
	SessionID        string   `json:"session_id"`
}

func (SDKAuthStatusMessage) sdkMessage() {}

// SDKTaskNotificationMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktasknotificationmessage
type SDKTaskNotificationMessage struct {
	Type       string                 `json:"type"`
	Subtype    SystemSubtype          `json:"subtype"`
	TaskID     string                 `json:"task_id"`
	ToolUseID  *string                `json:"tool_use_id,omitzero"`
	Status     TaskNotificationStatus `json:"status"`
	OutputFile string                 `json:"output_file"`
	Summary    string                 `json:"summary"`
	Usage      *TaskUsageSummary      `json:"usage,omitzero"`
	UUID       UUID                   `json:"uuid"`
	SessionID  string                 `json:"session_id"`
}

func (SDKTaskNotificationMessage) sdkMessage() {}

// SDKTaskStartedMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktaskstartedmessage
type SDKTaskStartedMessage struct {
	Type        string        `json:"type"`
	Subtype     SystemSubtype `json:"subtype"`
	TaskID      string        `json:"task_id"`
	ToolUseID   *string       `json:"tool_use_id,omitzero"`
	Description string        `json:"description"`
	TaskType    *string       `json:"task_type,omitzero"`
	UUID        UUID          `json:"uuid"`
	SessionID   string        `json:"session_id"`
}

func (SDKTaskStartedMessage) sdkMessage() {}

// SDKTaskProgressMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktaskprogressmessage
type SDKTaskProgressMessage struct {
	Type         string           `json:"type"`
	Subtype      SystemSubtype    `json:"subtype"`
	TaskID       string           `json:"task_id"`
	ToolUseID    *string          `json:"tool_use_id,omitzero"`
	Description  string           `json:"description"`
	SubagentType *string          `json:"subagent_type,omitzero"`
	Usage        TaskUsageSummary `json:"usage"`
	LastToolName *string          `json:"last_tool_name,omitzero"`
	Summary      *string          `json:"summary,omitzero"`
	UUID         UUID             `json:"uuid"`
	SessionID    string           `json:"session_id"`
}

func (SDKTaskProgressMessage) sdkMessage() {}

// SDKTaskUpdatedPatchStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktaskupdatedmessage
type SDKTaskUpdatedPatchStatus string

const (
	SDKTaskUpdatedPatchStatusPending   SDKTaskUpdatedPatchStatus = "pending"
	SDKTaskUpdatedPatchStatusRunning   SDKTaskUpdatedPatchStatus = "running"
	SDKTaskUpdatedPatchStatusCompleted SDKTaskUpdatedPatchStatus = "completed"
	SDKTaskUpdatedPatchStatusFailed    SDKTaskUpdatedPatchStatus = "failed"
	SDKTaskUpdatedPatchStatusKilled    SDKTaskUpdatedPatchStatus = "killed"
)

// SDKTaskUpdatedPatch is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktaskupdatedmessage
type SDKTaskUpdatedPatch struct {
	Status         *SDKTaskUpdatedPatchStatus `json:"status,omitzero"`
	Description    *string                    `json:"description,omitzero"`
	EndTime        *int64                     `json:"end_time,omitzero"`
	TotalPausedMs  *int64                     `json:"total_paused_ms,omitzero"`
	Error          *string                    `json:"error,omitzero"`
	IsBackgrounded *bool                      `json:"is_backgrounded,omitzero"`
}

// SDKTaskUpdatedMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktaskupdatedmessage
type SDKTaskUpdatedMessage struct {
	Type      string              `json:"type"`
	Subtype   SystemSubtype       `json:"subtype"`
	TaskID    string              `json:"task_id"`
	Patch     SDKTaskUpdatedPatch `json:"patch"`
	UUID      UUID                `json:"uuid"`
	SessionID string              `json:"session_id"`
}

func (SDKTaskUpdatedMessage) sdkMessage() {}

// SDKCommandsChangedMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkcommandschangedmessage
type SDKCommandsChangedMessage struct {
	Type      string         `json:"type"`
	Subtype   SystemSubtype  `json:"subtype"`
	Commands  []SlashCommand `json:"commands"`
	UUID      UUID           `json:"uuid"`
	SessionID string         `json:"session_id"`
}

func (SDKCommandsChangedMessage) sdkMessage() {}

// SDKPluginInstallStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkplugininstallmessage
type SDKPluginInstallStatus string

const (
	SDKPluginInstallStatusStarted   SDKPluginInstallStatus = "started"
	SDKPluginInstallStatusInstalled SDKPluginInstallStatus = "installed"
	SDKPluginInstallStatusFailed    SDKPluginInstallStatus = "failed"
	SDKPluginInstallStatusCompleted SDKPluginInstallStatus = "completed"
)

// SDKPluginInstallMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkplugininstallmessage
type SDKPluginInstallMessage struct {
	Type      string                 `json:"type"`
	Subtype   SystemSubtype          `json:"subtype"`
	Status    SDKPluginInstallStatus `json:"status"`
	Name      *string                `json:"name,omitzero"`
	Error     *string                `json:"error,omitzero"`
	UUID      UUID                   `json:"uuid"`
	SessionID string                 `json:"session_id"`
}

func (SDKPluginInstallMessage) sdkMessage() {}

// SDKPermissionDeniedMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkpermissiondeniedmessage
type SDKPermissionDeniedMessage struct {
	Type               string        `json:"type"`
	Subtype            SystemSubtype `json:"subtype"`
	ToolName           string        `json:"tool_name"`
	ToolUseID          string        `json:"tool_use_id"`
	AgentID            *string       `json:"agent_id,omitzero"`
	DecisionReasonType *string       `json:"decision_reason_type,omitzero"`
	DecisionReason     *string       `json:"decision_reason,omitzero"`
	Message            string        `json:"message"`
	UUID               UUID          `json:"uuid"`
	SessionID          string        `json:"session_id"`
}

func (SDKPermissionDeniedMessage) sdkMessage() {}

// SDKFilesPersistedEvent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkfilespersistedevent
type SDKFilesPersistedEvent struct {
	Type        string                  `json:"type"`
	Subtype     SystemSubtype           `json:"subtype"`
	Files       []FilesPersistedFile    `json:"files"`
	Failed      []FilesPersistedFailure `json:"failed"`
	ProcessedAt string                  `json:"processed_at"`
	UUID        UUID                    `json:"uuid"`
	SessionID   string                  `json:"session_id"`
}

func (SDKFilesPersistedEvent) sdkMessage() {}

// SDKToolUseSummaryMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdktoolusesummarymessage
type SDKToolUseSummaryMessage struct {
	Type                string   `json:"type"`
	Summary             string   `json:"summary"`
	PrecedingToolUseIDs []string `json:"preceding_tool_use_ids"`
	UUID                UUID     `json:"uuid"`
	SessionID           string   `json:"session_id"`
}

func (SDKToolUseSummaryMessage) sdkMessage() {}

// SDKRateLimitEvent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkratelimitevent
type SDKRateLimitEvent struct {
	Type          string        `json:"type"`
	RateLimitInfo RateLimitInfo `json:"rate_limit_info"`
	UUID          UUID          `json:"uuid"`
	SessionID     string        `json:"session_id"`
}

func (SDKRateLimitEvent) sdkMessage() {}

// SDKPromptSuggestionMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkpromptsuggestionmessage
type SDKPromptSuggestionMessage struct {
	Type       string `json:"type"`
	Suggestion string `json:"suggestion"`
	UUID       UUID   `json:"uuid"`
	SessionID  string `json:"session_id"`
}

func (SDKPromptSuggestionMessage) sdkMessage() {}

// SDKInformationalMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkinformationalmessage
type SDKInformationalMessage struct {
	Type                string                       `json:"type"`
	Subtype             SystemSubtype                `json:"subtype"`
	Content             string                       `json:"content"`
	Level               SDKInformationalMessageLevel `json:"level"`
	ToolUseID           *string                      `json:"tool_use_id,omitzero"`
	PreventContinuation *bool                        `json:"prevent_continuation,omitzero"`
	UUID                UUID                         `json:"uuid"`
	SessionID           string                       `json:"session_id"`
}

func (SDKInformationalMessage) sdkMessage() {}

// SDKWorkerShuttingDownMessage is a handwritten Claude Agent SDK type. Reason
// is a free-form snake_case string such as "host_exit" or
// "remote_control_disabled".
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkworkershuttingdownmessage
type SDKWorkerShuttingDownMessage struct {
	Type      string        `json:"type"`
	Subtype   SystemSubtype `json:"subtype"`
	Reason    string        `json:"reason"`
	UUID      UUID          `json:"uuid"`
	SessionID string        `json:"session_id"`
}

func (SDKWorkerShuttingDownMessage) sdkMessage() {}

// SDKBackgroundTasksChangedTask is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkbackgroundtaskschangedmessage
type SDKBackgroundTasksChangedTask struct {
	TaskID      string `json:"task_id"`
	TaskType    string `json:"task_type"`
	Description string `json:"description"`
}

// SDKBackgroundTasksChangedMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkbackgroundtaskschangedmessage
type SDKBackgroundTasksChangedMessage struct {
	Type      string                          `json:"type"`
	Subtype   SystemSubtype                   `json:"subtype"`
	Tasks     []SDKBackgroundTasksChangedTask `json:"tasks"`
	UUID      UUID                            `json:"uuid"`
	SessionID string                          `json:"session_id"`
}

func (SDKBackgroundTasksChangedMessage) sdkMessage() {}

// SDKThinkingTokensMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkthinkingtokensmessage
type SDKThinkingTokensMessage struct {
	Type                 string        `json:"type"`
	Subtype              SystemSubtype `json:"subtype"`
	EstimatedTokens      int64         `json:"estimated_tokens"`
	EstimatedTokensDelta int64         `json:"estimated_tokens_delta"`
	UUID                 UUID          `json:"uuid"`
	SessionID            string        `json:"session_id"`
}

func (SDKThinkingTokensMessage) sdkMessage() {}

// SDKConversationResetMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkconversationresetmessage
type SDKConversationResetMessage struct {
	Type              string `json:"type"`
	NewConversationID UUID   `json:"new_conversation_id"`
	UUID              UUID   `json:"uuid"`
	SessionID         string `json:"session_id"`
}

func (SDKConversationResetMessage) sdkMessage() {}

// SDKSessionStateChangedMessage is a handwritten Claude Agent SDK type. The
// docs name it as an SDKMessage member without defining it on the page, so the
// payload is preserved as raw JSON; decoding lands in [SDKMessageUnknown].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKSessionStateChangedMessage json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m SDKSessionStateChangedMessage) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *SDKSessionStateChangedMessage) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func (SDKSessionStateChangedMessage) sdkMessage() {}

// SDKNotificationMessage is a handwritten Claude Agent SDK type. The docs name
// it as an SDKMessage member without defining it on the page, so the payload is
// preserved as raw JSON; decoding lands in [SDKMessageUnknown].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKNotificationMessage json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m SDKNotificationMessage) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *SDKNotificationMessage) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func (SDKNotificationMessage) sdkMessage() {}

// SDKMemoryRecallMessage is a handwritten Claude Agent SDK type. The docs name
// it as an SDKMessage member without defining it on the page, so the payload is
// preserved as raw JSON; decoding lands in [SDKMessageUnknown].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKMemoryRecallMessage json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m SDKMemoryRecallMessage) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *SDKMemoryRecallMessage) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func (SDKMemoryRecallMessage) sdkMessage() {}

// SDKElicitationCompleteMessage is a handwritten Claude Agent SDK type. The
// docs name it as an SDKMessage member without defining it on the page, so the
// payload is preserved as raw JSON; decoding lands in [SDKMessageUnknown].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKElicitationCompleteMessage json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m SDKElicitationCompleteMessage) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *SDKElicitationCompleteMessage) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func (SDKElicitationCompleteMessage) sdkMessage() {}

// SDKAPIRetryMessage is a handwritten Claude Agent SDK type. The docs name it
// as an SDKMessage member without defining it on the page, so the payload is
// preserved as raw JSON; decoding lands in [SDKMessageUnknown].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKAPIRetryMessage json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m SDKAPIRetryMessage) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *SDKAPIRetryMessage) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func (SDKAPIRetryMessage) sdkMessage() {}

// SDKMirrorErrorMessage is a handwritten Claude Agent SDK type. The docs name
// it as an SDKMessage member without defining it on the page, so the payload is
// preserved as raw JSON; decoding lands in [SDKMessageUnknown].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkmessage
type SDKMirrorErrorMessage json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m SDKMirrorErrorMessage) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *SDKMirrorErrorMessage) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func (SDKMirrorErrorMessage) sdkMessage() {}

// BaseHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#basehookinput
type BaseHookInput struct {
	SessionID      string               `json:"session_id"`
	TranscriptPath string               `json:"transcript_path"`
	Cwd            string               `json:"cwd"`
	PromptID       *string              `json:"prompt_id,omitzero"`
	PermissionMode *string              `json:"permission_mode,omitzero"`
	Effort         *BaseHookInputEffort `json:"effort,omitzero"`
	AgentID        *string              `json:"agent_id,omitzero"`
	AgentType      *string              `json:"agent_type,omitzero"`
}

// BaseHookInputEffort is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#basehookinput
type BaseHookInputEffort struct {
	Level string `json:"level"`
}

// BackgroundTaskSummary is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#subagentstophookinput
type BackgroundTaskSummary struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Status      string  `json:"status"`
	Description string  `json:"description"`
	Command     *string `json:"command,omitzero"`
	AgentType   *string `json:"agent_type,omitzero"`
	Server      *string `json:"server,omitzero"`
	Tool        *string `json:"tool,omitzero"`
	Name        *string `json:"name,omitzero"`
}

// SessionCronSummary is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#subagentstophookinput
type SessionCronSummary struct {
	ID        string `json:"id"`
	Schedule  string `json:"schedule"`
	Recurring bool   `json:"recurring"`
	Prompt    string `json:"prompt"`
}

// HookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookinput
type HookInput struct {
	value HookInput_Value
}

// HookInput_Value is the variant interface implemented by every [HookInput] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookinput
type HookInput_Value interface{ hookInput() }

// MarshalJSON marshals the active [HookInput] variant.
func (o HookInput) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [HookInput] union value from JSON.
func (o *HookInput) UnmarshalJSON(data []byte) error {
	var disc struct {
		HookEventName HookEvent `json:"hook_event_name"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	switch disc.HookEventName {
	case HookEventPreToolUse:
		var v PreToolUseHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventPostToolUse:
		var v PostToolUseHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventPostToolUseFailure:
		var v PostToolUseFailureHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventNotification:
		var v NotificationHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventUserPromptSubmit:
		var v UserPromptSubmitHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventSessionStart:
		var v SessionStartHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventSessionEnd:
		var v SessionEndHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventStop:
		var v StopHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventSubagentStart:
		var v SubagentStartHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventSubagentStop:
		var v SubagentStopHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventPreCompact:
		var v PreCompactHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventPermissionRequest:
		var v PermissionRequestHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventSetup:
		var v SetupHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventTeammateIdle:
		var v TeammateIdleHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventTaskCompleted:
		var v TaskCompletedHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventConfigChange:
		var v ConfigChangeHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventWorktreeCreate:
		var v WorktreeCreateHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventWorktreeRemove:
		var v WorktreeRemoveHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventPostToolBatch:
		var v PostToolBatchHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventMessageDisplay:
		var v MessageDisplayHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventPermissionDenied:
		var v PermissionDeniedHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventUserPromptExpansion:
		var v UserPromptExpansionHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventStopFailure:
		var v StopFailureHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventPostCompact:
		var v PostCompactHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventTaskCreated:
		var v TaskCreatedHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventElicitation:
		var v ElicitationHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventElicitationResult:
		var v ElicitationResultHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventInstructionsLoaded:
		var v InstructionsLoadedHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventDirectoryAdded:
		var v DirectoryAddedHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventCwdChanged:
		var v CwdChangedHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case HookEventFileChanged:
		var v FileChangedHookInput
		o.value = &v
		return json.Unmarshal(data, &v)
	default:
		var v HookInputUnknown
		o.value = &v
		return v.UnmarshalJSON(data)
	}
}

var (
	_ HookInput_Value = (*HookInputUnknown)(nil)
	_ HookInput_Value = (*PreToolUseHookInput)(nil)
	_ HookInput_Value = (*PostToolUseHookInput)(nil)
	_ HookInput_Value = (*PostToolUseFailureHookInput)(nil)
	_ HookInput_Value = (*NotificationHookInput)(nil)
	_ HookInput_Value = (*UserPromptSubmitHookInput)(nil)
	_ HookInput_Value = (*SessionStartHookInput)(nil)
	_ HookInput_Value = (*SessionEndHookInput)(nil)
	_ HookInput_Value = (*StopHookInput)(nil)
	_ HookInput_Value = (*SubagentStartHookInput)(nil)
	_ HookInput_Value = (*SubagentStopHookInput)(nil)
	_ HookInput_Value = (*PreCompactHookInput)(nil)
	_ HookInput_Value = (*PermissionRequestHookInput)(nil)
	_ HookInput_Value = (*SetupHookInput)(nil)
	_ HookInput_Value = (*TeammateIdleHookInput)(nil)
	_ HookInput_Value = (*TaskCompletedHookInput)(nil)
	_ HookInput_Value = (*ConfigChangeHookInput)(nil)
	_ HookInput_Value = (*WorktreeCreateHookInput)(nil)
	_ HookInput_Value = (*WorktreeRemoveHookInput)(nil)
	_ HookInput_Value = (*PostToolBatchHookInput)(nil)
	_ HookInput_Value = (*MessageDisplayHookInput)(nil)
	_ HookInput_Value = (*PermissionDeniedHookInput)(nil)
	_ HookInput_Value = (*UserPromptExpansionHookInput)(nil)
	_ HookInput_Value = (*StopFailureHookInput)(nil)
	_ HookInput_Value = (*PostCompactHookInput)(nil)
	_ HookInput_Value = (*TaskCreatedHookInput)(nil)
	_ HookInput_Value = (*ElicitationHookInput)(nil)
	_ HookInput_Value = (*ElicitationResultHookInput)(nil)
	_ HookInput_Value = (*InstructionsLoadedHookInput)(nil)
	_ HookInput_Value = (*DirectoryAddedHookInput)(nil)
	_ HookInput_Value = (*CwdChangedHookInput)(nil)
	_ HookInput_Value = (*FileChangedHookInput)(nil)
)

// HookInputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookinput
type HookInputUnknown struct{ UnknownUnion }

func (HookInputUnknown) hookInput() {}

// PreToolUseHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#pretoolusehookinput
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEvent        `json:"hook_event_name"`
	ToolName      string           `json:"tool_name"`
	ToolInput     ToolInputSchemas `json:"tool_input"`
	ToolUseID     string           `json:"tool_use_id"`
}

func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#posttoolusehookinput
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEvent         `json:"hook_event_name"`
	ToolName      string            `json:"tool_name"`
	ToolInput     ToolInputSchemas  `json:"tool_input"`
	ToolResponse  ToolOutputSchemas `json:"tool_response"`
	ToolUseID     string            `json:"tool_use_id"`
	DurationMs    *int64            `json:"duration_ms,omitzero"`
}

func (PostToolUseHookInput) hookInput() {}

// PostToolUseFailureHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#posttoolusefailurehookinput
type PostToolUseFailureHookInput struct {
	BaseHookInput
	HookEventName HookEvent        `json:"hook_event_name"`
	ToolName      string           `json:"tool_name"`
	ToolInput     ToolInputSchemas `json:"tool_input"`
	ToolUseID     string           `json:"tool_use_id"`
	Error         string           `json:"error"`
	IsInterrupt   *bool            `json:"is_interrupt,omitzero"`
	DurationMs    *int64           `json:"duration_ms,omitzero"`
}

func (PostToolUseFailureHookInput) hookInput() {}

// PostToolBatchHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#posttoolbatchhookinput
type PostToolBatchHookInput struct {
	BaseHookInput
	HookEventName HookEvent               `json:"hook_event_name"`
	ToolCalls     []PostToolBatchToolCall `json:"tool_calls"`
}

func (PostToolBatchHookInput) hookInput() {}

// PostToolBatchToolCall is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#posttoolbatchhookinput
type PostToolBatchToolCall struct {
	ToolName     string          `json:"tool_name"`
	ToolInput    json.RawMessage `json:"tool_input"`
	ToolUseID    string          `json:"tool_use_id"`
	ToolResponse json.RawMessage `json:"tool_response,omitzero"`
}

// NotificationHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#notificationhookinput
type NotificationHookInput struct {
	BaseHookInput
	HookEventName    HookEvent `json:"hook_event_name"`
	Message          string    `json:"message"`
	Title            *string   `json:"title,omitzero"`
	NotificationType string    `json:"notification_type"`
}

func (NotificationHookInput) hookInput() {}

// UserPromptSubmitHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#userpromptsubmithookinput
type UserPromptSubmitHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	Prompt        string    `json:"prompt"`
	SessionTitle  *string   `json:"session_title,omitzero"`
}

func (UserPromptSubmitHookInput) hookInput() {}

// UserPromptExpansionHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#userpromptexpansionhookinput
type UserPromptExpansionHookInput struct {
	BaseHookInput
	HookEventName HookEvent               `json:"hook_event_name"`
	ExpansionType UserPromptExpansionType `json:"expansion_type"`
	CommandName   string                  `json:"command_name"`
	CommandArgs   string                  `json:"command_args"`
	CommandSource *string                 `json:"command_source,omitzero"`
	Prompt        string                  `json:"prompt"`
}

func (UserPromptExpansionHookInput) hookInput() {}

// SessionStartHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sessionstarthookinput
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName HookEvent          `json:"hook_event_name"`
	Source        SessionStartSource `json:"source"`
	Model         *string            `json:"model,omitzero"`
	SessionTitle  *string            `json:"session_title,omitzero"`
}

func (SessionStartHookInput) hookInput() {}

// SessionEndHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sessionendhookinput
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName HookEvent  `json:"hook_event_name"`
	Reason        ExitReason `json:"reason"`
}

func (SessionEndHookInput) hookInput() {}

// StopHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#stophookinput
type StopHookInput struct {
	BaseHookInput
	HookEventName        HookEvent               `json:"hook_event_name"`
	StopHookActive       bool                    `json:"stop_hook_active"`
	LastAssistantMessage *string                 `json:"last_assistant_message,omitzero"`
	BackgroundTasks      []BackgroundTaskSummary `json:"background_tasks,omitzero"`
	SessionCrons         []SessionCronSummary    `json:"session_crons,omitzero"`
}

func (StopHookInput) hookInput() {}

// StopFailureHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#stopfailurehookinput
type StopFailureHookInput struct {
	BaseHookInput
	HookEventName        HookEvent                `json:"hook_event_name"`
	Error                SDKAssistantMessageError `json:"error"`
	ErrorDetails         *string                  `json:"error_details,omitzero"`
	LastAssistantMessage *string                  `json:"last_assistant_message,omitzero"`
}

func (StopFailureHookInput) hookInput() {}

// SubagentStartHookInput is a handwritten Claude Agent SDK type. AgentID and
// AgentType shadow the embedded optional base fields because this variant
// requires them.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#subagentstarthookinput
type SubagentStartHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	AgentID       string    `json:"agent_id"`
	AgentType     string    `json:"agent_type"`
}

func (SubagentStartHookInput) hookInput() {}

// SubagentStopHookInput is a handwritten Claude Agent SDK type. AgentID and
// AgentType shadow the embedded optional base fields because this variant
// requires them.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#subagentstophookinput
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName        HookEvent               `json:"hook_event_name"`
	StopHookActive       bool                    `json:"stop_hook_active"`
	AgentID              string                  `json:"agent_id"`
	AgentTranscriptPath  string                  `json:"agent_transcript_path"`
	AgentType            string                  `json:"agent_type"`
	LastAssistantMessage *string                 `json:"last_assistant_message,omitzero"`
	BackgroundTasks      []BackgroundTaskSummary `json:"background_tasks,omitzero"`
	SessionCrons         []SessionCronSummary    `json:"session_crons,omitzero"`
}

func (SubagentStopHookInput) hookInput() {}

// PreCompactHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#precompacthookinput
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      HookEvent         `json:"hook_event_name"`
	Trigger            PreCompactTrigger `json:"trigger"`
	CustomInstructions *string           `json:"custom_instructions"`
}

func (PreCompactHookInput) hookInput() {}

// PostCompactHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#postcompacthookinput
type PostCompactHookInput struct {
	BaseHookInput
	HookEventName  HookEvent          `json:"hook_event_name"`
	Trigger        PostCompactTrigger `json:"trigger"`
	CompactSummary string             `json:"compact_summary"`
}

func (PostCompactHookInput) hookInput() {}

// PermissionRequestHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionrequesthookinput
type PermissionRequestHookInput struct {
	BaseHookInput
	HookEventName         HookEvent          `json:"hook_event_name"`
	ToolName              string             `json:"tool_name"`
	ToolInput             ToolInputSchemas   `json:"tool_input"`
	PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitzero"`
}

func (PermissionRequestHookInput) hookInput() {}

// PermissionDeniedHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissiondeniedhookinput
type PermissionDeniedHookInput struct {
	BaseHookInput
	HookEventName HookEvent        `json:"hook_event_name"`
	ToolName      string           `json:"tool_name"`
	ToolInput     ToolInputSchemas `json:"tool_input"`
	ToolUseID     string           `json:"tool_use_id"`
	Reason        string           `json:"reason"`
}

func (PermissionDeniedHookInput) hookInput() {}

// SetupHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#setuphookinput
type SetupHookInput struct {
	BaseHookInput
	HookEventName HookEvent    `json:"hook_event_name"`
	Trigger       SetupTrigger `json:"trigger"`
}

func (SetupHookInput) hookInput() {}

// TeammateIdleHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#teammateidlehookinput
type TeammateIdleHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	TeammateName  string    `json:"teammate_name"`
	// Deprecated: since v2.1.178. Carries the session-derived team name; will
	// be removed.
	TeamName string `json:"team_name"`
}

func (TeammateIdleHookInput) hookInput() {}

// TaskCreatedHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskcreatedhookinput
type TaskCreatedHookInput struct {
	BaseHookInput
	HookEventName   HookEvent `json:"hook_event_name"`
	TaskID          string    `json:"task_id"`
	TaskSubject     string    `json:"task_subject"`
	TaskDescription *string   `json:"task_description,omitzero"`
	TeammateName    *string   `json:"teammate_name,omitzero"`
	// Deprecated: since v2.1.178. Carries the session-derived team name; will
	// be removed.
	TeamName *string `json:"team_name,omitzero"`
}

func (TaskCreatedHookInput) hookInput() {}

// TaskCompletedHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskcompletedhookinput
type TaskCompletedHookInput struct {
	BaseHookInput
	HookEventName   HookEvent `json:"hook_event_name"`
	TaskID          string    `json:"task_id"`
	TaskSubject     string    `json:"task_subject"`
	TaskDescription *string   `json:"task_description,omitzero"`
	TeammateName    *string   `json:"teammate_name,omitzero"`
	// Deprecated: since v2.1.178. Carries the session-derived team name; will
	// be removed.
	TeamName *string `json:"team_name,omitzero"`
}

func (TaskCompletedHookInput) hookInput() {}

// ElicitationHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#elicitationhookinput
type ElicitationHookInput struct {
	BaseHookInput
	HookEventName   HookEvent        `json:"hook_event_name"`
	McpServerName   string           `json:"mcp_server_name"`
	Message         string           `json:"message"`
	Mode            *ElicitationMode `json:"mode,omitzero"`
	URL             *string          `json:"url,omitzero"`
	ElicitationID   *string          `json:"elicitation_id,omitzero"`
	RequestedSchema map[string]any   `json:"requested_schema,omitzero"`
}

func (ElicitationHookInput) hookInput() {}

// ElicitationResultHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#elicitationresulthookinput
type ElicitationResultHookInput struct {
	BaseHookInput
	HookEventName HookEvent               `json:"hook_event_name"`
	McpServerName string                  `json:"mcp_server_name"`
	ElicitationID *string                 `json:"elicitation_id,omitzero"`
	Mode          *ElicitationResultMode  `json:"mode,omitzero"`
	Action        ElicitationResultAction `json:"action"`
	Content       map[string]any          `json:"content,omitzero"`
}

func (ElicitationResultHookInput) hookInput() {}

// InstructionsLoadedHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#instructionsloadedhookinput
type InstructionsLoadedHookInput struct {
	BaseHookInput
	HookEventName   HookEvent              `json:"hook_event_name"`
	FilePath        string                 `json:"file_path"`
	MemoryType      InstructionsMemoryType `json:"memory_type"`
	LoadReason      InstructionsLoadReason `json:"load_reason"`
	Globs           []string               `json:"globs,omitzero"`
	TriggerFilePath *string                `json:"trigger_file_path,omitzero"`
	ParentFilePath  *string                `json:"parent_file_path,omitzero"`
}

func (InstructionsLoadedHookInput) hookInput() {}

// DirectoryAddedHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#directoryaddedhookinput
type DirectoryAddedHookInput struct {
	BaseHookInput
	HookEventName HookEvent            `json:"hook_event_name"`
	Directory     string               `json:"directory"`
	Source        DirectoryAddedSource `json:"source"`
}

func (DirectoryAddedHookInput) hookInput() {}

// CwdChangedHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#cwdchangedhookinput
type CwdChangedHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	OldCwd        string    `json:"old_cwd"`
	NewCwd        string    `json:"new_cwd"`
}

func (CwdChangedHookInput) hookInput() {}

// FileChangedHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#filechangedhookinput
type FileChangedHookInput struct {
	BaseHookInput
	HookEventName HookEvent        `json:"hook_event_name"`
	FilePath      string           `json:"file_path"`
	Event         FileChangedEvent `json:"event"`
}

func (FileChangedHookInput) hookInput() {}

// ConfigChangeHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#configchangehookinput
type ConfigChangeHookInput struct {
	BaseHookInput
	HookEventName HookEvent          `json:"hook_event_name"`
	Source        ConfigChangeSource `json:"source"`
	FilePath      *string            `json:"file_path,omitzero"`
}

func (ConfigChangeHookInput) hookInput() {}

// WorktreeCreateHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#worktreecreatehookinput
type WorktreeCreateHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	Name          string    `json:"name"`
}

func (WorktreeCreateHookInput) hookInput() {}

// WorktreeRemoveHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#worktreeremovehookinput
type WorktreeRemoveHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	WorktreePath  string    `json:"worktree_path"`
}

func (WorktreeRemoveHookInput) hookInput() {}

// MessageDisplayHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#messagedisplayhookinput
type MessageDisplayHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	TurnID        string    `json:"turn_id"`
	MessageID     string    `json:"message_id"`
	Index         int64     `json:"index"`
	Final         bool      `json:"final"`
	Delta         string    `json:"delta"`
}

func (MessageDisplayHookInput) hookInput() {}

// HookJSONOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookjsonoutput
type HookJSONOutput struct {
	value HookJSONOutput_Value
}

// HookJSONOutput_Value is the variant interface implemented by every [HookJSONOutput] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookjsonoutput
type HookJSONOutput_Value interface{ hookJSONOutput() }

// MarshalJSON marshals the active [HookJSONOutput] variant.
func (o HookJSONOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [HookJSONOutput] union value from JSON.
func (o *HookJSONOutput) UnmarshalJSON(data []byte) error {
	var probe struct {
		Async *bool `json:"async"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	if probe.Async != nil && *probe.Async {
		var v AsyncHookJSONOutput
		o.value = &v
		return v.UnmarshalJSON(data)
	}
	var syncV SyncHookJSONOutput
	if err := syncV.UnmarshalJSON(
		data,
	); err == nil &&
		(syncV.Continue != nil || syncV.SuppressOutput != nil || syncV.StopReason != nil || syncV.Decision != nil || syncV.SystemMessage != nil || syncV.TerminalSequence != nil || syncV.Reason != nil || syncV.HookSpecificOutput.GetValue() != nil) {
		o.value = &syncV
		return nil
	}
	var unknown HookJSONOutputUnknown
	o.value = &unknown
	return unknown.UnmarshalJSON(data)
}

var (
	_ HookJSONOutput_Value = (*HookJSONOutputUnknown)(nil)
	_ HookJSONOutput_Value = (*AsyncHookJSONOutput)(nil)
	_ HookJSONOutput_Value = (*SyncHookJSONOutput)(nil)
)

// HookJSONOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookjsonoutput
type HookJSONOutputUnknown struct{ UnknownUnion }

func (HookJSONOutputUnknown) hookJSONOutput() {}

// AsyncHookJSONOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#asynchookjsonoutput
type AsyncHookJSONOutput struct {
	Async        bool   `json:"async"`
	AsyncTimeout *int64 `json:"asyncTimeout,omitzero"`
}

func (AsyncHookJSONOutput) hookJSONOutput() {}

// HookSpecificOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutput struct {
	value HookSpecificOutput_Value
}

// HookSpecificOutput_Value is the variant interface implemented by every [HookSpecificOutput] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutput_Value interface{ hookSpecificOutput() }

// MarshalJSON marshals the active [HookSpecificOutput] variant.
func (o HookSpecificOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [HookSpecificOutput] union value from JSON.
func (o *HookSpecificOutput) UnmarshalJSON(data []byte) error {
	var disc struct {
		HookEventName HookEvent `json:"hookEventName"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	switch disc.HookEventName {
	case HookEventPreToolUse:
		var v HookSpecificOutputPreToolUse
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventUserPromptSubmit:
		var v HookSpecificOutputUserPromptSubmit
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventSessionStart:
		var v HookSpecificOutputSessionStart
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventSetup:
		var v HookSpecificOutputSetup
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventSubagentStart:
		var v HookSpecificOutputSubagentStart
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventPostToolUse:
		var v HookSpecificOutputPostToolUse
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventPostToolUseFailure:
		var v HookSpecificOutputPostToolUseFailure
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventPostToolBatch:
		var v HookSpecificOutputPostToolBatch
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventNotification:
		var v HookSpecificOutputNotification
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventPermissionRequest:
		var v HookSpecificOutputPermissionRequest
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventUserPromptExpansion:
		var v HookSpecificOutputUserPromptExpansion
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventStop:
		var v HookSpecificOutputStop
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventSubagentStop:
		var v HookSpecificOutputSubagentStop
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventPermissionDenied:
		var v HookSpecificOutputPermissionDenied
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventElicitation:
		var v HookSpecificOutputElicitation
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventElicitationResult:
		var v HookSpecificOutputElicitationResult
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventCwdChanged:
		var v HookSpecificOutputCwdChanged
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventFileChanged:
		var v HookSpecificOutputFileChanged
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventWorktreeCreate:
		var v HookSpecificOutputWorktreeCreate
		o.value = &v
		return v.UnmarshalJSON(data)
	case HookEventMessageDisplay:
		var v HookSpecificOutputMessageDisplay
		o.value = &v
		return v.UnmarshalJSON(data)
	default:
		var v HookSpecificOutputUnknown
		o.value = &v
		return v.UnmarshalJSON(data)
	}
}

var (
	_ HookSpecificOutput_Value = (*HookSpecificOutputUnknown)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputPreToolUse)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputUserPromptSubmit)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputSessionStart)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputSetup)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputSubagentStart)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputPostToolUse)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputPostToolUseFailure)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputPostToolBatch)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputNotification)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputPermissionRequest)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputUserPromptExpansion)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputStop)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputSubagentStop)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputPermissionDenied)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputElicitation)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputElicitationResult)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputCwdChanged)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputFileChanged)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputWorktreeCreate)(nil)
	_ HookSpecificOutput_Value = (*HookSpecificOutputMessageDisplay)(nil)
)

// HookSpecificOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputUnknown struct{ UnknownUnion }

func (HookSpecificOutputUnknown) hookSpecificOutput() {}

// HookSpecificOutputPreToolUse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputPreToolUse struct {
	HookEventName            HookEvent               `json:"hookEventName"`
	PermissionDecision       *HookPermissionDecision `json:"permissionDecision,omitzero"`
	PermissionDecisionReason *string                 `json:"permissionDecisionReason,omitzero"`
	UpdatedInput             map[string]any          `json:"updatedInput,omitzero"`
	AdditionalContext        *string                 `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputPreToolUse) hookSpecificOutput() {}

// HookSpecificOutputUserPromptSubmit is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputUserPromptSubmit struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
	SessionTitle      *string   `json:"sessionTitle,omitzero"`
	// SuppressOriginalPrompt omits the original prompt from the block message
	// when decision is "block".
	SuppressOriginalPrompt *bool `json:"suppressOriginalPrompt,omitzero"`
}

func (HookSpecificOutputUserPromptSubmit) hookSpecificOutput() {}

// HookSpecificOutputUserPromptExpansion is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputUserPromptExpansion struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputUserPromptExpansion) hookSpecificOutput() {}

// HookSpecificOutputSessionStart is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputSessionStart struct {
	HookEventName      HookEvent `json:"hookEventName"`
	AdditionalContext  *string   `json:"additionalContext,omitzero"`
	InitialUserMessage *string   `json:"initialUserMessage,omitzero"`
	SessionTitle       *string   `json:"sessionTitle,omitzero"`
	WatchPaths         []string  `json:"watchPaths,omitzero"`
	// ReloadSkills re-scans skill and command directories after SessionStart
	// hooks complete, so skills installed by the hook are available in the
	// same session.
	ReloadSkills *bool `json:"reloadSkills,omitzero"`
}

func (HookSpecificOutputSessionStart) hookSpecificOutput() {}

// HookSpecificOutputSetup is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputSetup struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputSetup) hookSpecificOutput() {}

// HookSpecificOutputSubagentStart is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputSubagentStart struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputSubagentStart) hookSpecificOutput() {}

// HookSpecificOutputPostToolUse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputPostToolUse struct {
	HookEventName     HookEvent       `json:"hookEventName"`
	AdditionalContext *string         `json:"additionalContext,omitzero"`
	UpdatedToolOutput json.RawMessage `json:"updatedToolOutput,omitzero"`
	// UpdatedMCPToolOutput is deprecated; use UpdatedToolOutput, which works for all tools.
	UpdatedMCPToolOutput json.RawMessage `json:"updatedMCPToolOutput,omitzero"`
}

func (HookSpecificOutputPostToolUse) hookSpecificOutput() {}

// HookSpecificOutputPostToolUseFailure is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputPostToolUseFailure struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputPostToolUseFailure) hookSpecificOutput() {}

// HookSpecificOutputPostToolBatch is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputPostToolBatch struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputPostToolBatch) hookSpecificOutput() {}

// HookSpecificOutputNotification is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputNotification struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputNotification) hookSpecificOutput() {}

// HookSpecificOutputStop is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputStop struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputStop) hookSpecificOutput() {}

// HookSpecificOutputSubagentStop is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputSubagentStop struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputSubagentStop) hookSpecificOutput() {}

// HookSpecificOutputPermissionDenied is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputPermissionDenied struct {
	HookEventName HookEvent `json:"hookEventName"`
	Retry         *bool     `json:"retry,omitzero"`
}

func (HookSpecificOutputPermissionDenied) hookSpecificOutput() {}

// HookSpecificOutputElicitationAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputElicitationAction string

const (
	HookSpecificOutputElicitationActionAccept  HookSpecificOutputElicitationAction = "accept"
	HookSpecificOutputElicitationActionDecline HookSpecificOutputElicitationAction = "decline"
	HookSpecificOutputElicitationActionCancel  HookSpecificOutputElicitationAction = "cancel"
)

// HookSpecificOutputElicitation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputElicitation struct {
	HookEventName HookEvent                            `json:"hookEventName"`
	Action        *HookSpecificOutputElicitationAction `json:"action,omitzero"`
	Content       map[string]any                       `json:"content,omitzero"`
}

func (HookSpecificOutputElicitation) hookSpecificOutput() {}

// HookSpecificOutputElicitationResultAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputElicitationResultAction string

const (
	HookSpecificOutputElicitationResultActionAccept  HookSpecificOutputElicitationResultAction = "accept"
	HookSpecificOutputElicitationResultActionDecline HookSpecificOutputElicitationResultAction = "decline"
	HookSpecificOutputElicitationResultActionCancel  HookSpecificOutputElicitationResultAction = "cancel"
)

// HookSpecificOutputElicitationResult is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputElicitationResult struct {
	HookEventName HookEvent                                  `json:"hookEventName"`
	Action        *HookSpecificOutputElicitationResultAction `json:"action,omitzero"`
	Content       map[string]any                             `json:"content,omitzero"`
}

func (HookSpecificOutputElicitationResult) hookSpecificOutput() {}

// HookSpecificOutputCwdChanged is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputCwdChanged struct {
	HookEventName HookEvent `json:"hookEventName"`
	WatchPaths    []string  `json:"watchPaths,omitzero"`
}

func (HookSpecificOutputCwdChanged) hookSpecificOutput() {}

// HookSpecificOutputFileChanged is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputFileChanged struct {
	HookEventName HookEvent `json:"hookEventName"`
	WatchPaths    []string  `json:"watchPaths,omitzero"`
}

func (HookSpecificOutputFileChanged) hookSpecificOutput() {}

// HookSpecificOutputWorktreeCreate is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputWorktreeCreate struct {
	HookEventName HookEvent `json:"hookEventName"`
	WorktreePath  string    `json:"worktreePath"`
}

func (HookSpecificOutputWorktreeCreate) hookSpecificOutput() {}

// HookSpecificOutputMessageDisplay is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputMessageDisplay struct {
	HookEventName HookEvent `json:"hookEventName"`
	// DisplayContent is the text displayed in place of the delta. Omit (or
	// return the delta unchanged) to display the original.
	DisplayContent *string `json:"displayContent,omitzero"`
}

func (HookSpecificOutputMessageDisplay) hookSpecificOutput() {}

// PermissionRequestDecision is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type PermissionRequestDecision struct {
	value PermissionRequestDecision_Value
}

// PermissionRequestDecision_Value is the variant interface implemented by every
// [PermissionRequestDecision] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type PermissionRequestDecision_Value interface{ permissionRequestDecision() }

// MarshalJSON marshals the active [PermissionRequestDecision] variant.
func (o PermissionRequestDecision) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [PermissionRequestDecision] union value from JSON.
func (o *PermissionRequestDecision) UnmarshalJSON(data []byte) error {
	var disc struct {
		Behavior PermissionRequestDecisionBehavior `json:"behavior"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	switch disc.Behavior {
	case PermissionRequestDecisionBehaviorAllow:
		var v PermissionRequestDecisionAllow
		o.value = &v
		return v.UnmarshalJSON(data)
	case PermissionRequestDecisionBehaviorDeny:
		var v PermissionRequestDecisionDeny
		o.value = &v
		return v.UnmarshalJSON(data)
	default:
		var v PermissionRequestDecisionUnknown
		o.value = &v
		return v.UnmarshalJSON(data)
	}
}

var (
	_ PermissionRequestDecision_Value = (*PermissionRequestDecisionUnknown)(nil)
	_ PermissionRequestDecision_Value = (*PermissionRequestDecisionAllow)(nil)
	_ PermissionRequestDecision_Value = (*PermissionRequestDecisionDeny)(nil)
)

// PermissionRequestDecisionUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type PermissionRequestDecisionUnknown struct{ UnknownUnion }

func (PermissionRequestDecisionUnknown) permissionRequestDecision() {}

// PermissionRequestDecisionAllow is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type PermissionRequestDecisionAllow struct {
	Behavior           PermissionRequestDecisionBehavior `json:"behavior"`
	UpdatedInput       map[string]any                    `json:"updatedInput,omitzero"`
	UpdatedPermissions []PermissionUpdate                `json:"updatedPermissions,omitzero"`
}

func (PermissionRequestDecisionAllow) permissionRequestDecision() {}

// PermissionRequestDecisionDeny is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type PermissionRequestDecisionDeny struct {
	Behavior  PermissionRequestDecisionBehavior `json:"behavior"`
	Message   *string                           `json:"message,omitzero"`
	Interrupt *bool                             `json:"interrupt,omitzero"`
}

func (PermissionRequestDecisionDeny) permissionRequestDecision() {}

// HookSpecificOutputPermissionRequest is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type HookSpecificOutputPermissionRequest struct {
	HookEventName HookEvent                 `json:"hookEventName"`
	Decision      PermissionRequestDecision `json:"decision"`
}

func (HookSpecificOutputPermissionRequest) hookSpecificOutput() {}

// SyncHookJSONOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#synchookjsonoutput
type SyncHookJSONOutput struct {
	Continue       *bool         `json:"continue,omitzero"`
	SuppressOutput *bool         `json:"suppressOutput,omitzero"`
	StopReason     *string       `json:"stopReason,omitzero"`
	Decision       *HookDecision `json:"decision,omitzero"`
	SystemMessage  *string       `json:"systemMessage,omitzero"`
	// TerminalSequence is a terminal escape sequence (e.g. OSC 9 / OSC 777
	// desktop-notification) for Claude Code to emit on your behalf. Only
	// notification/title OSCs (0, 1, 2, 9, 99, 777) and BEL are permitted; a
	// value containing anything else is ignored as a whole.
	TerminalSequence   *string            `json:"terminalSequence,omitzero"`
	Reason             *string            `json:"reason,omitzero"`
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput,omitzero"`
}

func (SyncHookJSONOutput) hookJSONOutput() {}

// ToolInputSchemas is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolinputschemas
type ToolInputSchemas struct {
	value ToolInputSchemas_Value
}

// ToolInputSchemas_Value is the variant interface implemented by every [ToolInputSchemas] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolinputschemas
type ToolInputSchemas_Value interface{ toolInputSchemas() }

// MarshalJSON marshals the active [ToolInputSchemas] variant.
func (o ToolInputSchemas) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalForTool decodes data into the [ToolInputSchemas] variant selected by toolName.
func (o *ToolInputSchemas) UnmarshalForTool(toolName string, data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if strings.HasPrefix(toolName, McpToolNamePrefix) {
		var v McpInput
		o.value = &v
		return json.Unmarshal(data, &v)
	}
	switch toolName {
	case ToolNameAskUserQuestion:
		var v AskUserQuestionInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameBash:
		var v BashInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameEdit:
		var v FileEditInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameEnterWorktree:
		var v EnterWorktreeInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameExitPlanMode:
		var v ExitPlanModeInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameGlob:
		var v GlobInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameGrep:
		var v GrepInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameListMcpResources:
		var v ListMcpResourcesInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameNotebookEdit:
		var v NotebookEditInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameRead:
		var v FileReadInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameReadMcpResource:
		var v ReadMcpResourceInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameReadMcpResourceDir:
		var v ReadMcpResourceDirInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameRefreshMcpTools:
		var v RefreshMcpToolsInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameAgent, ToolNameTask:
		var v AgentInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameMonitor:
		var v MonitorInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWorkflow:
		var v WorkflowInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskCreate:
		var v TaskCreateInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskUpdate:
		var v TaskUpdateInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskGet:
		var v TaskGetInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskList:
		var v TaskListInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskOutput:
		var v TaskOutputInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskStop:
		var v TaskStopInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTodoWrite:
		var v TodoWriteInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWebFetch:
		var v WebFetchInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWebSearch:
		var v WebSearchInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWrite:
		var v FileWriteInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameArtifact:
		var v ArtifactInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameCronCreate:
		var v CronCreateInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameCronDelete:
		var v CronDeleteInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameCronList:
		var v CronListInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameEnterPlanMode:
		var v EnterPlanModeInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameExitWorktree:
		var v ExitWorktreeInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameProjects:
		var v ProjectsInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNamePushNotification:
		var v PushNotificationInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameRemoteTrigger:
		var v RemoteTriggerInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameREPL:
		var v REPLInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameReportFindings:
		var v ReportFindingsInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameScheduleWakeup:
		var v ScheduleWakeupInput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameShowOnboardingRolePicker:
		var v ShowOnboardingRolePickerInput
		o.value = &v
		return json.Unmarshal(data, &v)
	default:
		if done, err := o.unmarshalCodexInputForTool(toolName, data); done {
			return err
		}
		var v ToolInputUnknown
		o.value = &v
		return v.UnmarshalJSON(data)
	}
}

var (
	_ ToolInputSchemas_Value = (*ToolInputUnknown)(nil)
	_ ToolInputSchemas_Value = (*AgentInput)(nil)
	_ ToolInputSchemas_Value = (*ArtifactInput)(nil)
	_ ToolInputSchemas_Value = (*AskUserQuestionInput)(nil)
	_ ToolInputSchemas_Value = (*BashInput)(nil)
	_ ToolInputSchemas_Value = (*TaskOutputInput)(nil)
	_ ToolInputSchemas_Value = (*CronCreateInput)(nil)
	_ ToolInputSchemas_Value = (*CronDeleteInput)(nil)
	_ ToolInputSchemas_Value = (*CronListInput)(nil)
	_ ToolInputSchemas_Value = (*EnterPlanModeInput)(nil)
	_ ToolInputSchemas_Value = (*EnterWorktreeInput)(nil)
	_ ToolInputSchemas_Value = (*ExitPlanModeInput)(nil)
	_ ToolInputSchemas_Value = (*ExitWorktreeInput)(nil)
	_ ToolInputSchemas_Value = (*FileEditInput)(nil)
	_ ToolInputSchemas_Value = (*FileReadInput)(nil)
	_ ToolInputSchemas_Value = (*FileWriteInput)(nil)
	_ ToolInputSchemas_Value = (*GlobInput)(nil)
	_ ToolInputSchemas_Value = (*GrepInput)(nil)
	_ ToolInputSchemas_Value = (*ListMcpResourcesInput)(nil)
	_ ToolInputSchemas_Value = (*McpInput)(nil)
	_ ToolInputSchemas_Value = (*MonitorInput)(nil)
	_ ToolInputSchemas_Value = (*ProjectsInput)(nil)
	_ ToolInputSchemas_Value = (*PushNotificationInput)(nil)
	_ ToolInputSchemas_Value = (*WorkflowInput)(nil)
	_ ToolInputSchemas_Value = (*TaskCreateInput)(nil)
	_ ToolInputSchemas_Value = (*TaskUpdateInput)(nil)
	_ ToolInputSchemas_Value = (*TaskGetInput)(nil)
	_ ToolInputSchemas_Value = (*TaskListInput)(nil)
	_ ToolInputSchemas_Value = (*NotebookEditInput)(nil)
	_ ToolInputSchemas_Value = (*ReadMcpResourceInput)(nil)
	_ ToolInputSchemas_Value = (*ReadMcpResourceDirInput)(nil)
	_ ToolInputSchemas_Value = (*RefreshMcpToolsInput)(nil)
	_ ToolInputSchemas_Value = (*RemoteTriggerInput)(nil)
	_ ToolInputSchemas_Value = (*REPLInput)(nil)
	_ ToolInputSchemas_Value = (*ReportFindingsInput)(nil)
	_ ToolInputSchemas_Value = (*ScheduleWakeupInput)(nil)
	_ ToolInputSchemas_Value = (*ShowOnboardingRolePickerInput)(nil)
	_ ToolInputSchemas_Value = (*TaskStopInput)(nil)
	_ ToolInputSchemas_Value = (*TodoWriteInput)(nil)
	_ ToolInputSchemas_Value = (*WebFetchInput)(nil)
	_ ToolInputSchemas_Value = (*WebSearchInput)(nil)
)

// ToolInputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolinputschemas
type ToolInputUnknown struct{ UnknownUnion }

func (ToolInputUnknown) toolInputSchemas() {}

// AgentInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent
type AgentInput struct {
	Description     string      `json:"description"`
	Prompt          string      `json:"prompt"`
	SubagentType    *string     `json:"subagent_type,omitzero"`
	Model           *AgentModel `json:"model,omitzero"`
	RunInBackground *bool       `json:"run_in_background,omitzero"`
	Name            *string     `json:"name,omitzero"`
	// Deprecated: ignored.
	TeamName *string `json:"team_name,omitzero"`
	// Deprecated: ignored.
	Mode      *AgentMode      `json:"mode,omitzero"`
	Isolation *AgentIsolation `json:"isolation,omitzero"`
}

func (AgentInput) toolInputSchemas() {}

// MonitorInputWs is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#monitor
type MonitorInputWs struct {
	URL       string   `json:"url"`
	Protocols []string `json:"protocols,omitzero"`
}

// MonitorInput is a handwritten Claude Agent SDK type. TimeoutMs and
// Persistent are required in the exported TS type because the schema fills in
// their defaults (300000 and false); a call that omits them validates.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#monitor
type MonitorInput struct {
	Command     *string         `json:"command,omitzero"`
	Ws          *MonitorInputWs `json:"ws,omitzero"`
	Description string          `json:"description"`
	TimeoutMs   int64           `json:"timeout_ms"`
	Persistent  bool            `json:"persistent"`
}

func (MonitorInput) toolInputSchemas() {}

// WorkflowInput is a handwritten Claude Agent SDK type. Title and Description
// are ignored by the runtime; the script's meta block sets them.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#workflow
type WorkflowInput struct {
	Script          *string         `json:"script,omitzero"`
	Name            *string         `json:"name,omitzero"`
	ScriptPath      *string         `json:"scriptPath,omitzero"`
	Args            json.RawMessage `json:"args,omitzero"`
	ResumeFromRunID *string         `json:"resumeFromRunId,omitzero"`
	Title           *string         `json:"title,omitzero"`
	Description     *string         `json:"description,omitzero"`
}

func (WorkflowInput) toolInputSchemas() {}

// TaskCreateInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskcreate
type TaskCreateInput struct {
	Subject     string         `json:"subject"`
	Description string         `json:"description"`
	ActiveForm  *string        `json:"activeForm,omitzero"`
	Metadata    map[string]any `json:"metadata,omitzero"`
}

func (TaskCreateInput) toolInputSchemas() {}

// TaskUpdateStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskupdate
type TaskUpdateStatus string

const (
	TaskUpdateStatusPending    TaskUpdateStatus = "pending"
	TaskUpdateStatusInProgress TaskUpdateStatus = "in_progress"
	TaskUpdateStatusCompleted  TaskUpdateStatus = "completed"
	TaskUpdateStatusDeleted    TaskUpdateStatus = "deleted"
)

// TaskUpdateInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskupdate
type TaskUpdateInput struct {
	TaskID       string            `json:"taskId"`
	Status       *TaskUpdateStatus `json:"status,omitzero"`
	Subject      *string           `json:"subject,omitzero"`
	Description  *string           `json:"description,omitzero"`
	ActiveForm   *string           `json:"activeForm,omitzero"`
	AddBlocks    []string          `json:"addBlocks,omitzero"`
	AddBlockedBy []string          `json:"addBlockedBy,omitzero"`
	Owner        *string           `json:"owner,omitzero"`
	Metadata     map[string]any    `json:"metadata,omitzero"`
}

func (TaskUpdateInput) toolInputSchemas() {}

// TaskGetInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskget
type TaskGetInput struct {
	TaskID string `json:"taskId"`
}

func (TaskGetInput) toolInputSchemas() {}

// TaskListInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tasklist
type TaskListInput struct{}

func (TaskListInput) toolInputSchemas() {}

// AskUserQuestionInputOption is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion
type AskUserQuestionInputOption struct {
	Label       string  `json:"label"`
	Description string  `json:"description"`
	Preview     *string `json:"preview,omitzero"`
}

// AskUserQuestionInputQuestion is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion
type AskUserQuestionInputQuestion struct {
	Question    string                       `json:"question"`
	Header      string                       `json:"header"`
	Options     []AskUserQuestionInputOption `json:"options"`
	MultiSelect bool                         `json:"multiSelect"`
}

// AskUserQuestionAnnotation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion
type AskUserQuestionAnnotation struct {
	Preview *string `json:"preview,omitzero"`
	Notes   *string `json:"notes,omitzero"`
}

// AskUserQuestionMetadata is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion
type AskUserQuestionMetadata struct {
	Source *string `json:"source,omitzero"`
}

// AskUserQuestionInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion
type AskUserQuestionInput struct {
	Questions   []AskUserQuestionInputQuestion       `json:"questions"`
	Answers     map[string]string                    `json:"answers,omitzero"`
	Annotations map[string]AskUserQuestionAnnotation `json:"annotations,omitzero"`
	Metadata    *AskUserQuestionMetadata             `json:"metadata,omitzero"`
}

func (AskUserQuestionInput) toolInputSchemas() {}

// BashInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash
type BashInput struct {
	Command                   string  `json:"command"`
	Timeout                   *int64  `json:"timeout,omitzero"`
	Description               *string `json:"description,omitzero"`
	RunInBackground           *bool   `json:"run_in_background,omitzero"`
	DangerouslyDisableSandbox *bool   `json:"dangerouslyDisableSandbox,omitzero"`
}

func (BashInput) toolInputSchemas() {}

// TaskOutputInput is a handwritten Claude Agent SDK type.
//
// Deprecated: the TaskOutput tool is deprecated since Claude Code v2.1.83;
// prefer Read on the task's output file path. The schema stays valid for hooks
// and permission handlers that still encounter the tool.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskoutput
type TaskOutputInput struct {
	TaskID  string `json:"task_id"`
	Block   bool   `json:"block"`
	Timeout int64  `json:"timeout"`
}

func (TaskOutputInput) toolInputSchemas() {}

// EnterWorktreeInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#enterworktree
type EnterWorktreeInput struct {
	Name *string `json:"name,omitzero"`
	Path *string `json:"path,omitzero"`
}

func (EnterWorktreeInput) toolInputSchemas() {}

// AllowedPromptTool is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitplanmode
type AllowedPromptTool string

const (
	AllowedPromptToolBash AllowedPromptTool = "Bash"
)

// AllowedPrompt is a handwritten Claude Agent SDK type.
//
// Deprecated: no longer used upstream.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitplanmode
type AllowedPrompt struct {
	Tool   AllowedPromptTool `json:"tool"`
	Prompt string            `json:"prompt"`
}

// ExitPlanModeInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitplanmode
type ExitPlanModeInput struct {
	AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitzero"`
	// Extra preserves the TS index-signature keys ([k: string]: unknown).
	Extra map[string]json.RawMessage `json:"-"`
}

// MarshalJSON merges the typed field with the preserved index-signature keys.
func (o ExitPlanModeInput) MarshalJSON() ([]byte, error) {
	out := make(map[string]json.RawMessage, len(o.Extra)+1)
	maps.Copy(out, o.Extra)
	if len(o.AllowedPrompts) > 0 {
		b, err := json.Marshal(o.AllowedPrompts)
		if err != nil {
			return nil, err
		}
		out["allowedPrompts"] = b
	}
	return json.Marshal(out)
}

// UnmarshalJSON splits the typed field from the preserved index-signature keys.
func (o *ExitPlanModeInput) UnmarshalJSON(data []byte) error {
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	*o = ExitPlanModeInput{}
	if raw, ok := all["allowedPrompts"]; ok {
		if err := json.Unmarshal(raw, &o.AllowedPrompts); err != nil {
			return err
		}
		delete(all, "allowedPrompts")
	}
	if len(all) > 0 {
		o.Extra = all
	}
	return nil
}

func (ExitPlanModeInput) toolInputSchemas() {}

// FileEditInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#edit
type FileEditInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll *bool  `json:"replace_all,omitzero"`
}

func (FileEditInput) toolInputSchemas() {}

// FileReadInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read
type FileReadInput struct {
	FilePath string  `json:"file_path"`
	Offset   *int64  `json:"offset,omitzero"`
	Limit    *int64  `json:"limit,omitzero"`
	Pages    *string `json:"pages,omitzero"`
}

func (FileReadInput) toolInputSchemas() {}

// FileWriteInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#write
type FileWriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

func (FileWriteInput) toolInputSchemas() {}

// GlobInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#glob
type GlobInput struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitzero"`
}

func (GlobInput) toolInputSchemas() {}

// GrepInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#grep
type GrepInput struct {
	Pattern         string          `json:"pattern"`
	Path            *string         `json:"path,omitzero"`
	Glob            *string         `json:"glob,omitzero"`
	Type            *string         `json:"type,omitzero"`
	OutputMode      *GrepOutputMode `json:"output_mode,omitzero"`
	CaseInsensitive *bool           `json:"-i,omitzero"`
	ShowLineNumbers *bool           `json:"-n,omitzero"`
	BeforeContext   *int64          `json:"-B,omitzero"`
	AfterContext    *int64          `json:"-A,omitzero"`
	Context         *int64          `json:"-C,omitzero"`
	ContextLines    *int64          `json:"context,omitzero"`
	HeadLimit       *int64          `json:"head_limit,omitzero"`
	Offset          *int64          `json:"offset,omitzero"`
	Multiline       *bool           `json:"multiline,omitzero"`
	// OnlyMatching prints only the matched parts of each line; requires
	// output_mode "content".
	OnlyMatching *bool `json:"-o,omitzero"`
}

func (GrepInput) toolInputSchemas() {}

// ListMcpResourcesInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#listmcpresources
type ListMcpResourcesInput struct {
	Server *string `json:"server,omitzero"`
}

func (ListMcpResourcesInput) toolInputSchemas() {}

// McpInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpinput
type McpInput map[string]any

func (McpInput) toolInputSchemas() {}

// NotebookEditInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#notebookedit
type NotebookEditInput struct {
	NotebookPath string            `json:"notebook_path"`
	CellID       *string           `json:"cell_id,omitzero"`
	NewSource    string            `json:"new_source"`
	CellType     *NotebookCellType `json:"cell_type,omitzero"`
	EditMode     *NotebookEditMode `json:"edit_mode,omitzero"`
}

func (NotebookEditInput) toolInputSchemas() {}

// ReadMcpResourceInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresource
type ReadMcpResourceInput struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}

func (ReadMcpResourceInput) toolInputSchemas() {}

// ReadMcpResourceDirInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresourcedir
type ReadMcpResourceDirInput struct {
	Server string `json:"server"`
	URI    string `json:"uri"`
}

func (ReadMcpResourceDirInput) toolInputSchemas() {}

// RefreshMcpToolsInput is a handwritten Claude Agent SDK type. Server refreshes
// only that server; omit it to refresh all connected servers.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#refreshmcptools
type RefreshMcpToolsInput struct {
	Server *string `json:"server,omitzero"`
}

func (RefreshMcpToolsInput) toolInputSchemas() {}

// TaskStopInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskstop
type TaskStopInput struct {
	TaskID *string `json:"task_id,omitzero"`
	// Deprecated: use TaskID.
	ShellID *string `json:"shell_id,omitzero"`
}

func (TaskStopInput) toolInputSchemas() {}

// TodoItem is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#todowrite
type TodoItem struct {
	Content    string     `json:"content"`
	Status     TodoStatus `json:"status"`
	ActiveForm string     `json:"activeForm"`
}

// TodoWriteInput is a handwritten Claude Agent SDK type.
//
// Deprecated: as of TypeScript Agent SDK 0.3.142 TodoWrite is disabled by
// default; use TaskCreate, TaskGet, TaskUpdate and TaskList instead.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#todowrite
type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

func (TodoWriteInput) toolInputSchemas() {}

// WebFetchInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#webfetch
type WebFetchInput struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

func (WebFetchInput) toolInputSchemas() {}

// WebSearchInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch
type WebSearchInput struct {
	Query          string   `json:"query"`
	AllowedDomains []string `json:"allowed_domains,omitzero"`
	BlockedDomains []string `json:"blocked_domains,omitzero"`
}

func (WebSearchInput) toolInputSchemas() {}

// ArtifactAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact
type ArtifactAction string

const (
	ArtifactActionPublish ArtifactAction = "publish"
	ArtifactActionList    ArtifactAction = "list"
)

// ArtifactScope is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact
type ArtifactScope string

const (
	ArtifactScopeMine   ArtifactScope = "mine"
	ArtifactScopeShared ArtifactScope = "shared"
	ArtifactScopeAll    ArtifactScope = "all"
)

// ArtifactInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact
type ArtifactInput struct {
	Action      *ArtifactAction `json:"action,omitzero"`
	FilePath    *string         `json:"file_path,omitzero"`
	Favicon     *string         `json:"favicon,omitzero"`
	Limit       *int64          `json:"limit,omitzero"`
	Scope       *ArtifactScope  `json:"scope,omitzero"`
	Title       *string         `json:"title,omitzero"`
	Description *string         `json:"description,omitzero"`
	Label       *string         `json:"label,omitzero"`
	URL         *string         `json:"url,omitzero"`
	Force       *bool           `json:"force,omitzero"`
}

func (ArtifactInput) toolInputSchemas() {}

// CronCreateInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#croncreate
type CronCreateInput struct {
	Cron      string `json:"cron"`
	Prompt    string `json:"prompt"`
	Recurring *bool  `json:"recurring,omitzero"`
	Durable   *bool  `json:"durable,omitzero"`
}

func (CronCreateInput) toolInputSchemas() {}

// CronDeleteInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#crondelete
type CronDeleteInput struct {
	ID string `json:"id"`
}

func (CronDeleteInput) toolInputSchemas() {}

// CronListInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#cronlist
type CronListInput struct{}

func (CronListInput) toolInputSchemas() {}

// EnterPlanModeInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#enterplanmode
type EnterPlanModeInput struct{}

func (EnterPlanModeInput) toolInputSchemas() {}

// ExitWorktreeAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitworktree
type ExitWorktreeAction string

const (
	ExitWorktreeActionKeep   ExitWorktreeAction = "keep"
	ExitWorktreeActionRemove ExitWorktreeAction = "remove"
)

// ExitWorktreeInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitworktree
type ExitWorktreeInput struct {
	Action         ExitWorktreeAction `json:"action"`
	DiscardChanges *bool              `json:"discard_changes,omitzero"`
}

func (ExitWorktreeInput) toolInputSchemas() {}

// ProjectsMethod is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects
type ProjectsMethod string

const (
	ProjectsMethodProjectInfo   ProjectsMethod = "project_info"
	ProjectsMethodProjectRead   ProjectsMethod = "project_read"
	ProjectsMethodProjectSearch ProjectsMethod = "project_search"
	ProjectsMethodProjectWrite  ProjectsMethod = "project_write"
	ProjectsMethodProjectDelete ProjectsMethod = "project_delete"
)

// ProjectsInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects
type ProjectsInput struct {
	Method        ProjectsMethod `json:"method"`
	Path          *string        `json:"path,omitzero"`
	Content       *string        `json:"content,omitzero"`
	LocalPath     *string        `json:"local_path,omitzero"`
	PresentToUser *bool          `json:"present_to_user,omitzero"`
	Query         *string        `json:"query,omitzero"`
	N             *int64         `json:"n,omitzero"`
}

func (ProjectsInput) toolInputSchemas() {}

// PushNotificationStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#pushnotification
type PushNotificationStatus string

const (
	PushNotificationStatusProactive PushNotificationStatus = "proactive"
)

// PushNotificationInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#pushnotification
type PushNotificationInput struct {
	Message string                 `json:"message"`
	Status  PushNotificationStatus `json:"status"`
}

func (PushNotificationInput) toolInputSchemas() {}

// RemoteTriggerAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#remotetrigger
type RemoteTriggerAction string

const (
	RemoteTriggerActionList   RemoteTriggerAction = "list"
	RemoteTriggerActionGet    RemoteTriggerAction = "get"
	RemoteTriggerActionCreate RemoteTriggerAction = "create"
	RemoteTriggerActionUpdate RemoteTriggerAction = "update"
	RemoteTriggerActionRun    RemoteTriggerAction = "run"
)

// RemoteTriggerInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#remotetrigger
type RemoteTriggerInput struct {
	Action    RemoteTriggerAction `json:"action"`
	TriggerID *string             `json:"trigger_id,omitzero"`
	Body      map[string]any      `json:"body,omitzero"`
}

func (RemoteTriggerInput) toolInputSchemas() {}

// REPLInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#repl
type REPLInput struct {
	Code        string  `json:"code"`
	Description *string `json:"description,omitzero"`
	Timeout     *int64  `json:"timeout,omitzero"`
}

func (REPLInput) toolInputSchemas() {}

// ReportFindingsLevel is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#reportfindings
type ReportFindingsLevel string

const (
	ReportFindingsLevelLow    ReportFindingsLevel = "low"
	ReportFindingsLevelMedium ReportFindingsLevel = "medium"
	ReportFindingsLevelHigh   ReportFindingsLevel = "high"
	ReportFindingsLevelXHigh  ReportFindingsLevel = "xhigh"
	ReportFindingsLevelMax    ReportFindingsLevel = "max"
)

// ReportFindingVerdict is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#reportfindings
type ReportFindingVerdict string

const (
	ReportFindingVerdictConfirmed ReportFindingVerdict = "CONFIRMED"
	ReportFindingVerdictPlausible ReportFindingVerdict = "PLAUSIBLE"
)

// ReportFindingOutcome is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#reportfindings
type ReportFindingOutcome string

const (
	ReportFindingOutcomeFixed          ReportFindingOutcome = "fixed"
	ReportFindingOutcomeSkipped        ReportFindingOutcome = "skipped"
	ReportFindingOutcomeNoChangeNeeded ReportFindingOutcome = "no_change_needed"
)

// ReportFinding is a handwritten Claude Agent SDK type. The docs define the
// same inline finding shape on both ReportFindingsInput and
// ReportFindingsOutput, so it is shared here.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#reportfindings
type ReportFinding struct {
	File            string                `json:"file"`
	Line            *int64                `json:"line,omitzero"`
	Summary         string                `json:"summary"`
	FailureScenario string                `json:"failure_scenario"`
	ShortSummary    *string               `json:"short_summary,omitzero"`
	Category        *string               `json:"category,omitzero"`
	Verdict         *ReportFindingVerdict `json:"verdict,omitzero"`
	Outcome         *ReportFindingOutcome `json:"outcome,omitzero"`
}

// ReportFindingsInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#reportfindings
type ReportFindingsInput struct {
	Level    *ReportFindingsLevel `json:"level,omitzero"`
	Findings []ReportFinding      `json:"findings"`
}

func (ReportFindingsInput) toolInputSchemas() {}

// ScheduleWakeupInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#schedulewakeup
type ScheduleWakeupInput struct {
	DelaySeconds *float64 `json:"delaySeconds,omitzero"`
	Reason       *string  `json:"reason,omitzero"`
	Prompt       *string  `json:"prompt,omitzero"`
	Stop         *bool    `json:"stop,omitzero"`
}

func (ScheduleWakeupInput) toolInputSchemas() {}

// ShowOnboardingRolePickerInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#showonboardingrolepicker
type ShowOnboardingRolePickerInput struct{}

func (ShowOnboardingRolePickerInput) toolInputSchemas() {}

// ToolOutputSchemas is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tooloutputschemas
type ToolOutputSchemas struct {
	value ToolOutputSchemas_Value
}

// ToolOutputSchemas_Value is the variant interface implemented by every [ToolOutputSchemas] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tooloutputschemas
type ToolOutputSchemas_Value interface{ toolOutputSchemas() }

// MarshalJSON marshals the active [ToolOutputSchemas] variant.
func (o ToolOutputSchemas) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalForTool decodes data into the [ToolOutputSchemas] variant selected by toolName.
func (o *ToolOutputSchemas) UnmarshalForTool(toolName string, data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if done, err := o.unmarshalCodexOutputForTool(toolName, data); done {
		return err
	}
	if strings.HasPrefix(toolName, McpToolNamePrefix) {
		var v McpOutput
		o.value = &v
		return v.UnmarshalJSON(data)
	}
	switch toolName {
	case ToolNameAskUserQuestion:
		var v AskUserQuestionOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameBash:
		var v BashOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameAgent, ToolNameTask:
		var disc struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(data, &disc); err != nil {
			return err
		}
		switch disc.Status {
		case "completed":
			var v AgentOutputCompleted
			o.value = &v
			return json.Unmarshal(data, &v)
		case "async_launched":
			var v AgentOutputAsyncLaunched
			o.value = &v
			return json.Unmarshal(data, &v)
		case "remote_launched":
			var v AgentOutputRemoteLaunched
			o.value = &v
			return json.Unmarshal(data, &v)
		default:
			var v AgentOutputUnknown
			o.value = &v
			return v.UnmarshalJSON(data)
		}
	case ToolNameRead:
		var disc struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &disc); err != nil {
			return err
		}
		switch disc.Type {
		case "text":
			var v FileReadOutputText
			o.value = &v
			return json.Unmarshal(data, &v)
		case "image":
			var v FileReadOutputImage
			o.value = &v
			return json.Unmarshal(data, &v)
		case "notebook":
			var v FileReadOutputNotebook
			o.value = &v
			return json.Unmarshal(data, &v)
		case "pdf":
			var v FileReadOutputPdf
			o.value = &v
			return json.Unmarshal(data, &v)
		case "parts":
			var v FileReadOutputParts
			o.value = &v
			return json.Unmarshal(data, &v)
		case "file_unchanged":
			var v FileReadOutputFileUnchanged
			o.value = &v
			return json.Unmarshal(data, &v)
		default:
			var v FileReadOutputUnknown
			o.value = &v
			return v.UnmarshalJSON(data)
		}
	case ToolNameEdit:
		var v FileEditOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameEnterWorktree:
		var v EnterWorktreeOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameExitPlanMode:
		var v ExitPlanModeOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameGlob:
		var v GlobOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameGrep:
		var v GrepOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameListMcpResources:
		var v ListMcpResourcesOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameNotebookEdit:
		var v NotebookEditOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameReadMcpResource:
		var v ReadMcpResourceOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameMonitor:
		var v MonitorOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWorkflow:
		var v WorkflowOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskCreate:
		var v TaskCreateOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskUpdate:
		var v TaskUpdateOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskGet:
		var v TaskGetOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskList:
		var v TaskListOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTaskStop:
		var v TaskStopOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameTodoWrite:
		var v TodoWriteOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWebFetch:
		var v WebFetchOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWebSearch:
		var v WebSearchOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameWrite:
		var v FileWriteOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameArtifact:
		var v ArtifactOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameCronCreate:
		var v CronCreateOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameCronDelete:
		var v CronDeleteOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameCronList:
		var v CronListOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameEnterPlanMode:
		var v EnterPlanModeOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameExitWorktree:
		var v ExitWorktreeOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameProjects:
		var v ProjectsOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNamePushNotification:
		var v PushNotificationOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameReadMcpResourceDir:
		var v ReadMcpResourceDirOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameRefreshMcpTools:
		var v RefreshMcpToolsOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameRemoteTrigger:
		var v RemoteTriggerOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameREPL:
		var v REPLOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameReportFindings:
		var v ReportFindingsOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameScheduleWakeup:
		var v ScheduleWakeupOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	case ToolNameShowOnboardingRolePicker:
		var v ShowOnboardingRolePickerOutput
		o.value = &v
		return json.Unmarshal(data, &v)
	default:
		var v ToolOutputUnknown
		o.value = &v
		return v.UnmarshalJSON(data)
	}
}

var (
	_ ToolOutputSchemas_Value = (*ToolOutputUnknown)(nil)
	_ ToolOutputSchemas_Value = (*AgentOutputUnknown)(nil)
	_ ToolOutputSchemas_Value = (*AgentOutputCompleted)(nil)
	_ ToolOutputSchemas_Value = (*AgentOutputAsyncLaunched)(nil)
	_ ToolOutputSchemas_Value = (*AgentOutputRemoteLaunched)(nil)
	_ ToolOutputSchemas_Value = (*ArtifactOutput)(nil)
	_ ToolOutputSchemas_Value = (*AskUserQuestionOutput)(nil)
	_ ToolOutputSchemas_Value = (*BashOutput)(nil)
	_ ToolOutputSchemas_Value = (*CronCreateOutput)(nil)
	_ ToolOutputSchemas_Value = (*CronDeleteOutput)(nil)
	_ ToolOutputSchemas_Value = (*CronListOutput)(nil)
	_ ToolOutputSchemas_Value = (*EnterPlanModeOutput)(nil)
	_ ToolOutputSchemas_Value = (*ExitWorktreeOutput)(nil)
	_ ToolOutputSchemas_Value = (*FileEditOutput)(nil)
	_ ToolOutputSchemas_Value = (*FileReadOutputUnknown)(nil)
	_ ToolOutputSchemas_Value = (*FileReadOutputText)(nil)
	_ ToolOutputSchemas_Value = (*FileReadOutputImage)(nil)
	_ ToolOutputSchemas_Value = (*FileReadOutputNotebook)(nil)
	_ ToolOutputSchemas_Value = (*FileReadOutputPdf)(nil)
	_ ToolOutputSchemas_Value = (*FileReadOutputParts)(nil)
	_ ToolOutputSchemas_Value = (*FileReadOutputFileUnchanged)(nil)
	_ ToolOutputSchemas_Value = (*FileWriteOutput)(nil)
	_ ToolOutputSchemas_Value = (*GlobOutput)(nil)
	_ ToolOutputSchemas_Value = (*GrepOutput)(nil)
	_ ToolOutputSchemas_Value = (*TaskStopOutput)(nil)
	_ ToolOutputSchemas_Value = (*NotebookEditOutput)(nil)
	_ ToolOutputSchemas_Value = (*ProjectsOutput)(nil)
	_ ToolOutputSchemas_Value = (*PushNotificationOutput)(nil)
	_ ToolOutputSchemas_Value = (*ReadMcpResourceDirOutput)(nil)
	_ ToolOutputSchemas_Value = (*RefreshMcpToolsOutput)(nil)
	_ ToolOutputSchemas_Value = (*RemoteTriggerOutput)(nil)
	_ ToolOutputSchemas_Value = (*REPLOutput)(nil)
	_ ToolOutputSchemas_Value = (*ReportFindingsOutput)(nil)
	_ ToolOutputSchemas_Value = (*ScheduleWakeupOutput)(nil)
	_ ToolOutputSchemas_Value = (*ShowOnboardingRolePickerOutput)(nil)
	_ ToolOutputSchemas_Value = (*McpOutput)(nil)
	_ ToolOutputSchemas_Value = (*WebFetchOutput)(nil)
	_ ToolOutputSchemas_Value = (*WebSearchOutput)(nil)
	_ ToolOutputSchemas_Value = (*TodoWriteOutput)(nil)
	_ ToolOutputSchemas_Value = (*ExitPlanModeOutput)(nil)
	_ ToolOutputSchemas_Value = (*ListMcpResourcesOutput)(nil)
	_ ToolOutputSchemas_Value = (*ReadMcpResourceOutput)(nil)
	_ ToolOutputSchemas_Value = (*EnterWorktreeOutput)(nil)
	_ ToolOutputSchemas_Value = (*MonitorOutput)(nil)
	_ ToolOutputSchemas_Value = (*WorkflowOutput)(nil)
	_ ToolOutputSchemas_Value = (*TaskCreateOutput)(nil)
	_ ToolOutputSchemas_Value = (*TaskUpdateOutput)(nil)
	_ ToolOutputSchemas_Value = (*TaskGetOutput)(nil)
	_ ToolOutputSchemas_Value = (*TaskListOutput)(nil)
)

// ToolOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tooloutputschemas
type ToolOutputUnknown struct{ UnknownUnion }

func (ToolOutputUnknown) toolOutputSchemas() {}

// AgentOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutput interface {
	toolOutputSchemas()
	agentOutput()
}

var (
	_ AgentOutput = (*AgentOutputUnknown)(nil)
	_ AgentOutput = (*AgentOutputCompleted)(nil)
	_ AgentOutput = (*AgentOutputAsyncLaunched)(nil)
	_ AgentOutput = (*AgentOutputRemoteLaunched)(nil)
)

// AgentOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputUnknown struct{ UnknownUnion }

func (AgentOutputUnknown) toolOutputSchemas() {}
func (AgentOutputUnknown) agentOutput()       {}

// AgentOutputContentBlock is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputContentBlock struct {
	Type      string            `json:"type"`
	Text      string            `json:"text"`
	Citations []json.RawMessage `json:"citations,omitzero"`
}

// AgentOutputServerToolUse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputServerToolUse struct {
	WebSearchRequests int64 `json:"web_search_requests"`
	WebFetchRequests  int64 `json:"web_fetch_requests"`
}

// AgentOutputCacheCreation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputCacheCreation struct {
	Ephemeral1HInputTokens int64 `json:"ephemeral_1h_input_tokens"`
	Ephemeral5MInputTokens int64 `json:"ephemeral_5m_input_tokens"`
}

// AgentOutputUsage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputUsage struct {
	InputTokens              int64                     `json:"input_tokens"`
	OutputTokens             int64                     `json:"output_tokens"`
	CacheCreationInputTokens *int64                    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64                    `json:"cache_read_input_tokens"`
	ServerToolUse            *AgentOutputServerToolUse `json:"server_tool_use"`
	// ServiceTier is an open string since v2.1.207 (previously the closed set
	// "standard" | "priority" | "batch").
	ServiceTier   *string                   `json:"service_tier"`
	CacheCreation *AgentOutputCacheCreation `json:"cache_creation"`
	InferenceGeo  *string                   `json:"inference_geo,omitzero"`
	Speed         *string                   `json:"speed,omitzero"`
	Iterations    json.RawMessage           `json:"iterations,omitzero"`
}

// AgentOutputToolStats is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputToolStats struct {
	ReadCount      int64  `json:"readCount"`
	SearchCount    int64  `json:"searchCount"`
	BashCount      int64  `json:"bashCount"`
	EditFileCount  int64  `json:"editFileCount"`
	LinesAdded     int64  `json:"linesAdded"`
	LinesRemoved   int64  `json:"linesRemoved"`
	OtherToolCount int64  `json:"otherToolCount"`
	FrameCount     *int64 `json:"frameCount,omitzero"`
}

// AgentOutputCompleted is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputCompleted struct {
	Status            string                    `json:"status"`
	AgentID           string                    `json:"agentId"`
	AgentType         *string                   `json:"agentType,omitzero"`
	Content           []AgentOutputContentBlock `json:"content"`
	TotalToolUseCount int64                     `json:"totalToolUseCount"`
	TotalDurationMs   int64                     `json:"totalDurationMs"`
	TotalTokens       int64                     `json:"totalTokens"`
	Usage             AgentOutputUsage          `json:"usage"`
	Prompt            string                    `json:"prompt"`
	ResolvedModel     *string                   `json:"resolvedModel,omitzero"`
	ModelsUsed        []string                  `json:"modelsUsed,omitzero"`
	ToolStats         *AgentOutputToolStats     `json:"toolStats,omitzero"`
	WorktreePath      *string                   `json:"worktreePath,omitzero"`
	WorktreeBranch    *string                   `json:"worktreeBranch,omitzero"`
}

func (AgentOutputCompleted) toolOutputSchemas() {}
func (AgentOutputCompleted) agentOutput()       {}

// AgentOutputAsyncLaunched is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputAsyncLaunched struct {
	Status            string   `json:"status"`
	AgentID           string   `json:"agentId"`
	IsAsync           *bool    `json:"isAsync,omitzero"`
	Description       string   `json:"description"`
	Prompt            string   `json:"prompt"`
	OutputFile        string   `json:"outputFile"`
	CanReadOutputFile *bool    `json:"canReadOutputFile,omitzero"`
	ResolvedModel     *string  `json:"resolvedModel,omitzero"`
	ModelsUsed        []string `json:"modelsUsed,omitzero"`
}

func (AgentOutputAsyncLaunched) toolOutputSchemas() {}
func (AgentOutputAsyncLaunched) agentOutput()       {}

// AgentOutputRemoteLaunched is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agent-2
type AgentOutputRemoteLaunched struct {
	Status      string `json:"status"`
	TaskID      string `json:"taskId"`
	SessionURL  string `json:"sessionUrl"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	OutputFile  string `json:"outputFile"`
}

func (AgentOutputRemoteLaunched) toolOutputSchemas() {}
func (AgentOutputRemoteLaunched) agentOutput()       {}

// AskUserQuestionOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion-2
type AskUserQuestionOutput struct {
	Questions    []AskUserQuestionInputQuestion       `json:"questions"`
	Answers      map[string]string                    `json:"answers"`
	Annotations  map[string]AskUserQuestionAnnotation `json:"annotations,omitzero"`
	Response     *string                              `json:"response,omitzero"`
	AFKTimeoutMs *int64                               `json:"afkTimeoutMs,omitzero"`
}

func (AskUserQuestionOutput) toolOutputSchemas() {}

// BashOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashOutput struct {
	Stdout                    string            `json:"stdout"`
	Stderr                    string            `json:"stderr"`
	RawOutputPath             *string           `json:"rawOutputPath,omitzero"`
	Interrupted               bool              `json:"interrupted"`
	IsImage                   *bool             `json:"isImage,omitzero"`
	TimedOutAfterMs           *int64            `json:"timedOutAfterMs,omitzero"`
	BackgroundTaskID          *string           `json:"backgroundTaskId,omitzero"`
	BackgroundedByUser        *bool             `json:"backgroundedByUser,omitzero"`
	BackgroundCwdHint         *string           `json:"backgroundCwdHint,omitzero"`
	DangerouslyDisableSandbox *bool             `json:"dangerouslyDisableSandbox,omitzero"`
	ReturnCodeInterpretation  *string           `json:"returnCodeInterpretation,omitzero"`
	NoOutputExpected          *bool             `json:"noOutputExpected,omitzero"`
	StaleReadFileStateHint    *string           `json:"staleReadFileStateHint,omitzero"`
	GhRateLimitHint           *string           `json:"ghRateLimitHint,omitzero"`
	StructuredContent         []json.RawMessage `json:"structuredContent,omitzero"`
	PersistedOutputPath       *string           `json:"persistedOutputPath,omitzero"`
	PersistedOutputSize       *int64            `json:"persistedOutputSize,omitzero"`
	GitOperation              *BashGitOperation `json:"gitOperation,omitzero"`
}

func (BashOutput) toolOutputSchemas() {}

// BashGitCommitKind is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitCommitKind string

const (
	BashGitCommitKindCommitted    BashGitCommitKind = "committed"
	BashGitCommitKindAmended      BashGitCommitKind = "amended"
	BashGitCommitKindCherryPicked BashGitCommitKind = "cherry-picked"
)

// BashGitBranchAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitBranchAction string

const (
	BashGitBranchActionMerged  BashGitBranchAction = "merged"
	BashGitBranchActionRebased BashGitBranchAction = "rebased"
)

// BashGitPrAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitPrAction string

const (
	BashGitPrActionCreated           BashGitPrAction = "created"
	BashGitPrActionEdited            BashGitPrAction = "edited"
	BashGitPrActionMerged            BashGitPrAction = "merged"
	BashGitPrActionCommented         BashGitPrAction = "commented"
	BashGitPrActionClosed            BashGitPrAction = "closed"
	BashGitPrActionReady             BashGitPrAction = "ready"
	BashGitPrActionDraft             BashGitPrAction = "draft"
	BashGitPrActionAutoMergeEnabled  BashGitPrAction = "auto-merge-enabled"
	BashGitPrActionAutoMergeDisabled BashGitPrAction = "auto-merge-disabled"
)

// BashGitOperationCommit is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitOperationCommit struct {
	Sha  string            `json:"sha"`
	Kind BashGitCommitKind `json:"kind"`
}

// BashGitOperationPush is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitOperationPush struct {
	Branch string `json:"branch"`
}

// BashGitOperationBranch is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitOperationBranch struct {
	Ref    string              `json:"ref"`
	Action BashGitBranchAction `json:"action"`
}

// BashGitOperationPr is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitOperationPr struct {
	Number int64           `json:"number"`
	URL    *string         `json:"url,omitzero"`
	Action BashGitPrAction `json:"action"`
}

// BashGitOperation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#bash-2
type BashGitOperation struct {
	Commit *BashGitOperationCommit `json:"commit,omitzero"`
	Push   *BashGitOperationPush   `json:"push,omitzero"`
	Branch *BashGitOperationBranch `json:"branch,omitzero"`
	Pr     *BashGitOperationPr     `json:"pr,omitzero"`
}

// StructuredPatchHunk is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#edit-2
type StructuredPatchHunk struct {
	OldStart int64    `json:"oldStart"`
	OldLines int64    `json:"oldLines"`
	NewStart int64    `json:"newStart"`
	NewLines int64    `json:"newLines"`
	Lines    []string `json:"lines"`
}

// GitDiffStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#edit-2
type GitDiffStatus string

const (
	GitDiffStatusModified GitDiffStatus = "modified"
	GitDiffStatusAdded    GitDiffStatus = "added"
)

// GitDiff is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#edit-2
type GitDiff struct {
	Filename   string        `json:"filename"`
	Status     GitDiffStatus `json:"status"`
	Additions  int64         `json:"additions"`
	Deletions  int64         `json:"deletions"`
	Changes    int64         `json:"changes"`
	Patch      string        `json:"patch"`
	Repository *string       `json:"repository,omitzero"`
}

// FileEditOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#edit-2
type FileEditOutput struct {
	FilePath        string                `json:"filePath"`
	OldString       string                `json:"oldString"`
	NewString       string                `json:"newString"`
	OriginalFile    *string               `json:"originalFile"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	UserModified    bool                  `json:"userModified"`
	ReplaceAll      bool                  `json:"replaceAll"`
	GitDiff         *GitDiff              `json:"gitDiff,omitzero"`
}

func (FileEditOutput) toolOutputSchemas() {}

// FileReadOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutput interface {
	toolOutputSchemas()
	fileReadOutput()
}

var (
	_ FileReadOutput = (*FileReadOutputUnknown)(nil)
	_ FileReadOutput = (*FileReadOutputText)(nil)
	_ FileReadOutput = (*FileReadOutputImage)(nil)
	_ FileReadOutput = (*FileReadOutputNotebook)(nil)
	_ FileReadOutput = (*FileReadOutputPdf)(nil)
	_ FileReadOutput = (*FileReadOutputParts)(nil)
	_ FileReadOutput = (*FileReadOutputFileUnchanged)(nil)
)

// FileReadOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputUnknown struct{ UnknownUnion }

func (FileReadOutputUnknown) toolOutputSchemas() {}
func (FileReadOutputUnknown) fileReadOutput()    {}

// FileReadOutputTextFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputTextFile struct {
	FilePath   string `json:"filePath"`
	Content    string `json:"content"`
	NumLines   int64  `json:"numLines"`
	StartLine  int64  `json:"startLine"`
	TotalLines int64  `json:"totalLines"`
	// TruncatedByTokenCap is true when a whole-file read was auto-paginated
	// because it exceeded the token cap.
	TruncatedByTokenCap *bool `json:"truncatedByTokenCap,omitzero"`
}

// FileReadOutputText is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputText struct {
	Type string                 `json:"type"`
	File FileReadOutputTextFile `json:"file"`
}

func (FileReadOutputText) toolOutputSchemas() {}
func (FileReadOutputText) fileReadOutput()    {}

// FileReadOutputImageDimensions is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputImageDimensions struct {
	OriginalWidth  *int64 `json:"originalWidth,omitzero"`
	OriginalHeight *int64 `json:"originalHeight,omitzero"`
	DisplayWidth   *int64 `json:"displayWidth,omitzero"`
	DisplayHeight  *int64 `json:"displayHeight,omitzero"`
}

// FileReadOutputImageFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputImageFile struct {
	Base64       string                         `json:"base64"`
	Type         FileReadImageMimeType          `json:"type"`
	OriginalSize int64                          `json:"originalSize"`
	Dimensions   *FileReadOutputImageDimensions `json:"dimensions,omitzero"`
}

// FileReadOutputImage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputImage struct {
	Type string                  `json:"type"`
	File FileReadOutputImageFile `json:"file"`
}

func (FileReadOutputImage) toolOutputSchemas() {}
func (FileReadOutputImage) fileReadOutput()    {}

// FileReadOutputNotebookFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputNotebookFile struct {
	FilePath string            `json:"filePath"`
	Cells    []json.RawMessage `json:"cells"`
}

// FileReadOutputNotebook is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputNotebook struct {
	Type string                     `json:"type"`
	File FileReadOutputNotebookFile `json:"file"`
}

func (FileReadOutputNotebook) toolOutputSchemas() {}
func (FileReadOutputNotebook) fileReadOutput()    {}

// FileReadOutputPdfFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputPdfFile struct {
	FilePath     string `json:"filePath"`
	Base64       string `json:"base64"`
	OriginalSize int64  `json:"originalSize"`
}

// FileReadOutputPdf is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputPdf struct {
	Type string                `json:"type"`
	File FileReadOutputPdfFile `json:"file"`
}

func (FileReadOutputPdf) toolOutputSchemas() {}
func (FileReadOutputPdf) fileReadOutput()    {}

// FileReadOutputPartsFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputPartsFile struct {
	FilePath     string `json:"filePath"`
	OriginalSize int64  `json:"originalSize"`
	Count        int64  `json:"count"`
	OutputDir    string `json:"outputDir"`
}

// FileReadOutputParts is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputParts struct {
	Type string                  `json:"type"`
	File FileReadOutputPartsFile `json:"file"`
}

func (FileReadOutputParts) toolOutputSchemas() {}
func (FileReadOutputParts) fileReadOutput()    {}

// FileReadOutputFileUnchangedSource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputFileUnchangedSource string

const (
	FileReadOutputFileUnchangedSourceSeeded FileReadOutputFileUnchangedSource = "seeded"
)

// FileReadOutputFileUnchangedFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputFileUnchangedFile struct {
	FilePath string `json:"filePath"`
}

// FileReadOutputFileUnchanged is a handwritten Claude Agent SDK type. Source
// is set when the dedup matched a startup-seeded entry (CLAUDE.md / nested
// memory) rather than a prior Read tool_result.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#read-2
type FileReadOutputFileUnchanged struct {
	Type   string                             `json:"type"`
	File   FileReadOutputFileUnchangedFile    `json:"file"`
	Source *FileReadOutputFileUnchangedSource `json:"source,omitzero"`
}

func (FileReadOutputFileUnchanged) toolOutputSchemas() {}
func (FileReadOutputFileUnchanged) fileReadOutput()    {}

// FileWriteOutputType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#write-2
type FileWriteOutputType string

const (
	FileWriteOutputTypeCreate FileWriteOutputType = "create"
	FileWriteOutputTypeUpdate FileWriteOutputType = "update"
)

// FileWriteOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#write-2
type FileWriteOutput struct {
	Type            FileWriteOutputType   `json:"type"`
	FilePath        string                `json:"filePath"`
	Content         string                `json:"content"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	OriginalFile    *string               `json:"originalFile"`
	UserModified    *bool                 `json:"userModified,omitzero"`
	GitDiff         *GitDiff              `json:"gitDiff,omitzero"`
}

func (FileWriteOutput) toolOutputSchemas() {}

// GlobOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#glob-2
type GlobOutput struct {
	DurationMs      int64    `json:"durationMs"`
	NumFiles        int64    `json:"numFiles"`
	Filenames       []string `json:"filenames"`
	Truncated       bool     `json:"truncated"`
	TotalMatches    *int64   `json:"totalMatches,omitzero"`
	CountIsComplete *bool    `json:"countIsComplete,omitzero"`
}

func (GlobOutput) toolOutputSchemas() {}

// GrepOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#grep-2
type GrepOutput struct {
	Mode          *GrepOutputMode `json:"mode,omitzero"`
	NumFiles      int64           `json:"numFiles"`
	Filenames     []string        `json:"filenames"`
	Content       *string         `json:"content,omitzero"`
	NumLines      *int64          `json:"numLines,omitzero"`
	NumMatches    *int64          `json:"numMatches,omitzero"`
	AppliedLimit  *int64          `json:"appliedLimit,omitzero"`
	AppliedOffset *int64          `json:"appliedOffset,omitzero"`
	TotalFiles    *int64          `json:"totalFiles,omitzero"`
	TotalLines    *int64          `json:"totalLines,omitzero"`
}

func (GrepOutput) toolOutputSchemas() {}

// TaskStopOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskstop-2
type TaskStopOutput struct {
	Message  string  `json:"message"`
	TaskID   string  `json:"task_id"`
	TaskType string  `json:"task_type"`
	Command  *string `json:"command,omitzero"`
}

func (TaskStopOutput) toolOutputSchemas() {}

// NotebookEditOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#notebookedit-2
type NotebookEditOutput struct {
	NewSource    string           `json:"new_source"`
	OldSource    *string          `json:"old_source,omitzero"`
	CellID       *string          `json:"cell_id,omitzero"`
	CellType     NotebookCellType `json:"cell_type"`
	Language     string           `json:"language"`
	EditMode     string           `json:"edit_mode"`
	Error        *string          `json:"error,omitzero"`
	NotebookPath string           `json:"notebook_path"`
	OriginalFile string           `json:"original_file"`
	UpdatedFile  string           `json:"updated_file"`
}

func (NotebookEditOutput) toolOutputSchemas() {}

// WebFetchArtifactRead is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#webfetch-2
type WebFetchArtifactRead struct {
	Slug string  `json:"slug"`
	Ver  *string `json:"ver,omitzero"`
}

// WebFetchOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#webfetch-2
type WebFetchOutput struct {
	Bytes        int64                 `json:"bytes"`
	Code         int64                 `json:"code"`
	CodeText     string                `json:"codeText"`
	Result       string                `json:"result"`
	DurationMs   int64                 `json:"durationMs"`
	URL          string                `json:"url"`
	ArtifactRead *WebFetchArtifactRead `json:"artifactRead,omitzero"`
}

func (WebFetchOutput) toolOutputSchemas() {}

// WebSearchOutputResult is a handwritten Claude Agent SDK type.
//
// Each result is either a structured block or a bare string. The union is
// untagged, so decoding dispatches on the JSON token (object vs string).
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutputResult struct {
	value WebSearchOutputResult_Value
}

// WebSearchOutputResult_Value is the variant interface implemented by every [WebSearchOutputResult]
// case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutputResult_Value interface{ webSearchOutputResult() }

// MarshalJSON marshals the active [WebSearchOutputResult] variant.
func (o WebSearchOutputResult) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [WebSearchOutputResult] union value from JSON. The
// SDK emits either a bare string or an object, so dispatch is by JSON token.
func (o *WebSearchOutputResult) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	switch {
	case trimmed == "":
		return nil
	case trimmed[0] == '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		text := WebSearchOutputResultText(s)
		o.value = &text
		return nil
	case trimmed[0] == '{':
		var v WebSearchOutputResultBlock
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		o.value = &v
		return nil
	default:
		var u WebSearchOutputResultUnknown
		o.value = &u
		return u.UnmarshalJSON(data)
	}
}

var (
	_ WebSearchOutputResult_Value = (*WebSearchOutputResultUnknown)(nil)
	_ WebSearchOutputResult_Value = WebSearchOutputResultText("")
	_ WebSearchOutputResult_Value = (*WebSearchOutputResultBlock)(nil)
)

// WebSearchOutputResultText is the bare-string variant of WebSearchOutputResult.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutputResultText string

func (WebSearchOutputResultText) webSearchOutputResult() {}

func (o WebSearchOutputResultText) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(o))
}

// WebSearchOutputResultContent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutputResultContent struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// WebSearchOutputResultBlock is the object variant of WebSearchOutputResult.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutputResultBlock struct {
	ToolUseID string                         `json:"tool_use_id"`
	Content   []WebSearchOutputResultContent `json:"content"`
}

func (*WebSearchOutputResultBlock) webSearchOutputResult() {}

// WebSearchOutputResultUnknown preserves an unrecognized WebSearchOutputResult.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutputResultUnknown struct{ UnknownUnion }

func (*WebSearchOutputResultUnknown) webSearchOutputResult() {}

// WebSearchOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutput struct {
	Query           string                  `json:"query"`
	Results         []WebSearchOutputResult `json:"results"`
	DurationSeconds float64                 `json:"durationSeconds"`
	SearchCount     *int64                  `json:"searchCount,omitzero"`
}

func (WebSearchOutput) toolOutputSchemas() {}

func (o WebSearchOutput) MarshalJSON() ([]byte, error) {
	type alias WebSearchOutput
	return json.Marshal(alias(o))
}

func (o *WebSearchOutput) UnmarshalJSON(data []byte) error {
	type alias WebSearchOutput
	type raw struct {
		alias
		Results []json.RawMessage `json:"results"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	results := make([]WebSearchOutputResult, 0, len(v.Results))
	for _, r := range v.Results {
		var got WebSearchOutputResult
		if err := got.UnmarshalJSON(r); err != nil {
			return err
		}
		results = append(results, got)
	}
	*o = WebSearchOutput(v.alias)
	o.Results = results
	return nil
}

// TodoWriteOutput is a handwritten Claude Agent SDK type.
//
// Deprecated: as of TypeScript Agent SDK 0.3.142 TodoWrite is disabled by
// default; use TaskCreate, TaskGet, TaskUpdate and TaskList instead.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#todowrite-2
type TodoWriteOutput struct {
	OldTodos []TodoItem `json:"oldTodos"`
	NewTodos []TodoItem `json:"newTodos"`
}

func (TodoWriteOutput) toolOutputSchemas() {}

// ExitPlanModeOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitplanmode-2
type ExitPlanModeOutput struct {
	Plan                   *string `json:"plan"`
	IsAgent                bool    `json:"isAgent"`
	FilePath               *string `json:"filePath,omitzero"`
	PlanWasEdited          *bool   `json:"planWasEdited,omitzero"`
	HasTaskTool            *bool   `json:"hasTaskTool,omitzero"`
	AwaitingLeaderApproval *bool   `json:"awaitingLeaderApproval,omitzero"`
	RequestID              *string `json:"requestId,omitzero"`
}

func (ExitPlanModeOutput) toolOutputSchemas() {}

// McpResource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#listmcpresources-2
type McpResource struct {
	URI         string  `json:"uri"`
	Name        string  `json:"name"`
	MimeType    *string `json:"mimeType,omitzero"`
	Description *string `json:"description,omitzero"`
	Server      string  `json:"server"`
}

// ListMcpResourcesOutput is a handwritten Claude Agent SDK type. It is a bare
// JSON array of resources.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#listmcpresources-2
type ListMcpResourcesOutput []McpResource

func (ListMcpResourcesOutput) toolOutputSchemas() {}

// TaskStatus is a handwritten Claude Agent SDK type. It is the lifecycle status
// reported by the Task read tools (TaskGet, TaskList).
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskget-2
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
)

// MonitorOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#monitor-2
type MonitorOutput struct {
	TaskID     string `json:"taskId"`
	TimeoutMs  int64  `json:"timeoutMs"`
	Persistent *bool  `json:"persistent,omitzero"`
}

func (MonitorOutput) toolOutputSchemas() {}

// WorkflowOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#workflow-2
type WorkflowOutput struct {
	Status        WorkflowOutputStatus    `json:"status"`
	TaskID        string                  `json:"taskId"`
	TaskType      *WorkflowOutputTaskType `json:"taskType,omitzero"`
	WorkflowName  *string                 `json:"workflowName,omitzero"`
	RunID         *string                 `json:"runId,omitzero"`
	SessionURL    *string                 `json:"sessionUrl,omitzero"`
	Summary       *string                 `json:"summary,omitzero"`
	TranscriptDir *string                 `json:"transcriptDir,omitzero"`
	ScriptPath    *string                 `json:"scriptPath,omitzero"`
	Warning       *string                 `json:"warning,omitzero"`
	Error         *string                 `json:"error,omitzero"`
}

// WorkflowOutputStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#workflow-2
type WorkflowOutputStatus string

const (
	WorkflowOutputStatusAsyncLaunched  WorkflowOutputStatus = "async_launched"
	WorkflowOutputStatusRemoteLaunched WorkflowOutputStatus = "remote_launched"
)

// WorkflowOutputTaskType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#workflow-2
type WorkflowOutputTaskType string

const (
	WorkflowOutputTaskTypeLocalWorkflow WorkflowOutputTaskType = "local_workflow"
	WorkflowOutputTaskTypeRemoteAgent   WorkflowOutputTaskType = "remote_agent"
)

func (WorkflowOutput) toolOutputSchemas() {}

// TaskCreateOutputTask is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskcreate-2
type TaskCreateOutputTask struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
}

// TaskCreateOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskcreate-2
type TaskCreateOutput struct {
	Task TaskCreateOutputTask `json:"task"`
}

func (TaskCreateOutput) toolOutputSchemas() {}

// TaskUpdateOutputStatusChange is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskupdate-2
type TaskUpdateOutputStatusChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// TaskUpdateOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskupdate-2
type TaskUpdateOutput struct {
	Success       bool                          `json:"success"`
	TaskID        string                        `json:"taskId"`
	UpdatedFields []string                      `json:"updatedFields"`
	Error         *string                       `json:"error,omitzero"`
	StatusChange  *TaskUpdateOutputStatusChange `json:"statusChange,omitzero"`
}

func (TaskUpdateOutput) toolOutputSchemas() {}

// TaskGetOutputTask is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskget-2
type TaskGetOutputTask struct {
	ID          string     `json:"id"`
	Subject     string     `json:"subject"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Blocks      []string   `json:"blocks"`
	BlockedBy   []string   `json:"blockedBy"`
}

// TaskGetOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskget-2
type TaskGetOutput struct {
	Task *TaskGetOutputTask `json:"task"`
}

func (TaskGetOutput) toolOutputSchemas() {}

// TaskListOutputTask is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tasklist-2
type TaskListOutputTask struct {
	ID        string     `json:"id"`
	Subject   string     `json:"subject"`
	Status    TaskStatus `json:"status"`
	Owner     *string    `json:"owner,omitzero"`
	BlockedBy []string   `json:"blockedBy"`
}

// TaskListOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tasklist-2
type TaskListOutput struct {
	Tasks []TaskListOutputTask `json:"tasks"`
}

func (TaskListOutput) toolOutputSchemas() {}

// ReadMcpResourceOutputContent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresource-2
type ReadMcpResourceOutputContent struct {
	URI         string  `json:"uri"`
	MimeType    *string `json:"mimeType,omitzero"`
	Text        *string `json:"text,omitzero"`
	BlobSavedTo *string `json:"blobSavedTo,omitzero"`
}

// ReadMcpResourceOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresource-2
type ReadMcpResourceOutput struct {
	Contents []ReadMcpResourceOutputContent `json:"contents"`
	Error    *string                        `json:"error,omitzero"`
}

func (ReadMcpResourceOutput) toolOutputSchemas() {}

// EnterWorktreeOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#enterworktree-2
type EnterWorktreeOutput struct {
	WorktreePath   string  `json:"worktreePath"`
	WorktreeBranch *string `json:"worktreeBranch,omitzero"`
	Message        string  `json:"message"`
}

func (EnterWorktreeOutput) toolOutputSchemas() {}

// ExitWorktreeOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitworktree-2
type ExitWorktreeOutput struct {
	Action           ExitWorktreeOutputAction `json:"action"`
	OriginalCwd      string                   `json:"originalCwd"`
	WorktreePath     string                   `json:"worktreePath"`
	WorktreeBranch   *string                  `json:"worktreeBranch,omitzero"`
	TmuxSessionName  *string                  `json:"tmuxSessionName,omitzero"`
	DiscardedFiles   *int64                   `json:"discardedFiles,omitzero"`
	DiscardedCommits *int64                   `json:"discardedCommits,omitzero"`
	Message          string                   `json:"message"`
}

func (ExitWorktreeOutput) toolOutputSchemas() {}

// ExitWorktreeOutputAction is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitworktree-2
type ExitWorktreeOutputAction string

const (
	ExitWorktreeOutputActionKeep   ExitWorktreeOutputAction = "keep"
	ExitWorktreeOutputActionRemove ExitWorktreeOutputAction = "remove"
)

// EnterPlanModeOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#enterplanmode-2
type EnterPlanModeOutput struct {
	Message string `json:"message"`
}

func (EnterPlanModeOutput) toolOutputSchemas() {}

// CronCreateOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#croncreate-2
type CronCreateOutput struct {
	ID            string `json:"id"`
	HumanSchedule string `json:"humanSchedule"`
	Recurring     bool   `json:"recurring"`
	Durable       *bool  `json:"durable,omitzero"`
}

func (CronCreateOutput) toolOutputSchemas() {}

// CronDeleteOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#crondelete-2
type CronDeleteOutput struct {
	ID string `json:"id"`
}

func (CronDeleteOutput) toolOutputSchemas() {}

// CronListOutputJob is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#cronlist-2
type CronListOutputJob struct {
	ID            string `json:"id"`
	Cron          string `json:"cron"`
	HumanSchedule string `json:"humanSchedule"`
	Prompt        string `json:"prompt"`
	Recurring     *bool  `json:"recurring,omitzero"`
	Durable       *bool  `json:"durable,omitzero"`
}

// CronListOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#cronlist-2
type CronListOutput struct {
	Jobs []CronListOutputJob `json:"jobs"`
}

func (CronListOutput) toolOutputSchemas() {}

// ScheduleWakeupOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#schedulewakeup-2
type ScheduleWakeupOutput struct {
	ScheduledFor        float64 `json:"scheduledFor"`
	ClampedDelaySeconds float64 `json:"clampedDelaySeconds"`
	WasClamped          bool    `json:"wasClamped"`
	Stopped             *bool   `json:"stopped,omitzero"`
	CancelledWakeups    *int64  `json:"cancelledWakeups,omitzero"`
}

func (ScheduleWakeupOutput) toolOutputSchemas() {}

// RemoteTriggerOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#remotetrigger-2
type RemoteTriggerOutput struct {
	Status  int64   `json:"status"`
	JSON    string  `json:"json"`
	Summary *string `json:"summary,omitzero"`
}

func (RemoteTriggerOutput) toolOutputSchemas() {}

// PushNotificationDisabledReason is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#pushnotification-2
type PushNotificationDisabledReason string

const (
	PushNotificationDisabledReasonConfigOff   PushNotificationDisabledReason = "config_off"
	PushNotificationDisabledReasonUserPresent PushNotificationDisabledReason = "user_present"
	PushNotificationDisabledReasonNoTransport PushNotificationDisabledReason = "no_transport"
)

// PushNotificationOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#pushnotification-2
type PushNotificationOutput struct {
	Message        string                          `json:"message"`
	PushSent       *bool                           `json:"pushSent,omitzero"`
	LocalSent      *bool                           `json:"localSent,omitzero"`
	DisabledReason *PushNotificationDisabledReason `json:"disabledReason,omitzero"`
	SentAt         *string                         `json:"sentAt,omitzero"`
}

func (PushNotificationOutput) toolOutputSchemas() {}

// REPLOutputImage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#repl-2
type REPLOutputImage struct {
	Base64    string `json:"base64"`
	MediaType string `json:"mediaType"`
}

// REPLOutputDocument is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#repl-2
type REPLOutputDocument struct {
	Base64 string `json:"base64"`
}

// REPLOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#repl-2
type REPLOutput struct {
	Code            string               `json:"code"`
	Result          map[string]any       `json:"result"`
	Stdout          string               `json:"stdout"`
	Stderr          string               `json:"stderr"`
	Error           *string              `json:"error,omitzero"`
	RegisteredTools []string             `json:"registeredTools,omitzero"`
	Images          []REPLOutputImage    `json:"images,omitzero"`
	Documents       []REPLOutputDocument `json:"documents,omitzero"`
}

func (REPLOutput) toolOutputSchemas() {}

// ReportFindingsOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#reportfindings-2
type ReportFindingsOutput struct {
	Count    int64                `json:"count"`
	Level    *ReportFindingsLevel `json:"level,omitzero"`
	Findings []ReportFinding      `json:"findings"`
}

func (ReportFindingsOutput) toolOutputSchemas() {}

// ReadMcpResourceDirOutputResource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresourcedir-2
type ReadMcpResourceDirOutputResource struct {
	URI      string  `json:"uri"`
	Name     string  `json:"name"`
	MimeType *string `json:"mimeType,omitzero"`
}

// ReadMcpResourceDirOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresourcedir-2
type ReadMcpResourceDirOutput struct {
	Resources []ReadMcpResourceDirOutputResource `json:"resources"`
	Error     *string                            `json:"error,omitzero"`
}

func (ReadMcpResourceDirOutput) toolOutputSchemas() {}

// RefreshMcpToolsStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#refreshmcptools-2
type RefreshMcpToolsStatus string

const (
	RefreshMcpToolsStatusRefreshed    RefreshMcpToolsStatus = "refreshed"
	RefreshMcpToolsStatusError        RefreshMcpToolsStatus = "error"
	RefreshMcpToolsStatusNotConnected RefreshMcpToolsStatus = "not_connected"
)

// RefreshMcpToolsOutputEntry is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#refreshmcptools-2
type RefreshMcpToolsOutputEntry struct {
	Server    string                `json:"server"`
	Status    RefreshMcpToolsStatus `json:"status"`
	ToolCount *int64                `json:"toolCount,omitzero"`
	Added     []string              `json:"added,omitzero"`
	Removed   []string              `json:"removed,omitzero"`
	Error     *string               `json:"error,omitzero"`
}

// RefreshMcpToolsOutput is a handwritten Claude Agent SDK type. The upstream
// output is a bare array.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#refreshmcptools-2
type RefreshMcpToolsOutput []RefreshMcpToolsOutputEntry

func (RefreshMcpToolsOutput) toolOutputSchemas() {}

// ShowOnboardingRolePickerOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#showonboardingrolepicker-2
type ShowOnboardingRolePickerOutput struct {
	Role      *string `json:"role,omitzero"`
	Dismissed *bool   `json:"dismissed,omitzero"`
}

func (ShowOnboardingRolePickerOutput) toolOutputSchemas() {}

// McpOutput is a handwritten Claude Agent SDK type. Upstream types it as
// string | { type: string; ... }[] | { ... }, so the payload is preserved as
// raw JSON.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#mcpoutput
type McpOutput json.RawMessage

// MarshalJSON emits the preserved raw payload verbatim.
func (m McpOutput) MarshalJSON() ([]byte, error) { return marshalRawPayload(m) }

// UnmarshalJSON preserves the raw payload verbatim.
func (m *McpOutput) UnmarshalJSON(data []byte) error {
	return unmarshalRawPayload((*json.RawMessage)(m), data)
}

func (McpOutput) toolOutputSchemas() {}

// ArtifactOutput is a handwritten Claude Agent SDK type. It is an untagged
// union: the publish result carries url/path, the list result carries
// artifacts. Dispatch is by key presence.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactOutput struct {
	value ArtifactOutput_Value
}

// ArtifactOutput_Value is the variant interface implemented by every
// [ArtifactOutput] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactOutput_Value interface{ artifactOutput() }

// MarshalJSON marshals the active [ArtifactOutput] variant.
func (o ArtifactOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes an [ArtifactOutput] union value from JSON.
func (o *ArtifactOutput) UnmarshalJSON(data []byte) error {
	var probe struct {
		URL       *string         `json:"url"`
		Artifacts json.RawMessage `json:"artifacts"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}
	var (
		v   ArtifactOutput_Value
		err error
	)
	switch {
	case probe.URL != nil:
		v, err = decodeUnionVariant[ArtifactOutputPublished](data)
	case len(probe.Artifacts) > 0:
		v, err = decodeUnionVariant[ArtifactOutputListing](data)
	default:
		v, err = decodeUnionVariant[ArtifactOutputUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ ArtifactOutput_Value = (*ArtifactOutputUnknown)(nil)
	_ ArtifactOutput_Value = (*ArtifactOutputPublished)(nil)
	_ ArtifactOutput_Value = (*ArtifactOutputListing)(nil)
)

// NewArtifactOutput wraps a variant into an [ArtifactOutput].
func NewArtifactOutput(v ArtifactOutput_Value) ArtifactOutput { return ArtifactOutput{value: v} }

// GetValue returns the active [ArtifactOutput] variant, or nil when unset.
func (o ArtifactOutput) GetValue() ArtifactOutput_Value { return o.value }

// GetPublished reports whether the active variant is [*ArtifactOutputPublished] and returns it.
func (o ArtifactOutput) GetPublished() (*ArtifactOutputPublished, bool) {
	v, ok := o.value.(*ArtifactOutputPublished)
	return v, ok
}

// GetListing reports whether the active variant is [*ArtifactOutputListing] and returns it.
func (o ArtifactOutput) GetListing() (*ArtifactOutputListing, bool) {
	v, ok := o.value.(*ArtifactOutputListing)
	return v, ok
}

// GetUnknown reports whether the active variant is [*ArtifactOutputUnknown] and returns it.
func (o ArtifactOutput) GetUnknown() (*ArtifactOutputUnknown, bool) {
	v, ok := o.value.(*ArtifactOutputUnknown)
	return v, ok
}

// ArtifactOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactOutputUnknown struct{ UnknownUnion }

// ArtifactOutputStored is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactOutputStored struct {
	Contract     string         `json:"contract"`
	Capabilities map[string]any `json:"capabilities,omitzero"`
}

// ArtifactOutputPublished is the publish-action variant of [ArtifactOutput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactOutputPublished struct {
	URL              string                `json:"url"`
	Path             string                `json:"path"`
	Title            *string               `json:"title,omitzero"`
	Version          *string               `json:"version,omitzero"`
	Capabilities     json.RawMessage       `json:"capabilities,omitzero"`
	Stored           *ArtifactOutputStored `json:"stored,omitzero"`
	Warnings         []string              `json:"warnings,omitzero"`
	Contract         *string               `json:"contract,omitzero"`
	Updated          *bool                 `json:"updated,omitzero"`
	LiveSubscription *string               `json:"liveSubscription,omitzero"`
}

// ArtifactListRel is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactListRel string

const (
	ArtifactListRelMine   ArtifactListRel = "mine"
	ArtifactListRelShared ArtifactListRel = "shared"
)

// ArtifactListScope is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactListScope string

const (
	ArtifactListScopeShared ArtifactListScope = "shared"
	ArtifactListScopeAll    ArtifactListScope = "all"
)

// ArtifactOutputListingRow is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactOutputListingRow struct {
	Title     string           `json:"title"`
	URL       string           `json:"url"`
	UpdatedAt *string          `json:"updatedAt,omitzero"`
	Rel       *ArtifactListRel `json:"rel,omitzero"`
}

// ArtifactOutputListing is the list-action variant of [ArtifactOutput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#artifact-2
type ArtifactOutputListing struct {
	Artifacts []ArtifactOutputListingRow `json:"artifacts"`
	Truncated *bool                      `json:"truncated,omitzero"`
	Scope     *ArtifactListScope         `json:"scope,omitzero"`
}

func (ArtifactOutputUnknown) artifactOutput()   {}
func (ArtifactOutputPublished) artifactOutput() {}
func (ArtifactOutputListing) artifactOutput()   {}

func (ArtifactOutput) toolOutputSchemas() {}

// ProjectsOutput is a handwritten Claude Agent SDK type, discriminated on the
// method field, mirroring [ProjectsInput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutput struct {
	value ProjectsOutput_Value
}

// ProjectsOutput_Value is the variant interface implemented by every
// [ProjectsOutput] case.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutput_Value interface{ projectsOutput() }

// MarshalJSON marshals the active [ProjectsOutput] variant.
func (o ProjectsOutput) MarshalJSON() ([]byte, error) { return json.Marshal(o.value) }

// UnmarshalJSON decodes a [ProjectsOutput] union value from JSON.
func (o *ProjectsOutput) UnmarshalJSON(data []byte) error {
	var disc struct {
		Method ProjectsMethod `json:"method"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	var (
		v   ProjectsOutput_Value
		err error
	)
	switch disc.Method {
	case ProjectsMethodProjectInfo:
		v, err = decodeUnionVariant[ProjectsOutputInfo](data)
	case ProjectsMethodProjectRead:
		v, err = decodeUnionVariant[ProjectsOutputRead](data)
	case ProjectsMethodProjectSearch:
		v, err = decodeUnionVariant[ProjectsOutputSearch](data)
	case ProjectsMethodProjectWrite:
		v, err = decodeUnionVariant[ProjectsOutputWrite](data)
	case ProjectsMethodProjectDelete:
		v, err = decodeUnionVariant[ProjectsOutputDelete](data)
	default:
		v, err = decodeUnionVariant[ProjectsOutputUnknown](data)
	}
	if err != nil {
		return err
	}
	o.value = v
	return nil
}

var (
	_ ProjectsOutput_Value = (*ProjectsOutputUnknown)(nil)
	_ ProjectsOutput_Value = (*ProjectsOutputInfo)(nil)
	_ ProjectsOutput_Value = (*ProjectsOutputRead)(nil)
	_ ProjectsOutput_Value = (*ProjectsOutputSearch)(nil)
	_ ProjectsOutput_Value = (*ProjectsOutputWrite)(nil)
	_ ProjectsOutput_Value = (*ProjectsOutputDelete)(nil)
)

// NewProjectsOutput wraps a variant into a [ProjectsOutput].
func NewProjectsOutput(v ProjectsOutput_Value) ProjectsOutput { return ProjectsOutput{value: v} }

// GetValue returns the active [ProjectsOutput] variant, or nil when unset.
func (o ProjectsOutput) GetValue() ProjectsOutput_Value { return o.value }

// ProjectsOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputUnknown struct{ UnknownUnion }

// ProjectsOutputDoc is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputDoc struct {
	Path      string  `json:"path"`
	CreatedAt *string `json:"created_at"`
}

// ProjectsOutputFile is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputFile struct {
	Path      string  `json:"path"`
	FileKind  string  `json:"file_kind"`
	CreatedAt *string `json:"created_at"`
}

// ProjectsOutputSyncSource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputSyncSource struct {
	Type   *string        `json:"type"`
	Config map[string]any `json:"config"`
}

// ProjectsOutputKnowledge is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputKnowledge struct {
	KnowledgeSize    int64 `json:"knowledge_size"`
	MaxKnowledgeSize int64 `json:"max_knowledge_size"`
}

// ProjectsOutputInfo is the project_info variant of [ProjectsOutput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputInfo struct {
	Method       ProjectsMethod             `json:"method"`
	Notice       *string                    `json:"notice,omitzero"`
	Name         string                     `json:"name"`
	Description  string                     `json:"description"`
	Instructions string                     `json:"instructions"`
	Docs         []ProjectsOutputDoc        `json:"docs"`
	Files        []ProjectsOutputFile       `json:"files,omitzero"`
	SyncSources  []ProjectsOutputSyncSource `json:"sync_sources,omitzero"`
	Knowledge    ProjectsOutputKnowledge    `json:"knowledge"`
}

// ProjectsOutputRead is the project_read variant of [ProjectsOutput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputRead struct {
	Method    ProjectsMethod `json:"method"`
	Notice    *string        `json:"notice,omitzero"`
	Path      string         `json:"path"`
	FileKind  *string        `json:"file_kind,omitzero"`
	Content   *string        `json:"content,omitzero"`
	LocalFile *string        `json:"local_file,omitzero"`
	CreatedAt *string        `json:"created_at"`
}

// ProjectsOutputSearchHit is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputSearchHit struct {
	Name    *string `json:"name,omitzero"`
	DocUUID *string `json:"doc_uuid,omitzero"`
	Text    *string `json:"text,omitzero"`
}

// ProjectsOutputSearch is the project_search variant of [ProjectsOutput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputSearch struct {
	Method ProjectsMethod            `json:"method"`
	Notice *string                   `json:"notice,omitzero"`
	Rag    bool                      `json:"rag"`
	Hits   []ProjectsOutputSearchHit `json:"hits,omitzero"`
	Docs   []string                  `json:"docs,omitzero"`
}

// ProjectsOutputWrite is the project_write variant of [ProjectsOutput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputWrite struct {
	Method        ProjectsMethod `json:"method"`
	Notice        *string        `json:"notice,omitzero"`
	Path          string         `json:"path"`
	DocUUID       string         `json:"doc_uuid"`
	Replaced      bool           `json:"replaced"`
	PresentToUser *bool          `json:"present_to_user,omitzero"`
	LocalPath     *string        `json:"local_path,omitzero"`
}

// ProjectsOutputDelete is the project_delete variant of [ProjectsOutput].
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#projects-2
type ProjectsOutputDelete struct {
	Method  ProjectsMethod `json:"method"`
	Notice  *string        `json:"notice,omitzero"`
	Path    string         `json:"path"`
	Deleted bool           `json:"deleted"`
}

func (ProjectsOutputUnknown) projectsOutput() {}
func (ProjectsOutputInfo) projectsOutput()    {}
func (ProjectsOutputRead) projectsOutput()    {}
func (ProjectsOutputSearch) projectsOutput()  {}
func (ProjectsOutputWrite) projectsOutput()   {}
func (ProjectsOutputDelete) projectsOutput()  {}

func (ProjectsOutput) toolOutputSchemas() {}

func cloneRawMessage(v json.RawMessage) json.RawMessage {
	if v == nil {
		return nil
	}
	out := make([]byte, len(v))
	copy(out, v)
	return out
}

func mapToProtoStruct(v map[string]any) (*structpb.Struct, error) {
	if v == nil {
		return nil, nil
	}
	return structpb.NewStruct(v)
}

func protoStructToMap(v *structpb.Struct) map[string]any {
	if v == nil {
		return nil
	}
	return v.AsMap()
}

func protoOpt[T comparable](
	msg interface{ ProtoReflect() protoreflect.Message },
	fieldName protoreflect.Name,
	v T,
) *T {
	field := msg.ProtoReflect().Descriptor().Fields().ByName(fieldName)
	if field == nil || !msg.ProtoReflect().Has(field) {
		return nil
	}
	return new(v)
}

func baseEffortToProto(e *BaseHookInputEffort) *pb.BaseHookInputEffort {
	if e == nil {
		return nil
	}
	return &pb.BaseHookInputEffort{Level: e.Level}
}

func baseEffortFromProto(e *pb.BaseHookInputEffort) *BaseHookInputEffort {
	if e == nil {
		return nil
	}
	return &BaseHookInputEffort{Level: e.GetLevel()}
}

func backgroundTasksToProto(in []BackgroundTaskSummary) []*pb.BackgroundTaskSummary {
	if in == nil {
		return nil
	}
	out := make([]*pb.BackgroundTaskSummary, len(in))
	for i, t := range in {
		out[i] = &pb.BackgroundTaskSummary{
			Id:          t.ID,
			Type:        t.Type,
			Status:      t.Status,
			Description: t.Description,
			Command:     t.Command,
			AgentType:   t.AgentType,
			Server:      t.Server,
			Tool:        t.Tool,
			Name:        t.Name,
		}
	}
	return out
}

func backgroundTasksFromProto(in []*pb.BackgroundTaskSummary) []BackgroundTaskSummary {
	if in == nil {
		return nil
	}
	out := make([]BackgroundTaskSummary, len(in))
	for i, t := range in {
		out[i] = BackgroundTaskSummary{
			ID:          t.GetId(),
			Type:        t.GetType(),
			Status:      t.GetStatus(),
			Description: t.GetDescription(),
			Command:     protoOpt(t, "command", t.GetCommand()),
			AgentType:   protoOpt(t, "agent_type", t.GetAgentType()),
			Server:      protoOpt(t, "server", t.GetServer()),
			Tool:        protoOpt(t, "tool", t.GetTool()),
			Name:        protoOpt(t, "name", t.GetName()),
		}
	}
	return out
}

func sessionCronsToProto(in []SessionCronSummary) []*pb.SessionCronSummary {
	if in == nil {
		return nil
	}
	out := make([]*pb.SessionCronSummary, len(in))
	for i, c := range in {
		out[i] = &pb.SessionCronSummary{
			Id:        c.ID,
			Schedule:  c.Schedule,
			Recurring: c.Recurring,
			Prompt:    c.Prompt,
		}
	}
	return out
}

func sessionCronsFromProto(in []*pb.SessionCronSummary) []SessionCronSummary {
	if in == nil {
		return nil
	}
	out := make([]SessionCronSummary, len(in))
	for i, c := range in {
		out[i] = SessionCronSummary{
			ID:        c.GetId(),
			Schedule:  c.GetSchedule(),
			Recurring: c.GetRecurring(),
			Prompt:    c.GetPrompt(),
		}
	}
	return out
}

func rawJSONToProto(raw json.RawMessage) *pb.JsonRawMessage {
	if len(raw) == 0 {
		return nil
	}
	return &pb.JsonRawMessage{RawJson: cloneRawMessage(raw)}
}

func rawJSONFromProto(m *pb.JsonRawMessage) json.RawMessage {
	if m == nil {
		return nil
	}
	return cloneRawMessage(m.GetRawJson())
}

func postToolBatchToolCallsToProto(in []PostToolBatchToolCall) []*pb.PostToolBatchToolCall {
	if in == nil {
		return nil
	}
	out := make([]*pb.PostToolBatchToolCall, len(in))
	for i, c := range in {
		out[i] = &pb.PostToolBatchToolCall{
			ToolName:     c.ToolName,
			ToolInput:    rawJSONToProto(c.ToolInput),
			ToolUseId:    c.ToolUseID,
			ToolResponse: rawJSONToProto(c.ToolResponse),
		}
	}
	return out
}

func postToolBatchToolCallsFromProto(in []*pb.PostToolBatchToolCall) []PostToolBatchToolCall {
	if in == nil {
		return nil
	}
	out := make([]PostToolBatchToolCall, len(in))
	for i, c := range in {
		out[i] = PostToolBatchToolCall{
			ToolName:     c.GetToolName(),
			ToolInput:    rawJSONFromProto(c.GetToolInput()),
			ToolUseID:    c.GetToolUseId(),
			ToolResponse: rawJSONFromProto(c.GetToolResponse()),
		}
	}
	return out
}

// rawToolInputToProto marshals a typed ToolInputSchemas to JSON and wraps
// it as the proto UnknownVariant. Returns nil for nil input.
func rawToolInputToProto(v ToolInputSchemas, discriminator string) *pb.ToolInput {
	if v.value == nil {
		return nil
	}
	if p := codexToolInputToProto(v.value); p != nil {
		return p
	}
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return &pb.ToolInput{
		Input: &pb.ToolInput_Unknown{
			Unknown: &pb.UnknownVariant{
				Discriminator: discriminator,
				RawJson:       cloneRawMessage(raw),
			},
		},
	}
}

func rawToolOutputToProto(v ToolOutputSchemas, discriminator string) *pb.ToolOutput {
	if v.value == nil {
		return nil
	}
	if p := codexToolOutputToProto(v.value); p != nil {
		return p
	}
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return &pb.ToolOutput{
		Output: &pb.ToolOutput_Unknown{
			Unknown: &pb.UnknownVariant{
				Discriminator: discriminator,
				RawJson:       cloneRawMessage(raw),
			},
		},
	}
}

// rawFromToolInputProto extracts a typed ToolInputSchemas from the proto
// wrapper. The discriminator (tool name) drives the dispatch.
func rawFromToolInputProto(v *pb.ToolInput, discriminator string) (ToolInputSchemas, error) {
	if v == nil {
		return ToolInputSchemas{}, nil
	}
	if c, ok, err := codexToolInputFromProto(v); ok {
		return c, err
	}
	var raw json.RawMessage
	if unknown := v.GetUnknown(); unknown != nil {
		raw = cloneRawMessage(unknown.GetRawJson())
	} else {
		b, err := protojson.Marshal(v)
		if err != nil {
			return ToolInputSchemas{}, err
		}
		raw = b
	}
	var out ToolInputSchemas
	err := out.UnmarshalForTool(discriminator, raw)
	return out, err
}

func rawFromToolOutputProto(v *pb.ToolOutput, discriminator string) (ToolOutputSchemas, error) {
	if v == nil {
		return ToolOutputSchemas{}, nil
	}
	if c, ok, err := codexToolOutputFromProto(v); ok {
		return c, err
	}
	var raw json.RawMessage
	if unknown := v.GetUnknown(); unknown != nil {
		raw = cloneRawMessage(unknown.GetRawJson())
	} else {
		b, err := protojson.Marshal(v)
		if err != nil {
			return ToolOutputSchemas{}, err
		}
		raw = b
	}
	var out ToolOutputSchemas
	err := out.UnmarshalForTool(discriminator, raw)
	return out, err
}

func permissionRuleValuesToProto(v []PermissionRuleValue) []*pb.PermissionRuleValue {
	out := make([]*pb.PermissionRuleValue, 0, len(v))
	for _, rule := range v {
		out = append(out, &pb.PermissionRuleValue{
			ToolName:    rule.ToolName,
			RuleContent: rule.RuleContent,
		})
	}
	return out
}

func permissionRuleValuesFromProto(v []*pb.PermissionRuleValue) []PermissionRuleValue {
	out := make([]PermissionRuleValue, 0, len(v))
	for _, rule := range v {
		out = append(out, PermissionRuleValue{
			ToolName:    rule.GetToolName(),
			RuleContent: protoOpt(rule, "rule_content", rule.GetRuleContent()),
		})
	}
	return out
}

func permissionUpdatesToProto(v []PermissionUpdate) ([]*pb.PermissionUpdate, error) {
	out := make([]*pb.PermissionUpdate, 0, len(v))
	for _, update := range v {
		p, err := PermissionUpdateToProto(update)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func permissionUpdatesFromProto(v []*pb.PermissionUpdate) ([]PermissionUpdate, error) {
	out := make([]PermissionUpdate, 0, len(v))
	for _, update := range v {
		got, err := PermissionUpdateFromProto(update)
		if err != nil {
			return nil, err
		}
		out = append(out, got)
	}
	return out, nil
}

func (u UnknownUnion) MarshalJSON() ([]byte, error) {
	if len(u.Raw) > 0 {
		return u.Raw, nil
	}
	type alias struct {
		Discriminator string `json:"discriminator"`
	}
	return json.Marshal(alias{Discriminator: u.Discriminator})
}

func (u *UnknownUnion) UnmarshalJSON(data []byte) error {
	u.Raw = cloneRawMessage(data)
	// An unknown union member can be any JSON value, not just an object:
	// Codex, for example, sends apply_patch's tool_response as a bare
	// string. Only objects carry a discriminator, so attempt the lookup
	// but tolerate a non-object value — the raw bytes are preserved either
	// way, and failing here would abort the whole hook-input parse.
	var m map[string]json.RawMessage
	if json.Unmarshal(data, &m) != nil {
		return nil
	}
	for _, key := range []string{"type", "hookEventName", "hook_event_name", "status", "subtype", "behavior"} {
		if raw, ok := m[key]; ok {
			_ = json.Unmarshal(raw, &u.Discriminator)
			break
		}
	}
	return nil
}

func (u UnknownUnion) ToProto() *pb.UnknownVariant {
	return &pb.UnknownVariant{
		Discriminator: u.Discriminator,
		RawJson:       cloneRawMessage(u.Raw),
	}
}

func (u *UnknownUnion) FromProto(v *pb.UnknownVariant) {
	if v == nil {
		*u = UnknownUnion{}
		return
	}
	u.Discriminator = v.GetDiscriminator()
	u.Raw = cloneRawMessage(v.GetRawJson())
}

func (o PermissionUpdateUnknown) MarshalJSON() ([]byte, error) { return o.UnknownUnion.MarshalJSON() }
func (o *PermissionUpdateUnknown) UnmarshalJSON(data []byte) error {
	return o.UnknownUnion.UnmarshalJSON(data)
}

func (o PermissionUpdateAddRules) MarshalJSON() ([]byte, error) {
	type alias PermissionUpdateAddRules
	o.Type = "addRules"
	return json.Marshal(alias(o))
}

func (o *PermissionUpdateAddRules) UnmarshalJSON(data []byte) error {
	type alias PermissionUpdateAddRules
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionUpdateAddRules(v)
	o.Type = "addRules"
	return nil
}

func (o PermissionUpdateReplaceRules) MarshalJSON() ([]byte, error) {
	type alias PermissionUpdateReplaceRules
	o.Type = "replaceRules"
	return json.Marshal(alias(o))
}

func (o *PermissionUpdateReplaceRules) UnmarshalJSON(data []byte) error {
	type alias PermissionUpdateReplaceRules
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionUpdateReplaceRules(v)
	o.Type = "replaceRules"
	return nil
}

func (o PermissionUpdateRemoveRules) MarshalJSON() ([]byte, error) {
	type alias PermissionUpdateRemoveRules
	o.Type = "removeRules"
	return json.Marshal(alias(o))
}

func (o *PermissionUpdateRemoveRules) UnmarshalJSON(data []byte) error {
	type alias PermissionUpdateRemoveRules
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionUpdateRemoveRules(v)
	o.Type = "removeRules"
	return nil
}

func (o PermissionUpdateSetMode) MarshalJSON() ([]byte, error) {
	type alias PermissionUpdateSetMode
	o.Type = "setMode"
	return json.Marshal(alias(o))
}

func (o *PermissionUpdateSetMode) UnmarshalJSON(data []byte) error {
	type alias PermissionUpdateSetMode
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionUpdateSetMode(v)
	o.Type = "setMode"
	return nil
}

func (o PermissionUpdateAddDirectories) MarshalJSON() ([]byte, error) {
	type alias PermissionUpdateAddDirectories
	o.Type = "addDirectories"
	return json.Marshal(alias(o))
}

func (o *PermissionUpdateAddDirectories) UnmarshalJSON(data []byte) error {
	type alias PermissionUpdateAddDirectories
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionUpdateAddDirectories(v)
	o.Type = "addDirectories"
	return nil
}

func (o PermissionUpdateRemoveDirectories) MarshalJSON() ([]byte, error) {
	type alias PermissionUpdateRemoveDirectories
	o.Type = "removeDirectories"
	return json.Marshal(alias(o))
}

func (o *PermissionUpdateRemoveDirectories) UnmarshalJSON(data []byte) error {
	type alias PermissionUpdateRemoveDirectories
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionUpdateRemoveDirectories(v)
	o.Type = "removeDirectories"
	return nil
}

// Tool names recognized by the Claude Code SDK for typed dispatch in
// [ToolInputSchemas.UnmarshalForTool] / [ToolOutputSchemas.UnmarshalForTool].
// User-defined MCP tools share the [McpToolNamePrefix] prefix.
const (
	ToolNameAgent              = "Agent"
	ToolNameArtifact           = "Artifact"
	ToolNameAskUserQuestion    = "AskUserQuestion"
	ToolNameBash               = "Bash"
	ToolNameCronCreate         = "CronCreate"
	ToolNameCronDelete         = "CronDelete"
	ToolNameCronList           = "CronList"
	ToolNameEdit               = "Edit"
	ToolNameEnterPlanMode      = "EnterPlanMode"
	ToolNameEnterWorktree      = "EnterWorktree"
	ToolNameExitPlanMode       = "ExitPlanMode"
	ToolNameExitWorktree       = "ExitWorktree"
	ToolNameGlob               = "Glob"
	ToolNameGrep               = "Grep"
	ToolNameListMcpResources   = "ListMcpResourcesTool"
	ToolNameMonitor            = "Monitor"
	ToolNameNotebookEdit       = "NotebookEdit"
	ToolNameProjects           = "Projects"
	ToolNamePushNotification   = "PushNotification"
	ToolNameRead               = "Read"
	ToolNameReadMcpResource    = "ReadMcpResourceTool"
	ToolNameReadMcpResourceDir = "ReadMcpResourceDirTool"
	ToolNameRefreshMcpTools    = "RefreshMcpTools"
	ToolNameRemoteTrigger      = "RemoteTrigger"
	ToolNameREPL               = "REPL"
	ToolNameReportFindings     = "ReportFindings"
	ToolNameScheduleWakeup     = "ScheduleWakeup"
	// ToolNameTask is the legacy alias for ToolNameAgent; still accepted on input.
	ToolNameTask                     = "Task"
	ToolNameTaskCreate               = "TaskCreate"
	ToolNameTaskGet                  = "TaskGet"
	ToolNameTaskList                 = "TaskList"
	ToolNameTaskOutput               = "TaskOutput"
	ToolNameTaskStop                 = "TaskStop"
	ToolNameTaskUpdate               = "TaskUpdate"
	ToolNameTodoWrite                = "TodoWrite"
	ToolNameShowOnboardingRolePicker = "ShowOnboardingRolePicker"
	ToolNameWebFetch                 = "WebFetch"
	ToolNameWebSearch                = "WebSearch"
	ToolNameWorkflow                 = "Workflow"
	ToolNameWrite                    = "Write"
)

// McpToolNamePrefix marks user-defined MCP tools, e.g. "mcp__server__tool".
const McpToolNamePrefix = "mcp__"

func PermissionUpdateToProto(v PermissionUpdate) (*pb.PermissionUpdate, error) {
	switch x := v.value.(type) {
	case *PermissionUpdateUnknown:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_Unknown{Unknown: x.ToProto()}}, nil
	case *PermissionUpdateAddRules:
		return &pb.PermissionUpdate{
			Value: &pb.PermissionUpdate_AddRules{AddRules: &pb.PermissionUpdateAddRules{
				Type:        "addRules",
				Rules:       permissionRuleValuesToProto(x.Rules),
				Behavior:    string(x.Behavior),
				Destination: string(x.Destination),
			}},
		}, nil
	case *PermissionUpdateReplaceRules:
		return &pb.PermissionUpdate{
			Value: &pb.PermissionUpdate_ReplaceRules{ReplaceRules: &pb.PermissionUpdateReplaceRules{
				Type:        "replaceRules",
				Rules:       permissionRuleValuesToProto(x.Rules),
				Behavior:    string(x.Behavior),
				Destination: string(x.Destination),
			}},
		}, nil
	case *PermissionUpdateRemoveRules:
		return &pb.PermissionUpdate{
			Value: &pb.PermissionUpdate_RemoveRules{RemoveRules: &pb.PermissionUpdateRemoveRules{
				Type:        "removeRules",
				Rules:       permissionRuleValuesToProto(x.Rules),
				Behavior:    string(x.Behavior),
				Destination: string(x.Destination),
			}},
		}, nil
	case *PermissionUpdateSetMode:
		return &pb.PermissionUpdate{
			Value: &pb.PermissionUpdate_SetMode{SetMode: &pb.PermissionUpdateSetMode{
				Type:        "setMode",
				Mode:        string(x.Mode),
				Destination: string(x.Destination),
			}},
		}, nil
	case *PermissionUpdateAddDirectories:
		return &pb.PermissionUpdate{
			Value: &pb.PermissionUpdate_AddDirectories{
				AddDirectories: &pb.PermissionUpdateAddDirectories{
					Type:        "addDirectories",
					Directories: append([]string(nil), x.Directories...),
					Destination: string(x.Destination),
				},
			},
		}, nil
	case *PermissionUpdateRemoveDirectories:
		return &pb.PermissionUpdate{
			Value: &pb.PermissionUpdate_RemoveDirectories{
				RemoveDirectories: &pb.PermissionUpdateRemoveDirectories{
					Type:        "removeDirectories",
					Directories: append([]string(nil), x.Directories...),
					Destination: string(x.Destination),
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported PermissionUpdate %T", v)
	}
}

func PermissionUpdateFromProto(v *pb.PermissionUpdate) (PermissionUpdate, error) {
	if v == nil {
		return PermissionUpdate{}, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.PermissionUpdate_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return NewPermissionUpdate(&PermissionUpdateUnknown{UnknownUnion: u}), nil
	case *pb.PermissionUpdate_AddRules:
		addRules := v.GetAddRules()
		return NewPermissionUpdate(&PermissionUpdateAddRules{
			Type:        "addRules",
			Rules:       permissionRuleValuesFromProto(addRules.GetRules()),
			Behavior:    PermissionBehavior(addRules.GetBehavior()),
			Destination: PermissionUpdateDestination(addRules.GetDestination()),
		}), nil
	case *pb.PermissionUpdate_ReplaceRules:
		replaceRules := v.GetReplaceRules()
		return NewPermissionUpdate(&PermissionUpdateReplaceRules{
			Type:        "replaceRules",
			Rules:       permissionRuleValuesFromProto(replaceRules.GetRules()),
			Behavior:    PermissionBehavior(replaceRules.GetBehavior()),
			Destination: PermissionUpdateDestination(replaceRules.GetDestination()),
		}), nil
	case *pb.PermissionUpdate_RemoveRules:
		removeRules := v.GetRemoveRules()
		return NewPermissionUpdate(&PermissionUpdateRemoveRules{
			Type:        "removeRules",
			Rules:       permissionRuleValuesFromProto(removeRules.GetRules()),
			Behavior:    PermissionBehavior(removeRules.GetBehavior()),
			Destination: PermissionUpdateDestination(removeRules.GetDestination()),
		}), nil
	case *pb.PermissionUpdate_SetMode:
		setMode := v.GetSetMode()
		return NewPermissionUpdate(&PermissionUpdateSetMode{
			Type:        "setMode",
			Mode:        PermissionMode(setMode.GetMode()),
			Destination: PermissionUpdateDestination(setMode.GetDestination()),
		}), nil
	case *pb.PermissionUpdate_AddDirectories:
		addDirectories := v.GetAddDirectories()
		return NewPermissionUpdate(&PermissionUpdateAddDirectories{
			Type:        "addDirectories",
			Directories: append([]string(nil), addDirectories.GetDirectories()...),
			Destination: PermissionUpdateDestination(addDirectories.GetDestination()),
		}), nil
	case *pb.PermissionUpdate_RemoveDirectories:
		removeDirectories := v.GetRemoveDirectories()
		return NewPermissionUpdate(&PermissionUpdateRemoveDirectories{
			Type:        "removeDirectories",
			Directories: append([]string(nil), removeDirectories.GetDirectories()...),
			Destination: PermissionUpdateDestination(removeDirectories.GetDestination()),
		}), nil
	default:
		return PermissionUpdate{}, fmt.Errorf("unsupported proto PermissionUpdate %T", x)
	}
}

func (u UnknownUnion) ToProtoPermissionUnknown() *pb.UnknownVariant { return u.ToProto() }

func (o PermissionResultUnknown) MarshalJSON() ([]byte, error) { return o.UnknownUnion.MarshalJSON() }
func (o *PermissionResultUnknown) UnmarshalJSON(data []byte) error {
	return o.UnknownUnion.UnmarshalJSON(data)
}

func (o PermissionResultAllow) MarshalJSON() ([]byte, error) {
	type alias PermissionResultAllow
	o.Behavior = PermissionResultBehaviorAllow
	return json.Marshal(alias(o))
}

func (o *PermissionResultAllow) UnmarshalJSON(data []byte) error {
	type alias PermissionResultAllow
	type raw struct {
		alias
		UpdatedPermissions []json.RawMessage `json:"updatedPermissions,omitzero"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	updates := make([]PermissionUpdate, 0, len(v.UpdatedPermissions))
	for _, rawUpdate := range v.UpdatedPermissions {
		var update PermissionUpdate
		err := update.UnmarshalJSON(rawUpdate)
		if err != nil {
			return err
		}
		updates = append(updates, update)
	}
	*o = PermissionResultAllow(v.alias)
	o.Behavior = PermissionResultBehaviorAllow
	o.UpdatedPermissions = updates
	return nil
}

func (o PermissionResultDeny) MarshalJSON() ([]byte, error) {
	type alias PermissionResultDeny
	o.Behavior = PermissionResultBehaviorDeny
	return json.Marshal(alias(o))
}

func (o *PermissionResultDeny) UnmarshalJSON(data []byte) error {
	type alias PermissionResultDeny
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionResultDeny(v)
	o.Behavior = PermissionResultBehaviorDeny
	return nil
}

func PermissionResultToProto(v PermissionResult) (*pb.PermissionResult, error) {
	switch x := v.value.(type) {
	case *PermissionResultUnknown:
		return &pb.PermissionResult{Value: &pb.PermissionResult_Unknown{Unknown: x.ToProto()}}, nil
	case *PermissionResultAllow:
		updatedInput, err := mapToProtoStruct(x.UpdatedInput)
		if err != nil {
			return nil, err
		}
		updatedPermissions, err := permissionUpdatesToProto(x.UpdatedPermissions)
		if err != nil {
			return nil, err
		}
		return &pb.PermissionResult{
			Value: &pb.PermissionResult_Allow{Allow: &pb.PermissionResultAllow{
				Behavior:           permissionResultBehaviorToProto(PermissionResultBehaviorAllow),
				UpdatedInput:       updatedInput,
				UpdatedPermissions: updatedPermissions,
				ToolUseId:          x.ToolUseID,
			}},
		}, nil
	case *PermissionResultDeny:
		return &pb.PermissionResult{Value: &pb.PermissionResult_Deny{Deny: &pb.PermissionResultDeny{
			Behavior:  permissionResultBehaviorToProto(PermissionResultBehaviorDeny),
			Message:   x.Message,
			Interrupt: x.Interrupt,
			ToolUseId: x.ToolUseID,
		}}}, nil
	default:
		return nil, fmt.Errorf("unsupported PermissionResult %T", v)
	}
}

func PermissionResultFromProto(v *pb.PermissionResult) (PermissionResult, error) {
	if v == nil {
		return PermissionResult{}, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.PermissionResult_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return NewPermissionResult(&PermissionResultUnknown{UnknownUnion: u}), nil
	case *pb.PermissionResult_Allow:
		allow := v.GetAllow()
		updates, err := permissionUpdatesFromProto(allow.GetUpdatedPermissions())
		if err != nil {
			return PermissionResult{}, err
		}
		return NewPermissionResult(&PermissionResultAllow{
			Behavior:           permissionResultBehaviorFromProto(allow.GetBehavior()),
			UpdatedInput:       protoStructToMap(allow.GetUpdatedInput()),
			UpdatedPermissions: updates,
			ToolUseID:          protoOpt(allow, "tool_use_id", allow.GetToolUseId()),
		}), nil
	case *pb.PermissionResult_Deny:
		deny := v.GetDeny()
		return NewPermissionResult(&PermissionResultDeny{
			Behavior:  permissionResultBehaviorFromProto(deny.GetBehavior()),
			Message:   deny.GetMessage(),
			Interrupt: protoOpt(deny, "interrupt", deny.GetInterrupt()),
			ToolUseID: protoOpt(deny, "tool_use_id", deny.GetToolUseId()),
		}), nil
	default:
		return PermissionResult{}, fmt.Errorf("unsupported proto PermissionResult %T", x)
	}
}

func (o PermissionRequestDecisionUnknown) MarshalJSON() ([]byte, error) {
	return o.UnknownUnion.MarshalJSON()
}

func (o *PermissionRequestDecisionUnknown) UnmarshalJSON(data []byte) error {
	return o.UnknownUnion.UnmarshalJSON(data)
}

func (o PermissionRequestDecisionAllow) MarshalJSON() ([]byte, error) {
	type alias PermissionRequestDecisionAllow
	o.Behavior = PermissionRequestDecisionBehaviorAllow
	return json.Marshal(alias(o))
}

func (o *PermissionRequestDecisionAllow) UnmarshalJSON(data []byte) error {
	type alias PermissionRequestDecisionAllow
	type raw struct {
		alias
		UpdatedPermissions []json.RawMessage `json:"updatedPermissions,omitzero"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	updates := make([]PermissionUpdate, 0, len(v.UpdatedPermissions))
	for _, rawUpdate := range v.UpdatedPermissions {
		var update PermissionUpdate
		err := update.UnmarshalJSON(rawUpdate)
		if err != nil {
			return err
		}
		updates = append(updates, update)
	}
	*o = PermissionRequestDecisionAllow(v.alias)
	o.Behavior = PermissionRequestDecisionBehaviorAllow
	o.UpdatedPermissions = updates
	return nil
}

func (o PermissionRequestDecisionDeny) MarshalJSON() ([]byte, error) {
	type alias PermissionRequestDecisionDeny
	o.Behavior = PermissionRequestDecisionBehaviorDeny
	return json.Marshal(alias(o))
}

func (o *PermissionRequestDecisionDeny) UnmarshalJSON(data []byte) error {
	type alias PermissionRequestDecisionDeny
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = PermissionRequestDecisionDeny(v)
	o.Behavior = PermissionRequestDecisionBehaviorDeny
	return nil
}

func PermissionRequestDecisionToProto(
	v PermissionRequestDecision,
) (*pb.PermissionRequestDecision, error) {
	switch x := v.value.(type) {
	case *PermissionRequestDecisionUnknown:
		return &pb.PermissionRequestDecision{
			Value: &pb.PermissionRequestDecision_Unknown{Unknown: x.ToProto()},
		}, nil
	case *PermissionRequestDecisionAllow:
		updatedInput, err := mapToProtoStruct(x.UpdatedInput)
		if err != nil {
			return nil, err
		}
		updatedPermissions, err := permissionUpdatesToProto(x.UpdatedPermissions)
		if err != nil {
			return nil, err
		}
		return &pb.PermissionRequestDecision{
			Value: &pb.PermissionRequestDecision_Allow{Allow: &pb.PermissionRequestDecisionAllow{
				Behavior: permissionRequestDecisionBehaviorToProto(
					PermissionRequestDecisionBehaviorAllow,
				),
				UpdatedInput:       updatedInput,
				UpdatedPermissions: updatedPermissions,
			}},
		}, nil
	case *PermissionRequestDecisionDeny:
		return &pb.PermissionRequestDecision{
			Value: &pb.PermissionRequestDecision_Deny{Deny: &pb.PermissionRequestDecisionDeny{
				Behavior: permissionRequestDecisionBehaviorToProto(
					PermissionRequestDecisionBehaviorDeny,
				),
				Message:   x.Message,
				Interrupt: x.Interrupt,
			}},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported PermissionRequestDecision %T", v)
	}
}

func PermissionRequestDecisionFromProto(
	v *pb.PermissionRequestDecision,
) (PermissionRequestDecision, error) {
	if v == nil {
		return PermissionRequestDecision{}, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.PermissionRequestDecision_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return NewPermissionRequestDecision(&PermissionRequestDecisionUnknown{UnknownUnion: u}), nil
	case *pb.PermissionRequestDecision_Allow:
		allow := v.GetAllow()
		updates, err := permissionUpdatesFromProto(allow.GetUpdatedPermissions())
		if err != nil {
			return PermissionRequestDecision{}, err
		}
		return NewPermissionRequestDecision(&PermissionRequestDecisionAllow{
			Behavior:           permissionRequestDecisionBehaviorFromProto(allow.GetBehavior()),
			UpdatedInput:       protoStructToMap(allow.GetUpdatedInput()),
			UpdatedPermissions: updates,
		}), nil
	case *pb.PermissionRequestDecision_Deny:
		deny := v.GetDeny()
		return NewPermissionRequestDecision(&PermissionRequestDecisionDeny{
			Behavior:  permissionRequestDecisionBehaviorFromProto(deny.GetBehavior()),
			Message:   protoOpt(deny, "message", deny.GetMessage()),
			Interrupt: protoOpt(deny, "interrupt", deny.GetInterrupt()),
		}), nil
	default:
		return PermissionRequestDecision{}, fmt.Errorf(
			"unsupported proto PermissionRequestDecision %T",
			x,
		)
	}
}

func (o HookSpecificOutputUnknown) MarshalJSON() ([]byte, error) { return o.UnknownUnion.MarshalJSON() }
func (o *HookSpecificOutputUnknown) UnmarshalJSON(data []byte) error {
	return o.UnknownUnion.UnmarshalJSON(data)
}

func (o HookSpecificOutputPreToolUse) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputPreToolUse
	o.HookEventName = HookEventPreToolUse
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputPreToolUse) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputPreToolUse
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputPreToolUse(v)
	o.HookEventName = HookEventPreToolUse
	return nil
}

func (o HookSpecificOutputUserPromptSubmit) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputUserPromptSubmit
	o.HookEventName = HookEventUserPromptSubmit
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputUserPromptSubmit) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputUserPromptSubmit
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputUserPromptSubmit(v)
	o.HookEventName = HookEventUserPromptSubmit
	return nil
}

func (o HookSpecificOutputSessionStart) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputSessionStart
	o.HookEventName = HookEventSessionStart
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputSessionStart) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputSessionStart
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputSessionStart(v)
	o.HookEventName = HookEventSessionStart
	return nil
}

func (o HookSpecificOutputSetup) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputSetup
	o.HookEventName = HookEventSetup
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputSetup) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputSetup
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputSetup(v)
	o.HookEventName = HookEventSetup
	return nil
}

func (o HookSpecificOutputSubagentStart) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputSubagentStart
	o.HookEventName = HookEventSubagentStart
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputSubagentStart) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputSubagentStart
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputSubagentStart(v)
	o.HookEventName = HookEventSubagentStart
	return nil
}

func (o HookSpecificOutputPostToolUse) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputPostToolUse
	o.HookEventName = HookEventPostToolUse
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputPostToolUse) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputPostToolUse
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputPostToolUse(v)
	o.HookEventName = HookEventPostToolUse
	return nil
}

func (o HookSpecificOutputPostToolUseFailure) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputPostToolUseFailure
	o.HookEventName = HookEventPostToolUseFailure
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputPostToolUseFailure) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputPostToolUseFailure
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputPostToolUseFailure(v)
	o.HookEventName = HookEventPostToolUseFailure
	return nil
}

func (o HookSpecificOutputPostToolBatch) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputPostToolBatch
	o.HookEventName = HookEventPostToolBatch
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputPostToolBatch) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputPostToolBatch
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputPostToolBatch(v)
	o.HookEventName = HookEventPostToolBatch
	return nil
}

func (o HookSpecificOutputNotification) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputNotification
	o.HookEventName = HookEventNotification
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputNotification) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputNotification
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputNotification(v)
	o.HookEventName = HookEventNotification
	return nil
}

func (o HookSpecificOutputUserPromptExpansion) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputUserPromptExpansion
	o.HookEventName = HookEventUserPromptExpansion
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputUserPromptExpansion) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputUserPromptExpansion
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputUserPromptExpansion(v)
	o.HookEventName = HookEventUserPromptExpansion
	return nil
}

func (o HookSpecificOutputStop) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputStop
	o.HookEventName = HookEventStop
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputStop) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputStop
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputStop(v)
	o.HookEventName = HookEventStop
	return nil
}

func (o HookSpecificOutputSubagentStop) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputSubagentStop
	o.HookEventName = HookEventSubagentStop
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputSubagentStop) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputSubagentStop
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputSubagentStop(v)
	o.HookEventName = HookEventSubagentStop
	return nil
}

func (o HookSpecificOutputPermissionDenied) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputPermissionDenied
	o.HookEventName = HookEventPermissionDenied
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputPermissionDenied) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputPermissionDenied
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputPermissionDenied(v)
	o.HookEventName = HookEventPermissionDenied
	return nil
}

func (o HookSpecificOutputElicitation) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputElicitation
	o.HookEventName = HookEventElicitation
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputElicitation) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputElicitation
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputElicitation(v)
	o.HookEventName = HookEventElicitation
	return nil
}

func (o HookSpecificOutputElicitationResult) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputElicitationResult
	o.HookEventName = HookEventElicitationResult
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputElicitationResult) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputElicitationResult
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputElicitationResult(v)
	o.HookEventName = HookEventElicitationResult
	return nil
}

func (o HookSpecificOutputCwdChanged) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputCwdChanged
	o.HookEventName = HookEventCwdChanged
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputCwdChanged) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputCwdChanged
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputCwdChanged(v)
	o.HookEventName = HookEventCwdChanged
	return nil
}

func (o HookSpecificOutputFileChanged) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputFileChanged
	o.HookEventName = HookEventFileChanged
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputFileChanged) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputFileChanged
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputFileChanged(v)
	o.HookEventName = HookEventFileChanged
	return nil
}

func (o HookSpecificOutputWorktreeCreate) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputWorktreeCreate
	o.HookEventName = HookEventWorktreeCreate
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputWorktreeCreate) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputWorktreeCreate
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputWorktreeCreate(v)
	o.HookEventName = HookEventWorktreeCreate
	return nil
}

func (o HookSpecificOutputMessageDisplay) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputMessageDisplay
	o.HookEventName = HookEventMessageDisplay
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputMessageDisplay) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputMessageDisplay
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = HookSpecificOutputMessageDisplay(v)
	o.HookEventName = HookEventMessageDisplay
	return nil
}

func (o HookSpecificOutputPermissionRequest) MarshalJSON() ([]byte, error) {
	type alias HookSpecificOutputPermissionRequest
	o.HookEventName = HookEventPermissionRequest
	return json.Marshal(alias(o))
}

func (o *HookSpecificOutputPermissionRequest) UnmarshalJSON(data []byte) error {
	type alias HookSpecificOutputPermissionRequest
	type raw struct {
		alias
		Decision json.RawMessage `json:"decision"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var dec PermissionRequestDecision
	if len(v.Decision) > 0 {
		if err := dec.UnmarshalJSON(v.Decision); err != nil {
			return err
		}
	}
	*o = HookSpecificOutputPermissionRequest(v.alias)
	o.HookEventName = HookEventPermissionRequest
	o.Decision = dec
	return nil
}

func HookSpecificOutputToProto(v HookSpecificOutput) (*pb.HookSpecificOutput, error) {
	switch x := v.value.(type) {
	case *HookSpecificOutputUnknown:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_Unknown{Unknown: x.ToProto()},
		}, nil
	case *HookSpecificOutputPreToolUse:
		updatedInput, err := mapToProtoStruct(x.UpdatedInput)
		if err != nil {
			return nil, err
		}
		var permissionDecision *string
		if x.PermissionDecision != nil {
			s := string(*x.PermissionDecision)
			permissionDecision = &s
		}
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_PreToolUse{PreToolUse: &pb.HookSpecificOutputPreToolUse{
				HookEventName:            string(HookEventPreToolUse),
				PermissionDecision:       permissionDecision,
				PermissionDecisionReason: x.PermissionDecisionReason,
				UpdatedInput:             updatedInput,
				AdditionalContext:        x.AdditionalContext,
			}},
		}, nil
	case *HookSpecificOutputUserPromptSubmit:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_UserPromptSubmit{
				UserPromptSubmit: &pb.HookSpecificOutputUserPromptSubmit{
					HookEventName: string(
						HookEventUserPromptSubmit,
					),
					AdditionalContext: x.AdditionalContext,
				},
			},
		}, nil
	case *HookSpecificOutputSessionStart:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_SessionStart{
				SessionStart: &pb.HookSpecificOutputSessionStart{
					HookEventName: string(
						HookEventSessionStart,
					),
					AdditionalContext: x.AdditionalContext,
				},
			},
		}, nil
	case *HookSpecificOutputSetup:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_Setup{Setup: &pb.HookSpecificOutputSetup{
				HookEventName: string(HookEventSetup), AdditionalContext: x.AdditionalContext,
			}},
		}, nil
	case *HookSpecificOutputSubagentStart:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_SubagentStart{
				SubagentStart: &pb.HookSpecificOutputSubagentStart{
					HookEventName: string(
						HookEventSubagentStart,
					),
					AdditionalContext: x.AdditionalContext,
				},
			},
		}, nil
	case *HookSpecificOutputPostToolUse:
		var updatedMCP *pb.JsonRawMessage
		if len(x.UpdatedMCPToolOutput) > 0 {
			updatedMCP = &pb.JsonRawMessage{RawJson: cloneRawMessage(x.UpdatedMCPToolOutput)}
		}
		var updated *pb.JsonRawMessage
		if len(x.UpdatedToolOutput) > 0 {
			updated = &pb.JsonRawMessage{RawJson: cloneRawMessage(x.UpdatedToolOutput)}
		}
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_PostToolUse{
				PostToolUse: &pb.HookSpecificOutputPostToolUse{
					HookEventName: string(
						HookEventPostToolUse,
					),
					AdditionalContext:    x.AdditionalContext,
					UpdatedToolOutput:    updated,
					UpdatedMcpToolOutput: updatedMCP,
				},
			},
		}, nil
	case *HookSpecificOutputPostToolBatch:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_PostToolBatch{
				PostToolBatch: &pb.HookSpecificOutputPostToolBatch{
					HookEventName:     string(HookEventPostToolBatch),
					AdditionalContext: x.AdditionalContext,
				},
			},
		}, nil
	case *HookSpecificOutputPostToolUseFailure:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_PostToolUseFailure{
				PostToolUseFailure: &pb.HookSpecificOutputPostToolUseFailure{
					HookEventName: string(
						HookEventPostToolUseFailure,
					),
					AdditionalContext: x.AdditionalContext,
				},
			},
		}, nil
	case *HookSpecificOutputNotification:
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_Notification{
				Notification: &pb.HookSpecificOutputNotification{
					HookEventName: string(
						HookEventNotification,
					),
					AdditionalContext: x.AdditionalContext,
				},
			},
		}, nil
	case *HookSpecificOutputPermissionRequest:
		dec, err := PermissionRequestDecisionToProto(x.Decision)
		if err != nil {
			return nil, err
		}
		return &pb.HookSpecificOutput{
			Value: &pb.HookSpecificOutput_PermissionRequest{
				PermissionRequest: &pb.HookSpecificOutputPermissionRequest{
					HookEventName: string(HookEventPermissionRequest), Decision: dec,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported HookSpecificOutput %T", v)
	}
}

func HookSpecificOutputFromProto(v *pb.HookSpecificOutput) (HookSpecificOutput, error) {
	if v == nil {
		return HookSpecificOutput{}, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.HookSpecificOutput_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return NewHookSpecificOutput(&HookSpecificOutputUnknown{UnknownUnion: u}), nil
	case *pb.HookSpecificOutput_PreToolUse:
		preToolUse := v.GetPreToolUse()
		var pd *HookPermissionDecision
		permissionDecision := preToolUse.GetPermissionDecision()
		if permissionDecision != "" {
			t := HookPermissionDecision(permissionDecision)
			pd = &t
		}
		return NewHookSpecificOutput(&HookSpecificOutputPreToolUse{
			HookEventName:      HookEventPreToolUse,
			PermissionDecision: pd,
			PermissionDecisionReason: protoOpt(
				preToolUse,
				"permission_decision_reason",
				preToolUse.GetPermissionDecisionReason(),
			),
			UpdatedInput: protoStructToMap(preToolUse.GetUpdatedInput()),
			AdditionalContext: protoOpt(
				preToolUse,
				"additional_context",
				preToolUse.GetAdditionalContext(),
			),
		}), nil
	case *pb.HookSpecificOutput_UserPromptSubmit:
		userPromptSubmit := v.GetUserPromptSubmit()
		return NewHookSpecificOutput(&HookSpecificOutputUserPromptSubmit{
			HookEventName: HookEventUserPromptSubmit,
			AdditionalContext: protoOpt(
				userPromptSubmit,
				"additional_context",
				userPromptSubmit.GetAdditionalContext(),
			),
		}), nil
	case *pb.HookSpecificOutput_SessionStart:
		sessionStart := v.GetSessionStart()
		return NewHookSpecificOutput(&HookSpecificOutputSessionStart{
			HookEventName: HookEventSessionStart,
			AdditionalContext: protoOpt(
				sessionStart,
				"additional_context",
				sessionStart.GetAdditionalContext(),
			),
		}), nil
	case *pb.HookSpecificOutput_Setup:
		setup := v.GetSetup()
		return NewHookSpecificOutput(&HookSpecificOutputSetup{
			HookEventName:     HookEventSetup,
			AdditionalContext: protoOpt(setup, "additional_context", setup.GetAdditionalContext()),
		}), nil
	case *pb.HookSpecificOutput_SubagentStart:
		subagentStart := v.GetSubagentStart()
		return NewHookSpecificOutput(&HookSpecificOutputSubagentStart{
			HookEventName: HookEventSubagentStart,
			AdditionalContext: protoOpt(
				subagentStart,
				"additional_context",
				subagentStart.GetAdditionalContext(),
			),
		}), nil
	case *pb.HookSpecificOutput_PostToolUse:
		postToolUse := v.GetPostToolUse()
		var updatedMCP json.RawMessage
		if postToolUse.GetUpdatedMcpToolOutput() != nil {
			updatedMCP = cloneRawMessage(postToolUse.GetUpdatedMcpToolOutput().GetRawJson())
		}
		var updated json.RawMessage
		if postToolUse.GetUpdatedToolOutput() != nil {
			updated = cloneRawMessage(postToolUse.GetUpdatedToolOutput().GetRawJson())
		}
		return NewHookSpecificOutput(&HookSpecificOutputPostToolUse{
			HookEventName: HookEventPostToolUse,
			AdditionalContext: protoOpt(
				postToolUse,
				"additional_context",
				postToolUse.GetAdditionalContext(),
			),
			UpdatedToolOutput:    updated,
			UpdatedMCPToolOutput: updatedMCP,
		}), nil
	case *pb.HookSpecificOutput_PostToolUseFailure:
		postToolUseFailure := v.GetPostToolUseFailure()
		return NewHookSpecificOutput(&HookSpecificOutputPostToolUseFailure{
			HookEventName: HookEventPostToolUseFailure,
			AdditionalContext: protoOpt(
				postToolUseFailure,
				"additional_context",
				postToolUseFailure.GetAdditionalContext(),
			),
		}), nil
	case *pb.HookSpecificOutput_PostToolBatch:
		postToolBatch := v.GetPostToolBatch()
		return NewHookSpecificOutput(&HookSpecificOutputPostToolBatch{
			HookEventName: HookEventPostToolBatch,
			AdditionalContext: protoOpt(
				postToolBatch,
				"additional_context",
				postToolBatch.GetAdditionalContext(),
			),
		}), nil
	case *pb.HookSpecificOutput_Notification:
		notification := v.GetNotification()
		return NewHookSpecificOutput(&HookSpecificOutputNotification{
			HookEventName: HookEventNotification,
			AdditionalContext: protoOpt(
				notification,
				"additional_context",
				notification.GetAdditionalContext(),
			),
		}), nil
	case *pb.HookSpecificOutput_PermissionRequest:
		dec, err := PermissionRequestDecisionFromProto(v.GetPermissionRequest().GetDecision())
		if err != nil {
			return HookSpecificOutput{}, err
		}
		return NewHookSpecificOutput(&HookSpecificOutputPermissionRequest{
			HookEventName: HookEventPermissionRequest,
			Decision:      dec,
		}), nil
	default:
		return HookSpecificOutput{}, fmt.Errorf("unsupported proto HookSpecificOutput %T", x)
	}
}

func (o HookJSONOutputUnknown) MarshalJSON() ([]byte, error) { return o.UnknownUnion.MarshalJSON() }
func (o *HookJSONOutputUnknown) UnmarshalJSON(data []byte) error {
	return o.UnknownUnion.UnmarshalJSON(data)
}

func (o AsyncHookJSONOutput) MarshalJSON() ([]byte, error) {
	type alias AsyncHookJSONOutput
	return json.Marshal(alias(o))
}

func (o *AsyncHookJSONOutput) UnmarshalJSON(data []byte) error {
	type alias AsyncHookJSONOutput
	var v alias
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = AsyncHookJSONOutput(v)
	return nil
}

func (o SyncHookJSONOutput) MarshalJSON() ([]byte, error) {
	type alias SyncHookJSONOutput
	return json.Marshal(alias(o))
}

func (o *SyncHookJSONOutput) UnmarshalJSON(data []byte) error {
	type alias SyncHookJSONOutput
	type raw struct {
		alias
		HookSpecificOutput json.RawMessage `json:"hookSpecificOutput"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var hso HookSpecificOutput
	if len(v.HookSpecificOutput) > 0 {
		if err := hso.UnmarshalJSON(v.HookSpecificOutput); err != nil {
			return err
		}
	}
	*o = SyncHookJSONOutput(v.alias)
	o.HookSpecificOutput = hso
	return nil
}

func HookJSONOutputToProto(v HookJSONOutput) (*pb.HookJSONOutput, error) {
	switch x := v.value.(type) {
	case *HookJSONOutputUnknown:
		return &pb.HookJSONOutput{Value: &pb.HookJSONOutput_Unknown{Unknown: x.ToProto()}}, nil
	case *AsyncHookJSONOutput:
		return &pb.HookJSONOutput{
			Value: &pb.HookJSONOutput_AsyncOutput{AsyncOutput: &pb.AsyncHookJSONOutput{
				Async: x.Async, AsyncTimeout: x.AsyncTimeout,
			}},
		}, nil
	case *SyncHookJSONOutput:
		var hso *pb.HookSpecificOutput
		var err error
		if x.HookSpecificOutput.GetValue() != nil {
			hso, err = HookSpecificOutputToProto(x.HookSpecificOutput)
			if err != nil {
				return nil, err
			}
		}
		var decision *string
		if x.Decision != nil {
			s := string(*x.Decision)
			decision = &s
		}
		return &pb.HookJSONOutput{
			Value: &pb.HookJSONOutput_SyncOutput{SyncOutput: &pb.SyncHookJSONOutput{
				Continue:           x.Continue,
				SuppressOutput:     x.SuppressOutput,
				StopReason:         x.StopReason,
				Decision:           decision,
				SystemMessage:      x.SystemMessage,
				Reason:             x.Reason,
				HookSpecificOutput: hso,
				TerminalSequence:   x.TerminalSequence,
			}},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported HookJSONOutput %T", v)
	}
}

func HookJSONOutputFromProto(v *pb.HookJSONOutput) (HookJSONOutput, error) {
	if v == nil {
		return HookJSONOutput{}, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.HookJSONOutput_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return NewHookJSONOutput(&HookJSONOutputUnknown{UnknownUnion: u}), nil
	case *pb.HookJSONOutput_AsyncOutput:
		asyncOutput := v.GetAsyncOutput()
		return NewHookJSONOutput(&AsyncHookJSONOutput{
			Async:        asyncOutput.GetAsync(),
			AsyncTimeout: protoOpt(asyncOutput, "async_timeout", asyncOutput.GetAsyncTimeout()),
		}), nil
	case *pb.HookJSONOutput_SyncOutput:
		syncOutput := v.GetSyncOutput()
		var decision *HookDecision
		if got := syncOutput.GetDecision(); got != "" {
			d := HookDecision(got)
			decision = &d
		}
		hso, err := HookSpecificOutputFromProto(syncOutput.GetHookSpecificOutput())
		if err != nil {
			return HookJSONOutput{}, err
		}
		return NewHookJSONOutput(&SyncHookJSONOutput{
			Continue: protoOpt(syncOutput, "continue", syncOutput.GetContinue()),
			SuppressOutput: protoOpt(
				syncOutput,
				"suppress_output",
				syncOutput.GetSuppressOutput(),
			),
			StopReason: protoOpt(syncOutput, "stop_reason", syncOutput.GetStopReason()),
			Decision:   decision,
			SystemMessage: protoOpt(
				syncOutput,
				"system_message",
				syncOutput.GetSystemMessage(),
			),
			TerminalSequence: protoOpt(
				syncOutput,
				"terminal_sequence",
				syncOutput.GetTerminalSequence(),
			),
			Reason:             protoOpt(syncOutput, "reason", syncOutput.GetReason()),
			HookSpecificOutput: hso,
		}), nil
	default:
		return HookJSONOutput{}, fmt.Errorf("unsupported proto HookJSONOutput %T", x)
	}
}

func (o HookInputUnknown) MarshalJSON() ([]byte, error) { return o.UnknownUnion.MarshalJSON() }
func (o *HookInputUnknown) UnmarshalJSON(data []byte) error {
	return o.UnknownUnion.UnmarshalJSON(data)
}

func (o PreToolUseHookInput) MarshalJSON() ([]byte, error) {
	type alias PreToolUseHookInput
	o.HookEventName = HookEventPreToolUse
	return json.Marshal(alias(o))
}

func (o *PreToolUseHookInput) UnmarshalJSON(data []byte) error {
	type alias PreToolUseHookInput
	type raw struct {
		alias
		ToolInput json.RawMessage `json:"tool_input"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var toolInput ToolInputSchemas
	err := toolInput.UnmarshalForTool(v.ToolName, v.ToolInput)
	if err != nil {
		return err
	}
	*o = PreToolUseHookInput(v.alias)
	o.HookEventName = HookEventPreToolUse
	o.ToolInput = toolInput
	return nil
}

func (o PostToolUseHookInput) MarshalJSON() ([]byte, error) {
	type alias PostToolUseHookInput
	o.HookEventName = HookEventPostToolUse
	return json.Marshal(alias(o))
}

func (o *PostToolUseHookInput) UnmarshalJSON(data []byte) error {
	type alias PostToolUseHookInput
	type raw struct {
		alias
		ToolInput    json.RawMessage `json:"tool_input"`
		ToolResponse json.RawMessage `json:"tool_response"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var toolInput ToolInputSchemas
	err := toolInput.UnmarshalForTool(v.ToolName, v.ToolInput)
	if err != nil {
		return err
	}
	var toolResponse ToolOutputSchemas
	err = toolResponse.UnmarshalForTool(v.ToolName, v.ToolResponse)
	if err != nil {
		return err
	}
	*o = PostToolUseHookInput(v.alias)
	o.HookEventName = HookEventPostToolUse
	o.ToolInput = toolInput
	o.ToolResponse = toolResponse
	return nil
}

func (o PostToolUseFailureHookInput) MarshalJSON() ([]byte, error) {
	type alias PostToolUseFailureHookInput
	o.HookEventName = HookEventPostToolUseFailure
	return json.Marshal(alias(o))
}

func (o *PostToolUseFailureHookInput) UnmarshalJSON(data []byte) error {
	type alias PostToolUseFailureHookInput
	type raw struct {
		alias
		ToolInput json.RawMessage `json:"tool_input"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var toolInput ToolInputSchemas
	err := toolInput.UnmarshalForTool(v.ToolName, v.ToolInput)
	if err != nil {
		return err
	}
	*o = PostToolUseFailureHookInput(v.alias)
	o.HookEventName = HookEventPostToolUseFailure
	o.ToolInput = toolInput
	return nil
}

func (o PermissionRequestHookInput) MarshalJSON() ([]byte, error) {
	type alias PermissionRequestHookInput
	o.HookEventName = HookEventPermissionRequest
	// UnmarshalJSON always allocates the slice, so an empty one carries no
	// information; nil it out to keep omitzero dropping the key rather than
	// emitting [] for a payload that never had it.
	if len(o.PermissionSuggestions) == 0 {
		o.PermissionSuggestions = nil
	}
	return json.Marshal(alias(o))
}

func (o *PermissionRequestHookInput) UnmarshalJSON(data []byte) error {
	type alias PermissionRequestHookInput
	type raw struct {
		alias
		ToolInput             json.RawMessage   `json:"tool_input"`
		PermissionSuggestions []json.RawMessage `json:"permission_suggestions,omitzero"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var toolInput ToolInputSchemas
	err := toolInput.UnmarshalForTool(v.ToolName, v.ToolInput)
	if err != nil {
		return err
	}
	suggestions := make([]PermissionUpdate, 0, len(v.PermissionSuggestions))
	for _, rawSuggestion := range v.PermissionSuggestions {
		var suggestion PermissionUpdate
		err := suggestion.UnmarshalJSON(rawSuggestion)
		if err != nil {
			return err
		}
		suggestions = append(suggestions, suggestion)
	}
	*o = PermissionRequestHookInput(v.alias)
	o.HookEventName = HookEventPermissionRequest
	o.ToolInput = toolInput
	o.PermissionSuggestions = suggestions
	return nil
}

func (o PermissionDeniedHookInput) MarshalJSON() ([]byte, error) {
	type alias PermissionDeniedHookInput
	o.HookEventName = HookEventPermissionDenied
	return json.Marshal(alias(o))
}

func (o *PermissionDeniedHookInput) UnmarshalJSON(data []byte) error {
	type alias PermissionDeniedHookInput
	type raw struct {
		alias
		ToolInput json.RawMessage `json:"tool_input"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	var toolInput ToolInputSchemas
	err := toolInput.UnmarshalForTool(v.ToolName, v.ToolInput)
	if err != nil {
		return err
	}
	*o = PermissionDeniedHookInput(v.alias)
	o.HookEventName = HookEventPermissionDenied
	o.ToolInput = toolInput
	return nil
}

func HookInputToProto(v HookInput) (*pb.HookInput, error) {
	switch x := v.value.(type) {
	case *HookInputUnknown:
		return &pb.HookInput{Value: &pb.HookInput_Unknown{Unknown: x.ToProto()}}, nil
	case *PreToolUseHookInput:
		return &pb.HookInput{Value: &pb.HookInput_PreToolUse{PreToolUse: &pb.PreToolUseHookInput{
			SessionId:      x.SessionID,
			TranscriptPath: x.TranscriptPath,
			Cwd:            x.Cwd,
			PermissionMode: x.PermissionMode,
			AgentId:        x.AgentID,
			AgentType:      x.AgentType,
			HookEventName:  string(x.HookEventName),
			ToolName:       x.ToolName,
			ToolInput:      rawToolInputToProto(x.ToolInput, x.ToolName),
			ToolUseId:      x.ToolUseID,
			Effort:         baseEffortToProto(x.Effort),
		}}}, nil
	case *PostToolUseHookInput:
		return &pb.HookInput{Value: &pb.HookInput_PostToolUse{PostToolUse: &pb.PostToolUseHookInput{
			SessionId:      x.SessionID,
			TranscriptPath: x.TranscriptPath,
			Cwd:            x.Cwd,
			PermissionMode: x.PermissionMode,
			AgentId:        x.AgentID,
			AgentType:      x.AgentType,
			HookEventName:  string(x.HookEventName),
			ToolName:       x.ToolName,
			ToolInput:      rawToolInputToProto(x.ToolInput, x.ToolName),
			ToolResponse:   rawToolOutputToProto(x.ToolResponse, x.ToolName),
			ToolUseId:      x.ToolUseID,
			DurationMs:     x.DurationMs,
			Effort:         baseEffortToProto(x.Effort),
		}}}, nil
	case *PostToolBatchHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_PostToolBatch{PostToolBatch: &pb.PostToolBatchHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				ToolCalls:      postToolBatchToolCallsToProto(x.ToolCalls),
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *PostToolUseFailureHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_PostToolUseFailure{
				PostToolUseFailure: &pb.PostToolUseFailureHookInput{
					SessionId:      x.SessionID,
					TranscriptPath: x.TranscriptPath,
					Cwd:            x.Cwd,
					PermissionMode: x.PermissionMode,
					AgentId:        x.AgentID,
					AgentType:      x.AgentType,
					HookEventName:  string(x.HookEventName),
					ToolName:       x.ToolName,
					ToolInput:      rawToolInputToProto(x.ToolInput, x.ToolName),
					ToolUseId:      x.ToolUseID,
					Error:          x.Error,
					IsInterrupt:    x.IsInterrupt,
					DurationMs:     x.DurationMs,
					Effort:         baseEffortToProto(x.Effort),
				},
			},
		}, nil
	case *NotificationHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_Notification{Notification: &pb.NotificationHookInput{
				SessionId:        x.SessionID,
				TranscriptPath:   x.TranscriptPath,
				Cwd:              x.Cwd,
				PermissionMode:   x.PermissionMode,
				AgentId:          x.AgentID,
				AgentType:        x.AgentType,
				HookEventName:    string(x.HookEventName),
				Message:          x.Message,
				Title:            x.Title,
				NotificationType: x.NotificationType,
				Effort:           baseEffortToProto(x.Effort),
			}},
		}, nil
	case *UserPromptSubmitHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_UserPromptSubmit{UserPromptSubmit: &pb.UserPromptSubmitHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				Prompt:         x.Prompt,
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *SessionStartHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_SessionStart{SessionStart: &pb.SessionStartHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				Source:         string(x.Source),
				Model:          x.Model,
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *SessionEndHookInput:
		return &pb.HookInput{Value: &pb.HookInput_SessionEnd{SessionEnd: &pb.SessionEndHookInput{
			SessionId:      x.SessionID,
			TranscriptPath: x.TranscriptPath,
			Cwd:            x.Cwd,
			PermissionMode: x.PermissionMode,
			AgentId:        x.AgentID,
			AgentType:      x.AgentType,
			HookEventName:  string(x.HookEventName),
			Reason:         string(x.Reason),
			Effort:         baseEffortToProto(x.Effort),
		}}}, nil
	case *StopHookInput:
		return &pb.HookInput{Value: &pb.HookInput_Stop{Stop: &pb.StopHookInput{
			SessionId:            x.SessionID,
			TranscriptPath:       x.TranscriptPath,
			Cwd:                  x.Cwd,
			PermissionMode:       x.PermissionMode,
			AgentId:              x.AgentID,
			AgentType:            x.AgentType,
			HookEventName:        string(x.HookEventName),
			StopHookActive:       x.StopHookActive,
			LastAssistantMessage: x.LastAssistantMessage,
			BackgroundTasks:      backgroundTasksToProto(x.BackgroundTasks),
			SessionCrons:         sessionCronsToProto(x.SessionCrons),
			Effort:               baseEffortToProto(x.Effort),
		}}}, nil
	case *SubagentStartHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_SubagentStart{SubagentStart: &pb.SubagentStartHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        new(x.AgentID),
				AgentType:      new(x.AgentType),
				HookEventName:  string(x.HookEventName),
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *SubagentStopHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_SubagentStop{SubagentStop: &pb.SubagentStopHookInput{
				SessionId:            x.SessionID,
				TranscriptPath:       x.TranscriptPath,
				Cwd:                  x.Cwd,
				PermissionMode:       x.PermissionMode,
				AgentId:              new(x.AgentID),
				AgentType:            new(x.AgentType),
				HookEventName:        string(x.HookEventName),
				StopHookActive:       x.StopHookActive,
				AgentTranscriptPath:  x.AgentTranscriptPath,
				LastAssistantMessage: x.LastAssistantMessage,
				BackgroundTasks:      backgroundTasksToProto(x.BackgroundTasks),
				SessionCrons:         sessionCronsToProto(x.SessionCrons),
				Effort:               baseEffortToProto(x.Effort),
			}},
		}, nil
	case *PreCompactHookInput:
		return &pb.HookInput{Value: &pb.HookInput_PreCompact{PreCompact: &pb.PreCompactHookInput{
			SessionId:          x.SessionID,
			TranscriptPath:     x.TranscriptPath,
			Cwd:                x.Cwd,
			PermissionMode:     x.PermissionMode,
			AgentId:            x.AgentID,
			AgentType:          x.AgentType,
			HookEventName:      string(x.HookEventName),
			Trigger:            string(x.Trigger),
			CustomInstructions: x.CustomInstructions,
			Effort:             baseEffortToProto(x.Effort),
		}}}, nil
	case *PermissionRequestHookInput:
		updates, err := permissionUpdatesToProto(x.PermissionSuggestions)
		if err != nil {
			return nil, err
		}
		return &pb.HookInput{
			Value: &pb.HookInput_PermissionRequest{
				PermissionRequest: &pb.PermissionRequestHookInput{
					SessionId:      x.SessionID,
					TranscriptPath: x.TranscriptPath,
					Cwd:            x.Cwd,
					PermissionMode: x.PermissionMode,
					AgentId:        x.AgentID,
					AgentType:      x.AgentType,
					HookEventName:  string(x.HookEventName),
					ToolName:       x.ToolName,
					ToolInput: rawToolInputToProto(
						x.ToolInput,
						x.ToolName,
					),
					PermissionSuggestions: updates,
					Effort:                baseEffortToProto(x.Effort),
				},
			},
		}, nil
	case *PermissionDeniedHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_PermissionDenied{
				PermissionDenied: &pb.PermissionDeniedHookInput{
					SessionId:      x.SessionID,
					TranscriptPath: x.TranscriptPath,
					Cwd:            x.Cwd,
					PermissionMode: x.PermissionMode,
					AgentId:        x.AgentID,
					AgentType:      x.AgentType,
					HookEventName:  string(x.HookEventName),
					ToolName:       x.ToolName,
					ToolInput:      rawToolInputToProto(x.ToolInput, x.ToolName),
					ToolUseId:      x.ToolUseID,
					Reason:         x.Reason,
					Effort:         baseEffortToProto(x.Effort),
				},
			},
		}, nil
	case *SetupHookInput:
		return &pb.HookInput{Value: &pb.HookInput_Setup{Setup: &pb.SetupHookInput{
			SessionId:      x.SessionID,
			TranscriptPath: x.TranscriptPath,
			Cwd:            x.Cwd,
			PermissionMode: x.PermissionMode,
			AgentId:        x.AgentID,
			AgentType:      x.AgentType,
			HookEventName:  string(x.HookEventName),
			Trigger:        string(x.Trigger),
			Effort:         baseEffortToProto(x.Effort),
		}}}, nil
	case *TeammateIdleHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_TeammateIdle{TeammateIdle: &pb.TeammateIdleHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				TeammateName:   x.TeammateName,
				TeamName:       x.TeamName,
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *TaskCompletedHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_TaskCompleted{TaskCompleted: &pb.TaskCompletedHookInput{
				SessionId:       x.SessionID,
				TranscriptPath:  x.TranscriptPath,
				Cwd:             x.Cwd,
				PermissionMode:  x.PermissionMode,
				AgentId:         x.AgentID,
				AgentType:       x.AgentType,
				HookEventName:   string(x.HookEventName),
				TaskId:          x.TaskID,
				TaskSubject:     x.TaskSubject,
				TaskDescription: x.TaskDescription,
				TeammateName:    x.TeammateName,
				TeamName:        x.TeamName,
				Effort:          baseEffortToProto(x.Effort),
			}},
		}, nil
	case *ConfigChangeHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_ConfigChange{ConfigChange: &pb.ConfigChangeHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				Source:         string(x.Source),
				FilePath:       x.FilePath,
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *WorktreeCreateHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_WorktreeCreate{WorktreeCreate: &pb.WorktreeCreateHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				Name:           x.Name,
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *WorktreeRemoveHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_WorktreeRemove{WorktreeRemove: &pb.WorktreeRemoveHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				WorktreePath:   x.WorktreePath,
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	case *MessageDisplayHookInput:
		return &pb.HookInput{
			Value: &pb.HookInput_MessageDisplay{MessageDisplay: &pb.MessageDisplayHookInput{
				SessionId:      x.SessionID,
				TranscriptPath: x.TranscriptPath,
				Cwd:            x.Cwd,
				PermissionMode: x.PermissionMode,
				AgentId:        x.AgentID,
				AgentType:      x.AgentType,
				HookEventName:  string(x.HookEventName),
				TurnId:         x.TurnID,
				MessageId:      x.MessageID,
				Index:          x.Index,
				Final:          x.Final,
				Delta:          x.Delta,
				Effort:         baseEffortToProto(x.Effort),
			}},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported HookInput %T", v)
	}
}

func HookInputFromProto(v *pb.HookInput) (HookInput, error) {
	if v == nil {
		return HookInput{}, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.HookInput_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return NewHookInput(&HookInputUnknown{UnknownUnion: u}), nil
	case *pb.HookInput_PreToolUse:
		preToolUse := v.GetPreToolUse()
		raw, err := rawFromToolInputProto(preToolUse.GetToolInput(), preToolUse.GetToolName())
		if err != nil {
			return HookInput{}, err
		}
		return NewHookInput(&PreToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      preToolUse.GetSessionId(),
				TranscriptPath: preToolUse.GetTranscriptPath(),
				Cwd:            preToolUse.GetCwd(),
				PermissionMode: protoOpt(
					preToolUse,
					"permission_mode",
					preToolUse.GetPermissionMode(),
				),
				AgentID:   protoOpt(preToolUse, "agent_id", preToolUse.GetAgentId()),
				AgentType: protoOpt(preToolUse, "agent_type", preToolUse.GetAgentType()),
				Effort:    baseEffortFromProto(preToolUse.GetEffort()),
			},
			HookEventName: HookEvent(preToolUse.GetHookEventName()),
			ToolName:      preToolUse.GetToolName(),
			ToolInput:     raw,
			ToolUseID:     preToolUse.GetToolUseId(),
		}), nil
	case *pb.HookInput_PostToolUse:
		postToolUse := v.GetPostToolUse()
		rawIn, err := rawFromToolInputProto(postToolUse.GetToolInput(), postToolUse.GetToolName())
		if err != nil {
			return HookInput{}, err
		}
		rawOut, err := rawFromToolOutputProto(
			postToolUse.GetToolResponse(),
			postToolUse.GetToolName(),
		)
		if err != nil {
			return HookInput{}, err
		}
		return NewHookInput(&PostToolUseHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      postToolUse.GetSessionId(),
				TranscriptPath: postToolUse.GetTranscriptPath(),
				Cwd:            postToolUse.GetCwd(),
				PermissionMode: protoOpt(
					postToolUse,
					"permission_mode",
					postToolUse.GetPermissionMode(),
				),
				AgentID:   protoOpt(postToolUse, "agent_id", postToolUse.GetAgentId()),
				AgentType: protoOpt(postToolUse, "agent_type", postToolUse.GetAgentType()),
				Effort:    baseEffortFromProto(postToolUse.GetEffort()),
			},
			HookEventName: HookEvent(postToolUse.GetHookEventName()),
			ToolName:      postToolUse.GetToolName(),
			ToolInput:     rawIn,
			ToolResponse:  rawOut,
			ToolUseID:     postToolUse.GetToolUseId(),
			DurationMs:    protoOpt(postToolUse, "duration_ms", postToolUse.GetDurationMs()),
		}), nil
	case *pb.HookInput_PostToolUseFailure:
		postToolUseFailure := v.GetPostToolUseFailure()
		raw, err := rawFromToolInputProto(
			postToolUseFailure.GetToolInput(),
			postToolUseFailure.GetToolName(),
		)
		if err != nil {
			return HookInput{}, err
		}
		return NewHookInput(&PostToolUseFailureHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      postToolUseFailure.GetSessionId(),
				TranscriptPath: postToolUseFailure.GetTranscriptPath(),
				Cwd:            postToolUseFailure.GetCwd(),
				PermissionMode: protoOpt(
					postToolUseFailure,
					"permission_mode",
					postToolUseFailure.GetPermissionMode(),
				),
				AgentID: protoOpt(
					postToolUseFailure,
					"agent_id",
					postToolUseFailure.GetAgentId(),
				),
				AgentType: protoOpt(
					postToolUseFailure,
					"agent_type",
					postToolUseFailure.GetAgentType(),
				),
				Effort: baseEffortFromProto(postToolUseFailure.GetEffort()),
			},
			HookEventName: HookEvent(postToolUseFailure.GetHookEventName()),
			ToolName:      postToolUseFailure.GetToolName(),
			ToolInput:     raw,
			ToolUseID:     postToolUseFailure.GetToolUseId(),
			Error:         postToolUseFailure.GetError(),
			IsInterrupt: protoOpt(
				postToolUseFailure,
				"is_interrupt",
				postToolUseFailure.GetIsInterrupt(),
			),
			DurationMs: protoOpt(
				postToolUseFailure,
				"duration_ms",
				postToolUseFailure.GetDurationMs(),
			),
		}), nil
	case *pb.HookInput_Notification:
		notification := v.GetNotification()
		return NewHookInput(&NotificationHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      notification.GetSessionId(),
				TranscriptPath: notification.GetTranscriptPath(),
				Cwd:            notification.GetCwd(),
				PermissionMode: protoOpt(
					notification,
					"permission_mode",
					notification.GetPermissionMode(),
				),
				AgentID:   protoOpt(notification, "agent_id", notification.GetAgentId()),
				AgentType: protoOpt(notification, "agent_type", notification.GetAgentType()),
				Effort:    baseEffortFromProto(notification.GetEffort()),
			},
			HookEventName:    HookEvent(notification.GetHookEventName()),
			Message:          notification.GetMessage(),
			Title:            protoOpt(notification, "title", notification.GetTitle()),
			NotificationType: notification.GetNotificationType(),
		}), nil
	case *pb.HookInput_UserPromptSubmit:
		userPromptSubmit := v.GetUserPromptSubmit()
		return NewHookInput(&UserPromptSubmitHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      userPromptSubmit.GetSessionId(),
				TranscriptPath: userPromptSubmit.GetTranscriptPath(),
				Cwd:            userPromptSubmit.GetCwd(),
				PermissionMode: protoOpt(
					userPromptSubmit,
					"permission_mode",
					userPromptSubmit.GetPermissionMode(),
				),
				AgentID: protoOpt(
					userPromptSubmit,
					"agent_id",
					userPromptSubmit.GetAgentId(),
				),
				AgentType: protoOpt(
					userPromptSubmit,
					"agent_type",
					userPromptSubmit.GetAgentType(),
				),
				Effort: baseEffortFromProto(userPromptSubmit.GetEffort()),
			},
			HookEventName: HookEvent(userPromptSubmit.GetHookEventName()),
			Prompt:        userPromptSubmit.GetPrompt(),
		}), nil
	case *pb.HookInput_SessionStart:
		sessionStart := v.GetSessionStart()
		return NewHookInput(&SessionStartHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      sessionStart.GetSessionId(),
				TranscriptPath: sessionStart.GetTranscriptPath(),
				Cwd:            sessionStart.GetCwd(),
				PermissionMode: protoOpt(
					sessionStart,
					"permission_mode",
					sessionStart.GetPermissionMode(),
				),
				AgentID:   protoOpt(sessionStart, "agent_id", sessionStart.GetAgentId()),
				AgentType: protoOpt(sessionStart, "agent_type", sessionStart.GetAgentType()),
				Effort:    baseEffortFromProto(sessionStart.GetEffort()),
			},
			HookEventName: HookEvent(sessionStart.GetHookEventName()),
			Source:        SessionStartSource(sessionStart.GetSource()),
			Model:         protoOpt(sessionStart, "model", sessionStart.GetModel()),
		}), nil
	case *pb.HookInput_SessionEnd:
		sessionEnd := v.GetSessionEnd()
		return NewHookInput(&SessionEndHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      sessionEnd.GetSessionId(),
				TranscriptPath: sessionEnd.GetTranscriptPath(),
				Cwd:            sessionEnd.GetCwd(),
				PermissionMode: protoOpt(
					sessionEnd,
					"permission_mode",
					sessionEnd.GetPermissionMode(),
				),
				AgentID:   protoOpt(sessionEnd, "agent_id", sessionEnd.GetAgentId()),
				AgentType: protoOpt(sessionEnd, "agent_type", sessionEnd.GetAgentType()),
				Effort:    baseEffortFromProto(sessionEnd.GetEffort()),
			},
			HookEventName: HookEvent(sessionEnd.GetHookEventName()),
			Reason:        ExitReason(sessionEnd.GetReason()),
		}), nil
	case *pb.HookInput_Stop:
		stop := v.GetStop()
		return NewHookInput(&StopHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      stop.GetSessionId(),
				TranscriptPath: stop.GetTranscriptPath(),
				Cwd:            stop.GetCwd(),
				PermissionMode: protoOpt(stop, "permission_mode", stop.GetPermissionMode()),
				AgentID:        protoOpt(stop, "agent_id", stop.GetAgentId()),
				AgentType:      protoOpt(stop, "agent_type", stop.GetAgentType()),
				Effort:         baseEffortFromProto(stop.GetEffort()),
			},
			HookEventName:  HookEvent(stop.GetHookEventName()),
			StopHookActive: stop.GetStopHookActive(),
			LastAssistantMessage: protoOpt(
				stop,
				"last_assistant_message",
				stop.GetLastAssistantMessage(),
			),
			BackgroundTasks: backgroundTasksFromProto(stop.GetBackgroundTasks()),
			SessionCrons:    sessionCronsFromProto(stop.GetSessionCrons()),
		}), nil
	case *pb.HookInput_SubagentStart:
		subagentStart := v.GetSubagentStart()
		return NewHookInput(&SubagentStartHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      subagentStart.GetSessionId(),
				TranscriptPath: subagentStart.GetTranscriptPath(),
				Cwd:            subagentStart.GetCwd(),
				PermissionMode: protoOpt(
					subagentStart,
					"permission_mode",
					subagentStart.GetPermissionMode(),
				),
				Effort: baseEffortFromProto(subagentStart.GetEffort()),
			},
			HookEventName: HookEvent(subagentStart.GetHookEventName()),
			AgentID:       subagentStart.GetAgentId(),
			AgentType:     subagentStart.GetAgentType(),
		}), nil
	case *pb.HookInput_SubagentStop:
		subagentStop := v.GetSubagentStop()
		return NewHookInput(&SubagentStopHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      subagentStop.GetSessionId(),
				TranscriptPath: subagentStop.GetTranscriptPath(),
				Cwd:            subagentStop.GetCwd(),
				PermissionMode: protoOpt(
					subagentStop,
					"permission_mode",
					subagentStop.GetPermissionMode(),
				),
				Effort: baseEffortFromProto(subagentStop.GetEffort()),
			},
			HookEventName:       HookEvent(subagentStop.GetHookEventName()),
			StopHookActive:      subagentStop.GetStopHookActive(),
			AgentID:             subagentStop.GetAgentId(),
			AgentTranscriptPath: subagentStop.GetAgentTranscriptPath(),
			AgentType:           subagentStop.GetAgentType(),
			LastAssistantMessage: protoOpt(
				subagentStop,
				"last_assistant_message",
				subagentStop.GetLastAssistantMessage(),
			),
			BackgroundTasks: backgroundTasksFromProto(subagentStop.GetBackgroundTasks()),
			SessionCrons:    sessionCronsFromProto(subagentStop.GetSessionCrons()),
		}), nil
	case *pb.HookInput_PreCompact:
		preCompact := v.GetPreCompact()
		return NewHookInput(&PreCompactHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      preCompact.GetSessionId(),
				TranscriptPath: preCompact.GetTranscriptPath(),
				Cwd:            preCompact.GetCwd(),
				PermissionMode: protoOpt(
					preCompact,
					"permission_mode",
					preCompact.GetPermissionMode(),
				),
				AgentID:   protoOpt(preCompact, "agent_id", preCompact.GetAgentId()),
				AgentType: protoOpt(preCompact, "agent_type", preCompact.GetAgentType()),
				Effort:    baseEffortFromProto(preCompact.GetEffort()),
			},
			HookEventName: HookEvent(preCompact.GetHookEventName()),
			Trigger:       PreCompactTrigger(preCompact.GetTrigger()),
			CustomInstructions: protoOpt(
				preCompact,
				"custom_instructions",
				preCompact.GetCustomInstructions(),
			),
		}), nil
	case *pb.HookInput_PermissionRequest:
		permissionRequest := v.GetPermissionRequest()
		raw, err := rawFromToolInputProto(
			permissionRequest.GetToolInput(),
			permissionRequest.GetToolName(),
		)
		if err != nil {
			return HookInput{}, err
		}
		updates, err := permissionUpdatesFromProto(permissionRequest.GetPermissionSuggestions())
		if err != nil {
			return HookInput{}, err
		}
		return NewHookInput(&PermissionRequestHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      permissionRequest.GetSessionId(),
				TranscriptPath: permissionRequest.GetTranscriptPath(),
				Cwd:            permissionRequest.GetCwd(),
				PermissionMode: protoOpt(
					permissionRequest,
					"permission_mode",
					permissionRequest.GetPermissionMode(),
				),
				AgentID: protoOpt(
					permissionRequest,
					"agent_id",
					permissionRequest.GetAgentId(),
				),
				AgentType: protoOpt(
					permissionRequest,
					"agent_type",
					permissionRequest.GetAgentType(),
				),
				Effort: baseEffortFromProto(permissionRequest.GetEffort()),
			},
			HookEventName:         HookEvent(permissionRequest.GetHookEventName()),
			ToolName:              permissionRequest.GetToolName(),
			ToolInput:             raw,
			PermissionSuggestions: updates,
		}), nil
	case *pb.HookInput_PermissionDenied:
		permissionDenied := v.GetPermissionDenied()
		raw, err := rawFromToolInputProto(
			permissionDenied.GetToolInput(),
			permissionDenied.GetToolName(),
		)
		if err != nil {
			return HookInput{}, err
		}
		return NewHookInput(&PermissionDeniedHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      permissionDenied.GetSessionId(),
				TranscriptPath: permissionDenied.GetTranscriptPath(),
				Cwd:            permissionDenied.GetCwd(),
				PermissionMode: protoOpt(
					permissionDenied,
					"permission_mode",
					permissionDenied.GetPermissionMode(),
				),
				AgentID: protoOpt(
					permissionDenied,
					"agent_id",
					permissionDenied.GetAgentId(),
				),
				AgentType: protoOpt(
					permissionDenied,
					"agent_type",
					permissionDenied.GetAgentType(),
				),
				Effort: baseEffortFromProto(permissionDenied.GetEffort()),
			},
			HookEventName: HookEvent(permissionDenied.GetHookEventName()),
			ToolName:      permissionDenied.GetToolName(),
			ToolInput:     raw,
			ToolUseID:     permissionDenied.GetToolUseId(),
			Reason:        permissionDenied.GetReason(),
		}), nil
	case *pb.HookInput_Setup:
		setup := v.GetSetup()
		return NewHookInput(&SetupHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      setup.GetSessionId(),
				TranscriptPath: setup.GetTranscriptPath(),
				Cwd:            setup.GetCwd(),
				PermissionMode: protoOpt(setup, "permission_mode", setup.GetPermissionMode()),
				AgentID:        protoOpt(setup, "agent_id", setup.GetAgentId()),
				AgentType:      protoOpt(setup, "agent_type", setup.GetAgentType()),
				Effort:         baseEffortFromProto(setup.GetEffort()),
			},
			HookEventName: HookEvent(setup.GetHookEventName()),
			Trigger:       SetupTrigger(setup.GetTrigger()),
		}), nil
	case *pb.HookInput_TeammateIdle:
		teammateIdle := v.GetTeammateIdle()
		return NewHookInput(&TeammateIdleHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      teammateIdle.GetSessionId(),
				TranscriptPath: teammateIdle.GetTranscriptPath(),
				Cwd:            teammateIdle.GetCwd(),
				PermissionMode: protoOpt(
					teammateIdle,
					"permission_mode",
					teammateIdle.GetPermissionMode(),
				),
				AgentID:   protoOpt(teammateIdle, "agent_id", teammateIdle.GetAgentId()),
				AgentType: protoOpt(teammateIdle, "agent_type", teammateIdle.GetAgentType()),
				Effort:    baseEffortFromProto(teammateIdle.GetEffort()),
			},
			HookEventName: HookEvent(teammateIdle.GetHookEventName()),
			TeammateName:  teammateIdle.GetTeammateName(),
			TeamName:      teammateIdle.GetTeamName(),
		}), nil
	case *pb.HookInput_TaskCompleted:
		taskCompleted := v.GetTaskCompleted()
		return NewHookInput(&TaskCompletedHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      taskCompleted.GetSessionId(),
				TranscriptPath: taskCompleted.GetTranscriptPath(),
				Cwd:            taskCompleted.GetCwd(),
				PermissionMode: protoOpt(
					taskCompleted,
					"permission_mode",
					taskCompleted.GetPermissionMode(),
				),
				AgentID:   protoOpt(taskCompleted, "agent_id", taskCompleted.GetAgentId()),
				AgentType: protoOpt(taskCompleted, "agent_type", taskCompleted.GetAgentType()),
				Effort:    baseEffortFromProto(taskCompleted.GetEffort()),
			},
			HookEventName: HookEvent(taskCompleted.GetHookEventName()),
			TaskID:        taskCompleted.GetTaskId(),
			TaskSubject:   taskCompleted.GetTaskSubject(),
			TaskDescription: protoOpt(
				taskCompleted,
				"task_description",
				taskCompleted.GetTaskDescription(),
			),
			TeammateName: protoOpt(
				taskCompleted,
				"teammate_name",
				taskCompleted.GetTeammateName(),
			),
			TeamName: protoOpt(taskCompleted, "team_name", taskCompleted.GetTeamName()),
		}), nil
	case *pb.HookInput_ConfigChange:
		configChange := v.GetConfigChange()
		return NewHookInput(&ConfigChangeHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      configChange.GetSessionId(),
				TranscriptPath: configChange.GetTranscriptPath(),
				Cwd:            configChange.GetCwd(),
				PermissionMode: protoOpt(
					configChange,
					"permission_mode",
					configChange.GetPermissionMode(),
				),
				AgentID:   protoOpt(configChange, "agent_id", configChange.GetAgentId()),
				AgentType: protoOpt(configChange, "agent_type", configChange.GetAgentType()),
				Effort:    baseEffortFromProto(configChange.GetEffort()),
			},
			HookEventName: HookEvent(configChange.GetHookEventName()),
			Source:        ConfigChangeSource(configChange.GetSource()),
			FilePath:      protoOpt(configChange, "file_path", configChange.GetFilePath()),
		}), nil
	case *pb.HookInput_WorktreeCreate:
		worktreeCreate := v.GetWorktreeCreate()
		return NewHookInput(&WorktreeCreateHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      worktreeCreate.GetSessionId(),
				TranscriptPath: worktreeCreate.GetTranscriptPath(),
				Cwd:            worktreeCreate.GetCwd(),
				PermissionMode: protoOpt(
					worktreeCreate,
					"permission_mode",
					worktreeCreate.GetPermissionMode(),
				),
				AgentID: protoOpt(worktreeCreate, "agent_id", worktreeCreate.GetAgentId()),
				AgentType: protoOpt(
					worktreeCreate,
					"agent_type",
					worktreeCreate.GetAgentType(),
				),
				Effort: baseEffortFromProto(worktreeCreate.GetEffort()),
			},
			HookEventName: HookEvent(worktreeCreate.GetHookEventName()),
			Name:          worktreeCreate.GetName(),
		}), nil
	case *pb.HookInput_WorktreeRemove:
		worktreeRemove := v.GetWorktreeRemove()
		return NewHookInput(&WorktreeRemoveHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      worktreeRemove.GetSessionId(),
				TranscriptPath: worktreeRemove.GetTranscriptPath(),
				Cwd:            worktreeRemove.GetCwd(),
				PermissionMode: protoOpt(
					worktreeRemove,
					"permission_mode",
					worktreeRemove.GetPermissionMode(),
				),
				AgentID: protoOpt(worktreeRemove, "agent_id", worktreeRemove.GetAgentId()),
				AgentType: protoOpt(
					worktreeRemove,
					"agent_type",
					worktreeRemove.GetAgentType(),
				),
				Effort: baseEffortFromProto(worktreeRemove.GetEffort()),
			},
			HookEventName: HookEvent(worktreeRemove.GetHookEventName()),
			WorktreePath:  worktreeRemove.GetWorktreePath(),
		}), nil
	case *pb.HookInput_PostToolBatch:
		postToolBatch := v.GetPostToolBatch()
		return NewHookInput(&PostToolBatchHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      postToolBatch.GetSessionId(),
				TranscriptPath: postToolBatch.GetTranscriptPath(),
				Cwd:            postToolBatch.GetCwd(),
				PermissionMode: protoOpt(
					postToolBatch,
					"permission_mode",
					postToolBatch.GetPermissionMode(),
				),
				AgentID:   protoOpt(postToolBatch, "agent_id", postToolBatch.GetAgentId()),
				AgentType: protoOpt(postToolBatch, "agent_type", postToolBatch.GetAgentType()),
				Effort:    baseEffortFromProto(postToolBatch.GetEffort()),
			},
			HookEventName: HookEvent(postToolBatch.GetHookEventName()),
			ToolCalls:     postToolBatchToolCallsFromProto(postToolBatch.GetToolCalls()),
		}), nil
	case *pb.HookInput_MessageDisplay:
		messageDisplay := v.GetMessageDisplay()
		return NewHookInput(&MessageDisplayHookInput{
			BaseHookInput: BaseHookInput{
				SessionID:      messageDisplay.GetSessionId(),
				TranscriptPath: messageDisplay.GetTranscriptPath(),
				Cwd:            messageDisplay.GetCwd(),
				PermissionMode: protoOpt(
					messageDisplay,
					"permission_mode",
					messageDisplay.GetPermissionMode(),
				),
				AgentID: protoOpt(messageDisplay, "agent_id", messageDisplay.GetAgentId()),
				AgentType: protoOpt(
					messageDisplay,
					"agent_type",
					messageDisplay.GetAgentType(),
				),
				Effort: baseEffortFromProto(messageDisplay.GetEffort()),
			},
			HookEventName: HookEvent(messageDisplay.GetHookEventName()),
			TurnID:        messageDisplay.GetTurnId(),
			MessageID:     messageDisplay.GetMessageId(),
			Index:         messageDisplay.GetIndex(),
			Final:         messageDisplay.GetFinal(),
			Delta:         messageDisplay.GetDelta(),
		}), nil
	default:
		return HookInput{}, fmt.Errorf("unsupported proto HookInput %T", x)
	}
}

// ---- generated union struct accessors ----

// NewSystemPrompt wraps a [SystemPrompt_Value] variant into a [SystemPrompt].
func NewSystemPrompt(v SystemPrompt_Value) SystemPrompt { return SystemPrompt{value: v} }

// GetValue returns the active [SystemPrompt_Value] variant, or nil when unset.
func (o SystemPrompt) GetValue() SystemPrompt_Value { return o.value }

// GetPreset reports whether the active variant is [*SystemPromptPreset] and returns it.
func (o SystemPrompt) GetPreset() (*SystemPromptPreset, bool) {
	v, ok := o.value.(*SystemPromptPreset)
	return v, ok
}

// GetString reports whether the active variant is [*SystemPromptString] and returns it.
func (o SystemPrompt) GetString() (*SystemPromptString, bool) {
	v, ok := o.value.(*SystemPromptString)
	return v, ok
}

// GetUnknown reports whether the active variant is [*SystemPromptUnknown] and returns it.
func (o SystemPrompt) GetUnknown() (*SystemPromptUnknown, bool) {
	v, ok := o.value.(*SystemPromptUnknown)
	return v, ok
}

// NewThinkingConfig wraps a [ThinkingConfig_Value] variant into a [ThinkingConfig].
func NewThinkingConfig(v ThinkingConfig_Value) ThinkingConfig { return ThinkingConfig{value: v} }

// GetValue returns the active [ThinkingConfig_Value] variant, or nil when unset.
func (o ThinkingConfig) GetValue() ThinkingConfig_Value { return o.value }

// GetAdaptive reports whether the active variant is [*ThinkingConfigAdaptive] and returns it.
func (o ThinkingConfig) GetAdaptive() (*ThinkingConfigAdaptive, bool) {
	v, ok := o.value.(*ThinkingConfigAdaptive)
	return v, ok
}

// GetDisabled reports whether the active variant is [*ThinkingConfigDisabled] and returns it.
func (o ThinkingConfig) GetDisabled() (*ThinkingConfigDisabled, bool) {
	v, ok := o.value.(*ThinkingConfigDisabled)
	return v, ok
}

// GetEnabled reports whether the active variant is [*ThinkingConfigEnabled] and returns it.
func (o ThinkingConfig) GetEnabled() (*ThinkingConfigEnabled, bool) {
	v, ok := o.value.(*ThinkingConfigEnabled)
	return v, ok
}

// GetUnknown reports whether the active variant is [*ThinkingConfigUnknown] and returns it.
func (o ThinkingConfig) GetUnknown() (*ThinkingConfigUnknown, bool) {
	v, ok := o.value.(*ThinkingConfigUnknown)
	return v, ok
}

// NewToolsConfig wraps a [ToolsConfig_Value] variant into a [ToolsConfig].
func NewToolsConfig(v ToolsConfig_Value) ToolsConfig { return ToolsConfig{value: v} }

// GetValue returns the active [ToolsConfig_Value] variant, or nil when unset.
func (o ToolsConfig) GetValue() ToolsConfig_Value { return o.value }

// GetList reports whether the active variant is [*ToolsConfigList] and returns it.
func (o ToolsConfig) GetList() (*ToolsConfigList, bool) {
	v, ok := o.value.(*ToolsConfigList)
	return v, ok
}

// GetPreset reports whether the active variant is [*ToolsConfigPreset] and returns it.
func (o ToolsConfig) GetPreset() (*ToolsConfigPreset, bool) {
	v, ok := o.value.(*ToolsConfigPreset)
	return v, ok
}

// GetUnknown reports whether the active variant is [*ToolsConfigUnknown] and returns it.
func (o ToolsConfig) GetUnknown() (*ToolsConfigUnknown, bool) {
	v, ok := o.value.(*ToolsConfigUnknown)
	return v, ok
}

// NewPermissionUpdate wraps a [PermissionUpdate_Value] variant into a [PermissionUpdate].
func NewPermissionUpdate(
	v PermissionUpdate_Value,
) PermissionUpdate {
	return PermissionUpdate{value: v}
}

// GetValue returns the active [PermissionUpdate_Value] variant, or nil when unset.
func (o PermissionUpdate) GetValue() PermissionUpdate_Value { return o.value }

// GetAddDirectories reports whether the active variant is [*PermissionUpdateAddDirectories] and
// returns it.
func (o PermissionUpdate) GetAddDirectories() (*PermissionUpdateAddDirectories, bool) {
	v, ok := o.value.(*PermissionUpdateAddDirectories)
	return v, ok
}

// GetAddRules reports whether the active variant is [*PermissionUpdateAddRules] and returns it.
func (o PermissionUpdate) GetAddRules() (*PermissionUpdateAddRules, bool) {
	v, ok := o.value.(*PermissionUpdateAddRules)
	return v, ok
}

// GetRemoveDirectories reports whether the active variant is [*PermissionUpdateRemoveDirectories]
// and returns it.
func (o PermissionUpdate) GetRemoveDirectories() (*PermissionUpdateRemoveDirectories, bool) {
	v, ok := o.value.(*PermissionUpdateRemoveDirectories)
	return v, ok
}

// GetRemoveRules reports whether the active variant is [*PermissionUpdateRemoveRules] and returns
// it.
func (o PermissionUpdate) GetRemoveRules() (*PermissionUpdateRemoveRules, bool) {
	v, ok := o.value.(*PermissionUpdateRemoveRules)
	return v, ok
}

// GetReplaceRules reports whether the active variant is [*PermissionUpdateReplaceRules] and returns
// it.
func (o PermissionUpdate) GetReplaceRules() (*PermissionUpdateReplaceRules, bool) {
	v, ok := o.value.(*PermissionUpdateReplaceRules)
	return v, ok
}

// GetSetMode reports whether the active variant is [*PermissionUpdateSetMode] and returns it.
func (o PermissionUpdate) GetSetMode() (*PermissionUpdateSetMode, bool) {
	v, ok := o.value.(*PermissionUpdateSetMode)
	return v, ok
}

// GetUnknown reports whether the active variant is [*PermissionUpdateUnknown] and returns it.
func (o PermissionUpdate) GetUnknown() (*PermissionUpdateUnknown, bool) {
	v, ok := o.value.(*PermissionUpdateUnknown)
	return v, ok
}

// NewPermissionResult wraps a [PermissionResult_Value] variant into a [PermissionResult].
func NewPermissionResult(
	v PermissionResult_Value,
) PermissionResult {
	return PermissionResult{value: v}
}

// GetValue returns the active [PermissionResult_Value] variant, or nil when unset.
func (o PermissionResult) GetValue() PermissionResult_Value { return o.value }

// GetAllow reports whether the active variant is [*PermissionResultAllow] and returns it.
func (o PermissionResult) GetAllow() (*PermissionResultAllow, bool) {
	v, ok := o.value.(*PermissionResultAllow)
	return v, ok
}

// GetDeny reports whether the active variant is [*PermissionResultDeny] and returns it.
func (o PermissionResult) GetDeny() (*PermissionResultDeny, bool) {
	v, ok := o.value.(*PermissionResultDeny)
	return v, ok
}

// GetUnknown reports whether the active variant is [*PermissionResultUnknown] and returns it.
func (o PermissionResult) GetUnknown() (*PermissionResultUnknown, bool) {
	v, ok := o.value.(*PermissionResultUnknown)
	return v, ok
}

// NewMcpServerConfig wraps a [McpServerConfig_Value] variant into a [McpServerConfig].
func NewMcpServerConfig(
	v McpServerConfig_Value,
) McpServerConfig {
	return McpServerConfig{value: v}
}

// GetValue returns the active [McpServerConfig_Value] variant, or nil when unset.
func (o McpServerConfig) GetValue() McpServerConfig_Value { return o.value }

// GetMcpHttpServerConfig reports whether the active variant is [*McpHttpServerConfig] and returns
// it.
func (o McpServerConfig) GetMcpHttpServerConfig() (*McpHttpServerConfig, bool) {
	v, ok := o.value.(*McpHttpServerConfig)
	return v, ok
}

// GetMcpSSEServerConfig reports whether the active variant is [*McpSSEServerConfig] and returns it.
func (o McpServerConfig) GetMcpSSEServerConfig() (*McpSSEServerConfig, bool) {
	v, ok := o.value.(*McpSSEServerConfig)
	return v, ok
}

// GetMcpSdkServerConfigWithInstance reports whether the active variant is
// [*McpSdkServerConfigWithInstance] and returns it.
func (o McpServerConfig) GetMcpSdkServerConfigWithInstance() (
	*McpSdkServerConfigWithInstance,
	bool,
) {
	v, ok := o.value.(*McpSdkServerConfigWithInstance)
	return v, ok
}

// GetUnknown reports whether the active variant is [*McpServerConfigUnknown] and returns it.
func (o McpServerConfig) GetUnknown() (*McpServerConfigUnknown, bool) {
	v, ok := o.value.(*McpServerConfigUnknown)
	return v, ok
}

// GetMcpStdioServerConfig reports whether the active variant is [*McpStdioServerConfig] and returns
// it.
func (o McpServerConfig) GetMcpStdioServerConfig() (*McpStdioServerConfig, bool) {
	v, ok := o.value.(*McpStdioServerConfig)
	return v, ok
}

// GetMcpStdioServerConfig reports whether the active variant is [*McpStdioServerConfig] and
// returns it.
func (o McpServerStatusConfig) GetMcpStdioServerConfig() (*McpStdioServerConfig, bool) {
	v, ok := o.value.(*McpStdioServerConfig)
	return v, ok
}

// GetMcpSSEServerConfig reports whether the active variant is [*McpSSEServerConfig] and returns it.
func (o McpServerStatusConfig) GetMcpSSEServerConfig() (*McpSSEServerConfig, bool) {
	v, ok := o.value.(*McpSSEServerConfig)
	return v, ok
}

// GetMcpHttpServerConfig reports whether the active variant is [*McpHttpServerConfig] and returns
// it.
func (o McpServerStatusConfig) GetMcpHttpServerConfig() (*McpHttpServerConfig, bool) {
	v, ok := o.value.(*McpHttpServerConfig)
	return v, ok
}

// GetMcpSdkServerConfig reports whether the active variant is [*McpSdkServerConfig] and returns it.
func (o McpServerStatusConfig) GetMcpSdkServerConfig() (*McpSdkServerConfig, bool) {
	v, ok := o.value.(*McpSdkServerConfig)
	return v, ok
}

// GetMcpClaudeAIProxyServerConfig reports whether the active variant is
// [*McpClaudeAIProxyServerConfig] and returns it.
func (o McpServerStatusConfig) GetMcpClaudeAIProxyServerConfig() (*McpClaudeAIProxyServerConfig, bool) {
	v, ok := o.value.(*McpClaudeAIProxyServerConfig)
	return v, ok
}

// GetUnknown reports whether the active variant is [*McpServerStatusConfigUnknown] and returns it.
func (o McpServerStatusConfig) GetUnknown() (*McpServerStatusConfigUnknown, bool) {
	v, ok := o.value.(*McpServerStatusConfigUnknown)
	return v, ok
}

// NewSDKMessage wraps a [SDKMessage_Value] variant into a [SDKMessage].
func NewSDKMessage(v SDKMessage_Value) SDKMessage { return SDKMessage{value: v} }

// GetValue returns the active [SDKMessage_Value] variant, or nil when unset.
func (o SDKMessage) GetValue() SDKMessage_Value { return o.value }

// GetSDKAssistantMessage reports whether the active variant is [*SDKAssistantMessage] and returns
// it.
func (o SDKMessage) GetSDKAssistantMessage() (*SDKAssistantMessage, bool) {
	v, ok := o.value.(*SDKAssistantMessage)
	return v, ok
}

// GetSDKAuthStatusMessage reports whether the active variant is [*SDKAuthStatusMessage] and returns
// it.
func (o SDKMessage) GetSDKAuthStatusMessage() (*SDKAuthStatusMessage, bool) {
	v, ok := o.value.(*SDKAuthStatusMessage)
	return v, ok
}

// GetSDKCommandsChangedMessage reports whether the active variant is [*SDKCommandsChangedMessage]
// and returns it.
func (o SDKMessage) GetSDKCommandsChangedMessage() (*SDKCommandsChangedMessage, bool) {
	v, ok := o.value.(*SDKCommandsChangedMessage)
	return v, ok
}

// GetSDKCompactBoundaryMessage reports whether the active variant is [*SDKCompactBoundaryMessage]
// and returns it.
func (o SDKMessage) GetSDKCompactBoundaryMessage() (*SDKCompactBoundaryMessage, bool) {
	v, ok := o.value.(*SDKCompactBoundaryMessage)
	return v, ok
}

// GetSDKFilesPersistedEvent reports whether the active variant is [*SDKFilesPersistedEvent] and
// returns it.
func (o SDKMessage) GetSDKFilesPersistedEvent() (*SDKFilesPersistedEvent, bool) {
	v, ok := o.value.(*SDKFilesPersistedEvent)
	return v, ok
}

// GetSDKHookProgressMessage reports whether the active variant is [*SDKHookProgressMessage] and
// returns it.
func (o SDKMessage) GetSDKHookProgressMessage() (*SDKHookProgressMessage, bool) {
	v, ok := o.value.(*SDKHookProgressMessage)
	return v, ok
}

// GetSDKHookResponseMessage reports whether the active variant is [*SDKHookResponseMessage] and
// returns it.
func (o SDKMessage) GetSDKHookResponseMessage() (*SDKHookResponseMessage, bool) {
	v, ok := o.value.(*SDKHookResponseMessage)
	return v, ok
}

// GetSDKHookStartedMessage reports whether the active variant is [*SDKHookStartedMessage] and
// returns it.
func (o SDKMessage) GetSDKHookStartedMessage() (*SDKHookStartedMessage, bool) {
	v, ok := o.value.(*SDKHookStartedMessage)
	return v, ok
}

// GetSDKLocalCommandOutputMessage reports whether the active variant is
// [*SDKLocalCommandOutputMessage] and returns it.
func (o SDKMessage) GetSDKLocalCommandOutputMessage() (*SDKLocalCommandOutputMessage, bool) {
	v, ok := o.value.(*SDKLocalCommandOutputMessage)
	return v, ok
}

// GetUnknown reports whether the active variant is [*SDKMessageUnknown] and returns it.
func (o SDKMessage) GetUnknown() (*SDKMessageUnknown, bool) {
	v, ok := o.value.(*SDKMessageUnknown)
	return v, ok
}

// GetSDKPartialAssistantMessage reports whether the active variant is [*SDKPartialAssistantMessage]
// and returns it.
func (o SDKMessage) GetSDKPartialAssistantMessage() (*SDKPartialAssistantMessage, bool) {
	v, ok := o.value.(*SDKPartialAssistantMessage)
	return v, ok
}

// GetSDKPermissionDeniedMessage reports whether the active variant is [*SDKPermissionDeniedMessage]
// and returns it.
func (o SDKMessage) GetSDKPermissionDeniedMessage() (*SDKPermissionDeniedMessage, bool) {
	v, ok := o.value.(*SDKPermissionDeniedMessage)
	return v, ok
}

// GetSDKPluginInstallMessage reports whether the active variant is [*SDKPluginInstallMessage] and
// returns it.
func (o SDKMessage) GetSDKPluginInstallMessage() (*SDKPluginInstallMessage, bool) {
	v, ok := o.value.(*SDKPluginInstallMessage)
	return v, ok
}

// GetSDKPromptSuggestionMessage reports whether the active variant is [*SDKPromptSuggestionMessage]
// and returns it.
func (o SDKMessage) GetSDKPromptSuggestionMessage() (*SDKPromptSuggestionMessage, bool) {
	v, ok := o.value.(*SDKPromptSuggestionMessage)
	return v, ok
}

// GetSDKRateLimitEvent reports whether the active variant is [*SDKRateLimitEvent] and returns it.
func (o SDKMessage) GetSDKRateLimitEvent() (*SDKRateLimitEvent, bool) {
	v, ok := o.value.(*SDKRateLimitEvent)
	return v, ok
}

// GetSDKResultMessageError reports whether the active variant is [*SDKResultMessageError] and
// returns it.
func (o SDKMessage) GetSDKResultMessageError() (*SDKResultMessageError, bool) {
	v, ok := o.value.(*SDKResultMessageError)
	return v, ok
}

// GetSDKResultMessageSuccess reports whether the active variant is [*SDKResultMessageSuccess] and
// returns it.
func (o SDKMessage) GetSDKResultMessageSuccess() (*SDKResultMessageSuccess, bool) {
	v, ok := o.value.(*SDKResultMessageSuccess)
	return v, ok
}

// GetSDKResultMessageUnknown reports whether the active variant is [*SDKResultMessageUnknown] and
// returns it.
func (o SDKMessage) GetSDKResultMessageUnknown() (*SDKResultMessageUnknown, bool) {
	v, ok := o.value.(*SDKResultMessageUnknown)
	return v, ok
}

// GetSDKStatusMessage reports whether the active variant is [*SDKStatusMessage] and returns it.
func (o SDKMessage) GetSDKStatusMessage() (*SDKStatusMessage, bool) {
	v, ok := o.value.(*SDKStatusMessage)
	return v, ok
}

// GetSDKSystemMessage reports whether the active variant is [*SDKSystemMessage] and returns it.
func (o SDKMessage) GetSDKSystemMessage() (*SDKSystemMessage, bool) {
	v, ok := o.value.(*SDKSystemMessage)
	return v, ok
}

// GetSDKTaskNotificationMessage reports whether the active variant is [*SDKTaskNotificationMessage]
// and returns it.
func (o SDKMessage) GetSDKTaskNotificationMessage() (*SDKTaskNotificationMessage, bool) {
	v, ok := o.value.(*SDKTaskNotificationMessage)
	return v, ok
}

// GetSDKTaskProgressMessage reports whether the active variant is [*SDKTaskProgressMessage] and
// returns it.
func (o SDKMessage) GetSDKTaskProgressMessage() (*SDKTaskProgressMessage, bool) {
	v, ok := o.value.(*SDKTaskProgressMessage)
	return v, ok
}

// GetSDKTaskStartedMessage reports whether the active variant is [*SDKTaskStartedMessage] and
// returns it.
func (o SDKMessage) GetSDKTaskStartedMessage() (*SDKTaskStartedMessage, bool) {
	v, ok := o.value.(*SDKTaskStartedMessage)
	return v, ok
}

// GetSDKTaskUpdatedMessage reports whether the active variant is [*SDKTaskUpdatedMessage] and
// returns it.
func (o SDKMessage) GetSDKTaskUpdatedMessage() (*SDKTaskUpdatedMessage, bool) {
	v, ok := o.value.(*SDKTaskUpdatedMessage)
	return v, ok
}

// GetSDKToolProgressMessage reports whether the active variant is [*SDKToolProgressMessage] and
// returns it.
func (o SDKMessage) GetSDKToolProgressMessage() (*SDKToolProgressMessage, bool) {
	v, ok := o.value.(*SDKToolProgressMessage)
	return v, ok
}

// GetSDKToolUseSummaryMessage reports whether the active variant is [*SDKToolUseSummaryMessage] and
// returns it.
func (o SDKMessage) GetSDKToolUseSummaryMessage() (*SDKToolUseSummaryMessage, bool) {
	v, ok := o.value.(*SDKToolUseSummaryMessage)
	return v, ok
}

// GetSDKUserMessage reports whether the active variant is [*SDKUserMessage] and returns it.
func (o SDKMessage) GetSDKUserMessage() (*SDKUserMessage, bool) {
	v, ok := o.value.(*SDKUserMessage)
	return v, ok
}

// GetSDKUserMessageReplay reports whether the active variant is [*SDKUserMessageReplay] and returns
// it.
func (o SDKMessage) GetSDKUserMessageReplay() (*SDKUserMessageReplay, bool) {
	v, ok := o.value.(*SDKUserMessageReplay)
	return v, ok
}

// NewHookInput wraps a [HookInput_Value] variant into a [HookInput].
func NewHookInput(v HookInput_Value) HookInput { return HookInput{value: v} }

// GetValue returns the active [HookInput_Value] variant, or nil when unset.
func (o HookInput) GetValue() HookInput_Value { return o.value }

// GetConfigChangeHookInput reports whether the active variant is [*ConfigChangeHookInput] and
// returns it.
func (o HookInput) GetConfigChangeHookInput() (*ConfigChangeHookInput, bool) {
	v, ok := o.value.(*ConfigChangeHookInput)
	return v, ok
}

// GetUnknown reports whether the active variant is [*HookInputUnknown] and returns it.
func (o HookInput) GetUnknown() (*HookInputUnknown, bool) {
	v, ok := o.value.(*HookInputUnknown)
	return v, ok
}

// GetMessageDisplayHookInput reports whether the active variant is [*MessageDisplayHookInput] and
// returns it.
func (o HookInput) GetMessageDisplayHookInput() (*MessageDisplayHookInput, bool) {
	v, ok := o.value.(*MessageDisplayHookInput)
	return v, ok
}

// GetNotificationHookInput reports whether the active variant is [*NotificationHookInput] and
// returns it.
func (o HookInput) GetNotificationHookInput() (*NotificationHookInput, bool) {
	v, ok := o.value.(*NotificationHookInput)
	return v, ok
}

// GetPermissionRequestHookInput reports whether the active variant is [*PermissionRequestHookInput]
// and returns it.
func (o HookInput) GetPermissionRequestHookInput() (*PermissionRequestHookInput, bool) {
	v, ok := o.value.(*PermissionRequestHookInput)
	return v, ok
}

// GetPostToolBatchHookInput reports whether the active variant is [*PostToolBatchHookInput] and
// returns it.
func (o HookInput) GetPostToolBatchHookInput() (*PostToolBatchHookInput, bool) {
	v, ok := o.value.(*PostToolBatchHookInput)
	return v, ok
}

// GetPostToolUseFailureHookInput reports whether the active variant is
// [*PostToolUseFailureHookInput] and returns it.
func (o HookInput) GetPostToolUseFailureHookInput() (*PostToolUseFailureHookInput, bool) {
	v, ok := o.value.(*PostToolUseFailureHookInput)
	return v, ok
}

// GetPostToolUseHookInput reports whether the active variant is [*PostToolUseHookInput] and returns
// it.
func (o HookInput) GetPostToolUseHookInput() (*PostToolUseHookInput, bool) {
	v, ok := o.value.(*PostToolUseHookInput)
	return v, ok
}

// GetPreCompactHookInput reports whether the active variant is [*PreCompactHookInput] and returns
// it.
func (o HookInput) GetPreCompactHookInput() (*PreCompactHookInput, bool) {
	v, ok := o.value.(*PreCompactHookInput)
	return v, ok
}

// GetPreToolUseHookInput reports whether the active variant is [*PreToolUseHookInput] and returns
// it.
func (o HookInput) GetPreToolUseHookInput() (*PreToolUseHookInput, bool) {
	v, ok := o.value.(*PreToolUseHookInput)
	return v, ok
}

// GetSessionEndHookInput reports whether the active variant is [*SessionEndHookInput] and returns
// it.
func (o HookInput) GetSessionEndHookInput() (*SessionEndHookInput, bool) {
	v, ok := o.value.(*SessionEndHookInput)
	return v, ok
}

// GetSessionStartHookInput reports whether the active variant is [*SessionStartHookInput] and
// returns it.
func (o HookInput) GetSessionStartHookInput() (*SessionStartHookInput, bool) {
	v, ok := o.value.(*SessionStartHookInput)
	return v, ok
}

// GetSetupHookInput reports whether the active variant is [*SetupHookInput] and returns it.
func (o HookInput) GetSetupHookInput() (*SetupHookInput, bool) {
	v, ok := o.value.(*SetupHookInput)
	return v, ok
}

// GetStopHookInput reports whether the active variant is [*StopHookInput] and returns it.
func (o HookInput) GetStopHookInput() (*StopHookInput, bool) {
	v, ok := o.value.(*StopHookInput)
	return v, ok
}

// GetSubagentStartHookInput reports whether the active variant is [*SubagentStartHookInput] and
// returns it.
func (o HookInput) GetSubagentStartHookInput() (*SubagentStartHookInput, bool) {
	v, ok := o.value.(*SubagentStartHookInput)
	return v, ok
}

// GetSubagentStopHookInput reports whether the active variant is [*SubagentStopHookInput] and
// returns it.
func (o HookInput) GetSubagentStopHookInput() (*SubagentStopHookInput, bool) {
	v, ok := o.value.(*SubagentStopHookInput)
	return v, ok
}

// GetTaskCompletedHookInput reports whether the active variant is [*TaskCompletedHookInput] and
// returns it.
func (o HookInput) GetTaskCompletedHookInput() (*TaskCompletedHookInput, bool) {
	v, ok := o.value.(*TaskCompletedHookInput)
	return v, ok
}

// GetTeammateIdleHookInput reports whether the active variant is [*TeammateIdleHookInput] and
// returns it.
func (o HookInput) GetTeammateIdleHookInput() (*TeammateIdleHookInput, bool) {
	v, ok := o.value.(*TeammateIdleHookInput)
	return v, ok
}

// GetUserPromptSubmitHookInput reports whether the active variant is [*UserPromptSubmitHookInput]
// and returns it.
func (o HookInput) GetUserPromptSubmitHookInput() (*UserPromptSubmitHookInput, bool) {
	v, ok := o.value.(*UserPromptSubmitHookInput)
	return v, ok
}

// GetWorktreeCreateHookInput reports whether the active variant is [*WorktreeCreateHookInput] and
// returns it.
func (o HookInput) GetWorktreeCreateHookInput() (*WorktreeCreateHookInput, bool) {
	v, ok := o.value.(*WorktreeCreateHookInput)
	return v, ok
}

// GetWorktreeRemoveHookInput reports whether the active variant is [*WorktreeRemoveHookInput] and
// returns it.
func (o HookInput) GetWorktreeRemoveHookInput() (*WorktreeRemoveHookInput, bool) {
	v, ok := o.value.(*WorktreeRemoveHookInput)
	return v, ok
}

// NewHookJSONOutput wraps a [HookJSONOutput_Value] variant into a [HookJSONOutput].
func NewHookJSONOutput(v HookJSONOutput_Value) HookJSONOutput { return HookJSONOutput{value: v} }

// GetValue returns the active [HookJSONOutput_Value] variant, or nil when unset.
func (o HookJSONOutput) GetValue() HookJSONOutput_Value { return o.value }

// GetAsyncHookJSONOutput reports whether the active variant is [*AsyncHookJSONOutput] and returns
// it.
func (o HookJSONOutput) GetAsyncHookJSONOutput() (*AsyncHookJSONOutput, bool) {
	v, ok := o.value.(*AsyncHookJSONOutput)
	return v, ok
}

// GetUnknown reports whether the active variant is [*HookJSONOutputUnknown] and returns it.
func (o HookJSONOutput) GetUnknown() (*HookJSONOutputUnknown, bool) {
	v, ok := o.value.(*HookJSONOutputUnknown)
	return v, ok
}

// GetSyncHookJSONOutput reports whether the active variant is [*SyncHookJSONOutput] and returns it.
func (o HookJSONOutput) GetSyncHookJSONOutput() (*SyncHookJSONOutput, bool) {
	v, ok := o.value.(*SyncHookJSONOutput)
	return v, ok
}

// NewHookSpecificOutput wraps a [HookSpecificOutput_Value] variant into a [HookSpecificOutput].
func NewHookSpecificOutput(v HookSpecificOutput_Value) HookSpecificOutput {
	return HookSpecificOutput{value: v}
}

// GetValue returns the active [HookSpecificOutput_Value] variant, or nil when unset.
func (o HookSpecificOutput) GetValue() HookSpecificOutput_Value { return o.value }

// GetNotification reports whether the active variant is [*HookSpecificOutputNotification] and
// returns it.
func (o HookSpecificOutput) GetNotification() (*HookSpecificOutputNotification, bool) {
	v, ok := o.value.(*HookSpecificOutputNotification)
	return v, ok
}

// GetPermissionRequest reports whether the active variant is [*HookSpecificOutputPermissionRequest]
// and returns it.
func (o HookSpecificOutput) GetPermissionRequest() (*HookSpecificOutputPermissionRequest, bool) {
	v, ok := o.value.(*HookSpecificOutputPermissionRequest)
	return v, ok
}

// GetPostToolBatch reports whether the active variant is [*HookSpecificOutputPostToolBatch] and
// returns it.
func (o HookSpecificOutput) GetPostToolBatch() (*HookSpecificOutputPostToolBatch, bool) {
	v, ok := o.value.(*HookSpecificOutputPostToolBatch)
	return v, ok
}

// GetPostToolUse reports whether the active variant is [*HookSpecificOutputPostToolUse] and returns
// it.
func (o HookSpecificOutput) GetPostToolUse() (*HookSpecificOutputPostToolUse, bool) {
	v, ok := o.value.(*HookSpecificOutputPostToolUse)
	return v, ok
}

// GetPostToolUseFailure reports whether the active variant is
// [*HookSpecificOutputPostToolUseFailure] and returns it.
func (o HookSpecificOutput) GetPostToolUseFailure() (*HookSpecificOutputPostToolUseFailure, bool) {
	v, ok := o.value.(*HookSpecificOutputPostToolUseFailure)
	return v, ok
}

// GetPreToolUse reports whether the active variant is [*HookSpecificOutputPreToolUse] and returns
// it.
func (o HookSpecificOutput) GetPreToolUse() (*HookSpecificOutputPreToolUse, bool) {
	v, ok := o.value.(*HookSpecificOutputPreToolUse)
	return v, ok
}

// GetSessionStart reports whether the active variant is [*HookSpecificOutputSessionStart] and
// returns it.
func (o HookSpecificOutput) GetSessionStart() (*HookSpecificOutputSessionStart, bool) {
	v, ok := o.value.(*HookSpecificOutputSessionStart)
	return v, ok
}

// GetSetup reports whether the active variant is [*HookSpecificOutputSetup] and returns it.
func (o HookSpecificOutput) GetSetup() (*HookSpecificOutputSetup, bool) {
	v, ok := o.value.(*HookSpecificOutputSetup)
	return v, ok
}

// GetSubagentStart reports whether the active variant is [*HookSpecificOutputSubagentStart] and
// returns it.
func (o HookSpecificOutput) GetSubagentStart() (*HookSpecificOutputSubagentStart, bool) {
	v, ok := o.value.(*HookSpecificOutputSubagentStart)
	return v, ok
}

// GetUnknown reports whether the active variant is [*HookSpecificOutputUnknown] and returns it.
func (o HookSpecificOutput) GetUnknown() (*HookSpecificOutputUnknown, bool) {
	v, ok := o.value.(*HookSpecificOutputUnknown)
	return v, ok
}

// GetUserPromptSubmit reports whether the active variant is [*HookSpecificOutputUserPromptSubmit]
// and returns it.
func (o HookSpecificOutput) GetUserPromptSubmit() (*HookSpecificOutputUserPromptSubmit, bool) {
	v, ok := o.value.(*HookSpecificOutputUserPromptSubmit)
	return v, ok
}

// NewPermissionRequestDecision wraps a [PermissionRequestDecision_Value] variant into a
// [PermissionRequestDecision].
func NewPermissionRequestDecision(v PermissionRequestDecision_Value) PermissionRequestDecision {
	return PermissionRequestDecision{value: v}
}

// GetValue returns the active [PermissionRequestDecision_Value] variant, or nil when unset.
func (o PermissionRequestDecision) GetValue() PermissionRequestDecision_Value { return o.value }

// GetAllow reports whether the active variant is [*PermissionRequestDecisionAllow] and returns it.
func (o PermissionRequestDecision) GetAllow() (*PermissionRequestDecisionAllow, bool) {
	v, ok := o.value.(*PermissionRequestDecisionAllow)
	return v, ok
}

// GetDeny reports whether the active variant is [*PermissionRequestDecisionDeny] and returns it.
func (o PermissionRequestDecision) GetDeny() (*PermissionRequestDecisionDeny, bool) {
	v, ok := o.value.(*PermissionRequestDecisionDeny)
	return v, ok
}

// GetUnknown reports whether the active variant is [*PermissionRequestDecisionUnknown] and returns
// it.
func (o PermissionRequestDecision) GetUnknown() (*PermissionRequestDecisionUnknown, bool) {
	v, ok := o.value.(*PermissionRequestDecisionUnknown)
	return v, ok
}

// NewToolInputSchemas wraps a [ToolInputSchemas_Value] variant into a [ToolInputSchemas].
func NewToolInputSchemas(
	v ToolInputSchemas_Value,
) ToolInputSchemas {
	return ToolInputSchemas{value: v}
}

// GetValue returns the active [ToolInputSchemas_Value] variant, or nil when unset.
func (o ToolInputSchemas) GetValue() ToolInputSchemas_Value { return o.value }

// GetAgentInput reports whether the active variant is [*AgentInput] and returns it.
func (o ToolInputSchemas) GetAgentInput() (*AgentInput, bool) {
	v, ok := o.value.(*AgentInput)
	return v, ok
}

// GetAskUserQuestionInput reports whether the active variant is [*AskUserQuestionInput] and returns
// it.
func (o ToolInputSchemas) GetAskUserQuestionInput() (*AskUserQuestionInput, bool) {
	v, ok := o.value.(*AskUserQuestionInput)
	return v, ok
}

// GetBashInput reports whether the active variant is [*BashInput] and returns it.
func (o ToolInputSchemas) GetBashInput() (*BashInput, bool) {
	v, ok := o.value.(*BashInput)
	return v, ok
}

// GetEnterWorktreeInput reports whether the active variant is [*EnterWorktreeInput] and returns it.
func (o ToolInputSchemas) GetEnterWorktreeInput() (*EnterWorktreeInput, bool) {
	v, ok := o.value.(*EnterWorktreeInput)
	return v, ok
}

// GetExitPlanModeInput reports whether the active variant is [*ExitPlanModeInput] and returns it.
func (o ToolInputSchemas) GetExitPlanModeInput() (*ExitPlanModeInput, bool) {
	v, ok := o.value.(*ExitPlanModeInput)
	return v, ok
}

// GetFileEditInput reports whether the active variant is [*FileEditInput] and returns it.
func (o ToolInputSchemas) GetFileEditInput() (*FileEditInput, bool) {
	v, ok := o.value.(*FileEditInput)
	return v, ok
}

// GetFileReadInput reports whether the active variant is [*FileReadInput] and returns it.
func (o ToolInputSchemas) GetFileReadInput() (*FileReadInput, bool) {
	v, ok := o.value.(*FileReadInput)
	return v, ok
}

// GetFileWriteInput reports whether the active variant is [*FileWriteInput] and returns it.
func (o ToolInputSchemas) GetFileWriteInput() (*FileWriteInput, bool) {
	v, ok := o.value.(*FileWriteInput)
	return v, ok
}

// GetGlobInput reports whether the active variant is [*GlobInput] and returns it.
func (o ToolInputSchemas) GetGlobInput() (*GlobInput, bool) {
	v, ok := o.value.(*GlobInput)
	return v, ok
}

// GetGrepInput reports whether the active variant is [*GrepInput] and returns it.
func (o ToolInputSchemas) GetGrepInput() (*GrepInput, bool) {
	v, ok := o.value.(*GrepInput)
	return v, ok
}

// GetListMcpResourcesInput reports whether the active variant is [*ListMcpResourcesInput] and
// returns it.
func (o ToolInputSchemas) GetListMcpResourcesInput() (*ListMcpResourcesInput, bool) {
	v, ok := o.value.(*ListMcpResourcesInput)
	return v, ok
}

// GetMcpInput reports whether the active variant is [*McpInput] and returns it.
func (o ToolInputSchemas) GetMcpInput() (*McpInput, bool) {
	v, ok := o.value.(*McpInput)
	return v, ok
}

// GetMonitorInput reports whether the active variant is [*MonitorInput] and returns it.
func (o ToolInputSchemas) GetMonitorInput() (*MonitorInput, bool) {
	v, ok := o.value.(*MonitorInput)
	return v, ok
}

// GetNotebookEditInput reports whether the active variant is [*NotebookEditInput] and returns it.
func (o ToolInputSchemas) GetNotebookEditInput() (*NotebookEditInput, bool) {
	v, ok := o.value.(*NotebookEditInput)
	return v, ok
}

// GetReadMcpResourceInput reports whether the active variant is [*ReadMcpResourceInput] and returns
// it.
func (o ToolInputSchemas) GetReadMcpResourceInput() (*ReadMcpResourceInput, bool) {
	v, ok := o.value.(*ReadMcpResourceInput)
	return v, ok
}

// GetTaskCreateInput reports whether the active variant is [*TaskCreateInput] and returns it.
func (o ToolInputSchemas) GetTaskCreateInput() (*TaskCreateInput, bool) {
	v, ok := o.value.(*TaskCreateInput)
	return v, ok
}

// GetTaskGetInput reports whether the active variant is [*TaskGetInput] and returns it.
func (o ToolInputSchemas) GetTaskGetInput() (*TaskGetInput, bool) {
	v, ok := o.value.(*TaskGetInput)
	return v, ok
}

// GetTaskListInput reports whether the active variant is [*TaskListInput] and returns it.
func (o ToolInputSchemas) GetTaskListInput() (*TaskListInput, bool) {
	v, ok := o.value.(*TaskListInput)
	return v, ok
}

// GetTaskOutputInput reports whether the active variant is [*TaskOutputInput] and returns it.
func (o ToolInputSchemas) GetTaskOutputInput() (*TaskOutputInput, bool) {
	v, ok := o.value.(*TaskOutputInput)
	return v, ok
}

// GetTaskStopInput reports whether the active variant is [*TaskStopInput] and returns it.
func (o ToolInputSchemas) GetTaskStopInput() (*TaskStopInput, bool) {
	v, ok := o.value.(*TaskStopInput)
	return v, ok
}

// GetTaskUpdateInput reports whether the active variant is [*TaskUpdateInput] and returns it.
func (o ToolInputSchemas) GetTaskUpdateInput() (*TaskUpdateInput, bool) {
	v, ok := o.value.(*TaskUpdateInput)
	return v, ok
}

// GetTodoWriteInput reports whether the active variant is [*TodoWriteInput] and returns it.
func (o ToolInputSchemas) GetTodoWriteInput() (*TodoWriteInput, bool) {
	v, ok := o.value.(*TodoWriteInput)
	return v, ok
}

// GetToolInputUnknown reports whether the active variant is [*ToolInputUnknown] and returns it.
func (o ToolInputSchemas) GetToolInputUnknown() (*ToolInputUnknown, bool) {
	v, ok := o.value.(*ToolInputUnknown)
	return v, ok
}

// GetWebFetchInput reports whether the active variant is [*WebFetchInput] and returns it.
func (o ToolInputSchemas) GetWebFetchInput() (*WebFetchInput, bool) {
	v, ok := o.value.(*WebFetchInput)
	return v, ok
}

// GetWebSearchInput reports whether the active variant is [*WebSearchInput] and returns it.
func (o ToolInputSchemas) GetWebSearchInput() (*WebSearchInput, bool) {
	v, ok := o.value.(*WebSearchInput)
	return v, ok
}

// GetWorkflowInput reports whether the active variant is [*WorkflowInput] and returns it.
func (o ToolInputSchemas) GetWorkflowInput() (*WorkflowInput, bool) {
	v, ok := o.value.(*WorkflowInput)
	return v, ok
}

// NewToolOutputSchemas wraps a [ToolOutputSchemas_Value] variant into a [ToolOutputSchemas].
func NewToolOutputSchemas(v ToolOutputSchemas_Value) ToolOutputSchemas {
	return ToolOutputSchemas{value: v}
}

// GetValue returns the active [ToolOutputSchemas_Value] variant, or nil when unset.
func (o ToolOutputSchemas) GetValue() ToolOutputSchemas_Value { return o.value }

// GetAgentOutputAsyncLaunched reports whether the active variant is [*AgentOutputAsyncLaunched] and
// returns it.
func (o ToolOutputSchemas) GetAgentOutputAsyncLaunched() (*AgentOutputAsyncLaunched, bool) {
	v, ok := o.value.(*AgentOutputAsyncLaunched)
	return v, ok
}

// GetAgentOutputCompleted reports whether the active variant is [*AgentOutputCompleted] and returns
// it.
func (o ToolOutputSchemas) GetAgentOutputCompleted() (*AgentOutputCompleted, bool) {
	v, ok := o.value.(*AgentOutputCompleted)
	return v, ok
}

// GetAgentOutputRemoteLaunched reports whether the active variant is
// [*AgentOutputRemoteLaunched] and returns it.
func (o ToolOutputSchemas) GetAgentOutputRemoteLaunched() (*AgentOutputRemoteLaunched, bool) {
	v, ok := o.value.(*AgentOutputRemoteLaunched)
	return v, ok
}

// GetAgentOutputUnknown reports whether the active variant is [*AgentOutputUnknown] and returns it.
func (o ToolOutputSchemas) GetAgentOutputUnknown() (*AgentOutputUnknown, bool) {
	v, ok := o.value.(*AgentOutputUnknown)
	return v, ok
}

// GetAskUserQuestionOutput reports whether the active variant is [*AskUserQuestionOutput] and
// returns it.
func (o ToolOutputSchemas) GetAskUserQuestionOutput() (*AskUserQuestionOutput, bool) {
	v, ok := o.value.(*AskUserQuestionOutput)
	return v, ok
}

// GetBashOutput reports whether the active variant is [*BashOutput] and returns it.
func (o ToolOutputSchemas) GetBashOutput() (*BashOutput, bool) {
	v, ok := o.value.(*BashOutput)
	return v, ok
}

// GetEnterWorktreeOutput reports whether the active variant is [*EnterWorktreeOutput] and returns
// it.
func (o ToolOutputSchemas) GetEnterWorktreeOutput() (*EnterWorktreeOutput, bool) {
	v, ok := o.value.(*EnterWorktreeOutput)
	return v, ok
}

// GetExitPlanModeOutput reports whether the active variant is [*ExitPlanModeOutput] and returns it.
func (o ToolOutputSchemas) GetExitPlanModeOutput() (*ExitPlanModeOutput, bool) {
	v, ok := o.value.(*ExitPlanModeOutput)
	return v, ok
}

// GetFileEditOutput reports whether the active variant is [*FileEditOutput] and returns it.
func (o ToolOutputSchemas) GetFileEditOutput() (*FileEditOutput, bool) {
	v, ok := o.value.(*FileEditOutput)
	return v, ok
}

// GetFileReadOutputImage reports whether the active variant is [*FileReadOutputImage] and returns
// it.
func (o ToolOutputSchemas) GetFileReadOutputImage() (*FileReadOutputImage, bool) {
	v, ok := o.value.(*FileReadOutputImage)
	return v, ok
}

// GetFileReadOutputNotebook reports whether the active variant is [*FileReadOutputNotebook] and
// returns it.
func (o ToolOutputSchemas) GetFileReadOutputNotebook() (*FileReadOutputNotebook, bool) {
	v, ok := o.value.(*FileReadOutputNotebook)
	return v, ok
}

// GetFileReadOutputParts reports whether the active variant is [*FileReadOutputParts] and returns
// it.
func (o ToolOutputSchemas) GetFileReadOutputParts() (*FileReadOutputParts, bool) {
	v, ok := o.value.(*FileReadOutputParts)
	return v, ok
}

// GetFileReadOutputPdf reports whether the active variant is [*FileReadOutputPdf] and returns it.
func (o ToolOutputSchemas) GetFileReadOutputPdf() (*FileReadOutputPdf, bool) {
	v, ok := o.value.(*FileReadOutputPdf)
	return v, ok
}

// GetFileReadOutputText reports whether the active variant is [*FileReadOutputText] and returns it.
func (o ToolOutputSchemas) GetFileReadOutputText() (*FileReadOutputText, bool) {
	v, ok := o.value.(*FileReadOutputText)
	return v, ok
}

// GetFileReadOutputUnknown reports whether the active variant is [*FileReadOutputUnknown] and
// returns it.
func (o ToolOutputSchemas) GetFileReadOutputUnknown() (*FileReadOutputUnknown, bool) {
	v, ok := o.value.(*FileReadOutputUnknown)
	return v, ok
}

// GetFileWriteOutput reports whether the active variant is [*FileWriteOutput] and returns it.
func (o ToolOutputSchemas) GetFileWriteOutput() (*FileWriteOutput, bool) {
	v, ok := o.value.(*FileWriteOutput)
	return v, ok
}

// GetGlobOutput reports whether the active variant is [*GlobOutput] and returns it.
func (o ToolOutputSchemas) GetGlobOutput() (*GlobOutput, bool) {
	v, ok := o.value.(*GlobOutput)
	return v, ok
}

// GetGrepOutput reports whether the active variant is [*GrepOutput] and returns it.
func (o ToolOutputSchemas) GetGrepOutput() (*GrepOutput, bool) {
	v, ok := o.value.(*GrepOutput)
	return v, ok
}

// GetListMcpResourcesOutput reports whether the active variant is [*ListMcpResourcesOutput] and
// returns it.
func (o ToolOutputSchemas) GetListMcpResourcesOutput() (*ListMcpResourcesOutput, bool) {
	v, ok := o.value.(*ListMcpResourcesOutput)
	return v, ok
}

// GetMonitorOutput reports whether the active variant is [*MonitorOutput] and returns it.
func (o ToolOutputSchemas) GetMonitorOutput() (*MonitorOutput, bool) {
	v, ok := o.value.(*MonitorOutput)
	return v, ok
}

// GetNotebookEditOutput reports whether the active variant is [*NotebookEditOutput] and returns it.
func (o ToolOutputSchemas) GetNotebookEditOutput() (*NotebookEditOutput, bool) {
	v, ok := o.value.(*NotebookEditOutput)
	return v, ok
}

// GetReadMcpResourceOutput reports whether the active variant is [*ReadMcpResourceOutput] and
// returns it.
func (o ToolOutputSchemas) GetReadMcpResourceOutput() (*ReadMcpResourceOutput, bool) {
	v, ok := o.value.(*ReadMcpResourceOutput)
	return v, ok
}

// GetTaskCreateOutput reports whether the active variant is [*TaskCreateOutput] and returns it.
func (o ToolOutputSchemas) GetTaskCreateOutput() (*TaskCreateOutput, bool) {
	v, ok := o.value.(*TaskCreateOutput)
	return v, ok
}

// GetTaskGetOutput reports whether the active variant is [*TaskGetOutput] and returns it.
func (o ToolOutputSchemas) GetTaskGetOutput() (*TaskGetOutput, bool) {
	v, ok := o.value.(*TaskGetOutput)
	return v, ok
}

// GetTaskListOutput reports whether the active variant is [*TaskListOutput] and returns it.
func (o ToolOutputSchemas) GetTaskListOutput() (*TaskListOutput, bool) {
	v, ok := o.value.(*TaskListOutput)
	return v, ok
}

// GetTaskStopOutput reports whether the active variant is [*TaskStopOutput] and returns it.
func (o ToolOutputSchemas) GetTaskStopOutput() (*TaskStopOutput, bool) {
	v, ok := o.value.(*TaskStopOutput)
	return v, ok
}

// GetTaskUpdateOutput reports whether the active variant is [*TaskUpdateOutput] and returns it.
func (o ToolOutputSchemas) GetTaskUpdateOutput() (*TaskUpdateOutput, bool) {
	v, ok := o.value.(*TaskUpdateOutput)
	return v, ok
}

// GetTodoWriteOutput reports whether the active variant is [*TodoWriteOutput] and returns it.
func (o ToolOutputSchemas) GetTodoWriteOutput() (*TodoWriteOutput, bool) {
	v, ok := o.value.(*TodoWriteOutput)
	return v, ok
}

// GetToolOutputUnknown reports whether the active variant is [*ToolOutputUnknown] and returns it.
func (o ToolOutputSchemas) GetToolOutputUnknown() (*ToolOutputUnknown, bool) {
	v, ok := o.value.(*ToolOutputUnknown)
	return v, ok
}

// GetWebFetchOutput reports whether the active variant is [*WebFetchOutput] and returns it.
func (o ToolOutputSchemas) GetWebFetchOutput() (*WebFetchOutput, bool) {
	v, ok := o.value.(*WebFetchOutput)
	return v, ok
}

// GetWebSearchOutput reports whether the active variant is [*WebSearchOutput] and returns it.
func (o ToolOutputSchemas) GetWebSearchOutput() (*WebSearchOutput, bool) {
	v, ok := o.value.(*WebSearchOutput)
	return v, ok
}

// GetWorkflowOutput reports whether the active variant is [*WorkflowOutput] and returns it.
func (o ToolOutputSchemas) GetWorkflowOutput() (*WorkflowOutput, bool) {
	v, ok := o.value.(*WorkflowOutput)
	return v, ok
}

// NewWebSearchOutputResult wraps a [WebSearchOutputResult_Value] variant into a
// [WebSearchOutputResult].
func NewWebSearchOutputResult(v WebSearchOutputResult_Value) WebSearchOutputResult {
	return WebSearchOutputResult{value: v}
}

// GetValue returns the active [WebSearchOutputResult_Value] variant, or nil when unset.
func (o WebSearchOutputResult) GetValue() WebSearchOutputResult_Value { return o.value }

// GetBlock reports whether the active variant is [*WebSearchOutputResultBlock] and returns it.
func (o WebSearchOutputResult) GetBlock() (*WebSearchOutputResultBlock, bool) {
	v, ok := o.value.(*WebSearchOutputResultBlock)
	return v, ok
}

// GetText reports whether the active variant is [*WebSearchOutputResultText] and returns it.
func (o WebSearchOutputResult) GetText() (*WebSearchOutputResultText, bool) {
	v, ok := o.value.(*WebSearchOutputResultText)
	return v, ok
}

// GetUnknown reports whether the active variant is [*WebSearchOutputResultUnknown] and returns it.
func (o WebSearchOutputResult) GetUnknown() (*WebSearchOutputResultUnknown, bool) {
	v, ok := o.value.(*WebSearchOutputResultUnknown)
	return v, ok
}

// decodeUnionVariant decodes data into a fresh T (invoking T's own UnmarshalJSON
// when defined) and returns a pointer for storage on a union value field. Each
// union's UnmarshalJSON / UnmarshalForTool (declared next to its type) uses it.
//
// Unions carry their discriminator in the payload (a type / behavior / subtype
// field, or the JSON token shape) so they decode themselves; ToolInputSchemas /
// ToolOutputSchemas are keyed by tool_name in the enclosing envelope instead, so
// they expose UnmarshalForTool. Unrecognized discriminators are preserved
// through each union's unknown variant.
func decodeUnionVariant[T any](data []byte) (*T, error) {
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}
