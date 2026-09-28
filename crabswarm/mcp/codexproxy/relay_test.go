package codexproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// startRelay serves a relay to upstream on a socket of its own until the test
// ends, and returns the socket's path.
func startRelay(t *testing.T, upstream, token string) string {
	t.Helper()
	r := newRelay(testLogger(t), upstream, DefaultServer)
	r.token = token
	sock := filepath.Join(shortDir(t), "proxy.sock")
	ln, err := net.Listen("unix", sock)
	assert.NilError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	var serving errgroup.Group
	serving.Go(func() error { return r.serve(ctx, ln) })
	t.Cleanup(func() {
		cancel()
		assert.NilError(t, serving.Wait())
	})
	return sock
}

// Every message that is no thread request crosses the proxy byte for byte,
// text and binary, in both directions, and a close either side sends reaches
// the other with its status.
func TestRelay_ForwardsEveryOtherMessageUnchanged(t *testing.T) {
	app := startFakeAppServer(t)
	tui := dialUnix(t, startRelay(t, app.path, "tokA"))

	initialize := fixtureLines(t, "codex-app-server.client.ndjson")[0]
	writeText(t, tui, initialize)
	got := app.waitGot(t, 1)
	assert.DeepEqual(t, got[0].Data, initialize)
	assert.DeepEqual(t, readFrame(t, tui),
		frame{Type: websocket.MessageText, Data: answerTo(json.RawMessage(`1`), "initialize")})

	var notification []byte
	for _, line := range fixtureLines(t, "codex-app-server.server.ndjson") {
		var msg struct {
			Id     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		assert.NilError(t, json.Unmarshal(line, &msg))
		if msg.Method != "" && len(msg.Id) == 0 {
			notification = line
			break
		}
	}
	assert.Assert(t, notification != nil, "the recording holds no notification")
	app.push(t, websocket.MessageText, notification)
	assert.DeepEqual(t, readFrame(t, tui), frame{Type: websocket.MessageText, Data: notification})

	binary := []byte{0, 1, 2, 0xff}
	assert.NilError(t, tui.Write(t.Context(), websocket.MessageBinary, binary))
	got = app.waitGot(t, 2)
	assert.DeepEqual(t, got[1], frame{Type: websocket.MessageBinary, Data: binary})
	app.push(t, websocket.MessageBinary, binary)
	assert.DeepEqual(t, readFrame(t, tui), frame{Type: websocket.MessageBinary, Data: binary})

	start := time.Now()
	assert.NilError(t, tui.Close(websocket.StatusNormalClosure, "bye"))
	ce, ok := errors.AsType[websocket.CloseError](app.waitEnded(t))
	assert.Assert(t, ok, "the fake's connection ended without a close")
	assert.Equal(t, ce.Code, websocket.StatusNormalClosure)
	assert.Equal(t, ce.Reason, "bye")
	assert.Assert(t, time.Since(start) < 3*time.Second,
		"the close took %s to reach the app server", time.Since(start))
}

// A TUI's thread lifecycle reaches the app server with the token on every
// thread/start, thread/resume and thread/fork, and on nothing else. Each
// resume is followed by a call to the focus tool on the resumed thread, whose
// answer the TUI never sees.
func TestRelay_StampsThreadRequestsAndFocusesAResume(t *testing.T) {
	app := startFakeAppServer(t)
	tui := dialUnix(t, startRelay(t, app.path, "tokA"))

	lifecycle := fixtureLines(t, "codex-tui-thread-lifecycle.client.ndjson")
	// A second resume on the same connection, as a recorder sent it: no config
	// at all, which the proxy has to create.
	var resume []byte
	for _, line := range fixtureLines(t, "codex-app-server-resume-loaded.client.ndjson") {
		if methodOf(t, line) == "thread/resume" {
			resume = line
		}
	}
	assert.Assert(t, resume != nil, "the recording holds no thread/resume")
	sent := append(slices.Clone(lifecycle), resume)

	methods := map[string]int{}
	for _, line := range sent {
		writeText(t, tui, line)
		methods[methodOf(t, line)]++
	}
	assert.Equal(t, methods["thread/start"], 3)
	assert.Equal(t, methods["thread/resume"], 2)
	assert.Equal(t, methods["thread/fork"], 1)

	// One call per resume above, naming the thread it resumed.
	focusCalls := []string{
		`{"jsonrpc":"2.0","id":"crabswarm-proxy-1","method":"mcpServer/tool/call",` +
			`"params":{"server":"crabswarm-mcp","threadId":"01a0df46-7c72-7691-85c9-6efb2f88f057",` +
			`"tool":"crabswarm_focus","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":"crabswarm-proxy-2","method":"mcpServer/tool/call",` +
			`"params":{"server":"crabswarm-mcp","threadId":"01a0e15d-b576-7410-a249-a1300176a36d",` +
			`"tool":"crabswarm_focus","arguments":{}}}`,
	}

	got := app.waitGot(t, len(sent)+len(focusCalls))
	i := 0
	for _, line := range sent {
		switch methodOf(t, line) {
		case "thread/start", "thread/fork":
			assert.DeepEqual(t, decoded(t, got[i].Data), withToken(t, line, "tokA"))
		case "thread/resume":
			assert.DeepEqual(t, decoded(t, got[i].Data), withToken(t, line, "tokA"))
			i++
			assert.Equal(t, string(got[i].Data), focusCalls[0])
			focusCalls = focusCalls[1:]
		default:
			assert.DeepEqual(t, got[i].Data, line)
		}
		i++
	}

	// The fake answered the focus calls too, right after each resume. The TUI
	// is handed the answers to its own requests only, in the order it sent
	// them.
	for _, line := range sent {
		answer := readFrame(t, tui)
		assert.Equal(t, requestId(t, answer.Data), requestId(t, line))
	}
}

// The answer to the proxy's own focus call never reaches the TUI. A refused one
// is a warning naming the server the call was made for, which is how a
// --server-name naming the wrong server shows; an answered one is not.
func TestRelay_WarnsOfARefusedFocusCall(t *testing.T) {
	var logs bytes.Buffer
	r := newRelay(slog.New(slog.NewTextHandler(&logs, nil)), "app.sock", "not-crabswarm")

	answered := []byte(`{"id":"crabswarm-proxy-1","result":{"content":[]}}`)
	assert.Equal(t, len(r.dropOwnAnswer(websocket.MessageText, answered)), 0)
	assert.Equal(t, logs.String(), "")

	// What Codex 0.157.1 answers a call naming a server it does not declare.
	refused := []byte(`{"error":{"code":-32603,"message":"unknown MCP server 'not-crabswarm'"},` +
		`"id":"crabswarm-proxy-2"}`)
	assert.Equal(t, len(r.dropOwnAnswer(websocket.MessageText, refused)), 0)
	line := logs.String()
	for _, want := range []string{"level=WARN", "server=not-crabswarm", "unknown MCP server"} {
		assert.Assert(t, strings.Contains(line, want), "the log %q lacks %q", line, want)
	}
}

// With no token the proxy is a plain relay: the thread lifecycle arrives byte
// for byte, and no focus call is made.
func TestRelay_WithoutATokenForwardsThreadRequestsUnchanged(t *testing.T) {
	app := startFakeAppServer(t)
	tui := dialUnix(t, startRelay(t, app.path, ""))

	lifecycle := fixtureLines(t, "codex-tui-thread-lifecycle.client.ndjson")
	for _, line := range lifecycle {
		writeText(t, tui, line)
	}
	for _, line := range lifecycle {
		assert.Equal(t, requestId(t, readFrame(t, tui).Data), requestId(t, line))
	}
	got := app.waitGot(t, len(lifecycle))
	for i, line := range lifecycle {
		assert.DeepEqual(t, got[i], frame{Type: websocket.MessageText, Data: line})
	}
}

// An app server that closes the connection has its close reach the TUI with
// the status and reason it gave.
func TestRelay_PassesTheAppServersCloseToTheTUI(t *testing.T) {
	app := startFakeAppServer(t)
	tui := dialUnix(t, startRelay(t, app.path, "tokA"))

	writeText(t, tui, fixtureLines(t, "codex-app-server.client.ndjson")[0])
	readFrame(t, tui)

	conns := app.connections()
	assert.Equal(t, len(conns), 1)
	assert.NilError(t, conns[0].Close(websocket.StatusGoingAway, "shutting down"))

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, _, err := tui.Read(ctx)
	ce, ok := errors.AsType[websocket.CloseError](err)
	assert.Assert(t, ok, "the TUI's connection ended without a close: %v", err)
	assert.Equal(t, ce.Code, websocket.StatusGoingAway)
	assert.Equal(t, ce.Reason, "shutting down")
}

// A TUI whose app server is not there is told so with a close, rather than
// left connected to nothing.
func TestRelay_ClosesTheTUIWhenTheAppServerIsUnreachable(t *testing.T) {
	missing := filepath.Join(shortDir(t), "none.sock")
	tui := dialUnix(t, startRelay(t, missing, "tokA"))

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, _, err := tui.Read(ctx)
	assert.Equal(t, websocket.CloseStatus(err), websocket.StatusBadGateway)
}
