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
// message as it is appended. A nil since starts live, and a zero one replays
// the room from its first message.
//
// A since past the room's newest seq starts from the newest. Only a room whose
// numbering started over hands a follower such a since: the room was deleted,
// or the database was replaced. Starting there lets the restarted room's next
// message through, where the stale since would hold back every message until
// the room grew past it again.
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
//
// A room deleted under the stream and spoken in again numbers from one again.
// The stream sends Followed once more, carrying zero, and goes on from the
// restarted room's first message.
func follow(
	ctx context.Context,
	store *Store,
	stream grpc.ServerStreamingServer[chatv1.FollowEvent],
	room string,
	since *int64,
	viewer *Sender,
) error {
	if since != nil && *since < 0 {
		return status.Errorf(codes.InvalidArgument,
			"since %d names no seq: seqs start at 1, and 0 replays the whole room", *since)
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
	if since != nil {
		f.last = min(*since, newest)
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
	// lastID is the id of the message at last when the stream sent it, empty
	// until it has sent one.
	lastID string
}

// live sends a message the feed announced, unless the stream is already past
// it, after whatever the store holds between the two.
//
// The store has every seq below one that was announced: a seq is claimed and
// its message written in one transaction, and the store runs one transaction
// at a time.
//
// A message at or below last is normally one the stream already sent. When
// the room's numbering started over instead, the stream says so with a fresh
// Followed and goes on from the restarted room's first message.
func (f *follower) live(ctx context.Context, msg *chatv1.Message) error {
	if msg.GetSeq() <= f.last {
		restarted, err := f.restarted(ctx)
		if err != nil {
			return err
		}
		if !restarted {
			return nil
		}
		if err := f.stream.Send(followedEvent(f.room, 0)); err != nil {
			return err
		}
		f.last, f.lastID = 0, ""
	}
	if err := f.catchUp(ctx, msg.GetSeq()); err != nil {
		return err
	}
	return f.send(msg)
}

// restarted reports whether the room's numbering started over after the stream
// reached last: the room was deleted, and messages said in it since took seqs
// from one again.
//
// A room newer than last is one that started over. One that has grown back to
// last or past it holds another message at last than the one the stream sent.
// A message pruned from last since it was sent is not taken for a restart: a
// room trimmed to its history cap keeps numbering on.
func (f *follower) restarted(ctx context.Context) (bool, error) {
	newest, err := f.store.newestSeq(ctx, f.room)
	if err != nil {
		return false, f.storeErr(ctx, err)
	}
	if newest < f.last {
		return true, nil
	}
	if f.lastID == "" {
		return false, nil
	}
	at, err := f.store.ReadRoom(ctx, f.room, ReadFilter{
		Cursor: CursorHead,
		Range:  1,
		Since:  f.last - 1,
		Until:  f.last + 1,
	})
	if err != nil {
		return false, f.storeErr(ctx, err)
	}
	return len(at) == 1 && at[0].Id != f.lastID, nil
}

// storeErr is the error a failed store call ends the stream with: the stream's
// own end when it ended, the store's failure otherwise.
func (f *follower) storeErr(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return storeStatus(err)
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
			return f.storeErr(ctx, err)
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
	f.last, f.lastID = msg.GetSeq(), msg.GetId()
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

// followedEvent opens a Follow stream, and opens it over again when the room's
// numbering starts over.
func followedEvent(room string, lastSeq int64) *chatv1.FollowEvent {
	return &chatv1.FollowEvent{
		Event: &chatv1.FollowEvent_Followed{
			Followed: &chatv1.Followed{Room: room, LastSeq: lastSeq},
		},
	}
}
