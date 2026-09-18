package harnessctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// The fake app server the codex cases run against. It speaks the wire the real
// one speaks — an HTTP upgrade on a unix socket, then one JSON-RPC message per
// text frame, with no version marker on anything it sends — and it answers out
// of the recording under e2e/crabswarm/testdata/harness rather than out of a
// shape written here a second time.

const (
	codexClientFixture = "codex-app-server.client.ndjson"
	codexServerFixture = "codex-app-server.server.ndjson"
)

// codexRecording is one captured session, indexed by what a fake needs to
// replay it: the result each call was answered with, and every notification
// frame the server pushed.
type codexRecording struct {
	results       map[string]json.RawMessage
	notifications map[string][][]byte
}

// loadCodexRecording reads the two halves of the capture and pairs them by
// request id, which is the only thing that says which answer belongs to which
// call once the directions are split apart.
func loadCodexRecording(t *testing.T) codexRecording {
	t.Helper()

	methods := map[string]string{}
	for _, line := range codexFixtureLines(t, codexClientFixture) {
		var frame struct {
			Id     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		assert.NilError(t, json.Unmarshal(line, &frame))
		if len(frame.Id) > 0 {
			methods[string(frame.Id)] = frame.Method
		}
	}

	rec := codexRecording{
		results:       map[string]json.RawMessage{},
		notifications: map[string][][]byte{},
	}
	for _, line := range codexFixtureLines(t, codexServerFixture) {
		var frame struct {
			Id     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Result json.RawMessage `json:"result"`
		}
		assert.NilError(t, json.Unmarshal(line, &frame))
		switch {
		case len(frame.Id) > 0:
			method, ok := methods[string(frame.Id)]
			assert.Assert(t, ok, "the server answered id %s, which the client never sent",
				frame.Id)
			// The first answer to a method is the one replayed; a session that
			// called one twice was answered the same shape both times.
			if _, seen := rec.results[method]; !seen {
				rec.results[method] = frame.Result
			}
		case frame.Method != "":
			rec.notifications[frame.Method] = append(
				rec.notifications[frame.Method], slices.Clone(line))
		}
	}
	return rec
}

// codexFixtureLines reads one recording as its non-blank lines.
func codexFixtureLines(t *testing.T, name string) [][]byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtureDir, name))
	assert.NilError(t, err)
	var out [][]byte
	for line := range bytes.SplitSeq(b, []byte("\n")) {
		if len(bytes.TrimSpace(line)) > 0 {
			out = append(out, line)
		}
	}
	assert.Assert(t, len(out) > 0, "%s holds no frame", name)
	return out
}

// notification is the nth recorded frame of method, failing when the capture
// holds fewer.
func (r codexRecording) notification(t *testing.T, method string, n int) []byte {
	t.Helper()
	frames := r.notifications[method]
	assert.Assert(t, n < len(frames),
		"the recording holds %d %s frames, wanted the one at %d", len(frames), method, n)
	return frames[n]
}

// codexFake is an app server on a unix socket, answering out of a recording.
type codexFake struct {
	path string
	rec  codexRecording

	mu       sync.Mutex
	loaded   []string
	requests []codexRequest
	conns    []*websocket.Conn
}

// codexRequest is one call the fake was asked to answer.
type codexRequest struct {
	Method string
	Params json.RawMessage
}

// startCodexFake listens on a socket of its own and serves until the test ends.
//
// The socket lives under a short temporary directory rather than under the
// test's own: a unix socket path is capped well below what a directory named
// after a Go test case comes to.
func startCodexFake(t *testing.T, loaded ...string) *codexFake {
	t.Helper()

	dir, err := os.MkdirTemp("", "cx")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	f := &codexFake{
		path:   filepath.Join(dir, "app.sock"),
		rec:    loadCodexRecording(t),
		loaded: loaded,
	}
	ln, err := net.Listen("unix", f.path)
	assert.NilError(t, err)

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

func (f *codexFake) serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(codexReadLimit)
	f.joined(conn)
	defer f.left(conn)
	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			return
		}
		f.answer(conn, data)
	}
}

// answer records one incoming message and replies to it out of the recording.
//
// The reply carries no version marker and the recorded result verbatim, which
// is how the real app server writes one: a fake that wrote well-formed
// JSON-RPC would exercise a repair the client does not have to make.
func (f *codexFake) answer(conn *websocket.Conn, data []byte) {
	var frame struct {
		Id     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if json.Unmarshal(data, &frame) != nil {
		return
	}
	f.mu.Lock()
	f.requests = append(f.requests, codexRequest{Method: frame.Method, Params: frame.Params})
	loaded := slices.Clone(f.loaded)
	f.mu.Unlock()

	if len(frame.Id) == 0 {
		return
	}
	result, ok := f.rec.results[frame.Method]
	if !ok {
		result = json.RawMessage(`{}`)
	}
	if frame.Method == "thread/loaded/list" {
		ids, err := json.Marshal(loaded)
		if err != nil {
			return
		}
		result = json.RawMessage(`{"data":` + string(ids) + `,"nextCursor":null}`)
	}
	reply, err := json.Marshal(map[string]json.RawMessage{"id": frame.Id, "result": result})
	if err != nil {
		return
	}
	_ = conn.Write(context.Background(), websocket.MessageText, reply)
}

func (f *codexFake) joined(conn *websocket.Conn) {
	f.mu.Lock()
	f.conns = append(f.conns, conn)
	f.mu.Unlock()
}

func (f *codexFake) left(conn *websocket.Conn) {
	f.mu.Lock()
	f.conns = slices.DeleteFunc(f.conns, func(c *websocket.Conn) bool { return c == conn })
	f.mu.Unlock()
}

// push writes one frame to every connection, exactly as the bytes give it.
func (f *codexFake) push(frame []byte) {
	f.mu.Lock()
	conns := slices.Clone(f.conns)
	f.mu.Unlock()
	for _, conn := range conns {
		_ = conn.Write(context.Background(), websocket.MessageText, frame)
	}
}

// disconnect drops every connection the way an app server that went away
// does, without closing the listener: what connects next reaches the same fake.
func (f *codexFake) disconnect() {
	f.mu.Lock()
	conns := slices.Clone(f.conns)
	f.mu.Unlock()
	for _, conn := range conns {
		conn.CloseNow()
	}
}

// asked returns every call the fake has been handed, oldest first.
func (f *codexFake) asked() []codexRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

// methods returns the names of those calls, which is what most cases pin.
func (f *codexFake) methods() []string {
	var out []string
	for _, req := range f.asked() {
		out = append(out, req.Method)
	}
	return out
}

// paramsOf returns the parameters of the first call to method, failing when
// nothing called it.
func (f *codexFake) paramsOf(t *testing.T, method string) json.RawMessage {
	t.Helper()
	for _, req := range f.asked() {
		if req.Method == method {
			return req.Params
		}
	}
	t.Fatalf("nothing called %s; the fake was asked %v", method, f.methods())
	return nil
}

// waitAsked blocks until the fake has been handed method, so a case does not
// read the log before the call it is about has arrived.
func (f *codexFake) waitAsked(t *testing.T, method string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if slices.Contains(f.methods(), method) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("%s was never called; the fake was asked %v", method, f.methods())
}

// handshakes is how many times the fake has been introduced to, which is one
// per connection that reached it.
func (f *codexFake) handshakes() int {
	return count(f.methods(), "initialize")
}

// codexWaitingFrame is a thread status the recording does not hold: the
// captured session was never asked to approve anything, so the one active flag
// that matters here has to be written out. Everything else about the frame —
// the missing version marker, the stamp beside the parameters — is copied from
// the frames that were captured.
func codexWaitingFrame(threadId string) []byte {
	return fmt.Appendf(nil,
		`{"method":"thread/status/changed","params":{"threadId":%q,`+
			`"status":{"type":"active","activeFlags":["waitingOnApproval"]}},`+
			`"emittedAtMs":1789654916664}`, threadId)
}
