package models

import "encoding/json"

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
}

func (PermissionUpdateAddRules) permissionUpdate() {}
func (PermissionUpdateReplaceRules) permissionUpdate() {}
func (PermissionUpdateRemoveRules) permissionUpdate() {}
func (PermissionUpdateSetMode) permissionUpdate() {}
func (PermissionUpdateAddDirectories) permissionUpdate() {}
func (PermissionUpdateRemoveDirectories) permissionUpdate() {}

func unmarshalPermissionUpdate(data []byte) (_ PermissionUpdate, err error) {}

type PermissionUpdateAddRules struct {
type: "addRules";
      rules: PermissionRuleValue[];
      behavior: PermissionBehavior;
      destination: PermissionUpdateDestination;
    }
  type PermissionUpdateReplaceRules  struct {
      type: "replaceRules";
      rules: PermissionRuleValue[];
      behavior: PermissionBehavior;
      destination: PermissionUpdateDestination;
    }
type PermissionUpdateRemoveRules  struct {
      type: "removeRules";
      rules: PermissionRuleValue[];
      behavior: PermissionBehavior;
      destination: PermissionUpdateDestination;
    }
 type PermissionUpdateSetMode struct {
      type: "setMode";
      mode: PermissionMode;
      destination: PermissionUpdateDestination;
    }
 type PermissionUpdateAddDirectories  struct {
      type: "addDirectories";
      directories: string[];
      destination: PermissionUpdateDestination;
    }
type PermissionUpdateRemoveDirectories  struct {
      type: "removeDirectories";
      directories: string[];
      destination: PermissionUpdateDestination;
    }

 type PermissionBehavior string

 const (
	 PermissionBehaviorAllow PermissionBehavior = "allow" 
	 PermissionBehaviorDeny PermissionBehavior = "deny"
	 PermissionBehaviorAsk PermissionBehavior = "ask"
)

type PermissionUpdateDestination string 
const (
	PermissionUpdateDestinationUserSettings PermissionUpdateDestination = "userSettings" // Global user settings
PermissionUpdateDestinationProjectSettings  PermissionUpdateDestination= "projectSettings" // Per-directory project settings
PermissionUpdateDestinationLocalSettings PermissionUpdateDestination=  "localSettings" // Gitignored local settings
PermissionUpdateDestinationSession PermissionUpdateDestination=    "session" // Current session only
PermissionUpdateDestinationCliArg PermissionUpdateDestination=  "cliArg"; // CLI argument
)

// PermissionRuleValue represents a single permission rule.
type PermissionRuleValue struct {
	ToolName    string  `json:"tool_name"`
	RuleContent *string `json:"rule_content,omitempty"`
}


