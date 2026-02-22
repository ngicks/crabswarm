package crabswarm

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/claude_hook/v1"
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/testing/protocmp"
	"gotest.tools/v3/assert"
)

type mockAuditClient struct {
	req  *pb.AuditRequest
	opts []grpc.CallOption
	err  error
}

func (m *mockAuditClient) SendAuditEvent(ctx context.Context, in *pb.AuditRequest, opts ...grpc.CallOption) (*pb.AuditResponse, error) {
	m.req = in
	m.opts = opts
	if m.err != nil {
		return nil, m.err
	}
	return &pb.AuditResponse{}, nil
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

func ptr[V any](v V) *V {
	return &v
}

var expected = &sdktypes.PreToolUseHookInput{
	SessionId:      "12345",
	TranscriptPath: "/root/.config/claude/projects/-yay/12345.jsonl",
	Cwd:            "/yay",
	PermissionMode: ptr("default"),
	HookEventName:  "PreToolUse",
	ToolName:       "Read",
	ToolInput: &sdktypes.ToolInput{
		Input: &sdktypes.ToolInput_FileRead{
			FileRead: &sdktypes.FileReadInput{
				FilePath: "/yay/buf.gen.yaml",
			},
		},
	},
	ToolUseId: "tooluse_12345",
}

func TestHookAudit_ValidInput(t *testing.T) {
	mock := &mockAuditClient{}
	err := HookAudit(context.Background(), strings.NewReader(validInput), mock)

	if _, ok := errors.AsType[*handler.HandlerError](err); !ok {
		t.Fatalf("expected HandlerError, got %T: %v", err, err)
	}

	if mock.req == nil {
		t.Fatal("SendAuditEvent was not called")
	}

	assert.DeepEqual(t, mock.req.Input, expected, protocmp.Transform())

	if mock.req.Timestamp == nil {
		t.Error("timestamp is nil")
	}
	if len(mock.opts) == 0 {
		t.Error("expected call options (WaitForReady), got none")
	}
}

func TestHookAudit_InvalidJSON(t *testing.T) {
	mock := &mockAuditClient{}
	err := HookAudit(context.Background(), strings.NewReader("not json"), mock)

	var he *handler.HandlerError
	if errors.As(err, &he) {
		t.Fatal("expected regular error, got HandlerError")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}

func TestHookAudit_ReadError(t *testing.T) {
	mock := &mockAuditClient{}
	err := HookAudit(context.Background(), errReader{}, mock)

	var he *handler.HandlerError
	if errors.As(err, &he) {
		t.Fatal("expected regular error, got HandlerError")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHookAudit_SendError(t *testing.T) {
	mock := &mockAuditClient{err: io.ErrUnexpectedEOF}
	err := HookAudit(context.Background(), strings.NewReader(validInput), mock)

	var he *handler.HandlerError
	if errors.As(err, &he) {
		t.Fatal("expected regular error, got HandlerError")
	}
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
