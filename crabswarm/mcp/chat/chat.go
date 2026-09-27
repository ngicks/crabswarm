// Package chat serves a crabswarm chat room's member verbs as the MCP server's
// first tool family.
//
// No chat logic lives here. Every tool forwards to the same ChatService call
// the matching `crabswarm chat` subcommand makes, through the same
// [github.com/ngicks/crabswarm/crabswarm/chat/cli.Client], and hands back the
// text that client rendered. That is what
// keeps a tool result and a CLI verb's output the same words: a room reads the
// same whether a member is wired to it through MCP or through a shell.
//
// Beside the tools, the room's attendance is offered as a resource a harness
// can hold open and subscribe to, so a view of who is around and what each of
// them is doing costs no turn. It is answered as structured data rather than in
// the CLI's words, since its reader is the harness rather than the model.
package chat

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	crabmcp "github.com/ngicks/crabswarm/crabswarm/mcp"
)

// Host is the crabswarm MCP server the family registers onto, whichever
// transport it serves: a stdio [crabmcp.Server] acting as one member, or an
// [crabmcp.HTTPServer] acting as a member per session's token.
//
// MemberOf takes the context of the call being answered as well as its
// session, since a session a shared OpenCode server opened carries the calls
// of several members and only the call itself names the one it is for.
type Host interface {
	MCP() *mcp.Server
	Client() *cli.Client
	AddResource(res *mcp.Resource, handler mcp.ResourceHandler)
	AnnounceOnRosterChange(uri string)
	MemberOf(ctx context.Context, session *mcp.ServerSession) (*crabmcp.Member, error)
}

// family is the chat verbs bound to the server they act through. One server
// carries one of these, and every call asks the server which member the
// calling session is: what a tool may do follows from that member's
// attendance.
type family struct {
	server Host
}

// Register adds the chat verbs and the room's roster to server.
//
// It is called before the server runs, so both are advertised during the
// handshake rather than announced as a change the harness has to notice.
func Register(server Host) {
	f := &family{server: server}
	f.addTools()
	f.addResources()
}

// caller is the token of the member session acts as, once that member has an
// answer about its attendance.
//
// The member is looked up first, so a session with no identity reports what is
// missing rather than waiting on an attendance that was never going to happen.
// Attendance is waited on next, because none of the daemon's member calls mean
// anything from outside the room, and a member whose attendance never landed
// would otherwise get the daemon's answer to a question it should not have
// asked.
func (f *family) caller(ctx context.Context, session *mcp.ServerSession) (string, error) {
	member, err := f.server.MemberOf(ctx, session)
	if err != nil {
		return "", err
	}
	if err := member.AwaitAttendance(ctx); err != nil {
		return "", err
	}
	return member.Token(), nil
}
