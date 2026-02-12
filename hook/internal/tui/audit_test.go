package tui

import (
	"strings"
	"testing"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestFormatAuditEvent(t *testing.T) {
	ts := timestamppb.Now()
	event := &pb.AuditEvent{
		Request: &pb.PermissionRequest{
			HookEventName: "PreToolUse",
			ToolName:      "Bash",
			SessionId:     "sess-123",
		},
		Timestamp: ts,
	}

	line := formatAuditEvent(event)

	if !strings.Contains(line, "PreToolUse") {
		t.Errorf("expected line to contain event name, got: %s", line)
	}
	if !strings.Contains(line, "Bash") {
		t.Errorf("expected line to contain tool name, got: %s", line)
	}
	if !strings.Contains(line, "sess-123") {
		t.Errorf("expected line to contain session id, got: %s", line)
	}
}

func TestFormatAuditEvent_NilRequest(t *testing.T) {
	ts := timestamppb.Now()
	event := &pb.AuditEvent{
		Timestamp: ts,
	}

	line := formatAuditEvent(event)

	if !strings.Contains(line, "no request data") {
		t.Errorf("expected fallback text for nil request, got: %s", line)
	}
}
