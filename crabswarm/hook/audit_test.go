package hook

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	pb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/hook/v1"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"google.golang.org/grpc"
	"gotest.tools/v3/assert"
)

type mockAuditClient struct {
	req  *pb.ReportHookInputEventRequest
	opts []grpc.CallOption
	err  error
}

func (m *mockAuditClient) ReportHookInputEvent(
	ctx context.Context,
	in *pb.ReportHookInputEventRequest,
	opts ...grpc.CallOption,
) (*pb.ReportHookInputEventResponse, error) {
	m.req = in
	m.opts = opts
	if m.err != nil {
		return nil, m.err
	}
	return &pb.ReportHookInputEventResponse{}, nil
}

const validInput = `{
  "session_id": "12345",
  "transcript_path": "/root/.config/claude/projects/-yay/12345.jsonl",
  "cwd": "/yay",
  "permission_mode": "default",
  "hook_event_name": "PreToolUse",
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/yay/buf.gen.yaml"
  },
  "tool_use_id": "tooluse_12345"
}`

func TestAudit_ValidInput(t *testing.T) {
	mock := &mockAuditClient{}
	err := Audit(context.Background(), strings.NewReader(validInput), mock)

	if _, ok := errors.AsType[*handler.HandlerError](err); !ok {
		t.Fatalf("expected HandlerError, got %T: %v", err, err)
	}

	if mock.req == nil {
		t.Fatal("ReportHookInputEvent was not called")
	}

	assert.Assert(t, bytes.Equal(mock.req.HookInput, []byte(validInput)))

	if mock.req.Timestamp == nil {
		t.Error("timestamp is nil")
	}
	if len(mock.opts) == 0 {
		t.Error("expected call options (WaitForReady), got none")
	}
}

func TestAudit_InvalidJSON(t *testing.T) {
	mock := &mockAuditClient{}
	err := Audit(context.Background(), strings.NewReader("not json"), mock)
	if _, ok := errors.AsType[*handler.HandlerError](err); !ok {
		t.Fatalf("expected HandlerError, got %T: %v", err, err)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestAudit_ReadError(t *testing.T) {
	mock := &mockAuditClient{}
	err := Audit(context.Background(), errReader{}, mock)

	var he *handler.HandlerError
	if errors.As(err, &he) {
		t.Fatal("expected regular error, got HandlerError")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAudit_SendError(t *testing.T) {
	mock := &mockAuditClient{err: io.ErrUnexpectedEOF}
	err := Audit(context.Background(), strings.NewReader(validInput), mock)

	var he *handler.HandlerError
	if errors.As(err, &he) {
		t.Fatal("expected regular error, got HandlerError")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
