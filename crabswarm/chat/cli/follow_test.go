package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
)

func followedEv(room string, lastSeq int64) *chatv1.FollowEvent {
	return &chatv1.FollowEvent{Event: &chatv1.FollowEvent_Followed{
		Followed: &chatv1.Followed{Room: room, LastSeq: lastSeq},
	}}
}

func messageEv(seq int64, text string, mentioned bool) *chatv1.FollowEvent {
	return &chatv1.FollowEvent{
		Event: &chatv1.FollowEvent_Message{Message: &chatv1.Message{
			Id:   "id-" + text,
			Seq:  seq,
			From: member("alpha", "bob", "/work/proj"),
			Text: text,
		}},
		MentionedYou: mentioned,
	}
}

// decodeLines decodes every JSON line of out, keeping numbers as written.
func decodeLines(t *testing.T, out string) []map[string]any {
	t.Helper()
	assert.Assert(t, strings.HasSuffix(out, "\n"), "output %q does not end in a newline", out)
	var decoded []map[string]any
	for line := range strings.SplitSeq(strings.TrimSuffix(out, "\n"), "\n") {
		dec := json.NewDecoder(strings.NewReader(line))
		dec.UseNumber()
		var m map[string]any
		assert.NilError(t, dec.Decode(&m), "line %q", line)
		decoded = append(decoded, m)
	}
	return decoded
}

// summary reduces decoded lines to what the loop tests compare: the message of
// a status line, the seq of a message line.
func summary(lines []map[string]any) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		switch l["type"] {
		case "status":
			out[i] = "status: " + l["message"].(string)
		case "message":
			out[i] = "message " + l["seq"].(json.Number).String()
		default:
			out[i] = "unknown line"
		}
	}
	return out
}

func TestWriteFollowMessage(t *testing.T) {
	msg := &chatv1.Message{
		Id:   "0199c1a2-0000-7000-8000-000000000001",
		Seq:  131,
		From: member("team", "alice", "/work/proj"),
		Target: &chatv1.Target{Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{
			Roles: []*chatv1.MemberTarget{{Team: "team", Name: "bob"}},
		}}},
		Text:   "please rebase onto <main> & push",
		SentAt: timestamppb.New(time.Date(2026, 10, 8, 1, 23, 45, 0, time.UTC)),
		// A read's mention, which the event's replaces.
		MentionedYou: true,
	}

	for _, mentioned := range []bool{true, false} {
		var out bytes.Buffer
		assert.NilError(t, writeFollowMessage(&out, msg, mentioned))

		assert.Equal(t, strings.Count(out.String(), "\n"), 1)
		// Chat text keeps its "<" and "&" as written.
		assert.Assert(t, strings.Contains(out.String(), "<main> & push"), out.String())

		got := decodeLines(t, out.String())[0]
		assert.DeepEqual(t, got, map[string]any{
			"type":    "message",
			"message": "team/alice: please rebase onto <main> & push",
			"inject":  mentioned,
			"id":      "0199c1a2-0000-7000-8000-000000000001",
			"seq":     json.Number("131"),
			"from":    map[string]any{"name": "alice", "team": "team", "room": "/work/proj"},
			"target": map[string]any{
				"roles": map[string]any{
					"roles": []any{map[string]any{"team": "team", "name": "bob"}},
				},
			},
			"text":          "please rebase onto <main> & push",
			"sent_at":       "2026-10-08T01:23:45Z",
			"mentioned_you": mentioned,
		})
	}
}

// The operator speaks in no team, and a board post has no target at all.
func TestWriteFollowMessage_OperatorPost(t *testing.T) {
	var out bytes.Buffer
	assert.NilError(t, writeFollowMessage(&out, &chatv1.Message{
		Id: "id-1", Seq: 2, From: &chatv1.Member{Name: "admin"}, Text: "hi",
	}, false))

	got := decodeLines(t, out.String())[0]
	assert.Equal(t, got["message"], "admin: hi")
	_, hasTarget := got["target"]
	assert.Assert(t, !hasTarget, "line %q names a target", out.String())
}

func TestWriteFollowStatus(t *testing.T) {
	var out bytes.Buffer
	assert.NilError(t, writeFollowStatus(&out, "following /work/a&b"))
	assert.Equal(t, out.String(), `{"message":"following /work/a&b","type":"status"}`+"\n")
}

// fakeFollowStream hands out events, then err.
type fakeFollowStream struct {
	events []*chatv1.FollowEvent
	err    error
	// onEnd runs as the events run out, before err is returned.
	onEnd func()
}

func (s *fakeFollowStream) Recv() (*chatv1.FollowEvent, error) {
	if len(s.events) > 0 {
		ev := s.events[0]
		s.events = s.events[1:]
		return ev, nil
	}
	if s.onEnd != nil {
		s.onEnd()
	}
	return nil, s.err
}

// fakeOpen is one answer of the opener: a stream, or an error opening one.
type fakeOpen struct {
	stream *fakeFollowStream
	err    error
}

// fakeOpener answers each open with the next of opens and records the since it
// was asked for.
type fakeOpener struct {
	opens  []fakeOpen
	sinces []int64
}

func (f *fakeOpener) open(_ context.Context, since int64) (FollowStream, error) {
	f.sinces = append(f.sinces, since)
	if len(f.opens) == 0 {
		return nil, status.Error(codes.Internal, "the test opened more streams than it scripted")
	}
	next := f.opens[0]
	f.opens = f.opens[1:]
	if next.err != nil {
		return nil, next.err
	}
	return next.stream, nil
}

// fakeSleep records every pause without taking it.
type fakeSleep struct {
	pauses []time.Duration
}

func (f *fakeSleep) sleep(ctx context.Context, d time.Duration) error {
	f.pauses = append(f.pauses, d)
	return ctx.Err()
}

var errUnavailable = status.Error(codes.Unavailable, "connection refused")

// A lost stream is followed again after the last message written: nothing is
// written twice and nothing between the two streams is skipped.
func TestFollowInto_ResumesAfterTheLastMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	opener := &fakeOpener{opens: []fakeOpen{
		{stream: &fakeFollowStream{
			events: []*chatv1.FollowEvent{
				followedEv(
					"/work/proj",
					5,
				),
				messageEv(6, "six", true),
				messageEv(7, "seven", false),
			},
			err: errUnavailable,
		}},
		{stream: &fakeFollowStream{
			events: []*chatv1.FollowEvent{
				// The resumed stream reports the room's newest seq before the
				// messages up to it.
				followedEv("/work/proj", 9),
				messageEv(8, "eight", false),
				// A stream repeating what was written is not written again.
				messageEv(8, "eight", false),
				messageEv(9, "nine", true),
				messageEv(10, "ten", true),
			},
			err:   status.Error(codes.Canceled, "context canceled"),
			onEnd: cancel,
		}},
	}}
	sleep := &fakeSleep{}

	var out bytes.Buffer
	assert.NilError(t, followInto(ctx, nil, &out, opener.open, sleep.sleep))

	assert.DeepEqual(t, opener.sinces, []int64{0, 7})
	assert.DeepEqual(t, sleep.pauses, []time.Duration{time.Second})
	assert.DeepEqual(t, summary(decodeLines(t, out.String())), []string{
		"status: following /work/proj",
		"message 6",
		"message 7",
		"status: reconnecting to the daemon",
		"status: following /work/proj",
		"message 8",
		"message 9",
		"message 10",
	})
}

// With nothing written yet, a stream resumes after the seq the room stood at
// when the first stream opened. A later stream's Followed does not move it: the
// messages up to that one's newest seq had not been written when it was lost.
//
// An outage is announced once however many attempts it takes, the pause
// doubles on each, and a Followed starts the next outage over.
func TestFollowInto_ResumesAfterTheFirstFollowedAndAnnouncesEachOutageOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	opener := &fakeOpener{opens: []fakeOpen{
		{stream: &fakeFollowStream{
			events: []*chatv1.FollowEvent{followedEv("/r", 5)},
			err:    errUnavailable,
		}},
		{err: errUnavailable},
		{stream: &fakeFollowStream{
			events: []*chatv1.FollowEvent{followedEv("/r", 8)},
			// The daemon ended the stream while the follower still wanted it.
			err: io.EOF,
		}},
		{stream: &fakeFollowStream{
			events: []*chatv1.FollowEvent{followedEv("/r", 8), messageEv(6, "six", false)},
			err:    status.Error(codes.ResourceExhausted, "follower fell behind"),
		}},
		{stream: &fakeFollowStream{
			events: []*chatv1.FollowEvent{followedEv("/r", 8), messageEv(7, "seven", false)},
			onEnd:  cancel,
			err:    status.Error(codes.Canceled, "context canceled"),
		}},
	}}
	sleep := &fakeSleep{}

	var out bytes.Buffer
	assert.NilError(t, followInto(ctx, nil, &out, opener.open, sleep.sleep))

	assert.DeepEqual(t, opener.sinces, []int64{0, 5, 5, 5, 6})
	assert.DeepEqual(t, sleep.pauses,
		[]time.Duration{time.Second, 2 * time.Second, time.Second, time.Second})
	assert.DeepEqual(t, summary(decodeLines(t, out.String())), []string{
		"status: following /r",
		"status: reconnecting to the daemon",
		"status: following /r",
		"status: reconnecting to the daemon",
		"status: following /r",
		"message 6",
		"status: reconnecting to the daemon",
		"status: following /r",
		"message 7",
	})
}

// A daemon that stays away is tried less and less often, down to once every
// thirty seconds.
func TestFollowInto_BackoffIsCapped(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	const attempts = 8
	opener := &fakeOpener{}
	for range attempts {
		opener.opens = append(opener.opens, fakeOpen{err: errUnavailable})
	}
	sleep := &fakeSleep{}
	stopAfter := func(ctx context.Context, d time.Duration) error {
		if len(sleep.pauses) == attempts-1 {
			cancel()
		}
		return sleep.sleep(ctx, d)
	}

	var out bytes.Buffer
	assert.NilError(t, followInto(ctx, nil, &out, opener.open, stopAfter))

	assert.DeepEqual(t, sleep.pauses, []time.Duration{
		time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second,
		16 * time.Second, 30 * time.Second, 30 * time.Second, 30 * time.Second,
	})
	assert.DeepEqual(t, summary(decodeLines(t, out.String())),
		[]string{"status: reconnecting to the daemon"})
}

// A refusal would come back the same way every time, so it ends the follow
// rather than being retried.
func TestFollowInto_ARefusalEndsIt(t *testing.T) {
	for _, code := range []codes.Code{
		codes.Unauthenticated, codes.PermissionDenied, codes.InvalidArgument,
	} {
		t.Run(code.String(), func(t *testing.T) {
			refusal := status.Error(code, "refused")
			opener := &fakeOpener{opens: []fakeOpen{{stream: &fakeFollowStream{err: refusal}}}}
			sleep := &fakeSleep{}

			var out bytes.Buffer
			err := followInto(t.Context(), nil, &out, opener.open, sleep.sleep)
			assert.ErrorIs(t, err, refusal)
			assert.Equal(t, len(opener.sinces), 1)
			assert.Equal(t, len(sleep.pauses), 0)
			assert.Equal(t, out.String(), "")
		})
	}
}

// Being stopped is how following ends, mid-stream or mid-pause.
func TestFollowInto_StoppingIsNotAFailure(t *testing.T) {
	t.Run("while the stream is open", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		opener := &fakeOpener{opens: []fakeOpen{{stream: &fakeFollowStream{
			events: []*chatv1.FollowEvent{followedEv("/r", 0)},
			onEnd:  cancel,
			err:    status.Error(codes.Canceled, "context canceled"),
		}}}}
		var out bytes.Buffer
		assert.NilError(t, FollowInto(ctx, nil, &out, opener.open))
	})

	t.Run("while waiting to reconnect", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		opener := &fakeOpener{opens: []fakeOpen{{err: errUnavailable}}}
		var out bytes.Buffer
		done := make(chan error, 1)
		go func() { done <- FollowInto(ctx, nil, &out, opener.open) }()
		// The real pause is a second; the cancel lands inside it.
		time.Sleep(50 * time.Millisecond)
		cancel()
		select {
		case err := <-done:
			assert.NilError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("FollowInto did not return once ctx was done")
		}
	})
}

// flushRecorder records the order of writes and flushes.
type flushRecorder struct {
	mu  sync.Mutex
	ops []string
}

func (r *flushRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ops = append(r.ops, "write "+string(p))
	return len(p), nil
}

func (r *flushRecorder) Flush() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ops = append(r.ops, "flush")
	return nil
}

// Every line is one write, flushed before the next: a reader waiting on the
// pipe sees each line whole, as soon as it is complete.
func TestFollowInto_WritesAndFlushesEachLine(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	opener := &fakeOpener{opens: []fakeOpen{{stream: &fakeFollowStream{
		events: []*chatv1.FollowEvent{followedEv("/r", 0), messageEv(1, "one", true)},
		onEnd:  cancel,
		err:    status.Error(codes.Canceled, "context canceled"),
	}}}}
	rec := &flushRecorder{}

	assert.NilError(t, followInto(ctx, nil, rec, opener.open, (&fakeSleep{}).sleep))

	assert.Equal(t, len(rec.ops), 4)
	for i := 0; i < len(rec.ops); i += 2 {
		line, ok := strings.CutPrefix(rec.ops[i], "write ")
		assert.Assert(t, ok, "op %d = %q, want a write", i, rec.ops[i])
		assert.Equal(t, strings.Count(line, "\n"), 1)
		assert.Assert(t, strings.HasSuffix(line, "\n"))
		assert.Equal(t, rec.ops[i+1], "flush")
	}
}

// A line that cannot be written ends the follow: nobody is reading.
func TestFollowInto_AFailedWriteEndsIt(t *testing.T) {
	opener := &fakeOpener{opens: []fakeOpen{{stream: &fakeFollowStream{
		events: []*chatv1.FollowEvent{followedEv("/r", 0)},
		err:    errUnavailable,
	}}}}
	pr, pw := io.Pipe()
	_ = pr.Close()

	err := followInto(t.Context(), nil, pw, opener.open, (&fakeSleep{}).sleep)
	assert.ErrorIs(t, err, io.ErrClosedPipe)
}

// fakeFollowService is a member plane whose Follow answers each call with the
// next of answers: events to send, then the error to end the stream with.
type fakeFollowService struct {
	chatv1.UnimplementedChatServiceServer

	mu      sync.Mutex
	answers []fakeFollowAnswer
	sinces  []int64
}

type fakeFollowAnswer struct {
	events []*chatv1.FollowEvent
	err    error
}

func (f *fakeFollowService) Follow(
	req *chatv1.FollowRequest,
	stream grpc.ServerStreamingServer[chatv1.FollowEvent],
) error {
	f.mu.Lock()
	f.sinces = append(f.sinces, req.GetSince())
	var answer fakeFollowAnswer
	if len(f.answers) > 0 {
		answer, f.answers = f.answers[0], f.answers[1:]
	}
	f.mu.Unlock()
	for _, ev := range answer.events {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	return answer.err
}

// Through the real client, a stream the daemon could not serve is followed
// again, and a refusal ends the follow in the daemon's own words. The token
// rides on every stream.
func TestClient_FollowInto(t *testing.T) {
	fake := &fakeFollowService{answers: []fakeFollowAnswer{
		{err: status.Error(codes.Unavailable, "draining")},
		{
			events: []*chatv1.FollowEvent{followedEv("/work/proj", 3), messageEv(4, "four", true)},
			err:    status.Error(codes.Unauthenticated, "no team information for this token"),
		},
	}}
	d := serveTestDaemon(t, fake, nil)
	sleep := &fakeSleep{}

	var out bytes.Buffer
	err := followInto(t.Context(), nil, &out,
		func(ctx context.Context, since int64) (FollowStream, error) {
			return d.client.Follow(ctx, "tok-a", since)
		}, sleep.sleep)
	assert.Error(t, err, "no team information for this token")
	assert.Equal(t, status.Code(err), codes.Unauthenticated)

	assert.DeepEqual(t, fake.sinces, []int64{0, 0})
	assert.DeepEqual(t, d.seenTokens(), []string{"tok-a", "tok-a"})
	assert.DeepEqual(t, sleep.pauses, []time.Duration{time.Second})
	assert.DeepEqual(t, summary(decodeLines(t, out.String())), []string{
		"status: reconnecting to the daemon",
		"status: following /work/proj",
		"message 4",
	})
}

// fakeAdminFollow is the admin plane of admin_test.go with a Follow that
// records what each stream was opened with.
type fakeAdminFollow struct {
	*fakeAdminService

	mu      sync.Mutex
	follows []*chatv1.AdminFollowRequest
	bearers []string
}

func (f *fakeAdminFollow) Follow(
	req *chatv1.AdminFollowRequest,
	stream grpc.ServerStreamingServer[chatv1.FollowEvent],
) error {
	bearer, _ := auth.BearerFromContext(stream.Context())
	f.mu.Lock()
	f.follows = append(f.follows, req)
	f.bearers = append(f.bearers, bearer)
	f.mu.Unlock()
	if err := stream.Send(followedEv(req.GetRoom(), 0)); err != nil {
		return err
	}
	return status.Error(codes.PermissionDenied, "stop here")
}

// An admin stream answers a challenge of its own, names its room, and resumes
// as a member stream does.
func TestAdminClient_Follow(t *testing.T) {
	path, recipient := newIdentityFile(t)
	fake := &fakeAdminFollow{
		fakeAdminService: &fakeAdminService{recipient: recipient, nonce: "nonce-abc123"},
	}
	d := serveTestDaemon(t, nil, fake)

	stream, err := d.client.Admin(path).Follow(t.Context(), "/work/proj", 7)
	assert.NilError(t, err)
	ev, err := stream.Recv()
	assert.NilError(t, err)
	assert.Equal(t, ev.GetFollowed().GetRoom(), "/work/proj")
	_, err = stream.Recv()
	assert.Error(t, err, "stop here")

	assert.Equal(t, fake.nonceCalls, 1)
	assert.DeepEqual(t, fake.bearers, []string{"nonce-abc123"})
	assert.Equal(t, len(fake.follows), 1)
	assert.Equal(t, fake.follows[0].GetRoom(), "/work/proj")
	assert.Equal(t, fake.follows[0].GetSince(), int64(7))
	// Admin calls carry no identity token.
	assert.DeepEqual(t, d.seenTokens(), []string{"", ""})
}

// An admin follow takes the room it is named, else the room of the working
// directory, else the working directory itself.
func TestRoomToFollow(t *testing.T) {
	t.Run("a named room passes through", func(t *testing.T) {
		lister := &fakeLister{rooms: []string{"/work/proj"}}
		wd := &fakeGetwd{dir: "/work/proj"}

		room, err := RoomToFollow(t.Context(), lister, "/work/other", wd.getwd)
		assert.NilError(t, err)
		assert.Equal(t, room, "/work/other")
		assert.Equal(t, lister.calls, 0)
		assert.Equal(t, wd.calls, 0)
	})

	t.Run("a directory inside a room follows that room", func(t *testing.T) {
		lister := &fakeLister{rooms: []string{"/work/other", "/work/proj"}}
		wd := &fakeGetwd{dir: "/work/proj/src/pkg"}

		room, err := RoomToFollow(t.Context(), lister, "", wd.getwd)
		assert.NilError(t, err)
		assert.Equal(t, room, "/work/proj")
	})

	t.Run("a directory under no room is followed itself", func(t *testing.T) {
		lister := &fakeLister{rooms: []string{"/work/proj"}}
		wd := &fakeGetwd{dir: "/home/operator/next"}

		room, err := RoomToFollow(t.Context(), lister, "", wd.getwd)
		assert.NilError(t, err)
		assert.Equal(t, room, "/home/operator/next")
	})

	t.Run("failures are returned", func(t *testing.T) {
		listErr := errors.New("daemon down")
		_, err := RoomToFollow(t.Context(), &fakeLister{err: listErr}, "",
			(&fakeGetwd{dir: "/work"}).getwd)
		assert.ErrorIs(t, err, listErr)

		wdErr := errors.New("no cwd")
		_, err = RoomToFollow(t.Context(), &fakeLister{}, "", (&fakeGetwd{err: wdErr}).getwd)
		assert.ErrorIs(t, err, wdErr)
	})
}
