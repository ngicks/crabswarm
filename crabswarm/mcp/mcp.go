// Package mcp is the crabswarm MCP server a harness reaches its agent's room
// through; chat is its first tool family.
//
// A harness starts it as a stdio subprocess of its own, so an agent gets
// crabswarm's verbs as tools it is offered rather than as commands it has to
// remember to type — and, because the server attends the room the moment it
// starts and holds that attendance for the rest of the session, the agent is a
// member before its first turn instead of whenever it first thinks to say
// something, and a member again once a daemon that went away comes back.
//
// A harness server that hosts several agents at once reaches it over HTTP
// instead: one process serves every session, and each session names the member
// it acts as in a request header, so every agent behind that harness server is
// a member of its own. See [HTTPServer].
//
// Attendance is one open stream, and holding it is the whole of what membership
// means. The server opens it at startup and opens it again every time it ends,
// so there is nothing for a tool to declare: a tool called while the stream is
// down reports that instead of acting as somebody the room does not have.
//
// Reading that stream is also how a mention reaches the agent. The server knows
// which harness it serves, because the harness names itself in the MCP
// handshake; where that harness has a channel of its own, the member attends as
// one the daemon never types at and the server pushes the notice through the
// channel instead, off the same feed everything else is read from.
//
// What the server offers is not its own. A tool family is a sub-package that
// registers its tools and its resources onto a server before it runs, so this
// package knows the room it attends and the harness it answers, and each family
// knows the verbs it serves.
package mcp

import (
	"context"
	"log/slog"
	"os"
	"sync"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// serverName identifies the server to the harness, which lists it beside every
// other MCP server it was configured with. It names crabswarm rather than the
// family it happens to carry: one agent gets one of these, and what it offers
// grows without the harness's configuration changing.
const serverName = "crabswarm-mcp"

// Server bridges one agent's MCP stdio session to the crabswarm daemon.
type Server struct {
	host *host
	// token is the value the caller was configured with, kept as it was given.
	// It is resolved against the environment when the session starts rather
	// than here, so a harness that starts the server with no identity at all
	// gets one whose tools say what is missing instead of a subprocess that
	// exited before the handshake.
	token string

	// member is the one member this server attends as, and noIdentity is why
	// there is none. Both are written once as the session starts, before
	// anything that reads them runs.
	member     *Member
	noIdentity error

	// handshake reads the harness off the one session this server serves. It is
	// the first naming that counts: a session names one client for its whole
	// life, and a member cannot change what it attends as without attending
	// again.
	handshake sync.Once

	// claudeChannel is the channel the launcher declared the Claude Code
	// session registered, read off the environment once.
	claudeChannel harnessctl.ClaudeChannel
	// channel is the transport a direct channel pushes its events through. It
	// is set by [Server.Serve] before the session starts, and only for a
	// direct channel.
	channel *channel
}

// New dials sockPath and prepares the MCP server. Attendance is declared from
// Run, so nothing about the daemon — being down, refusing this caller, not
// existing yet — keeps the harness from getting a server it can talk to.
//
// It comes back with no tools and no resources: a family registers its own onto
// it, and everything registered before [Server.Run] is advertised during the
// handshake rather than announced as a change the harness has to notice.
//
// token is resolved the way every member verb resolves it — the value given
// here first, then the environment — but not until the session starts, so a
// server started with no identity at all still answers the handshake and
// reports the missing token through every tool the agent calls. A nil logger
// discards logs; a logger writing to stdout would corrupt the MCP stream, so
// the caller owns that choice.
func New(logger *slog.Logger, sockPath, token string) (*Server, error) {
	return newServer(logger, sockPath, token, os.Getenv)
}

// newServer is [New] with the environment handed in. What the server declares
// about a channel is read out of it before the session starts, so a test plays
// a launcher through this rather than setting a variable on the process running
// the suite — which would point a real channel at whatever is listening on the
// plugin's own port.
func newServer(
	logger *slog.Logger, sockPath, token string, getenv func(string) string,
) (*Server, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	ch, err := harnessctl.ClaudeChannelFromEnv(getenv)
	if err != nil {
		return nil, err
	}
	// Nothing is pinged: the one session ends with the harness's end of stdio,
	// which closes as the harness exits however it exits.
	h, err := newHost(logger, sockPath, getenv, 0, func(opts *mcpsdk.ServerOptions) {
		switch ch.Kind {
		case harnessctl.ClaudeChannelFakechat:
			opts.Instructions = claudeChannelInstructions
		case harnessctl.ClaudeChannelDirect:
			opts.Capabilities = claudeChannelCapabilities()
			opts.Instructions = claudeDirectChannelInstructions
		}
	})
	if err != nil {
		return nil, err
	}
	s := &Server{host: h, token: token, claudeChannel: ch}
	// The reserved calls are answered and learn nothing here: a stdio server
	// serves one session, and the thread a delivery goes to is the one a person
	// used last on the app server — see [harnessctl.Detect]. The OpenCode
	// session a call names is taken off it for the same reason: the one member
	// this server attends as is who every call acts as.
	middleware := []mcpsdk.Middleware{
		s.readsHandshake,
		answersReservedCalls(nil),
		takesOpenCodeSession,
	}
	if ch.Kind == harnessctl.ClaudeChannelDirect {
		middleware = append(middleware, refusesTheNewerHandshake)
	}
	h.mcp.AddReceivingMiddleware(middleware...)
	return s, nil
}

// claudeChannelCapabilities declares the channel Claude Code registers a
// listener for.
//
// It is declared only for a session that was launched as a direct channel: the
// capability is what makes Claude Code listen, and a listener on a server that
// was never registered would be one nothing ever reaches.
//
// Logging is spelled out because setting any capability at all replaces the
// SDK's default set, and the default is logging alone; leaving it out would
// quietly take logging away from a channel session and no other. Everything
// else the server offers is inferred from what the families registered.
func claudeChannelCapabilities() *mcpsdk.ServerCapabilities {
	return &mcpsdk.ServerCapabilities{
		//nolint:staticcheck // deprecated only from the revision above the cap this server offers
		Logging: &mcpsdk.LoggingCapabilities{},
		Experimental: map[string]any{
			harnessctl.ClaudeChannelCapability: map[string]any{},
		},
	}
}

// claudeDirectChannelInstructions tells the model what a direct channel event
// from this server is and what to do about it.
//
// The harness shows the model an event and nothing else — no tool call, no
// transcript entry saying where it came from — so without this the first one
// reads as a line that appeared from nowhere. It also says not to answer down
// the channel: the channel carries one direction, and a reply written into it
// would go nowhere while the room waited.
const claudeDirectChannelInstructions = "Events from " + serverName +
	` arrive as <channel source="` + serverName +
	`" from="<team/name>" room="<room>">. Each one says a chat message is ` +
	"waiting for you: read it with the chat_read tool and answer with " +
	"chat_send. Do not reply through the channel."

// claudeChannelInstructions tells the model what the event a room notice reaches
// it as is, and which tool answers it.
//
// The harness shows the model an event another plugin's channel pushed, and that
// plugin's own instructions tell the model to answer with its reply tool — which
// reaches a browser tab nobody is watching, while the room goes on waiting. So
// the event is named as precisely as this end can name it: the source the plugin
// labels it with, the message id this server numbers its uploads by, and the
// opening of the line the room worded.
const claudeChannelInstructions = "Room notices from " + serverName +
	` reach you as <channel source="fakechat" chat_id="web" ` +
	`message_id="crabswarm-..."> events whose content opens with ` +
	"[crabswarm chat]. Each one says a chat message is waiting for you: " +
	"read it with the chat_read tool and answer with chat_send. " +
	"Never answer such an event with fakechat's reply tool."

// readsHandshake watches what the harness sends for the point at which it has
// said who it is.
//
// The middleware runs after the handler rather than on one named method,
// because which frame carries the client's name depends on the protocol the two
// sides settled on: the handshake a harness opens with today names it in the
// initialize request, and the newer one that replaces that handshake carries the
// same name in the metadata of whatever the client asks first. Reading it off
// the session afterwards covers both, where waiting for the initialized
// notification would leave a member on the newer protocol attending nothing at
// all — it never sends one.
func (s *Server) readsHandshake(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
	return func(
		ctx context.Context, method string, req mcpsdk.Request,
	) (mcpsdk.Result, error) {
		res, err := next(ctx, method, req)
		if session, ok := req.GetSession().(*mcpsdk.ServerSession); ok {
			s.handshook(session)
		}
		return res, err
	}
}

// handshook reads the harness off session once it names a client, which is
// where the member learns what it runs and how a mention can reach it.
func (s *Server) handshook(session *mcpsdk.ServerSession) {
	params := session.InitializeParams()
	if params == nil {
		return
	}
	s.handshake.Do(func() {
		opts := harnessctl.Options{ClaudeChannel: s.claudeChannel}
		// Assigned only when set: a nil *channel in the interface would read as
		// a session to push through.
		if s.channel != nil {
			opts.Session = s.channel
		}
		harness := s.host.harnessOf(params, opts)
		if s.member != nil {
			s.member.retarget(harness, chatv1.NudgeDelivery_NUDGE_DELIVERY_UNSPECIFIED)
		}
	})
}

// MCP is the SDK server a family adds its tools to. It is handed over rather
// than wrapped because the SDK adds a tool through a generic function, which a
// method here could not be.
func (s *Server) MCP() *mcpsdk.Server {
	return s.host.mcp
}

// Client is the connection to the daemon every tool acts through.
func (s *Server) Client() *cli.Client {
	return s.host.client
}

// AddResource registers a resource and lets a harness subscribe to it.
func (s *Server) AddResource(res *mcpsdk.Resource, handler mcpsdk.ResourceHandler) {
	s.host.addResource(res, handler)
}

// AnnounceOnRosterChange has the subscribed sessions told to read uri again
// whenever the room's attendance or anyone's state changes — including after an
// attendance the server had to open again, since whatever happened while
// nothing was attending went unannounced.
func (s *Server) AnnounceOnRosterChange(uri string) {
	s.host.announceOnRosterChange(uri)
}

// MemberOf is the member a tool called over session acts as. A stdio server
// serves one session and attends as one member, so neither the call nor the
// session names anything it does not already know.
//
// A server that resolved no identity answers with why, which is the words every
// member verb answers a missing token with; a family hands that to the agent
// rather than acting as nobody.
func (s *Server) MemberOf(context.Context, *mcpsdk.ServerSession) (*Member, error) {
	switch {
	case s.member != nil:
		return s.member, nil
	case s.noIdentity != nil:
		return nil, s.noIdentity
	default:
		return nil, errNotAttending
	}
}

// Run serves MCP on stdin/stdout until ctx is done.
//
// It serves the one session its harness starts it for and closes the
// connection to the daemon on the way out, so a Server is not reusable: a
// harness that reconnects starts a new process, which is the only thing a
// stdio server can mean by a new session.
func (s *Server) Run(ctx context.Context) error {
	return s.Serve(ctx, &mcpsdk.StdioTransport{})
}

// Serve is [Server.Run] with the transport given, so a caller can drive a
// session over something other than the process's own stdio.
func (s *Server) Serve(ctx context.Context, transport mcpsdk.Transport) error {
	defer func() { _ = s.host.client.Close() }()

	// A direct channel pushes its events down the transport the session runs
	// on, so the transport is wrapped to keep hold of its connection, and to cap
	// the protocol revision Claude Code still delivers channel events on. Set
	// before the session starts: the handshake is what hands it to the harness.
	if s.claudeChannel.Kind == harnessctl.ClaudeChannelDirect {
		s.channel = newChannel(transport)
		transport = s.channel
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)
	token, err := cli.ResolveToken(s.token)
	if err != nil {
		// Neither the flag nor the environment changes while the process runs,
		// so a server with no identity to resolve has nothing to attend with,
		// and retrying would only say so again. It still serves, and the tools
		// report this same refusal, which is where the agent reads what is
		// missing.
		s.noIdentity = err
		s.host.logger.Error("no chat identity; the tools will report it", "error", err)
	} else {
		s.member = s.host.newMember(token)
		g.Go(func() error {
			s.member.run(gctx)
			return nil
		})
	}
	g.Go(func() error {
		// The session ending ends the attendance too: a server whose harness is
		// gone has nobody left to attend for, and without this the loop would
		// hold the session open for the rest of its backoff.
		defer cancel()
		return s.host.mcp.Run(gctx, transport)
	})
	return g.Wait()
}
