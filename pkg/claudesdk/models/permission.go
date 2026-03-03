package models

import "encoding/json"

// PermissionRuleValue represents a single permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"tool_name"`
	RuleContent *string `json:"rule_content,omitempty"`
}

// PermissionUpdate is a flattened oneof representation for permission updates.
type PermissionUpdate struct {
	AddRules          *AddRulesUpdate          `json:"add_rules,omitempty"`
	ReplaceRules      *ReplaceRulesUpdate      `json:"replace_rules,omitempty"`
	RemoveRules       *RemoveRulesUpdate       `json:"remove_rules,omitempty"`
	SetMode           *SetModeUpdate           `json:"set_mode,omitempty"`
	AddDirectories    *AddDirectoriesUpdate    `json:"add_directories,omitempty"`
	RemoveDirectories *RemoveDirectoriesUpdate `json:"remove_directories,omitempty"`
}

func (p *PermissionUpdate) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if v, ok := raw["add_rules"]; ok {
		var x AddRulesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.AddRules = &x
	}
	if v, ok := raw["addRules"]; ok {
		var x AddRulesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.AddRules = &x
	}
	if v, ok := raw["replace_rules"]; ok {
		var x ReplaceRulesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.ReplaceRules = &x
	}
	if v, ok := raw["replaceRules"]; ok {
		var x ReplaceRulesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.ReplaceRules = &x
	}
	if v, ok := raw["remove_rules"]; ok {
		var x RemoveRulesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.RemoveRules = &x
	}
	if v, ok := raw["removeRules"]; ok {
		var x RemoveRulesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.RemoveRules = &x
	}
	if v, ok := raw["set_mode"]; ok {
		var x SetModeUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.SetMode = &x
	}
	if v, ok := raw["setMode"]; ok {
		var x SetModeUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.SetMode = &x
	}
	if v, ok := raw["add_directories"]; ok {
		var x AddDirectoriesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.AddDirectories = &x
	}
	if v, ok := raw["addDirectories"]; ok {
		var x AddDirectoriesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.AddDirectories = &x
	}
	if v, ok := raw["remove_directories"]; ok {
		var x RemoveDirectoriesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.RemoveDirectories = &x
	}
	if v, ok := raw["removeDirectories"]; ok {
		var x RemoveDirectoriesUpdate
		if err := json.Unmarshal(v, &x); err != nil {
			return err
		}
		p.RemoveDirectories = &x
	}
	return nil
}

// AddRulesUpdate adds permission rules.
type AddRulesUpdate struct {
	Rules       []PermissionRuleValue `json:"rules"`
	Behavior    string                `json:"behavior"`
	Destination string                `json:"destination"`
}

// ReplaceRulesUpdate replaces permission rules.
type ReplaceRulesUpdate struct {
	Rules       []PermissionRuleValue `json:"rules"`
	Behavior    string                `json:"behavior"`
	Destination string                `json:"destination"`
}

// RemoveRulesUpdate removes permission rules.
type RemoveRulesUpdate struct {
	Rules       []PermissionRuleValue `json:"rules"`
	Behavior    string                `json:"behavior"`
	Destination string                `json:"destination"`
}

// SetModeUpdate sets permission mode.
type SetModeUpdate struct {
	Mode        string `json:"mode"`
	Destination string `json:"destination"`
}

// AddDirectoriesUpdate adds allowed directories.
type AddDirectoriesUpdate struct {
	Directories []string `json:"directories"`
	Destination string   `json:"destination"`
}

// RemoveDirectoriesUpdate removes allowed directories.
type RemoveDirectoriesUpdate struct {
	Directories []string `json:"directories"`
	Destination string   `json:"destination"`
}
