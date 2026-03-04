package models

import (
	"encoding/json"
	"fmt"
	"maps"
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
	type raw struct {
		Continue           *bool                       `json:"continue,omitempty"`
		SuppressOutput     *bool                       `json:"suppressOutput,omitempty"`
		StopReason         *string                     `json:"stopReason,omitempty"`
		Decision           *SyncHookJSONOutputDecision `json:"decision,omitempty"`
		SystemMessage      *string                     `json:"systemMessage,omitempty"`
		Reason             *string                     `json:"reason,omitempty"`
		HookSpecificOutput json.RawMessage             `json:"hookSpecificOutput,omitempty"`
	}

	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	s.Continue = r.Continue
	s.SuppressOutput = r.SuppressOutput
	s.StopReason = r.StopReason
	s.Decision = r.Decision
	s.SystemMessage = r.SystemMessage
	s.Reason = r.Reason

	if len(r.HookSpecificOutput) > 0 {
		var err error
		s.HookSpecificOutput, err = unmarshalHookSpecificOutput(r.HookSpecificOutput)
		if err != nil {
			return err
		}
	}

	return nil
}

func unmarshalHookSpecificOutput(data []byte) (_ HookSpecificOutput, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("SyncHookJSONOutput: %w", err)
		}
	}()

	var d struct {
		HookEventName HookEventName `json:"hookEventName"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	switch d.HookEventName {
	default:
		if d.HookEventName == "" {
			return nil, fmt.Errorf("empty hookEventName")
		}
		var v HookSpecificOutputUnknown
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNamePreToolUse:
		var v HookSpecificOutputPreToolUse
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNameUserPromptSubmit:
		var v HookSpecificOutputUserPromptSubmit
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNameSessionStart:
		var v HookSpecificOutputSessionStart
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNameSetup:
		var v HookSpecificOutputSetup
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNameSubagentStart:
		var v HookSpecificOutputSubagentStart
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNamePostToolUse:
		var v HookSpecificOutputPostToolUse
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNamePostToolUseFailure:
		var v HookSpecificOutputPostToolUseFailure
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNameNotification:
		var v HookSpecificOutputNotification
		err := json.Unmarshal(data, &v)
		return v, err
	case HookEventNamePermissionRequest:
		var v HookSpecificOutputPermissionRequest
		err := json.Unmarshal(data, &v)
		return v, err
	}
}

type SyncHookJSONOutputDecision string

const (
	SyncHookJSONOutputDecisionApprove SyncHookJSONOutputDecision = "approve"
	SyncHookJSONOutputDecisionBlock   SyncHookJSONOutputDecision = "block"
)

var allSyncHookJSONOutputDecision = [...]SyncHookJSONOutputDecision{
	SyncHookJSONOutputDecisionApprove,
	SyncHookJSONOutputDecisionBlock,
}

func IsSyncHookJSONOutputDecision(s string) bool {
	return slices.Contains(
		allSyncHookJSONOutputDecision[:],
		SyncHookJSONOutputDecision(s),
	)
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

func (HookSpecificOutputPreToolUse) hookSpecificOutput()         {}
func (HookSpecificOutputUserPromptSubmit) hookSpecificOutput()   {}
func (HookSpecificOutputSessionStart) hookSpecificOutput()       {}
func (HookSpecificOutputSetup) hookSpecificOutput()              {}
func (HookSpecificOutputSubagentStart) hookSpecificOutput()      {}
func (HookSpecificOutputPostToolUse) hookSpecificOutput()        {}
func (HookSpecificOutputPostToolUseFailure) hookSpecificOutput() {}
func (HookSpecificOutputNotification) hookSpecificOutput()       {}
func (HookSpecificOutputPermissionRequest) hookSpecificOutput()  {}

// fallback target

func (HookSpecificOutputUnknown) hookSpecificOutput() {}

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

type HookSpecificOutputPreToolUse struct {
	HookEventName            HookEventName       `json:"hookEventName"`
	PermissionDecision       *PermissionDecision `json:"permissionDecision,omitempty"`
	PermissionDecisionReason *string             `json:"permissionDecisionReason,omitempty"`
	// Partial tool input.
	// Same structure to input but only updated field
	UpdatedInput      json.RawMessage `json:"updatedInput,omitempty"`
	AdditionalContext *string         `json:"additionalContext,omitempty"`
}

func (o HookSpecificOutputPreToolUse) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputPreToolUse
	o.HookEventName = HookEventNamePreToolUse
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputPreToolUse) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputPreToolUse
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputPreToolUse(p)
	o.HookEventName = HookEventNamePreToolUse
	return nil
}

type PermissionDecision string

const (
	PermissionDecisionAllow PermissionDecision = "allow"
	PermissionDecisionDeny  PermissionDecision = "deny"
	PermissionDecisionAsk   PermissionDecision = "ask"
)

var allPermissionDecision = [...]PermissionDecision{
	PermissionDecisionAllow,
	PermissionDecisionDeny,
	PermissionDecisionAsk,
}

func IsPermissionDecision(s string) bool {
	return slices.Contains(
		allPermissionDecision[:],
		PermissionDecision(s),
	)
}

type HookSpecificOutputUserPromptSubmit struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

func (o HookSpecificOutputUserPromptSubmit) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputUserPromptSubmit
	o.HookEventName = HookEventNameUserPromptSubmit
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputUserPromptSubmit) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputUserPromptSubmit
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputUserPromptSubmit(p)
	o.HookEventName = HookEventNameUserPromptSubmit
	return nil
}

type HookSpecificOutputSessionStart struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

func (o HookSpecificOutputSessionStart) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputSessionStart
	o.HookEventName = HookEventNameSessionStart
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputSessionStart) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputSessionStart
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputSessionStart(p)
	o.HookEventName = HookEventNameSessionStart
	return nil
}

type HookSpecificOutputSetup struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

func (o HookSpecificOutputSetup) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputSetup
	o.HookEventName = HookEventNameSetup
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputSetup) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputSetup
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputSetup(p)
	o.HookEventName = HookEventNameSetup
	return nil
}

type HookSpecificOutputSubagentStart struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

func (o HookSpecificOutputSubagentStart) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputSubagentStart
	o.HookEventName = HookEventNameSubagentStart
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputSubagentStart) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputSubagentStart
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputSubagentStart(p)
	o.HookEventName = HookEventNameSubagentStart
	return nil
}

type HookSpecificOutputPostToolUse struct {
	HookEventName        HookEventName   `json:"hookEventName"`
	AdditionalContext    *string         `json:"additionalContext,omitempty"`
	UpdatedMCPToolOutput json.RawMessage `json:"updatedMCPToolOutput,omitempty"`
}

func (o HookSpecificOutputPostToolUse) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputPostToolUse
	o.HookEventName = HookEventNamePostToolUse
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputPostToolUse) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputPostToolUse
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputPostToolUse(p)
	o.HookEventName = HookEventNamePostToolUse
	return nil
}

type HookSpecificOutputPostToolUseFailure struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

func (o HookSpecificOutputPostToolUseFailure) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputPostToolUseFailure
	o.HookEventName = HookEventNamePostToolUseFailure
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputPostToolUseFailure) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputPostToolUseFailure
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputPostToolUseFailure(p)
	o.HookEventName = HookEventNamePostToolUseFailure
	return nil
}

type HookSpecificOutputNotification struct {
	HookEventName     HookEventName `json:"hookEventName"`
	AdditionalContext *string       `json:"additionalContext,omitempty"`
}

func (o HookSpecificOutputNotification) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputNotification
	o.HookEventName = HookEventNameNotification
	return json.Marshal(plain(o))
}

func (o *HookSpecificOutputNotification) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputNotification
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = HookSpecificOutputNotification(p)
	o.HookEventName = HookEventNameNotification
	return nil
}

type HookSpecificOutputPermissionRequest struct {
	HookEventName HookEventName             `json:"hookEventName"`
	Decision      PermissionRequestDecision `json:"decision,omitempty"`
}

func (p HookSpecificOutputPermissionRequest) MarshalJSON() ([]byte, error) {
	type plain HookSpecificOutputPermissionRequest
	p.HookEventName = HookEventNamePermissionRequest
	return json.Marshal(plain(p))
}

func (p *HookSpecificOutputPermissionRequest) UnmarshalJSON(data []byte) error {
	type plain HookSpecificOutputPermissionRequest
	var q plain
	if err := json.Unmarshal(data, &q); err != nil {
		return err
	}
	*p = HookSpecificOutputPermissionRequest(q)
	p.HookEventName = HookEventNamePermissionRequest

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw["decision"]) > 0 {
		var err error
		p.Decision, err = unmarshalPermissionRequestDecision(raw["decision"])
		if err != nil {
			return err
		}
	}

	return nil
}

func unmarshalPermissionRequestDecision(data []byte) (PermissionRequestDecision, error) {
	var d struct {
		Behavior PermissionRequestBehavior `json:"behavior"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	switch d.Behavior {
	default:
		return nil, fmt.Errorf("HookSpecificOutputPermissionRequest: unknown behavior: %q", d.Behavior)
	case PermissionRequestBehaviorAllow:
		var v PermissionRequestDecisionAllow
		err := json.Unmarshal(data, &v)
		if err != nil {
			return nil, fmt.Errorf("HookSpecificOutputPermissionRequest: behavior=allow: %w", err)
		}
		return v, nil
	case PermissionRequestBehaviorDeny:
		var v PermissionRequestDecisionDeny
		err := json.Unmarshal(data, &v)
		if err != nil {
			return nil, fmt.Errorf("HookSpecificOutputPermissionRequest: behavior=deny: %w", err)
		}
		return v, nil
	}
}

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

type PermissionRequestBehavior string

const (
	PermissionRequestBehaviorAllow PermissionRequestBehavior = "allow"
	PermissionRequestBehaviorDeny  PermissionRequestBehavior = "deny"
)

type HookSpecificOutputUnknown struct {
	HookEventName HookEventName `json:"hookEventName"`
	// Rest of message
	Unknown map[string]json.RawMessage
}

func (o HookSpecificOutputUnknown) MarshalJSON() ([]byte, error) {
	m := maps.Clone(o.Unknown)
	if m == nil {
		// panic instead?
		m = map[string]json.RawMessage{}
	}
	raw, _ := json.Marshal(o.HookEventName)
	m["hookEventName"] = json.RawMessage(raw)
	return json.Marshal(m)
}

func (o *HookSpecificOutputUnknown) UnmarshalJSON(data []byte) error {
	unknown := map[string]json.RawMessage{}
	err := json.Unmarshal(data, &unknown)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(unknown["hookEventName"], &o.HookEventName); err != nil {
		return fmt.Errorf("HookSpecificOutputUnknown: %w", err)
	}
	if o.HookEventName == "" {
		return fmt.Errorf("HookSpecificOutputUnknown: empty hookEventName")
	}
	o.Unknown = unknown
	return nil
}
