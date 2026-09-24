package harnessctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
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

	// The helpers pair read every loaded thread without its turns, which is the
	// thread/read a binding makes.
	codexHelpersClientFixture = "codex-app-server-helpers.client.ndjson"
	codexHelpersServerFixture = "codex-app-server-helpers.server.ndjson"
)

// codexRecording is one captured session, indexed by what a fake needs to
// replay it: the result each call was answered with, and every notification
// frame the server pushed.
type codexRecording struct {
	results       map[string]json.RawMessage
	notifications map[string][][]byte
}

// loadCodexRecording reads the two halves of a capture and pairs them by
// request id, which is the only thing that says which answer belongs to which
// call once the directions are split apart.
func loadCodexRecording(t *testing.T, clientFixture, serverFixture string) codexRecording {
	t.Helper()

	methods := map[string]string{}
	for _, line := range codexFixtureLines(t, clientFixture) {
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
	for _, line := range codexFixtureLines(t, serverFixture) {
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
	// threadRead is the answer every thread/read is re-addressed from. It comes
	// from the helpers recording, since the main one read its thread with every
	// turn in it, which is not the call a binding makes.
	threadRead json.RawMessage

	mu     sync.Mutex
	loaded []string
	// sources is the threadSource each loaded thread reports, by id. A thread
	// missing here is a person's, which is what the recorded answer says.
	sources map[string]string
	// recency is the recencyAt each loaded thread reports, by id. A thread
	// missing here keeps the recorded one, so two such threads tie.
	recency map[string]int64
	// resumeStatus is the thread status each thread's resume answer carries, as
	// JSON, by id. A thread missing here keeps the recorded one, which is idle.
	resumeStatus map[string]string
	trail        []codexRequest
	conns        []*websocket.Conn
}

// codexRequest is one message in the fake's trail: a call it was asked to
// answer, or a notification it pushed.
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

	helpers := loadCodexRecording(t, codexHelpersClientFixture, codexHelpersServerFixture)
	threadRead, ok := helpers.results["thread/read"]
	assert.Assert(t, ok, "%s answers no thread/read", codexHelpersServerFixture)

	f := &codexFake{
		path:       filepath.Join(dir, "app.sock"),
		rec:        loadCodexRecording(t, codexClientFixture, codexServerFixture),
		threadRead: threadRead,
		loaded:     loaded,
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
	f.trail = append(f.trail, codexRequest{Method: frame.Method, Params: frame.Params})
	loaded := slices.Clone(f.loaded)
	sources := maps.Clone(f.sources)
	recency := maps.Clone(f.recency)
	resumeStatus := maps.Clone(f.resumeStatus)
	f.mu.Unlock()

	if len(frame.Id) == 0 {
		return
	}
	result, ok := f.rec.results[frame.Method]
	if !ok {
		result = json.RawMessage(`{}`)
	}
	switch frame.Method {
	case "thread/loaded/list":
		ids, err := json.Marshal(loaded)
		if err != nil {
			return
		}
		result = json.RawMessage(`{"data":` + string(ids) + `,"nextCursor":null}`)
	case "thread/read":
		answer, err := threadReadAnswer(f.threadRead, frame.Params, sources, recency)
		if err != nil {
			return
		}
		result = answer
	case "thread/resume":
		answer, err := resumeAnswer(result, frame.Params, resumeStatus)
		if err != nil {
			return
		}
		result = answer
	}
	reply, err := json.Marshal(map[string]json.RawMessage{"id": frame.Id, "result": result})
	if err != nil {
		return
	}
	_ = conn.Write(context.Background(), websocket.MessageText, reply)
}

// threadReadAnswer is the recorded thread/read answer re-addressed to the thread
// params asked about, with the source and recency that thread was given. The
// recorded answer is about a person's thread; a case about the helpers Codex
// loads beside a session, or about two of a person's threads, needs the same
// shape to say something else.
func threadReadAnswer(
	recorded, params json.RawMessage, sources map[string]string, recency map[string]int64,
) (json.RawMessage, error) {
	var asked struct {
		ThreadId string `json:"threadId"`
	}
	if err := json.Unmarshal(params, &asked); err != nil {
		return nil, err
	}
	var answer struct {
		Thread map[string]json.RawMessage `json:"thread"`
	}
	if err := json.Unmarshal(recorded, &answer); err != nil {
		return nil, err
	}
	source := codexUserThread
	if s, ok := sources[asked.ThreadId]; ok {
		source = s
	}
	var err error
	if answer.Thread["id"], err = json.Marshal(asked.ThreadId); err != nil {
		return nil, err
	}
	if answer.Thread["threadSource"], err = json.Marshal(source); err != nil {
		return nil, err
	}
	if at, ok := recency[asked.ThreadId]; ok {
		if answer.Thread["recencyAt"], err = json.Marshal(at); err != nil {
			return nil, err
		}
	}
	return json.Marshal(answer)
}

// resumeAnswer is the recorded thread/resume answer re-addressed to the thread
// params asked about, with the status that thread was given. The recording
// resumed one thread, idle; a case about the feed moving from one thread to
// another needs each thread to answer for itself.
func resumeAnswer(
	recorded, params json.RawMessage, statuses map[string]string,
) (json.RawMessage, error) {
	var asked struct {
		ThreadId string `json:"threadId"`
	}
	if err := json.Unmarshal(params, &asked); err != nil {
		return nil, err
	}
	var answer map[string]json.RawMessage
	if err := json.Unmarshal(recorded, &answer); err != nil {
		return nil, err
	}
	var thread map[string]json.RawMessage
	if err := json.Unmarshal(answer["thread"], &thread); err != nil {
		return nil, err
	}
	var err error
	if thread["id"], err = json.Marshal(asked.ThreadId); err != nil {
		return nil, err
	}
	if status, ok := statuses[asked.ThreadId]; ok {
		thread["status"] = json.RawMessage(status)
	}
	if answer["thread"], err = json.Marshal(thread); err != nil {
		return nil, err
	}
	return json.Marshal(answer)
}

// update changes what the fake holds while connections are open to it, under
// the lock its answers read it with.
func (f *codexFake) update(change func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	change()
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

// push writes one frame to every connection, exactly as the bytes give it, and
// enters it in the trail before any of them can answer it.
func (f *codexFake) push(frame []byte) {
	var note struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(frame, &note)
	f.mu.Lock()
	f.trail = append(f.trail, codexRequest{Method: note.Method, Params: note.Params})
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

// asked returns the trail: every call the fake has been handed and every
// notification it pushed, oldest first.
func (f *codexFake) asked() []codexRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.trail)
}

// methods returns the names in the trail, which is what most cases pin.
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

// threadsOf returns the thread each call to method named, oldest first.
func (f *codexFake) threadsOf(t *testing.T, method string) []string {
	t.Helper()
	var out []string
	for _, req := range f.asked() {
		if req.Method != method {
			continue
		}
		var params struct {
			ThreadId string `json:"threadId"`
		}
		assert.NilError(t, json.Unmarshal(req.Params, &params))
		out = append(out, params.ThreadId)
	}
	return out
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

// codexThreadStartedFrame announces a thread the way the app server does when
// a TUI starts one. No such frame was recorded, so it is written out here in
// the captured notifications' form: no version marker, a stamp beside the
// parameters. The feed reads nothing inside the parameters, so the thread
// object under them only names the thread.
func codexThreadStartedFrame(threadId string) []byte {
	return fmt.Appendf(nil,
		`{"method":"thread/started","params":{"thread":{"id":%q}},`+
			`"emittedAtMs":1789654916664}`, threadId)
}

// codexFrameAbout is a recorded notification frame said about threadId instead,
// with everything else in it as it was captured.
func codexFrameAbout(t *testing.T, frame []byte, threadId string) []byte {
	t.Helper()
	var msg map[string]json.RawMessage
	assert.NilError(t, json.Unmarshal(frame, &msg))
	var params map[string]json.RawMessage
	assert.NilError(t, json.Unmarshal(msg["params"], &params))
	var err error
	params["threadId"], err = json.Marshal(threadId)
	assert.NilError(t, err)
	msg["params"], err = json.Marshal(params)
	assert.NilError(t, err)
	out, err := json.Marshal(msg)
	assert.NilError(t, err)
	return out
}
