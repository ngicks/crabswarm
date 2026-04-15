// Package v1 contains handwritten Claude Agent SDK shapes derived from:
// https://code.claude.com/docs/en/agent-sdk/typescript
package sdktypesv1

import (
	"encoding/json"
	"fmt"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdktypes/v1"
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

// MessageParam is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkusermessage
type MessageParam json.RawMessage

// BetaRawMessageStreamEvent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkpartialassistantmessage
type BetaRawMessageStreamEvent json.RawMessage

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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

// PermissionUpdateDestination is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#accountinfo
type ApiKeySource string

const (
	ApiKeySourceUser      ApiKeySource = "user"
	ApiKeySourceProject   ApiKeySource = "project"
	ApiKeySourceOrg       ApiKeySource = "org"
	ApiKeySourceTemporary ApiKeySource = "temporary"
	ApiKeySourceOauth     ApiKeySource = "oauth"
)

// SdkBeta is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SdkBeta string

const (
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
	EffortMax    Effort = "max"
)

// AgentModel is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentdefinition
type AgentModel string

const (
	AgentModelSonnet  AgentModel = "sonnet"
	AgentModelOpus    AgentModel = "opus"
	AgentModelHaiku   AgentModel = "haiku"
	AgentModelInherit AgentModel = "inherit"
)

// AgentMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type AgentMode string

const (
	AgentModeAcceptEdits       AgentMode = "acceptEdits"
	AgentModeBypassPermissions AgentMode = "bypassPermissions"
	AgentModeDefault           AgentMode = "default"
	AgentModeDontAsk           AgentMode = "dontAsk"
	AgentModePlan              AgentMode = "plan"
)

// AgentIsolation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type AgentIsolation string

const (
	AgentIsolationWorktree AgentIsolation = "worktree"
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#config-2
type ConfigScope string

const (
	ConfigScopeLocal   ConfigScope = "local"
	ConfigScopeUser    ConfigScope = "user"
	ConfigScopeProject ConfigScope = "project"
)

// ServiceTier is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentoutput
type ServiceTier string

const (
	ServiceTierStandard ServiceTier = "standard"
	ServiceTierPriority ServiceTier = "priority"
	ServiceTierBatch    ServiceTier = "batch"
)

// HookEvent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hook-input
type HookEvent string

const (
	HookEventPreToolUse         HookEvent = "PreToolUse"
	HookEventPostToolUse        HookEvent = "PostToolUse"
	HookEventPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookEventNotification       HookEvent = "Notification"
	HookEventUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookEventSessionStart       HookEvent = "SessionStart"
	HookEventSessionEnd         HookEvent = "SessionEnd"
	HookEventStop               HookEvent = "Stop"
	HookEventSubagentStart      HookEvent = "SubagentStart"
	HookEventSubagentStop       HookEvent = "SubagentStop"
	HookEventPreCompact         HookEvent = "PreCompact"
	HookEventPermissionRequest  HookEvent = "PermissionRequest"
	HookEventSetup              HookEvent = "Setup"
	HookEventTeammateIdle       HookEvent = "TeammateIdle"
	HookEventTaskCompleted      HookEvent = "TaskCompleted"
	HookEventConfigChange       HookEvent = "ConfigChange"
	HookEventWorktreeCreate     HookEvent = "WorktreeCreate"
	HookEventWorktreeRemove     HookEvent = "WorktreeRemove"
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sync-hook-json-output
type HookDecision string

const (
	HookDecisionApprove HookDecision = "approve"
	HookDecisionBlock   HookDecision = "block"
)

// HookPermissionDecision is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookPermissionDecision string

const (
	HookPermissionDecisionAllow HookPermissionDecision = "allow"
	HookPermissionDecisionDeny  HookPermissionDecision = "deny"
	HookPermissionDecisionAsk   HookPermissionDecision = "ask"
)

// PermissionResultBehavior is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#canusetool
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type PermissionRequestDecisionBehavior string

const (
	PermissionRequestDecisionBehaviorAllow PermissionRequestDecisionBehavior = "allow"
	PermissionRequestDecisionBehaviorDeny  PermissionRequestDecisionBehavior = "deny"
)

func permissionRequestDecisionBehaviorToProto(v PermissionRequestDecisionBehavior) pb.PermissionRequestDecisionBehavior {
	switch v {
	case PermissionRequestDecisionBehaviorAllow:
		return pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_ALLOW
	case PermissionRequestDecisionBehaviorDeny:
		return pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_DENY
	default:
		return pb.PermissionRequestDecisionBehavior_PERMISSION_REQUEST_DECISION_BEHAVIOR_UNSPECIFIED
	}
}

func permissionRequestDecisionBehaviorFromProto(v pb.PermissionRequestDecisionBehavior) PermissionRequestDecisionBehavior {
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

// RateLimitStatus is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#ratelimitinfo
type RateLimitStatus string

const (
	RateLimitStatusAllowed        RateLimitStatus = "allowed"
	RateLimitStatusAllowedWarning RateLimitStatus = "allowed_warning"
	RateLimitStatusRejected       RateLimitStatus = "rejected"
)

// ToolUseSummaryType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type ToolUseSummaryType string

const (
	ToolUseSummaryTypeToolUseSummary ToolUseSummaryType = "tool_use_summary"
)

// PromptSuggestionType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type PromptSuggestionType string

const (
	PromptSuggestionTypePromptSuggestion PromptSuggestionType = "prompt_suggestion"
)

// SystemMessageType is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type SystemMessageType string

const (
	SystemMessageTypeSystem         SystemMessageType = "system"
	SystemMessageTypeStreamEvent    SystemMessageType = "stream_event"
	SystemMessageTypeAuthStatus     SystemMessageType = "auth_status"
	SystemMessageTypeToolProgress   SystemMessageType = "tool_progress"
	SystemMessageTypeRateLimitEvent SystemMessageType = "rate_limit_event"
	SystemMessageTypeAssistant      SystemMessageType = "assistant"
	SystemMessageTypeUser           SystemMessageType = "user"
	SystemMessageTypeResult         SystemMessageType = "result"
)

// SystemSubtype is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type SystemSubtype string

const (
	SystemSubtypeInit               SystemSubtype = "init"
	SystemSubtypeCompactBoundary    SystemSubtype = "compact_boundary"
	SystemSubtypeStatus             SystemSubtype = "status"
	SystemSubtypeTaskNotification   SystemSubtype = "task_notification"
	SystemSubtypeHookStarted        SystemSubtype = "hook_started"
	SystemSubtypeHookProgress       SystemSubtype = "hook_progress"
	SystemSubtypeHookResponse       SystemSubtype = "hook_response"
	SystemSubtypeTaskStarted        SystemSubtype = "task_started"
	SystemSubtypeTaskProgress       SystemSubtype = "task_progress"
	SystemSubtypeFilesPersisted     SystemSubtype = "files_persisted"
	SystemSubtypeLocalCommandOutput SystemSubtype = "local_command_output"
)

// SDKAssistantMessageError is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkassistantmessage
type SDKAssistantMessageError string

const (
	SDKAssistantMessageErrorAuthenticationFailed SDKAssistantMessageError = "authentication_failed"
	SDKAssistantMessageErrorBillingError         SDKAssistantMessageError = "billing_error"
	SDKAssistantMessageErrorRateLimit            SDKAssistantMessageError = "rate_limit"
	SDKAssistantMessageErrorInvalidRequest       SDKAssistantMessageError = "invalid_request"
	SDKAssistantMessageErrorServerError          SDKAssistantMessageError = "server_error"
	SDKAssistantMessageErrorMaxOutputTokens      SDKAssistantMessageError = "max_output_tokens"
	SDKAssistantMessageErrorUnknown              SDKAssistantMessageError = "unknown"
)

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
	Status      RateLimitStatus `json:"status"`
	ResetsAt    *float64        `json:"resetsAt,omitzero"`
	Utilization *float64        `json:"utilization,omitzero"`
}

// ToolAnnotations is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#toolannotations
type ToolAnnotations struct {
	ReadOnlyHint    *bool `json:"readOnlyHint,omitzero"`
	DestructiveHint *bool `json:"destructiveHint,omitzero"`
	IdempotentHint  *bool `json:"idempotentHint,omitzero"`
	OpenWorldHint   *bool `json:"openWorldHint,omitzero"`
}

// AskUserQuestionPreviewFormat is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type AskUserQuestionPreviewFormat string

const (
	AskUserQuestionPreviewFormatMarkdown AskUserQuestionPreviewFormat = "markdown"
	AskUserQuestionPreviewFormatHTML     AskUserQuestionPreviewFormat = "html"
)

// ToolConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ToolConfig struct {
	AskUserQuestion *AskUserQuestionToolConfig `json:"askUserQuestion,omitzero"`
}

// AskUserQuestionToolConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
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
type SystemPrompt interface{ systemPrompt() }

var (
	_ SystemPrompt = (*SystemPromptUnknown)(nil)
	_ SystemPrompt = (*SystemPromptString)(nil)
	_ SystemPrompt = (*SystemPromptPreset)(nil)
)

// SystemPromptUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPromptUnknown struct{ UnknownUnion }

// SystemPromptString is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPromptString struct{ Value string }

// SystemPromptPreset is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SystemPromptPreset struct {
	Type   string                 `json:"type"`
	Preset SystemPromptPresetName `json:"preset"`
	Append *string                `json:"append,omitzero"`
}

func (SystemPromptUnknown) systemPrompt() {}
func (SystemPromptString) systemPrompt()  {}
func (SystemPromptPreset) systemPrompt()  {}

// ThinkingConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfig interface{ thinkingConfig() }

var (
	_ ThinkingConfig = (*ThinkingConfigUnknown)(nil)
	_ ThinkingConfig = (*ThinkingConfigAdaptive)(nil)
	_ ThinkingConfig = (*ThinkingConfigEnabled)(nil)
	_ ThinkingConfig = (*ThinkingConfigDisabled)(nil)
)

// ThinkingConfigUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfigUnknown struct{ UnknownUnion }

// ThinkingConfigAdaptive is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfigAdaptive struct {
	Type string `json:"type"`
}

// ThinkingConfigEnabled is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type ThinkingConfigEnabled struct {
	Type         string `json:"type"`
	BudgetTokens *int64 `json:"budgetTokens,omitzero"`
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
type ToolsConfig interface{ toolsConfig() }

var (
	_ ToolsConfig = (*ToolsConfigUnknown)(nil)
	_ ToolsConfig = (*ToolsConfigList)(nil)
	_ ToolsConfig = (*ToolsConfigPreset)(nil)
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionRuleValue struct {
	ToolName    string  `json:"toolName"`
	RuleContent *string `json:"ruleContent,omitzero"`
}

// PermissionUpdate is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdate interface{ permissionUpdate() }

var (
	_ PermissionUpdate = (*PermissionUpdateUnknown)(nil)
	_ PermissionUpdate = (*PermissionUpdateAddRules)(nil)
	_ PermissionUpdate = (*PermissionUpdateReplaceRules)(nil)
	_ PermissionUpdate = (*PermissionUpdateRemoveRules)(nil)
	_ PermissionUpdate = (*PermissionUpdateSetMode)(nil)
	_ PermissionUpdate = (*PermissionUpdateAddDirectories)(nil)
	_ PermissionUpdate = (*PermissionUpdateRemoveDirectories)(nil)
)

// PermissionUpdateUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateUnknown struct{ UnknownUnion }

// PermissionUpdateAddRules is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateAddRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateReplaceRules is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateReplaceRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateRemoveRules is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateRemoveRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateSetMode is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateSetMode struct {
	Type        string                      `json:"type"`
	Mode        PermissionMode              `json:"mode"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateAddDirectories is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateAddDirectories struct {
	Type        string                      `json:"type"`
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
}

// PermissionUpdateRemoveDirectories is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permission-update
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#canusetool
type PermissionResult interface{ permissionResult() }

var (
	_ PermissionResult = (*PermissionResultUnknown)(nil)
	_ PermissionResult = (*PermissionResultAllow)(nil)
	_ PermissionResult = (*PermissionResultDeny)(nil)
)

// PermissionResultUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#canusetool
type PermissionResultUnknown struct{ UnknownUnion }

// PermissionResultAllow is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#canusetool
type PermissionResultAllow struct {
	Behavior           PermissionResultBehavior `json:"behavior"`
	UpdatedInput       map[string]any           `json:"updatedInput,omitzero"`
	UpdatedPermissions []PermissionUpdate       `json:"updatedPermissions,omitzero"`
	ToolUseID          *string                  `json:"toolUseID,omitzero"`
}

// PermissionResultDeny is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#canusetool
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type McpServerConfig interface{ mcpServerConfig() }

var (
	_ McpServerConfig = (*McpServerConfigUnknown)(nil)
	_ McpServerConfig = (*McpStdioServerConfig)(nil)
	_ McpServerConfig = (*McpSSEServerConfig)(nil)
	_ McpServerConfig = (*McpHttpServerConfig)(nil)
	_ McpServerConfig = (*McpSdkServerConfigWithInstance)(nil)
	_ McpServerConfig = (*McpClaudeAIProxyServerConfig)(nil)
)

// McpServerConfigUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type McpServerConfigUnknown struct{ UnknownUnion }

// McpStdioServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type McpStdioServerConfig struct {
	Type    *string           `json:"type,omitzero"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitzero"`
	Env     map[string]string `json:"env,omitzero"`
}

// McpSSEServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type McpSSEServerConfig struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitzero"`
}

// McpHttpServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type McpHttpServerConfig struct {
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitzero"`
}

// McpSdkServerConfigWithInstance is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#createsdkmcpserver
type McpSdkServerConfigWithInstance struct {
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Instance json.RawMessage `json:"instance"`
}

// McpClaudeAIProxyServerConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
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
func (McpClaudeAIProxyServerConfig) mcpServerConfig()   {}

// SdkPluginConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SdkPluginConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

// SandboxNetworkConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SandboxNetworkConfig struct {
	AllowedDomains          []string `json:"allowedDomains,omitzero"`
	AllowManagedDomainsOnly *bool    `json:"allowManagedDomainsOnly,omitzero"`
	AllowLocalBinding       *bool    `json:"allowLocalBinding,omitzero"`
	AllowUnixSockets        []string `json:"allowUnixSockets,omitzero"`
	AllowAllUnixSockets     *bool    `json:"allowAllUnixSockets,omitzero"`
	HttpProxyPort           *int64   `json:"httpProxyPort,omitzero"`
	SocksProxyPort          *int64   `json:"socksProxyPort,omitzero"`
}

// SandboxFilesystemConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SandboxFilesystemConfig struct {
	AllowWrite []string `json:"allowWrite,omitzero"`
	DenyWrite  []string `json:"denyWrite,omitzero"`
	DenyRead   []string `json:"denyRead,omitzero"`
}

// RipgrepConfig is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type RipgrepConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitzero"`
}

// SandboxSettings is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#options
type SandboxSettings struct {
	Enabled                   *bool                    `json:"enabled,omitzero"`
	AutoAllowBashIfSandboxed  *bool                    `json:"autoAllowBashIfSandboxed,omitzero"`
	ExcludedCommands          []string                 `json:"excludedCommands,omitzero"`
	AllowUnsandboxedCommands  *bool                    `json:"allowUnsandboxedCommands,omitzero"`
	Network                   *SandboxNetworkConfig    `json:"network,omitzero"`
	Filesystem                *SandboxFilesystemConfig `json:"filesystem,omitzero"`
	IgnoreViolations          map[string][]string      `json:"ignoreViolations,omitzero"`
	EnableWeakerNestedSandbox *bool                    `json:"enableWeakerNestedSandbox,omitzero"`
	Ripgrep                   *RipgrepConfig           `json:"ripgrep,omitzero"`
}

// AgentDefinition is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentdefinition
type AgentDefinition struct {
	Description                        string            `json:"description"`
	Tools                              []string          `json:"tools,omitzero"`
	DisallowedTools                    []string          `json:"disallowedTools,omitzero"`
	Prompt                             string            `json:"prompt"`
	Model                              *AgentModel       `json:"model,omitzero"`
	McpServers                         []json.RawMessage `json:"mcpServers,omitzero"`
	Skills                             []string          `json:"skills,omitzero"`
	MaxTurns                           *int64            `json:"maxTurns,omitzero"`
	CriticalSystemReminderExperimental *string           `json:"criticalSystemReminder_EXPERIMENTAL,omitzero"`
}

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
	Executable                      *string                    `json:"executable,omitzero"`
	ExecutableArgs                  []string                   `json:"executableArgs,omitzero"`
	ExtraArgs                       map[string]*string         `json:"extraArgs,omitzero"`
	FallbackModel                   *string                    `json:"fallbackModel,omitzero"`
	ForkSession                     *bool                      `json:"forkSession,omitzero"`
	IncludePartialMessages          *bool                      `json:"includePartialMessages,omitzero"`
	MaxBudgetUsd                    *float64                   `json:"maxBudgetUsd,omitzero"`
	MaxThinkingTokens               *int64                     `json:"maxThinkingTokens,omitzero"`
	MaxTurns                        *int64                     `json:"maxTurns,omitzero"`
	McpServers                      map[string]McpServerConfig `json:"mcpServers,omitzero"`
	Model                           *string                    `json:"model,omitzero"`
	OutputFormat                    *OutputFormat              `json:"outputFormat,omitzero"`
	PathToClaudeCodeExecutable      *string                    `json:"pathToClaudeCodeExecutable,omitzero"`
	PermissionMode                  *PermissionMode            `json:"permissionMode,omitzero"`
	PermissionPromptToolName        *string                    `json:"permissionPromptToolName,omitzero"`
	PersistSession                  *bool                      `json:"persistSession,omitzero"`
	Plugins                         []SdkPluginConfig          `json:"plugins,omitzero"`
	PromptSuggestions               *bool                      `json:"promptSuggestions,omitzero"`
	Resume                          *string                    `json:"resume,omitzero"`
	ResumeSessionAt                 *string                    `json:"resumeSessionAt,omitzero"`
	Sandbox                         *SandboxSettings           `json:"sandbox,omitzero"`
	SessionID                       *string                    `json:"sessionId,omitzero"`
	SettingSources                  []SettingSource            `json:"settingSources,omitzero"`
	StrictMcpConfig                 *bool                      `json:"strictMcpConfig,omitzero"`
	SystemPrompt                    SystemPrompt               `json:"systemPrompt,omitzero"`
	Thinking                        ThinkingConfig             `json:"thinking,omitzero"`
	ToolConfig                      *ToolConfig                `json:"toolConfig,omitzero"`
	Tools                           ToolsConfig                `json:"tools,omitzero"`
}

// SlashCommand is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#slashcommand
type SlashCommand struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argumentHint"`
}

// ModelInfo is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#modelinfo
type ModelInfo struct {
	Value                    string   `json:"value"`
	DisplayName              string   `json:"displayName"`
	Description              string   `json:"description"`
	SupportsEffort           *bool    `json:"supportsEffort,omitzero"`
	SupportedEffortLevels    []Effort `json:"supportedEffortLevels,omitzero"`
	SupportsAdaptiveThinking *bool    `json:"supportsAdaptiveThinking,omitzero"`
	SupportsFastMode         *bool    `json:"supportsFastMode,omitzero"`
}

// AgentInfo is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#agentdefinition
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
}

// Usage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#usage
type Usage struct {
	InputTokens              *int64 `json:"input_tokens"`
	OutputTokens             *int64 `json:"output_tokens"`
	CacheCreationInputTokens *int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64 `json:"cache_read_input_tokens"`
}

// NonNullableUsage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#nonnullableusage
type NonNullableUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// CallToolResultContent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#calltoolresult
type CallToolResultContent struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitzero"`
}

// CallToolResult is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#calltoolresult
type CallToolResult struct {
	Content []CallToolResultContent `json:"content"`
	IsError *bool                   `json:"isError,omitzero"`
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
	Trigger   string `json:"trigger"`
	PreTokens int64  `json:"pre_tokens"`
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type SDKMessage interface{ sdkMessage() }

var (
	_ SDKMessage = (*SDKMessageUnknown)(nil)
	_ SDKMessage = (*SDKAssistantMessage)(nil)
	_ SDKMessage = (*SDKUserMessage)(nil)
	_ SDKMessage = (*SDKUserMessageReplay)(nil)
	_ SDKMessage = (*SDKResultMessageUnknown)(nil)
	_ SDKMessage = (*SDKResultMessageSuccess)(nil)
	_ SDKMessage = (*SDKResultMessageError)(nil)
	_ SDKMessage = (*SDKSystemMessage)(nil)
	_ SDKMessage = (*SDKPartialAssistantMessage)(nil)
	_ SDKMessage = (*SDKCompactBoundaryMessage)(nil)
	_ SDKMessage = (*SDKStatusMessage)(nil)
	_ SDKMessage = (*SDKLocalCommandOutputMessage)(nil)
	_ SDKMessage = (*SDKHookStartedMessage)(nil)
	_ SDKMessage = (*SDKHookProgressMessage)(nil)
	_ SDKMessage = (*SDKHookResponseMessage)(nil)
	_ SDKMessage = (*SDKToolProgressMessage)(nil)
	_ SDKMessage = (*SDKAuthStatusMessage)(nil)
	_ SDKMessage = (*SDKTaskNotificationMessage)(nil)
	_ SDKMessage = (*SDKTaskStartedMessage)(nil)
	_ SDKMessage = (*SDKTaskProgressMessage)(nil)
	_ SDKMessage = (*SDKFilesPersistedEvent)(nil)
	_ SDKMessage = (*SDKToolUseSummaryMessage)(nil)
	_ SDKMessage = (*SDKRateLimitEvent)(nil)
	_ SDKMessage = (*SDKPromptSuggestionMessage)(nil)
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
}

func (SDKAssistantMessage) sdkMessage() {}

// SDKUserMessage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkusermessage
type SDKUserMessage struct {
	Type            string          `json:"type"`
	UUID            *UUID           `json:"uuid,omitzero"`
	SessionID       string          `json:"session_id"`
	Message         MessageParam    `json:"message"`
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	IsSynthetic     *bool           `json:"isSynthetic,omitzero"`
	ToolUseResult   json.RawMessage `json:"tool_use_result,omitzero"`
}

func (SDKUserMessage) sdkMessage() {}

// SDKUserMessageReplay is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sdkusermessagereplay
type SDKUserMessageReplay struct {
	Type            string          `json:"type"`
	UUID            UUID            `json:"uuid"`
	SessionID       string          `json:"session_id"`
	Message         MessageParam    `json:"message"`
	ParentToolUseID *string         `json:"parent_tool_use_id"`
	IsSynthetic     *bool           `json:"isSynthetic,omitzero"`
	ToolUseResult   json.RawMessage `json:"tool_use_result,omitzero"`
	IsReplay        bool            `json:"isReplay"`
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
	NumTurns          int64                 `json:"num_turns"`
	Result            string                `json:"result"`
	StopReason        *string               `json:"stop_reason"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	Usage             NonNullableUsage      `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []SDKPermissionDenial `json:"permission_denials"`
	StructuredOutput  json.RawMessage       `json:"structured_output,omitzero"`
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
	Type               string  `json:"type"`
	ToolUseID          string  `json:"tool_use_id"`
	ToolName           string  `json:"tool_name"`
	ParentToolUseID    *string `json:"parent_tool_use_id"`
	ElapsedTimeSeconds float64 `json:"elapsed_time_seconds"`
	TaskID             *string `json:"task_id,omitzero"`
	UUID               UUID    `json:"uuid"`
	SessionID          string  `json:"session_id"`
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
	Usage        TaskUsageSummary `json:"usage"`
	LastToolName *string          `json:"last_tool_name,omitzero"`
	UUID         UUID             `json:"uuid"`
	SessionID    string           `json:"session_id"`
}

func (SDKTaskProgressMessage) sdkMessage() {}

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

// BaseHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hook-input
type BaseHookInput struct {
	SessionID      string  `json:"session_id"`
	TranscriptPath string  `json:"transcript_path"`
	Cwd            string  `json:"cwd"`
	PermissionMode *string `json:"permission_mode,omitzero"`
	AgentID        *string `json:"agent_id,omitzero"`
	AgentType      *string `json:"agent_type,omitzero"`
}

// HookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hook-input
type HookInput interface{ hookInput() }

var (
	_ HookInput = (*HookInputUnknown)(nil)
	_ HookInput = (*PreToolUseHookInput)(nil)
	_ HookInput = (*PostToolUseHookInput)(nil)
	_ HookInput = (*PostToolUseFailureHookInput)(nil)
	_ HookInput = (*NotificationHookInput)(nil)
	_ HookInput = (*UserPromptSubmitHookInput)(nil)
	_ HookInput = (*SessionStartHookInput)(nil)
	_ HookInput = (*SessionEndHookInput)(nil)
	_ HookInput = (*StopHookInput)(nil)
	_ HookInput = (*SubagentStartHookInput)(nil)
	_ HookInput = (*SubagentStopHookInput)(nil)
	_ HookInput = (*PreCompactHookInput)(nil)
	_ HookInput = (*PermissionRequestHookInput)(nil)
	_ HookInput = (*SetupHookInput)(nil)
	_ HookInput = (*TeammateIdleHookInput)(nil)
	_ HookInput = (*TaskCompletedHookInput)(nil)
	_ HookInput = (*ConfigChangeHookInput)(nil)
	_ HookInput = (*WorktreeCreateHookInput)(nil)
	_ HookInput = (*WorktreeRemoveHookInput)(nil)
)

// HookInputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hook-input
type HookInputUnknown struct{ UnknownUnion }

func (HookInputUnknown) hookInput() {}

// PreToolUseHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#pretoolusehookinput
type PreToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEvent       `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolUseID     string          `json:"tool_use_id"`
}

func (PreToolUseHookInput) hookInput() {}

// PostToolUseHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#posttoolusehookinput
type PostToolUseHookInput struct {
	BaseHookInput
	HookEventName HookEvent       `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolResponse  json.RawMessage `json:"tool_response"`
	ToolUseID     string          `json:"tool_use_id"`
}

func (PostToolUseHookInput) hookInput() {}

// PostToolUseFailureHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#posttoolusefailurehookinput
type PostToolUseFailureHookInput struct {
	BaseHookInput
	HookEventName HookEvent       `json:"hook_event_name"`
	ToolName      string          `json:"tool_name"`
	ToolInput     json.RawMessage `json:"tool_input"`
	ToolUseID     string          `json:"tool_use_id"`
	Error         string          `json:"error"`
	IsInterrupt   *bool           `json:"is_interrupt,omitzero"`
}

func (PostToolUseFailureHookInput) hookInput() {}

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
}

func (UserPromptSubmitHookInput) hookInput() {}

// SessionStartHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sessionstarthookinput
type SessionStartHookInput struct {
	BaseHookInput
	HookEventName HookEvent          `json:"hook_event_name"`
	Source        SessionStartSource `json:"source"`
	Model         *string            `json:"model,omitzero"`
}

func (SessionStartHookInput) hookInput() {}

// SessionEndHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sessionendhookinput
type SessionEndHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	Reason        string    `json:"reason"`
}

func (SessionEndHookInput) hookInput() {}

// StopHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#stophookinput
type StopHookInput struct {
	BaseHookInput
	HookEventName  HookEvent `json:"hook_event_name"`
	StopHookActive bool      `json:"stop_hook_active"`
}

func (StopHookInput) hookInput() {}

// SubagentStartHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#subagentstarthookinput
type SubagentStartHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	Prompt        string    `json:"prompt"`
}

func (SubagentStartHookInput) hookInput() {}

// SubagentStopHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#subagentstophookinput
type SubagentStopHookInput struct {
	BaseHookInput
	HookEventName HookEvent `json:"hook_event_name"`
	StopReason    string    `json:"stop_reason"`
}

func (SubagentStopHookInput) hookInput() {}

// PreCompactHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#precompacthookinput
type PreCompactHookInput struct {
	BaseHookInput
	HookEventName      HookEvent `json:"hook_event_name"`
	Trigger            string    `json:"trigger"`
	CustomInstructions string    `json:"custom_instructions"`
}

func (PreCompactHookInput) hookInput() {}

// PermissionRequestHookInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#permissionrequesthookinput
type PermissionRequestHookInput struct {
	BaseHookInput
	HookEventName         HookEvent          `json:"hook_event_name"`
	ToolName              string             `json:"tool_name"`
	ToolInput             json.RawMessage    `json:"tool_input"`
	ToolUseID             string             `json:"tool_use_id"`
	PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitzero"`
}

func (PermissionRequestHookInput) hookInput() {}

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
}

func (TeammateIdleHookInput) hookInput() {}

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
	TeamName        *string   `json:"team_name,omitzero"`
}

func (TaskCompletedHookInput) hookInput() {}

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

// HookJSONOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hook-json-output
type HookJSONOutput interface{ hookJSONOutput() }

var (
	_ HookJSONOutput = (*HookJSONOutputUnknown)(nil)
	_ HookJSONOutput = (*AsyncHookJSONOutput)(nil)
	_ HookJSONOutput = (*SyncHookJSONOutput)(nil)
)

// HookJSONOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hook-json-output
type HookJSONOutputUnknown struct{ UnknownUnion }

func (HookJSONOutputUnknown) hookJSONOutput() {}

// AsyncHookJSONOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#async-hook-json-output
type AsyncHookJSONOutput struct {
	Async        bool   `json:"async"`
	AsyncTimeout *int64 `json:"asyncTimeout,omitzero"`
}

func (AsyncHookJSONOutput) hookJSONOutput() {}

// HookSpecificOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutput interface{ hookSpecificOutput() }

var (
	_ HookSpecificOutput = (*HookSpecificOutputUnknown)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputPreToolUse)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputUserPromptSubmit)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputSessionStart)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputSetup)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputSubagentStart)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputPostToolUse)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputPostToolUseFailure)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputNotification)(nil)
	_ HookSpecificOutput = (*HookSpecificOutputPermissionRequest)(nil)
)

// HookSpecificOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputUnknown struct{ UnknownUnion }

func (HookSpecificOutputUnknown) hookSpecificOutput() {}

// HookSpecificOutputPreToolUse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputUserPromptSubmit struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputUserPromptSubmit) hookSpecificOutput() {}

// HookSpecificOutputSessionStart is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputSessionStart struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputSessionStart) hookSpecificOutput() {}

// HookSpecificOutputSetup is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputSetup struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputSetup) hookSpecificOutput() {}

// HookSpecificOutputSubagentStart is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputSubagentStart struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputSubagentStart) hookSpecificOutput() {}

// HookSpecificOutputPostToolUse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputPostToolUse struct {
	HookEventName        HookEvent       `json:"hookEventName"`
	AdditionalContext    *string         `json:"additionalContext,omitzero"`
	UpdatedMCPToolOutput json.RawMessage `json:"updatedMCPToolOutput,omitzero"`
}

func (HookSpecificOutputPostToolUse) hookSpecificOutput() {}

// HookSpecificOutputPostToolUseFailure is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputPostToolUseFailure struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputPostToolUseFailure) hookSpecificOutput() {}

// HookSpecificOutputNotification is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputNotification struct {
	HookEventName     HookEvent `json:"hookEventName"`
	AdditionalContext *string   `json:"additionalContext,omitzero"`
}

func (HookSpecificOutputNotification) hookSpecificOutput() {}

// PermissionRequestDecision is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type PermissionRequestDecision interface{ permissionRequestDecision() }

var (
	_ PermissionRequestDecision = (*PermissionRequestDecisionUnknown)(nil)
	_ PermissionRequestDecision = (*PermissionRequestDecisionAllow)(nil)
	_ PermissionRequestDecision = (*PermissionRequestDecisionDeny)(nil)
)

// PermissionRequestDecisionUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type PermissionRequestDecisionUnknown struct{ UnknownUnion }

func (PermissionRequestDecisionUnknown) permissionRequestDecision() {}

// PermissionRequestDecisionAllow is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type PermissionRequestDecisionAllow struct {
	Behavior           PermissionRequestDecisionBehavior `json:"behavior"`
	UpdatedInput       map[string]any                    `json:"updatedInput,omitzero"`
	UpdatedPermissions []PermissionUpdate                `json:"updatedPermissions,omitzero"`
}

func (PermissionRequestDecisionAllow) permissionRequestDecision() {}

// PermissionRequestDecisionDeny is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type PermissionRequestDecisionDeny struct {
	Behavior  PermissionRequestDecisionBehavior `json:"behavior"`
	Message   *string                           `json:"message,omitzero"`
	Interrupt *bool                             `json:"interrupt,omitzero"`
}

func (PermissionRequestDecisionDeny) permissionRequestDecision() {}

// HookSpecificOutputPermissionRequest is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#hookspecificoutput
type HookSpecificOutputPermissionRequest struct {
	HookEventName HookEvent                 `json:"hookEventName"`
	Decision      PermissionRequestDecision `json:"decision"`
}

func (HookSpecificOutputPermissionRequest) hookSpecificOutput() {}

// SyncHookJSONOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#sync-hook-json-output
type SyncHookJSONOutput struct {
	Continue           *bool              `json:"continue,omitzero"`
	SuppressOutput     *bool              `json:"suppressOutput,omitzero"`
	StopReason         *string            `json:"stopReason,omitzero"`
	Decision           *HookDecision      `json:"decision,omitzero"`
	SystemMessage      *string            `json:"systemMessage,omitzero"`
	Reason             *string            `json:"reason,omitzero"`
	HookSpecificOutput HookSpecificOutput `json:"hookSpecificOutput,omitzero"`
}

func (SyncHookJSONOutput) hookJSONOutput() {}

// ToolInputSchemas is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-input-types
type ToolInputSchemas interface{ toolInputSchemas() }

var (
	_ ToolInputSchemas = (*ToolInputUnknown)(nil)
	_ ToolInputSchemas = (*AgentInput)(nil)
	_ ToolInputSchemas = (*AskUserQuestionInput)(nil)
	_ ToolInputSchemas = (*BashInput)(nil)
	_ ToolInputSchemas = (*TaskOutputInput)(nil)
	_ ToolInputSchemas = (*ConfigInput)(nil)
	_ ToolInputSchemas = (*EnterWorktreeInput)(nil)
	_ ToolInputSchemas = (*ExitPlanModeInput)(nil)
	_ ToolInputSchemas = (*FileEditInput)(nil)
	_ ToolInputSchemas = (*FileReadInput)(nil)
	_ ToolInputSchemas = (*FileWriteInput)(nil)
	_ ToolInputSchemas = (*GlobInput)(nil)
	_ ToolInputSchemas = (*GrepInput)(nil)
	_ ToolInputSchemas = (*ListMcpResourcesInput)(nil)
	_ ToolInputSchemas = (*McpInput)(nil)
	_ ToolInputSchemas = (*NotebookEditInput)(nil)
	_ ToolInputSchemas = (*ReadMcpResourceInput)(nil)
	_ ToolInputSchemas = (*SubscribeMcpResourceInput)(nil)
	_ ToolInputSchemas = (*SubscribePollingInput)(nil)
	_ ToolInputSchemas = (*TaskStopInput)(nil)
	_ ToolInputSchemas = (*TodoWriteInput)(nil)
	_ ToolInputSchemas = (*UnsubscribeMcpResourceInput)(nil)
	_ ToolInputSchemas = (*UnsubscribePollingInput)(nil)
	_ ToolInputSchemas = (*WebFetchInput)(nil)
	_ ToolInputSchemas = (*WebSearchInput)(nil)
)

// ToolInputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-input-types
type ToolInputUnknown struct{ UnknownUnion }

func (ToolInputUnknown) toolInputSchemas() {}

// AgentInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task
type AgentInput struct {
	Description     string          `json:"description"`
	Prompt          string          `json:"prompt"`
	SubagentType    string          `json:"subagent_type"`
	Model           *AgentModel     `json:"model,omitzero"`
	Resume          *string         `json:"resume,omitzero"`
	RunInBackground *bool           `json:"run_in_background,omitzero"`
	MaxTurns        *int64          `json:"max_turns,omitzero"`
	Name            *string         `json:"name,omitzero"`
	TeamName        *string         `json:"team_name,omitzero"`
	Mode            *AgentMode      `json:"mode,omitzero"`
	Isolation       *AgentIsolation `json:"isolation,omitzero"`
}

func (AgentInput) toolInputSchemas() {}

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

// AskUserQuestionInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion
type AskUserQuestionInput struct {
	Questions []AskUserQuestionInputQuestion `json:"questions"`
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskoutput
type TaskOutputInput struct {
	TaskID  string `json:"task_id"`
	Block   bool   `json:"block"`
	Timeout int64  `json:"timeout"`
}

func (TaskOutputInput) toolInputSchemas() {}

// ConfigInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#config
type ConfigInput struct {
	Setting string          `json:"setting"`
	Value   json.RawMessage `json:"value,omitzero"`
}

func (ConfigInput) toolInputSchemas() {}

// EnterWorktreeInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#enterworktree
type EnterWorktreeInput struct {
	Name *string `json:"name,omitzero"`
}

func (EnterWorktreeInput) toolInputSchemas() {}

// AllowedPrompt is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitplanmode
type AllowedPrompt struct {
	Tool   string `json:"tool"`
	Prompt string `json:"prompt"`
}

// ExitPlanModeInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitplanmode
type ExitPlanModeInput struct {
	AllowedPrompts []AllowedPrompt `json:"allowedPrompts,omitzero"`
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-input-types
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

// SubscribeMcpResourceInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-input-types
type SubscribeMcpResourceInput map[string]any

func (SubscribeMcpResourceInput) toolInputSchemas() {}

// SubscribePollingInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-input-types
type SubscribePollingInput map[string]any

func (SubscribePollingInput) toolInputSchemas() {}

// TaskStopInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#taskstop
type TaskStopInput struct {
	TaskID  *string `json:"task_id,omitzero"`
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
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#todowrite
type TodoWriteInput struct {
	Todos []TodoItem `json:"todos"`
}

func (TodoWriteInput) toolInputSchemas() {}

// UnsubscribeMcpResourceInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-input-types
type UnsubscribeMcpResourceInput map[string]any

func (UnsubscribeMcpResourceInput) toolInputSchemas() {}

// UnsubscribePollingInput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-input-types
type UnsubscribePollingInput map[string]any

func (UnsubscribePollingInput) toolInputSchemas() {}

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

// ToolOutputSchemas is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-output-schemas
type ToolOutputSchemas interface{ toolOutputSchemas() }

var (
	_ ToolOutputSchemas = (*ToolOutputUnknown)(nil)
	_ ToolOutputSchemas = (*AgentOutputUnknown)(nil)
	_ ToolOutputSchemas = (*AgentOutputCompleted)(nil)
	_ ToolOutputSchemas = (*AgentOutputAsyncLaunched)(nil)
	_ ToolOutputSchemas = (*AgentOutputSubAgentEntered)(nil)
	_ ToolOutputSchemas = (*AskUserQuestionOutput)(nil)
	_ ToolOutputSchemas = (*BashOutput)(nil)
	_ ToolOutputSchemas = (*FileEditOutput)(nil)
	_ ToolOutputSchemas = (*FileReadOutputUnknown)(nil)
	_ ToolOutputSchemas = (*FileReadOutputText)(nil)
	_ ToolOutputSchemas = (*FileReadOutputImage)(nil)
	_ ToolOutputSchemas = (*FileReadOutputNotebook)(nil)
	_ ToolOutputSchemas = (*FileReadOutputPdf)(nil)
	_ ToolOutputSchemas = (*FileReadOutputParts)(nil)
	_ ToolOutputSchemas = (*FileWriteOutput)(nil)
	_ ToolOutputSchemas = (*GlobOutput)(nil)
	_ ToolOutputSchemas = (*GrepOutput)(nil)
	_ ToolOutputSchemas = (*TaskStopOutput)(nil)
	_ ToolOutputSchemas = (*NotebookEditOutput)(nil)
	_ ToolOutputSchemas = (*WebFetchOutput)(nil)
	_ ToolOutputSchemas = (*WebSearchOutput)(nil)
	_ ToolOutputSchemas = (*TodoWriteOutput)(nil)
	_ ToolOutputSchemas = (*ExitPlanModeOutput)(nil)
	_ ToolOutputSchemas = (*ListMcpResourcesOutput)(nil)
	_ ToolOutputSchemas = (*ReadMcpResourceOutput)(nil)
	_ ToolOutputSchemas = (*ConfigOutput)(nil)
	_ ToolOutputSchemas = (*EnterWorktreeOutput)(nil)
)

// ToolOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#tool-output-schemas
type ToolOutputUnknown struct{ UnknownUnion }

func (ToolOutputUnknown) toolOutputSchemas() {}

// AgentOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutput interface {
	toolOutputSchemas()
	agentOutput()
}

var (
	_ AgentOutput = (*AgentOutputUnknown)(nil)
	_ AgentOutput = (*AgentOutputCompleted)(nil)
	_ AgentOutput = (*AgentOutputAsyncLaunched)(nil)
	_ AgentOutput = (*AgentOutputSubAgentEntered)(nil)
)

// AgentOutputUnknown is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputUnknown struct{ UnknownUnion }

func (AgentOutputUnknown) toolOutputSchemas() {}
func (AgentOutputUnknown) agentOutput()       {}

// AgentOutputContentBlock is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AgentOutputServerToolUse is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputServerToolUse struct {
	WebSearchRequests int64 `json:"web_search_requests"`
	WebFetchRequests  int64 `json:"web_fetch_requests"`
}

// AgentOutputCacheCreation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputCacheCreation struct {
	Ephemeral1HInputTokens int64 `json:"ephemeral_1h_input_tokens"`
	Ephemeral5MInputTokens int64 `json:"ephemeral_5m_input_tokens"`
}

// AgentOutputUsage is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputUsage struct {
	InputTokens              int64                     `json:"input_tokens"`
	OutputTokens             int64                     `json:"output_tokens"`
	CacheCreationInputTokens *int64                    `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64                    `json:"cache_read_input_tokens"`
	ServerToolUse            *AgentOutputServerToolUse `json:"server_tool_use"`
	ServiceTier              *ServiceTier              `json:"service_tier"`
	CacheCreation            *AgentOutputCacheCreation `json:"cache_creation"`
}

// AgentOutputCompleted is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputCompleted struct {
	Status            string                    `json:"status"`
	AgentID           string                    `json:"agentId"`
	Content           []AgentOutputContentBlock `json:"content"`
	TotalToolUseCount int64                     `json:"totalToolUseCount"`
	TotalDurationMs   int64                     `json:"totalDurationMs"`
	TotalTokens       int64                     `json:"totalTokens"`
	Usage             AgentOutputUsage          `json:"usage"`
	Prompt            string                    `json:"prompt"`
}

func (AgentOutputCompleted) toolOutputSchemas() {}
func (AgentOutputCompleted) agentOutput()       {}

// AgentOutputAsyncLaunched is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputAsyncLaunched struct {
	Status            string `json:"status"`
	AgentID           string `json:"agentId"`
	Description       string `json:"description"`
	Prompt            string `json:"prompt"`
	OutputFile        string `json:"outputFile"`
	CanReadOutputFile *bool  `json:"canReadOutputFile,omitzero"`
}

func (AgentOutputAsyncLaunched) toolOutputSchemas() {}
func (AgentOutputAsyncLaunched) agentOutput()       {}

// AgentOutputSubAgentEntered is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#task-2
type AgentOutputSubAgentEntered struct {
	Status      string `json:"status"`
	Description string `json:"description"`
	Message     string `json:"message"`
}

func (AgentOutputSubAgentEntered) toolOutputSchemas() {}
func (AgentOutputSubAgentEntered) agentOutput()       {}

// AskUserQuestionOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#askuserquestion-2
type AskUserQuestionOutput struct {
	Questions []AskUserQuestionInputQuestion `json:"questions"`
	Answers   map[string]string              `json:"answers"`
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
	BackgroundTaskID          *string           `json:"backgroundTaskId,omitzero"`
	BackgroundedByUser        *bool             `json:"backgroundedByUser,omitzero"`
	DangerouslyDisableSandbox *bool             `json:"dangerouslyDisableSandbox,omitzero"`
	ReturnCodeInterpretation  *string           `json:"returnCodeInterpretation,omitzero"`
	StructuredContent         []json.RawMessage `json:"structuredContent,omitzero"`
	PersistedOutputPath       *string           `json:"persistedOutputPath,omitzero"`
	PersistedOutputSize       *int64            `json:"persistedOutputSize,omitzero"`
}

func (BashOutput) toolOutputSchemas() {}

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

// GitDiff is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#edit-2
type GitDiff struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int64  `json:"additions"`
	Deletions int64  `json:"deletions"`
	Changes   int64  `json:"changes"`
	Patch     string `json:"patch"`
}

// FileEditOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#edit-2
type FileEditOutput struct {
	FilePath        string                `json:"filePath"`
	OldString       string                `json:"oldString"`
	NewString       string                `json:"newString"`
	OriginalFile    string                `json:"originalFile"`
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

// FileWriteOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#write-2
type FileWriteOutput struct {
	Type            string                `json:"type"`
	FilePath        string                `json:"filePath"`
	Content         string                `json:"content"`
	StructuredPatch []StructuredPatchHunk `json:"structuredPatch"`
	OriginalFile    *string               `json:"originalFile"`
	GitDiff         *GitDiff              `json:"gitDiff,omitzero"`
}

func (FileWriteOutput) toolOutputSchemas() {}

// GlobOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#glob-2
type GlobOutput struct {
	DurationMs int64    `json:"durationMs"`
	NumFiles   int64    `json:"numFiles"`
	Filenames  []string `json:"filenames"`
	Truncated  bool     `json:"truncated"`
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

// WebFetchOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#webfetch-2
type WebFetchOutput struct {
	Bytes      int64  `json:"bytes"`
	Code       int64  `json:"code"`
	CodeText   string `json:"codeText"`
	Result     string `json:"result"`
	DurationMs int64  `json:"durationMs"`
	URL        string `json:"url"`
}

func (WebFetchOutput) toolOutputSchemas() {}

// WebSearchOutputResultEntry is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutputResultEntry struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#websearch-2
type WebSearchOutput struct {
	Query   string                       `json:"query"`
	Results []WebSearchOutputResultEntry `json:"results"`
}

func (WebSearchOutput) toolOutputSchemas() {}

// TodoWriteOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#todowrite-2
type TodoWriteOutput struct {
	OldTodos string `json:"oldTodos"`
	NewTodos string `json:"newTodos"`
}

func (TodoWriteOutput) toolOutputSchemas() {}

// ExitPlanModeOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#exitplanmode-2
type ExitPlanModeOutput struct {
	ApprovedPrompts []string `json:"approvedPrompts"`
}

func (ExitPlanModeOutput) toolOutputSchemas() {}

// McpResource is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#listmcpresources-2
type McpResource struct {
	URI         string  `json:"uri"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitzero"`
	MimeType    *string `json:"mimeType,omitzero"`
}

// ListMcpResourcesOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#listmcpresources-2
type ListMcpResourcesOutput struct {
	Resources []McpResource `json:"resources"`
}

func (ListMcpResourcesOutput) toolOutputSchemas() {}

// ReadMcpResourceOutputContent is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresource-2
type ReadMcpResourceOutputContent struct {
	URI      string  `json:"uri"`
	MimeType *string `json:"mimeType,omitzero"`
	Text     *string `json:"text,omitzero"`
}

// ReadMcpResourceOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#readmcpresource-2
type ReadMcpResourceOutput struct {
	Contents []ReadMcpResourceOutputContent `json:"contents"`
}

func (ReadMcpResourceOutput) toolOutputSchemas() {}

// ConfigOutputOperation is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#message-types
type ConfigOutputOperation string

const (
	ConfigOutputOperationGet ConfigOutputOperation = "get"
	ConfigOutputOperationSet ConfigOutputOperation = "set"
)

// ConfigOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#config-2
type ConfigOutput struct {
	Success       bool                   `json:"success"`
	Operation     *ConfigOutputOperation `json:"operation,omitzero"`
	Setting       *string                `json:"setting,omitzero"`
	Value         json.RawMessage        `json:"value,omitzero"`
	PreviousValue json.RawMessage        `json:"previousValue,omitzero"`
	NewValue      json.RawMessage        `json:"newValue,omitzero"`
	Error         *string                `json:"error,omitzero"`
}

func (ConfigOutput) toolOutputSchemas() {}

// EnterWorktreeOutput is a handwritten Claude Agent SDK type.
//
// Source: https://code.claude.com/docs/en/agent-sdk/typescript#enterworktree-2
type EnterWorktreeOutput struct {
	WorktreePath   string  `json:"worktreePath"`
	WorktreeBranch *string `json:"worktreeBranch,omitzero"`
	Message        string  `json:"message"`
}

func (EnterWorktreeOutput) toolOutputSchemas() {}

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

func opt[T comparable](v T) *T {
	return &v
}

func protoOpt[T comparable](msg interface{ ProtoReflect() protoreflect.Message }, fieldName protoreflect.Name, v T) *T {
	field := msg.ProtoReflect().Descriptor().Fields().ByName(fieldName)
	if field == nil || !msg.ProtoReflect().Has(field) {
		return nil
	}
	return opt(v)
}

func rawToolInputToProto(raw json.RawMessage, discriminator string) *pb.ToolInput {
	if len(raw) == 0 {
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

func rawToolOutputToProto(raw json.RawMessage, discriminator string) *pb.ToolOutput {
	if len(raw) == 0 {
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

func rawFromToolInputProto(v *pb.ToolInput) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	if unknown := v.GetUnknown(); unknown != nil {
		return cloneRawMessage(unknown.GetRawJson()), nil
	}
	b, err := protojson.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func rawFromToolOutputProto(v *pb.ToolOutput) (json.RawMessage, error) {
	if v == nil {
		return nil, nil
	}
	if unknown := v.GetUnknown(); unknown != nil {
		return cloneRawMessage(unknown.GetRawJson()), nil
	}
	b, err := protojson.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
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
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
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

// UnmarshalPermissionUpdate unmarshals a PermissionUpdate union value from JSON.
func UnmarshalPermissionUpdate(data []byte) (PermissionUpdate, error) {
	var disc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return nil, err
	}
	switch disc.Type {
	case "addRules":
		var v PermissionUpdateAddRules
		return &v, v.UnmarshalJSON(data)
	case "replaceRules":
		var v PermissionUpdateReplaceRules
		return &v, v.UnmarshalJSON(data)
	case "removeRules":
		var v PermissionUpdateRemoveRules
		return &v, v.UnmarshalJSON(data)
	case "setMode":
		var v PermissionUpdateSetMode
		return &v, v.UnmarshalJSON(data)
	case "addDirectories":
		var v PermissionUpdateAddDirectories
		return &v, v.UnmarshalJSON(data)
	case "removeDirectories":
		var v PermissionUpdateRemoveDirectories
		return &v, v.UnmarshalJSON(data)
	default:
		var v PermissionUpdateUnknown
		return &v, v.UnmarshalJSON(data)
	}
}

func PermissionUpdateToProto(v PermissionUpdate) (*pb.PermissionUpdate, error) {
	switch x := v.(type) {
	case *PermissionUpdateUnknown:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_Unknown{Unknown: x.ToProto()}}, nil
	case *PermissionUpdateAddRules:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_AddRules{AddRules: &pb.PermissionUpdateAddRules{
			Type:        "addRules",
			Rules:       permissionRuleValuesToProto(x.Rules),
			Behavior:    string(x.Behavior),
			Destination: string(x.Destination),
		}}}, nil
	case *PermissionUpdateReplaceRules:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_ReplaceRules{ReplaceRules: &pb.PermissionUpdateReplaceRules{
			Type:        "replaceRules",
			Rules:       permissionRuleValuesToProto(x.Rules),
			Behavior:    string(x.Behavior),
			Destination: string(x.Destination),
		}}}, nil
	case *PermissionUpdateRemoveRules:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_RemoveRules{RemoveRules: &pb.PermissionUpdateRemoveRules{
			Type:        "removeRules",
			Rules:       permissionRuleValuesToProto(x.Rules),
			Behavior:    string(x.Behavior),
			Destination: string(x.Destination),
		}}}, nil
	case *PermissionUpdateSetMode:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_SetMode{SetMode: &pb.PermissionUpdateSetMode{
			Type:        "setMode",
			Mode:        string(x.Mode),
			Destination: string(x.Destination),
		}}}, nil
	case *PermissionUpdateAddDirectories:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_AddDirectories{AddDirectories: &pb.PermissionUpdateAddDirectories{
			Type:        "addDirectories",
			Directories: append([]string(nil), x.Directories...),
			Destination: string(x.Destination),
		}}}, nil
	case *PermissionUpdateRemoveDirectories:
		return &pb.PermissionUpdate{Value: &pb.PermissionUpdate_RemoveDirectories{RemoveDirectories: &pb.PermissionUpdateRemoveDirectories{
			Type:        "removeDirectories",
			Directories: append([]string(nil), x.Directories...),
			Destination: string(x.Destination),
		}}}, nil
	default:
		return nil, fmt.Errorf("unsupported PermissionUpdate %T", v)
	}
}

func PermissionUpdateFromProto(v *pb.PermissionUpdate) (PermissionUpdate, error) {
	if v == nil {
		return nil, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.PermissionUpdate_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return &PermissionUpdateUnknown{UnknownUnion: u}, nil
	case *pb.PermissionUpdate_AddRules:
		addRules := v.GetAddRules()
		return &PermissionUpdateAddRules{
			Type:        "addRules",
			Rules:       permissionRuleValuesFromProto(addRules.GetRules()),
			Behavior:    PermissionBehavior(addRules.GetBehavior()),
			Destination: PermissionUpdateDestination(addRules.GetDestination()),
		}, nil
	case *pb.PermissionUpdate_ReplaceRules:
		replaceRules := v.GetReplaceRules()
		return &PermissionUpdateReplaceRules{
			Type:        "replaceRules",
			Rules:       permissionRuleValuesFromProto(replaceRules.GetRules()),
			Behavior:    PermissionBehavior(replaceRules.GetBehavior()),
			Destination: PermissionUpdateDestination(replaceRules.GetDestination()),
		}, nil
	case *pb.PermissionUpdate_RemoveRules:
		removeRules := v.GetRemoveRules()
		return &PermissionUpdateRemoveRules{
			Type:        "removeRules",
			Rules:       permissionRuleValuesFromProto(removeRules.GetRules()),
			Behavior:    PermissionBehavior(removeRules.GetBehavior()),
			Destination: PermissionUpdateDestination(removeRules.GetDestination()),
		}, nil
	case *pb.PermissionUpdate_SetMode:
		setMode := v.GetSetMode()
		return &PermissionUpdateSetMode{
			Type:        "setMode",
			Mode:        PermissionMode(setMode.GetMode()),
			Destination: PermissionUpdateDestination(setMode.GetDestination()),
		}, nil
	case *pb.PermissionUpdate_AddDirectories:
		addDirectories := v.GetAddDirectories()
		return &PermissionUpdateAddDirectories{
			Type:        "addDirectories",
			Directories: append([]string(nil), addDirectories.GetDirectories()...),
			Destination: PermissionUpdateDestination(addDirectories.GetDestination()),
		}, nil
	case *pb.PermissionUpdate_RemoveDirectories:
		removeDirectories := v.GetRemoveDirectories()
		return &PermissionUpdateRemoveDirectories{
			Type:        "removeDirectories",
			Directories: append([]string(nil), removeDirectories.GetDirectories()...),
			Destination: PermissionUpdateDestination(removeDirectories.GetDestination()),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proto PermissionUpdate %T", x)
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
	type raw struct {
		Behavior           PermissionResultBehavior `json:"behavior"`
		UpdatedInput       map[string]any           `json:"updatedInput,omitzero"`
		UpdatedPermissions []json.RawMessage        `json:"updatedPermissions,omitzero"`
		ToolUseID          *string                  `json:"toolUseID,omitzero"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	updates := make([]PermissionUpdate, 0, len(v.UpdatedPermissions))
	for _, rawUpdate := range v.UpdatedPermissions {
		update, err := UnmarshalPermissionUpdate(rawUpdate)
		if err != nil {
			return err
		}
		updates = append(updates, update)
	}
	*o = PermissionResultAllow{
		Behavior:           PermissionResultBehaviorAllow,
		UpdatedInput:       v.UpdatedInput,
		UpdatedPermissions: updates,
		ToolUseID:          v.ToolUseID,
	}
	o.Behavior = PermissionResultBehaviorAllow
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
	switch x := v.(type) {
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
		return &pb.PermissionResult{Value: &pb.PermissionResult_Allow{Allow: &pb.PermissionResultAllow{
			Behavior:           permissionResultBehaviorToProto(PermissionResultBehaviorAllow),
			UpdatedInput:       updatedInput,
			UpdatedPermissions: updatedPermissions,
			ToolUseId:          x.ToolUseID,
		}}}, nil
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
		return nil, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.PermissionResult_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return &PermissionResultUnknown{UnknownUnion: u}, nil
	case *pb.PermissionResult_Allow:
		allow := v.GetAllow()
		updates, err := permissionUpdatesFromProto(allow.GetUpdatedPermissions())
		if err != nil {
			return nil, err
		}
		return &PermissionResultAllow{
			Behavior:           permissionResultBehaviorFromProto(allow.GetBehavior()),
			UpdatedInput:       protoStructToMap(allow.GetUpdatedInput()),
			UpdatedPermissions: updates,
			ToolUseID:          protoOpt(allow, "tool_use_id", allow.GetToolUseId()),
		}, nil
	case *pb.PermissionResult_Deny:
		deny := v.GetDeny()
		return &PermissionResultDeny{
			Behavior:  permissionResultBehaviorFromProto(deny.GetBehavior()),
			Message:   deny.GetMessage(),
			Interrupt: protoOpt(deny, "interrupt", deny.GetInterrupt()),
			ToolUseID: protoOpt(deny, "tool_use_id", deny.GetToolUseId()),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proto PermissionResult %T", x)
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
	type raw struct {
		Behavior           PermissionRequestDecisionBehavior `json:"behavior"`
		UpdatedInput       map[string]any                    `json:"updatedInput,omitzero"`
		UpdatedPermissions []json.RawMessage                 `json:"updatedPermissions,omitzero"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	updates := make([]PermissionUpdate, 0, len(v.UpdatedPermissions))
	for _, rawUpdate := range v.UpdatedPermissions {
		update, err := UnmarshalPermissionUpdate(rawUpdate)
		if err != nil {
			return err
		}
		updates = append(updates, update)
	}
	*o = PermissionRequestDecisionAllow{
		Behavior:           PermissionRequestDecisionBehaviorAllow,
		UpdatedInput:       v.UpdatedInput,
		UpdatedPermissions: updates,
	}
	o.Behavior = PermissionRequestDecisionBehaviorAllow
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

// UnmarshalPermissionRequestDecision unmarshals a PermissionRequestDecision union value from JSON.
func UnmarshalPermissionRequestDecision(data []byte) (PermissionRequestDecision, error) {
	var disc struct {
		Behavior PermissionRequestDecisionBehavior `json:"behavior"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return nil, err
	}
	switch disc.Behavior {
	case PermissionRequestDecisionBehaviorAllow:
		var v PermissionRequestDecisionAllow
		return &v, v.UnmarshalJSON(data)
	case PermissionRequestDecisionBehaviorDeny:
		var v PermissionRequestDecisionDeny
		return &v, v.UnmarshalJSON(data)
	default:
		var v PermissionRequestDecisionUnknown
		return &v, v.UnmarshalJSON(data)
	}
}

func PermissionRequestDecisionToProto(v PermissionRequestDecision) (*pb.PermissionRequestDecision, error) {
	switch x := v.(type) {
	case *PermissionRequestDecisionUnknown:
		return &pb.PermissionRequestDecision{Value: &pb.PermissionRequestDecision_Unknown{Unknown: x.ToProto()}}, nil
	case *PermissionRequestDecisionAllow:
		updatedInput, err := mapToProtoStruct(x.UpdatedInput)
		if err != nil {
			return nil, err
		}
		updatedPermissions, err := permissionUpdatesToProto(x.UpdatedPermissions)
		if err != nil {
			return nil, err
		}
		return &pb.PermissionRequestDecision{Value: &pb.PermissionRequestDecision_Allow{Allow: &pb.PermissionRequestDecisionAllow{
			Behavior:           permissionRequestDecisionBehaviorToProto(PermissionRequestDecisionBehaviorAllow),
			UpdatedInput:       updatedInput,
			UpdatedPermissions: updatedPermissions,
		}}}, nil
	case *PermissionRequestDecisionDeny:
		return &pb.PermissionRequestDecision{Value: &pb.PermissionRequestDecision_Deny{Deny: &pb.PermissionRequestDecisionDeny{
			Behavior:  permissionRequestDecisionBehaviorToProto(PermissionRequestDecisionBehaviorDeny),
			Message:   x.Message,
			Interrupt: x.Interrupt,
		}}}, nil
	default:
		return nil, fmt.Errorf("unsupported PermissionRequestDecision %T", v)
	}
}

func PermissionRequestDecisionFromProto(v *pb.PermissionRequestDecision) (PermissionRequestDecision, error) {
	if v == nil {
		return nil, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.PermissionRequestDecision_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return &PermissionRequestDecisionUnknown{UnknownUnion: u}, nil
	case *pb.PermissionRequestDecision_Allow:
		allow := v.GetAllow()
		updates, err := permissionUpdatesFromProto(allow.GetUpdatedPermissions())
		if err != nil {
			return nil, err
		}
		return &PermissionRequestDecisionAllow{
			Behavior:           permissionRequestDecisionBehaviorFromProto(allow.GetBehavior()),
			UpdatedInput:       protoStructToMap(allow.GetUpdatedInput()),
			UpdatedPermissions: updates,
		}, nil
	case *pb.PermissionRequestDecision_Deny:
		deny := v.GetDeny()
		return &PermissionRequestDecisionDeny{
			Behavior:  permissionRequestDecisionBehaviorFromProto(deny.GetBehavior()),
			Message:   protoOpt(deny, "message", deny.GetMessage()),
			Interrupt: protoOpt(deny, "interrupt", deny.GetInterrupt()),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proto PermissionRequestDecision %T", x)
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

func (o HookSpecificOutputPermissionRequest) MarshalJSON() ([]byte, error) {
	type raw struct {
		HookEventName HookEvent       `json:"hookEventName"`
		Decision      json.RawMessage `json:"decision"`
	}
	var dec json.RawMessage
	if o.Decision != nil {
		b, err := marshalPermissionRequestDecision(o.Decision)
		if err != nil {
			return nil, err
		}
		dec = b
	}
	return json.Marshal(raw{HookEventName: HookEventPermissionRequest, Decision: dec})
}

func (o *HookSpecificOutputPermissionRequest) UnmarshalJSON(data []byte) error {
	type raw struct {
		HookEventName HookEvent       `json:"hookEventName"`
		Decision      json.RawMessage `json:"decision"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.HookEventName = HookEventPermissionRequest
	if len(v.Decision) > 0 {
		dec, err := UnmarshalPermissionRequestDecision(v.Decision)
		if err != nil {
			return err
		}
		o.Decision = dec
	}
	return nil
}

func marshalPermissionRequestDecision(v PermissionRequestDecision) ([]byte, error) {
	switch x := v.(type) {
	case *PermissionRequestDecisionUnknown:
		return x.MarshalJSON()
	case *PermissionRequestDecisionAllow:
		return x.MarshalJSON()
	case *PermissionRequestDecisionDeny:
		return x.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported PermissionRequestDecision %T", v)
	}
}

// UnmarshalHookSpecificOutput unmarshals a HookSpecificOutput union value from JSON.
func UnmarshalHookSpecificOutput(data []byte) (HookSpecificOutput, error) {
	var disc struct {
		HookEventName HookEvent `json:"hookEventName"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return nil, err
	}
	switch disc.HookEventName {
	case HookEventPreToolUse:
		var v HookSpecificOutputPreToolUse
		return &v, v.UnmarshalJSON(data)
	case HookEventUserPromptSubmit:
		var v HookSpecificOutputUserPromptSubmit
		return &v, v.UnmarshalJSON(data)
	case HookEventSessionStart:
		var v HookSpecificOutputSessionStart
		return &v, v.UnmarshalJSON(data)
	case HookEventSetup:
		var v HookSpecificOutputSetup
		return &v, v.UnmarshalJSON(data)
	case HookEventSubagentStart:
		var v HookSpecificOutputSubagentStart
		return &v, v.UnmarshalJSON(data)
	case HookEventPostToolUse:
		var v HookSpecificOutputPostToolUse
		return &v, v.UnmarshalJSON(data)
	case HookEventPostToolUseFailure:
		var v HookSpecificOutputPostToolUseFailure
		return &v, v.UnmarshalJSON(data)
	case HookEventNotification:
		var v HookSpecificOutputNotification
		return &v, v.UnmarshalJSON(data)
	case HookEventPermissionRequest:
		var v HookSpecificOutputPermissionRequest
		return &v, v.UnmarshalJSON(data)
	default:
		var v HookSpecificOutputUnknown
		return &v, v.UnmarshalJSON(data)
	}
}

func HookSpecificOutputToProto(v HookSpecificOutput) (*pb.HookSpecificOutput, error) {
	switch x := v.(type) {
	case *HookSpecificOutputUnknown:
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_Unknown{Unknown: x.ToProto()}}, nil
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
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_PreToolUse{PreToolUse: &pb.HookSpecificOutputPreToolUse{
			HookEventName:            string(HookEventPreToolUse),
			PermissionDecision:       permissionDecision,
			PermissionDecisionReason: x.PermissionDecisionReason,
			UpdatedInput:             updatedInput,
			AdditionalContext:        x.AdditionalContext,
		}}}, nil
	case *HookSpecificOutputUserPromptSubmit:
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_UserPromptSubmit{UserPromptSubmit: &pb.HookSpecificOutputUserPromptSubmit{
			HookEventName: string(HookEventUserPromptSubmit), AdditionalContext: x.AdditionalContext,
		}}}, nil
	case *HookSpecificOutputSessionStart:
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_SessionStart{SessionStart: &pb.HookSpecificOutputSessionStart{
			HookEventName: string(HookEventSessionStart), AdditionalContext: x.AdditionalContext,
		}}}, nil
	case *HookSpecificOutputSetup:
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_Setup{Setup: &pb.HookSpecificOutputSetup{
			HookEventName: string(HookEventSetup), AdditionalContext: x.AdditionalContext,
		}}}, nil
	case *HookSpecificOutputSubagentStart:
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_SubagentStart{SubagentStart: &pb.HookSpecificOutputSubagentStart{
			HookEventName: string(HookEventSubagentStart), AdditionalContext: x.AdditionalContext,
		}}}, nil
	case *HookSpecificOutputPostToolUse:
		var updated *pb.JsonRawMessage
		if len(x.UpdatedMCPToolOutput) > 0 {
			updated = &pb.JsonRawMessage{RawJson: cloneRawMessage(x.UpdatedMCPToolOutput)}
		}
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_PostToolUse{PostToolUse: &pb.HookSpecificOutputPostToolUse{
			HookEventName: string(HookEventPostToolUse), AdditionalContext: x.AdditionalContext, UpdatedMcpToolOutput: updated,
		}}}, nil
	case *HookSpecificOutputPostToolUseFailure:
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_PostToolUseFailure{PostToolUseFailure: &pb.HookSpecificOutputPostToolUseFailure{
			HookEventName: string(HookEventPostToolUseFailure), AdditionalContext: x.AdditionalContext,
		}}}, nil
	case *HookSpecificOutputNotification:
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_Notification{Notification: &pb.HookSpecificOutputNotification{
			HookEventName: string(HookEventNotification), AdditionalContext: x.AdditionalContext,
		}}}, nil
	case *HookSpecificOutputPermissionRequest:
		dec, err := PermissionRequestDecisionToProto(x.Decision)
		if err != nil {
			return nil, err
		}
		return &pb.HookSpecificOutput{Value: &pb.HookSpecificOutput_PermissionRequest{PermissionRequest: &pb.HookSpecificOutputPermissionRequest{
			HookEventName: string(HookEventPermissionRequest), Decision: dec,
		}}}, nil
	default:
		return nil, fmt.Errorf("unsupported HookSpecificOutput %T", v)
	}
}

func HookSpecificOutputFromProto(v *pb.HookSpecificOutput) (HookSpecificOutput, error) {
	if v == nil {
		return nil, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.HookSpecificOutput_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return &HookSpecificOutputUnknown{UnknownUnion: u}, nil
	case *pb.HookSpecificOutput_PreToolUse:
		preToolUse := v.GetPreToolUse()
		var pd *HookPermissionDecision
		permissionDecision := preToolUse.GetPermissionDecision()
		if permissionDecision != "" {
			t := HookPermissionDecision(permissionDecision)
			pd = &t
		}
		return &HookSpecificOutputPreToolUse{
			HookEventName:            HookEventPreToolUse,
			PermissionDecision:       pd,
			PermissionDecisionReason: protoOpt(preToolUse, "permission_decision_reason", preToolUse.GetPermissionDecisionReason()),
			UpdatedInput:             protoStructToMap(preToolUse.GetUpdatedInput()),
			AdditionalContext:        protoOpt(preToolUse, "additional_context", preToolUse.GetAdditionalContext()),
		}, nil
	case *pb.HookSpecificOutput_UserPromptSubmit:
		userPromptSubmit := v.GetUserPromptSubmit()
		return &HookSpecificOutputUserPromptSubmit{HookEventName: HookEventUserPromptSubmit, AdditionalContext: protoOpt(userPromptSubmit, "additional_context", userPromptSubmit.GetAdditionalContext())}, nil
	case *pb.HookSpecificOutput_SessionStart:
		sessionStart := v.GetSessionStart()
		return &HookSpecificOutputSessionStart{HookEventName: HookEventSessionStart, AdditionalContext: protoOpt(sessionStart, "additional_context", sessionStart.GetAdditionalContext())}, nil
	case *pb.HookSpecificOutput_Setup:
		setup := v.GetSetup()
		return &HookSpecificOutputSetup{HookEventName: HookEventSetup, AdditionalContext: protoOpt(setup, "additional_context", setup.GetAdditionalContext())}, nil
	case *pb.HookSpecificOutput_SubagentStart:
		subagentStart := v.GetSubagentStart()
		return &HookSpecificOutputSubagentStart{HookEventName: HookEventSubagentStart, AdditionalContext: protoOpt(subagentStart, "additional_context", subagentStart.GetAdditionalContext())}, nil
	case *pb.HookSpecificOutput_PostToolUse:
		postToolUse := v.GetPostToolUse()
		var updated json.RawMessage
		if postToolUse.GetUpdatedMcpToolOutput() != nil {
			updated = cloneRawMessage(postToolUse.GetUpdatedMcpToolOutput().GetRawJson())
		}
		return &HookSpecificOutputPostToolUse{HookEventName: HookEventPostToolUse, AdditionalContext: protoOpt(postToolUse, "additional_context", postToolUse.GetAdditionalContext()), UpdatedMCPToolOutput: updated}, nil
	case *pb.HookSpecificOutput_PostToolUseFailure:
		postToolUseFailure := v.GetPostToolUseFailure()
		return &HookSpecificOutputPostToolUseFailure{HookEventName: HookEventPostToolUseFailure, AdditionalContext: protoOpt(postToolUseFailure, "additional_context", postToolUseFailure.GetAdditionalContext())}, nil
	case *pb.HookSpecificOutput_Notification:
		notification := v.GetNotification()
		return &HookSpecificOutputNotification{HookEventName: HookEventNotification, AdditionalContext: protoOpt(notification, "additional_context", notification.GetAdditionalContext())}, nil
	case *pb.HookSpecificOutput_PermissionRequest:
		dec, err := PermissionRequestDecisionFromProto(v.GetPermissionRequest().GetDecision())
		if err != nil {
			return nil, err
		}
		return &HookSpecificOutputPermissionRequest{HookEventName: HookEventPermissionRequest, Decision: dec}, nil
	default:
		return nil, fmt.Errorf("unsupported proto HookSpecificOutput %T", x)
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
	type raw struct {
		Continue           *bool           `json:"continue,omitempty"`
		SuppressOutput     *bool           `json:"suppressOutput,omitempty"`
		StopReason         *string         `json:"stopReason,omitempty"`
		Decision           *HookDecision   `json:"decision,omitempty"`
		SystemMessage      *string         `json:"systemMessage,omitempty"`
		Reason             *string         `json:"reason,omitempty"`
		HookSpecificOutput json.RawMessage `json:"hookSpecificOutput,omitempty"`
	}
	var hso json.RawMessage
	if o.HookSpecificOutput != nil {
		b, err := marshalHookSpecificOutput(o.HookSpecificOutput)
		if err != nil {
			return nil, err
		}
		hso = b
	}
	return json.Marshal(raw{
		Continue: o.Continue, SuppressOutput: o.SuppressOutput, StopReason: o.StopReason,
		Decision: o.Decision, SystemMessage: o.SystemMessage, Reason: o.Reason, HookSpecificOutput: hso,
	})
}

func (o *SyncHookJSONOutput) UnmarshalJSON(data []byte) error {
	type raw struct {
		Continue           *bool           `json:"continue"`
		SuppressOutput     *bool           `json:"suppressOutput"`
		StopReason         *string         `json:"stopReason"`
		Decision           *HookDecision   `json:"decision"`
		SystemMessage      *string         `json:"systemMessage"`
		Reason             *string         `json:"reason"`
		HookSpecificOutput json.RawMessage `json:"hookSpecificOutput"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Continue = v.Continue
	o.SuppressOutput = v.SuppressOutput
	o.StopReason = v.StopReason
	o.Decision = v.Decision
	o.SystemMessage = v.SystemMessage
	o.Reason = v.Reason
	if len(v.HookSpecificOutput) > 0 {
		got, err := UnmarshalHookSpecificOutput(v.HookSpecificOutput)
		if err != nil {
			return err
		}
		o.HookSpecificOutput = got
	}
	return nil
}

func marshalHookSpecificOutput(v HookSpecificOutput) ([]byte, error) {
	switch x := v.(type) {
	case *HookSpecificOutputUnknown:
		return x.MarshalJSON()
	case *HookSpecificOutputPreToolUse:
		return x.MarshalJSON()
	case *HookSpecificOutputUserPromptSubmit:
		return x.MarshalJSON()
	case *HookSpecificOutputSessionStart:
		return x.MarshalJSON()
	case *HookSpecificOutputSetup:
		return x.MarshalJSON()
	case *HookSpecificOutputSubagentStart:
		return x.MarshalJSON()
	case *HookSpecificOutputPostToolUse:
		return x.MarshalJSON()
	case *HookSpecificOutputPostToolUseFailure:
		return x.MarshalJSON()
	case *HookSpecificOutputNotification:
		return x.MarshalJSON()
	case *HookSpecificOutputPermissionRequest:
		return x.MarshalJSON()
	default:
		return nil, fmt.Errorf("unsupported HookSpecificOutput %T", v)
	}
}

// UnmarshalHookJSONOutput unmarshals a HookJSONOutput union value from JSON.
func UnmarshalHookJSONOutput(data []byte) (HookJSONOutput, error) {
	var probe struct {
		Async *bool `json:"async"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, err
	}
	if probe.Async != nil && *probe.Async {
		var v AsyncHookJSONOutput
		return &v, v.UnmarshalJSON(data)
	}
	var syncV SyncHookJSONOutput
	if err := syncV.UnmarshalJSON(data); err == nil && (syncV.Continue != nil || syncV.SuppressOutput != nil || syncV.StopReason != nil || syncV.Decision != nil || syncV.SystemMessage != nil || syncV.Reason != nil || syncV.HookSpecificOutput != nil) {
		return &syncV, nil
	}
	var unknown HookJSONOutputUnknown
	return &unknown, unknown.UnmarshalJSON(data)
}

func HookJSONOutputToProto(v HookJSONOutput) (*pb.HookJSONOutput, error) {
	switch x := v.(type) {
	case *HookJSONOutputUnknown:
		return &pb.HookJSONOutput{Value: &pb.HookJSONOutput_Unknown{Unknown: x.ToProto()}}, nil
	case *AsyncHookJSONOutput:
		return &pb.HookJSONOutput{Value: &pb.HookJSONOutput_AsyncOutput{AsyncOutput: &pb.AsyncHookJSONOutput{
			Async: x.Async, AsyncTimeout: x.AsyncTimeout,
		}}}, nil
	case *SyncHookJSONOutput:
		var hso *pb.HookSpecificOutput
		var err error
		if x.HookSpecificOutput != nil {
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
		return &pb.HookJSONOutput{Value: &pb.HookJSONOutput_SyncOutput{SyncOutput: &pb.SyncHookJSONOutput{
			Continue: x.Continue, SuppressOutput: x.SuppressOutput, StopReason: x.StopReason,
			Decision: decision, SystemMessage: x.SystemMessage, Reason: x.Reason, HookSpecificOutput: hso,
		}}}, nil
	default:
		return nil, fmt.Errorf("unsupported HookJSONOutput %T", v)
	}
}

func HookJSONOutputFromProto(v *pb.HookJSONOutput) (HookJSONOutput, error) {
	if v == nil {
		return nil, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.HookJSONOutput_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return &HookJSONOutputUnknown{UnknownUnion: u}, nil
	case *pb.HookJSONOutput_AsyncOutput:
		asyncOutput := v.GetAsyncOutput()
		return &AsyncHookJSONOutput{Async: asyncOutput.GetAsync(), AsyncTimeout: protoOpt(asyncOutput, "async_timeout", asyncOutput.GetAsyncTimeout())}, nil
	case *pb.HookJSONOutput_SyncOutput:
		syncOutput := v.GetSyncOutput()
		var decision *HookDecision
		if got := syncOutput.GetDecision(); got != "" {
			d := HookDecision(got)
			decision = &d
		}
		hso, err := HookSpecificOutputFromProto(syncOutput.GetHookSpecificOutput())
		if err != nil {
			return nil, err
		}
		return &SyncHookJSONOutput{
			Continue:           protoOpt(syncOutput, "continue", syncOutput.GetContinue()),
			SuppressOutput:     protoOpt(syncOutput, "suppress_output", syncOutput.GetSuppressOutput()),
			StopReason:         protoOpt(syncOutput, "stop_reason", syncOutput.GetStopReason()),
			Decision:           decision,
			SystemMessage:      protoOpt(syncOutput, "system_message", syncOutput.GetSystemMessage()),
			Reason:             protoOpt(syncOutput, "reason", syncOutput.GetReason()),
			HookSpecificOutput: hso,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proto HookJSONOutput %T", x)
	}
}

func (o HookInputUnknown) MarshalJSON() ([]byte, error) { return o.UnknownUnion.MarshalJSON() }
func (o *HookInputUnknown) UnmarshalJSON(data []byte) error {
	return o.UnknownUnion.UnmarshalJSON(data)
}

func (o PermissionRequestHookInput) MarshalJSON() ([]byte, error) {
	type raw struct {
		BaseHookInput
		HookEventName         HookEvent         `json:"hook_event_name"`
		ToolName              string            `json:"tool_name"`
		ToolInput             json.RawMessage   `json:"tool_input"`
		ToolUseID             string            `json:"tool_use_id"`
		PermissionSuggestions []json.RawMessage `json:"permission_suggestions,omitzero"`
	}
	out := raw{
		BaseHookInput: o.BaseHookInput,
		HookEventName: HookEventPermissionRequest,
		ToolName:      o.ToolName,
		ToolInput:     o.ToolInput,
		ToolUseID:     o.ToolUseID,
	}
	if len(o.PermissionSuggestions) > 0 {
		out.PermissionSuggestions = make([]json.RawMessage, 0, len(o.PermissionSuggestions))
		for _, suggestion := range o.PermissionSuggestions {
			b, err := json.Marshal(suggestion)
			if err != nil {
				return nil, err
			}
			out.PermissionSuggestions = append(out.PermissionSuggestions, b)
		}
	}
	return json.Marshal(out)
}

func (o *PermissionRequestHookInput) UnmarshalJSON(data []byte) error {
	type raw struct {
		BaseHookInput
		HookEventName         HookEvent         `json:"hook_event_name"`
		ToolName              string            `json:"tool_name"`
		ToolInput             json.RawMessage   `json:"tool_input"`
		ToolUseID             string            `json:"tool_use_id"`
		PermissionSuggestions []json.RawMessage `json:"permission_suggestions,omitzero"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	suggestions := make([]PermissionUpdate, 0, len(v.PermissionSuggestions))
	for _, rawSuggestion := range v.PermissionSuggestions {
		suggestion, err := UnmarshalPermissionUpdate(rawSuggestion)
		if err != nil {
			return err
		}
		suggestions = append(suggestions, suggestion)
	}
	*o = PermissionRequestHookInput{
		BaseHookInput:         v.BaseHookInput,
		HookEventName:         HookEventPermissionRequest,
		ToolName:              v.ToolName,
		ToolInput:             v.ToolInput,
		ToolUseID:             v.ToolUseID,
		PermissionSuggestions: suggestions,
	}
	o.HookEventName = HookEventPermissionRequest
	return nil
}

// UnmarshalHookInput unmarshals a HookInput union value from JSON.
func UnmarshalHookInput(data []byte) (HookInput, error) {
	var disc struct {
		HookEventName HookEvent `json:"hook_event_name"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return nil, err
	}
	switch disc.HookEventName {
	case HookEventPreToolUse:
		var v PreToolUseHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventPostToolUse:
		var v PostToolUseHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventPostToolUseFailure:
		var v PostToolUseFailureHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventNotification:
		var v NotificationHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventUserPromptSubmit:
		var v UserPromptSubmitHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventSessionStart:
		var v SessionStartHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventSessionEnd:
		var v SessionEndHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventStop:
		var v StopHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventSubagentStart:
		var v SubagentStartHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventSubagentStop:
		var v SubagentStopHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventPreCompact:
		var v PreCompactHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventPermissionRequest:
		var v PermissionRequestHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventSetup:
		var v SetupHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventTeammateIdle:
		var v TeammateIdleHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventTaskCompleted:
		var v TaskCompletedHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventConfigChange:
		var v ConfigChangeHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventWorktreeCreate:
		var v WorktreeCreateHookInput
		return &v, json.Unmarshal(data, &v)
	case HookEventWorktreeRemove:
		var v WorktreeRemoveHookInput
		return &v, json.Unmarshal(data, &v)
	default:
		var v HookInputUnknown
		return &v, v.UnmarshalJSON(data)
	}
}

func HookInputToProto(v HookInput) (*pb.HookInput, error) {
	switch x := v.(type) {
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
		}}}, nil
	case *PostToolUseFailureHookInput:
		return &pb.HookInput{Value: &pb.HookInput_PostToolUseFailure{PostToolUseFailure: &pb.PostToolUseFailureHookInput{
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
		}}}, nil
	case *NotificationHookInput:
		return &pb.HookInput{Value: &pb.HookInput_Notification{Notification: &pb.NotificationHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Message: x.Message, Title: x.Title, NotificationType: x.NotificationType,
		}}}, nil
	case *UserPromptSubmitHookInput:
		return &pb.HookInput{Value: &pb.HookInput_UserPromptSubmit{UserPromptSubmit: &pb.UserPromptSubmitHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Prompt: x.Prompt,
		}}}, nil
	case *SessionStartHookInput:
		return &pb.HookInput{Value: &pb.HookInput_SessionStart{SessionStart: &pb.SessionStartHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Source: string(x.Source), Model: x.Model,
		}}}, nil
	case *SessionEndHookInput:
		return &pb.HookInput{Value: &pb.HookInput_SessionEnd{SessionEnd: &pb.SessionEndHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Reason: x.Reason,
		}}}, nil
	case *StopHookInput:
		return &pb.HookInput{Value: &pb.HookInput_Stop{Stop: &pb.StopHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), StopHookActive: x.StopHookActive,
		}}}, nil
	case *SubagentStartHookInput:
		return &pb.HookInput{Value: &pb.HookInput_SubagentStart{SubagentStart: &pb.SubagentStartHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Prompt: x.Prompt,
		}}}, nil
	case *SubagentStopHookInput:
		return &pb.HookInput{Value: &pb.HookInput_SubagentStop{SubagentStop: &pb.SubagentStopHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), StopReason: x.StopReason,
		}}}, nil
	case *PreCompactHookInput:
		return &pb.HookInput{Value: &pb.HookInput_PreCompact{PreCompact: &pb.PreCompactHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Trigger: x.Trigger, CustomInstructions: x.CustomInstructions,
		}}}, nil
	case *PermissionRequestHookInput:
		updates, err := permissionUpdatesToProto(x.PermissionSuggestions)
		if err != nil {
			return nil, err
		}
		return &pb.HookInput{Value: &pb.HookInput_PermissionRequest{PermissionRequest: &pb.PermissionRequestHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), ToolName: x.ToolName,
			ToolInput: rawToolInputToProto(x.ToolInput, x.ToolName), ToolUseId: x.ToolUseID, PermissionSuggestions: updates,
		}}}, nil
	case *SetupHookInput:
		return &pb.HookInput{Value: &pb.HookInput_Setup{Setup: &pb.SetupHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Trigger: string(x.Trigger),
		}}}, nil
	case *TeammateIdleHookInput:
		return &pb.HookInput{Value: &pb.HookInput_TeammateIdle{TeammateIdle: &pb.TeammateIdleHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName),
		}}}, nil
	case *TaskCompletedHookInput:
		return &pb.HookInput{Value: &pb.HookInput_TaskCompleted{TaskCompleted: &pb.TaskCompletedHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), TaskId: x.TaskID, TaskSubject: x.TaskSubject,
			TaskDescription: x.TaskDescription, TeammateName: x.TeammateName, TeamName: x.TeamName,
		}}}, nil
	case *ConfigChangeHookInput:
		return &pb.HookInput{Value: &pb.HookInput_ConfigChange{ConfigChange: &pb.ConfigChangeHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Source: string(x.Source), FilePath: x.FilePath,
		}}}, nil
	case *WorktreeCreateHookInput:
		return &pb.HookInput{Value: &pb.HookInput_WorktreeCreate{WorktreeCreate: &pb.WorktreeCreateHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), Name: x.Name,
		}}}, nil
	case *WorktreeRemoveHookInput:
		return &pb.HookInput{Value: &pb.HookInput_WorktreeRemove{WorktreeRemove: &pb.WorktreeRemoveHookInput{
			SessionId: x.SessionID, TranscriptPath: x.TranscriptPath, Cwd: x.Cwd, PermissionMode: x.PermissionMode,
			AgentId: x.AgentID, AgentType: x.AgentType, HookEventName: string(x.HookEventName), WorktreePath: x.WorktreePath,
		}}}, nil
	default:
		return nil, fmt.Errorf("unsupported HookInput %T", v)
	}
}

func HookInputFromProto(v *pb.HookInput) (HookInput, error) {
	if v == nil {
		return nil, nil
	}
	switch x := v.GetValue().(type) {
	case *pb.HookInput_Unknown:
		var u UnknownUnion
		u.FromProto(v.GetUnknown())
		return &HookInputUnknown{UnknownUnion: u}, nil
	case *pb.HookInput_PreToolUse:
		preToolUse := v.GetPreToolUse()
		raw, err := rawFromToolInputProto(preToolUse.GetToolInput())
		if err != nil {
			return nil, err
		}
		return &PreToolUseHookInput{BaseHookInput: BaseHookInput{SessionID: preToolUse.GetSessionId(), TranscriptPath: preToolUse.GetTranscriptPath(), Cwd: preToolUse.GetCwd(), PermissionMode: protoOpt(preToolUse, "permission_mode", preToolUse.GetPermissionMode()), AgentID: protoOpt(preToolUse, "agent_id", preToolUse.GetAgentId()), AgentType: protoOpt(preToolUse, "agent_type", preToolUse.GetAgentType())}, HookEventName: HookEvent(preToolUse.GetHookEventName()), ToolName: preToolUse.GetToolName(), ToolInput: raw, ToolUseID: preToolUse.GetToolUseId()}, nil
	case *pb.HookInput_PostToolUse:
		postToolUse := v.GetPostToolUse()
		rawIn, err := rawFromToolInputProto(postToolUse.GetToolInput())
		if err != nil {
			return nil, err
		}
		rawOut, err := rawFromToolOutputProto(postToolUse.GetToolResponse())
		if err != nil {
			return nil, err
		}
		return &PostToolUseHookInput{BaseHookInput: BaseHookInput{SessionID: postToolUse.GetSessionId(), TranscriptPath: postToolUse.GetTranscriptPath(), Cwd: postToolUse.GetCwd(), PermissionMode: protoOpt(postToolUse, "permission_mode", postToolUse.GetPermissionMode()), AgentID: protoOpt(postToolUse, "agent_id", postToolUse.GetAgentId()), AgentType: protoOpt(postToolUse, "agent_type", postToolUse.GetAgentType())}, HookEventName: HookEvent(postToolUse.GetHookEventName()), ToolName: postToolUse.GetToolName(), ToolInput: rawIn, ToolResponse: rawOut, ToolUseID: postToolUse.GetToolUseId()}, nil
	case *pb.HookInput_PostToolUseFailure:
		postToolUseFailure := v.GetPostToolUseFailure()
		raw, err := rawFromToolInputProto(postToolUseFailure.GetToolInput())
		if err != nil {
			return nil, err
		}
		return &PostToolUseFailureHookInput{BaseHookInput: BaseHookInput{SessionID: postToolUseFailure.GetSessionId(), TranscriptPath: postToolUseFailure.GetTranscriptPath(), Cwd: postToolUseFailure.GetCwd(), PermissionMode: protoOpt(postToolUseFailure, "permission_mode", postToolUseFailure.GetPermissionMode()), AgentID: protoOpt(postToolUseFailure, "agent_id", postToolUseFailure.GetAgentId()), AgentType: protoOpt(postToolUseFailure, "agent_type", postToolUseFailure.GetAgentType())}, HookEventName: HookEvent(postToolUseFailure.GetHookEventName()), ToolName: postToolUseFailure.GetToolName(), ToolInput: raw, ToolUseID: postToolUseFailure.GetToolUseId(), Error: postToolUseFailure.GetError(), IsInterrupt: protoOpt(postToolUseFailure, "is_interrupt", postToolUseFailure.GetIsInterrupt())}, nil
	case *pb.HookInput_Notification:
		notification := v.GetNotification()
		return &NotificationHookInput{BaseHookInput: BaseHookInput{SessionID: notification.GetSessionId(), TranscriptPath: notification.GetTranscriptPath(), Cwd: notification.GetCwd(), PermissionMode: protoOpt(notification, "permission_mode", notification.GetPermissionMode()), AgentID: protoOpt(notification, "agent_id", notification.GetAgentId()), AgentType: protoOpt(notification, "agent_type", notification.GetAgentType())}, HookEventName: HookEvent(notification.GetHookEventName()), Message: notification.GetMessage(), Title: protoOpt(notification, "title", notification.GetTitle()), NotificationType: notification.GetNotificationType()}, nil
	case *pb.HookInput_UserPromptSubmit:
		userPromptSubmit := v.GetUserPromptSubmit()
		return &UserPromptSubmitHookInput{BaseHookInput: BaseHookInput{SessionID: userPromptSubmit.GetSessionId(), TranscriptPath: userPromptSubmit.GetTranscriptPath(), Cwd: userPromptSubmit.GetCwd(), PermissionMode: protoOpt(userPromptSubmit, "permission_mode", userPromptSubmit.GetPermissionMode()), AgentID: protoOpt(userPromptSubmit, "agent_id", userPromptSubmit.GetAgentId()), AgentType: protoOpt(userPromptSubmit, "agent_type", userPromptSubmit.GetAgentType())}, HookEventName: HookEvent(userPromptSubmit.GetHookEventName()), Prompt: userPromptSubmit.GetPrompt()}, nil
	case *pb.HookInput_SessionStart:
		sessionStart := v.GetSessionStart()
		return &SessionStartHookInput{BaseHookInput: BaseHookInput{SessionID: sessionStart.GetSessionId(), TranscriptPath: sessionStart.GetTranscriptPath(), Cwd: sessionStart.GetCwd(), PermissionMode: protoOpt(sessionStart, "permission_mode", sessionStart.GetPermissionMode()), AgentID: protoOpt(sessionStart, "agent_id", sessionStart.GetAgentId()), AgentType: protoOpt(sessionStart, "agent_type", sessionStart.GetAgentType())}, HookEventName: HookEvent(sessionStart.GetHookEventName()), Source: SessionStartSource(sessionStart.GetSource()), Model: protoOpt(sessionStart, "model", sessionStart.GetModel())}, nil
	case *pb.HookInput_SessionEnd:
		sessionEnd := v.GetSessionEnd()
		return &SessionEndHookInput{BaseHookInput: BaseHookInput{SessionID: sessionEnd.GetSessionId(), TranscriptPath: sessionEnd.GetTranscriptPath(), Cwd: sessionEnd.GetCwd(), PermissionMode: protoOpt(sessionEnd, "permission_mode", sessionEnd.GetPermissionMode()), AgentID: protoOpt(sessionEnd, "agent_id", sessionEnd.GetAgentId()), AgentType: protoOpt(sessionEnd, "agent_type", sessionEnd.GetAgentType())}, HookEventName: HookEvent(sessionEnd.GetHookEventName()), Reason: sessionEnd.GetReason()}, nil
	case *pb.HookInput_Stop:
		stop := v.GetStop()
		return &StopHookInput{BaseHookInput: BaseHookInput{SessionID: stop.GetSessionId(), TranscriptPath: stop.GetTranscriptPath(), Cwd: stop.GetCwd(), PermissionMode: protoOpt(stop, "permission_mode", stop.GetPermissionMode()), AgentID: protoOpt(stop, "agent_id", stop.GetAgentId()), AgentType: protoOpt(stop, "agent_type", stop.GetAgentType())}, HookEventName: HookEvent(stop.GetHookEventName()), StopHookActive: stop.GetStopHookActive()}, nil
	case *pb.HookInput_SubagentStart:
		subagentStart := v.GetSubagentStart()
		return &SubagentStartHookInput{BaseHookInput: BaseHookInput{SessionID: subagentStart.GetSessionId(), TranscriptPath: subagentStart.GetTranscriptPath(), Cwd: subagentStart.GetCwd(), PermissionMode: protoOpt(subagentStart, "permission_mode", subagentStart.GetPermissionMode()), AgentID: protoOpt(subagentStart, "agent_id", subagentStart.GetAgentId()), AgentType: protoOpt(subagentStart, "agent_type", subagentStart.GetAgentType())}, HookEventName: HookEvent(subagentStart.GetHookEventName()), Prompt: subagentStart.GetPrompt()}, nil
	case *pb.HookInput_SubagentStop:
		subagentStop := v.GetSubagentStop()
		return &SubagentStopHookInput{BaseHookInput: BaseHookInput{SessionID: subagentStop.GetSessionId(), TranscriptPath: subagentStop.GetTranscriptPath(), Cwd: subagentStop.GetCwd(), PermissionMode: protoOpt(subagentStop, "permission_mode", subagentStop.GetPermissionMode()), AgentID: protoOpt(subagentStop, "agent_id", subagentStop.GetAgentId()), AgentType: protoOpt(subagentStop, "agent_type", subagentStop.GetAgentType())}, HookEventName: HookEvent(subagentStop.GetHookEventName()), StopReason: subagentStop.GetStopReason()}, nil
	case *pb.HookInput_PreCompact:
		preCompact := v.GetPreCompact()
		return &PreCompactHookInput{BaseHookInput: BaseHookInput{SessionID: preCompact.GetSessionId(), TranscriptPath: preCompact.GetTranscriptPath(), Cwd: preCompact.GetCwd(), PermissionMode: protoOpt(preCompact, "permission_mode", preCompact.GetPermissionMode()), AgentID: protoOpt(preCompact, "agent_id", preCompact.GetAgentId()), AgentType: protoOpt(preCompact, "agent_type", preCompact.GetAgentType())}, HookEventName: HookEvent(preCompact.GetHookEventName()), Trigger: preCompact.GetTrigger(), CustomInstructions: preCompact.GetCustomInstructions()}, nil
	case *pb.HookInput_PermissionRequest:
		permissionRequest := v.GetPermissionRequest()
		raw, err := rawFromToolInputProto(permissionRequest.GetToolInput())
		if err != nil {
			return nil, err
		}
		updates, err := permissionUpdatesFromProto(permissionRequest.GetPermissionSuggestions())
		if err != nil {
			return nil, err
		}
		return &PermissionRequestHookInput{BaseHookInput: BaseHookInput{SessionID: permissionRequest.GetSessionId(), TranscriptPath: permissionRequest.GetTranscriptPath(), Cwd: permissionRequest.GetCwd(), PermissionMode: protoOpt(permissionRequest, "permission_mode", permissionRequest.GetPermissionMode()), AgentID: protoOpt(permissionRequest, "agent_id", permissionRequest.GetAgentId()), AgentType: protoOpt(permissionRequest, "agent_type", permissionRequest.GetAgentType())}, HookEventName: HookEvent(permissionRequest.GetHookEventName()), ToolName: permissionRequest.GetToolName(), ToolInput: raw, ToolUseID: permissionRequest.GetToolUseId(), PermissionSuggestions: updates}, nil
	case *pb.HookInput_Setup:
		setup := v.GetSetup()
		return &SetupHookInput{BaseHookInput: BaseHookInput{SessionID: setup.GetSessionId(), TranscriptPath: setup.GetTranscriptPath(), Cwd: setup.GetCwd(), PermissionMode: protoOpt(setup, "permission_mode", setup.GetPermissionMode()), AgentID: protoOpt(setup, "agent_id", setup.GetAgentId()), AgentType: protoOpt(setup, "agent_type", setup.GetAgentType())}, HookEventName: HookEvent(setup.GetHookEventName()), Trigger: SetupTrigger(setup.GetTrigger())}, nil
	case *pb.HookInput_TeammateIdle:
		teammateIdle := v.GetTeammateIdle()
		return &TeammateIdleHookInput{BaseHookInput: BaseHookInput{SessionID: teammateIdle.GetSessionId(), TranscriptPath: teammateIdle.GetTranscriptPath(), Cwd: teammateIdle.GetCwd(), PermissionMode: protoOpt(teammateIdle, "permission_mode", teammateIdle.GetPermissionMode()), AgentID: protoOpt(teammateIdle, "agent_id", teammateIdle.GetAgentId()), AgentType: protoOpt(teammateIdle, "agent_type", teammateIdle.GetAgentType())}, HookEventName: HookEvent(teammateIdle.GetHookEventName())}, nil
	case *pb.HookInput_TaskCompleted:
		taskCompleted := v.GetTaskCompleted()
		return &TaskCompletedHookInput{BaseHookInput: BaseHookInput{SessionID: taskCompleted.GetSessionId(), TranscriptPath: taskCompleted.GetTranscriptPath(), Cwd: taskCompleted.GetCwd(), PermissionMode: protoOpt(taskCompleted, "permission_mode", taskCompleted.GetPermissionMode()), AgentID: protoOpt(taskCompleted, "agent_id", taskCompleted.GetAgentId()), AgentType: protoOpt(taskCompleted, "agent_type", taskCompleted.GetAgentType())}, HookEventName: HookEvent(taskCompleted.GetHookEventName()), TaskID: taskCompleted.GetTaskId(), TaskSubject: taskCompleted.GetTaskSubject(), TaskDescription: protoOpt(taskCompleted, "task_description", taskCompleted.GetTaskDescription()), TeammateName: protoOpt(taskCompleted, "teammate_name", taskCompleted.GetTeammateName()), TeamName: protoOpt(taskCompleted, "team_name", taskCompleted.GetTeamName())}, nil
	case *pb.HookInput_ConfigChange:
		configChange := v.GetConfigChange()
		return &ConfigChangeHookInput{BaseHookInput: BaseHookInput{SessionID: configChange.GetSessionId(), TranscriptPath: configChange.GetTranscriptPath(), Cwd: configChange.GetCwd(), PermissionMode: protoOpt(configChange, "permission_mode", configChange.GetPermissionMode()), AgentID: protoOpt(configChange, "agent_id", configChange.GetAgentId()), AgentType: protoOpt(configChange, "agent_type", configChange.GetAgentType())}, HookEventName: HookEvent(configChange.GetHookEventName()), Source: ConfigChangeSource(configChange.GetSource()), FilePath: protoOpt(configChange, "file_path", configChange.GetFilePath())}, nil
	case *pb.HookInput_WorktreeCreate:
		worktreeCreate := v.GetWorktreeCreate()
		return &WorktreeCreateHookInput{BaseHookInput: BaseHookInput{SessionID: worktreeCreate.GetSessionId(), TranscriptPath: worktreeCreate.GetTranscriptPath(), Cwd: worktreeCreate.GetCwd(), PermissionMode: protoOpt(worktreeCreate, "permission_mode", worktreeCreate.GetPermissionMode()), AgentID: protoOpt(worktreeCreate, "agent_id", worktreeCreate.GetAgentId()), AgentType: protoOpt(worktreeCreate, "agent_type", worktreeCreate.GetAgentType())}, HookEventName: HookEvent(worktreeCreate.GetHookEventName()), Name: worktreeCreate.GetName()}, nil
	case *pb.HookInput_WorktreeRemove:
		worktreeRemove := v.GetWorktreeRemove()
		return &WorktreeRemoveHookInput{BaseHookInput: BaseHookInput{SessionID: worktreeRemove.GetSessionId(), TranscriptPath: worktreeRemove.GetTranscriptPath(), Cwd: worktreeRemove.GetCwd(), PermissionMode: protoOpt(worktreeRemove, "permission_mode", worktreeRemove.GetPermissionMode()), AgentID: protoOpt(worktreeRemove, "agent_id", worktreeRemove.GetAgentId()), AgentType: protoOpt(worktreeRemove, "agent_type", worktreeRemove.GetAgentType())}, HookEventName: HookEvent(worktreeRemove.GetHookEventName()), WorktreePath: worktreeRemove.GetWorktreePath()}, nil
	default:
		return nil, fmt.Errorf("unsupported proto HookInput %T", x)
	}
}
