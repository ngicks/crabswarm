package cli

import (
	"context"
	"io"

	"google.golang.org/grpc"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Join declares attendance under name — empty for the name the daemon derives
// from the token — and reports the identity the daemon settled on, which is
// where the caller learns its own room and team.
func (c *Client) Join(ctx context.Context, w io.Writer, token, name string) error {
	resp, err := c.chat.Join(withToken(ctx, token), &chatv1.JoinRequest{Name: name})
	if err != nil {
		return callError(err)
	}
	return RenderJoined(w, resp.GetSelf())
}

// Send delivers text to one member of the caller's room, addressed as "name"
// or "team/name". The address is passed through untouched: resolving it is the
// daemon's job, and its error names the qualified form to retry a colliding
// bare name with.
func (c *Client) Send(ctx context.Context, w io.Writer, token, to, text string) error {
	resp, err := c.chat.Send(withToken(ctx, token), &chatv1.SendRequest{To: to, Text: text})
	if err != nil {
		return callError(err)
	}
	return RenderSent(w, resp.GetRecipient())
}

// Broadcast delivers text to every other member of the caller's room.
func (c *Client) Broadcast(ctx context.Context, w io.Writer, token, text string) error {
	resp, err := c.chat.Broadcast(withToken(ctx, token), &chatv1.BroadcastRequest{Text: text})
	if err != nil {
		return callError(err)
	}
	return RenderBroadcast(w, resp.GetDeliveredCount())
}

// ReadOptions tunes a read for the caller driving it. The zero value is the
// read a human types: an empty inbox says so, and the read changes nothing but
// the inbox.
type ReadOptions struct {
	// Quiet drops the empty-inbox line, so the output is non-empty exactly
	// when messages were handed over.
	//
	// It exists for harness hooks, which have to decide whether they have
	// anything to deliver. Without it the decision is a comparison against the
	// sentence [RenderMessages] prints, which puts a wording nobody thinks of
	// as an interface between the renderer and every hook that reads it.
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

// Read prints the caller's pending messages and consumes them. A failure to
// write them is returned, but the daemon has already handed them over by then:
// they are gone either way, which is why the rendering is kept simple enough
// not to fail on its own.
func (c *Client) Read(ctx context.Context, w io.Writer, token string, opts ReadOptions) error {
	resp, err := c.chat.Read(withToken(ctx, token), &chatv1.ReadRequest{})
	if err != nil {
		return callError(err)
	}
	messages := resp.GetMessages()
	if len(messages) == 0 {
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
	return RenderMessages(w, messages)
}

// ListMembers prints everyone attending the caller's room.
func (c *Client) ListMembers(ctx context.Context, w io.Writer, token string) error {
	resp, err := c.chat.ListMembers(withToken(ctx, token), &chatv1.ListMembersRequest{})
	if err != nil {
		return callError(err)
	}
	return RenderMembers(w, resp.GetMembers())
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

// Address spells a member the way every chat verb addresses one — "team/name",
// or the bare name when the daemon reported no team.
//
// It is exported for the callers that present the roster themselves instead of
// printing [RenderMembers]'s lines: they owe their reader the same address, and
// assembling a second one would be a second spelling to keep in step.
func Address(m *chatv1.Member) string {
	return qualify(m)
}

// MemberAddresses returns the room's attendance as the addresses `chat send`
// takes. It backs shell completion, which needs the values themselves rather
// than the listing [Client.ListMembers] prints.
func (c *Client) MemberAddresses(ctx context.Context, token string) ([]string, error) {
	members, err := c.Members(ctx, token)
	if err != nil {
		return nil, err
	}
	addresses := make([]string, 0, len(members))
	for _, m := range members {
		addresses = append(addresses, qualify(m))
	}
	return addresses, nil
}

// WatchRoom opens the caller's feed of what happens in its room. The stream
// runs until ctx is cancelled or the daemon ends it; a caller that falls behind
// is dropped with an error, and the answer to that is to list the room again
// and watch anew.
//
// It hands back the stream rather than rendering it: an event feed has no CLI
// verb of its own, and the callers that have one — a bridge keeping a view of
// the room current — act on the events instead of printing them.
func (c *Client) WatchRoom(
	ctx context.Context,
	token string,
) (grpc.ServerStreamingClient[chatv1.RoomEvent], error) {
	stream, err := c.chat.WatchRoom(withToken(ctx, token), &chatv1.WatchRoomRequest{})
	if err != nil {
		return nil, callError(err)
	}
	return stream, nil
}

// Leave withdraws the caller's attendance.
func (c *Client) Leave(ctx context.Context, w io.Writer, token string) error {
	if _, err := c.chat.Leave(withToken(ctx, token), &chatv1.LeaveRequest{}); err != nil {
		return callError(err)
	}
	return RenderLeft(w)
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
