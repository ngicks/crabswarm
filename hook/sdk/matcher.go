// Package sdk provides utilities for building Claude Code hooks.
package sdk

import (
	"github.com/ngicks/crabswarm/hook/model"
)

// ToolHandler is a function that handles a tool invocation.
// It receives the hook input and the parsed tool input (may be nil).
// Returns the hook output to send back to Claude Code.
type ToolHandler func(input *model.HookInput, toolInput interface{}) model.HookOutput

// Matcher routes hook inputs to appropriate handlers based on tool category.
type Matcher struct {
	// OnCommand handles Bash tool invocations.
	OnCommand ToolHandler

	// OnFilePath handles file operation tools (Read, Write, Edit, Glob, Grep).
	OnFilePath ToolHandler

	// OnWeb handles web operation tools (WebFetch, WebSearch).
	OnWeb ToolHandler

	// OnTask handles agent/task tools (Task, TaskCreate, TaskUpdate, etc.).
	OnTask ToolHandler

	// OnUserInteraction handles user interaction tools (AskUserQuestion).
	OnUserInteraction ToolHandler

	// OnMCP handles MCP server tools (mcp__*).
	OnMCP ToolHandler

	// OnOther handles all other tools or non-tool hooks.
	OnOther ToolHandler

	// Default is the fallback handler when no category-specific handler is set.
	// If nil, defaults to allowing the operation.
	Default ToolHandler
}

// NewMatcher creates a new Matcher with a default allow handler.
func NewMatcher() *Matcher {
	return &Matcher{
		Default: func(input *model.HookInput, toolInput interface{}) model.HookOutput {
			return model.Allow()
		},
	}
}

// Handle processes a hook input and returns the appropriate output.
func (m *Matcher) Handle(input *model.HookInput) model.HookOutput {
	if input == nil {
		return m.handleDefault(nil, nil)
	}

	// Parse tool input if available
	var toolInput interface{}
	var err error
	if len(input.ToolInput) > 0 {
		toolInput, err = input.ParseToolInput()
		if err != nil {
			// If parsing fails, still route to the appropriate handler
			// but with nil toolInput
			toolInput = nil
		}
	}

	// Route based on category
	category := input.Category()
	return m.routeByCategory(input, toolInput, category)
}

// routeByCategory routes to the appropriate handler based on tool category.
func (m *Matcher) routeByCategory(input *model.HookInput, toolInput interface{}, category model.ToolCategory) model.HookOutput {
	switch category {
	case model.CategoryCommand:
		if m.OnCommand != nil {
			return m.OnCommand(input, toolInput)
		}
	case model.CategoryFilePath:
		if m.OnFilePath != nil {
			return m.OnFilePath(input, toolInput)
		}
	case model.CategoryWeb:
		if m.OnWeb != nil {
			return m.OnWeb(input, toolInput)
		}
	case model.CategoryTask:
		if m.OnTask != nil {
			return m.OnTask(input, toolInput)
		}
	case model.CategoryUserInteraction:
		if m.OnUserInteraction != nil {
			return m.OnUserInteraction(input, toolInput)
		}
	case model.CategoryMCP:
		if m.OnMCP != nil {
			return m.OnMCP(input, toolInput)
		}
	case model.CategoryOther:
		if m.OnOther != nil {
			return m.OnOther(input, toolInput)
		}
	}

	return m.handleDefault(input, toolInput)
}

// handleDefault calls the default handler or returns Allow if none is set.
func (m *Matcher) handleDefault(input *model.HookInput, toolInput interface{}) model.HookOutput {
	if m.Default != nil {
		return m.Default(input, toolInput)
	}
	return model.Allow()
}

// WithCommand sets the command handler and returns the matcher for chaining.
func (m *Matcher) WithCommand(handler ToolHandler) *Matcher {
	m.OnCommand = handler
	return m
}

// WithFilePath sets the file path handler and returns the matcher for chaining.
func (m *Matcher) WithFilePath(handler ToolHandler) *Matcher {
	m.OnFilePath = handler
	return m
}

// WithWeb sets the web handler and returns the matcher for chaining.
func (m *Matcher) WithWeb(handler ToolHandler) *Matcher {
	m.OnWeb = handler
	return m
}

// WithTask sets the task handler and returns the matcher for chaining.
func (m *Matcher) WithTask(handler ToolHandler) *Matcher {
	m.OnTask = handler
	return m
}

// WithUserInteraction sets the user interaction handler and returns the matcher for chaining.
func (m *Matcher) WithUserInteraction(handler ToolHandler) *Matcher {
	m.OnUserInteraction = handler
	return m
}

// WithMCP sets the MCP handler and returns the matcher for chaining.
func (m *Matcher) WithMCP(handler ToolHandler) *Matcher {
	m.OnMCP = handler
	return m
}

// WithOther sets the other handler and returns the matcher for chaining.
func (m *Matcher) WithOther(handler ToolHandler) *Matcher {
	m.OnOther = handler
	return m
}

// WithDefault sets the default handler and returns the matcher for chaining.
func (m *Matcher) WithDefault(handler ToolHandler) *Matcher {
	m.Default = handler
	return m
}

// MatcherBuilder provides a fluent API for building a Matcher.
type MatcherBuilder struct {
	matcher *Matcher
}

// NewMatcherBuilder creates a new MatcherBuilder.
func NewMatcherBuilder() *MatcherBuilder {
	return &MatcherBuilder{
		matcher: NewMatcher(),
	}
}

// OnCommand sets the command handler.
func (b *MatcherBuilder) OnCommand(handler ToolHandler) *MatcherBuilder {
	b.matcher.OnCommand = handler
	return b
}

// OnFilePath sets the file path handler.
func (b *MatcherBuilder) OnFilePath(handler ToolHandler) *MatcherBuilder {
	b.matcher.OnFilePath = handler
	return b
}

// OnWeb sets the web handler.
func (b *MatcherBuilder) OnWeb(handler ToolHandler) *MatcherBuilder {
	b.matcher.OnWeb = handler
	return b
}

// OnTask sets the task handler.
func (b *MatcherBuilder) OnTask(handler ToolHandler) *MatcherBuilder {
	b.matcher.OnTask = handler
	return b
}

// OnUserInteraction sets the user interaction handler.
func (b *MatcherBuilder) OnUserInteraction(handler ToolHandler) *MatcherBuilder {
	b.matcher.OnUserInteraction = handler
	return b
}

// OnMCP sets the MCP handler.
func (b *MatcherBuilder) OnMCP(handler ToolHandler) *MatcherBuilder {
	b.matcher.OnMCP = handler
	return b
}

// OnOther sets the other handler.
func (b *MatcherBuilder) OnOther(handler ToolHandler) *MatcherBuilder {
	b.matcher.OnOther = handler
	return b
}

// Default sets the default handler.
func (b *MatcherBuilder) Default(handler ToolHandler) *MatcherBuilder {
	b.matcher.Default = handler
	return b
}

// Build returns the configured Matcher.
func (b *MatcherBuilder) Build() *Matcher {
	return b.matcher
}

// EventHandler is a function that handles hook events (not tool-specific).
type EventHandler func(input *model.HookInput) model.HookOutput

// EventMatcher routes hook inputs based on event type.
type EventMatcher struct {
	handlers map[model.HookEventName]EventHandler
	// Default handler for unmatched events.
	Default EventHandler
}

// NewEventMatcher creates a new EventMatcher.
func NewEventMatcher() *EventMatcher {
	return &EventMatcher{
		handlers: make(map[model.HookEventName]EventHandler),
		Default: func(input *model.HookInput) model.HookOutput {
			return model.Allow()
		},
	}
}

// On registers a handler for a specific event type.
func (m *EventMatcher) On(event model.HookEventName, handler EventHandler) *EventMatcher {
	m.handlers[event] = handler
	return m
}

// OnSessionStart registers a handler for SessionStart events.
func (m *EventMatcher) OnSessionStart(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventSessionStart, handler)
}

// OnSessionEnd registers a handler for SessionEnd events.
func (m *EventMatcher) OnSessionEnd(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventSessionEnd, handler)
}

// OnUserPromptSubmit registers a handler for UserPromptSubmit events.
func (m *EventMatcher) OnUserPromptSubmit(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventUserPromptSubmit, handler)
}

// OnPreToolUse registers a handler for PreToolUse events.
func (m *EventMatcher) OnPreToolUse(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventPreToolUse, handler)
}

// OnPostToolUse registers a handler for PostToolUse events.
func (m *EventMatcher) OnPostToolUse(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventPostToolUse, handler)
}

// OnPostToolUseFailure registers a handler for PostToolUseFailure events.
func (m *EventMatcher) OnPostToolUseFailure(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventPostToolUseFailure, handler)
}

// OnPermissionRequest registers a handler for PermissionRequest events.
func (m *EventMatcher) OnPermissionRequest(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventPermissionRequest, handler)
}

// OnNotification registers a handler for Notification events.
func (m *EventMatcher) OnNotification(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventNotification, handler)
}

// OnSubagentStart registers a handler for SubagentStart events.
func (m *EventMatcher) OnSubagentStart(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventSubagentStart, handler)
}

// OnSubagentStop registers a handler for SubagentStop events.
func (m *EventMatcher) OnSubagentStop(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventSubagentStop, handler)
}

// OnStop registers a handler for Stop events.
func (m *EventMatcher) OnStop(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventStop, handler)
}

// OnPreCompact registers a handler for PreCompact events.
func (m *EventMatcher) OnPreCompact(handler EventHandler) *EventMatcher {
	return m.On(model.HookEventPreCompact, handler)
}

// WithDefault sets the default handler.
func (m *EventMatcher) WithDefault(handler EventHandler) *EventMatcher {
	m.Default = handler
	return m
}

// Handle processes a hook input and returns the appropriate output.
func (m *EventMatcher) Handle(input *model.HookInput) model.HookOutput {
	if input == nil {
		if m.Default != nil {
			return m.Default(nil)
		}
		return model.Allow()
	}

	if handler, ok := m.handlers[input.HookName]; ok {
		return handler(input)
	}

	if m.Default != nil {
		return m.Default(input)
	}
	return model.Allow()
}

// CombinedMatcher combines event-level and tool-level matching.
type CombinedMatcher struct {
	eventMatcher *EventMatcher
	toolMatcher  *Matcher
}

// NewCombinedMatcher creates a new CombinedMatcher.
func NewCombinedMatcher() *CombinedMatcher {
	return &CombinedMatcher{
		eventMatcher: NewEventMatcher(),
		toolMatcher:  NewMatcher(),
	}
}

// Events returns the event matcher for configuration.
func (m *CombinedMatcher) Events() *EventMatcher {
	return m.eventMatcher
}

// Tools returns the tool matcher for configuration.
func (m *CombinedMatcher) Tools() *Matcher {
	return m.toolMatcher
}

// Handle processes a hook input, first checking event handlers, then tool handlers.
func (m *CombinedMatcher) Handle(input *model.HookInput) model.HookOutput {
	if input == nil {
		return model.Allow()
	}

	// Check if there's a specific event handler
	if handler, ok := m.eventMatcher.handlers[input.HookName]; ok {
		return handler(input)
	}

	// For tool-related events, use the tool matcher
	if input.HookName.IsToolRelated() && input.ToolName != "" {
		return m.toolMatcher.Handle(input)
	}

	// Fall back to event matcher's default
	if m.eventMatcher.Default != nil {
		return m.eventMatcher.Default(input)
	}

	return model.Allow()
}
