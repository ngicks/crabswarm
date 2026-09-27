package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// One Codex app server shared by several agents, each an MCP session per
// thread on one HTTP server. The fake app server below stands in for Codex,
// and forwards the tool calls it is asked to make into the MCP session of the
// thread they name, the way the real one does. pkg/harnessctl replays the
// recorded app server against a fake of its own, which a test file there
// cannot share; this is the small second copy, answering only the calls the
// harness makes.

// Three threads as Codex names them: one each for two agents, and a second
// thread of the first.
const (
	threadA  = "01a0e508-d67b-7a83-beea-9bbe77546150"
	threadB  = "01a0e508-f1f5-74d0-9e01-cdc8b500a958"
	threadA2 = "01a0e509-39ae-7be3-981a-74678ea1f8d9"
)

// The two agents' tokens.
const (
	tokenA = "tok-a"
	tokenB = "tok-b"
)

// codexApp is a fake app server on a unix socket.
type codexApp struct {
	path string

	mu     sync.Mutex
	loaded []string
	// sessions is the MCP session the app server opened for each thread, which
	// a tool call on that thread is forwarded into.
	sessions map[string]*mcpsdk.ClientSession
	// refuses has the app server answer every tool call as its MCP server
	// refusing it, as an app server whose threads run another MCP server does.
	refuses bool
	turns   []string
	resumed []string
	conns   []*websocket.Conn
	opened  int

	forwarding errgroup.Group
}

func startCodexApp(t *testing.T) *codexApp {
	t.Helper()

	// Short for the reason pkg/harnessctl's fake gives: a unix socket path is
	// capped well below what a directory named after a test comes to.
	dir, err := os.MkdirTemp("", "cx")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	app := &codexApp{
		path:     filepath.Join(dir, "app.sock"),
		sessions: map[string]*mcpsdk.ClientSession{},
	}
	ln, err := net.Listen("unix", app.path)
	assert.NilError(t, err)
	srv := &http.Server{Handler: http.HandlerFunc(app.serve), ReadHeaderTimeout: 10 * time.Second}
	var serving errgroup.Group
	serving.Go(func() error { return srv.Serve(ln) })
	t.Cleanup(func() {
		_ = srv.Close()
		_ = serving.Wait()
		_ = app.forwarding.Wait()
	})
	return app
}

// environ is the environment of a server launched beside the app server.
func (app *codexApp) environ(name string) string {
	if name == harnessctl.CodexAppServerEnv {
		return "unix://" + app.path
	}
	return ""
}

func (app *codexApp) serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(1 << 20)
	app.mu.Lock()
	app.conns = append(app.conns, conn)
	app.opened++
	app.mu.Unlock()
	defer func() {
		app.mu.Lock()
		app.conns = slices.DeleteFunc(app.conns, func(c *websocket.Conn) bool { return c == conn })
		app.mu.Unlock()
	}()
	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			return
		}
		app.answer(conn, data)
	}
}

// answer replies to one call, with no version marker, as the real app server
// does. A tool call is forwarded on a goroutine of its own, since the MCP
// server it reaches may ask this connection something before it answers.
func (app *codexApp) answer(conn *websocket.Conn, data []byte) {
	var frame struct {
		Id     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			ThreadId  string         `json:"threadId"`
			Tool      string         `json:"tool"`
			Arguments map[string]any `json:"arguments"`
		} `json:"params"`
	}
	if json.Unmarshal(data, &frame) != nil || len(frame.Id) == 0 {
		return
	}
	thread := frame.Params.ThreadId
	reply := func(result string) {
		_ = conn.Write(context.Background(), websocket.MessageText,
			fmt.Appendf(nil, `{"id":%s,"result":%s}`, frame.Id, result))
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	switch frame.Method {
	case "thread/loaded/list":
		ids, _ := json.Marshal(app.loaded)
		reply(`{"data":` + string(ids) + `,"nextCursor":null}`)
	case "thread/read":
		reply(fmt.Sprintf(`{"thread":{"id":%q,"threadSource":"user","recencyAt":0}}`, thread))
	case "thread/resume":
		app.resumed = append(app.resumed, thread)
		reply(fmt.Sprintf(`{"thread":{"id":%q,"status":{"type":"idle"}}}`, thread))
	case "thread/unsubscribe":
		reply(`{"status":"unsubscribed"}`)
	case "turn/start":
		app.turns = append(app.turns, thread)
		reply(`{"turn":{"id":"turn","status":"inProgress"}}`)
	case "mcpServer/tool/call":
		session := app.sessions[thread]
		if app.refuses {
			session = nil
		}
		app.forwarding.Go(func() error {
			result, err := forwardToolCall(
				session,
				thread,
				frame.Params.Tool,
				frame.Params.Arguments,
			)
			if err != nil {
				_ = conn.Write(context.Background(), websocket.MessageText,
					fmt.Appendf(nil, `{"id":%s,"error":{"code":-32602,"message":%q}}`,
						frame.Id, err.Error()))
				return nil
			}
			reply(result)
			return nil
		})
	default:
		reply(`{}`)
	}
}

// forwardToolCall makes one tool call on session with the thread in its
// metadata, as the app server does, and answers with what the session said.
func forwardToolCall(
	session *mcpsdk.ClientSession, thread, tool string, args map[string]any,
) (string, error) {
	if session == nil {
		return "", fmt.Errorf("unknown tool %q", tool)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Meta:      mcpsdk.Meta{"threadId": thread},
		Name:      tool,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}
	out, err := json.Marshal(res)
	return string(out), err
}

// open is the app server loading thread and opening session as its MCP
// session, and announcing the thread the way it does when a TUI starts one.
func (app *codexApp) open(thread string, session *mcpsdk.ClientSession) {
	app.mu.Lock()
	app.loaded = append(app.loaded, thread)
	app.sessions[thread] = session
	app.mu.Unlock()
	app.push(fmt.Sprintf(`{"method":"thread/started","params":{"thread":{"id":%q}}}`, thread))
}

// focus is what a proxy in front of a TUI does after the TUI resumes thread.
func (app *codexApp) focus(t *testing.T, thread string) {
	t.Helper()
	app.mu.Lock()
	session := app.sessions[thread]
	app.mu.Unlock()
	_, err := forwardToolCall(session, thread, focusTool, nil)
	assert.NilError(t, err)
}

// push writes one notification to every connection.
func (app *codexApp) push(frame string) {
	app.mu.Lock()
	conns := slices.Clone(app.conns)
	app.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Write(context.Background(), websocket.MessageText, []byte(frame))
	}
}

// pushStatus says what thread is doing now.
func (app *codexApp) pushStatus(thread, status string) {
	app.push(fmt.Sprintf(`{"method":"thread/status/changed","params":{"threadId":%q,"status":%s}}`,
		thread, status))
}

// pushTurnStarted says a turn started on thread.
func (app *codexApp) pushTurnStarted(thread string) {
	app.push(fmt.Sprintf(`{"method":"turn/started","params":{"threadId":%q,"turn":{"id":"turn"}}}`,
		thread))
}

func (app *codexApp) startedTurns() []string {
	app.mu.Lock()
	defer app.mu.Unlock()
	return slices.Clone(app.turns)
}

func (app *codexApp) resumedThreads() []string {
	app.mu.Lock()
	defer app.mu.Unlock()
	return slices.Clone(app.resumed)
}

func (app *codexApp) connections() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.opened
}

// waitTurns blocks until the app server has started turns on exactly want,
// oldest first.
func waitTurns(t *testing.T, app *codexApp, want ...string) {
	t.Helper()
	var got []string
	deadline := time.Now().Add(eventTimeout)
	for time.Now().Before(deadline) {
		got = app.startedTurns()
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("turns were started on %v, want %v", got, want)
}

// roomChat is a daemon with a member per token, each attendance fed from a feed
// of its own, so a case can mention one agent and not the other.
type roomChat struct {
	chatv1.UnimplementedChatServiceServer

	selves map[string]*chatv1.Member

	mu       sync.Mutex
	feeds    map[string]chan *chatv1.RoomEvent
	attends  map[string]int
	reported map[string][]chatv1.HarnessState
}

func newRoomChat(selves map[string]*chatv1.Member) *roomChat {
	return &roomChat{
		selves:   selves,
		feeds:    map[string]chan *chatv1.RoomEvent{},
		attends:  map[string]int{},
		reported: map[string][]chatv1.HarnessState{},
	}
}

// tokenOf is the token a call carried.
func tokenOf(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	if v := md.Get(chat.TokenMetadataKey); len(v) > 0 {
		return v[0]
	}
	return ""
}

func (r *roomChat) feed(token string) chan *chatv1.RoomEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.feeds[token] == nil {
		r.feeds[token] = make(chan *chatv1.RoomEvent)
	}
	return r.feeds[token]
}

func (r *roomChat) Attend(
	_ *chatv1.AttendRequest, stream grpc.ServerStreamingServer[chatv1.RoomEvent],
) error {
	token := tokenOf(stream.Context())
	self := r.selves[token]
	if self == nil {
		return status.Error(codes.Unauthenticated, "unknown identity token")
	}
	feed := r.feed(token)
	r.mu.Lock()
	r.attends[token]++
	r.mu.Unlock()
	for _, ev := range []*chatv1.RoomEvent{
		{Event: &chatv1.RoomEvent_Attended{Attended: &chatv1.Attended{Self: self}}},
		{Event: &chatv1.RoomEvent_MemberJoined{MemberJoined: &chatv1.MemberJoined{Member: self}}},
	} {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case ev := <-feed:
			if err := stream.Send(ev); err != nil {
				return err
			}
		}
	}
}

func (r *roomChat) ReportState(
	ctx context.Context, req *chatv1.ReportStateRequest,
) (*chatv1.ReportStateResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	token := tokenOf(ctx)
	r.reported[token] = append(r.reported[token], req.GetState())
	return &chatv1.ReportStateResponse{}, nil
}

func (r *roomChat) CountUnread(
	context.Context, *chatv1.CountUnreadRequest,
) (*chatv1.CountUnreadResponse, error) {
	return &chatv1.CountUnreadResponse{}, nil
}

func (r *roomChat) attendCount(token string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.attends[token]
}

func (r *roomChat) reportedBy(token string) []chatv1.HarnessState {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.reported[token])
}

// mention hands the member token names a mention on its own feed.
func (r *roomChat) mention(t *testing.T, token string) {
	t.Helper()
	select {
	case r.feed(token) <- mentionOf(member("frontend", "bob", testRoom), r.selves[token], "look"):
	case <-time.After(eventTimeout):
		t.Fatalf("%s is not attending", token)
	}
}

// codexRoom is the two agents' room, an app server, and an HTTP server launched
// beside that app server, running.
type codexRoom struct {
	room     *roomChat
	app      *codexApp
	srv      *HTTPServer
	endpoint string
}

func startCodexRoom(t *testing.T) codexRoom {
	t.Helper()
	room := newRoomChat(map[string]*chatv1.Member{
		tokenA: doneSelf("backend", "alice", testRoom),
		tokenB: doneSelf("infra", "carol", testRoom),
	})
	app := startCodexApp(t)
	srv, err := NewHTTP(slog.New(slog.DiscardHandler), serveTestDaemon(t, room), app.environ)
	assert.NilError(t, err)
	return codexRoom{room: room, app: app, srv: srv, endpoint: serveHTTP(t, srv).endpoint}
}

// open is an agent's TUI starting thread: the app server opens an MCP session
// for it, which names token, and loads the thread.
func (c codexRoom) open(t *testing.T, token, thread string) *mcpsdk.ClientSession {
	t.Helper()
	session := connectHTTPAs(t, c.endpoint, token, "codex-mcp-client")
	c.app.open(thread, session)
	return session
}

// threadOf is the thread the server bound session to, "" for none.
func (c codexRoom) threadOf(session *mcpsdk.ClientSession) string {
	c.srv.mu.Lock()
	defer c.srv.mu.Unlock()
	for s, b := range c.srv.sessions {
		if s.ID() == session.ID() {
			return b.thread
		}
	}
	return ""
}

// waitBound blocks until the server has bound session to thread.
func (c codexRoom) waitBound(t *testing.T, session *mcpsdk.ClientSession, thread string) {
	t.Helper()
	waitFor(t, "the session was never bound to "+thread, func() bool {
		return c.threadOf(session) == thread
	})
}

// Two agents on one app server, neither of whose sessions has called a tool.
// Each session is bound by the ping that reaches it, and a mention becomes a
// turn on the thread of the agent it names, over the one connection both
// agents share.
func TestHTTPServer_CodexMentionsReachTheirOwnAgentsThread(t *testing.T) {
	c := startCodexRoom(t)
	sa := c.open(t, tokenA, threadA)
	sb := c.open(t, tokenB, threadB)
	c.waitBound(t, sa, threadA)
	c.waitBound(t, sb, threadB)
	waitFor(t, "both agents never attended", func() bool {
		return c.room.attendCount(tokenA) == 1 && c.room.attendCount(tokenB) == 1
	})
	waitFor(t, "the feeds never followed both threads", func() bool {
		resumed := c.app.resumedThreads()
		return slices.Contains(resumed, threadA) && slices.Contains(resumed, threadB)
	})

	c.room.mention(t, tokenA)
	waitTurns(t, c.app, threadA)
	c.room.mention(t, tokenB)
	waitTurns(t, c.app, threadA, threadB)
	assert.Equal(t, c.app.connections(), 1, "the agents did not share one connection")
}

// Each agent's state is what its own thread says, and nothing another agent's
// thread says.
func TestHTTPServer_CodexStateFollowsTheAgentsOwnThread(t *testing.T) {
	c := startCodexRoom(t)
	sa := c.open(t, tokenA, threadA)
	sb := c.open(t, tokenB, threadB)
	c.waitBound(t, sa, threadA)
	c.waitBound(t, sb, threadB)
	done := []chatv1.HarnessState{chatv1.HarnessState_HARNESS_STATE_DONE}
	waitFor(t, "the feeds never said what the threads were doing", func() bool {
		return slices.Equal(c.room.reportedBy(tokenA), done) &&
			slices.Equal(c.room.reportedBy(tokenB), done)
	})

	c.app.pushStatus(threadB, `{"type":"active","activeFlags":[]}`)
	waitFor(t, "B never reported its thread working", func() bool {
		return slices.Equal(c.room.reportedBy(tokenB), append(done,
			chatv1.HarnessState_HARNESS_STATE_WORKING))
	})
	c.app.pushStatus(threadA, `{"type":"active","activeFlags":["waitingOnApproval"]}`)
	waitFor(t, "A never reported its thread waiting", func() bool {
		return slices.Equal(c.room.reportedBy(tokenA), append(done,
			chatv1.HarnessState_HARNESS_STATE_WAITING))
	})
	assert.DeepEqual(t, c.room.reportedBy(tokenB), append(done,
		chatv1.HarnessState_HARNESS_STATE_WORKING))
}

// An agent is reached in the session it was last seen in: a new session the
// moment it opens, an earlier one once a focus call arrives on it, and any of
// them once a turn starts on its thread. The other agent is reached where it
// was all along.
func TestHTTPServer_CodexMentionsFollowWhereTheAgentWasLastSeen(t *testing.T) {
	c := startCodexRoom(t)
	sa := c.open(t, tokenA, threadA)
	sb := c.open(t, tokenB, threadB)
	c.waitBound(t, sa, threadA)
	c.waitBound(t, sb, threadB)
	waitFor(t, "A never attended", func() bool { return c.room.attendCount(tokenA) == 1 })

	// A new thread's session has called nothing, and takes the next mention.
	sa2 := c.open(t, tokenA, threadA2)
	c.room.mention(t, tokenA)
	waitTurns(t, c.app, threadA2)
	assert.Equal(t, c.threadOf(sa2), threadA2)

	// The TUI resumed the first thread, and the proxy in front of it said so.
	c.app.focus(t, threadA)
	c.room.mention(t, tokenA)
	waitTurns(t, c.app, threadA2, threadA)

	// A turn started on the second thread, from its TUI.
	c.app.pushTurnStarted(threadA2)
	waitFor(t, "the turn start was never recorded", func() bool {
		c.srv.mu.Lock()
		defer c.srv.mu.Unlock()
		return c.srv.turns[threadA2] > 0
	})
	c.room.mention(t, tokenA)
	waitTurns(t, c.app, threadA2, threadA, threadA2)

	c.room.mention(t, tokenB)
	waitTurns(t, c.app, threadA2, threadA, threadA2, threadB)
	// One member however many sessions it has.
	assert.Equal(t, c.room.attendCount(tokenA), 1)
}

// One agent alone on an app server whose threads run a server that answers no
// ping still gets its mentions: the one person's thread the app server holds
// is the only thread it can be in.
func TestHTTPServer_CodexSingleAgentWithoutPings(t *testing.T) {
	c := startCodexRoom(t)
	c.app.refuses = true
	sa := c.open(t, tokenA, threadA)
	waitFor(t, "A never attended", func() bool { return c.room.attendCount(tokenA) == 1 })

	c.room.mention(t, tokenA)
	waitTurns(t, c.app, threadA)
	assert.Equal(t, c.threadOf(sa), "")
}

// countedTool registers a tool under name that counts the calls reaching it.
func countedTool(server *mcpsdk.Server, name string) *atomic.Int64 {
	var calls atomic.Int64
	mcpsdk.AddTool(
		server,
		&mcpsdk.Tool{Name: name, Description: "counts its calls"},
		func(context.Context, *mcpsdk.CallToolRequest, struct{}) (*mcpsdk.CallToolResult, any, error) {
			calls.Add(1)
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{&mcpsdk.TextContent{Text: "reached"}},
			}, nil, nil
		},
	)
	return &calls
}

// assertAnsweredItself calls name on session and asserts the server answered
// it with a success that carries nothing.
func assertAnsweredItself(
	t *testing.T, session *mcpsdk.ClientSession, name string, args map[string]any,
) {
	t.Helper()
	res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{
		Meta: mcpsdk.Meta{"threadId": threadA}, Name: name, Arguments: args,
	})
	assert.NilError(t, err)
	assert.Assert(t, !res.IsError, "%s answered an error", name)
	assert.Equal(t, len(res.Content), 0, "%s answered %v", name, res.Content)
}

// A reserved call is the server's own, answered with an empty success whether
// or not a tool of that name exists: none reaches a tool, so none reaches the
// model. A tool call of any other name still does.
func TestServer_AnswersReservedCallsItself(t *testing.T) {
	assertReserved := func(
		t *testing.T, server *mcpsdk.Server, connect func() *mcpsdk.ClientSession,
	) {
		t.Helper()
		reserved, plain := countedTool(server, pingTool), countedTool(server, "plain")
		session := connect()

		assertAnsweredItself(t, session, focusTool, nil)
		assertAnsweredItself(t, session, pingTool, map[string]any{pingNonceArg: "n"})
		assertAnsweredItself(t, session, reservedToolPrefix+"anything", nil)
		assert.Equal(t, reserved.Load(), int64(0), "a reserved call reached a tool")

		res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "plain"})
		assert.NilError(t, err)
		assert.Equal(t, toolText(t, res), "reached")
		assert.Equal(t, plain.Load(), int64(1))
	}

	t.Run("stdio", func(t *testing.T) {
		bridge := newTestBridge(t, &fakeChatService{self: doneSelf("backend", "alice", testRoom)})
		assertReserved(
			t,
			bridge.MCP(),
			func() *mcpsdk.ClientSession { return serveBridge(t, bridge) },
		)
	})
	t.Run("http", func(t *testing.T) {
		srv := newTestHTTP(t, &fakeChatService{self: doneSelf("backend", "alice", testRoom)})
		assertReserved(t, srv.MCP(), func() *mcpsdk.ClientSession {
			return connectHTTP(t, serveHTTP(t, srv).endpoint, testToken)
		})
	})
}

// A session learns its thread from the metadata of the first tool call a turn
// made on it, and keeps it; a reserved call addressed to a thread overrides
// it, since the app server routed that call to this session by its thread.
func TestHTTPServer_ASessionLearnsItsThreadFromItsCalls(t *testing.T) {
	c := startCodexRoom(t)
	countedTool(c.srv.MCP(), "plain")
	session := connectHTTPAs(t, c.endpoint, tokenA, "codex-mcp-client")
	call := func(name, thread string) {
		_, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{
			Meta: mcpsdk.Meta{"threadId": thread}, Name: name,
		})
		assert.NilError(t, err)
	}

	call("plain", threadA)
	assert.Equal(t, c.threadOf(session), threadA)
	call("plain", threadB)
	assert.Equal(t, c.threadOf(session), threadA)
	call(focusTool, threadA2)
	assert.Equal(t, c.threadOf(session), threadA2)
}

// feedHarness is a native harness with a feed of its own, which counts the
// watches running on it, so a case sees which session's harness is watched.
type feedHarness struct {
	kind    chatv1.Harness
	nudge   chatv1.NudgeDelivery
	states  chan chatv1.HarnessState
	mu      sync.Mutex
	running int
}

func newFeedHarness(nudge chatv1.NudgeDelivery) *feedHarness {
	return &feedHarness{
		kind:   chatv1.Harness_HARNESS_OTHER,
		nudge:  nudge,
		states: make(chan chatv1.HarnessState),
	}
}

func (h *feedHarness) Kind() chatv1.Harness                             { return h.kind }
func (h *feedHarness) Nudge() chatv1.NudgeDelivery                      { return h.nudge }
func (h *feedHarness) Deliver(context.Context, harnessctl.Notice) error { return nil }

func (h *feedHarness) Watch(ctx context.Context, report func(chatv1.HarnessState)) {
	h.mu.Lock()
	h.running++
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		h.running--
		h.mu.Unlock()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case state := <-h.states:
			report(state)
		}
	}
}

func (h *feedHarness) watched() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

// The state a member reports is its current harness's, and the attendance
// declares what that harness is: a member seen in a session of a harness that
// declares something else attends again to say so, and the watch moves with
// it.
func TestHTTPServer_FollowsTheHarnessTheMemberWasLastSeenIn(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	srv.host.pace.attendBackoffBase = 10 * time.Millisecond
	srv.host.pace.attendBackoffMax = 20 * time.Millisecond
	native := newFeedHarness(chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
	typed := newFeedHarness(chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
	byClient := map[string]harnessctl.Harness{"native": native, "typed": typed}
	srv.host.detect = func(client string, _ func(string) string) harnessctl.Harness {
		return byClient[client]
	}
	endpoint := serveHTTP(t, srv).endpoint

	connectHTTPAs(t, endpoint, testToken, "native")
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetNudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
	waitFor(t, "the native harness was never watched", func() bool { return native.watched() == 1 })

	second := connectHTTPAs(t, endpoint, testToken, "typed")
	waitFor(t, "the member never declared the harness it was last seen in", func() bool {
		return fake.attendCount() == 2 &&
			fake.lastAttend().GetNudge() == chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL
	})
	waitFor(t, "the watch never moved to the new harness", func() bool {
		return native.watched() == 0 && typed.watched() == 1
	})
	select {
	case typed.states <- chatv1.HarnessState_HARNESS_STATE_WORKING:
	case <-time.After(eventTimeout):
		t.Fatal("nothing is watching the new harness")
	}
	waitReported(t, fake, chatv1.HarnessState_HARNESS_STATE_WORKING)

	// The session it was last seen in closed, which leaves the first one.
	assert.NilError(t, second.Close())
	waitFor(t, "the member never declared the harness it is back in", func() bool {
		return fake.attendCount() == 3 &&
			fake.lastAttend().GetNudge() == chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
	})
	waitFor(t, "the watch never moved back", func() bool {
		return native.watched() == 1 && typed.watched() == 0
	})
}
