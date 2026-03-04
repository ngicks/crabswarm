package models

import (
	"encoding/json"
	"fmt"
	"slices"
)

// AsyncHookJSONOutput represents the output of an async hook.
// https://platform.claude.com/docs/en/agent-sdk/typescript#async-hook-json-output
type AsyncHookJSONOutput struct {
	// always true.
	Async        bool   `json:"async"`
	AsyncTimeout *int32 `json:"asyncTimeout,omitempty"`
}

// SyncHookJSONOutput represents the output of a synchronous hook.
// https://platform.claude.com/docs/en/agent-sdk/typescript#sync-hook-json-output
type SyncHookJSONOutput struct {
	Continue           *bool                       `json:"continue,omitempty"`
	SuppressOutput     *bool                       `json:"suppressOutput,omitempty"`
	StopReason         *string                     `json:"stopReason,omitempty"`
	Decision           *SyncHookJSONOutputDecision `json:"decision,omitempty"`
	SystemMessage      *string                     `json:"systemMessage,omitempty"`
	Reason             *string                     `json:"reason,omitempty"`
	HookSpecificOutput HookSpecificOutput          `json:"hookSpecificOutput,omitempty"`
}

func (s *SyncHookJSONOutput) UnmarshalJSON(data []byte) error {
	type alias struct {
		Continue           *bool                       `json:"continue,omitempty"`
		SuppressOutput     *bool                       `json:"suppressOutput,omitempty"`
		StopReason         *string                     `json:"stopReason,omitempty"`
		Decision           *SyncHookJSONOutputDecision `json:"decision,omitempty"`
		SystemMessage      *string                     `json:"systemMessage,omitempty"`
		Reason             *string                     `json:"reason,omitempty"`
		HookSpecificOutput json.RawMessage             `json:"hookSpecificOutput,omitempty"`
	}

	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	s.Continue = a.Continue
	s.SuppressOutput = a.SuppressOutput
	s.StopReason = a.StopReason
	s.Decision = a.Decision
	s.SystemMessage = a.SystemMessage
	s.Reason = a.Reason

	if len(a.HookSpecificOutput) > 0 {
		hso, err := unmarshalHookSpecificOutput(a.HookSpecificOutput)
		if err != nil {
			return err
		}
		s.HookSpecificOutput = hso
	}

	return nil
}

type SyncHookJSONOutputDecision string

const (
	SyncHookJSONOutputDecisionApprove SyncHookJSONOutputDecision = "approve"
	SyncHookJSONOutputDecisionBlock   SyncHookJSONOutputDecision = "block"
)

func IsSyncHookJSONOutputDecision(s string) bool {
	return slices.Contains(
		[]SyncHookJSONOutputDecision{
			SyncHookJSONOutputDecisionApprove,
			SyncHookJSONOutputDecisionBlock,
		},
		SyncHookJSONOutputDecision(s),
	)
}

func (d SyncHookJSONOutputDecision) MarshalJSON() ([]byte, error) {
	if !IsSyncHookJSONOutputDecision(string(d)) {
		return nil, fmt.Errorf("SyncHookJSONOutputDecision: unknown value: %q", d)
	}
	return []byte(d), nil
}

func (d *SyncHookJSONOutputDecision) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	if !IsSyncHookJSONOutputDecision(s) {
		return fmt.Errorf("SyncHookJSONOutputDecision: unknown value: %q", s)
	}
	*d = SyncHookJSONOutputDecision(s)
	return nil
}

/*
   | {
        hookEventName: "PreToolUse";
        permissionDecision?: "allow" | "deny" | "ask";
        permissionDecisionReason?: string;
        updatedInput?: Record<string, unknown>;
        additionalContext?: string;
      }
    | {
        hookEventName: "UserPromptSubmit";
        additionalContext?: string;
      }
    | {
        hookEventName: "SessionStart";
        additionalContext?: string;
      }
    | {
        hookEventName: "Setup";
        additionalContext?: string;
      }
    | {
        hookEventName: "SubagentStart";
        additionalContext?: string;
      }
    | {
        hookEventName: "PostToolUse";
        additionalContext?: string;
        updatedMCPToolOutput?: unknown;
      }
    | {
        hookEventName: "PostToolUseFailure";
        additionalContext?: string;
      }
    | {
        hookEventName: "Notification";
        additionalContext?: string;
      }
    | {
        hookEventName: "PermissionRequest";
        decision:
          | {
              behavior: "allow";
              updatedInput?: Record<string, unknown>;
              updatedPermissions?: PermissionUpdate[];
            }
          | {
              behavior: "deny";
              message?: string;
              interrupt?: boolean;
            };
      };
*/

type HookSpecificOutput interface {
	hookSpecificOutput()
}

func (PreToolUseHookSpecificOutput) hookSpecificOutput()         {}
func (UserPromptSubmitHookSpecificOutput) hookSpecificOutput()   {}
func (SessionStartHookSpecificOutput) hookSpecificOutput()       {}
func (SetupHookSpecificOutput) hookSpecificOutput()              {}
func (SubagentStartHookSpecificOutput) hookSpecificOutput()      {}
func (PostToolUseHookSpecificOutput) hookSpecificOutput()        {}
func (PostToolUseFailureHookSpecificOutput) hookSpecificOutput() {}
func (NotificationHookSpecificOutput) hookSpecificOutput()       {}
func (PermissionRequestHookSpecificOutput) hookSpecificOutput()  {}

type HookEventName string

const (
	HookEventNamePreToolUse         HookEventName = "PreToolUse"
	HookEventNameUserPromptSubmit   HookEventName = "UserPromptSubmit"
	HookEventNameSessionStart       HookEventName = "SessionStart"
	HookEventNameSetup              HookEventName = "Setup"
	HookEventNameSubagentStart      HookEventName = "SubagentStart"
	HookEventNamePostToolUse        HookEventName = "PostToolUse"
	HookEventNamePostToolUseFailure HookEventName = "PostToolUseFailure"
	HookEventNameNotification       HookEventName = "Notification"
	HookEventNamePermissionRequest  HookEventName = "PermissionRequest"
)

type PreToolUseHookSpecificOutput struct {
	HookEventName            HookEventName       `json:"hookEventName"`
	PermissionDecision       *PermissionDecision `json:"permissionDecision,omitempty"`
	PermissionDecisionReason *string             `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             json.RawMessage     `json:"updatedInput,omitempty"`
	AdditionalContext        *string             `json:"additionalContext,omitempty"`
}

type PermissionDecision string

const (
	PermissionDecisionAllow PermissionDecision = "allow"
	PermissionDecisionDeny  PermissionDecision = "deny"
	PermissionDecisionAsk   PermissionDecision = "ask"
)

type UserPromptSubmitHookSpecificOutput struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

type SessionStartHookSpecificOutput struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

type SetupHookSpecificOutput struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

type SubagentStartHookSpecificOutput struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

type PostToolUseHookSpecificOutput struct {
	HookEventName        HookEventName   `json:"hookEventName"`
	AdditionalContext    *string         `json:"additionalContext,omitempty"`
	UpdatedMCPToolOutput json.RawMessage `json:"updatedMCPToolOutput,omitempty"`
}

type PostToolUseFailureHookSpecificOutput struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

type NotificationHookSpecificOutput struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

type PermissionRequestHookSpecificOutput struct {
	HookEventName HookEventName             `json:"hookEventName"`
	Decision      PermissionRequestDecision `json:"decision,omitempty"`
}

func (p PermissionRequestHookSpecificOutput) MarshalJSON() ([]byte, error) {
	type alias struct {
		HookEventName HookEventName   `json:"hookEventName"`
		Decision      json.RawMessage `json:"decision,omitempty"`
	}

	var decision json.RawMessage
	if p.Decision != nil {
		b, err := json.Marshal(p.Decision)
		if err != nil {
			return nil, err
		}
		decision = b
	}

	return json.Marshal(alias{HookEventName: p.HookEventName, Decision: decision})
}

func (p *PermissionRequestHookSpecificOutput) UnmarshalJSON(data []byte) error {
	type alias struct {
		HookEventName HookEventName   `json:"hookEventName"`
		Decision      json.RawMessage `json:"decision,omitempty"`
	}

	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	p.HookEventName = a.HookEventName

	if len(a.Decision) > 0 {
		decision, err := unmarshalPermissionRequestDecision(a.Decision)
		if err != nil {
			return err
		}
		p.Decision = decision
	}

	return nil
}

type PermissionRequestBehavior string

type PermissionRequestDecision interface {
	permissionRequestDecision()
}

func (PermissionRequestDecisionAllow) permissionRequestDecision() {}
func (PermissionRequestDecisionDeny) permissionRequestDecision()  {}

type PermissionRequestDecisionAllow struct {
	Behavior           PermissionRequestBehavior `json:"behavior"`
	UpdatedInput       json.RawMessage           `json:"updatedInput,omitempty"`
	UpdatedPermissions []PermissionUpdate        `json:"updatedPermissions,omitempty"`
}

type PermissionRequestDecisionDeny struct {
	Behavior  PermissionRequestBehavior `json:"behavior"`
	Message   *string                   `json:"message,omitempty"`
	Interrupt *bool                     `json:"interrupt,omitempty"`
}

const (
	PermissionRequestBehaviorAllow PermissionRequestBehavior = "allow"
	PermissionRequestBehaviorDeny  PermissionRequestBehavior = "deny"
)

func unmarshalHookSpecificOutput(data []byte) (HookSpecificOutput, error) {
	var d struct {
		HookEventName HookEventName `json:"hookEventName"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	switch d.HookEventName {
	case HookEventNamePreToolUse:
		var v PreToolUseHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNameUserPromptSubmit:
		var v UserPromptSubmitHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNameSessionStart:
		var v SessionStartHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNameSetup:
		var v SetupHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNameSubagentStart:
		var v SubagentStartHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNamePostToolUse:
		var v PostToolUseHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNamePostToolUseFailure:
		var v PostToolUseFailureHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNameNotification:
		var v NotificationHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case HookEventNamePermissionRequest:
		var v PermissionRequestHookSpecificOutput
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unknown hookEventName: %q", d.HookEventName)
	}
}

func unmarshalPermissionRequestDecision(data []byte) (PermissionRequestDecision, error) {
	var d struct {
		Behavior PermissionRequestBehavior `json:"behavior"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	switch d.Behavior {
	case PermissionRequestBehaviorAllow:
		var v PermissionRequestDecisionAllow
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	case PermissionRequestBehaviorDeny:
		var v PermissionRequestDecisionDeny
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		return v, nil
	default:
		return nil, fmt.Errorf("unknown permission request behavior: %q", d.Behavior)
	}
}
