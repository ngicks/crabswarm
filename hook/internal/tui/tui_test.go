package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"github.com/ngicks/crabswarm/hook/internal/server"
)

type testReq struct {
	msg     permissionRequestMsg
	readyCh chan permissionResult
}

func makeReq(toolName, toolInputJSON string) testReq {
	ch := make(chan permissionResult, 1)
	return testReq{
		msg: permissionRequestMsg{
			req: &pb.PermissionRequest{
				HookEventName: "PreToolUse",
				ToolName:      toolName,
				ToolInputJson: toolInputJSON,
				SessionId:     "test-session",
				MessageId:     "test-message",
			},
			replyCh: ch,
		},
		readyCh: ch,
	}
}

// initModel creates a rootModel and sends a WindowSizeMsg to initialize the viewport.
func initModel(width, height int) rootModel {
	m := rootModel{}
	result, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return result.(rootModel)
}

func TestRootModel_IdleState(t *testing.T) {
	m := rootModel{}
	if m.state != stateIdle {
		t.Errorf("initial state = %d, want idle", m.state)
	}

	view := m.View()
	if view == "" {
		t.Error("idle view should not be empty")
	}
}

func TestRootModel_PermissionRequest(t *testing.T) {
	m := rootModel{}
	tr := makeReq("Bash", `{"command":"ls"}`)

	result, _ := m.Update(tr.msg)
	m = result.(rootModel)

	if m.state != statePermission {
		t.Errorf("state = %d, want statePermission", m.state)
	}
	if m.replyCh == nil {
		t.Error("replyCh should be set")
	}

	view := m.View()
	if view == "" {
		t.Error("permission view should not be empty")
	}
}

func TestRootModel_AskUserRequest(t *testing.T) {
	m := rootModel{}
	inputJSON := `{"questions":[{"question":"Which?","header":"Q","options":[{"label":"A"},{"label":"B"}],"multiSelect":false}]}`
	tr := makeReq("AskUserQuestion", inputJSON)

	result, _ := m.Update(tr.msg)
	m = result.(rootModel)

	if m.state != stateAskUser {
		t.Errorf("state = %d, want stateAskUser", m.state)
	}
}

func TestRootModel_Queueing(t *testing.T) {
	m := rootModel{}

	// First request activates
	tr1 := makeReq("Bash", `{"command":"ls"}`)
	result, _ := m.Update(tr1.msg)
	m = result.(rootModel)

	if m.state != statePermission {
		t.Fatalf("state = %d, want statePermission", m.state)
	}

	// Second request is queued
	tr2 := makeReq("Write", `{"file_path":"/tmp/test"}`)
	result, _ = m.Update(tr2.msg)
	m = result.(rootModel)

	if len(m.queuedReqs) != 1 {
		t.Errorf("queued = %d, want 1", len(m.queuedReqs))
	}

	// Complete first request → dequeues second
	result, _ = m.Update(promptCompleteMsg{
		response: &pb.PermissionResponse{ShouldContinue: true},
	})
	m = result.(rootModel)

	// First reply should be sent
	select {
	case r := <-tr1.readyCh:
		if r.response == nil {
			t.Error("expected non-nil response for first request")
		}
	default:
		t.Error("expected reply on first request's channel")
	}

	if m.state != statePermission {
		t.Errorf("state after dequeue = %d, want statePermission", m.state)
	}
	if len(m.queuedReqs) != 0 {
		t.Errorf("queued after dequeue = %d, want 0", len(m.queuedReqs))
	}
}

func TestPermissionModel_CursorNav(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInputJson: `{"command":"rm -rf /"}`,
	}
	m := newPermissionModel(req, 80, 24)

	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", m.cursor)
	}

	// Move down again
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("cursor after second down = %d, want 2", m.cursor)
	}

	// Move down at boundary (should stay)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 2 {
		t.Errorf("cursor at bottom boundary = %d, want 2", m.cursor)
	}

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 1 {
		t.Errorf("cursor after up = %d, want 1", m.cursor)
	}
}

func TestPermissionModel_AllowShortcut(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
	}
	m := newPermissionModel(req, 80, 24)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if cmd == nil {
		t.Fatal("expected command from 'a' shortcut")
	}

	msg := cmd()
	complete, ok := msg.(promptCompleteMsg)
	if !ok {
		t.Fatal("expected promptCompleteMsg")
	}
	if complete.response.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ALLOW {
		t.Error("expected ALLOW decision")
	}
}

func TestPermissionModel_DenyWithReason(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
	}
	m := newPermissionModel(req, 80, 24)

	// Press 'd' to enter deny reason mode
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if !m.inputReason {
		t.Fatal("expected inputReason=true after 'd'")
	}
	if cmd == nil {
		t.Fatal("expected blink command")
	}

	// Type a reason
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	// Confirm
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command after enter in reason mode")
	}

	msg := cmd()
	complete, ok := msg.(promptCompleteMsg)
	if !ok {
		t.Fatal("expected promptCompleteMsg")
	}
	if complete.response.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_DENY {
		t.Error("expected DENY decision")
	}
	if complete.response.HookSpecificOutput.PermissionDecisionReason != "bad" {
		t.Errorf("reason = %q, want %q", complete.response.HookSpecificOutput.PermissionDecisionReason, "bad")
	}
}

func TestAskUserModel_SingleSelect(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "AskUserQuestion",
		ToolInputJson: `{"questions":[{"question":"Pick one?","header":"Q","options":[{"label":"A"},{"label":"B"}],"multiSelect":false}]}`,
	}
	input, err := server.ParseAskUserInput(req.ToolInputJson)
	if err != nil {
		t.Fatal(err)
	}
	m := newAskUserModel(req, input, 80, 24)

	// Select first option (cursor already at 0)
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command after enter")
	}

	msg := cmd()
	complete, ok := msg.(promptCompleteMsg)
	if !ok {
		t.Fatal("expected promptCompleteMsg")
	}
	if complete.response == nil {
		t.Fatal("expected non-nil response")
	}
	if complete.response.HookSpecificOutput.PermissionDecision != pb.PermissionDecision_PERMISSION_DECISION_ALLOW {
		t.Error("expected ALLOW")
	}
}

func TestAskUserModel_MultiSelect(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "AskUserQuestion",
		ToolInputJson: `{"questions":[{"question":"Which?","header":"Q","options":[{"label":"A"},{"label":"B"},{"label":"C"}],"multiSelect":true}]}`,
	}
	input, err := server.ParseAskUserInput(req.ToolInputJson)
	if err != nil {
		t.Fatal(err)
	}
	m := newAskUserModel(req, input, 80, 24)

	// Toggle first option (space)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.selected[0] {
		t.Error("expected option 0 selected")
	}

	// Move down and toggle third
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if !m.selected[2] {
		t.Error("expected option 2 selected")
	}

	// Confirm
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command after enter")
	}

	msg := cmd()
	complete, ok := msg.(promptCompleteMsg)
	if !ok {
		t.Fatal("expected promptCompleteMsg")
	}
	if complete.response == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestAskUserModel_CustomInput(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "AskUserQuestion",
		ToolInputJson: `{"questions":[{"question":"Pick?","header":"Q","options":[{"label":"A"}],"multiSelect":false}]}`,
	}
	input, err := server.ParseAskUserInput(req.ToolInputJson)
	if err != nil {
		t.Fatal(err)
	}
	m := newAskUserModel(req, input, 80, 24)

	// Navigate to "Other" (index 1, since 1 option + Other)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("cursor = %d, want 1", m.cursor)
	}

	// Select "Other"
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.customMode {
		t.Fatal("expected customMode=true")
	}

	// Type custom answer
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	// Confirm
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command after custom input confirm")
	}

	msg := cmd()
	complete, ok := msg.(promptCompleteMsg)
	if !ok {
		t.Fatal("expected promptCompleteMsg")
	}
	if complete.response == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestPermissionModel_View(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "Bash",
		ToolInputJson: `{"command":"ls -la"}`,
		SessionId:     "sess-123",
		Cwd:           "/home/user",
	}
	m := newPermissionModel(req, 80, 24)
	view := m.View()

	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestAskUserModel_View(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "AskUserQuestion",
		ToolInputJson: `{"questions":[{"question":"Which?","header":"Q","options":[{"label":"A","description":"Option A"},{"label":"B"}],"multiSelect":false}]}`,
	}
	input, err := server.ParseAskUserInput(req.ToolInputJson)
	if err != nil {
		t.Fatal(err)
	}
	m := newAskUserModel(req, input, 80, 24)
	view := m.View()

	if view == "" {
		t.Error("view should not be empty")
	}
}

func TestAskUserModel_ViewAfterComplete(t *testing.T) {
	req := &pb.PermissionRequest{
		HookEventName: "PreToolUse",
		ToolName:      "AskUserQuestion",
		ToolInputJson: `{"questions":[{"question":"Pick one?","header":"Q","options":[{"label":"A"},{"label":"B"}],"multiSelect":false}]}`,
	}
	input, err := server.ParseAskUserInput(req.ToolInputJson)
	if err != nil {
		t.Fatal(err)
	}
	m := newAskUserModel(req, input, 80, 24)

	// Select first option to complete the only question
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected command after enter")
	}
	if !m.completed {
		t.Fatal("expected completed=true after answering all questions")
	}

	// View() must not panic and should contain "Completing"
	view := m.View()
	if !strings.Contains(view, "Completing") {
		t.Errorf("expected view to contain 'Completing', got: %s", view)
	}

	// Update after completion should be a no-op
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Error("expected nil cmd after update on completed model")
	}
}

func TestRootModel_AuditEventAppend(t *testing.T) {
	m := initModel(80, 40)

	if !m.vpReady {
		t.Fatal("viewport should be ready after WindowSizeMsg")
	}

	// Send audit events
	result, _ := m.Update(auditEventMsg{line: "[12:00:00] PreToolUse  tool=Bash  session=s1"})
	m = result.(rootModel)

	if len(m.logLines) != 1 {
		t.Fatalf("logLines = %d, want 1", len(m.logLines))
	}

	result, _ = m.Update(auditEventMsg{line: "[12:00:01] PreToolUse  tool=Read  session=s1"})
	m = result.(rootModel)

	if len(m.logLines) != 2 {
		t.Fatalf("logLines = %d, want 2", len(m.logLines))
	}

	view := m.View()
	if !strings.Contains(view, "Bash") {
		t.Error("view should contain audit event with Bash")
	}
	if !strings.Contains(view, "Read") {
		t.Error("view should contain audit event with Read")
	}
	if !strings.Contains(view, "Audit Log") {
		t.Error("view should contain log panel header")
	}
}

func TestRootModel_PgUpPgDown(t *testing.T) {
	m := initModel(80, 20)

	// Add many lines to exceed viewport
	for i := 0; i < 50; i++ {
		result, _ := m.Update(auditEventMsg{line: strings.Repeat("x", 10)})
		m = result.(rootModel)
	}

	if len(m.logLines) != 50 {
		t.Fatalf("logLines = %d, want 50", len(m.logLines))
	}

	// PgUp should not panic
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	m = result.(rootModel)

	// PgDown should not panic
	result, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = result.(rootModel)

	// View should still render
	view := m.View()
	if view == "" {
		t.Error("view should not be empty after PgUp/PgDn")
	}
}

func TestRootModel_ViewportResizeOnStateChange(t *testing.T) {
	m := initModel(80, 40)

	idleHeight := m.viewport.Height
	if idleHeight != 38 { // 40 - 2 (header + status)
		t.Errorf("idle viewport height = %d, want 38", idleHeight)
	}

	// Activate a permission request
	tr := makeReq("Bash", `{"command":"ls"}`)
	result, _ := m.Update(tr.msg)
	m = result.(rootModel)

	activeHeight := m.viewport.Height
	expectedActive := 40 - promptAreaHeight // 40 - 16 = 24
	if activeHeight != expectedActive {
		t.Errorf("active viewport height = %d, want %d", activeHeight, expectedActive)
	}

	// Complete the request → back to idle
	result, _ = m.Update(promptCompleteMsg{
		response: &pb.PermissionResponse{ShouldContinue: true},
	})
	m = result.(rootModel)

	restoredHeight := m.viewport.Height
	if restoredHeight != idleHeight {
		t.Errorf("restored viewport height = %d, want %d", restoredHeight, idleHeight)
	}
}
