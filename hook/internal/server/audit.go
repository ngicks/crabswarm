package server

import (
	"context"
	"log/slog"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
)

// AuditHandler processes audit events from hook invocations.
type AuditHandler interface {
	HandleAuditEvent(ctx context.Context, event *pb.AuditEvent) error
	Close() error
}

// NoOpAuditHandler discards all audit events.
type NoOpAuditHandler struct{}

func (h *NoOpAuditHandler) HandleAuditEvent(_ context.Context, _ *pb.AuditEvent) error {
	return nil
}

func (h *NoOpAuditHandler) Close() error {
	return nil
}

// LogAuditHandler writes audit events as structured log records via slog.
type LogAuditHandler struct {
	logger *slog.Logger
}

// NewLogAuditHandler creates a LogAuditHandler that logs via the given logger.
func NewLogAuditHandler(logger *slog.Logger) *LogAuditHandler {
	return &LogAuditHandler{
		logger: logger,
	}
}

func (h *LogAuditHandler) HandleAuditEvent(ctx context.Context, event *pb.AuditEvent) error {
	attrs := []slog.Attr{
		slog.String("event_timestamp", event.GetTimestamp().AsTime().String()),
	}

	if req := event.GetRequest(); req != nil {
		attrs = append(attrs,
			slog.String("event", req.GetHookEventName()),
			slog.String("tool", req.GetToolName()),
			slog.String("session", req.GetSessionId()),
			slog.String("message_id", req.GetMessageId()),
		)
	}

	h.logger.LogAttrs(ctx, slog.LevelInfo, "audit_event", attrs...)
	return nil
}

// Close is a no-op. The caller manages the lifecycle of the underlying writer.
func (h *LogAuditHandler) Close() error {
	return nil
}
