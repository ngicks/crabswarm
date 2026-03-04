// Package handler defines handler for claude hook
package handler

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ngicks/crabswarm/pkg/claudesdk/models"
)

// --- Typed string constants ---

// HookDecision is the top-level "decision" field value.
// Used by: UserPromptSubmit, PostToolUse, PostToolUseFailure, Stop, SubagentStop, ConfigChange.
// See https://code.claude.com/docs/en/hooks#decision-control
type HookDecision string

const (
	HookDecisionBlock HookDecision = "block"
)

// PermissionDecision is the "permissionDecision" field inside hookSpecificOutput for PreToolUse.
// See https://code.claude.com/docs/en/hooks#pretooluse-decision-control
type PermissionDecision string

const (
	PermissionDecisionAllow PermissionDecision = "allow"
	PermissionDecisionDeny  PermissionDecision = "deny"
	PermissionDecisionAsk   PermissionDecision = "ask"
)

// PermissionRequestBehavior is the "behavior" field inside hookSpecificOutput.decision for PermissionRequest.
// See https://code.claude.com/docs/en/hooks#permissionrequest-decision-control
type PermissionRequestBehavior string

const (
	PermissionRequestBehaviorAllow PermissionRequestBehavior = "allow"
	PermissionRequestBehaviorDeny  PermissionRequestBehavior = "deny"
)

// HandlerError is returned by hook subcommands to signal the hook result.
// It implements error as it requires special handling.
type HandlerError struct {
	// Output is the structured JSON to write to stdout on exit 0.
	Output *models.SyncHookJSONOutput
}

func (e *HandlerError) Error() string {
	if e.Output != nil && e.Output.Decision != nil && *e.Output.Decision == models.SyncHookJSONOutputDecision(HookDecisionBlock) {
		reason := ""
		if e.Output.Reason != nil {
			reason = *e.Output.Reason
		}
		return fmt.Sprintf("hook: block: %s", reason)
	}
	return "hook: allow"
}

// Handle writes output and exits the process.
func (e *HandlerError) Handle() {
	if e.Output == nil {
		os.Exit(0)
	}

	if e.Output.Decision != nil && *e.Output.Decision == models.SyncHookJSONOutputDecision(HookDecisionBlock) {
		if e.Output.Reason != nil {
			fmt.Fprint(os.Stderr, *e.Output.Reason)
		}
		os.Exit(2)
	}

	// JSON output to stdout, exit 0
	data, err := json.Marshal(e.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hook: marshaling SyncHookJSONOutput: json.Marshal: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, string(data))
	os.Exit(0)
}

// NewPermissionRequestAllowError creates a HandlerError that outputs a PermissionRequest
// approval response (hookSpecificOutput with decision.behavior = "allow").
func NewPermissionRequestAllowError() *HandlerError {
	return &HandlerError{
		Output: &models.SyncHookJSONOutput{
			HookSpecificOutput: models.HookSpecificOutputPermissionRequest{
				HookEventName: models.HookEventNamePermissionRequest,
				Decision: models.PermissionRequestDecisionAllow{
					Behavior: models.PermissionRequestBehavior(PermissionRequestBehaviorAllow),
				},
			},
		},
	}
}
