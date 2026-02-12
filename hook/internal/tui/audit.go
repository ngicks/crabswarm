package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"github.com/ngicks/crabswarm/hook/internal/server"
)

// auditEventMsg carries a pre-formatted audit log line into the bubbletea event loop.
type auditEventMsg struct {
	line string
}

// TUIAuditHandler sends formatted audit events to the bubbletea TUI and
// optionally delegates to another AuditHandler (e.g. for file/slog logging).
type TUIAuditHandler struct {
	program  *tea.Program
	delegate server.AuditHandler
}

// NewTUIAuditHandler creates a TUIAuditHandler. If delegate is nil, a NoOpAuditHandler is used.
func NewTUIAuditHandler(program *tea.Program, delegate server.AuditHandler) *TUIAuditHandler {
	if delegate == nil {
		delegate = &server.NoOpAuditHandler{}
	}
	return &TUIAuditHandler{
		program:  program,
		delegate: delegate,
	}
}

func (h *TUIAuditHandler) HandleAuditEvent(ctx context.Context, event *pb.AuditEvent) error {
	line := formatAuditEvent(event)
	h.program.Send(auditEventMsg{line: line})
	return h.delegate.HandleAuditEvent(ctx, event)
}

func (h *TUIAuditHandler) Close() error {
	return h.delegate.Close()
}

// formatAuditEvent converts an AuditEvent protobuf into a single display line.
func formatAuditEvent(event *pb.AuditEvent) string {
	ts := ""
	if t := event.GetTimestamp(); t != nil {
		ts = t.AsTime().Format("15:04:05")
	}

	req := event.GetRequest()
	if req == nil {
		return fmt.Sprintf("[%s] audit event (no request data)", ts)
	}

	return fmt.Sprintf("[%s] %-12s tool=%-10s session=%s",
		ts,
		req.GetHookEventName(),
		req.GetToolName(),
		req.GetSessionId(),
	)
}
