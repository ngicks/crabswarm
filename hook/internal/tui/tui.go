package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"github.com/ngicks/crabswarm/hook/internal/server"
)

// State represents the current view state.
type state int

const (
	stateIdle       state = iota
	statePermission       // standard permission prompt
	stateAskUser          // AskUserQuestion prompt
	stateExitPlan         // ExitPlanMode plan approval prompt
)

// promptAreaHeight is the number of lines reserved for the prompt panel.
const promptAreaHeight = 16

// permissionRequestMsg is sent from gRPC goroutines into bubbletea via Program.Send().
type permissionRequestMsg struct {
	req     *pb.PermissionRequest
	replyCh chan<- permissionResult
}

// permissionResult carries the response back to the gRPC handler.
type permissionResult struct {
	response *pb.PermissionResponse
	err      error
}

// promptCompleteMsg is emitted by sub-models when the user completes a prompt.
type promptCompleteMsg struct {
	response *pb.PermissionResponse
	err      error
}

// rootModel is the top-level bubbletea model.
type rootModel struct {
	state      state
	permModel  permissionModel
	askModel   askUserModel
	exitModel  exitPlanModel
	replyCh    chan<- permissionResult
	queuedReqs []permissionRequestMsg
	width      int
	height     int

	viewport viewport.Model
	logLines []string
	vpReady  bool
}

func (m rootModel) Init() tea.Cmd {
	return nil
}

func (m rootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.vpReady {
			m.viewport = viewport.New(msg.Width, m.viewportHeight())
			m.viewport.MouseWheelEnabled = true
			m.syncViewportContent()
			m.vpReady = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = m.viewportHeight()
		}
		return m, nil

	case auditEventMsg:
		m.logLines = append(m.logLines, msg.line)
		atBottom := m.viewport.AtBottom()
		m.syncViewportContent()
		if atBottom {
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.KeyMsg:
		// Global quit on ctrl+c
		if msg.Type == tea.KeyCtrlC {
			// Drain any pending requests with an error
			if m.replyCh != nil {
				m.replyCh <- permissionResult{err: fmt.Errorf("TUI terminated")}
				m.replyCh = nil
			}
			for _, qr := range m.queuedReqs {
				qr.replyCh <- permissionResult{err: fmt.Errorf("TUI terminated")}
			}
			m.queuedReqs = nil
			return m, tea.Quit
		}

		// Route PgUp/PgDn to viewport
		if msg.Type == tea.KeyPgUp || msg.Type == tea.KeyPgDown {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		// Delegate to active sub-model
		switch m.state {
		case statePermission:
			var cmd tea.Cmd
			m.permModel, cmd = m.permModel.Update(msg)
			return m, cmd
		case stateAskUser:
			var cmd tea.Cmd
			m.askModel, cmd = m.askModel.Update(msg)
			return m, cmd
		case stateExitPlan:
			var cmd tea.Cmd
			m.exitModel, cmd = m.exitModel.Update(msg)
			return m, cmd
		}

	case permissionRequestMsg:
		if m.state == stateIdle {
			return m.activateRequest(msg), nil
		}
		// Queue the request
		m.queuedReqs = append(m.queuedReqs, msg)
		return m, nil

	case promptCompleteMsg:
		// Send result back to gRPC handler
		if m.replyCh != nil {
			m.replyCh <- permissionResult{response: msg.response, err: msg.err}
			m.replyCh = nil
		}

		// Dequeue next request
		if len(m.queuedReqs) > 0 {
			next := m.queuedReqs[0]
			m.queuedReqs = m.queuedReqs[1:]
			return m.activateRequest(next), nil
		}

		m.state = stateIdle
		if m.vpReady {
			m.viewport.Height = m.viewportHeight()
		}
		return m, nil
	}

	return m, nil
}

func (m rootModel) activateRequest(msg permissionRequestMsg) rootModel {
	m.replyCh = msg.replyCh

	if msg.req.ToolName == "AskUserQuestion" && msg.req.ToolInputJson != "" {
		input, err := server.ParseAskUserInput(msg.req.ToolInputJson)
		if err == nil && len(input.Questions) > 0 {
			m.state = stateAskUser
			m.askModel = newAskUserModel(msg.req, input, m.width, m.height)
			if m.vpReady {
				m.viewport.Height = m.viewportHeight()
			}
			return m
		}
	}

	if msg.req.ToolName == "ExitPlanMode" && msg.req.ToolInputJson != "" {
		input, err := server.ParseExitPlanModeInput(msg.req.ToolInputJson)
		if err == nil {
			m.state = stateExitPlan
			m.exitModel = newExitPlanModel(msg.req, input, m.width, m.height)
			if m.vpReady {
				m.viewport.Height = m.viewportHeight()
			}
			return m
		}
	}

	m.state = statePermission
	m.permModel = newPermissionModel(msg.req, m.width, m.height)
	if m.vpReady {
		m.viewport.Height = m.viewportHeight()
	}
	return m
}

func (m rootModel) viewportHeight() int {
	if m.state == stateIdle {
		// Full screen minus header line and status line
		h := m.height - 2
		if h < 1 {
			h = 1
		}
		return h
	}
	h := m.height - promptAreaHeight
	if h < 3 {
		h = 3
	}
	return h
}

func (m *rootModel) syncViewportContent() {
	m.viewport.SetContent(strings.Join(m.logLines, "\n"))
}

func (m rootModel) View() string {
	var b strings.Builder

	// Log panel header
	header := logPanelHeaderStyle.Render(" Audit Log (PgUp/PgDn to scroll) ")
	b.WriteString(header)
	b.WriteString("\n")

	// Viewport
	if m.vpReady {
		b.WriteString(m.viewport.View())
	}
	b.WriteString("\n")

	// Separator
	sep := strings.Repeat("─", m.width)
	b.WriteString(logSeparatorStyle.Render(sep))
	b.WriteString("\n")

	// Bottom panel: active prompt or idle message
	switch m.state {
	case statePermission:
		b.WriteString(m.permModel.View())
	case stateAskUser:
		b.WriteString(m.askModel.View())
	case stateExitPlan:
		b.WriteString(m.exitModel.View())
	default:
		b.WriteString(statusBarStyle.Render("  Waiting for permission requests..."))
	}

	// Queue status
	if len(m.queuedReqs) > 0 {
		b.WriteString(statusBarStyle.Render(fmt.Sprintf("  %d request(s) queued", len(m.queuedReqs))))
	}

	return b.String()
}

// TUIPrompter implements the Prompter interface using a bubbletea TUI.
type TUIPrompter struct {
	program *tea.Program
}

// Prompt sends a permission request into the bubbletea event loop and blocks until the user responds.
func (t *TUIPrompter) Prompt(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	replyCh := make(chan permissionResult, 1)

	t.program.Send(permissionRequestMsg{
		req:     req,
		replyCh: replyCh,
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-replyCh:
		return result.response, result.err
	}
}

// New creates a TUIPrompter and the associated bubbletea Program.
func New() (*TUIPrompter, *tea.Program) {
	model := rootModel{}
	program := tea.NewProgram(model, tea.WithAltScreen())

	prompter := &TUIPrompter{
		program: program,
	}

	return prompter, program
}
