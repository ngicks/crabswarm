package crabswarm_test

import (
	"context"
	"encoding/json"
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
	"golang.org/x/sync/errgroup"

	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// A Codex session in a swarm is hosted by `codex app-server --listen unix://…`,
// and the MCP server reaches its agent by starting a turn on it. The
// suite cannot run Codex, so what stands in for it here is a fake app server
// speaking the same wire: an HTTP upgrade on a unix socket, then one JSON-RPC
// message per text frame, answered the way the real one answers — with no
// version marker on anything it sends. testdata/harness/README.md records the
// exchange this was written from.
//
// The unit tests in pkg/harnessctl replay the captured session against a
// fake of their own; that one lives in a test file and cannot be imported, so
// this is the small second copy.

// codexFakeThread is the thread the fake says a person is sitting in front of,
// which is the one the MCP server has to bind to.
const codexFakeThread = "01a0afbe-2c74-7452-a0fd-6858a3d7a885"

// codexFakeGuardian is loaded beside it, the way Codex keeps its guardian
// approval reviewer loaded under a session. It is there so a delivery has to
// tell the two apart the way it must on a real app server.
const codexFakeGuardian = "01a0c58c-ce42-78b2-8eea-5d3d232fc421"

// codexAppServer is a fake app server on a unix socket.
type codexAppServer struct {
	addr string

	mu     sync.Mutex
	turns  []string
	loaded []string
	// sources is the threadSource each loaded thread reports.
	sources map[string]string
	conns   []*websocket.Conn
	// accepted is every connection the fake has taken since it started, the
	// closed ones included. Counted rather than measured off the live ones: a
	// delivery that dialled a connection of its own closes it again, and by the
	// time a case looked there would be nothing left to see.
	accepted int
}

// startCodexAppServer listens on a socket of its own until the test ends.
//
// The socket sits under a short temporary directory rather than under the
// test's: a unix socket path is capped well below what a directory named after
// a Go test case comes to, and the real app server refuses a long one outright.
func startCodexAppServer(t *testing.T) *codexAppServer {
	t.Helper()

	dir, err := os.MkdirTemp("", "cx")
	if err != nil {
		t.Fatalf("make a socket dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	f := &codexAppServer{
		addr:   filepath.Join(dir, "app.sock"),
		loaded: []string{codexFakeGuardian, codexFakeThread},
		sources: map[string]string{
			codexFakeThread:   "user",
			codexFakeGuardian: "guardian_review",
		},
	}
	ln, err := net.Listen("unix", f.addr)
	if err != nil {
		t.Fatalf("listen on %s: %v", f.addr, err)
	}
	srv := &http.Server{
		Handler:           http.HandlerFunc(f.serve),
		ReadHeaderTimeout: 10 * time.Second,
	}
	var serving errgroup.Group
	serving.Go(func() error { return srv.Serve(ln) })
	t.Cleanup(func() {
		_ = srv.Close()
		_ = serving.Wait()
	})
	return f
}

func (f *codexAppServer) serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(32 << 20)
	f.mu.Lock()
	f.conns = append(f.conns, conn)
	f.accepted++
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.conns = slices.DeleteFunc(f.conns, func(c *websocket.Conn) bool { return c == conn })
		f.mu.Unlock()
	}()
	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			return
		}
		f.answer(conn, data)
	}
}

// answer records a call and replies to it, leaving the version marker off the
// way the real app server does.
func (f *codexAppServer) answer(conn *websocket.Conn, data []byte) {
	var frame struct {
		Id     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Input []struct {
				Text string `json:"text"`
			} `json:"input"`
		} `json:"params"`
	}
	if json.Unmarshal(data, &frame) != nil {
		return
	}
	result := `{}`
	switch frame.Method {
	case "thread/loaded/list":
		f.mu.Lock()
		ids, err := json.Marshal(f.loaded)
		f.mu.Unlock()
		if err != nil {
			return
		}
		result = `{"data":` + string(ids) + `,"nextCursor":null}`
	case "turn/start":
		var text string
		for _, in := range frame.Params.Input {
			text += in.Text
		}
		f.mu.Lock()
		f.turns = append(f.turns, text)
		f.mu.Unlock()
		result = `{"turn":{"id":"turn-1","status":"inProgress"}}`
	}
	if len(frame.Id) == 0 {
		return
	}
	_ = conn.Write(context.Background(), websocket.MessageText,
		fmt.Appendf(nil, `{"id":%s,"result":%s}`, frame.Id, result))
}

// started is the text of every turn the fake was asked to start, oldest first.
func (f *codexAppServer) started() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.turns)
}

// pushStatus tells every connection what the thread is doing now, in the frame
// shape the real server pushes it in — stamp beside the parameters, no version
// marker.
func (f *codexAppServer) pushStatus(status string) {
	frame := fmt.Appendf(nil,
		`{"method":"thread/status/changed","params":{"threadId":%q,"status":%s},`+
			`"emittedAtMs":1789654916664}`, codexFakeThread, status)
	f.mu.Lock()
	conns := slices.Clone(f.conns)
	f.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Write(context.Background(), websocket.MessageText, frame)
	}
}

// connections is how many connections the fake has taken in all.
func (f *codexAppServer) connections() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.accepted
}

// waitCodexConnection blocks until the MCP server has connected, which is what
// a status push needs: the fake pushes to the connections it holds, and one
// pushed before the MCP server arrived reaches nobody.
func waitCodexConnection(t *testing.T, f *codexAppServer) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if f.connections() > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the MCP server never connected to the app server")
}

// waitCodexTurns blocks until the fake has been asked to start exactly want,
// so a case pins both what reached the agent and that nothing else did.
func waitCodexTurns(t *testing.T, f *codexAppServer, want ...string) {
	t.Helper()
	var got []string
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		got = f.started()
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the app server was asked to start\n%q\nwant\n%q", got, want)
}

// The whole Codex story in one session. The MCP server reads the app server's
// address out of its environment, so the member attends as one the daemon never
// types at, and every notice it is owed arrives as a turn on the one thread the
// app server has loaded — while the agent is idle, never mid-turn, and once the
// turn ends as a count of what piled up meanwhile.
func TestChatCodex_BridgeStartsATurnForEveryMention(t *testing.T) {
	app := startCodexAppServer(t)
	cfg := startChatDaemon(t)
	startChatBridgeAs(t, cfg, "tok-ana", "codex-mcp-client",
		append(chatEnviron(), harnessctl.CodexAppServerEnv+"=unix://"+app.addr))
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	// The daemon has it from the handshake: which CLI the member runs, and that
	// its mentions are its own server's to deliver.
	roster := lines(runChat(t, cfg, "tok-bob", "members"))
	slices.Sort(roster)
	want := []string{
		chatBridgeAna + "  agent  done  codex  native",
		chatBridgeBob + "  agent  done  other  terminal",
	}
	if !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}

	// Mentioned while idle: the notice becomes a turn, and no keystroke is
	// typed anywhere.
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	arrival := "[crabswarm chat] new message from " + chatBridgeBob +
		" — read it with the chat_read tool and respond with chat_send," +
		" both from crabswarm-mcp"
	waitCodexTurns(t, app, arrival)
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Mentioned mid-turn: nothing is started on a session that is working, and
	// the mention waits in the room where it already is.
	runChat(t, cfg, "tok-ana", "report-state", "working")
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "and the rebase after it")

	// The turn ends and what waited becomes one turn of its own. Both messages
	// are still unread — a notice is not a read — so the count is two.
	runChat(t, cfg, "tok-ana", "report-state", "done")
	waitCodexTurns(t, app, arrival,
		"[crabswarm chat] 2 unread messages mention you"+
			" — read them with the chat_read tool and respond with chat_send,"+
			" both from crabswarm-mcp")
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Delivered is not read: both messages are waiting where the agent's own
	// read finds them.
	read := runChat(t, cfg, "tok-ana", "read")
	for _, want := range []string{"the migration needs you", "and the rebase after it"} {
		if !strings.Contains(read, want) {
			t.Errorf("read as %s = %q, want it to carry %q", chatBridgeAna, read, want)
		}
	}
}

// The app server is also where Codex says what it is doing, so a session going
// busy and quiet again reaches the room without a hook reporting anything — and
// what the room hears is what decides when the next mention may be delivered.
//
// The half of that inside the MCP server — the connection, the subscription and
// the mapping onto harness states — is pinned by the unit tests in
// pkg/harnessctl. What is asserted here is the other half: the MCP
// server passing what it heard on to the daemon.
func TestChatCodex_TheAppServerFeedBecomesTheMemberState(t *testing.T) {
	app := startCodexAppServer(t)
	cfg := startChatDaemon(t)
	startChatBridgeAs(t, cfg, "tok-ana", "codex-mcp-client",
		append(chatEnviron(), harnessctl.CodexAppServerEnv+"=unix://"+app.addr))
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	// A status pushed before the MCP server connected reaches nobody.
	waitCodexConnection(t, app)

	// The daemon publishes every report onto the stub cmdman's status log, so
	// the whole trail is visible rather than only the state it ended on. The
	// first line is the attendance itself; each one after it is a status below.
	trail := []string{codexStatusLine("done")}
	for _, step := range []struct{ status, state string }{
		{`{"type":"idle"}`, "done"},
		{`{"type":"active","activeFlags":[]}`, "working"},
		{`{"type":"idle"}`, "done"},
	} {
		app.pushStatus(step.status)
		trail = append(trail, codexStatusLine(step.state))
		// One at a time: a real session's statuses are a turn apart, and two
		// frames arriving in the same instant reach the MCP server in whichever
		// order their handlers win the JSON-RPC client's lock, so three pushed
		// together would make this a case about that race.
		waitCodexStatusTrail(t, cfg, trail)
	}

	// The session said it was idle again, so a mention arriving now is delivered
	// on the spot — as a turn over the connection the feed is already holding,
	// rather than over one dialled for the delivery alone.
	//
	// Bob attends only here: his own attendance publishes a status of its own,
	// and the trail above is ana's.
	attendChatBridges(t, cfg, "tok-bob")
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	waitCodexTurns(t, app, "[crabswarm chat] new message from "+chatBridgeBob+
		" — read it with the chat_read tool and respond with chat_send,"+
		" both from crabswarm-mcp")
	if conns := app.connections(); conns != 1 {
		t.Errorf("the app server took %d connections, want the watched one alone", conns)
	}
}

// codexStatusLine is one `cmdman status` invocation as the stub records it, for
// the member this case watches.
func codexStatusLine(state string) string {
	return "set " + state + " tok-ana --detail crabswarm chat"
}

// waitCodexStatusTrail blocks until the daemon has published exactly want
// through the stub cmdman, oldest first, so a case pins the order of the states
// as much as the states themselves.
func waitCodexStatusTrail(t *testing.T, cfgPath string, want []string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var published []string
	for time.Now().Before(deadline) {
		published = stubStatus(t, cfgPath)
		if slices.Equal(published, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("cmdman status invocations =\n%q\nwant\n%q", published, want)
}

// Codex's hook file wires the delivery and nothing else. Its state comes off
// the app server the MCP server is already connected to, and a hook reporting
// the same thing would race that feed with a slower, coarser answer.
func TestChatCodex_HooksLeaveTheStateToTheAppServer(t *testing.T) {
	codex := readCodexHooks(t)

	for event, groups := range codex.Hooks {
		for _, group := range groups {
			for _, h := range group.Hooks {
				if strings.Contains(h.Command, "report-state") {
					t.Errorf(
						"%s command %q reports state; the app server does that",
						event,
						h.Command,
					)
				}
				assertSelfContainedHookEntry(t, event, h)
			}
		}
	}
}

// readCodexHooks decodes the Codex hook file out of the checkout under test.
func readCodexHooks(t *testing.T) chatHookConfig {
	t.Helper()
	path := filepath.Join(repoRoot(), "apm-package", "crabswarm-mcp",
		".apm", "hooks", "codex-hooks.json")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read hook file %s: %v", path, err)
	}
	var cfg chatHookConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		t.Fatalf("decode hook file %s: %v", path, err)
	}
	if len(cfg.Hooks) == 0 {
		t.Fatalf("hook file %s wires no events", path)
	}
	return cfg
}

// Codex's Stop hook still drains the inbox, and still refuses to drain it
// twice: with an earlier block already in force the command renders empty,
// which `hook exec` runs as a plain allow. Nothing is read, so the messages are
// still there for the turn the block bought.
func TestChatCodex_StopDeliversOnceAndThenStandsAside(t *testing.T) {
	stop := readCodexHooks(t).command(t, "Stop")

	t.Run("messages in hand at turn end", func(t *testing.T) {
		cfg := startChatRoomWithMail(t)
		res := runChatHook(t, cfg, "tok-ana", stop, chatStopEnvelope)
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s",
				res.exitCode, res.stdout, res.stderr)
		}
		obj := hookObject(t, res.stdout)
		assertKeys(t, obj, "decision", "reason")
		if got := hookString(t, obj, "decision"); got != "block" {
			t.Errorf("decision = %q, want %q", got, "block")
		}
		if reason := hookString(t, obj, "reason"); !strings.Contains(reason, chatMailLine) {
			t.Errorf("reason = %q, want it to carry %q", reason, chatMailLine)
		}
		assertInboxWasConsumed(t, cfg)
	})

	t.Run("an earlier block is already in force", func(t *testing.T) {
		cfg := startChatRoomWithMail(t)
		assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", stop, chatStopActiveEnvelope))
		assertInboxStillHoldsTheMail(t, cfg)
	})
}

// Every command the Codex file wires, on every event, against a daemon that is
// not running: none of them says anything and none of them fails. A chat nobody
// is hosting must not break a turn or a tool call.
func TestChatCodex_HooksAreHarmlessWithoutADaemon(t *testing.T) {
	cfg := chatAbsentDaemonConfig(t)
	hooks := readCodexHooks(t)
	for _, event := range slices.Sorted(maps.Keys(hooks.Hooks)) {
		for _, group := range hooks.Hooks[event] {
			envelope := chatHookEnvelope(event, group.Matcher)
			if envelope == "" {
				t.Fatalf("event %s is wired but has no envelope in chatHookEnvelopes", event)
			}
			for i, h := range group.Hooks {
				t.Run(fmt.Sprintf("%s/%s/%d", event, group.Matcher, i), func(t *testing.T) {
					assertHookIsSilent(t, runChatHook(t, cfg, "tok-ana", h.Command, envelope))
				})
			}
		}
	}
}
