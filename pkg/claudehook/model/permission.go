package model

// PermissionRuleValue represents a permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"tool_name"`
	RuleContent *string `json:"rule_content,omitempty"`
}

// PermissionUpdate is a union type for permission updates.
type PermissionUpdate struct {
	AddRules          *AddRulesUpdate          `json:"add_rules,omitempty"`
	ReplaceRules      *ReplaceRulesUpdate      `json:"replace_rules,omitempty"`
	RemoveRules       *RemoveRulesUpdate       `json:"remove_rules,omitempty"`
	SetMode           *SetModeUpdate           `json:"set_mode,omitempty"`
	AddDirectories    *AddDirectoriesUpdate    `json:"add_directories,omitempty"`
	RemoveDirectories *RemoveDirectoriesUpdate `json:"remove_directories,omitempty"`
}

// AddRulesUpdate represents an add-rules permission update.
type AddRulesUpdate struct {
	Rules       []*PermissionRuleValue `json:"rules,omitempty"`
	Behavior    string                 `json:"behavior"`
	Destination string                 `json:"destination"`
}

// ReplaceRulesUpdate represents a replace-rules permission update.
type ReplaceRulesUpdate struct {
	Rules       []*PermissionRuleValue `json:"rules,omitempty"`
	Behavior    string                 `json:"behavior"`
	Destination string                 `json:"destination"`
}

// RemoveRulesUpdate represents a remove-rules permission update.
type RemoveRulesUpdate struct {
	Rules       []*PermissionRuleValue `json:"rules,omitempty"`
	Behavior    string                 `json:"behavior"`
	Destination string                 `json:"destination"`
}

// SetModeUpdate represents a set-mode permission update.
type SetModeUpdate struct {
	Mode        string `json:"mode"`
	Destination string `json:"destination"`
}

// AddDirectoriesUpdate represents an add-directories permission update.
type AddDirectoriesUpdate struct {
	Directories []string `json:"directories,omitempty"`
	Destination string   `json:"destination"`
}

// RemoveDirectoriesUpdate represents a remove-directories permission update.
type RemoveDirectoriesUpdate struct {
	Directories []string `json:"directories,omitempty"`
	Destination string   `json:"destination"`
}
