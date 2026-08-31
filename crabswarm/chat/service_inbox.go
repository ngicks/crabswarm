package chat

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Send delivers the message to the one member the address resolves to within
// the caller's room, then reports the delivery to the notifier.
func (s *Service) Send(
	ctx context.Context,
	req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	from, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetText() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty message text")
	}
	recipient, err := s.deliver.send(ctx, from, req.GetTo(), req.GetText(), time.Now())
	if err != nil {
		return nil, storeStatus(err)
	}
	return &chatv1.SendResponse{Recipient: memberProto(recipient)}, nil
}

// Broadcast delivers the message to every other member of the caller's room.
//
// The sender is left out: an agent announcing something already knows it, and
// the echo would only cost it a nudge and a read.
func (s *Service) Broadcast(
	ctx context.Context,
	req *chatv1.BroadcastRequest,
) (*chatv1.BroadcastResponse, error) {
	from, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetText() == "" {
		return nil, status.Error(codes.InvalidArgument, "empty message text")
	}
	recipients, err := s.deliver.broadcast(ctx, from, req.GetText(), time.Now(), true)
	if err != nil {
		return nil, storeStatus(err)
	}
	return &chatv1.BroadcastResponse{DeliveredCount: int32(len(recipients))}, nil
}

// Read hands back the caller's pending messages and drains the inbox.
func (s *Service) Read(
	ctx context.Context,
	_ *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	messages, err := s.store.Read(ctx, caller.Token)
	if err != nil {
		return nil, storeStatus(err)
	}
	out := make([]*chatv1.Message, len(messages))
	for i, m := range messages {
		out[i] = &chatv1.Message{
			From:   senderProto(m.From),
			Text:   m.Text,
			SentAt: timestamppb.New(m.SentAt),
		}
	}
	return &chatv1.ReadResponse{Messages: out}, nil
}
