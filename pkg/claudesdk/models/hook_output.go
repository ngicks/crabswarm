package models

import "encoding/json"

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
	Continue           *bool                      `json:"continue,omitempty"`
	SuppressOutput     *bool                      `json:"suppressOutput,omitempty"`
	StopReason         *string                    `json:"stopReason,omitempty"`
	Decision           SyncHookJSONOutputDecision `json:"decision,omitempty"`
	SystemMessage      *string                    `json:"systemMessage,omitempty"`
	Reason             *string                    `json:"reason,omitempty"`
	HookSpecificOutput *HookSpecificOutput        `json:"hookSpecificOutput,omitempty"`
}

type SyncHookJSONOutputDecision string

const (
	SyncHookJSONOutputDecisionApprove = "approve"
	SyncHookJSONOutputDecisionBlock   = "block"
)

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

type HookSpecificOutputPreToolUse struct{
	      hookEventName: "PreToolUse";
        permissionDecision?: "allow" | "deny" | "ask";
        permissionDecisionReason?: string;
        updatedInput?: Record<string, unknown>;
        additionalContext?: string;
}

type HookSpecificOutputUserPromptSubmit struct{
        hookEventName: "UserPromptSubmit";
        additionalContext?: string;
}

type HookSpecificOutputUserSessionStart struct {
        hookEventName: "SessionStart";
        additionalContext?: string;
}
type HookSpecificOutputUserSetup struct {
        hookEventName: "Setup";
        additionalContext?: string;
}
type HookSpecificOutputUserSubagentStart struct {
        hookEventName: "SubagentStart";
        additionalContext?: string;
}
type HookSpecificOutputUserPostToolUse struct {
        hookEventName: "PostToolUse";
        additionalContext?: string;
        updatedMCPToolOutput?: unknown;
}
type HookSpecificOutputUserPostToolUseFailure struct {
        hookEventName: "PostToolUseFailure";
        additionalContext?: string;
}
type HookSpecificOutputUserNotification struct {
        hookEventName: "Notification";
        additionalContext?: string;
}
type HookSpecificOutputUserPermissionRequest struct {
        hookEventName: "PermissionRequest";
        decision: HookSpecificOutputUserPermissionRequestDecision
}

type HookSpecificOutputUserPermissionRequestDecision interface {
hookSpecificOutputUserPermissionRequestDecision()
}


type HookSpecificOutputUserPermissionRequestDecisionAllow struct{	
              behavior: "allow";
              updatedInput?: Record<string, unknown>;
              updatedPermissions?: PermissionUpdate[];
}

type HookSpecificOutputUserPermissionRequestDecisionDeny struct{	
              behavior: "deny";
              message?: string;
              interrupt?: boolean;
}



