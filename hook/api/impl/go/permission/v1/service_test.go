package permissionv1impl

import (
	"context"
	"net"
	"sync"
	"testing"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// stubPermissionHandler satisfies PermissionHandler for tests.
type stubPermissionHandler struct{}

func (h *stubPermissionHandler) HandlePermissionRequest(_ context.Context, _ *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	return &pb.PermissionResponse{ShouldContinue: true}, nil
}

// recordingAuditHandler records all audit events it receives.
type recordingAuditHandler struct {
	mu     sync.Mutex
	events []*pb.AuditEvent
}

func (h *recordingAuditHandler) HandleAuditEvent(_ context.Context, event *pb.AuditEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
	return nil
}

func (h *recordingAuditHandler) Events() []*pb.AuditEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]*pb.AuditEvent(nil), h.events...)
}

func TestServiceAudit(t *testing.T) {
	auditHandler := &recordingAuditHandler{}
	svc := NewService(&stubPermissionHandler{}, auditHandler)

	// Start a real gRPC server
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterPermissionServiceServer(grpcServer, svc)
	go grpcServer.Serve(lis)
	defer grpcServer.Stop()

	// Connect client
	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPermissionServiceClient(conn)

	// Send 2 audit events
	stream, err := client.Audit(context.Background())
	if err != nil {
		t.Fatalf("failed to open audit stream: %v", err)
	}

	for i := range 2 {
		event := &pb.AuditEvent{
			Request: &pb.PermissionRequest{
				HookEventName: "PreToolUse",
				ToolName:      "Bash",
				SessionId:     "session-" + string(rune('A'+i)),
			},
			Timestamp: timestamppb.Now(),
		}
		if err := stream.Send(event); err != nil {
			t.Fatalf("failed to send event %d: %v", i, err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv error: %v", err)
	}

	if resp.EventsReceived != 2 {
		t.Errorf("expected 2 events_received, got %d", resp.EventsReceived)
	}
	if !resp.Success {
		t.Errorf("expected success=true, got false: %s", resp.Message)
	}

	events := auditHandler.Events()
	if len(events) != 2 {
		t.Fatalf("expected 2 recorded events, got %d", len(events))
	}
	if events[0].GetRequest().GetToolName() != "Bash" {
		t.Errorf("expected tool_name=Bash, got %s", events[0].GetRequest().GetToolName())
	}
}
