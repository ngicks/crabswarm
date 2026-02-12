package sdk

import (
	"encoding/json"
	"testing"

	"github.com/ngicks/crabswarm/hook/model"
)

func TestMatcherRouting(t *testing.T) {
	tests := []struct {
		name        string
		input       *model.HookInput
		expectRoute string
	}{
		{
			name: "Bash routes to Command",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolNameBash,
				ToolInput: json.RawMessage(`{"command":"ls"}`),
			},
			expectRoute: "command",
		},
		{
			name: "Read routes to FilePath",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolNameRead,
				ToolInput: json.RawMessage(`{"file_path":"/tmp/test"}`),
			},
			expectRoute: "filepath",
		},
		{
			name: "Write routes to FilePath",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolNameWrite,
				ToolInput: json.RawMessage(`{"file_path":"/tmp/test","content":"hello"}`),
			},
			expectRoute: "filepath",
		},
		{
			name: "WebFetch routes to Web",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolNameWebFetch,
				ToolInput: json.RawMessage(`{"url":"https://example.com","prompt":"test"}`),
			},
			expectRoute: "web",
		},
		{
			name: "Task routes to Task",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolNameTask,
				ToolInput: json.RawMessage(`{"description":"test","prompt":"do something","subagent_type":"Bash"}`),
			},
			expectRoute: "task",
		},
		{
			name: "AskUserQuestion routes to UserInteraction",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolNameAskUserQuestion,
				ToolInput: json.RawMessage(`{"questions":[]}`),
			},
			expectRoute: "userinteraction",
		},
		{
			name: "MCP tool routes to MCP",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolName("mcp__serena__find_symbol"),
				ToolInput: json.RawMessage(`{"name_path":"Foo"}`),
			},
			expectRoute: "mcp",
		},
		{
			name: "ExitPlanMode routes to PlanMode",
			input: &model.HookInput{
				HookEventName: model.HookEventPreToolUse,
				ToolName:      model.ToolNameExitPlanMode,
				ToolInput:     json.RawMessage(`{}`),
			},
			expectRoute: "planmode",
		},
		{
			name: "EnterPlanMode routes to PlanMode",
			input: &model.HookInput{
				HookEventName: model.HookEventPreToolUse,
				ToolName:      model.ToolNameEnterPlanMode,
				ToolInput:     json.RawMessage(`{}`),
			},
			expectRoute: "planmode",
		},
		{
			name: "Unknown tool routes to Other",
			input: &model.HookInput{
				HookEventName:  model.HookEventPreToolUse,
				ToolName:  model.ToolName("UnknownTool"),
				ToolInput: json.RawMessage(`{}`),
			},
			expectRoute: "other",
		},
		{
			name: "No tool routes to Other",
			input: &model.HookInput{
				HookEventName: model.HookEventSessionStart,
			},
			expectRoute: "other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualRoute string

			matcher := NewMatcher().
				WithCommand(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "command"
					return model.Allow()
				}).
				WithFilePath(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "filepath"
					return model.Allow()
				}).
				WithWeb(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "web"
					return model.Allow()
				}).
				WithTask(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "task"
					return model.Allow()
				}).
				WithUserInteraction(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "userinteraction"
					return model.Allow()
				}).
				WithPlanMode(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "planmode"
					return model.Allow()
				}).
				WithMCP(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "mcp"
					return model.Allow()
				}).
				WithOther(func(input *model.HookInput, toolInput any) model.HookOutput {
					actualRoute = "other"
					return model.Allow()
				})

			matcher.Handle(tt.input)

			if actualRoute != tt.expectRoute {
				t.Errorf("Expected route %s, got %s", tt.expectRoute, actualRoute)
			}
		})
	}
}

func TestMatcherNilInput(t *testing.T) {
	matcher := NewMatcher()
	output := matcher.Handle(nil)

	// Empty output (Allow) has nil HookSpecificOutput
	if output.HookSpecificOutput != nil {
		t.Errorf("Expected nil HookSpecificOutput for nil input")
	}
}

func TestMatcherDefaultHandler(t *testing.T) {
	matcher := NewMatcher().
		WithDefault(func(input *model.HookInput, toolInput any) model.HookOutput {
			return model.Deny(input.HookEventName, "blocked by default")
		})

	input := &model.HookInput{
		HookEventName: model.HookEventPreToolUse,
		ToolName:      model.ToolNameBash,
	}

	output := matcher.Handle(input)

	if output.HookSpecificOutput == nil {
		t.Fatal("Expected non-nil HookSpecificOutput")
	}
	if output.HookSpecificOutput.PermissionDecision != model.PermissionDeny {
		t.Errorf("Expected deny decision, got %s", output.HookSpecificOutput.PermissionDecision)
	}
	if output.HookSpecificOutput.PermissionDecisionReason != "blocked by default" {
		t.Errorf("Expected reason 'blocked by default', got '%s'", output.HookSpecificOutput.PermissionDecisionReason)
	}
}

func TestMatcherParsedInput(t *testing.T) {
	var receivedInput *model.BashInput

	matcher := NewMatcher().
		WithCommand(func(input *model.HookInput, toolInput any) model.HookOutput {
			if bi, ok := toolInput.(*model.BashInput); ok {
				receivedInput = bi
			}
			return model.Allow()
		})

	hookInput := &model.HookInput{
		HookEventName:  model.HookEventPreToolUse,
		ToolName:  model.ToolNameBash,
		ToolInput: json.RawMessage(`{"command":"echo hello","description":"Print hello"}`),
	}

	matcher.Handle(hookInput)

	if receivedInput == nil {
		t.Fatal("Expected parsed BashInput, got nil")
	}
	if receivedInput.Command != "echo hello" {
		t.Errorf("Expected command 'echo hello', got '%s'", receivedInput.Command)
	}
	if receivedInput.Description != "Print hello" {
		t.Errorf("Expected description 'Print hello', got '%s'", receivedInput.Description)
	}
}

func TestEventMatcherRouting(t *testing.T) {
	tests := []struct {
		name        string
		hookName    model.HookEventName
		expectEvent string
	}{
		{model.HookEventSessionStart.String(), model.HookEventSessionStart, "sessionstart"},
		{"SessionEnd", model.HookEventSessionEnd, "sessionend"},
		{"PreToolUse", model.HookEventPreToolUse, "pretooluse"},
		{"PostToolUse", model.HookEventPostToolUse, "posttooluse"},
		{"Unknown", model.HookEventName("Unknown"), "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualEvent string

			em := NewEventMatcher().
				OnSessionStart(func(input *model.HookInput) model.HookOutput {
					actualEvent = "sessionstart"
					return model.Allow()
				}).
				OnSessionEnd(func(input *model.HookInput) model.HookOutput {
					actualEvent = "sessionend"
					return model.Allow()
				}).
				OnPreToolUse(func(input *model.HookInput) model.HookOutput {
					actualEvent = "pretooluse"
					return model.Allow()
				}).
				OnPostToolUse(func(input *model.HookInput) model.HookOutput {
					actualEvent = "posttooluse"
					return model.Allow()
				}).
				WithDefault(func(input *model.HookInput) model.HookOutput {
					actualEvent = "default"
					return model.Allow()
				})

			em.Handle(&model.HookInput{HookEventName: tt.hookName})

			if actualEvent != tt.expectEvent {
				t.Errorf("Expected event %s, got %s", tt.expectEvent, actualEvent)
			}
		})
	}
}

func TestCombinedMatcher(t *testing.T) {
	var handledBy string

	cm := NewCombinedMatcher()
	cm.Events().OnSessionStart(func(input *model.HookInput) model.HookOutput {
		handledBy = "session_event"
		return model.Allow()
	})
	cm.Tools().WithCommand(func(input *model.HookInput, toolInput any) model.HookOutput {
		handledBy = "command_tool"
		return model.Allow()
	})

	// Test event handling
	cm.Handle(&model.HookInput{HookEventName: model.HookEventSessionStart})
	if handledBy != "session_event" {
		t.Errorf("Expected session_event, got %s", handledBy)
	}

	// Test tool handling
	cm.Handle(&model.HookInput{
		HookEventName: model.HookEventPreToolUse,
		ToolName: model.ToolNameBash,
	})
	if handledBy != "command_tool" {
		t.Errorf("Expected command_tool, got %s", handledBy)
	}
}

func TestMatcherBuilder(t *testing.T) {
	var routed string

	matcher := NewMatcherBuilder().
		OnCommand(func(input *model.HookInput, toolInput any) model.HookOutput {
			routed = "command"
			return model.Allow()
		}).
		OnFilePath(func(input *model.HookInput, toolInput any) model.HookOutput {
			routed = "filepath"
			return model.Allow()
		}).
		Build()

	matcher.Handle(&model.HookInput{
		HookEventName: model.HookEventPreToolUse,
		ToolName: model.ToolNameBash,
	})
	if routed != "command" {
		t.Errorf("Expected command, got %s", routed)
	}

	matcher.Handle(&model.HookInput{
		HookEventName: model.HookEventPreToolUse,
		ToolName: model.ToolNameRead,
	})
	if routed != "filepath" {
		t.Errorf("Expected filepath, got %s", routed)
	}
}
