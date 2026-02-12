package tool_iov1

import "fmt"

func (x HookEvent) MarshalText() ([]byte, error) {
	switch x {
	case HookEvent_HOOK_EVENT_PRE_TOOL_USE:
		return []byte("PreToolUse"), nil
	case HookEvent_HOOK_EVENT_POST_TOOL_USE:
		return []byte("PostToolUse"), nil
	case HookEvent_HOOK_EVENT_POST_TOOL_USE_FAILURE:
		return []byte("PostToolUseFailure"), nil
	case HookEvent_HOOK_EVENT_NOTIFICATION:
		return []byte("Notification"), nil
	case HookEvent_HOOK_EVENT_USER_PROMPT_SUBMIT:
		return []byte("UserPromptSubmit"), nil
	case HookEvent_HOOK_EVENT_SESSION_START:
		return []byte("SessionStart"), nil
	case HookEvent_HOOK_EVENT_SESSION_END:
		return []byte("SessionEnd"), nil
	case HookEvent_HOOK_EVENT_STOP:
		return []byte("Stop"), nil
	case HookEvent_HOOK_EVENT_SUBAGENT_START:
		return []byte("SubagentStart"), nil
	case HookEvent_HOOK_EVENT_SUBAGENT_STOP:
		return []byte("SubagentStop"), nil
	case HookEvent_HOOK_EVENT_PRE_COMPACT:
		return []byte("PreCompact"), nil
	case HookEvent_HOOK_EVENT_PERMISSION_REQUEST:
		return []byte("PermissionRequest"), nil
	default:
		return nil, nil
	}
}

func (x *HookEvent) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = HookEvent_HOOK_EVENT_UNSPECIFIED
	case "PreToolUse":
		*x = HookEvent_HOOK_EVENT_PRE_TOOL_USE
	case "PostToolUse":
		*x = HookEvent_HOOK_EVENT_POST_TOOL_USE
	case "PostToolUseFailure":
		*x = HookEvent_HOOK_EVENT_POST_TOOL_USE_FAILURE
	case "Notification":
		*x = HookEvent_HOOK_EVENT_NOTIFICATION
	case "UserPromptSubmit":
		*x = HookEvent_HOOK_EVENT_USER_PROMPT_SUBMIT
	case "SessionStart":
		*x = HookEvent_HOOK_EVENT_SESSION_START
	case "SessionEnd":
		*x = HookEvent_HOOK_EVENT_SESSION_END
	case "Stop":
		*x = HookEvent_HOOK_EVENT_STOP
	case "SubagentStart":
		*x = HookEvent_HOOK_EVENT_SUBAGENT_START
	case "SubagentStop":
		*x = HookEvent_HOOK_EVENT_SUBAGENT_STOP
	case "PreCompact":
		*x = HookEvent_HOOK_EVENT_PRE_COMPACT
	case "PermissionRequest":
		*x = HookEvent_HOOK_EVENT_PERMISSION_REQUEST
	default:
		return fmt.Errorf("unknown HookEvent text: %q", string(b))
	}
	return nil
}

func (x SessionStartSource) MarshalText() ([]byte, error) {
	switch x {
	case SessionStartSource_SESSION_START_SOURCE_STARTUP:
		return []byte("startup"), nil
	case SessionStartSource_SESSION_START_SOURCE_RESUME:
		return []byte("resume"), nil
	case SessionStartSource_SESSION_START_SOURCE_CLEAR:
		return []byte("clear"), nil
	case SessionStartSource_SESSION_START_SOURCE_COMPACT:
		return []byte("compact"), nil
	default:
		return nil, nil
	}
}

func (x *SessionStartSource) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = SessionStartSource_SESSION_START_SOURCE_UNSPECIFIED
	case "startup":
		*x = SessionStartSource_SESSION_START_SOURCE_STARTUP
	case "resume":
		*x = SessionStartSource_SESSION_START_SOURCE_RESUME
	case "clear":
		*x = SessionStartSource_SESSION_START_SOURCE_CLEAR
	case "compact":
		*x = SessionStartSource_SESSION_START_SOURCE_COMPACT
	default:
		return fmt.Errorf("unknown SessionStartSource text: %q", string(b))
	}
	return nil
}

func (x PreCompactTrigger) MarshalText() ([]byte, error) {
	switch x {
	case PreCompactTrigger_PRE_COMPACT_TRIGGER_MANUAL:
		return []byte("manual"), nil
	case PreCompactTrigger_PRE_COMPACT_TRIGGER_AUTO:
		return []byte("auto"), nil
	default:
		return nil, nil
	}
}

func (x *PreCompactTrigger) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = PreCompactTrigger_PRE_COMPACT_TRIGGER_UNSPECIFIED
	case "manual":
		*x = PreCompactTrigger_PRE_COMPACT_TRIGGER_MANUAL
	case "auto":
		*x = PreCompactTrigger_PRE_COMPACT_TRIGGER_AUTO
	default:
		return fmt.Errorf("unknown PreCompactTrigger text: %q", string(b))
	}
	return nil
}
