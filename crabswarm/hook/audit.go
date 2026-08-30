package hook

import (
	"context"
	"fmt"
	"io"
	"time"

	pb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/hook/v1"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Audit reads hook input from r and sends it as an audit event.
func Audit(ctx context.Context, r io.Reader, client pb.AuditServiceClient) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	_, err = client.ReportHookInputEvent(
		ctx,
		&pb.ReportHookInputEventRequest{
			HookInput: raw,
			Timestamp: timestamppb.New(time.Now()),
		},
		grpc.WaitForReady(true),
	)
	if err != nil {
		return fmt.Errorf("sending audit event: %w", err)
	}

	return &handler.HandlerError{}
}
