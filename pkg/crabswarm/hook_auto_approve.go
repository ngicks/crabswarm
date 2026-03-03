package crabswarm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/ngicks/crabswarm/pkg/claudesdk/models"
	"github.com/ngicks/crabswarm/pkg/crabswarm/planreview"
)

// AutoApproveConfig holds configuration for the auto-approve hook.
type AutoApproveConfig struct {
	ToolPatterns []string // regex patterns to match tool_name
	UnderDirs    []string // directories — approve if file_path is under any
}

// Validate checks that both ToolPatterns and UnderDirs are populated.
func (c AutoApproveConfig) Validate() error {
	if len(c.ToolPatterns) == 0 {
		return fmt.Errorf("at least one --tool pattern is required")
	}
	if len(c.UnderDirs) == 0 {
		return fmt.Errorf("at least one --under directory is required")
	}
	return nil
}

// HookAutoApprove reads a PermissionRequestHookInput from r and auto-approves
// if all configured conditions are met. Both ToolPatterns and UnderDirs must be set.
func HookAutoApprove(_ context.Context, r io.Reader, cfg AutoApproveConfig) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input models.PermissionRequestHookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("parsing PermissionRequestHookInput: %w", err)
	}

	// Check tool name patterns.
	matched := false
	for _, pattern := range cfg.ToolPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("compiling tool pattern %q: %w", pattern, err)
		}
		if re.MatchString(input.ToolName) {
			matched = true
			break
		}
	}
	if !matched {
		return &handler.HandlerError{}
	}

	// Check directory containment.
	filePath := extractFilePath(input.ToolInput)
	if filePath == "" {
		return &handler.HandlerError{}
	}

	underAny := false
	for _, dir := range cfg.UnderDirs {
		under, err := planreview.PathWithinDir(filePath, dir)
		if err != nil {
			return &handler.HandlerError{}
		}
		if under {
			underAny = true
			break
		}
	}
	if !underAny {
		return &handler.HandlerError{}
	}

	// All conditions matched — auto-approve.
	return handler.NewPermissionRequestAllowError()
}

// extractFilePath extracts the file_path field from raw tool_input JSON.
func extractFilePath(toolInput any) string {
	if toolInput == nil {
		return ""
	}
	raw, err := json.Marshal(toolInput)
	if err != nil {
		return ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	raw, ok := m["file_path"]
	if !ok {
		return ""
	}
	var fp string
	if err := json.Unmarshal(raw, &fp); err != nil {
		return ""
	}
	return fp
}
