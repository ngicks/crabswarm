package exec

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"text/template"

	"github.com/ngicks/crabswarm/internal/templateutil"
	"github.com/ngicks/crabswarm/pkg/claudehook/types"
)

// OutputData is the value passed to the output template at render time. It
// embeds the [Data] the command template already saw — so .File, .Event and
// friends stay reachable — and adds the result of the command that ran.
type OutputData struct {
	Data
	// Command is the rendered command line that was executed.
	Command string
	// ExitCode is the child process's exit status; 0 on success.
	ExitCode int
	// Success reports ExitCode == 0.
	Success bool
	// Error is the run error's message — exactly what the built-in block
	// reason prints on its `exit:` line, e.g. "exit status 1" or
	// "signal: killed". Empty on success, and empty in the --dry-run
	// preview, where nothing ran.
	Error string
	// Output is the combined stdout+stderr capture with the trailing
	// newline trimmed. Interleaving is best-effort.
	Output string
	// Stdout is the stdout-only capture, trailing newline trimmed.
	Stdout string
	// Stderr is the stderr-only capture, trailing newline trimmed.
	Stderr string
}

// outputBuilder accumulates a [types.SyncHookJSONOutput] across the template
// functions an output template calls. Every function records onto the
// builder and renders as the empty string, so the template body is pure
// control flow and this package — not the user's template — owns the JSON
// assembly.
//
// Repeated calls to the same function are last-wins: the builder is mutable
// and a later call simply overwrites the field an earlier one set.
type outputBuilder struct {
	// event is the incoming hook event. It selects which hookSpecificOutput
	// variant the event-scoped functions may touch.
	event types.HookEvent
	// out holds the top-level fields recorded so far.
	out types.SyncHookJSONOutput
	// hso is the hookSpecificOutput variant for event, created on first use
	// by an event-scoped function.
	hso types.HookSpecificOutput_Value
	// set reports whether any function fired. A template that recorded
	// nothing means plain allow, which is a nil output rather than an empty
	// object.
	set bool
}

// build returns the accumulated output, or nil when no function fired.
func (b *outputBuilder) build() *types.SyncHookJSONOutput {
	if !b.set {
		return nil
	}
	out := b.out
	if b.hso != nil {
		out.HookSpecificOutput = types.NewHookSpecificOutput(b.hso)
	}
	return &out
}

// hookSpecific returns the hookSpecificOutput variant for the builder's
// event, creating it on first use. fn names the calling template function
// so the error points at the offending template.
func (b *outputBuilder) hookSpecific(fn string) (types.HookSpecificOutput_Value, error) {
	if b.hso == nil {
		v := newHookSpecificOutput(b.event)
		if v == nil {
			return nil, unsupportedEvent(fn, b.event)
		}
		b.hso = v
	}
	return b.hso, nil
}

// hookSpecificPtr constrains [hookSpecificAs] to pointers of the
// hookSpecificOutput variant structs, which is what the union actually
// carries.
type hookSpecificPtr[T any] interface {
	*T
	types.HookSpecificOutput_Value
}

// hookSpecificAs is the single-variant form of [outputBuilder.hookSpecific]:
// it returns the variant as *T, erroring when the incoming event maps to a
// different variant — i.e. the template called fn for an event that does not
// support it. Functions spanning several variants type-switch instead.
func hookSpecificAs[T any, PT hookSpecificPtr[T]](b *outputBuilder, fn string) (PT, error) {
	v, err := b.hookSpecific(fn)
	if err != nil {
		return nil, err
	}
	t, ok := v.(PT)
	if !ok {
		return nil, unsupportedEvent(fn, b.event)
	}
	return t, nil
}

// unsupportedEvent reports an event-scoped function called for an event that
// has no place to record it. Such a mismatch fails the render rather than
// emitting a hookSpecificOutput Claude Code would ignore.
func unsupportedEvent(fn string, event types.HookEvent) error {
	return fmt.Errorf("%s: not supported for hook event %q", fn, event)
}

// errSurplusArgs reports a call carrying more arguments than the function
// accepts. text/template type-checks only a variadic function's fixed
// parameters, so trailing extras arrive silently; every optional argument is
// bounded explicitly rather than letting a template typo be dropped.
// usage lists the accepted arguments and want/got count all of them, fixed
// ones included, so the message lines up with the documented usage string.
func errSurplusArgs(fn, usage string, want, got int) error {
	return fmt.Errorf("%s: want at most %d arguments (%s), got %d", fn, want, usage, got)
}

// newHookSpecificOutput returns a fresh hookSpecificOutput variant for event
// with its discriminator filled in, or nil when the event has no variant.
// It mirrors the event → variant mapping of
// [types.HookSpecificOutput.UnmarshalJSON].
func newHookSpecificOutput(event types.HookEvent) types.HookSpecificOutput_Value {
	switch event {
	case types.HookEventPreToolUse:
		return &types.HookSpecificOutputPreToolUse{HookEventName: event}
	case types.HookEventPostToolUse:
		return &types.HookSpecificOutputPostToolUse{HookEventName: event}
	case types.HookEventPostToolUseFailure:
		return &types.HookSpecificOutputPostToolUseFailure{HookEventName: event}
	case types.HookEventPostToolBatch:
		return &types.HookSpecificOutputPostToolBatch{HookEventName: event}
	case types.HookEventNotification:
		return &types.HookSpecificOutputNotification{HookEventName: event}
	case types.HookEventUserPromptSubmit:
		return &types.HookSpecificOutputUserPromptSubmit{HookEventName: event}
	case types.HookEventUserPromptExpansion:
		return &types.HookSpecificOutputUserPromptExpansion{HookEventName: event}
	case types.HookEventSessionStart:
		return &types.HookSpecificOutputSessionStart{HookEventName: event}
	case types.HookEventSetup:
		return &types.HookSpecificOutputSetup{HookEventName: event}
	case types.HookEventSubagentStart:
		return &types.HookSpecificOutputSubagentStart{HookEventName: event}
	case types.HookEventStop:
		return &types.HookSpecificOutputStop{HookEventName: event}
	case types.HookEventSubagentStop:
		return &types.HookSpecificOutputSubagentStop{HookEventName: event}
	case types.HookEventPermissionRequest:
		return &types.HookSpecificOutputPermissionRequest{HookEventName: event}
	case types.HookEventPermissionDenied:
		return &types.HookSpecificOutputPermissionDenied{HookEventName: event}
	case types.HookEventElicitation:
		return &types.HookSpecificOutputElicitation{HookEventName: event}
	case types.HookEventElicitationResult:
		return &types.HookSpecificOutputElicitationResult{HookEventName: event}
	case types.HookEventCwdChanged:
		return &types.HookSpecificOutputCwdChanged{HookEventName: event}
	case types.HookEventFileChanged:
		return &types.HookSpecificOutputFileChanged{HookEventName: event}
	case types.HookEventWorktreeCreate:
		return &types.HookSpecificOutputWorktreeCreate{HookEventName: event}
	case types.HookEventMessageDisplay:
		return &types.HookSpecificOutputMessageDisplay{HookEventName: event}
	default:
		return nil
	}
}

// --- top-level SyncHookJSONOutput fields (available to every event) ---

// blockDecision is not spelled `block`: text/template lexes `block` as the
// block-definition keyword in every position, so it cannot be registered as
// a function name at all.
func (b *outputBuilder) blockDecision(reason string) (string, error) {
	b.out.Decision = new(types.HookDecisionBlock)
	b.out.Reason = &reason
	b.set = true
	return "", nil
}

func (b *outputBuilder) stop(reason string) (string, error) {
	b.out.Continue = new(false)
	b.out.StopReason = &reason
	b.set = true
	return "", nil
}

func (b *outputBuilder) systemMessage(msg string) (string, error) {
	b.out.SystemMessage = &msg
	b.set = true
	return "", nil
}

func (b *outputBuilder) suppressOutput() (string, error) {
	b.out.SuppressOutput = new(true)
	b.set = true
	return "", nil
}

func (b *outputBuilder) terminalSequence(seq string) (string, error) {
	b.out.TerminalSequence = &seq
	b.set = true
	return "", nil
}

// --- event-aware ---

// additionalContext backs the `context` function. The type switch doubles as
// the allow-list: exactly the variants carrying an additionalContext field
// accept it, and anything else — an event whose variant lacks the field, or
// one with no variant at all — fails the render.
func (b *outputBuilder) additionalContext(text string) (string, error) {
	v, err := b.hookSpecific("context")
	if err != nil {
		return "", err
	}
	switch v := v.(type) {
	case *types.HookSpecificOutputPreToolUse:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputPostToolUse:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputPostToolUseFailure:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputPostToolBatch:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputNotification:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputUserPromptSubmit:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputUserPromptExpansion:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputSessionStart:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputSetup:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputSubagentStart:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputStop:
		v.AdditionalContext = &text
	case *types.HookSpecificOutputSubagentStop:
		v.AdditionalContext = &text
	default:
		return "", unsupportedEvent("context", b.event)
	}
	b.set = true
	return "", nil
}

// --- event-specific ---

func (b *outputBuilder) permission(decision string, reason ...string) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputPreToolUse](b, "permission")
	if err != nil {
		return "", err
	}
	if len(reason) > 1 {
		return "", errSurplusArgs("permission", "DECISION, REASON", 2, 1+len(reason))
	}
	d := types.HookPermissionDecision(decision)
	switch d {
	case types.HookPermissionDecisionAllow,
		types.HookPermissionDecisionDeny,
		types.HookPermissionDecisionAsk:
	default:
		return "", fmt.Errorf(
			"permission: unknown decision %q; want one of allow, deny, ask", decision)
	}
	v.PermissionDecision = &d
	// Cleared first: calls are last-wins, so an omitted REASON must drop the
	// one an earlier call recorded instead of leaving it attached to this
	// decision.
	v.PermissionDecisionReason = nil
	if len(reason) > 0 {
		v.PermissionDecisionReason = &reason[0]
	}
	b.set = true
	return "", nil
}

func (b *outputBuilder) updatedInput(jsonObject string) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputPreToolUse](b, "updatedInput")
	if err != nil {
		return "", err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonObject), &m); err != nil {
		return "", fmt.Errorf("updatedInput: parsing %q as a JSON object: %w", jsonObject, err)
	}
	v.UpdatedInput = m
	b.set = true
	return "", nil
}

func (b *outputBuilder) updatedToolOutput(jsonValue string) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputPostToolUse](b, "updatedToolOutput")
	if err != nil {
		return "", err
	}
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(jsonValue), &raw); err != nil {
		return "", fmt.Errorf("updatedToolOutput: parsing %q as JSON: %w", jsonValue, err)
	}
	v.UpdatedToolOutput = raw
	b.set = true
	return "", nil
}

// permissionAllow mirrors the handler package's PermissionAllow, whose
// approval carries an optional updatedInput and updatedPermissions. Both
// arrive as JSON text, like updatedInput's argument: a template has no way to
// build a map or a slice otherwise.
func (b *outputBuilder) permissionAllow(args ...string) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputPermissionRequest](b, "permissionAllow")
	if err != nil {
		return "", err
	}
	if len(args) > 2 {
		return "", errSurplusArgs(
			"permissionAllow", "UPDATED_INPUT_JSON, UPDATED_PERMISSIONS_JSON", 2, len(args))
	}
	allow := &types.PermissionRequestDecisionAllow{
		Behavior: types.PermissionRequestDecisionBehaviorAllow,
	}
	if len(args) > 0 {
		if err := json.Unmarshal([]byte(args[0]), &allow.UpdatedInput); err != nil {
			return "", fmt.Errorf(
				"permissionAllow: parsing %q as a JSON object: %w", args[0], err)
		}
	}
	if len(args) > 1 {
		if err := json.Unmarshal([]byte(args[1]), &allow.UpdatedPermissions); err != nil {
			return "", fmt.Errorf(
				"permissionAllow: parsing %q as a JSON array: %w", args[1], err)
		}
	}
	v.Decision = types.NewPermissionRequestDecision(allow)
	b.set = true
	return "", nil
}

func (b *outputBuilder) permissionDeny(message string, interrupt ...bool) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputPermissionRequest](b, "permissionDeny")
	if err != nil {
		return "", err
	}
	if len(interrupt) > 1 {
		return "", errSurplusArgs("permissionDeny", "MSG, INTERRUPT", 2, 1+len(interrupt))
	}
	deny := &types.PermissionRequestDecisionDeny{
		Behavior: types.PermissionRequestDecisionBehaviorDeny,
		Message:  &message,
	}
	// Only an explicitly passed INTERRUPT is recorded; false is a
	// meaningful value, so it must not collapse back to unset.
	if len(interrupt) > 0 {
		deny.Interrupt = &interrupt[0]
	}
	v.Decision = types.NewPermissionRequestDecision(deny)
	b.set = true
	return "", nil
}

func (b *outputBuilder) retry() (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputPermissionDenied](b, "retry")
	if err != nil {
		return "", err
	}
	v.Retry = new(true)
	b.set = true
	return "", nil
}

func (b *outputBuilder) sessionTitle(title string) (string, error) {
	v, err := b.hookSpecific("sessionTitle")
	if err != nil {
		return "", err
	}
	switch v := v.(type) {
	case *types.HookSpecificOutputSessionStart:
		v.SessionTitle = &title
	case *types.HookSpecificOutputUserPromptSubmit:
		v.SessionTitle = &title
	default:
		return "", unsupportedEvent("sessionTitle", b.event)
	}
	b.set = true
	return "", nil
}

func (b *outputBuilder) initialUserMessage(msg string) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputSessionStart](b, "initialUserMessage")
	if err != nil {
		return "", err
	}
	v.InitialUserMessage = &msg
	b.set = true
	return "", nil
}

func (b *outputBuilder) watchPaths(paths ...string) (string, error) {
	if len(paths) == 0 {
		return "", fmt.Errorf("watchPaths: at least one PATH is required")
	}
	v, err := b.hookSpecific("watchPaths")
	if err != nil {
		return "", err
	}
	switch v := v.(type) {
	case *types.HookSpecificOutputSessionStart:
		v.WatchPaths = paths
	case *types.HookSpecificOutputCwdChanged:
		v.WatchPaths = paths
	case *types.HookSpecificOutputFileChanged:
		v.WatchPaths = paths
	default:
		return "", unsupportedEvent("watchPaths", b.event)
	}
	b.set = true
	return "", nil
}

func (b *outputBuilder) reloadSkills() (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputSessionStart](b, "reloadSkills")
	if err != nil {
		return "", err
	}
	v.ReloadSkills = new(true)
	b.set = true
	return "", nil
}

func (b *outputBuilder) suppressOriginalPrompt() (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputUserPromptSubmit](
		b, "suppressOriginalPrompt")
	if err != nil {
		return "", err
	}
	v.SuppressOriginalPrompt = new(true)
	b.set = true
	return "", nil
}

func (b *outputBuilder) displayContent(text string) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputMessageDisplay](b, "displayContent")
	if err != nil {
		return "", err
	}
	v.DisplayContent = &text
	b.set = true
	return "", nil
}

// elicitation backs both Elicitation and ElicitationResult. The two variants
// spell their action as distinct Go types with identical values, so each
// branch converts and validates on its own.
func (b *outputBuilder) elicitation(action string, content ...string) (string, error) {
	v, err := b.hookSpecific("elicitation")
	if err != nil {
		return "", err
	}
	m, err := elicitationContent(content)
	if err != nil {
		return "", err
	}
	switch v := v.(type) {
	case *types.HookSpecificOutputElicitation:
		a := types.HookSpecificOutputElicitationAction(action)
		switch a {
		case types.HookSpecificOutputElicitationActionAccept,
			types.HookSpecificOutputElicitationActionDecline,
			types.HookSpecificOutputElicitationActionCancel:
		default:
			return "", errElicitationAction(action)
		}
		v.Action = &a
		v.Content = m
	case *types.HookSpecificOutputElicitationResult:
		a := types.HookSpecificOutputElicitationResultAction(action)
		switch a {
		case types.HookSpecificOutputElicitationResultActionAccept,
			types.HookSpecificOutputElicitationResultActionDecline,
			types.HookSpecificOutputElicitationResultActionCancel:
		default:
			return "", errElicitationAction(action)
		}
		v.Action = &a
		v.Content = m
	default:
		return "", unsupportedEvent("elicitation", b.event)
	}
	b.set = true
	return "", nil
}

// elicitationContent decodes the optional JSON-object argument shared by both
// elicitation variants. A missing argument yields a nil map, which the callers
// record as-is: calls are last-wins, so an omitted CONTENT drops what an
// earlier call recorded.
func elicitationContent(args []string) (map[string]any, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if len(args) > 1 {
		return nil, errSurplusArgs("elicitation", "ACTION, CONTENT", 2, 1+len(args))
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(args[0]), &m); err != nil {
		return nil, fmt.Errorf("elicitation: parsing %q as a JSON object: %w", args[0], err)
	}
	return m, nil
}

func errElicitationAction(action string) error {
	return fmt.Errorf(
		"elicitation: unknown action %q; want one of accept, decline, cancel", action)
}

func (b *outputBuilder) worktreePath(path string) (string, error) {
	v, err := hookSpecificAs[types.HookSpecificOutputWorktreeCreate](b, "worktreePath")
	if err != nil {
		return "", err
	}
	v.WorktreePath = path
	b.set = true
	return "", nil
}

// outputFuncMap returns the side-effect functions bound to b. A fresh map is
// built per render because the builder carries the render's accumulated
// state; it must never be shared across invocations.
func outputFuncMap(b *outputBuilder) template.FuncMap {
	return template.FuncMap{
		"blockDecision":          b.blockDecision,
		"stop":                   b.stop,
		"systemMessage":          b.systemMessage,
		"suppressOutput":         b.suppressOutput,
		"terminalSequence":       b.terminalSequence,
		"context":                b.additionalContext,
		"permission":             b.permission,
		"updatedInput":           b.updatedInput,
		"updatedToolOutput":      b.updatedToolOutput,
		"permissionAllow":        b.permissionAllow,
		"permissionDeny":         b.permissionDeny,
		"retry":                  b.retry,
		"sessionTitle":           b.sessionTitle,
		"initialUserMessage":     b.initialUserMessage,
		"watchPaths":             b.watchPaths,
		"reloadSkills":           b.reloadSkills,
		"suppressOriginalPrompt": b.suppressOriginalPrompt,
		"displayContent":         b.displayContent,
		"elicitation":            b.elicitation,
		"worktreePath":           b.worktreePath,
	}
}

// renderOutput parses and executes src against data, returning the hook
// output its template functions accumulated. It returns a nil output when no
// function fired: the template chose plain allow.
//
// The func map is the builder's side-effect functions merged over
// [templateutil.FuncMap], so the generic string helpers stay available to the
// output template exactly as they are to the command template.
//
// Rendered non-whitespace text is a hard error. The functions are the only
// sanctioned output path, so a stray literal fails the invocation instead of
// silently degrading to allow-everything.
func renderOutput(src string, data OutputData) (*types.SyncHookJSONOutput, error) {
	b := &outputBuilder{event: data.Event}
	funcs := templateutil.FuncMap()
	maps.Copy(funcs, outputFuncMap(b))

	tmpl, err := template.New("output").
		Option("missingkey=zero").
		Funcs(funcs).
		Parse(src)
	if err != nil {
		return nil, fmt.Errorf("parsing output template: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("rendering output template: %w", err)
	}
	if text := strings.TrimSpace(buf.String()); text != "" {
		return nil, fmt.Errorf(
			"output template rendered text %q; it may only produce output through the output functions",
			text,
		)
	}
	return b.build(), nil
}

// OutputFuncDoc documents a single output-template function. It mirrors
// [templateutil.FuncDoc]; the type is redeclared here because that package is
// internal and cannot appear in this package's exported signatures.
type OutputFuncDoc struct {
	// Name is the bare function name as registered in outputFuncMap.
	Name string
	// Usage is the function name together with its argument placeholders,
	// e.g. "blockDecision REASON".
	Usage string
	// Desc is a one-line human description, naming the events the function
	// applies to when it is event-scoped.
	Desc string
}

// OutputFuncDocs returns documentation for every output-template function, in
// a stable display order. It is the single source of truth behind
// [OutputFuncHelp] and the command help text; it is kept in sync with
// outputFuncMap (guarded by a test).
func OutputFuncDocs() []OutputFuncDoc {
	return []OutputFuncDoc{
		{
			Name:  "blockDecision",
			Usage: "blockDecision REASON",
			Desc:  "block the event with REASON shown to the agent",
		},
		{
			Name:  "stop",
			Usage: "stop REASON",
			Desc:  "halt the agent entirely with REASON (continue=false)",
		},
		{
			Name:  "systemMessage",
			Usage: "systemMessage MSG",
			Desc:  "warning MSG shown to the user, not to the agent",
		},
		{
			Name:  "suppressOutput",
			Usage: "suppressOutput",
			Desc:  "hide the hook's stdout from the transcript",
		},
		{
			Name:  "terminalSequence",
			Usage: "terminalSequence SEQ",
			Desc:  "terminal escape sequence for Claude Code to emit",
		},
		{
			Name:  "context",
			Usage: "context TEXT",
			Desc: "additional context for the agent; " +
				"PreToolUse, PostToolUse, PostToolUseFailure, PostToolBatch, Notification, " +
				"UserPromptSubmit, UserPromptExpansion, SessionStart, Setup, SubagentStart, " +
				"Stop, SubagentStop",
		},
		{
			Name:  "permission",
			Usage: "permission DECISION [REASON]",
			Desc:  "allow / deny / ask the pending tool call; PreToolUse",
		},
		{
			Name:  "updatedInput",
			Usage: "updatedInput JSON_OBJECT",
			Desc:  "replace the tool input with the given JSON object; PreToolUse",
		},
		{
			Name:  "updatedToolOutput",
			Usage: "updatedToolOutput JSON",
			Desc:  "replace the tool result with the given JSON; PostToolUse",
		},
		{
			Name:  "permissionAllow",
			Usage: "permissionAllow [UPDATED_INPUT_JSON [UPDATED_PERMISSIONS_JSON]]",
			Desc: "approve the permission request, optionally rewriting the tool " +
				"input and the permission rules; PermissionRequest",
		},
		{
			Name:  "permissionDeny",
			Usage: "permissionDeny MSG [INTERRUPT]",
			Desc:  "reject the permission request with MSG; PermissionRequest",
		},
		{
			Name:  "retry",
			Usage: "retry",
			Desc:  "ask the agent to retry the denied call; PermissionDenied",
		},
		{
			Name:  "sessionTitle",
			Usage: "sessionTitle TITLE",
			Desc:  "set the session title; SessionStart, UserPromptSubmit",
		},
		{
			Name:  "initialUserMessage",
			Usage: "initialUserMessage MSG",
			Desc:  "seed the session with MSG as the first user message; SessionStart",
		},
		{
			Name:  "watchPaths",
			Usage: "watchPaths PATH...",
			Desc:  "paths to watch for changes; SessionStart, CwdChanged, FileChanged",
		},
		{
			Name:  "reloadSkills",
			Usage: "reloadSkills",
			Desc:  "re-scan skill and command directories; SessionStart",
		},
		{
			Name:  "suppressOriginalPrompt",
			Usage: "suppressOriginalPrompt",
			Desc:  "omit the original prompt from the block message; UserPromptSubmit",
		},
		{
			Name:  "displayContent",
			Usage: "displayContent TEXT",
			Desc:  "text displayed in place of the message delta; MessageDisplay",
		},
		{
			Name:  "elicitation",
			Usage: "elicitation ACTION [JSON_OBJECT]",
			Desc: "accept / decline / cancel with optional content; " +
				"Elicitation, ElicitationResult",
		},
		{
			Name:  "worktreePath",
			Usage: "worktreePath PATH",
			Desc:  "path of the worktree to use; WorktreeCreate",
		},
	}
}

// OutputFuncHelp renders [OutputFuncDocs] as an aligned, indented block
// suitable for embedding in command help text. Each line is
// "  <usage>  <desc>" with the usage column padded to a common width; the
// block ends with a trailing newline.
func OutputFuncHelp() string {
	docs := OutputFuncDocs()
	width := 0
	for _, d := range docs {
		width = max(width, len(d.Usage))
	}
	var b strings.Builder
	for _, d := range docs {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, d.Usage, d.Desc)
	}
	return b.String()
}
