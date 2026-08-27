package cli

import (
	"context"
	"io"

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

// Read prints the caller's pending messages and consumes them. A failure to
// write them is returned, but the daemon has already handed them over by then:
// they are gone either way, which is why the rendering is kept simple enough
// not to fail on its own.
func (c *Client) Read(ctx context.Context, w io.Writer, token string) error {
	resp, err := c.chat.Read(withToken(ctx, token), &chatv1.ReadRequest{})
	if err != nil {
		return callError(err)
	}
	return RenderMessages(w, resp.GetMessages())
}

// ListMembers prints everyone attending the caller's room.
func (c *Client) ListMembers(ctx context.Context, w io.Writer, token string) error {
	resp, err := c.chat.ListMembers(withToken(ctx, token), &chatv1.ListMembersRequest{})
	if err != nil {
		return callError(err)
	}
	return RenderMembers(w, resp.GetMembers())
}

// MemberAddresses returns the room's attendance as the addresses `chat send`
// takes. It backs shell completion, which needs the values themselves rather
// than the listing [Client.ListMembers] prints.
func (c *Client) MemberAddresses(ctx context.Context, token string) ([]string, error) {
	resp, err := c.chat.ListMembers(withToken(ctx, token), &chatv1.ListMembersRequest{})
	if err != nil {
		return nil, callError(err)
	}
	addresses := make([]string, 0, len(resp.GetMembers()))
	for _, m := range resp.GetMembers() {
		addresses = append(addresses, qualify(m))
	}
	return addresses, nil
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
	_, err = c.chat.ReportState(
		withToken(ctx, token),
		&chatv1.ReportStateRequest{State: parsed},
	)
	return callError(err)
}
