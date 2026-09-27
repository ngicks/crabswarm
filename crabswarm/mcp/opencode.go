package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// A TUI attached to a shared `opencode serve` is a member of its own, though no
// MCP session is: the server opens one session for every TUI attached to it,
// and a tool call on that session names no TUI. Two plugins close the gap. The
// TUI's plugin runs with that TUI's own token and registers over the routes
// below: it holds a notice stream open, which keeps the member in the room and
// is the channel a mention reaches it through, and it says which OpenCode
// session the TUI shows. The server's plugin adds the OpenCode session making a
// call to the call's arguments, and the call acts as the member whose TUI shows
// that session.
//
// The routes sit beside the MCP handler rather than behind it. The TUI's plugin
// may reach this listener from another network namespace, under a host name
// that is not loopback, and the MCP handler refuses any such request to a
// loopback listener.

const (
	// openCodeNoticesRoute is the notice stream a TUI's plugin holds open. Its
	// [TokenHeader] names the member.
	openCodeNoticesRoute = "GET /opencode/notices"
	// openCodeSessionRoute records the OpenCode session a TUI shows, in an
	// [openCodeShowing] body. Its [TokenHeader] names the member.
	openCodeSessionRoute = "PUT /opencode/session"
	// openCodeReadRoute reads as the member whose TUI shows the session in the
	// path, which the server's plugin hands to a turn in progress.
	openCodeReadRoute = "POST /opencode/sessions/{sessionID}/read"
)

// openCodeSessionArg is the argument the server's plugin adds to every call of
// this server's tools, naming the OpenCode session that made the call.
const openCodeSessionArg = "_crabswarm_session"

// openCodeSessionKey carries the OpenCode session a tool call named, from the
// middleware that took it off the call to [HTTPServer.MemberOf].
type openCodeSessionKey struct{}

// takesOpenCodeSession takes the OpenCode session argument off every tool call
// and hands what it named on in the context. A tool never sees it: no tool's
// schema declares it, and a call still carrying it would be refused before the
// tool ran.
func takesOpenCodeSession(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
	return func(ctx context.Context, method string, req mcpsdk.Request) (mcpsdk.Result, error) {
		if method != methodCallTool {
			return next(ctx, method, req)
		}
		params, _ := req.GetParams().(*mcpsdk.CallToolParamsRaw)
		if params == nil {
			return next(ctx, method, req)
		}
		if session, ok := takeSessionArg(params); ok {
			ctx = context.WithValue(ctx, openCodeSessionKey{}, session)
		}
		return next(ctx, method, req)
	}
}

// takeSessionArg removes the OpenCode session argument from params, reporting
// what it named and whether it was there. A value that is not a string names
// no session.
func takeSessionArg(params *mcpsdk.CallToolParamsRaw) (string, bool) {
	var args map[string]json.RawMessage
	if json.Unmarshal(params.Arguments, &args) != nil {
		return "", false
	}
	raw, ok := args[openCodeSessionArg]
	if !ok {
		return "", false
	}
	delete(args, openCodeSessionArg)
	rest, err := json.Marshal(args)
	if err != nil {
		return "", false
	}
	params.Arguments = rest
	var session string
	_ = json.Unmarshal(raw, &session)
	return session, true
}

// servesOpenCode reports whether b is a session an OpenCode server opened,
// whose calls each name the member they are for.
func (b *binding) servesOpenCode() bool {
	return b.harness != nil && b.harness.Kind() == chatv1.Harness_HARNESS_OPENCODE
}

// errNoOpenCodeSession refuses a call on an OpenCode session that named no
// OpenCode session.
var errNoOpenCodeSession = errors.New("no crabswarm identity: this call came from " +
	"a shared OpenCode server without the " + openCodeSessionArg + " argument, " +
	"which names the OpenCode session making the call and which the crabswarm " +
	"plugin inside opencode serve adds")

// openCodeCaller is the member whose TUI shows the OpenCode session ctx names.
// It runs with mu held.
func (s *HTTPServer) openCodeCaller(ctx context.Context) (*Member, error) {
	session, ok := ctx.Value(openCodeSessionKey{}).(string)
	if !ok {
		return nil, errNoOpenCodeSession
	}
	a := s.openCodeSessions[session]
	if a == nil {
		return nil, fmt.Errorf("this OpenCode session (%q) has no crabswarm identity: "+
			"no attached TUI registered it as the session it shows, "+
			"and a subagent's session is never registered", session)
	}
	return a.member, nil
}

// routeOpenCode serves the routes a TUI's plugin and the server's plugin reach
// this server through.
func (s *HTTPServer) routeOpenCode(mux *http.ServeMux) {
	mux.HandleFunc(openCodeNoticesRoute, s.serveNotices)
	mux.HandleFunc(openCodeSessionRoute, s.recordShowing)
	mux.HandleFunc(openCodeReadRoute, s.readShowing)
}

// noStreamToken answers a TUI route that named no member.
const noStreamToken = "no " + TokenHeader + " header: it names the member the request is for"

// serveNotices holds one notice stream open for the member the request's token
// names, which attends for as long as one is open. Each notice is one JSON line
// of the stream.
func (s *HTTPServer) serveNotices(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeader)
	if token == "" {
		http.Error(w, noStreamToken, http.StatusBadRequest)
		return
	}
	a, stream := s.attachStream(token, w)
	if stream == nil {
		http.Error(w, "the server is shutting down", http.StatusServiceUnavailable)
		return
	}
	defer s.detachStream(a, stream)
	if err := stream.Open(); err != nil {
		s.host.logger.Warn("opening an opencode notice stream failed", "error", err)
		return
	}
	select {
	case <-r.Context().Done():
	case <-stream.Broken():
	}
}

// attachStream makes w a notice stream of the member token names, which
// attends from the first one on. A nil stream is a server shutting down.
//
// The stream is attached before anything is written to w, so the member finds
// it from the first notice on, and the plugin — which registers its session as
// soon as the stream answers — finds the member.
func (s *HTTPServer) attachStream(
	token string, w http.ResponseWriter,
) (*attendee, *harnessctl.OpenCodeStream) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, nil
	}
	a := s.attendeeOf(token)
	if a.openCode == nil {
		a.openCode = harnessctl.NewOpenCodeNotices()
	}
	stream := a.openCode.Attach(w)
	a.streams++
	s.retarget(a)
	return a, stream
}

// detachStream ends a notice stream, and takes the member out of the room when
// it was the last thing keeping it there.
//
// The OpenCode session the member's TUI showed goes with its last stream: a TUI
// with no stream open is one nothing reaches, and a call naming its session
// acts as nobody until the TUI registers again.
func (s *HTTPServer) detachStream(a *attendee, stream *harnessctl.OpenCodeStream) {
	stream.Detach()
	s.mu.Lock()
	a.streams--
	if a.streams == 0 {
		s.forgetShowing(a)
	}
	left := s.unkept(a)
	s.mu.Unlock()
	if left {
		a.stop()
		s.host.logger.Info("the last notice stream of a chat member closed; leaving the room",
			"member", cli.Address(a.member.selfMember()))
	}
}

// openCodeShowing is what a TUI's plugin says its TUI shows.
type openCodeShowing struct {
	SessionID string `json:"sessionID"`
}

// maxShowingBody bounds a registration's body, which is one id in braces.
const maxShowingBody = 4 << 10

// recordShowing records the OpenCode session the TUI of the request's member
// shows. It is refused while no notice stream is open for that member: the
// session is where its notices go and where its calls come from, and a member
// nothing keeps in the room has neither.
func (s *HTTPServer) recordShowing(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeader)
	if token == "" {
		http.Error(w, noStreamToken, http.StatusBadRequest)
		return
	}
	var showing openCodeShowing
	err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxShowingBody)).Decode(&showing)
	switch {
	case err != nil:
		http.Error(w, `the body is not {"sessionID": "..."}: `+err.Error(), http.StatusBadRequest)
		return
	case showing.SessionID == "":
		http.Error(w, "the body names no sessionID", http.StatusBadRequest)
		return
	}
	if !s.show(token, showing.SessionID) {
		http.Error(w, "no notice stream is open for this "+TokenHeader+
			"; hold GET /opencode/notices open first", http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// show records that the TUI of the member token names shows session, and
// reports false when no notice stream is open for token. A session another TUI
// showed until now is this one's from here on.
func (s *HTTPServer) show(token, session string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.members[token]
	if a == nil || a.streams == 0 {
		return false
	}
	s.forgetShowing(a)
	a.openCodeSession = session
	s.openCodeSessions[session] = a
	return true
}

// forgetShowing drops the OpenCode session a's TUI showed, unless another TUI
// has shown it since. It runs with mu held.
func (s *HTTPServer) forgetShowing(a *attendee) {
	if a.openCodeSession != "" && s.openCodeSessions[a.openCodeSession] == a {
		delete(s.openCodeSessions, a.openCodeSession)
	}
	a.openCodeSession = ""
}

// readShowing reads as the member whose TUI shows the session in the path, and
// answers with exactly what `crabswarm chat read --quiet` prints for that
// member. Nothing to read is 204, a session no TUI shows 404, a member not in
// the room 503, and a read the daemon failed 502.
func (s *HTTPServer) readShowing(w http.ResponseWriter, r *http.Request) {
	session := r.PathValue("sessionID")
	s.mu.Lock()
	a := s.openCodeSessions[session]
	s.mu.Unlock()
	if a == nil {
		http.Error(w, fmt.Sprintf("OpenCode session %q has no crabswarm identity", session),
			http.StatusNotFound)
		return
	}
	ctx := r.Context()
	if err := a.member.AwaitAttendance(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	var out bytes.Buffer
	if err := readQuietly(ctx, s.host.client, &out, a.member.Token()); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if out.Len() == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write(out.Bytes())
}

// readQuietly is `crabswarm chat read --quiet` given no other flag: the unread
// from the member's read position, and nothing at all when there is none.
func readQuietly(ctx context.Context, client *cli.Client, w io.Writer, token string) error {
	filter, err := cli.ReadFlags{Cursor: "unread"}.Filter()
	if err != nil {
		return err
	}
	return client.ReadInto(ctx, w, token, cli.ReadOptions{Filter: filter, Quiet: true})
}
