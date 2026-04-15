package crabswarm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/crabhook/v1"
	sdktypesv1 "github.com/ngicks/crabswarm/pkg/api/types/sdktypes/v1"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HookAudit reads a PreToolUseHookInput from r and sends it as an audit event.
func HookAudit(ctx context.Context, r io.Reader, client pb.AuditServiceClient) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input sdktypesv1.PreToolUseHookInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("parsing PreToolUseHookInput: %w", err)
	}

	protoInput, err := sdktypesv1.HookInputToProto(&input)
	if err != nil {
		return fmt.Errorf("converting to proto: %w", err)
	}
	protoBytes, err := proto.Marshal(protoInput)
	if err != nil {
		return fmt.Errorf("marshaling hook input proto: %w", err)
	}

	_, err = client.ReportHookInputEvent(
		ctx,
		&pb.ReportHookInputEventRequest{
			HookInput: protoBytes,
			Timestamp: timestamppb.New(time.Now()),
		},
		grpc.WaitForReady(true),
	)
	if err != nil {
		return fmt.Errorf("sending audit event: %w", err)
	}

	return &handler.HandlerError{}
}
