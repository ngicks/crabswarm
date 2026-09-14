package cli

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Attendance is an open Attend stream: the caller's membership of the room for
// as long as it lasts, and the room's event feed while it does.
//
// Cancelling the context [Client.Attend] was given is what ends it — there is
// nothing else to close, and the daemon serves the stream until the client goes
// away. A caller that stops reading the feed without cancelling keeps attending,
// so one that means to attend again gives each attendance a context of its own.
//
// It is handed back rather than rendered because attendance has no verb of its
// own any more. What holds one is a session: the MCP bridge for an agent, the
// daemon's own registration for a person.
type Attendance struct {
	self   *chatv1.Member
	stream grpc.ServerStreamingClient[chatv1.RoomEvent]
}

// Attend declares attendance under name — empty for the default the daemon
// derives from the compose labels, or from the token when it has none — and
// returns once the daemon has acknowledged it, so a caller knows the attendance
// landed before it starts reading the feed.
//
// kind says what attends. An agent harness is typed into when a message
// arrives; anything else — a person at a shell, a script — is only ever handed
// its messages when it asks. The daemon refuses a kind it was not told, so a
// caller passes one.
//
// The stream is lazy: nothing is sent until the first event is waited for, so a
// refusal — an unknown token, a role already attending — surfaces here rather
// than at the call above.
func (c *Client) Attend(
	ctx context.Context,
	token, name string,
	kind chatv1.MemberKind,
) (*Attendance, error) {
	stream, err := c.chat.Attend(withToken(ctx, token),
		&chatv1.AttendRequest{Name: name, Kind: kind})
	if err != nil {
		return nil, attendError(ctx, err)
	}
	ev, err := stream.Recv()
	if err != nil {
		return nil, attendError(ctx, err)
	}
	self := ev.GetAttended().GetSelf()
	if self == nil {
		return nil, fmt.Errorf(
			"the daemon opened the attendance with %T instead of naming the member",
			ev.GetEvent())
	}
	return &Attendance{self: self, stream: stream}, nil
}

// attendError reports a caller's own cancellation as itself rather than as the
// transport failure the stream raises on the way down. A caller that gave up
// would otherwise read its exit as a refusal worth retrying, which is the one
// thing a bridge shutting down must not do.
func attendError(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return callError(err)
}

// Self is the member the token resolved to, which is where the caller learns
// its own room, team and name. It is also what a caller matches the feed
// against: an event names a member by team and name and nothing else.
func (a *Attendance) Self() *chatv1.Member {
	return a.self
}

// Recv hands over the next event of the room, for a caller that drives the feed
// itself — one that acts on some events and stops on others.
func (a *Attendance) Recv() (*chatv1.RoomEvent, error) {
	ev, err := a.stream.Recv()
	if err != nil {
		return nil, callError(err)
	}
	return ev, nil
}

// Forward hands every event to onEvent until the stream ends, an event is
// refused by onEvent — whose error is returned as it stands, so a caller can
// stop the feed by returning a sentinel of its own — or ctx is done.
//
// ctx is the one the attendance was opened with, or one derived from it:
// returning on a sentinel leaves the stream open, and cancelling that context
// is the only thing that ends it.
//
// A context that ended is reported as its own error rather than as the
// transport failure the stream raises on the way down, so a caller retrying a
// dropped attendance can tell that from its own shutdown.
func (a *Attendance) Forward(ctx context.Context, onEvent func(*chatv1.RoomEvent) error) error {
	for {
		ev, err := a.stream.Recv()
		if err != nil {
			return attendError(ctx, err)
		}
		if err := onEvent(ev); err != nil {
			return err
		}
	}
}

// Send appends text to the caller's room, addressed to target — nil for a board
// post, which mentions nobody.
//
// It hands the answer back instead of printing it: the answer says which roles
// were mentioned and which of them nobody is attending, and a caller that is
// not a command line acts on that rather than rendering it. [RenderSent] is
// what prints one.
func (c *Client) Send(
	ctx context.Context,
	token string,
	target *chatv1.Target,
	text string,
) (*chatv1.SendResponse, error) {
	resp, err := c.chat.Send(withToken(ctx, token),
		&chatv1.SendRequest{Target: target, Text: text})
	if err != nil {
		return nil, callError(err)
	}
	return resp, nil
}

// Read hands back messages of the caller's room from the filter's cursor and
// moves the caller's read position to the newest one shown, whichever cursor
// was used. A nil filter is the daemon's default: the first ten unread
// mentions.
func (c *Client) Read(
	ctx context.Context,
	token string,
	filter *chatv1.ReadFilter,
) (*chatv1.ReadResponse, error) {
	resp, err := c.chat.Read(withToken(ctx, token), &chatv1.ReadRequest{Filter: filter})
	if err != nil {
		return nil, callError(err)
	}
	return resp, nil
}

// ReadOptions is a read as a command line makes one: which messages, and what
// to do about having found none. The zero value is the read a human types.
type ReadOptions struct {
	// Filter says which messages to show. Nil leaves it to the daemon.
	Filter *chatv1.ReadFilter
	// Quiet drops the empty-read line, so the output is non-empty exactly when
	// messages were handed over.
	//
	// It exists for harness hooks, which have to decide whether they have
	// anything to deliver. Without it the decision is a comparison against the
	// sentence [RenderRead] prints, which puts a wording nobody thinks of as an
	// interface between the renderer and every hook that reads it.
	Quiet bool
	// DoneWhenEmpty reports the caller done when the read handed nothing over.
	//
	// It rides on the read rather than being a hook entry of its own because
	// the two decisions are the same decision: a turn-ending drain either
	// delivers messages — and the turn continues, so the member is not done —
	// or it delivers none and the member goes quiet. Hooks wired to one event
	// run concurrently, so a separate report-state entry would race the
	// delivering path and mark a continuing turn done.
	//
	// A read that failed reports nothing: the caller's state is unknown, and
	// the daemon that would hear the report is the one that just did not
	// answer.
	DoneWhenEmpty bool
}

// ReadInto reads and prints, which is what the `chat read` verb and the bridge
// tool behind it both do.
//
// A failure to write the messages is returned, but the read has already moved
// the caller's position by then: the same messages will not come back as
// unread, which is why the rendering is kept simple enough not to fail on its
// own.
func (c *Client) ReadInto(
	ctx context.Context,
	w io.Writer,
	token string,
	opts ReadOptions,
) error {
	resp, err := c.Read(ctx, token, opts.Filter)
	if err != nil {
		return err
	}
	if len(resp.GetMessages()) == 0 {
		if opts.DoneWhenEmpty {
			done := chatv1.HarnessState_HARNESS_STATE_DONE
			if err := c.reportState(ctx, token, done); err != nil {
				return err
			}
		}
		if opts.Quiet {
			return nil
		}
	}
	return RenderRead(w, resp)
}

// ListMembers prints everyone attending the caller's room.
func (c *Client) ListMembers(ctx context.Context, w io.Writer, token string) error {
	members, err := c.Members(ctx, token)
	if err != nil {
		return err
	}
	return RenderMembers(w, members)
}

// Members returns the room's attendance as the daemon reports it, states
// included. It is for the callers that present the roster some other way than
// the listing [Client.ListMembers] prints — a structured document, say — and so
// need the members themselves rather than lines about them.
func (c *Client) Members(ctx context.Context, token string) ([]*chatv1.Member, error) {
	resp, err := c.chat.ListMembers(withToken(ctx, token), &chatv1.ListMembersRequest{})
	if err != nil {
		return nil, callError(err)
	}
	return resp.GetMembers(), nil
}

// MemberAddresses returns the room's attendance as the addresses a target names
// a role by. It backs shell completion, which needs the values themselves
// rather than the listing [Client.ListMembers] prints.
func (c *Client) MemberAddresses(ctx context.Context, token string) ([]string, error) {
	members, err := c.Members(ctx, token)
	if err != nil {
		return nil, err
	}
	addresses := make([]string, 0, len(members))
	for _, m := range members {
		addresses = append(addresses, Address(m))
	}
	return addresses, nil
}

// ReportState records the state of the harness the caller runs under, naming it
// with one of [HarnessStateNames].
//
// It prints nothing. Harness hooks drive this on every turn, and their stdout
// is read back by the harness itself, so a confirmation line would be noise the
// agent has to skip past.
func (c *Client) ReportState(ctx context.Context, token, state string) error {
	parsed, err := ParseHarnessState(state)
	if err != nil {
		return err
	}
	return c.reportState(ctx, token, parsed)
}

// reportState is [Client.ReportState] past the word-to-enum step, for the
// callers that already hold the state as a value rather than as something a
// user typed.
func (c *Client) reportState(
	ctx context.Context,
	token string,
	state chatv1.HarnessState,
) error {
	_, err := c.chat.ReportState(
		withToken(ctx, token),
		&chatv1.ReportStateRequest{State: state},
	)
	return callError(err)
}
