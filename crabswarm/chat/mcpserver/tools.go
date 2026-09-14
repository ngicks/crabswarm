package mcpserver

import (
	"bytes"
	"context"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The tools are the member verbs of `crabswarm chat`, named with the prefix a
// harness shows the model, and worded for one: the descriptions say what the
// agent gets out of a call, since that is what it picks a tool by.
//
// Only the three an agent needs mid-turn are here. Reporting harness state is
// deliberately absent: the daemon learns that from the harness's own hooks,
// and a model asked to report its own state would be reporting the one thing
// it cannot observe about itself.

// sendArgs says who a message is for and what it says. The target is passed
// through as written: resolving a role is the daemon's job, and its refusal
// names the qualified form to retry an ambiguous bare name with.
//
// Both fields are required, so a board post is written as an empty target
// rather than as a missing one — the same choice `chat send "" <text>` makes a
// person type. Who a message is for is the decision a sender is making, and a
// tool that let it be left out would let it be forgotten.
//
//nolint:lll // a struct tag cannot be wrapped, and the description is what the model reads
type sendArgs struct {
	To      string `json:"to" jsonschema:"who the message is for: everyone, a role as name or team/name, several separated by commas, or empty for a board post"`
	Message string `json:"message"`
}

// readArgs is one read as its caller writes it, mirroring the flags `chat read`
// takes. Every field is optional: each one left out is the daemon's default for
// the read being made, which is the read an agent catching up wants.
//
//nolint:lll // a struct tag cannot be wrapped, and the descriptions are what the model reads
type readArgs struct {
	Cursor string `json:"cursor,omitempty" jsonschema:"where to read from: unread (default), head or tail. Whatever is shown counts as read"`
	Range  int32  `json:"range,omitempty" jsonschema:"how many messages from the cursor, forward when positive, backward when negative; default ten in the cursor's direction"`
	To     string `json:"to,omitempty" jsonschema:"only messages naming any of these roles, or everyone"`
	Since  int64  `json:"since,omitempty" jsonschema:"only messages after this seq"`
	Until  int64  `json:"until,omitempty" jsonschema:"only messages before this seq"`
}

// noArgs is the input of the tool that acts on the caller alone. The schema has
// to be an object, so it is an empty struct rather than nothing.
type noArgs struct{}

// addTools registers the three verbs on the MCP server. They are added before
// the first session so the tools capability is advertised during the
// handshake, rather than announced as a change the harness has to notice.
func (s *Server) addTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "chat_send",
		Description: "Send a message into your room, to somebody or to nobody. " +
			`"everyone" is the whole room; a comma-separated list of roles ` +
			"mentions each of them, written as team/name or as a bare name " +
			"that resolves within your team first and then room-wide when it " +
			"is unique there; an empty target is a board post, which is in the " +
			"room and mentions nobody. The answer names every role the target " +
			"resolved to, and warns about any of them nobody is attending " +
			"under — that mention waits until somebody does.",
	}, s.send)
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "chat_read",
		Description: "Read messages of your room, oldest first, and move your " +
			"read position to the newest one shown — whichever cursor was " +
			"used, the tail included, so what a read shows you it does not " +
			"show you again as unread. With no arguments it hands over the " +
			"first ten unread: the messages naming you or everyone that you " +
			"have not been shown. A line marked [mentioned you] is addressed " +
			"to you; a trailing count says how much unread is left.",
	}, s.read)
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "chat_members",
		Description: "List everyone attending your room, one member per line. " +
			"The first column is exactly the role chat_send addresses; the " +
			"ones after it are the kind — agent for a harness a message is " +
			"typed into, human for someone who reads when it asks — and the " +
			"state that harness last reported.",
	}, s.members)
}

func (s *Server) send(
	ctx context.Context, _ *mcp.CallToolRequest, in sendArgs,
) (*mcp.CallToolResult, any, error) {
	target, err := cli.ParseTarget(in.To)
	if err != nil {
		return nil, nil, err
	}
	return s.call(ctx, func(w io.Writer, token string) error {
		resp, err := s.client.Send(ctx, token, target, in.Message)
		if err != nil {
			return err
		}
		// The warnings go to the same writer as the delivery lines. On a
		// command line they are held apart because one is the answer and the
		// other is for the person watching; a tool result has one text, and a
		// mention nobody is attending under is exactly what its reader has to
		// act on.
		return cli.RenderSent(w, w, resp)
	})
}

func (s *Server) read(
	ctx context.Context, _ *mcp.CallToolRequest, in readArgs,
) (*mcp.CallToolResult, any, error) {
	filter, err := cli.ReadFlags{
		Cursor: in.Cursor,
		Range:  in.Range,
		To:     in.To,
		Since:  in.Since,
		Until:  in.Until,
	}.Filter()
	if err != nil {
		return nil, nil, err
	}
	return s.call(ctx, func(w io.Writer, token string) error {
		// The two options `chat read` carries beside the filter exist for
		// harness hooks deciding whether they have mail to deliver, which is
		// not a decision the model calling this tool is making.
		return s.client.ReadInto(ctx, w, token, cli.ReadOptions{Filter: filter})
	})
}

func (s *Server) members(
	ctx context.Context, _ *mcp.CallToolRequest, _ noArgs,
) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, func(w io.Writer, token string) error {
		return s.client.ListMembers(ctx, w, token)
	})
}

// call runs one chat call and hands the client's own rendering back as the
// tool's result, unchanged down to the trailing newline.
//
// Rendering through [cli.Client] rather than formatting here is the whole
// point: the tool result and the matching `crabswarm chat` verb print the same
// text, so the two ways of being in a room stay one thing to learn — and a
// change to how a message reads reaches both at once.
//
// Attendance is waited on first because none of these calls mean anything from
// outside the room, and a member whose attendance never landed would otherwise
// get the daemon's answer to a question it should not have asked. The token is
// resolved before that, so a bridge configured with none reports what is
// missing rather than waiting on an attendance that was never going to happen.
func (s *Server) call(
	ctx context.Context,
	rpc func(w io.Writer, token string) error,
) (*mcp.CallToolResult, any, error) {
	token, err := cli.ResolveToken(s.token)
	if err != nil {
		return nil, nil, err
	}
	if err := s.awaitAttendance(ctx); err != nil {
		return nil, nil, err
	}
	var rendered bytes.Buffer
	if err := rpc(&rendered, token); err != nil {
		return nil, nil, err
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: rendered.String()}},
	}, nil, nil
}
