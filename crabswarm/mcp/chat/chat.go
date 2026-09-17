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
	crabmcp "github.com/ngicks/crabswarm/crabswarm/mcp"
)

// family is the chat verbs bound to the server they act through. One server
// carries one of these: what a tool may do follows from the attendance that
// server holds, so the two are never apart.
type family struct {
	server *crabmcp.Server
}

// Register adds the chat verbs and the room's roster to server.
//
// It is called before the server runs, so both are advertised during the
// handshake rather than announced as a change the harness has to notice.
func Register(server *crabmcp.Server) {
	f := &family{server: server}
	f.addTools()
	f.addResources()
}
