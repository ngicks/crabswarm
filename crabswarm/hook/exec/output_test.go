package exec

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ngicks/crabswarm/internal/templateutil"
	"github.com/ngicks/crabswarm/pkg/claudehook/types"
	"gotest.tools/v3/assert"
)

// outputDataFor builds the minimal OutputData an output-template test needs:
// the event drives every event-scoped function.
func outputDataFor(event types.HookEvent) OutputData {
	return OutputData{Data: Data{Event: event}}
}

// renderOutputOK renders src for event and fails the test on error.
func renderOutputOK(t *testing.T, event types.HookEvent, src string) *types.SyncHookJSONOutput {
	t.Helper()
	out, err := renderOutput(src, outputDataFor(event))
	assert.NilError(t, err)
	return out
}

// hsoAs returns the recorded hookSpecificOutput as PT.
func hsoAs[T any, PT hookSpecificPtr[T]](
	t *testing.T,
	out *types.SyncHookJSONOutput,
) PT {
	t.Helper()
	assert.Assert(t, out != nil, "expected an output, got nil (plain allow)")
	v, ok := out.HookSpecificOutput.GetValue().(PT)
	assert.Assert(t, ok, "hookSpecificOutput is %T", out.HookSpecificOutput.GetValue())
	return v
}

// deref fails the test when p is nil, otherwise returns *p.
func deref[T any](t *testing.T, p *T) T {
	t.Helper()
	assert.Assert(t, p != nil, "expected a recorded value, got nil")
	return *p
}

// contextEvents is every event whose hookSpecificOutput variant carries an
// additionalContext field; it mirrors the `context` type switch in output.go.
var contextEvents = []types.HookEvent{
	types.HookEventPreToolUse,
	types.HookEventPostToolUse,
	types.HookEventPostToolUseFailure,
	types.HookEventPostToolBatch,
	types.HookEventNotification,
	types.HookEventUserPromptSubmit,
	types.HookEventUserPromptExpansion,
	types.HookEventSessionStart,
	types.HookEventSetup,
	types.HookEventSubagentStart,
	types.HookEventStop,
	types.HookEventSubagentStop,
}

func TestRenderOutput_TopLevelFuncs(t *testing.T) {
	ev := types.HookEventPreToolUse

	out := renderOutputOK(t, ev, `{{ blockDecision "nope" }}`)
	assert.Equal(t, deref(t, out.Decision), types.HookDecisionBlock)
	assert.Equal(t, deref(t, out.Reason), "nope")

	// continue=false must survive: false is meaningful here, not "unset".
	out = renderOutputOK(t, ev, `{{ stop "halt" }}`)
	assert.Equal(t, deref(t, out.Continue), false)
	assert.Equal(t, deref(t, out.StopReason), "halt")

	out = renderOutputOK(t, ev, `{{ systemMessage "fyi" }}`)
	assert.Equal(t, deref(t, out.SystemMessage), "fyi")

	out = renderOutputOK(t, ev, `{{ suppressOutput }}`)
	assert.Equal(t, deref(t, out.SuppressOutput), true)

	out = renderOutputOK(t, ev, `{{ terminalSequence "\a" }}`)
	assert.Equal(t, deref(t, out.TerminalSequence), "\a")
}

func TestRenderOutput_TopLevelFuncsWorkForAnyEvent(t *testing.T) {
	// Top-level fields are event-agnostic — even an event with no
	// hookSpecificOutput variant at all can set them.
	out := renderOutputOK(t, types.HookEventSessionEnd, `{{ blockDecision "nope" }}`)
	assert.Equal(t, deref(t, out.Decision), types.HookDecisionBlock)
	assert.Assert(t, out.HookSpecificOutput.GetValue() == nil)
}

func TestRenderOutput_EventSpecificFuncs(t *testing.T) {
	t.Run("permission", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPreToolUse,
			`{{ permission "deny" "not allowed" }}`)
		v := hsoAs[types.HookSpecificOutputPreToolUse](t, out)
		assert.Equal(t, v.HookEventName, types.HookEventPreToolUse)
		assert.Equal(t, deref(t, v.PermissionDecision), types.HookPermissionDecisionDeny)
		assert.Equal(t, deref(t, v.PermissionDecisionReason), "not allowed")
	})

	t.Run("permission without reason", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPreToolUse, `{{ permission "ask" }}`)
		v := hsoAs[types.HookSpecificOutputPreToolUse](t, out)
		assert.Equal(t, deref(t, v.PermissionDecision), types.HookPermissionDecisionAsk)
		assert.Assert(t, v.PermissionDecisionReason == nil)
	})

	t.Run("updatedInput", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPreToolUse,
			`{{ updatedInput "{\"file_path\":\"/x\"}" }}`)
		v := hsoAs[types.HookSpecificOutputPreToolUse](t, out)
		assert.Equal(t, v.UpdatedInput["file_path"], "/x")
	})

	t.Run("updatedToolOutput", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPostToolUse,
			`{{ updatedToolOutput "[1,2]" }}`)
		v := hsoAs[types.HookSpecificOutputPostToolUse](t, out)
		assert.Equal(t, string(v.UpdatedToolOutput), "[1,2]")
	})

	t.Run("permissionAllow", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPermissionRequest, `{{ permissionAllow }}`)
		v := hsoAs[types.HookSpecificOutputPermissionRequest](t, out)
		allow, ok := v.Decision.GetValue().(*types.PermissionRequestDecisionAllow)
		assert.Assert(t, ok, "decision is %T", v.Decision.GetValue())
		assert.Equal(t, allow.Behavior, types.PermissionRequestDecisionBehaviorAllow)
	})

	t.Run("permissionAllow with updatedInput", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPermissionRequest,
			`{{ permissionAllow "{\"file_path\":\"/x\"}" }}`)
		v := hsoAs[types.HookSpecificOutputPermissionRequest](t, out)
		allow, ok := v.Decision.GetValue().(*types.PermissionRequestDecisionAllow)
		assert.Assert(t, ok, "decision is %T", v.Decision.GetValue())
		assert.Equal(t, allow.UpdatedInput["file_path"], "/x")
		assert.Assert(t, allow.UpdatedPermissions == nil)
	})

	t.Run("permissionAllow with updatedPermissions", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPermissionRequest,
			`{{ permissionAllow "{}" `+
				`"[{\"type\":\"setMode\",\"mode\":\"acceptEdits\",\"destination\":\"session\"}]" }}`)
		v := hsoAs[types.HookSpecificOutputPermissionRequest](t, out)
		allow, ok := v.Decision.GetValue().(*types.PermissionRequestDecisionAllow)
		assert.Assert(t, ok, "decision is %T", v.Decision.GetValue())
		assert.Equal(t, len(allow.UpdatedPermissions), 1)
		mode, ok := allow.UpdatedPermissions[0].GetValue().(*types.PermissionUpdateSetMode)
		assert.Assert(t, ok, "update is %T", allow.UpdatedPermissions[0].GetValue())
		assert.Equal(t, mode.Mode, types.PermissionModeAcceptEdits)
	})

	t.Run("permissionDeny", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPermissionRequest,
			`{{ permissionDeny "no" true }}`)
		v := hsoAs[types.HookSpecificOutputPermissionRequest](t, out)
		deny, ok := v.Decision.GetValue().(*types.PermissionRequestDecisionDeny)
		assert.Assert(t, ok, "decision is %T", v.Decision.GetValue())
		assert.Equal(t, deny.Behavior, types.PermissionRequestDecisionBehaviorDeny)
		assert.Equal(t, deref(t, deny.Message), "no")
		assert.Equal(t, deref(t, deny.Interrupt), true)
	})

	t.Run("permissionDeny interrupt=false stays recorded", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPermissionRequest,
			`{{ permissionDeny "no" false }}`)
		v := hsoAs[types.HookSpecificOutputPermissionRequest](t, out)
		deny, ok := v.Decision.GetValue().(*types.PermissionRequestDecisionDeny)
		assert.Assert(t, ok, "decision is %T", v.Decision.GetValue())
		assert.Equal(t, deref(t, deny.Interrupt), false)
	})

	t.Run("permissionDeny without interrupt", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPermissionRequest, `{{ permissionDeny "no" }}`)
		v := hsoAs[types.HookSpecificOutputPermissionRequest](t, out)
		deny, ok := v.Decision.GetValue().(*types.PermissionRequestDecisionDeny)
		assert.Assert(t, ok, "decision is %T", v.Decision.GetValue())
		assert.Assert(t, deny.Interrupt == nil)
	})

	t.Run("retry", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPermissionDenied, `{{ retry }}`)
		v := hsoAs[types.HookSpecificOutputPermissionDenied](t, out)
		assert.Equal(t, deref(t, v.Retry), true)
	})

	t.Run("sessionTitle on SessionStart", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventSessionStart, `{{ sessionTitle "t" }}`)
		v := hsoAs[types.HookSpecificOutputSessionStart](t, out)
		assert.Equal(t, deref(t, v.SessionTitle), "t")
	})

	t.Run("sessionTitle on UserPromptSubmit", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventUserPromptSubmit, `{{ sessionTitle "t" }}`)
		v := hsoAs[types.HookSpecificOutputUserPromptSubmit](t, out)
		assert.Equal(t, deref(t, v.SessionTitle), "t")
	})

	t.Run("initialUserMessage", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventSessionStart, `{{ initialUserMessage "hi" }}`)
		v := hsoAs[types.HookSpecificOutputSessionStart](t, out)
		assert.Equal(t, deref(t, v.InitialUserMessage), "hi")
	})

	t.Run("watchPaths", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventFileChanged, `{{ watchPaths "/a" "/b" }}`)
		v := hsoAs[types.HookSpecificOutputFileChanged](t, out)
		assert.DeepEqual(t, v.WatchPaths, []string{"/a", "/b"})
	})

	t.Run("reloadSkills", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventSessionStart, `{{ reloadSkills }}`)
		v := hsoAs[types.HookSpecificOutputSessionStart](t, out)
		assert.Equal(t, deref(t, v.ReloadSkills), true)
	})

	t.Run("suppressOriginalPrompt", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventUserPromptSubmit, `{{ suppressOriginalPrompt }}`)
		v := hsoAs[types.HookSpecificOutputUserPromptSubmit](t, out)
		assert.Equal(t, deref(t, v.SuppressOriginalPrompt), true)
	})

	t.Run("displayContent", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventMessageDisplay, `{{ displayContent "shown" }}`)
		v := hsoAs[types.HookSpecificOutputMessageDisplay](t, out)
		assert.Equal(t, deref(t, v.DisplayContent), "shown")
	})

	t.Run("elicitation", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventElicitation,
			`{{ elicitation "accept" "{\"a\":1}" }}`)
		v := hsoAs[types.HookSpecificOutputElicitation](t, out)
		assert.Equal(t, deref(t, v.Action), types.HookSpecificOutputElicitationActionAccept)
		assert.Equal(t, v.Content["a"], float64(1))
	})

	t.Run("elicitation result", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventElicitationResult, `{{ elicitation "cancel" }}`)
		v := hsoAs[types.HookSpecificOutputElicitationResult](t, out)
		assert.Equal(t, deref(t, v.Action),
			types.HookSpecificOutputElicitationResultActionCancel)
		assert.Assert(t, v.Content == nil)
	})

	t.Run("worktreePath", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventWorktreeCreate, `{{ worktreePath "/wt" }}`)
		v := hsoAs[types.HookSpecificOutputWorktreeCreate](t, out)
		assert.Equal(t, v.WorktreePath, "/wt")
	})
}

func TestRenderOutput_ContextCoversEveryAdditionalContextEvent(t *testing.T) {
	// Reflection keeps this table honest without duplicating the type
	// switch: every listed event must yield a variant whose
	// AdditionalContext ends up set.
	for _, ev := range contextEvents {
		t.Run(string(ev), func(t *testing.T) {
			out := renderOutputOK(t, ev, `{{ context "extra" }}`)
			v := out.HookSpecificOutput.GetValue()
			assert.Assert(t, v != nil)
			f := reflect.ValueOf(v).Elem().FieldByName("AdditionalContext")
			assert.Assert(t, f.IsValid(), "%T has no AdditionalContext field", v)
			assert.Assert(t, !f.IsNil(), "%T left AdditionalContext unset", v)
			assert.Equal(t, f.Elem().String(), "extra")
		})
	}
}

func TestRenderOutput_EventMismatchIsAnError(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event types.HookEvent
		src   string
	}{
		{"permission", types.HookEventPostToolUse, `{{ permission "allow" }}`},
		{"updatedInput", types.HookEventPostToolUse, `{{ updatedInput "{}" }}`},
		{"updatedToolOutput", types.HookEventPreToolUse, `{{ updatedToolOutput "{}" }}`},
		{"permissionAllow", types.HookEventPreToolUse, `{{ permissionAllow }}`},
		{"permissionDeny", types.HookEventPreToolUse, `{{ permissionDeny "no" }}`},
		{"retry", types.HookEventPreToolUse, `{{ retry }}`},
		{"sessionTitle", types.HookEventPreToolUse, `{{ sessionTitle "t" }}`},
		{"initialUserMessage", types.HookEventUserPromptSubmit, `{{ initialUserMessage "m" }}`},
		{"watchPaths", types.HookEventPreToolUse, `{{ watchPaths "/a" }}`},
		{"reloadSkills", types.HookEventUserPromptSubmit, `{{ reloadSkills }}`},
		{"suppressOriginalPrompt", types.HookEventSessionStart, `{{ suppressOriginalPrompt }}`},
		{"displayContent", types.HookEventPreToolUse, `{{ displayContent "x" }}`},
		{"elicitation", types.HookEventPreToolUse, `{{ elicitation "accept" }}`},
		{"worktreePath", types.HookEventPreToolUse, `{{ worktreePath "/wt" }}`},
		// The event has a variant, but it carries no additionalContext.
		{"context on variant without the field", types.HookEventWorktreeCreate,
			`{{ context "x" }}`},
		// The event has no hookSpecificOutput variant at all.
		{"context on event without a variant", types.HookEventSessionEnd, `{{ context "x" }}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := renderOutput(tc.src, outputDataFor(tc.event))
			assert.Assert(t, err != nil, "expected an event-mismatch error")
			assert.Assert(t, strings.Contains(err.Error(), "not supported for hook event"),
				"got %v", err)
		})
	}
}

func TestRenderOutput_ArgumentValidation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event types.HookEvent
		src   string
		want  string
	}{
		{"unknown permission decision", types.HookEventPreToolUse,
			`{{ permission "maybe" }}`, "unknown decision"},
		{"unknown elicitation action", types.HookEventElicitation,
			`{{ elicitation "maybe" }}`, "unknown action"},
		{"unknown elicitation result action", types.HookEventElicitationResult,
			`{{ elicitation "maybe" }}`, "unknown action"},
		{"malformed updatedInput", types.HookEventPreToolUse,
			`{{ updatedInput "not json" }}`, "as a JSON object"},
		{"malformed updatedToolOutput", types.HookEventPostToolUse,
			`{{ updatedToolOutput "not json" }}`, "as JSON"},
		{"malformed elicitation content", types.HookEventElicitation,
			`{{ elicitation "accept" "not json" }}`, "as a JSON object"},
		{"watchPaths without a path", types.HookEventSessionStart,
			`{{ watchPaths }}`, "at least one PATH"},
		{"malformed permissionAllow updatedInput", types.HookEventPermissionRequest,
			`{{ permissionAllow "not json" }}`, "as a JSON object"},
		{"malformed permissionAllow updatedPermissions", types.HookEventPermissionRequest,
			`{{ permissionAllow "{}" "not json" }}`, "as a JSON array"},
		{"permissionAllow with too many arguments", types.HookEventPermissionRequest,
			`{{ permissionAllow "{}" "[]" "extra" }}`, "at most 2 arguments"},
		// Every optional trailing argument is bounded: text/template happily
		// passes extras to a variadic func, so an unchecked one would drop a
		// template typo instead of reporting it.
		{"permission with too many arguments", types.HookEventPreToolUse,
			`{{ permission "allow" "because" "extra" }}`, "at most 2 arguments"},
		{"permissionDeny with too many arguments", types.HookEventPermissionRequest,
			`{{ permissionDeny "no" true false }}`, "at most 2 arguments"},
		{"elicitation with too many arguments", types.HookEventElicitation,
			`{{ elicitation "accept" "{}" "extra" }}`, "at most 2 arguments"},
		{"elicitation result with too many arguments", types.HookEventElicitationResult,
			`{{ elicitation "accept" "{}" "extra" }}`, "at most 2 arguments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := renderOutput(tc.src, outputDataFor(tc.event))
			assert.Assert(t, err != nil, "expected a validation error")
			assert.Assert(t, strings.Contains(err.Error(), tc.want), "got %v", err)
		})
	}
}

func TestRenderOutput_NoFuncFiredIsPlainAllow(t *testing.T) {
	out, err := renderOutput(`{{ if .Success }}{{ end }}`,
		OutputData{Data: Data{Event: types.HookEventPreToolUse}, Success: true})
	assert.NilError(t, err)
	assert.Assert(t, out == nil, "expected nil output (plain allow), got %+v", out)
}

func TestRenderOutput_WhitespaceOnlyRenderIsAccepted(t *testing.T) {
	// Templates are readable when they may span lines; only non-whitespace
	// text is the sanctioned-path violation.
	out := renderOutputOK(t, types.HookEventPreToolUse, "  \n\t{{ blockDecision \"nope\" }}\n  ")
	assert.Equal(t, deref(t, out.Reason), "nope")
}

func TestRenderOutput_StrayTextIsAnError(t *testing.T) {
	_, err := renderOutput(
		`{{ blockDecision "nope" }}oops`,
		outputDataFor(types.HookEventPreToolUse),
	)
	assert.Assert(t, err != nil, "expected stray text to fail the render")
	assert.Assert(t, strings.Contains(err.Error(), `rendered text "oops"`), "got %v", err)
}

func TestRenderOutput_UndefinedFuncFailsAtParse(t *testing.T) {
	_, err := renderOutput(`{{ blokDecision "nope" }}`, outputDataFor(types.HookEventPreToolUse))
	assert.Assert(t, err != nil)
	assert.Assert(t, strings.Contains(err.Error(), "parsing output template"), "got %v", err)
}

func TestRenderOutput_ExposesResultAndCommandTemplateData(t *testing.T) {
	data := OutputData{
		Data:     Data{Event: types.HookEventPostToolUse, File: "/pkg/x.go"},
		Command:  "go vet ./...",
		ExitCode: 1,
		Output:   "combined",
		Stdout:   "out",
		Stderr:   "err",
	}
	src := `{{ blockDecision (printf "%s|%d|%v|%s|%s|%s|%s" ` +
		`.Command .ExitCode .Success .Output .Stdout .Stderr (basename .File)) }}`
	out, err := renderOutput(src, data)
	assert.NilError(t, err)
	assert.Equal(t, deref(t, out.Reason), "go vet ./...|1|false|combined|out|err|x.go")
}

func TestRenderOutput_LastCallWins(t *testing.T) {
	out := renderOutputOK(t, types.HookEventPreToolUse,
		`{{ blockDecision "first" }}{{ blockDecision "second" }}`)
	assert.Equal(t, deref(t, out.Reason), "second")
}

func TestRenderOutput_OmittedOptionalArgClearsPreviousCall(t *testing.T) {
	// Last-wins has to cover the optional trailing arguments too: leaving the
	// earlier call's value attached to the later decision would report
	// something no call actually asked for.
	t.Run("permission reason", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventPreToolUse,
			`{{ permission "deny" "not allowed" }}{{ permission "allow" }}`)
		v := hsoAs[types.HookSpecificOutputPreToolUse](t, out)
		assert.Equal(t, deref(t, v.PermissionDecision), types.HookPermissionDecisionAllow)
		assert.Assert(t, v.PermissionDecisionReason == nil,
			"reason: %v", v.PermissionDecisionReason)
	})

	t.Run("elicitation content", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventElicitation,
			`{{ elicitation "accept" "{\"a\":1}" }}{{ elicitation "decline" }}`)
		v := hsoAs[types.HookSpecificOutputElicitation](t, out)
		assert.Equal(t, deref(t, v.Action), types.HookSpecificOutputElicitationActionDecline)
		assert.Assert(t, v.Content == nil, "content: %v", v.Content)
	})

	t.Run("elicitation result content", func(t *testing.T) {
		out := renderOutputOK(t, types.HookEventElicitationResult,
			`{{ elicitation "accept" "{\"a\":1}" }}{{ elicitation "cancel" }}`)
		v := hsoAs[types.HookSpecificOutputElicitationResult](t, out)
		assert.Equal(t, deref(t, v.Action),
			types.HookSpecificOutputElicitationResultActionCancel)
		assert.Assert(t, v.Content == nil, "content: %v", v.Content)
	})
}

func TestRenderOutput_AccumulatesAcrossFuncs(t *testing.T) {
	// Several calls compose onto one output: top-level fields and repeated
	// touches of the same hookSpecificOutput variant.
	out := renderOutputOK(t, types.HookEventSessionStart,
		`{{ systemMessage "note" }}{{ context "extra" }}{{ sessionTitle "t" }}`)
	assert.Equal(t, deref(t, out.SystemMessage), "note")
	v := hsoAs[types.HookSpecificOutputSessionStart](t, out)
	assert.Equal(t, deref(t, v.AdditionalContext), "extra")
	assert.Equal(t, deref(t, v.SessionTitle), "t")
}

func TestRenderOutput_TemplateutilFuncsRemainAvailable(t *testing.T) {
	out := renderOutputOK(t, types.HookEventPreToolUse, `{{ blockDecision (trim "  padded  ") }}`)
	assert.Equal(t, deref(t, out.Reason), "padded")
}

func TestOutputFuncMap_DoesNotCollideWithTemplateutil(t *testing.T) {
	// The two maps are merged in renderOutput; an overlapping name would
	// silently shadow one side.
	generic := templateutil.FuncMap()
	for name := range outputFuncMap(&outputBuilder{}) {
		if _, ok := generic[name]; ok {
			t.Errorf("output func %q collides with templateutil.FuncMap", name)
		}
	}
}

func TestOutputFuncDocs_MatchesFuncMap(t *testing.T) {
	fm := outputFuncMap(&outputBuilder{})
	docs := OutputFuncDocs()

	if len(docs) != len(fm) {
		t.Fatalf("OutputFuncDocs has %d entries, outputFuncMap has %d", len(docs), len(fm))
	}
	seen := map[string]bool{}
	for _, d := range docs {
		if _, ok := fm[d.Name]; !ok {
			t.Errorf("OutputFuncDocs documents %q which is not in outputFuncMap", d.Name)
		}
		if seen[d.Name] {
			t.Errorf("OutputFuncDocs has duplicate entry for %q", d.Name)
		}
		seen[d.Name] = true
		if d.Usage == "" || d.Desc == "" {
			t.Errorf("OutputFuncDocs entry %q has empty Usage or Desc", d.Name)
		}
		if !strings.HasPrefix(d.Usage, d.Name) {
			t.Errorf("OutputFuncDocs entry %q has usage %q not starting with the name",
				d.Name, d.Usage)
		}
	}
	for name := range fm {
		if !seen[name] {
			t.Errorf("outputFuncMap has %q with no OutputFuncDocs entry", name)
		}
	}
}

func TestOutputFuncHelp_RendersEveryFunc(t *testing.T) {
	help := OutputFuncHelp()
	if !strings.HasSuffix(help, "\n") {
		t.Error("OutputFuncHelp should end with a trailing newline")
	}
	for _, d := range OutputFuncDocs() {
		if !strings.Contains(help, d.Usage) {
			t.Errorf("OutputFuncHelp missing usage %q", d.Usage)
		}
		if !strings.Contains(help, d.Desc) {
			t.Errorf("OutputFuncHelp missing desc for %q", d.Name)
		}
	}
}
