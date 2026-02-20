package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

// HookEvent converters

func HookEventFromProto(pb sdktypes.HookEvent) HookEvent {
	switch pb {
	case sdktypes.HookEvent_HOOK_EVENT_PRE_TOOL_USE:
		return HookEventPreToolUse
	case sdktypes.HookEvent_HOOK_EVENT_POST_TOOL_USE:
		return HookEventPostToolUse
	case sdktypes.HookEvent_HOOK_EVENT_POST_TOOL_USE_FAILURE:
		return HookEventPostToolUseFailure
	case sdktypes.HookEvent_HOOK_EVENT_NOTIFICATION:
		return HookEventNotification
	case sdktypes.HookEvent_HOOK_EVENT_USER_PROMPT_SUBMIT:
		return HookEventUserPromptSubmit
	case sdktypes.HookEvent_HOOK_EVENT_SESSION_START:
		return HookEventSessionStart
	case sdktypes.HookEvent_HOOK_EVENT_SESSION_END:
		return HookEventSessionEnd
	case sdktypes.HookEvent_HOOK_EVENT_STOP:
		return HookEventStop
	case sdktypes.HookEvent_HOOK_EVENT_SUBAGENT_START:
		return HookEventSubagentStart
	case sdktypes.HookEvent_HOOK_EVENT_SUBAGENT_STOP:
		return HookEventSubagentStop
	case sdktypes.HookEvent_HOOK_EVENT_PRE_COMPACT:
		return HookEventPreCompact
	case sdktypes.HookEvent_HOOK_EVENT_PERMISSION_REQUEST:
		return HookEventPermissionRequest
	default:
		return ""
	}
}

func (e HookEvent) ToProto() sdktypes.HookEvent {
	switch e {
	case HookEventPreToolUse:
		return sdktypes.HookEvent_HOOK_EVENT_PRE_TOOL_USE
	case HookEventPostToolUse:
		return sdktypes.HookEvent_HOOK_EVENT_POST_TOOL_USE
	case HookEventPostToolUseFailure:
		return sdktypes.HookEvent_HOOK_EVENT_POST_TOOL_USE_FAILURE
	case HookEventNotification:
		return sdktypes.HookEvent_HOOK_EVENT_NOTIFICATION
	case HookEventUserPromptSubmit:
		return sdktypes.HookEvent_HOOK_EVENT_USER_PROMPT_SUBMIT
	case HookEventSessionStart:
		return sdktypes.HookEvent_HOOK_EVENT_SESSION_START
	case HookEventSessionEnd:
		return sdktypes.HookEvent_HOOK_EVENT_SESSION_END
	case HookEventStop:
		return sdktypes.HookEvent_HOOK_EVENT_STOP
	case HookEventSubagentStart:
		return sdktypes.HookEvent_HOOK_EVENT_SUBAGENT_START
	case HookEventSubagentStop:
		return sdktypes.HookEvent_HOOK_EVENT_SUBAGENT_STOP
	case HookEventPreCompact:
		return sdktypes.HookEvent_HOOK_EVENT_PRE_COMPACT
	case HookEventPermissionRequest:
		return sdktypes.HookEvent_HOOK_EVENT_PERMISSION_REQUEST
	default:
		return sdktypes.HookEvent_HOOK_EVENT_UNSPECIFIED
	}
}

// SessionStartSource converters

func SessionStartSourceFromProto(pb sdktypes.SessionStartSource) SessionStartSource {
	switch pb {
	case sdktypes.SessionStartSource_SESSION_START_SOURCE_STARTUP:
		return SessionStartSourceStartup
	case sdktypes.SessionStartSource_SESSION_START_SOURCE_RESUME:
		return SessionStartSourceResume
	case sdktypes.SessionStartSource_SESSION_START_SOURCE_CLEAR:
		return SessionStartSourceClear
	case sdktypes.SessionStartSource_SESSION_START_SOURCE_COMPACT:
		return SessionStartSourceCompact
	default:
		return ""
	}
}

func (e SessionStartSource) ToProto() sdktypes.SessionStartSource {
	switch e {
	case SessionStartSourceStartup:
		return sdktypes.SessionStartSource_SESSION_START_SOURCE_STARTUP
	case SessionStartSourceResume:
		return sdktypes.SessionStartSource_SESSION_START_SOURCE_RESUME
	case SessionStartSourceClear:
		return sdktypes.SessionStartSource_SESSION_START_SOURCE_CLEAR
	case SessionStartSourceCompact:
		return sdktypes.SessionStartSource_SESSION_START_SOURCE_COMPACT
	default:
		return sdktypes.SessionStartSource_SESSION_START_SOURCE_UNSPECIFIED
	}
}

// PreCompactTrigger converters

func PreCompactTriggerFromProto(pb sdktypes.PreCompactTrigger) PreCompactTrigger {
	switch pb {
	case sdktypes.PreCompactTrigger_PRE_COMPACT_TRIGGER_MANUAL:
		return PreCompactTriggerManual
	case sdktypes.PreCompactTrigger_PRE_COMPACT_TRIGGER_AUTO:
		return PreCompactTriggerAuto
	default:
		return ""
	}
}

func (e PreCompactTrigger) ToProto() sdktypes.PreCompactTrigger {
	switch e {
	case PreCompactTriggerManual:
		return sdktypes.PreCompactTrigger_PRE_COMPACT_TRIGGER_MANUAL
	case PreCompactTriggerAuto:
		return sdktypes.PreCompactTrigger_PRE_COMPACT_TRIGGER_AUTO
	default:
		return sdktypes.PreCompactTrigger_PRE_COMPACT_TRIGGER_UNSPECIFIED
	}
}

// GrepOutputMode converters

func GrepOutputModeFromProto(pb sdktypes.GrepOutputMode) GrepOutputMode {
	switch pb {
	case sdktypes.GrepOutputMode_GREP_OUTPUT_MODE_CONTENT:
		return GrepOutputModeContent
	case sdktypes.GrepOutputMode_GREP_OUTPUT_MODE_FILES_WITH_MATCHES:
		return GrepOutputModeFilesWithMatches
	case sdktypes.GrepOutputMode_GREP_OUTPUT_MODE_COUNT:
		return GrepOutputModeCount
	default:
		return ""
	}
}

func (e GrepOutputMode) ToProto() sdktypes.GrepOutputMode {
	switch e {
	case GrepOutputModeContent:
		return sdktypes.GrepOutputMode_GREP_OUTPUT_MODE_CONTENT
	case GrepOutputModeFilesWithMatches:
		return sdktypes.GrepOutputMode_GREP_OUTPUT_MODE_FILES_WITH_MATCHES
	case GrepOutputModeCount:
		return sdktypes.GrepOutputMode_GREP_OUTPUT_MODE_COUNT
	default:
		return sdktypes.GrepOutputMode_GREP_OUTPUT_MODE_UNSPECIFIED
	}
}

// NotebookCellType converters

func NotebookCellTypeFromProto(pb sdktypes.NotebookCellType) NotebookCellType {
	switch pb {
	case sdktypes.NotebookCellType_NOTEBOOK_CELL_TYPE_CODE:
		return NotebookCellTypeCode
	case sdktypes.NotebookCellType_NOTEBOOK_CELL_TYPE_MARKDOWN:
		return NotebookCellTypeMarkdown
	default:
		return ""
	}
}

func (e NotebookCellType) ToProto() sdktypes.NotebookCellType {
	switch e {
	case NotebookCellTypeCode:
		return sdktypes.NotebookCellType_NOTEBOOK_CELL_TYPE_CODE
	case NotebookCellTypeMarkdown:
		return sdktypes.NotebookCellType_NOTEBOOK_CELL_TYPE_MARKDOWN
	default:
		return sdktypes.NotebookCellType_NOTEBOOK_CELL_TYPE_UNSPECIFIED
	}
}

// NotebookEditMode converters

func NotebookEditModeFromProto(pb sdktypes.NotebookEditMode) NotebookEditMode {
	switch pb {
	case sdktypes.NotebookEditMode_NOTEBOOK_EDIT_MODE_REPLACE:
		return NotebookEditModeReplace
	case sdktypes.NotebookEditMode_NOTEBOOK_EDIT_MODE_INSERT:
		return NotebookEditModeInsert
	case sdktypes.NotebookEditMode_NOTEBOOK_EDIT_MODE_DELETE:
		return NotebookEditModeDelete
	default:
		return ""
	}
}

func (e NotebookEditMode) ToProto() sdktypes.NotebookEditMode {
	switch e {
	case NotebookEditModeReplace:
		return sdktypes.NotebookEditMode_NOTEBOOK_EDIT_MODE_REPLACE
	case NotebookEditModeInsert:
		return sdktypes.NotebookEditMode_NOTEBOOK_EDIT_MODE_INSERT
	case NotebookEditModeDelete:
		return sdktypes.NotebookEditMode_NOTEBOOK_EDIT_MODE_DELETE
	default:
		return sdktypes.NotebookEditMode_NOTEBOOK_EDIT_MODE_UNSPECIFIED
	}
}

// TodoStatus converters

func TodoStatusFromProto(pb sdktypes.TodoStatus) TodoStatus {
	switch pb {
	case sdktypes.TodoStatus_TODO_STATUS_PENDING:
		return TodoStatusPending
	case sdktypes.TodoStatus_TODO_STATUS_IN_PROGRESS:
		return TodoStatusInProgress
	case sdktypes.TodoStatus_TODO_STATUS_COMPLETED:
		return TodoStatusCompleted
	default:
		return ""
	}
}

func (e TodoStatus) ToProto() sdktypes.TodoStatus {
	switch e {
	case TodoStatusPending:
		return sdktypes.TodoStatus_TODO_STATUS_PENDING
	case TodoStatusInProgress:
		return sdktypes.TodoStatus_TODO_STATUS_IN_PROGRESS
	case TodoStatusCompleted:
		return sdktypes.TodoStatus_TODO_STATUS_COMPLETED
	default:
		return sdktypes.TodoStatus_TODO_STATUS_UNSPECIFIED
	}
}

// SDKResultSubtype converters

func SDKResultSubtypeFromProto(pb sdktypes.SDKResultSubtype) SDKResultSubtype {
	switch pb {
	case sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_SUCCESS:
		return SDKResultSubtypeSuccess
	case sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_TURNS:
		return SDKResultSubtypeErrorMaxTurns
	case sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_DURING_EXECUTION:
		return SDKResultSubtypeErrorDuringExecution
	case sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_BUDGET_USD:
		return SDKResultSubtypeErrorMaxBudgetUsd
	case sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_STRUCTURED_OUTPUT_RETRIES:
		return SDKResultSubtypeErrorMaxStructuredOutputRetries
	default:
		return ""
	}
}

func (e SDKResultSubtype) ToProto() sdktypes.SDKResultSubtype {
	switch e {
	case SDKResultSubtypeSuccess:
		return sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_SUCCESS
	case SDKResultSubtypeErrorMaxTurns:
		return sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_TURNS
	case SDKResultSubtypeErrorDuringExecution:
		return sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_DURING_EXECUTION
	case SDKResultSubtypeErrorMaxBudgetUsd:
		return sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_BUDGET_USD
	case SDKResultSubtypeErrorMaxStructuredOutputRetries:
		return sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_STRUCTURED_OUTPUT_RETRIES
	default:
		return sdktypes.SDKResultSubtype_SDK_RESULT_SUBTYPE_UNSPECIFIED
	}
}

// ApiKeySource converters

func ApiKeySourceFromProto(pb sdktypes.ApiKeySource) ApiKeySource {
	switch pb {
	case sdktypes.ApiKeySource_API_KEY_SOURCE_USER:
		return ApiKeySourceUser
	case sdktypes.ApiKeySource_API_KEY_SOURCE_PROJECT:
		return ApiKeySourceProject
	case sdktypes.ApiKeySource_API_KEY_SOURCE_ORG:
		return ApiKeySourceOrg
	case sdktypes.ApiKeySource_API_KEY_SOURCE_TEMPORARY:
		return ApiKeySourceTemporary
	default:
		return ""
	}
}

func (e ApiKeySource) ToProto() sdktypes.ApiKeySource {
	switch e {
	case ApiKeySourceUser:
		return sdktypes.ApiKeySource_API_KEY_SOURCE_USER
	case ApiKeySourceProject:
		return sdktypes.ApiKeySource_API_KEY_SOURCE_PROJECT
	case ApiKeySourceOrg:
		return sdktypes.ApiKeySource_API_KEY_SOURCE_ORG
	case ApiKeySourceTemporary:
		return sdktypes.ApiKeySource_API_KEY_SOURCE_TEMPORARY
	default:
		return sdktypes.ApiKeySource_API_KEY_SOURCE_UNSPECIFIED
	}
}

// SdkBeta converters

func SdkBetaFromProto(pb sdktypes.SdkBeta) SdkBeta {
	switch pb {
	case sdktypes.SdkBeta_SDK_BETA_CONTEXT_1M_2025_08_07:
		return SdkBetaContext1m20250807
	default:
		return ""
	}
}

func (e SdkBeta) ToProto() sdktypes.SdkBeta {
	switch e {
	case SdkBetaContext1m20250807:
		return sdktypes.SdkBeta_SDK_BETA_CONTEXT_1M_2025_08_07
	default:
		return sdktypes.SdkBeta_SDK_BETA_UNSPECIFIED
	}
}

// ConfigScope converters

func ConfigScopeFromProto(pb sdktypes.ConfigScope) ConfigScope {
	switch pb {
	case sdktypes.ConfigScope_CONFIG_SCOPE_LOCAL:
		return ConfigScopeLocal
	case sdktypes.ConfigScope_CONFIG_SCOPE_USER:
		return ConfigScopeUser
	case sdktypes.ConfigScope_CONFIG_SCOPE_PROJECT:
		return ConfigScopeProject
	default:
		return ""
	}
}

func (e ConfigScope) ToProto() sdktypes.ConfigScope {
	switch e {
	case ConfigScopeLocal:
		return sdktypes.ConfigScope_CONFIG_SCOPE_LOCAL
	case ConfigScopeUser:
		return sdktypes.ConfigScope_CONFIG_SCOPE_USER
	case ConfigScopeProject:
		return sdktypes.ConfigScope_CONFIG_SCOPE_PROJECT
	default:
		return sdktypes.ConfigScope_CONFIG_SCOPE_UNSPECIFIED
	}
}

// McpServerConnectionStatus converters

func McpServerConnectionStatusFromProto(pb sdktypes.McpServerConnectionStatus) McpServerConnectionStatus {
	switch pb {
	case sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_CONNECTED:
		return McpServerConnectionStatusConnected
	case sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_FAILED:
		return McpServerConnectionStatusFailed
	case sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_NEEDS_AUTH:
		return McpServerConnectionStatusNeedsAuth
	case sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_PENDING:
		return McpServerConnectionStatusPending
	default:
		return ""
	}
}

func (e McpServerConnectionStatus) ToProto() sdktypes.McpServerConnectionStatus {
	switch e {
	case McpServerConnectionStatusConnected:
		return sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_CONNECTED
	case McpServerConnectionStatusFailed:
		return sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_FAILED
	case McpServerConnectionStatusNeedsAuth:
		return sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_NEEDS_AUTH
	case McpServerConnectionStatusPending:
		return sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_PENDING
	default:
		return sdktypes.McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_UNSPECIFIED
	}
}

// CallToolResultContentType converters

func CallToolResultContentTypeFromProto(pb sdktypes.CallToolResultContentType) CallToolResultContentType {
	switch pb {
	case sdktypes.CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_TEXT:
		return CallToolResultContentTypeText
	case sdktypes.CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_IMAGE:
		return CallToolResultContentTypeImage
	case sdktypes.CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_RESOURCE:
		return CallToolResultContentTypeResource
	default:
		return ""
	}
}

func (e CallToolResultContentType) ToProto() sdktypes.CallToolResultContentType {
	switch e {
	case CallToolResultContentTypeText:
		return sdktypes.CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_TEXT
	case CallToolResultContentTypeImage:
		return sdktypes.CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_IMAGE
	case CallToolResultContentTypeResource:
		return sdktypes.CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_RESOURCE
	default:
		return sdktypes.CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_UNSPECIFIED
	}
}

// BashOutputStatus converters

func BashOutputStatusFromProto(pb sdktypes.BashOutputStatus) BashOutputStatus {
	switch pb {
	case sdktypes.BashOutputStatus_BASH_OUTPUT_STATUS_RUNNING:
		return BashOutputStatusRunning
	case sdktypes.BashOutputStatus_BASH_OUTPUT_STATUS_COMPLETED:
		return BashOutputStatusCompleted
	case sdktypes.BashOutputStatus_BASH_OUTPUT_STATUS_FAILED:
		return BashOutputStatusFailed
	default:
		return ""
	}
}

func (e BashOutputStatus) ToProto() sdktypes.BashOutputStatus {
	switch e {
	case BashOutputStatusRunning:
		return sdktypes.BashOutputStatus_BASH_OUTPUT_STATUS_RUNNING
	case BashOutputStatusCompleted:
		return sdktypes.BashOutputStatus_BASH_OUTPUT_STATUS_COMPLETED
	case BashOutputStatusFailed:
		return sdktypes.BashOutputStatus_BASH_OUTPUT_STATUS_FAILED
	default:
		return sdktypes.BashOutputStatus_BASH_OUTPUT_STATUS_UNSPECIFIED
	}
}

// NotebookEditType converters

func NotebookEditTypeFromProto(pb sdktypes.NotebookEditType) NotebookEditType {
	switch pb {
	case sdktypes.NotebookEditType_NOTEBOOK_EDIT_TYPE_REPLACED:
		return NotebookEditTypeReplaced
	case sdktypes.NotebookEditType_NOTEBOOK_EDIT_TYPE_INSERTED:
		return NotebookEditTypeInserted
	case sdktypes.NotebookEditType_NOTEBOOK_EDIT_TYPE_DELETED:
		return NotebookEditTypeDeleted
	default:
		return ""
	}
}

func (e NotebookEditType) ToProto() sdktypes.NotebookEditType {
	switch e {
	case NotebookEditTypeReplaced:
		return sdktypes.NotebookEditType_NOTEBOOK_EDIT_TYPE_REPLACED
	case NotebookEditTypeInserted:
		return sdktypes.NotebookEditType_NOTEBOOK_EDIT_TYPE_INSERTED
	case NotebookEditTypeDeleted:
		return sdktypes.NotebookEditType_NOTEBOOK_EDIT_TYPE_DELETED
	default:
		return sdktypes.NotebookEditType_NOTEBOOK_EDIT_TYPE_UNSPECIFIED
	}
}

// PermissionMode converters

func PermissionModeFromProto(pb sdktypes.PermissionMode) PermissionMode {
	switch pb {
	case sdktypes.PermissionMode_PERMISSION_MODE_DEFAULT:
		return PermissionModeDefault
	case sdktypes.PermissionMode_PERMISSION_MODE_ACCEPT_EDITS:
		return PermissionModeAcceptEdits
	case sdktypes.PermissionMode_PERMISSION_MODE_BYPASS_PERMISSIONS:
		return PermissionModeBypassPermissions
	case sdktypes.PermissionMode_PERMISSION_MODE_PLAN:
		return PermissionModePlan
	default:
		return ""
	}
}

func (e PermissionMode) ToProto() sdktypes.PermissionMode {
	switch e {
	case PermissionModeDefault:
		return sdktypes.PermissionMode_PERMISSION_MODE_DEFAULT
	case PermissionModeAcceptEdits:
		return sdktypes.PermissionMode_PERMISSION_MODE_ACCEPT_EDITS
	case PermissionModeBypassPermissions:
		return sdktypes.PermissionMode_PERMISSION_MODE_BYPASS_PERMISSIONS
	case PermissionModePlan:
		return sdktypes.PermissionMode_PERMISSION_MODE_PLAN
	default:
		return sdktypes.PermissionMode_PERMISSION_MODE_UNSPECIFIED
	}
}

// PermissionBehavior converters

func PermissionBehaviorFromProto(pb sdktypes.PermissionBehavior) PermissionBehavior {
	switch pb {
	case sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_ALLOW:
		return PermissionBehaviorAllow
	case sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_DENY:
		return PermissionBehaviorDeny
	case sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_ASK:
		return PermissionBehaviorAsk
	default:
		return ""
	}
}

func (e PermissionBehavior) ToProto() sdktypes.PermissionBehavior {
	switch e {
	case PermissionBehaviorAllow:
		return sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_ALLOW
	case PermissionBehaviorDeny:
		return sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_DENY
	case PermissionBehaviorAsk:
		return sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_ASK
	default:
		return sdktypes.PermissionBehavior_PERMISSION_BEHAVIOR_UNSPECIFIED
	}
}

// PermissionUpdateDestination converters

func PermissionUpdateDestinationFromProto(pb sdktypes.PermissionUpdateDestination) PermissionUpdateDestination {
	switch pb {
	case sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_USER_SETTINGS:
		return PermissionUpdateDestinationUserSettings
	case sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_PROJECT_SETTINGS:
		return PermissionUpdateDestinationProjectSettings
	case sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_LOCAL_SETTINGS:
		return PermissionUpdateDestinationLocalSettings
	case sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_SESSION:
		return PermissionUpdateDestinationSession
	default:
		return ""
	}
}

func (e PermissionUpdateDestination) ToProto() sdktypes.PermissionUpdateDestination {
	switch e {
	case PermissionUpdateDestinationUserSettings:
		return sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_USER_SETTINGS
	case PermissionUpdateDestinationProjectSettings:
		return sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_PROJECT_SETTINGS
	case PermissionUpdateDestinationLocalSettings:
		return sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_LOCAL_SETTINGS
	case PermissionUpdateDestinationSession:
		return sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_SESSION
	default:
		return sdktypes.PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_UNSPECIFIED
	}
}

// SettingSource converters

func SettingSourceFromProto(pb sdktypes.SettingSource) SettingSource {
	switch pb {
	case sdktypes.SettingSource_SETTING_SOURCE_USER:
		return SettingSourceUser
	case sdktypes.SettingSource_SETTING_SOURCE_PROJECT:
		return SettingSourceProject
	case sdktypes.SettingSource_SETTING_SOURCE_LOCAL:
		return SettingSourceLocal
	default:
		return ""
	}
}

func (e SettingSource) ToProto() sdktypes.SettingSource {
	switch e {
	case SettingSourceUser:
		return sdktypes.SettingSource_SETTING_SOURCE_USER
	case SettingSourceProject:
		return sdktypes.SettingSource_SETTING_SOURCE_PROJECT
	case SettingSourceLocal:
		return sdktypes.SettingSource_SETTING_SOURCE_LOCAL
	default:
		return sdktypes.SettingSource_SETTING_SOURCE_UNSPECIFIED
	}
}

// AgentModel converters

func AgentModelFromProto(pb sdktypes.AgentModel) AgentModel {
	switch pb {
	case sdktypes.AgentModel_AGENT_MODEL_SONNET:
		return AgentModelSonnet
	case sdktypes.AgentModel_AGENT_MODEL_OPUS:
		return AgentModelOpus
	case sdktypes.AgentModel_AGENT_MODEL_HAIKU:
		return AgentModelHaiku
	case sdktypes.AgentModel_AGENT_MODEL_INHERIT:
		return AgentModelInherit
	default:
		return ""
	}
}

func (e AgentModel) ToProto() sdktypes.AgentModel {
	switch e {
	case AgentModelSonnet:
		return sdktypes.AgentModel_AGENT_MODEL_SONNET
	case AgentModelOpus:
		return sdktypes.AgentModel_AGENT_MODEL_OPUS
	case AgentModelHaiku:
		return sdktypes.AgentModel_AGENT_MODEL_HAIKU
	case AgentModelInherit:
		return sdktypes.AgentModel_AGENT_MODEL_INHERIT
	default:
		return sdktypes.AgentModel_AGENT_MODEL_UNSPECIFIED
	}
}
