package claude_hookv1

import (
	"context"
	"encoding/json"
	"log/slog"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/claude_hook/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

// AuditServiceImpl implements the AuditService gRPC server.
type AuditServiceImpl struct {
	pb.UnimplementedAuditServiceServer
	Logger *slog.Logger
}

func (s *AuditServiceImpl) SendAuditEvent(ctx context.Context, req *pb.AuditRequest) (*pb.AuditResponse, error) {
	input := req.GetInput()
	// Works on opaque-api enabled proto.
	bin, err := protojson.Marshal(input)
	if err != nil {
		panic(err)
	}
	s.Logger.InfoContext(
		ctx, "audit event received",
		slog.Any("input", json.RawMessage(bin)),
	)
	return &pb.AuditResponse{}, nil
}
