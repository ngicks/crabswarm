// Package handler defines handler for claude hook
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ngicks/crabswarm/pkg/claudehook/types"
)

// HandlerError is returned by hook subcommands to signal the hook result.
// It implements error as it requires special handling.
type HandlerError struct {
	// Output is the structured JSON to write to stdout on exit 0.
	Output *types.SyncHookJSONOutput
}

// Error is a display-string approximation of the hook result, for logs and
// generic error reporting only. It is never what the hook emits on the wire:
// [HandlerError.Handle] owns that and always writes the crafted JSON.
func (e *HandlerError) Error() string {
	if e.Output != nil && e.Output.Decision != nil &&
		*e.Output.Decision == types.HookDecisionBlock {
		reason := ""
		if e.Output.Reason != nil {
			reason = *e.Output.Reason
		}
		return fmt.Sprintf("hook: block: %s", reason)
	}
	return "hook: allow"
}

// Handle writes the hook output and exits the process. A non-nil Output is
// always marshaled to stdout as one JSON line, followed by exit 0 — including
// a decision=block, which is valid hook protocol in JSON form. A nil Output is
// a plain allow: nothing written, exit 0.
//
// The exit-2 + reason-on-stderr form is no longer used: it can carry only
// decision and reason, so choosing it would mean classifying outputs by which
// fields happen to be set, and any field upstream adds to
// [types.SyncHookJSONOutput] would silently reclassify them.
func (e *HandlerError) Handle() {
	if e.Output == nil {
		os.Exit(0)
	}

	data, err := json.Marshal(e.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hook: marshaling SyncHookJSONOutput: json.Marshal: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, string(data))
	os.Exit(0)
}

func Handle(err error) {
	hook, ok := errors.AsType[*HandlerError](err)
	if !ok {
		return
	}
	hook.Handle()
}

// PermissionAllow creates a HandlerError that outputs a PermissionRequest
// approval response (hookSpecificOutput with decision.behavior =
// PermissionRequestDecisionBehaviorAllow).
func PermissionAllow(
	updatedInput map[string]any,
	updatedPermission []types.PermissionUpdate,
) *HandlerError {
	return &HandlerError{
		Output: &types.SyncHookJSONOutput{
			HookSpecificOutput: types.NewHookSpecificOutput(
				&types.HookSpecificOutputPermissionRequest{
					HookEventName: types.HookEventPermissionRequest,
					Decision: types.NewPermissionRequestDecision(
						&types.PermissionRequestDecisionAllow{
							Behavior:           types.PermissionRequestDecisionBehaviorAllow,
							UpdatedInput:       updatedInput,
							UpdatedPermissions: updatedPermission,
						},
					),
				},
			),
		},
	}
}

func opt[T comparable](t T) *T {
	if t == *new(T) {
		return nil
	}
	return &t
}

func PermissionDeny(message string, interrupt bool) *HandlerError {
	return &HandlerError{
		Output: &types.SyncHookJSONOutput{
			HookSpecificOutput: types.NewHookSpecificOutput(
				&types.HookSpecificOutputPermissionRequest{
					HookEventName: types.HookEventPermissionRequest,
					Decision: types.NewPermissionRequestDecision(
						&types.PermissionRequestDecisionDeny{
							Behavior:  types.PermissionRequestDecisionBehaviorDeny,
							Message:   opt(message),
							Interrupt: opt(interrupt),
						},
					),
				},
			),
		},
	}
}

// Block creates a HandlerError that blocks the current hook event
// with `reason` shown to the LLM. Generic across PreToolUse, PostToolUse,
// Stop, and other events that accept decision=block. The exec hook uses
// this to surface lint/formatter output back to the agent.
func Block(reason string) *HandlerError {
	return &HandlerError{
		Output: &types.SyncHookJSONOutput{
			Decision: new(types.HookDecisionBlock),
			Reason:   &reason,
		},
	}
}
