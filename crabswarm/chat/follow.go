package chat

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// followPage is how many stored messages one read of a catch-up takes. A
// follower resuming far back is fed in pages rather than handed the whole
// retained conversation at once, which a room without a history cap makes
// unbounded.
const followPage = 100

// follow serves one Follow stream of room: Followed first, carrying the room's
// newest seq; then the stored messages past since when since is set; then each
// message as it is appended. A zero since starts live.
//
// viewer is the role mentioned_you is measured for. Nil follows as nobody,
// which is how the operator follows, and marks nothing.
//
// Every message goes out once and in seq order, which is what lets a client
// that lost its stream follow again from the last seq it saw. The room's feed
// alone cannot promise that: a message is announced after the store lets go of
// it, so two sends racing each other can be announced in the opposite order,
// and one committed just before the stream subscribed is announced just after.
// The stream therefore keeps the newest seq it sent, drops whatever is not past
// it, and reads a gap out of the store before the message that revealed it.
func follow(
	ctx context.Context,
	store *Store,
	stream grpc.ServerStreamingServer[chatv1.FollowEvent],
	room string,
	since int64,
	viewer *Sender,
) error {
	if since < 0 {
		return status.Errorf(codes.InvalidArgument,
			"since %d names no seq: seqs start at 1, and 0 starts live", since)
	}

	// Subscribed before anything is read: a message appended between the read
	// and the subscription would reach this stream never. One appended before
	// the read and announced after it comes down the feed as well, and is
	// dropped there as already sent.
	sub := store.events.subscribe(room)
	defer store.events.unsubscribe(sub)

	newest, err := store.newestSeq(ctx, room)
	if err != nil {
		return storeStatus(err)
	}
	if err := stream.Send(followedEvent(room, newest)); err != nil {
		return err
	}

	f := follower{store: store, stream: stream, room: room, viewer: viewer, last: newest}
	if since > 0 {
		f.last = since
		if err := f.catchUp(ctx, newest+1); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-sub.events:
			if !ok {
				return status.Errorf(
					codes.ResourceExhausted,
					"follower fell more than %d events behind; follow again since the last seq it saw",
					roomEventBuffer,
				)
			}
			// The feed is the room's, so it also carries who came, went and
			// changed state. Following is the conversation alone.
			msg := ev.GetMessageAppended().GetMessage()
			if msg == nil {
				continue
			}
			if err := f.live(ctx, msg); err != nil {
				return err
			}
		}
	}
}

// follower is how far one Follow stream has got.
type follower struct {
	store  *Store
	stream grpc.ServerStreamingServer[chatv1.FollowEvent]
	room   string
	viewer *Sender
	// last is the newest seq the stream has sent, or the one it started after.
	last int64
}

// live sends a message the feed announced, unless the stream is already past
// it, after whatever the store holds between the two.
//
// The store has every seq below one that was announced: a seq is claimed and
// its message written in one transaction, and the store runs one transaction
// at a time.
func (f *follower) live(ctx context.Context, msg *chatv1.Message) error {
	if msg.GetSeq() <= f.last {
		return nil
	}
	if err := f.catchUp(ctx, msg.GetSeq()); err != nil {
		return err
	}
	return f.send(msg)
}

// catchUp sends the stored messages past f.last and before until, oldest
// first. A message pruned in the meantime is simply not there to send.
func (f *follower) catchUp(ctx context.Context, until int64) error {
	for f.last+1 < until {
		page, err := f.store.ReadRoom(ctx, f.room, ReadFilter{
			Cursor: CursorHead,
			Range:  followPage,
			Since:  f.last,
			Until:  until,
		})
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return storeStatus(err)
		}
		for _, m := range page {
			if err := f.send(messageProto(m)); err != nil {
				return err
			}
		}
		if len(page) < followPage {
			return nil
		}
	}
	return nil
}

// send writes msg down the stream and moves the stream past it.
//
// msg may be the very message the feed handed every watcher of the room, so it
// is wrapped rather than touched: whether it mentions this follower is a fact
// about this follower.
func (f *follower) send(msg *chatv1.Message) error {
	err := f.stream.Send(&chatv1.FollowEvent{
		Event:        &chatv1.FollowEvent_Message{Message: msg},
		MentionedYou: f.mentioned(msg),
	})
	if err != nil {
		return err
	}
	f.last = msg.GetSeq()
	return nil
}

// mentioned reports whether msg is for the follower, ignoring the read
// position, which following does not move.
func (f *follower) mentioned(msg *chatv1.Message) bool {
	if f.viewer == nil {
		return false
	}
	from := Sender{Name: msg.GetFrom().GetName(), Team: msg.GetFrom().GetTeam()}
	return addressedTo(from, targetOf(msg.GetTarget()), *f.viewer)
}

// followedEvent is the first event of a Follow stream.
func followedEvent(room string, lastSeq int64) *chatv1.FollowEvent {
	return &chatv1.FollowEvent{
		Event: &chatv1.FollowEvent_Followed{
			Followed: &chatv1.Followed{Room: room, LastSeq: lastSeq},
		},
	}
}
