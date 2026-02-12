package server

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func testEvent() *pb.AuditEvent {
	return &pb.AuditEvent{
		Request: &pb.PermissionRequest{
			HookEventName: "PreToolUse",
			ToolName:      "Bash",
			SessionId:     "session-123",
			MessageId:     "msg-456",
		},
		Timestamp: timestamppb.Now(),
	}
}

func TestNoOpAuditHandler(t *testing.T) {
	h := &NoOpAuditHandler{}
	if err := h.HandleAuditEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestLogAuditHandler_Text(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	h := NewLogAuditHandler(logger)

	if err := h.HandleAuditEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{"PreToolUse", "Bash", "session-123", "msg-456"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q: %s", want, output)
		}
	}

	if err := h.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestLogAuditHandler_JSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	h := NewLogAuditHandler(logger)

	if err := h.HandleAuditEvent(context.Background(), testEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	// Check expected slog fields
	for _, key := range []string{"msg", "event", "tool", "session", "message_id"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("JSON output missing %q field", key)
		}
	}

	if err := h.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
}

func TestLogAuditHandler_CloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	h := NewLogAuditHandler(logger)

	// Multiple closes should not panic or error
	if err := h.Close(); err != nil {
		t.Fatalf("first close error: %v", err)
	}
	if err := h.Close(); err != nil {
		t.Fatalf("second close error: %v", err)
	}
}
