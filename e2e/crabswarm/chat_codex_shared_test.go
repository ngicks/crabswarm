package crabswarm_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// Agent replicas that share one Codex app server, end to end: the daemon, the
// fake app server of codexfake_shared_test.go, the one `crabswarm mcp
// --transport http` the MCP session of every thread reaches, and a real
// `crabswarm mcp codex-proxy` in front of each replica's TUI. The TUI is this
// test binary; the proxy runs it as its command.

// The recordings the TUIs' frames come from.
const (
	// codexSharedLifecycleFixture is one remote TUI's thread requests: the
	// thread/start it sent at startup (the first), a thread/start of a helper
	// thread (the second), the thread/start of /new (the third) and the
	// thread/resume of /resume.
	codexSharedLifecycleFixture = "codex-tui-thread-lifecycle.client.ndjson"
	// codexSharedToolCallFixture is a client's handshake and its
	// mcpServer/tool/call of chat_members.
	codexSharedToolCallFixture = "codex-app-server-tool-call.client.ndjson"
)

// codexSharedArrival is the notice a mention from the writer becomes.
const codexSharedArrival = "[crabswarm chat] new message from " + chatBridgeCid +
	" — read it with the chat_read tool and respond with chat_send," +
	" both from crabswarm-mcp"

// codexSharedTUI is a TUI a case drives: it sends one frame and hands back the
// answer to it, nil for a notification.
type codexSharedTUI interface {
	call(t *testing.T, frame []byte) json.RawMessage
}

// codexSharedFrame is the nth frame of method a recorded client sent, counted
// from zero.
func codexSharedFrame(t *testing.T, fixture, method string, nth int) []byte {
	t.Helper()
	for line := range bytes.SplitSeq(harnessFixtureBytes(t, fixture), []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var frame struct {
			Method string `json:"method"`
		}
		if err := json.Unmarshal(line, &frame); err != nil {
			t.Fatalf("decode a frame of %s: %v", fixture, err)
		}
		if frame.Method != method {
			continue
		}
		if nth == 0 {
			return line
		}
		nth--
	}
	t.Fatalf("%s holds too few %s frames", fixture, method)
	return nil
}

// codexSharedOnThread is frame addressed to thread instead of the thread it
// was recorded against. A frame naming an MCP server names codexSharedServer
// instead: the recordings predate the shared package's own entry name.
func codexSharedOnThread(t *testing.T, frame []byte, thread string) []byte {
	t.Helper()
	var msg, params map[string]json.RawMessage
	if err := json.Unmarshal(frame, &msg); err != nil {
		t.Fatalf("decode %s: %v", frame, err)
	}
	if err := json.Unmarshal(msg["params"], &params); err != nil {
		t.Fatalf("decode the params of %s: %v", frame, err)
	}
	var err error
	if params["threadId"], err = json.Marshal(thread); err != nil {
		t.Fatalf("encode a thread id: %v", err)
	}
	if _, ok := params["server"]; ok {
		if params["server"], err = json.Marshal(codexSharedServer); err != nil {
			t.Fatalf("encode a server name: %v", err)
		}
	}
	if msg["params"], err = json.Marshal(params); err != nil {
		t.Fatalf("encode params: %v", err)
	}
	out, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("encode a frame: %v", err)
	}
	return out
}

// codexSharedHandshake opens tui's session with the handshake a recorded
// client opened with.
func codexSharedHandshake(t *testing.T, tui codexSharedTUI) {
	t.Helper()
	for _, method := range []string{"initialize", "initialized"} {
		tui.call(t, codexSharedFrame(t, codexSharedToolCallFixture, method, 0))
	}
}

// codexSharedStartThread sends a recorded thread/start through tui and returns
// the id of the thread the app server started.
func codexSharedStartThread(t *testing.T, tui codexSharedTUI, frame []byte) string {
	t.Helper()
	answer := tui.call(t, frame)
	var res struct {
		Result struct {
			Thread struct {
				Id string `json:"id"`
			} `json:"thread"`
		} `json:"result"`
	}
	if err := json.Unmarshal(answer, &res); err != nil || res.Result.Thread.Id == "" {
		t.Fatalf("thread/start was answered %s, want a thread", answer)
	}
	return res.Result.Thread.Id
}

// codexSharedResume sends the recorded thread/resume of /resume through tui,
// addressed to thread.
func codexSharedResume(t *testing.T, tui codexSharedTUI, thread string) {
	t.Helper()
	frame := codexSharedOnThread(t,
		codexSharedFrame(t, codexSharedLifecycleFixture, "thread/resume", 0), thread)
	answer := tui.call(t, frame)
	var res struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(answer, &res); err != nil || len(res.Result) == 0 {
		t.Fatalf("thread/resume of %s was answered %s, want a result", thread, answer)
	}
}

// codexSharedReplica is one agent replica: a real `crabswarm mcp codex-proxy`
// whose command is this test binary acting as the replica's remote TUI. The TUI
// inherits the proxy's stdio, so a line written to the proxy's stdin is a
// frame the TUI sends, and a line on its stdout is an answer the TUI got.
type codexSharedReplica struct {
	proxy *exec.Cmd
	in    io.WriteCloser
	out   *bufio.Reader

	stopOnce sync.Once
	stopErr  error
}

// startCodexSharedReplica starts a proxy on cfg with args in front of a TUI
// attached to app, with env added to the suite's environment, and returns once
// the TUI has made its handshake. cfg names no socket directory, so the proxy
// serves its socket under runtimeDir.
func startCodexSharedReplica(
	t *testing.T, cfg string, app *codexSharedApp, runtimeDir string, args []string,
	env ...string,
) *codexSharedReplica {
	t.Helper()
	tui, err := os.Executable()
	if err != nil {
		t.Fatalf("locate the test binary: %v", err)
	}
	argv := append([]string{"mcp", "codex-proxy", "--config", cfg}, args...)
	argv = append(argv, "--", tui, "--remote", "unix://"+app.path)
	proxy := exec.Command(crabswarmBin, argv...)
	proxy.Env = append(chatEnviron(), "XDG_RUNTIME_DIR="+runtimeDir, codexSharedTUIEnv+"=1")
	proxy.Env = append(proxy.Env, env...)
	proxy.Stderr = os.Stderr
	in, err := proxy.StdinPipe()
	if err != nil {
		t.Fatalf("open the proxy's stdin: %v", err)
	}
	out, err := proxy.StdoutPipe()
	if err != nil {
		t.Fatalf("open the proxy's stdout: %v", err)
	}
	if err := proxy.Start(); err != nil {
		t.Fatalf("start crabswarm mcp codex-proxy: %v", err)
	}
	r := &codexSharedReplica{proxy: proxy, in: in, out: bufio.NewReader(out)}
	t.Cleanup(func() { _ = r.stop() })
	codexSharedHandshake(t, r)
	return r
}

func (r *codexSharedReplica) call(t *testing.T, frame []byte) json.RawMessage {
	t.Helper()
	id, err := codexSharedId(frame)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.in.Write(append(slices.Clone(frame), '\n')); err != nil {
		t.Fatalf("hand the TUI %s: %v", frame, err)
	}
	if id == nil {
		return nil
	}
	answer, err := r.out.ReadBytes('\n')
	if err != nil {
		t.Fatalf("the TUI got no answer to %s: %v", frame, err)
	}
	return bytes.TrimSuffix(answer, []byte("\n"))
}

// stop closes the TUI's input, which is the TUI quitting, and returns what the
// proxy exited with. A proxy still running after codexSharedTimeout is killed.
func (r *codexSharedReplica) stop() error {
	r.stopOnce.Do(func() {
		_ = r.in.Close()
		timer := time.AfterFunc(codexSharedTimeout, func() { _ = r.proxy.Process.Kill() })
		defer timer.Stop()
		r.stopErr = r.proxy.Wait()
	})
	return r.stopErr
}

// codexSharedDirect is a client on the app server's own socket, past any
// proxy, as a TUI started without one is.
type codexSharedDirect struct {
	client *codexSharedClient
}

func dialCodexSharedDirect(t *testing.T, app *codexSharedApp) *codexSharedDirect {
	t.Helper()
	// Dialled on the test's context, the connection ends with the test, and the
	// fake unloads the thread it started then.
	c, err := dialCodexShared(t.Context(), app.path)
	if err != nil {
		t.Fatal(err)
	}
	d := &codexSharedDirect{client: c}
	codexSharedHandshake(t, d)
	return d
}

func (d *codexSharedDirect) call(t *testing.T, frame []byte) json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), codexSharedTimeout)
	defer cancel()
	answer, err := d.client.send(ctx, frame)
	if err != nil {
		t.Fatalf("send %s to the app server: %v", frame, err)
	}
	return answer
}

// startCodexSharedMCP starts the one `crabswarm mcp --transport http` every
// thread's session reaches, launched beside the app server the way the app
// server's own launcher would, and returns once it listens on addr.
func startCodexSharedMCP(t *testing.T, cfg, addr string, app *codexSharedApp) {
	t.Helper()
	cmd := exec.Command(crabswarmBin, "mcp", "--config", cfg,
		"--transport", "http", "--listen", addr)
	cmd.Env = append(chatEnviron(), harnessctl.CodexAppServerEnv+"=unix://"+app.path)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start crabswarm mcp --transport http: %v", err)
	}
	t.Cleanup(func() { stopProcess(t, cmd) })
	waitCodexShared(t, "crabswarm mcp listening on "+addr, func() bool {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		_ = conn.Close()
		return true
	})
}

// waitCodexShared blocks until cond holds, and fails the test naming what never
// happened.
func waitCodexShared(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(codexSharedTimeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s did not happen within %s", what, codexSharedTimeout)
}

// waitCodexSharedRoster blocks until `chat members` as observer lists exactly
// want, in any order.
func waitCodexSharedRoster(t *testing.T, cfg, observer string, want ...string) {
	t.Helper()
	want = slices.Sorted(slices.Values(want))
	var got []string
	deadline := time.Now().Add(codexSharedTimeout)
	for time.Now().Before(deadline) {
		got = lines(runChat(t, cfg, observer, "members"))
		slices.Sort(got)
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("members as %s =\n%q\nwant\n%q", observer, got, want)
}

// The roster rows of the three members every case starts with.
const (
	codexSharedRowAna = chatBridgeAna + "  agent  done  codex  native"
	codexSharedRowBob = chatBridgeBob + "  agent  done  codex  native"
	codexSharedRowCid = chatBridgeCid + "  agent  done  other  terminal"
)

// codexShared is two agent replicas on one Codex app server and one member who
// writes to them.
type codexShared struct {
	cfg  string
	app  *codexSharedApp
	a, b *codexSharedReplica
	// a1 and b1 are the thread each replica's TUI started at startup.
	a1, b1 string
}

// startCodexShared starts everything the cases share and returns once each
// replica's TUI has started its first thread and both replicas attend.
//
// Replica A names its token on the proxy's command line. Replica B carries it
// as CMDMAN_CMD_ID, the way a replica cmdman runs does, so the proxy is run
// with both of the sources an identity comes from.
func startCodexShared(t *testing.T) codexShared {
	t.Helper()
	cfg := startChatDaemon(t)
	addr := freeAddr(t)
	app := startCodexSharedApp(t, "http://"+addr+"/mcp")
	startCodexSharedMCP(t, cfg, addr, app)
	attendChatBridges(t, cfg, "tok-cid")

	// One short directory for both proxies: each names its socket after its own
	// pid, and a unix socket path is capped well below what a test's own
	// temporary directory comes to.
	runtimeDir := shortSocketDir(t)
	a := startCodexSharedReplica(t, cfg, app, runtimeDir, []string{"--token", "tok-ana"})
	b := startCodexSharedReplica(t, cfg, app, runtimeDir, nil, "CMDMAN_CMD_ID=tok-bob")
	startup := codexSharedFrame(t, codexSharedLifecycleFixture, "thread/start", 0)
	c := codexShared{
		cfg: cfg,
		app: app,
		a:   a,
		b:   b,
		a1:  codexSharedStartThread(t, a, startup),
		b1:  codexSharedStartThread(t, b, startup),
	}
	waitChatAttendance(t, cfg, "tok-ana", codexSharedTimeout)
	waitChatAttendance(t, cfg, "tok-bob", codexSharedTimeout)
	return c
}

// mention has the writer mention to.
func (c codexShared) mention(t *testing.T, to string) {
	t.Helper()
	runChat(t, c.cfg, "tok-cid", "send", to, "the migration needs you")
}

// Two replicas share one app server and one MCP server, and each is a member of
// its own. The proxy in front of each TUI stamps its token on the thread the
// TUI starts, the thread's MCP session names that token, and the member the
// session attends as is the one the token names. A mention becomes a turn on
// the thread of the replica it names and on no other, and nothing is typed at
// a terminal.
func TestChatCodexShared_EachReplicaAttendsAsItsOwnMember(t *testing.T) {
	c := startCodexShared(t)
	waitCodexSharedRoster(t, c.cfg, "tok-cid",
		codexSharedRowAna, codexSharedRowBob, codexSharedRowCid)

	c.mention(t, chatBridgeAna)
	waitCodexSharedTurns(t, c.app, codexSharedTurn{c.a1, codexSharedArrival})

	c.mention(t, chatBridgeBob)
	waitCodexSharedTurns(t, c.app,
		codexSharedTurn{c.a1, codexSharedArrival},
		codexSharedTurn{c.b1, codexSharedArrival})

	if keys := stubSendKeys(t, c.cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}
}

// A replica is reached on the thread its person is in front of. A thread the
// TUI starts takes the very next mention, before anything was called on it. A
// thread the TUI resumes takes the mention back, although resuming a thread
// that is still loaded opens no session: the proxy calls the focus tool on it.
// The other replica is reached where it was all along.
func TestChatCodexShared_MentionsFollowTheThreadTheReplicaIsIn(t *testing.T) {
	c := startCodexShared(t)
	c.mention(t, chatBridgeAna)
	turns := []codexSharedTurn{{c.a1, codexSharedArrival}}
	waitCodexSharedTurns(t, c.app, turns...)

	// /new in A's TUI.
	a2 := codexSharedStartThread(t, c.a,
		codexSharedFrame(t, codexSharedLifecycleFixture, "thread/start", 2))
	c.mention(t, chatBridgeAna)
	turns = append(turns, codexSharedTurn{a2, codexSharedArrival})
	waitCodexSharedTurns(t, c.app, turns...)

	// /resume in A's TUI, choosing the first thread. The proxy sends its focus
	// call after the resume it stamped, and the TUI has its answer before the
	// call reaches the thread's session.
	codexSharedResume(t, c.a, c.a1)
	waitCodexSharedCall(t, c.app, codexSharedCall{c.a1, "crabswarm_focus"})
	c.mention(t, chatBridgeAna)
	turns = append(turns, codexSharedTurn{c.a1, codexSharedArrival})
	waitCodexSharedTurns(t, c.app, turns...)

	c.mention(t, chatBridgeBob)
	turns = append(turns, codexSharedTurn{c.b1, codexSharedArrival})
	waitCodexSharedTurns(t, c.app, turns...)
}

// A replica that stops takes its thread's session with it, and with its last
// session its member leaves the room. The other replica on the same servers
// stays, and is still reached on its thread.
func TestChatCodexShared_AStoppedReplicaLeavesAlone(t *testing.T) {
	c := startCodexShared(t)
	waitCodexSharedRoster(t, c.cfg, "tok-cid",
		codexSharedRowAna, codexSharedRowBob, codexSharedRowCid)

	if err := c.b.stop(); err != nil {
		t.Fatalf("replica B's proxy exited with %v, want its TUI's clean exit", err)
	}
	waitCodexSharedRoster(t, c.cfg, "tok-cid", codexSharedRowAna, codexSharedRowCid)
	if _, stderr, err := execChat(t, c.cfg, "tok-bob", "members"); err == nil {
		t.Errorf(
			"members as tok-bob succeeded after its replica stopped; want a refusal\nstderr:\n%s",
			stderr,
		)
	}

	c.mention(t, chatBridgeAna)
	waitCodexSharedTurns(t, c.app, codexSharedTurn{c.a1, codexSharedArrival})
}

// A thread started past the proxy carries no header, so its MCP session acts as
// nobody. A tool called on it says what is missing instead of acting as some
// member, no member attends for it, and the replicas are still reached on
// their own threads.
func TestChatCodexShared_AThreadStartedPastTheProxyActsAsNobody(t *testing.T) {
	c := startCodexShared(t)
	waitCodexSharedRoster(t, c.cfg, "tok-cid",
		codexSharedRowAna, codexSharedRowBob, codexSharedRowCid)

	direct := dialCodexSharedDirect(t, c.app)
	bare := codexSharedStartThread(t, direct,
		codexSharedFrame(t, codexSharedLifecycleFixture, "thread/start", 0))
	answer := direct.call(t, codexSharedOnThread(t,
		codexSharedFrame(t, codexSharedToolCallFixture, "mcpServer/tool/call", 0), bare))

	var res struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(answer, &res); err != nil {
		t.Fatalf("decode the tool call's answer %s: %v", answer, err)
	}
	if !res.Result.IsError || len(res.Result.Content) != 1 {
		t.Fatalf("chat_members on a thread with no header was answered %s, "+
			"want one text block reporting an error", answer)
	}
	text := res.Result.Content[0].Text
	for _, want := range []string{"no chat identity token", "X-Crabswarm-Token"} {
		if !strings.Contains(text, want) {
			t.Errorf("chat_members on a thread with no header answered %q, want it to say %q",
				text, want)
		}
	}

	roster := lines(runChat(t, c.cfg, "tok-cid", "members"))
	slices.Sort(roster)
	if want := []string{
		codexSharedRowAna,
		codexSharedRowBob,
		codexSharedRowCid,
	}; !slices.Equal(
		roster,
		want,
	) {
		t.Errorf("members as tok-cid = %q, want %q", roster, want)
	}

	c.mention(t, chatBridgeAna)
	waitCodexSharedTurns(t, c.app, codexSharedTurn{c.a1, codexSharedArrival})
	c.mention(t, chatBridgeBob)
	waitCodexSharedTurns(t, c.app,
		codexSharedTurn{c.a1, codexSharedArrival},
		codexSharedTurn{c.b1, codexSharedArrival})
}
