package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/crabswarm/chat/nudge"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// A shared `opencode serve` opens one MCP session for every TUI attached to it.
// Each TUI's plugin registers over the routes beside the MCP handler, and the
// server's plugin names the OpenCode session making each call. The cases below
// play both plugins with plain HTTP requests and MCP calls.

// sentBy is one message the daemon was handed, and the token it came with.
type sentBy struct {
	Token string
	Text  string
}

// openCodeChat is [roomChat] answering sends and reads as well, and keeping
// what each member declared and how many of its attendances it let go of.
type openCodeChat struct {
	*roomChat

	mu       sync.Mutex
	declared map[string]*chatv1.AttendRequest
	cancels  map[string]int
	sent     []sentBy
	// unread is what the next read hands each member over, and what counting
	// its unread counts.
	unread map[string][]*chatv1.Message
}

func newOpenCodeChat(selves map[string]*chatv1.Member) *openCodeChat {
	return &openCodeChat{
		roomChat: newRoomChat(selves),
		declared: map[string]*chatv1.AttendRequest{},
		cancels:  map[string]int{},
		unread:   map[string][]*chatv1.Message{},
	}
}

func (c *openCodeChat) Attend(
	req *chatv1.AttendRequest, stream grpc.ServerStreamingServer[chatv1.RoomEvent],
) error {
	token := tokenOf(stream.Context())
	c.mu.Lock()
	c.declared[token] = req
	c.mu.Unlock()
	err := c.roomChat.Attend(req, stream)
	if stream.Context().Err() != nil {
		c.mu.Lock()
		c.cancels[token]++
		c.mu.Unlock()
	}
	return err
}

func (c *openCodeChat) Send(
	ctx context.Context, req *chatv1.SendRequest,
) (*chatv1.SendResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, sentBy{Token: tokenOf(ctx), Text: req.GetText()})
	return &chatv1.SendResponse{}, nil
}

func (c *openCodeChat) Read(
	ctx context.Context, req *chatv1.ReadRequest,
) (*chatv1.ReadResponse, error) {
	token := tokenOf(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	if req.GetFilter().GetCursor() != chatv1.ReadCursor_READ_CURSOR_UNREAD {
		return &chatv1.ReadResponse{}, nil
	}
	messages := c.unread[token]
	delete(c.unread, token)
	return &chatv1.ReadResponse{Messages: messages}, nil
}

func (c *openCodeChat) CountUnread(
	ctx context.Context, _ *chatv1.CountUnreadRequest,
) (*chatv1.CountUnreadResponse, error) {
	token := tokenOf(ctx)
	c.mu.Lock()
	defer c.mu.Unlock()
	return &chatv1.CountUnreadResponse{UnreadMentions: int32(len(c.unread[token]))}, nil
}

func (c *openCodeChat) setUnread(token string, messages ...*chatv1.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.unread[token] = messages
}

func (c *openCodeChat) declaredBy(token string) *chatv1.AttendRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.declared[token]
}

func (c *openCodeChat) cancelsOf(token string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cancels[token]
}

func (c *openCodeChat) sentSoFar() []sentBy {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.sent)
}

// The two agents, each a TUI attached to one `opencode serve`.
var (
	selfA = doneSelf("backend", "alice", testRoom)
	selfB = doneSelf("infra", "carol", testRoom)
)

// openCodeRoom is the two agents' room and an HTTP server beside their
// `opencode serve`, running.
type openCodeRoom struct {
	chat     *openCodeChat
	srv      *HTTPServer
	base     string
	endpoint string
}

func startOpenCodeRoom(t *testing.T) openCodeRoom {
	t.Helper()
	room := newOpenCodeChat(map[string]*chatv1.Member{tokenA: selfA, tokenB: selfB})
	srv := newTestHTTP(t, room)
	endpoint := serveHTTP(t, srv).endpoint
	return openCodeRoom{
		chat:     room,
		srv:      srv,
		base:     strings.TrimSuffix(endpoint, HTTPPath),
		endpoint: endpoint,
	}
}

// request builds a request to path on the server, naming token unless it is
// empty.
func (c openCodeRoom) request(
	t *testing.T, method, path, token string, body io.Reader,
) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), method, c.base+path, body)
	assert.NilError(t, err)
	if token != "" {
		req.Header.Set(TokenHeader, token)
	}
	return req
}

// answer sends req and returns the status, the content type and the body.
func answer(t *testing.T, req *http.Request) (int, string, string) {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	assert.NilError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	assert.NilError(t, err)
	return resp.StatusCode, resp.Header.Get("Content-Type"), string(body)
}

// show is the TUI's plugin registering the session its TUI shows, answering
// with the status.
func (c openCodeRoom) show(t *testing.T, token, body string) int {
	t.Helper()
	status, _, _ := answer(t, c.request(t, http.MethodPut, "/opencode/session", token,
		strings.NewReader(body)))
	return status
}

// read is the server's plugin reading for the member showing session.
func (c openCodeRoom) read(t *testing.T, session string) (int, string, string) {
	t.Helper()
	return answer(t, c.request(t, http.MethodPost, "/opencode/sessions/"+session+"/read", "", nil))
}

// push hands the member token names one event on its own feed.
func (c openCodeRoom) push(t *testing.T, token string, ev *chatv1.RoomEvent) {
	t.Helper()
	select {
	case c.chat.feed(token) <- ev:
	case <-time.After(eventTimeout):
		t.Fatalf("%s is not attending", token)
	}
}

// openCodeLine is one line of a notice stream, as the plugin reads it.
type openCodeLine struct {
	Content string `json:"content"`
	From    string `json:"from"`
}

// noticeStream is a TUI's plugin holding a notice stream open.
type noticeStream struct {
	lines   chan openCodeLine
	body    io.Closer
	reading errgroup.Group
}

// listen opens a notice stream as token and returns once the server answered
// it. The stream closes when the test ends, if the case did not close it.
func (c openCodeRoom) listen(t *testing.T, token string) *noticeStream {
	t.Helper()
	resp, err := http.DefaultClient.Do(
		c.request(t, http.MethodGet, "/opencode/notices", token, nil),
	)
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, resp.Header.Get("Content-Type"), harnessctl.OpenCodeNoticeContentType)

	n := &noticeStream{lines: make(chan openCodeLine, 8), body: resp.Body}
	n.reading.Go(func() error {
		defer close(n.lines)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			var line openCodeLine
			if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
				return err
			}
			n.lines <- line
		}
		return nil
	})
	t.Cleanup(func() { n.close(t) })
	return n
}

// close ends the stream, the way a TUI that quit does.
func (n *noticeStream) close(t *testing.T) {
	t.Helper()
	_ = n.body.Close()
	assert.NilError(t, n.reading.Wait())
}

// next is the next notice on the stream.
func (n *noticeStream) next(t *testing.T) openCodeLine {
	t.Helper()
	select {
	case line, ok := <-n.lines:
		assert.Assert(t, ok, "the notice stream ended")
		return line
	case <-time.After(eventTimeout):
		t.Fatal("no notice arrived on the stream")
		return openCodeLine{}
	}
}

// newMessageFrom is the notice of a mention from sender.
func newMessageFrom(sender *chatv1.Member) openCodeLine {
	addr := nudge.Address(sender.GetTeam(), sender.GetName())
	return openCodeLine{Content: nudge.NewMessage(addr), From: addr}
}

// streamsOf is how many notice streams keep the member token names in the
// room.
func streamsOf(srv *HTTPServer, token string) int {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if a := srv.members[token]; a != nil {
		return a.streams
	}
	return 0
}

// Two TUIs, each with a stream of its own, are two members attending as
// OpenCode members the daemon never types at, and a mention of one reaches its
// stream and no other.
func TestHTTPServer_OpenCodeMentionsReachOnlyTheirOwnStream(t *testing.T) {
	c := startOpenCodeRoom(t)
	sa, sb := c.listen(t, tokenA), c.listen(t, tokenB)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusNoContent)
	assert.Equal(t, c.show(t, tokenB, `{"sessionID":"ses_b"}`), http.StatusNoContent)
	waitFor(t, "both agents never attended", func() bool {
		return c.chat.attendCount(tokenA) == 1 && c.chat.attendCount(tokenB) == 1
	})
	for _, token := range []string{tokenA, tokenB} {
		declared := c.chat.declaredBy(token)
		assert.Equal(t, declared.GetHarness(), chatv1.Harness_HARNESS_OPENCODE, token)
		assert.Equal(t, declared.GetNudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE, token)
	}

	bob := member("frontend", "bob", testRoom)
	dave := member("frontend", "dave", testRoom)
	erin := member("frontend", "erin", testRoom)
	c.push(t, tokenA, mentionOf(bob, selfA, "look"))
	assert.DeepEqual(t, sa.next(t), newMessageFrom(bob))
	c.push(t, tokenB, mentionOf(dave, selfB, "look"))
	assert.DeepEqual(t, sb.next(t), newMessageFrom(dave))
	// What comes next on the first stream is its own next mention, not a copy of
	// the one the second member was handed.
	c.push(t, tokenA, mentionOf(erin, selfA, "again"))
	assert.DeepEqual(t, sa.next(t), newMessageFrom(erin))
}

// A mention arriving mid-turn waits, as it does for every channel: the turn
// ending is what delivers it, as a count of what is unread.
func TestHTTPServer_OpenCodeNoticesWaitForTheTurnToEnd(t *testing.T) {
	c := startOpenCodeRoom(t)
	sa := c.listen(t, tokenA)
	waitFor(t, "the agent never attended", func() bool { return c.chat.attendCount(tokenA) == 1 })

	bob := member("frontend", "bob", testRoom)
	c.push(t, tokenA, stateOf(selfA, chatv1.HarnessState_HARNESS_STATE_WORKING))
	c.push(t, tokenA, mentionOf(bob, selfA, "look"))
	c.chat.setUnread(tokenA, &chatv1.Message{From: bob, Text: "look"})
	c.push(t, tokenA, stateOf(selfA, chatv1.HarnessState_HARNESS_STATE_DONE))

	assert.DeepEqual(t, sa.next(t), openCodeLine{Content: nudge.Waiting(1)})
}

// A registration names the member in its header and the session in its body,
// and a member with no notice stream open has nothing to register.
func TestHTTPServer_OpenCodeRegistrationNeedsANoticeStream(t *testing.T) {
	c := startOpenCodeRoom(t)

	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusConflict)
	assert.Equal(t, c.show(t, "", `{"sessionID":"ses_a"}`), http.StatusBadRequest)
	status, _, _ := answer(t, c.request(t, http.MethodGet, "/opencode/notices", "", nil))
	assert.Equal(t, status, http.StatusBadRequest)

	c.listen(t, tokenA)
	assert.Equal(t, c.show(t, tokenA, `{}`), http.StatusBadRequest)
	assert.Equal(t, c.show(t, tokenA, `ses_a`), http.StatusBadRequest)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusNoContent)
}

// sentTool stands in for the chat family's chat_send: it takes the same
// arguments, acts as the member the call names, and keeps the arguments it was
// handed.
func sentTool(t *testing.T, srv *HTTPServer) func() []map[string]any {
	t.Helper()
	type sendArgs struct {
		To      string `json:"to"`
		Message string `json:"message"`
	}
	var mu sync.Mutex
	var seen []map[string]any
	mcpsdk.AddTool(srv.MCP(), &mcpsdk.Tool{Name: "chat_send", Description: "send a message"},
		func(
			ctx context.Context, req *mcpsdk.CallToolRequest, in sendArgs,
		) (*mcpsdk.CallToolResult, any, error) {
			var args map[string]any
			if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
				return nil, nil, err
			}
			mu.Lock()
			seen = append(seen, args)
			mu.Unlock()
			member, err := srv.MemberOf(ctx, req.Session)
			if err != nil {
				return nil, nil, err
			}
			if err := member.AwaitAttendance(ctx); err != nil {
				return nil, nil, err
			}
			target, err := cli.ParseTarget(in.To)
			if err != nil {
				return nil, nil, err
			}
			if _, err := srv.Client().Send(ctx, member.Token(), target, in.Message); err != nil {
				return nil, nil, err
			}
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "sent"}},
			}, nil, nil
		})
	return func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}
}

// send calls chat_send on session with the OpenCode session argument set to
// openCodeSession, or left out when it is empty.
func send(
	t *testing.T, session *mcpsdk.ClientSession, openCodeSession, message string,
) *mcpsdk.CallToolResult {
	t.Helper()
	args := map[string]any{"to": "everyone", "message": message}
	if openCodeSession != "" {
		args[openCodeSessionArg] = openCodeSession
	}
	res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{
		Name: "chat_send", Arguments: args,
	})
	assert.NilError(t, err)
	return res
}

// Every call on the one session OpenCode opened acts as the member whose TUI
// shows the OpenCode session the call names, and the tool never sees the
// argument that named it.
func TestHTTPServer_OpenCodeCallsActAsTheMemberShowingTheirSession(t *testing.T) {
	c := startOpenCodeRoom(t)
	seen := sentTool(t, c.srv)
	c.listen(t, tokenA)
	c.listen(t, tokenB)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusNoContent)
	assert.Equal(t, c.show(t, tokenB, `{"sessionID":"ses_b"}`), http.StatusNoContent)
	shared := connectHTTPAs(t, c.endpoint, "", "opencode")

	for _, call := range []struct{ session, message string }{
		{"ses_a", "from alice"},
		{"ses_b", "from carol"},
	} {
		res := send(t, shared, call.session, call.message)
		assert.Assert(t, !res.IsError, "the tool answered %q", toolText(t, res))
	}

	assert.DeepEqual(t, c.chat.sentSoFar(), []sentBy{
		{Token: tokenA, Text: "from alice"},
		{Token: tokenB, Text: "from carol"},
	})
	assert.DeepEqual(t, seen(), []map[string]any{
		{"to": "everyone", "message": "from alice"},
		{"to": "everyone", "message": "from carol"},
	})
}

// A call naming an OpenCode session no TUI shows — a subagent's, or one the TUI
// has moved away from — acts as nobody, and so does a call naming none. The
// token the session itself carries is never who it acts as instead.
func TestHTTPServer_OpenCodeCallsNamingNoShownSessionActAsNobody(t *testing.T) {
	c := startOpenCodeRoom(t)
	sentTool(t, c.srv)
	c.listen(t, tokenA)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusNoContent)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a2"}`), http.StatusNoContent)
	shared := connectHTTPAs(t, c.endpoint, tokenA, "opencode")

	for _, session := range []string{"ses_child", "ses_a"} {
		res := send(t, shared, session, "hello")
		assert.Assert(t, res.IsError)
		assert.Assert(t, strings.Contains(toolText(t, res), "has no crabswarm identity"),
			"the tool answered %q", toolText(t, res))
	}
	res := send(t, shared, "", "hello")
	assert.Assert(t, res.IsError)
	assert.Equal(t, toolText(t, res), errNoOpenCodeSession.Error())

	assert.Equal(t, len(c.chat.sentSoFar()), 0)
}

// The server's plugin reads for the member showing a session and gets what
// `crabswarm chat read --quiet` prints for that member: the unread, then
// nothing.
func TestHTTPServer_OpenCodeReadsForTheSessionsMember(t *testing.T) {
	c := startOpenCodeRoom(t)
	c.listen(t, tokenA)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusNoContent)
	waitFor(t, "the agent never attended", func() bool { return c.chat.attendCount(tokenA) == 1 })

	unread := []*chatv1.Message{{
		Seq:  7,
		From: member("frontend", "bob", testRoom),
		Target: &chatv1.Target{Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{
			Roles: []*chatv1.MemberTarget{{Team: "backend", Name: "alice"}},
		}}},
		Text: "rebase please",
	}}
	c.chat.setUnread(tokenA, unread...)
	var want bytes.Buffer
	assert.NilError(t, cli.RenderRead(&want, &chatv1.ReadResponse{Messages: unread}))

	status, contentType, body := c.read(t, "ses_a")
	assert.Equal(t, status, http.StatusOK)
	assert.Equal(t, contentType, "text/plain; charset=utf-8")
	assert.Equal(t, body, want.String())

	status, _, body = c.read(t, "ses_a")
	assert.Equal(t, status, http.StatusNoContent)
	assert.Equal(t, body, "")

	status, _, _ = c.read(t, "ses_unknown")
	assert.Equal(t, status, http.StatusNotFound)
}

// A member attends while any of its notice streams is open. The last one
// closing takes it out of the room, and the session its TUI showed with it.
func TestHTTPServer_OpenCodeMemberLeavesWithItsLastStream(t *testing.T) {
	c := startOpenCodeRoom(t)
	first, second := c.listen(t, tokenA), c.listen(t, tokenA)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusNoContent)
	waitFor(t, "the agent never attended", func() bool { return c.chat.attendCount(tokenA) == 1 })
	assert.Equal(t, streamsOf(c.srv, tokenA), 2)

	first.close(t)
	waitFor(t, "the closed stream still keeps the member", func() bool {
		return streamsOf(c.srv, tokenA) == 1
	})
	assert.Equal(t, c.chat.cancelsOf(tokenA), 0)
	status, _, _ := c.read(t, "ses_a")
	assert.Equal(t, status, http.StatusNoContent)

	second.close(t)
	waitFor(t, "the member outlived its last stream", func() bool {
		return c.chat.cancelsOf(tokenA) == 1
	})
	assert.Equal(t, sessionsOf(c.srv, tokenA), 0)
	status, _, _ = c.read(t, "ses_a")
	assert.Equal(t, status, http.StatusNotFound)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusConflict)
}

// sendsAs asserts that a call naming openCodeSession acts as the member token
// names: the message goes out with that token.
func sendsAs(
	t *testing.T,
	c openCodeRoom,
	shared *mcpsdk.ClientSession,
	openCodeSession, token string,
) {
	t.Helper()
	before := len(c.chat.sentSoFar())
	res := send(t, shared, openCodeSession, "from "+openCodeSession)
	assert.Assert(t, !res.IsError, "the tool answered %q", toolText(t, res))
	sent := c.chat.sentSoFar()
	assert.Equal(t, len(sent), before+1)
	assert.Equal(t, sent[before].Token, token, "a call from %s", openCodeSession)
}

// readsAs asserts that the server's plugin reading for openCodeSession reads
// what the member token names has unread.
func readsAs(t *testing.T, c openCodeRoom, openCodeSession, token string) {
	t.Helper()
	text := "for " + token
	c.chat.setUnread(token, &chatv1.Message{
		Seq: 1, From: member("frontend", "bob", testRoom), Text: text,
	})
	status, _, body := c.read(t, openCodeSession)
	assert.Equal(t, status, http.StatusOK)
	assert.Assert(
		t,
		strings.Contains(body, text),
		"the read of %s answered %q",
		openCodeSession,
		body,
	)
}

// A TUI moving to a session another TUI showed first takes nothing over. Its
// registration is answered 202, the calls from the session and the reads for it
// stay the first TUI's member's, and registering again changes neither. The
// first TUI registering the session again keeps it.
func TestHTTPServer_OpenCodeASessionStaysWithTheTUIThatShowedItFirst(t *testing.T) {
	c := startOpenCodeRoom(t)
	sentTool(t, c.srv)
	c.listen(t, tokenA)
	c.listen(t, tokenB)
	waitFor(t, "both agents never attended", func() bool {
		return c.chat.attendCount(tokenA) == 1 && c.chat.attendCount(tokenB) == 1
	})
	shared := connectHTTPAs(t, c.endpoint, "", "opencode")

	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_x"}`), http.StatusNoContent)
	status, contentType, body := answer(t, c.request(t, http.MethodPut, "/opencode/session", tokenB,
		strings.NewReader(`{"sessionID":"ses_x"}`)))
	assert.Equal(t, status, http.StatusAccepted)
	assert.Equal(t, contentType, "text/plain; charset=utf-8")
	assert.Assert(t, strings.Contains(body, cli.Address(selfA)), "the 202 answered %q", body)
	sendsAs(t, c, shared, "ses_x", tokenA)
	readsAs(t, c, "ses_x", tokenA)

	assert.Equal(t, c.show(t, tokenB, `{"sessionID":"ses_x"}`), http.StatusAccepted)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_x"}`), http.StatusNoContent)
	sendsAs(t, c, shared, "ses_x", tokenA)

	assert.Equal(t, c.show(t, tokenB, `{"sessionID":"ses_b"}`), http.StatusNoContent)
	sendsAs(t, c, shared, "ses_x", tokenA)
	sendsAs(t, c, shared, "ses_b", tokenB)
}

// A session whose owner moves to another one passes to the TUI that showed it
// next, which then registers it as its own. The first TUI coming back to it is
// the one that takes nothing over.
func TestHTTPServer_OpenCodeASessionPassesOnWhenItsOwnerMovesAway(t *testing.T) {
	c := startOpenCodeRoom(t)
	sentTool(t, c.srv)
	c.listen(t, tokenA)
	c.listen(t, tokenB)
	waitFor(t, "both agents never attended", func() bool {
		return c.chat.attendCount(tokenA) == 1 && c.chat.attendCount(tokenB) == 1
	})
	shared := connectHTTPAs(t, c.endpoint, "", "opencode")

	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_x"}`), http.StatusNoContent)
	assert.Equal(t, c.show(t, tokenB, `{"sessionID":"ses_x"}`), http.StatusAccepted)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_a"}`), http.StatusNoContent)

	sendsAs(t, c, shared, "ses_x", tokenB)
	readsAs(t, c, "ses_x", tokenB)
	assert.Equal(t, c.show(t, tokenB, `{"sessionID":"ses_x"}`), http.StatusNoContent)
	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_x"}`), http.StatusAccepted)
	sendsAs(t, c, shared, "ses_x", tokenB)
}

// A session passes on the same way when its owner's last notice stream closes.
func TestHTTPServer_OpenCodeASessionPassesOnWhenItsOwnersLastStreamCloses(t *testing.T) {
	c := startOpenCodeRoom(t)
	sentTool(t, c.srv)
	sa := c.listen(t, tokenA)
	c.listen(t, tokenB)
	waitFor(t, "both agents never attended", func() bool {
		return c.chat.attendCount(tokenA) == 1 && c.chat.attendCount(tokenB) == 1
	})
	shared := connectHTTPAs(t, c.endpoint, "", "opencode")

	assert.Equal(t, c.show(t, tokenA, `{"sessionID":"ses_x"}`), http.StatusNoContent)
	assert.Equal(t, c.show(t, tokenB, `{"sessionID":"ses_x"}`), http.StatusAccepted)

	sa.close(t)
	waitFor(t, "the closed stream still keeps the member", func() bool {
		return streamsOf(c.srv, tokenA) == 0
	})
	sendsAs(t, c, shared, "ses_x", tokenB)
	readsAs(t, c, "ses_x", tokenB)
}

// A TUI's plugin may reach the listener from another network namespace, under
// a host name that is not loopback. The MCP handler refuses such a request; the
// routes beside it answer it.
func TestHTTPServer_OpenCodeRoutesAnswerAnyHostName(t *testing.T) {
	c := startOpenCodeRoom(t)
	const elsewhere = "tui.internal:47300"
	under := func(req *http.Request) *http.Request {
		req.Host = elsewhere
		return req
	}

	status, _, _ := answer(t, under(c.request(t, http.MethodPost, HTTPPath, tokenA,
		strings.NewReader(`{}`))))
	assert.Equal(t, status, http.StatusForbidden, "the MCP handler took a foreign host name")

	resp, err := http.DefaultClient.Do(under(c.request(t, http.MethodGet, "/opencode/notices",
		tokenA, nil)))
	assert.NilError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	status, _, _ = answer(t, under(c.request(t, http.MethodPut, "/opencode/session", tokenA,
		strings.NewReader(`{"sessionID":"ses_a"}`))))
	assert.Equal(t, status, http.StatusNoContent)
	status, _, _ = answer(t, under(c.request(t, http.MethodPost,
		"/opencode/sessions/ses_a/read", "", nil)))
	assert.Equal(t, status, http.StatusNoContent)
}

// A stdio server takes the OpenCode session argument off a call too: an
// OpenCode running the plugin adds it to every call, and no tool's schema
// takes it.
func TestServer_TakesTheOpenCodeSessionOffACall(t *testing.T) {
	bridge := newTestBridge(t, &fakeChatService{self: doneSelf("backend", "alice", testRoom)})
	reached := countedTool(bridge.MCP(), "plain")
	session := serveBridgeAs(t, bridge, "opencode")

	res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{
		Name: "plain", Arguments: map[string]any{openCodeSessionArg: "ses_a"},
	})
	assert.NilError(t, err)
	assert.Equal(t, toolText(t, res), "reached")
	assert.Equal(t, reached.Load(), int64(1))
}
