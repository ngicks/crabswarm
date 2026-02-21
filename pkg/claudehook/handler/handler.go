// Package handler defines handler for claude hook
package handler

import (
	"fmt"
	"os"

	sdk_typesv1 "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"google.golang.org/protobuf/encoding/protojson"
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
	Output *sdk_typesv1.SyncHookJSONOutput
}

func (e *HandlerError) Error() string {
	if e.Output.GetDecision() == string(HookDecisionBlock) {
		return fmt.Sprintf("hook: block: %s", e.Output.GetReason())
	}
	return "hook: allow"
}

// Handle writes output and exits the process.
func (e *HandlerError) Handle() {
	if e.Output == nil {
		os.Exit(0)
	}

	if e.Output.GetDecision() == string(HookDecisionBlock) {
		fmt.Fprint(os.Stderr, e.Output.GetReason())
		os.Exit(2)
	}

	// JSON output to stdout, exit 0
	data, err := protojson.Marshal(e.Output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hook: marshaling SyncHookJSONOutput: protojson.Marshal: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, string(data))
	os.Exit(0)
}
