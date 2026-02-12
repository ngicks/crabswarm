package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"github.com/ngicks/crabswarm/hook/internal/server"
)

type permissionModel struct {
	req         *pb.PermissionRequest
	cursor      int
	choices     []string
	inputReason bool
	reasonInput textinput.Model
	prettyJSON  string
	width       int
	height      int
}

func newPermissionModel(req *pb.PermissionRequest, width, height int) permissionModel {
	ti := textinput.New()
	ti.Placeholder = "Reason (optional, press Enter to skip)"
	ti.CharLimit = 256
	ti.Width = 50

	var prettyJSON string
	if req.ToolInputJson != "" {
		var obj map[string]any
		if err := json.Unmarshal([]byte(req.ToolInputJson), &obj); err == nil {
			formatted, _ := json.MarshalIndent(obj, "", "  ")
			prettyJSON = string(formatted)
		} else {
			prettyJSON = req.ToolInputJson
		}
	}

	return permissionModel{
		req:         req,
		cursor:      0,
		choices:     []string{"Allow", "Deny", "Ask"},
		reasonInput: ti,
		prettyJSON:  prettyJSON,
		width:       width,
		height:      height,
	}
}

func (m permissionModel) Update(msg tea.Msg) (permissionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inputReason {
			return m.updateReasonInput(msg)
		}
		return m.updateChoiceSelection(msg)
	}
	return m, nil
}

func (m permissionModel) updateChoiceSelection(msg tea.KeyMsg) (permissionModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		if m.cursor > 0 {
			m.cursor--
		}
	case tea.KeyDown:
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}
	case tea.KeyEnter:
		return m.selectChoice()
	case tea.KeyRunes:
		switch msg.String() {
		case "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "a":
			m.cursor = 0
			return m.selectChoice()
		case "d":
			m.cursor = 1
			return m.selectChoice()
		case " ":
			return m.selectChoice()
		}
	}
	return m, nil
}

func (m permissionModel) selectChoice() (permissionModel, tea.Cmd) {
	switch m.choices[m.cursor] {
	case "Allow":
		resp := server.BuildPermissionResponse(m.req, pb.PermissionDecision_PERMISSION_DECISION_ALLOW, "")
		return m, func() tea.Msg { return promptCompleteMsg{response: resp} }
	case "Deny":
		m.inputReason = true
		m.reasonInput.Focus()
		return m, textinput.Blink
	case "Ask":
		resp := server.BuildPermissionResponse(m.req, pb.PermissionDecision_PERMISSION_DECISION_ASK, "")
		return m, func() tea.Msg { return promptCompleteMsg{response: resp} }
	}
	return m, nil
}

func (m permissionModel) updateReasonInput(msg tea.KeyMsg) (permissionModel, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		reason := m.reasonInput.Value()
		resp := server.BuildPermissionResponse(m.req, pb.PermissionDecision_PERMISSION_DECISION_DENY, reason)
		return m, func() tea.Msg { return promptCompleteMsg{response: resp} }
	case tea.KeyEsc:
		m.inputReason = false
		m.reasonInput.Blur()
		m.reasonInput.Reset()
		return m, nil
	}

	var cmd tea.Cmd
	m.reasonInput, cmd = m.reasonInput.Update(msg)
	return m, cmd
}

func (m permissionModel) View() string {
	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render(fmt.Sprintf("Permission Request: %s", m.req.HookEventName)))
	b.WriteString("\n\n")

	// Tool info
	b.WriteString(fmt.Sprintf("  Tool:    %s\n", toolNameStyle.Render(m.req.ToolName)))
	b.WriteString(fmt.Sprintf("  Session: %s\n", m.req.SessionId))
	if m.req.Cwd != "" {
		b.WriteString(fmt.Sprintf("  Cwd:     %s\n", m.req.Cwd))
	}

	// JSON input
	if m.prettyJSON != "" {
		b.WriteString("\n")
		b.WriteString(jsonStyle.Render(m.prettyJSON))
		b.WriteString("\n")
	}

	b.WriteString("\n")

	if m.inputReason {
		b.WriteString("  Enter deny reason (Esc to cancel):\n\n")
		b.WriteString("  " + m.reasonInput.View())
		b.WriteString("\n")
	} else {
		for i, choice := range m.choices {
			cursor := "  "
			if m.cursor == i {
				cursor = cursorStyle.Render("> ")
			}

			shortcut := ""
			switch choice {
			case "Allow":
				shortcut = "(a)"
			case "Deny":
				shortcut = "(d)"
			case "Ask":
				shortcut = "(k)"
			}

			if m.cursor == i {
				b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, selectedStyle.Render(choice), unselectedStyle.Render(shortcut)))
			} else {
				b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, unselectedStyle.Render(choice), unselectedStyle.Render(shortcut)))
			}
		}
	}

	return b.String()
}
