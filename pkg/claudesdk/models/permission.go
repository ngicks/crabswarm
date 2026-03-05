package models

import (
	"encoding/json"
	"fmt"
	"slices"
)

// As per https://platform.claude.com/docs/en/agent-sdk/typescript#permission-update
/*
type PermissionUpdate =
  | {
      type: "addRules";
      rules: PermissionRuleValue[];
      behavior: PermissionBehavior;
      destination: PermissionUpdateDestination;
    }
  | {
      type: "replaceRules";
      rules: PermissionRuleValue[];
      behavior: PermissionBehavior;
      destination: PermissionUpdateDestination;
    }
  | {
      type: "removeRules";
      rules: PermissionRuleValue[];
      behavior: PermissionBehavior;
      destination: PermissionUpdateDestination;
    }
  | {
      type: "setMode";
      mode: PermissionMode;
      destination: PermissionUpdateDestination;
    }
  | {
      type: "addDirectories";
      directories: string[];
      destination: PermissionUpdateDestination;
    }
  | {
      type: "removeDirectories";
      directories: string[];
      destination: PermissionUpdateDestination;
    };
*/

type PermissionUpdate interface {
	permissionUpdate()
	json.Marshaler
	json.Unmarshaler
}

func (*PermissionUpdateAddRules) permissionUpdate()          {}
func (*PermissionUpdateReplaceRules) permissionUpdate()      {}
func (*PermissionUpdateRemoveRules) permissionUpdate()       {}
func (*PermissionUpdateSetMode) permissionUpdate()           {}
func (*PermissionUpdateAddDirectories) permissionUpdate()    {}
func (*PermissionUpdateRemoveDirectories) permissionUpdate() {}

func unmarshalPermissionUpdate(data []byte) (_ PermissionUpdate, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("PermissionUpdate: %w", err)
		}
	}()

	var d struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}

	switch d.Type {
	default:
		return nil, fmt.Errorf("unknown type: %q", d.Type)
	case "":
		return nil, fmt.Errorf("empty type")
	case "addRules":
		var v PermissionUpdateAddRules
		err := json.Unmarshal(data, &v)
		return &v, err
	case "replaceRules":
		var v PermissionUpdateReplaceRules
		err := json.Unmarshal(data, &v)
		return &v, err
	case "removeRules":
		var v PermissionUpdateRemoveRules
		err := json.Unmarshal(data, &v)
		return &v, err
	case "setMode":
		var v PermissionUpdateSetMode
		err := json.Unmarshal(data, &v)
		return &v, err
	case "addDirectories":
		var v PermissionUpdateAddDirectories
		err := json.Unmarshal(data, &v)
		return &v, err
	case "removeDirectories":
		var v PermissionUpdateRemoveDirectories
		err := json.Unmarshal(data, &v)
		return &v, err
	}
}

// PermissionUpdateAddRules is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateAddRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (o PermissionUpdateAddRules) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateAddRules
	o.Type = "addRules"
	return json.Marshal(plain(o))
}

func (o *PermissionUpdateAddRules) UnmarshalJSON(data []byte) error {
	type plain PermissionUpdateAddRules
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = PermissionUpdateAddRules(p)
	o.Type = "addRules"
	return nil
}

// PermissionUpdateReplaceRules is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateReplaceRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (o PermissionUpdateReplaceRules) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateReplaceRules
	o.Type = "replaceRules"
	return json.Marshal(plain(o))
}

func (o *PermissionUpdateReplaceRules) UnmarshalJSON(data []byte) error {
	type plain PermissionUpdateReplaceRules
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = PermissionUpdateReplaceRules(p)
	o.Type = "replaceRules"
	return nil
}

// PermissionUpdateRemoveRules is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateRemoveRules struct {
	Type        string                      `json:"type"`
	Rules       []PermissionRuleValue       `json:"rules"`
	Behavior    PermissionBehavior          `json:"behavior"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (o PermissionUpdateRemoveRules) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateRemoveRules
	o.Type = "removeRules"
	return json.Marshal(plain(o))
}

func (o *PermissionUpdateRemoveRules) UnmarshalJSON(data []byte) error {
	type plain PermissionUpdateRemoveRules
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = PermissionUpdateRemoveRules(p)
	o.Type = "removeRules"
	return nil
}

// PermissionUpdateSetMode is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateSetMode struct {
	Type        string                      `json:"type"`
	Mode        PermissionMode              `json:"mode"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (o PermissionUpdateSetMode) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateSetMode
	o.Type = "setMode"
	return json.Marshal(plain(o))
}

func (o *PermissionUpdateSetMode) UnmarshalJSON(data []byte) error {
	type plain PermissionUpdateSetMode
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = PermissionUpdateSetMode(p)
	o.Type = "setMode"
	return nil
}

// PermissionUpdateAddDirectories is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateAddDirectories struct {
	Type        string                      `json:"type"`
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (o PermissionUpdateAddDirectories) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateAddDirectories
	o.Type = "addDirectories"
	return json.Marshal(plain(o))
}

func (o *PermissionUpdateAddDirectories) UnmarshalJSON(data []byte) error {
	type plain PermissionUpdateAddDirectories
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = PermissionUpdateAddDirectories(p)
	o.Type = "addDirectories"
	return nil
}

// PermissionUpdateRemoveDirectories is direct translation of https://platform.claude.com/docs/en/agent-sdk/typescript#permission-update
type PermissionUpdateRemoveDirectories struct {
	Type        string                      `json:"type"`
	Directories []string                    `json:"directories"`
	Destination PermissionUpdateDestination `json:"destination"`
}

func (o PermissionUpdateRemoveDirectories) MarshalJSON() ([]byte, error) {
	type plain PermissionUpdateRemoveDirectories
	o.Type = "removeDirectories"
	return json.Marshal(plain(o))
}

func (o *PermissionUpdateRemoveDirectories) UnmarshalJSON(data []byte) error {
	type plain PermissionUpdateRemoveDirectories
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*o = PermissionUpdateRemoveDirectories(p)
	o.Type = "removeDirectories"
	return nil
}

// PermissionBehavior represents the behavior of a permission rule.
type PermissionBehavior string

const (
	PermissionBehaviorAllow PermissionBehavior = "allow"
	PermissionBehaviorDeny  PermissionBehavior = "deny"
	PermissionBehaviorAsk   PermissionBehavior = "ask"
)

var allPermissionBehavior = [...]PermissionBehavior{
	PermissionBehaviorAllow,
	PermissionBehaviorDeny,
	PermissionBehaviorAsk,
}

func IsPermissionBehavior(s string) bool {
	return slices.Contains(
		allPermissionBehavior[:],
		PermissionBehavior(s),
	)
}

// PermissionMode represents a permission mode.
type PermissionMode string

const (
	PermissionModeDefault   PermissionMode = "default"
	PermissionModePlan      PermissionMode = "plan"
	PermissionModeBypassAll PermissionMode = "bypassAll"
)

var allPermissionMode = [...]PermissionMode{
	PermissionModeDefault,
	PermissionModePlan,
	PermissionModeBypassAll,
}

func IsPermissionMode(s string) bool {
	return slices.Contains(
		allPermissionMode[:],
		PermissionMode(s),
	)
}

// PermissionUpdateDestination represents where a permission update is applied.
type PermissionUpdateDestination string

const (
	PermissionUpdateDestinationUserSettings    PermissionUpdateDestination = "userSettings"    // Global user settings
	PermissionUpdateDestinationProjectSettings PermissionUpdateDestination = "projectSettings" // Per-directory project settings
	PermissionUpdateDestinationLocalSettings   PermissionUpdateDestination = "localSettings"   // Gitignored local settings
	PermissionUpdateDestinationSession         PermissionUpdateDestination = "session"          // Current session only
	PermissionUpdateDestinationCliArg          PermissionUpdateDestination = "cliArg"           // CLI argument
)

var allPermissionUpdateDestination = [...]PermissionUpdateDestination{
	PermissionUpdateDestinationUserSettings,
	PermissionUpdateDestinationProjectSettings,
	PermissionUpdateDestinationLocalSettings,
	PermissionUpdateDestinationSession,
	PermissionUpdateDestinationCliArg,
}

func IsPermissionUpdateDestination(s string) bool {
	return slices.Contains(
		allPermissionUpdateDestination[:],
		PermissionUpdateDestination(s),
	)
}

// PermissionRuleValue represents a single permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"tool_name"`
	RuleContent *string `json:"rule_content,omitempty"`
}
