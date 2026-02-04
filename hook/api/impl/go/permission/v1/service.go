// Package permissionv1impl provides the server implementation for the PermissionService.
package permissionv1impl

import (
	"context"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
)

// PermissionHandler is the interface that must be implemented to handle permission requests.
type PermissionHandler interface {
	// HandlePermissionRequest processes a permission request and returns a decision.
	HandlePermissionRequest(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error)
}

// Service implements the PermissionServiceServer interface.
type Service struct {
	pb.UnimplementedPermissionServiceServer
	handler PermissionHandler
}

// NewService creates a new Service with the given handler.
func NewService(handler PermissionHandler) *Service {
	return &Service{handler: handler}
}

// RequestPermission implements the PermissionServiceServer interface.
func (s *Service) RequestPermission(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	return s.handler.HandlePermissionRequest(ctx, req)
}
