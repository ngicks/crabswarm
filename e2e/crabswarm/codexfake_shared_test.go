package crabswarm_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
)

// Several remote Codex TUIs can share one `codex app-server`, and the app
// server runs the MCP servers of every thread itself, so what tells one agent's
// threads from another's is the header each thread's MCP session sends. The
// fake below stands in for such an app server.
//
// Unlike the fake in chat_codex_test.go, which holds the one session thread it
// was given, this one hosts the threads its clients start. For each thread it
// opens a streamable-HTTP MCP session to the crabswarm-mcp server it declares,
// sending every mcp_servers.crabswarm-mcp.http_headers.* key of the thread's
// config as a request header, and it forwards an mcpServer/tool/call into the
// session of the thread the call names, with that thread in the call's
// metadata. testdata/harness/README.md records both from a real app server:
// codex-mcp-http-thread-header.jsonl and the tool-call recordings.
//
// It runs no turns. A turn/start is recorded and answered, and nothing is
// pushed about the turn afterwards, so every member stays done and every
// mention is delivered the moment it arrives.

// codexSharedServer is the name the app server declares crabswarm's MCP server
// under, which is the name the proxy stamps a thread's header for.
const codexSharedServer = "crabswarm-mcp"

// codexSharedHeaderKey begins every per-thread config key that sets a request
// header on the sessions of that server.
const codexSharedHeaderKey = "mcp_servers." + codexSharedServer + ".http_headers."

// codexSharedTimeout bounds every wait of the shared-app-server cases and of
// the processes they run.
const codexSharedTimeout = 30 * time.Second

// codexSharedApp is a fake app server on a unix socket, shared by every client
// that dials it.
type codexSharedApp struct {
	path string
	// endpoint is the URL the app server declares its crabswarm-mcp server at.
	endpoint string

	mu sync.Mutex
	// closed is set once the test is over. Nothing is forwarded after it.
	closed bool
	// threads are the loaded threads, oldest first.
	threads []*codexSharedThread
	// numbered counts the threads started so far, which numbers the next id.
	numbered int
	conns    []*websocket.Conn
	turns    []codexSharedTurn
	calls    []codexSharedCall
	// forwarding holds the tool calls on their way into a thread's session.
	forwarding errgroup.Group
}

// codexSharedThread is one loaded thread.
type codexSharedThread struct {
	id string
	// owner is the client connection that started the thread.
	owner *websocket.Conn
	// session is the MCP session the app server opened for the thread's
	// crabswarm-mcp server.
	session *mcp.ClientSession
}

// codexSharedTurn is one turn the fake was asked to start: the thread it was
// started on and the text it was handed.
type codexSharedTurn struct {
	thread, text string
}

// codexSharedCall is one tool call the fake forwarded and the session of its
// thread answered.
type codexSharedCall struct {
	thread, tool string
}

// startCodexSharedApp listens on a socket of its own until the test ends, and
// declares the crabswarm-mcp server of every thread at endpoint.
func startCodexSharedApp(t *testing.T, endpoint string) *codexSharedApp {
	t.Helper()

	app := &codexSharedApp{
		path:     filepath.Join(shortSocketDir(t), "app.sock"),
		endpoint: endpoint,
	}
	ln, err := net.Listen("unix", app.path)
	if err != nil {
		t.Fatalf("listen on %s: %v", app.path, err)
	}
	srv := &http.Server{
		Handler:           http.HandlerFunc(app.serve),
		ReadHeaderTimeout: 10 * time.Second,
	}
	var serving errgroup.Group
	serving.Go(func() error { return srv.Serve(ln) })
	t.Cleanup(func() {
		_ = srv.Close()
		_ = serving.Wait()
		app.shutdown()
	})
	return app
}

// shutdown drops every connection, waits for the tool calls in flight, and
// closes the session of every thread still loaded.
func (app *codexSharedApp) shutdown() {
	app.mu.Lock()
	app.closed = true
	conns, threads := slices.Clone(app.conns), app.threads
	app.threads = nil
	app.mu.Unlock()
	for _, conn := range conns {
		_ = conn.CloseNow()
	}
	_ = app.forwarding.Wait()
	for _, th := range threads {
		_ = th.session.Close()
	}
}

func (app *codexSharedApp) serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(32 << 20)
	app.mu.Lock()
	app.conns = append(app.conns, conn)
	app.mu.Unlock()
	defer app.unload(conn)
	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			return
		}
		app.answer(conn, data)
	}
}

// unload is the client on conn going away: the threads it started unload,
// their MCP sessions close, and every other connection hears of it. A real app
// server keeps a stopped TUI's thread loaded for about a minute before it does
// the same; the fake does not wait.
func (app *codexSharedApp) unload(conn *websocket.Conn) {
	app.mu.Lock()
	app.conns = slices.DeleteFunc(app.conns, func(c *websocket.Conn) bool { return c == conn })
	var gone []*codexSharedThread
	app.threads = slices.DeleteFunc(app.threads, func(th *codexSharedThread) bool {
		if th.owner != conn {
			return false
		}
		gone = append(gone, th)
		return true
	})
	app.mu.Unlock()
	for _, th := range gone {
		_ = th.session.Close()
		app.push(fmt.Sprintf(`{"method":"thread/status/changed",`+
			`"params":{"threadId":%q,"status":{"type":"notLoaded"}}}`, th.id))
		app.push(fmt.Sprintf(`{"method":"thread/closed","params":{"threadId":%q}}`, th.id))
	}
}

// answer replies to one request, leaving the version marker off the way the
// real app server does. A notification is not answered.
func (app *codexSharedApp) answer(conn *websocket.Conn, data []byte) {
	var frame struct {
		Id     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			ThreadId  string                     `json:"threadId"`
			Config    map[string]json.RawMessage `json:"config"`
			Server    string                     `json:"server"`
			Tool      string                     `json:"tool"`
			Arguments map[string]any             `json:"arguments"`
			Input     []struct {
				Text string `json:"text"`
			} `json:"input"`
		} `json:"params"`
	}
	if json.Unmarshal(data, &frame) != nil || len(frame.Id) == 0 {
		return
	}
	p := frame.Params
	switch frame.Method {
	case "thread/start":
		app.startThread(conn, frame.Id, p.Config)
	case "thread/loaded/list":
		ids, err := json.Marshal(app.loaded())
		if err != nil {
			codexSharedRefuse(conn, frame.Id, -32603, err.Error())
			return
		}
		codexSharedReply(conn, frame.Id, `{"data":`+string(ids)+`,"nextCursor":null}`)
	case "thread/read":
		if app.thread(p.ThreadId) == nil {
			codexSharedRefuse(conn, frame.Id, -32600, "thread not loaded: "+p.ThreadId)
			return
		}
		codexSharedReply(conn, frame.Id, fmt.Sprintf(
			`{"thread":{"id":%q,"threadSource":"user","recencyAt":0}}`, p.ThreadId))
	case "thread/resume":
		// A thread that is still loaded keeps the MCP session it has, whatever
		// header the resume carries: the real app server does not restart the
		// thread's MCP server either.
		if app.thread(p.ThreadId) == nil {
			codexSharedRefuse(conn, frame.Id, -32600, "no rollout found for thread id "+p.ThreadId)
			return
		}
		codexSharedReply(conn, frame.Id, fmt.Sprintf(
			`{"thread":{"id":%q,"threadSource":"user","status":{"type":"idle"}}}`, p.ThreadId))
	case "thread/unsubscribe":
		codexSharedReply(conn, frame.Id, `{"status":"unsubscribed"}`)
	case "turn/start":
		if app.thread(p.ThreadId) == nil {
			codexSharedRefuse(conn, frame.Id, -32600, "thread not loaded: "+p.ThreadId)
			return
		}
		var text string
		for _, in := range p.Input {
			text += in.Text
		}
		app.mu.Lock()
		app.turns = append(app.turns, codexSharedTurn{thread: p.ThreadId, text: text})
		n := len(app.turns)
		app.mu.Unlock()
		codexSharedReply(
			conn,
			frame.Id,
			fmt.Sprintf(`{"turn":{"id":"turn-%d","status":"inProgress"}}`, n),
		)
	case "mcpServer/tool/call":
		app.callTool(conn, frame.Id, p.ThreadId, p.Server, p.Tool, p.Arguments)
	default:
		codexSharedReply(conn, frame.Id, `{}`)
	}
}

// startThread loads a new thread and starts its crabswarm-mcp server the way
// Codex starts a streamable-HTTP one: an MCP session of the thread's own, whose
// every request carries the headers the thread's config sets. The thread is
// answered once that session is up, since Codex reports the thread's MCP
// servers ready before it answers, and then announced to every connection.
func (app *codexSharedApp) startThread(
	conn *websocket.Conn, id json.RawMessage, config map[string]json.RawMessage,
) {
	header := http.Header{}
	for key, raw := range config {
		name, ok := strings.CutPrefix(key, codexSharedHeaderKey)
		if !ok {
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			codexSharedRefuse(conn, id, -32600, fmt.Sprintf("config %s: %v", key, err))
			return
		}
		header.Set(name, value)
	}
	app.mu.Lock()
	app.numbered++
	thread := fmt.Sprintf("01a0f000-0000-7000-8000-%012d", app.numbered)
	app.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), codexSharedTimeout)
	defer cancel()
	// The name and version Codex 0.157.1 gave itself on the recorded session.
	client := mcp.NewClient(&mcp.Implementation{
		Name: "codex-mcp-client", Title: "Codex", Version: "0.157.1",
	}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   app.endpoint,
		HTTPClient: &http.Client{Transport: codexSharedHeaders(header)},
	}, nil)
	if err != nil {
		codexSharedRefuse(conn, id, -32603,
			"starting the "+codexSharedServer+" server: "+err.Error())
		return
	}
	app.mu.Lock()
	if app.closed {
		app.mu.Unlock()
		_ = session.Close()
		return
	}
	app.threads = append(app.threads, &codexSharedThread{id: thread, owner: conn, session: session})
	app.mu.Unlock()
	codexSharedReply(conn, id, fmt.Sprintf(
		`{"thread":{"id":%q,"threadSource":"user","status":{"type":"idle"},"turns":[]}}`, thread))
	app.push(fmt.Sprintf(`{"method":"thread/started","params":{"thread":{"id":%q}}}`, thread))
}

// codexSharedHeaders is the headers a thread's config sets, sent on every
// request of that thread's MCP session.
type codexSharedHeaders http.Header

func (h codexSharedHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	maps.Copy(req.Header, h)
	return http.DefaultTransport.RoundTrip(req)
}

// callTool forwards one tool call into the session of the thread it names, on
// a goroutine of its own: the server behind that session may ask this very
// connection something before it answers.
func (app *codexSharedApp) callTool(
	conn *websocket.Conn, id json.RawMessage, thread, server, tool string, args map[string]any,
) {
	th := app.thread(thread)
	if th == nil || server != codexSharedServer {
		codexSharedRefuse(conn, id, -32602,
			fmt.Sprintf("no MCP server %q on thread %s", server, thread))
		return
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.closed {
		return
	}
	app.forwarding.Go(func() error {
		app.forward(conn, id, th, tool, args)
		return nil
	})
}

// forward makes the tool call on th's session with th in its metadata, as the
// app server does, and answers with what the session said.
func (app *codexSharedApp) forward(
	conn *websocket.Conn,
	id json.RawMessage,
	th *codexSharedThread,
	tool string,
	args map[string]any,
) {
	ctx, cancel := context.WithTimeout(context.Background(), codexSharedTimeout)
	defer cancel()
	res, err := th.session.CallTool(ctx, &mcp.CallToolParams{
		Meta:      mcp.Meta{"threadId": th.id},
		Name:      tool,
		Arguments: args,
	})
	if err != nil {
		codexSharedRefuse(conn, id, -32602, err.Error())
		return
	}
	out, err := json.Marshal(res)
	if err != nil {
		codexSharedRefuse(conn, id, -32603, err.Error())
		return
	}
	app.mu.Lock()
	app.calls = append(app.calls, codexSharedCall{thread: th.id, tool: tool})
	app.mu.Unlock()
	codexSharedReply(conn, id, string(out))
}

// push writes one notification to every connection.
func (app *codexSharedApp) push(frame string) {
	app.mu.Lock()
	conns := slices.Clone(app.conns)
	app.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Write(context.Background(), websocket.MessageText, []byte(frame))
	}
}

// loaded is the ids of the loaded threads, oldest first.
func (app *codexSharedApp) loaded() []string {
	app.mu.Lock()
	defer app.mu.Unlock()
	ids := make([]string, 0, len(app.threads))
	for _, th := range app.threads {
		ids = append(ids, th.id)
	}
	return ids
}

// thread is the loaded thread id names, nil for none.
func (app *codexSharedApp) thread(id string) *codexSharedThread {
	app.mu.Lock()
	defer app.mu.Unlock()
	for _, th := range app.threads {
		if th.id == id {
			return th
		}
	}
	return nil
}

// startedTurns is every turn the fake was asked to start, oldest first.
func (app *codexSharedApp) startedTurns() []codexSharedTurn {
	app.mu.Lock()
	defer app.mu.Unlock()
	return slices.Clone(app.turns)
}

// answeredCalls is every tool call a thread's session answered, oldest first.
func (app *codexSharedApp) answeredCalls() []codexSharedCall {
	app.mu.Lock()
	defer app.mu.Unlock()
	return slices.Clone(app.calls)
}

func codexSharedReply(conn *websocket.Conn, id json.RawMessage, result string) {
	_ = conn.Write(context.Background(), websocket.MessageText,
		fmt.Appendf(nil, `{"id":%s,"result":%s}`, id, result))
}

func codexSharedRefuse(conn *websocket.Conn, id json.RawMessage, code int, message string) {
	msg, err := json.Marshal(message)
	if err != nil {
		return
	}
	_ = conn.Write(context.Background(), websocket.MessageText,
		fmt.Appendf(nil, `{"id":%s,"error":{"code":%d,"message":%s}}`, id, code, msg))
}

// waitCodexSharedTurns blocks until the fake has been asked to start exactly
// want, oldest first, so a case pins which thread each mention reached and
// that no other turn was started anywhere.
func waitCodexSharedTurns(t *testing.T, app *codexSharedApp, want ...codexSharedTurn) {
	t.Helper()
	var got []codexSharedTurn
	deadline := time.Now().Add(codexSharedTimeout)
	for time.Now().Before(deadline) {
		got = app.startedTurns()
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the app server was asked to start (thread, text)\n%q\nwant\n%q", got, want)
}

// waitCodexSharedCall blocks until a thread's session has answered want.
func waitCodexSharedCall(t *testing.T, app *codexSharedApp, want codexSharedCall) {
	t.Helper()
	var got []codexSharedCall
	deadline := time.Now().Add(codexSharedTimeout)
	for time.Now().Before(deadline) {
		got = app.answeredCalls()
		if slices.Contains(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the threads' sessions answered (thread, tool)\n%q\nnone of which is %q", got, want)
}

// codexSharedClient is one client of the app server, as a remote TUI is one:
// it sends one frame at a time and waits for the answer to a request, reading
// past whatever the server pushes meanwhile.
type codexSharedClient struct {
	conn *websocket.Conn
}

// dialCodexShared connects to the app server listening on the unix socket at
// path. The connection lives as long as ctx does.
func dialCodexShared(ctx context.Context, path string) (*codexSharedClient, error) {
	httpClient := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		},
	}}
	conn, _, err := websocket.Dial(ctx, "ws://localhost/", &websocket.DialOptions{
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("dialing the app server on %s: %w", path, err)
	}
	conn.SetReadLimit(32 << 20)
	return &codexSharedClient{conn: conn}, nil
}

// send writes frame and, when it is a request, returns the frame answering it.
// A notification is answered by nothing, and send returns nil for it.
func (c *codexSharedClient) send(ctx context.Context, frame []byte) (json.RawMessage, error) {
	id, err := codexSharedId(frame)
	if err != nil {
		return nil, err
	}
	if err := c.conn.Write(ctx, websocket.MessageText, frame); err != nil {
		return nil, err
	}
	if id == nil {
		return nil, nil
	}
	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return nil, err
		}
		var answer struct {
			Id     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(data, &answer) != nil || answer.Method != "" {
			continue
		}
		if got, err := codexSharedCompact(answer.Id); err == nil && bytes.Equal(got, id) {
			return data, nil
		}
	}
}

// close says goodbye the way a TUI that quits does.
func (c *codexSharedClient) close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// codexSharedId is the id of the request frame is, compacted so that two
// spellings of one id compare equal, or nil for a notification.
func codexSharedId(frame []byte) (json.RawMessage, error) {
	var req struct {
		Id json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(frame, &req); err != nil {
		return nil, fmt.Errorf("reading the frame %s: %w", frame, err)
	}
	if len(req.Id) == 0 {
		return nil, nil
	}
	return codexSharedCompact(req.Id)
}

func codexSharedCompact(raw json.RawMessage) (json.RawMessage, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// codexSharedTUIEnv marks a run of this test binary as a remote Codex TUI
// rather than as the suite. `crabswarm mcp codex-proxy` runs the TUI it sits
// in front of as a command of its own, so a case hands it this binary with the
// variable set. It sits under CRABSWARM_ so that chatEnviron keeps it from
// every other process the suite starts.
const codexSharedTUIEnv = "CRABSWARM_E2E_CODEX_TUI"

// init turns the binary into the TUI before the suite's TestMain runs, which
// would otherwise build crabswarm and run every test.
func init() {
	if os.Getenv(codexSharedTUIEnv) == "" {
		return
	}
	if err := runCodexSharedTUI(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "fake codex TUI:", err)
		os.Exit(1)
	}
	os.Exit(0)
}

// runCodexSharedTUI attaches to the app server `--remote unix://PATH` names,
// sends every line of stdin as one frame, and writes the answer to each
// request to stdout as one line. Closing stdin is the TUI quitting: it closes
// the connection the way a TUI does and exits 0.
func runCodexSharedTUI(args []string) error {
	if len(args) != 2 || args[0] != "--remote" {
		return fmt.Errorf("want --remote unix://PATH, got %q", args)
	}
	path, ok := strings.CutPrefix(args[1], "unix://")
	if !ok {
		return fmt.Errorf("%q is no unix:// address", args[1])
	}
	// The connection outlives every request, so it is dialled on a context of
	// its own; each request below is bounded by one of its own instead.
	c, err := dialCodexShared(context.Background(), path)
	if err != nil {
		return err
	}
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(nil, 1<<20)
	for in.Scan() {
		ctx, cancel := context.WithTimeout(context.Background(), codexSharedTimeout)
		answer, err := c.send(ctx, in.Bytes())
		cancel()
		if err != nil {
			_ = c.conn.CloseNow()
			return err
		}
		if answer == nil {
			continue
		}
		if _, err := os.Stdout.Write(append(answer, '\n')); err != nil {
			_ = c.conn.CloseNow()
			return err
		}
	}
	if err := in.Err(); err != nil {
		_ = c.conn.CloseNow()
		return err
	}
	if err := c.close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return err
	}
	return nil
}
