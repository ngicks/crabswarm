// Package handler defines handler for claude hook
package handler

import (
	"encoding/json"
	"fmt"
	"os"

	sdktypesv1 "github.com/ngicks/crabswarm/pkg/api/types/sdk_types/v1"
)

// HandlerError is returned by hook subcommands to signal the hook result.
// It implements error as it requires special handling.
type HandlerError struct {
	// Output is the structured JSON to write to stdout on exit 0.
	Output *sdktypesv1.SyncHookJSONOutput
}

func (e *HandlerError) Error() string {
	if e.Output != nil && e.Output.Decision != nil && *e.Output.Decision == sdktypesv1.HookDecisionBlock {
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

	if e.Output.Decision != nil && *e.Output.Decision == sdktypesv1.HookDecisionBlock {
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
		Output: &sdktypesv1.SyncHookJSONOutput{
			HookSpecificOutput: sdktypesv1.HookSpecificOutputPermissionRequest{
				HookEventName: sdktypesv1.HookEventPermissionRequest,
				Decision:      sdktypesv1.PermissionRequestDecisionAllow{},
			},
		},
	}
}
