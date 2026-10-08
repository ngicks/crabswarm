package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

// FollowStream is one open Follow stream: the Followed event first, then the
// room's messages. A [Client] stream reports a failed receive the way every
// other call of this package does, through [callError].
type FollowStream interface {
	Recv() (*chatv1.FollowEvent, error)
}

// FollowOpener opens a Follow stream resuming after since, where zero starts
// live. [FollowInto] calls it again for every reconnect.
type FollowOpener func(ctx context.Context, since int64) (FollowStream, error)

// Follow opens a Follow stream of the room token stands for. Following is not
// attendance and moves no read position, so a token nobody attends under
// follows as well as one that does.
//
// The stream is lazy: a refusal surfaces on the first Recv rather than here.
func (c *Client) Follow(ctx context.Context, token string, since int64) (FollowStream, error) {
	stream, err := c.chat.Follow(withToken(ctx, token), &chatv1.FollowRequest{Since: since})
	if err != nil {
		return nil, callError(err)
	}
	return followStream{stream}, nil
}

// Follow opens a Follow stream of room as the host operator, who attends no
// room and so is mentioned by nothing.
//
// It takes its own challenge, as every admin call does; the daemon spends the
// nonce once, when the stream opens.
func (a *AdminClient) Follow(ctx context.Context, room string, since int64) (FollowStream, error) {
	nonce, err := a.client.nonce(ctx, a.identity)
	if err != nil {
		return nil, err
	}
	stream, err := a.client.admin.Follow(auth.ContextWithBearer(ctx, nonce),
		&chatv1.AdminFollowRequest{Room: room, Since: since})
	if err != nil {
		return nil, callError(err)
	}
	return followStream{stream}, nil
}

// followStream maps a stream's failures onto the errors the CLI reports. io.EOF
// is left as it is: callError passes on anything that is not a gRPC status.
type followStream struct {
	stream grpc.ServerStreamingClient[chatv1.FollowEvent]
}

func (s followStream) Recv() (*chatv1.FollowEvent, error) {
	ev, err := s.stream.Recv()
	if err != nil {
		return nil, callError(err)
	}
	return ev, nil
}

const (
	// followRetryMin is the first pause before following again after the stream
	// was lost; each failed attempt doubles it up to followRetryMax.
	followRetryMin = time.Second
	followRetryMax = 30 * time.Second
)

// The lines FollowInto writes are told apart by their "type".
const (
	followLineStatus  = "status"
	followLineMessage = "message"
)

// statusReconnecting is the status line written once when the stream is lost.
const statusReconnecting = "reconnecting to the daemon"

// FollowInto prints the stream open opens as JSON Lines to w, and keeps it open
// until ctx is done.
//
// Each line is one JSON object ending in "\n", written by a single Write and
// followed by a Flush when w has one, so a reader sees every line as soon as
// it is complete. A Followed event writes
//
//	{"type":"status","message":"following <room>"}
//
// and a message writes the message's protojson, field names as in the schema,
// with "type" set to "message", "message" to "<sender>: <text>", and "inject"
// and "mentioned_you" both to whether the message is for the follower. "seq" is
// a JSON number rather than the string the proto JSON mapping makes of an
// int64: a reader compares it and hands it back to `chat read --skip`.
//
// A stream lost to Unavailable, ResourceExhausted or an end the daemon gave
// while ctx is alive is followed again after a pause that starts at a second
// and doubles up to thirty, reset by the next Followed. The first loss of an
// outage writes {"type":"status","message":"reconnecting to the daemon"} and
// logs its cause; the retries after it write nothing. Following again resumes
// after the last message written, or after the seq the room stood at when the
// stream first opened, so no message is written twice or skipped. The one
// exception is a room that was empty when the first stream opened and whose
// stream was lost before it carried a message: there is no seq to resume after,
// since zero means live, so what was sent while the stream was lost is not
// written.
//
// Any other failure is returned. ctx ending returns nil: being stopped is how
// following ends.
//
// A nil logger logs nothing.
func FollowInto(ctx context.Context, logger *slog.Logger, w io.Writer, open FollowOpener) error {
	return followInto(ctx, logger, w, open, sleepCtx)
}

// sleepCtx waits d, or until ctx is done, which it reports.
func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// followInto is [FollowInto] with the pause between attempts taken as a
// function, so a test drives the backoff without waiting it out.
func followInto(
	ctx context.Context,
	logger *slog.Logger,
	w io.Writer,
	open FollowOpener,
	sleep func(ctx context.Context, d time.Duration) error,
) error {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	f := &following{w: w, delay: followRetryMin}
	for {
		err := f.session(ctx, open)
		if ctx.Err() != nil {
			return nil
		}
		if !followRetryable(err) {
			return err
		}
		if !f.reconnecting {
			f.reconnecting = true
			logger.WarnContext(ctx, "lost the chat follow stream",
				slog.Int64("resume_after", f.since), slog.Any("err", err))
			if err := writeFollowStatus(w, statusReconnecting); err != nil {
				return err
			}
		}
		if err := sleep(ctx, f.delay); err != nil {
			return nil
		}
		f.delay = min(f.delay*2, followRetryMax)
	}
}

// followRetryable reports whether a lost stream is worth opening again: the
// daemon went away or was restarting, or dropped a follower that fell behind.
// Anything else — a token nobody vouches for, a refused credential, a request
// the daemon will refuse again — would fail the same way every time.
func followRetryable(err error) bool {
	if errors.Is(err, io.EOF) {
		return true
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

// following is how far FollowInto has got across every stream it opened.
type following struct {
	w io.Writer
	// since is what the next stream resumes after: the seq of the last message
	// written, else the seq the room stood at when the first stream opened, and
	// zero, which starts live, until a stream has opened at all.
	since int64
	// delay is the pause before the next attempt.
	delay time.Duration
	// reconnecting is set from the first loss of an outage to the next
	// Followed, so an outage is announced once however many attempts it takes.
	reconnecting bool
}

// session opens one stream and writes what it carries until it fails.
func (f *following) session(ctx context.Context, open FollowOpener) error {
	stream, err := open(ctx, f.since)
	if err != nil {
		return err
	}
	for {
		ev, err := stream.Recv()
		if err != nil {
			return err
		}
		switch e := ev.GetEvent().(type) {
		case *chatv1.FollowEvent_Followed:
			// A stream that resumed reports the room's newest seq, but the
			// messages up to it are still to come, so only the first stream
			// takes its starting point from here.
			if f.since == 0 {
				f.since = e.Followed.GetLastSeq()
			}
			f.delay = followRetryMin
			f.reconnecting = false
			if err := writeFollowStatus(f.w, "following "+e.Followed.GetRoom()); err != nil {
				return err
			}
		case *chatv1.FollowEvent_Message:
			msg := e.Message
			// The daemon sends each message once per stream and resumes after
			// since, so this only drops what a misbehaving stream repeats.
			if msg.GetSeq() <= f.since {
				continue
			}
			if err := writeFollowMessage(f.w, msg, ev.GetMentionedYou()); err != nil {
				return err
			}
			f.since = msg.GetSeq()
		}
	}
}

// writeFollowStatus writes one status line.
func writeFollowStatus(w io.Writer, message string) error {
	return writeFollowLine(w, map[string]any{
		"type":    followLineStatus,
		"message": message,
	})
}

// writeFollowMessage writes one message line.
//
// The message's own mentioned_you is about a read, which following is not; the
// event's is the one for this follower, and it replaces the message's.
func writeFollowMessage(w io.Writer, msg *chatv1.Message, mentioned bool) error {
	b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(msg)
	if err != nil {
		return fmt.Errorf("encoding message %d: %w", msg.GetSeq(), err)
	}
	// Decoded rather than written out as it is: protojson varies its whitespace
	// on purpose, and the fields below are laid over it. Raw values keep the
	// rest exactly as protojson spelled it.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("decoding message %d: %w", msg.GetSeq(), err)
	}
	fields := make(map[string]any, len(raw)+3)
	for k, v := range raw {
		fields[k] = v
	}
	fields["type"] = followLineMessage
	fields["message"] = Address(msg.GetFrom()) + ": " + msg.GetText()
	fields["inject"] = mentioned
	fields["mentioned_you"] = mentioned
	fields["seq"] = json.Number(strconv.FormatInt(msg.GetSeq(), 10))
	return writeFollowLine(w, fields)
}

// writeFollowLine encodes line as one JSON line and hands it to w in one Write,
// then flushes w if it buffers.
//
// HTML escaping is off: the text is chat, where "<" and "&" are common, and the
// reader is a program rather than a page.
func writeFollowLine(w io.Writer, line map[string]any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(line); err != nil {
		return err
	}
	if _, err := w.Write(b.Bytes()); err != nil {
		return err
	}
	if f, ok := w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}
