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

// defaultHistoryWindow is how much of the conversation a caller that asked for
// no particular amount gets: a screenful of recent context rather than the
// whole retained room, which is what a member catching up is after.
const defaultHistoryWindow = 50

// History hands back the tail of the caller's room's conversation without
// consuming anything, so the same window can be read again.
func (s *Service) History(
	ctx context.Context,
	req *chatv1.HistoryRequest,
) (*chatv1.HistoryResponse, error) {
	caller, err := s.caller(ctx)
	if err != nil {
		return nil, err
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultHistoryWindow
	}
	// Asking for more than the room keeps is not an error: the answer to it is
	// everything there is, which is what the retention cap left. A store that
	// records nothing has no cap to clamp against — it answers with whatever it
	// held before it was switched off, and that is already a short list.
	if kept := s.store.historyLimit; kept > 0 && limit > kept {
		limit = kept
	}
	entries, err := s.store.History(ctx, caller.Room, limit)
	if err != nil {
		return nil, storeStatus(err)
	}
	out := make([]*chatv1.HistoryEntry, len(entries))
	for i, e := range entries {
		entry := &chatv1.HistoryEntry{
			From:   senderProto(e.From),
			Text:   e.Text,
			SentAt: timestamppb.New(e.SentAt),
		}
		if e.To != nil {
			entry.To = senderProto(*e.To)
		}
		out[i] = entry
	}
	return &chatv1.HistoryResponse{Entries: out}, nil
}
