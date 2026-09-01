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
// Only the four an agent needs mid-turn are here. Reporting harness state is
// deliberately absent: the daemon learns that from the harness's own hooks,
// and a model asked to report its own state would be reporting the one thing
// it cannot observe about itself.

// sendArgs addresses one member. The address is passed through untouched:
// resolving it is the daemon's job, and its refusal names the qualified form
// to retry an ambiguous bare name with.
type sendArgs struct {
	To      string `json:"to" jsonschema:"the member to deliver to, as name or team/name"`
	Message string `json:"message" jsonschema:"the text to deliver"`
}

type broadcastArgs struct {
	Message string `json:"message" jsonschema:"the text to deliver to everyone else in the room"`
}

// noArgs is the input of the two tools that act on the caller alone. The
// schema has to be an object, so it is an empty struct rather than nothing.
type noArgs struct{}

// addTools registers the four verbs on the MCP server. They are added before
// the first session so the tools capability is advertised during the
// handshake, rather than announced as a change the harness has to notice.
func (s *Server) addTools() {
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "chat_send",
		Description: "Send a message to one member of your room. " +
			"A bare name resolves within your team first, then room-wide when " +
			"it is unique there; a name several teams use is rejected, and the " +
			"refusal names the team/name form to retry with.",
	}, s.send)
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "chat_broadcast",
		Description: "Send a message to every other member of your room, " +
			"across teams, and report how many inboxes it reached.",
	}, s.broadcast)
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "chat_read",
		Description: "Take the messages waiting for you, oldest first. Each " +
			"message is handed over exactly once, so a later read shows only " +
			"what arrived in between.",
	}, s.read)
	mcp.AddTool(s.mcp, &mcp.Tool{
		Name: "chat_members",
		Description: "List everyone attending your room, one team-qualified " +
			"member per line. Each line is exactly the address chat_send takes.",
	}, s.members)
}

func (s *Server) send(
	ctx context.Context, _ *mcp.CallToolRequest, in sendArgs,
) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, func(w io.Writer) error {
		return s.client.Send(ctx, w, s.token, in.To, in.Message)
	})
}

func (s *Server) broadcast(
	ctx context.Context, _ *mcp.CallToolRequest, in broadcastArgs,
) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, func(w io.Writer) error {
		return s.client.Broadcast(ctx, w, s.token, in.Message)
	})
}

func (s *Server) read(
	ctx context.Context, _ *mcp.CallToolRequest, _ noArgs,
) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, func(w io.Writer) error {
		// The zero options are the read a human types: an empty inbox says so,
		// and nothing but the inbox changes. The two flags `chat read` carries
		// exist for harness hooks deciding whether they have mail to deliver,
		// which is not a decision the model calling this tool is making.
		return s.client.Read(ctx, w, s.token, cli.ReadOptions{})
	})
}

func (s *Server) members(
	ctx context.Context, _ *mcp.CallToolRequest, _ noArgs,
) (*mcp.CallToolResult, any, error) {
	return s.call(ctx, func(w io.Writer) error {
		return s.client.ListMembers(ctx, w, s.token)
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
// Attendance is checked first because none of these calls mean anything from
// outside the room, and a member whose startup join never landed would
// otherwise get the daemon's answer to a question it should not have asked.
// A refusal on the way out re-opens that question, so a bridge whose member
// the daemon has since forgotten attends again on the next call.
func (s *Server) call(
	ctx context.Context,
	rpc func(w io.Writer) error,
) (*mcp.CallToolResult, any, error) {
	if err := s.ensureJoined(ctx); err != nil {
		return nil, nil, err
	}
	var rendered bytes.Buffer
	if err := rpc(&rendered); err != nil {
		return nil, nil, s.forgetJoined(err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: rendered.String()}},
	}, nil, nil
}
