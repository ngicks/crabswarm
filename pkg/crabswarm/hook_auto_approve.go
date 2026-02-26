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

// HookAutoApprove reads a PermissionRequestHookInput from r and auto-approves
// if all configured conditions are met.
func HookAutoApprove(_ context.Context, r io.Reader, cfg AutoApproveConfig) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input models.PermissionRequestHookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("parsing PermissionRequestHookInput: %w", err)
	}

	// Check tool name patterns.
	if len(cfg.ToolPatterns) > 0 {
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
			// Tool name doesn't match any pattern — pass through.
			return &handler.HandlerError{}
		}
	}

	// Check directory containment.
	if len(cfg.UnderDirs) > 0 {
		// Extract file_path from tool_input.
		filePath := extractFilePath(input.ToolInput)
		if filePath == "" {
			// No file_path in tool_input — pass through.
			return &handler.HandlerError{}
		}

		underAny := false
		for _, dir := range cfg.UnderDirs {
			under, err := planreview.PathWithinDir(filePath, dir)
			if err != nil {
				// Can't verify containment — pass through.
				return &handler.HandlerError{}
			}
			if under {
				underAny = true
				break
			}
		}
		if !underAny {
			// File not under any configured directory — pass through.
			return &handler.HandlerError{}
		}
	}

	// All conditions matched — auto-approve.
	return handler.NewPermissionRequestAllowError()
}

// extractFilePath extracts the file_path field from raw tool_input JSON.
func extractFilePath(toolInput json.RawMessage) string {
	if len(toolInput) == 0 {
		return ""
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(toolInput, &m); err != nil {
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
