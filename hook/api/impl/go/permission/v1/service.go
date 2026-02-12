// Package permissionv1impl provides the server implementation for the PermissionService.
package permissionv1impl

import (
	"context"
	"io"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"google.golang.org/grpc"
)

// PermissionHandler is the interface that must be implemented to handle permission requests.
type PermissionHandler interface {
	// HandlePermissionRequest processes a permission request and returns a decision.
	HandlePermissionRequest(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error)
}

// AuditHandler processes audit events received from hook clients.
type AuditHandler interface {
	HandleAuditEvent(ctx context.Context, event *pb.AuditEvent) error
}

// Service implements the PermissionServiceServer interface.
type Service struct {
	pb.UnimplementedPermissionServiceServer
	handler      PermissionHandler
	auditHandler AuditHandler
}

// NewService creates a new Service with the given handlers.
func NewService(handler PermissionHandler, auditHandler AuditHandler) *Service {
	return &Service{handler: handler, auditHandler: auditHandler}
}

// RequestPermission implements the PermissionServiceServer interface.
func (s *Service) RequestPermission(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	return s.handler.HandlePermissionRequest(ctx, req)
}

// Audit implements the PermissionServiceServer client-streaming Audit RPC.
func (s *Service) Audit(stream grpc.ClientStreamingServer[pb.AuditEvent, pb.AuditResponse]) error {
	var count int32
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.AuditResponse{
				EventsReceived: count,
				Success:        true,
				Message:        "audit events processed",
			})
		}
		if err != nil {
			return err
		}

		if err := s.auditHandler.HandleAuditEvent(stream.Context(), event); err != nil {
			return stream.SendAndClose(&pb.AuditResponse{
				EventsReceived: count,
				Success:        false,
				Message:        err.Error(),
			})
		}
		count++
	}
}
