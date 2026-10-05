package codexproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
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

const fixtureDir = "../../../e2e/crabswarm/testdata/harness"

// fixtureLines reads one recording as its non-blank lines.
func fixtureLines(t *testing.T, name string) [][]byte {
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

// shortDir is a fresh directory short enough to hold a unix socket: a
// directory named after a Go test case passes the length a socket path is
// capped at.
func shortDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cxp")
	assert.NilError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func testLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(t.Output(), &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// frame is one WebSocket message as it crossed the wire.
type frame struct {
	Type websocket.MessageType
	Data []byte
}

// fakeAppServer is an app server on a unix socket that records every message
// it is sent and answers every request with {"answered": <method>}.
type fakeAppServer struct {
	path string

	mu    sync.Mutex
	got   []frame
	conns []*websocket.Conn
	// ended is how each connection's read loop ended.
	ended []error
}

func startFakeAppServer(t *testing.T) *fakeAppServer {
	t.Helper()
	f := &fakeAppServer{path: filepath.Join(shortDir(t), "app.sock")}
	ln, err := net.Listen("unix", f.path)
	assert.NilError(t, err)
	srv := &http.Server{Handler: http.HandlerFunc(f.serve), ReadHeaderTimeout: 10 * time.Second}
	var serving errgroup.Group
	serving.Go(func() error { return srv.Serve(ln) })
	t.Cleanup(func() {
		_ = srv.Close()
		_ = serving.Wait()
		for _, c := range f.connections() {
			_ = c.CloseNow()
		}
	})
	return f
}

func (f *fakeAppServer) serve(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(-1)
	f.mu.Lock()
	f.conns = append(f.conns, conn)
	f.mu.Unlock()
	for {
		typ, data, err := conn.Read(context.Background())
		if err != nil {
			f.mu.Lock()
			f.ended = append(f.ended, err)
			f.mu.Unlock()
			return
		}
		f.mu.Lock()
		f.got = append(f.got, frame{Type: typ, Data: data})
		f.mu.Unlock()
		if typ != websocket.MessageText {
			continue
		}
		var req struct {
			Id     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(data, &req) != nil || len(req.Id) == 0 || req.Method == "" {
			continue
		}
		_ = conn.Write(context.Background(), websocket.MessageText, answerTo(req.Id, req.Method))
	}
}

// answerTo is the answer the fake writes to the request id calling method.
func answerTo(id json.RawMessage, method string) []byte {
	b, err := json.Marshal(
		map[string]any{"id": id, "result": map[string]string{"answered": method}},
	)
	if err != nil {
		panic(err)
	}
	return b
}

// connections is every connection the fake accepted.
func (f *fakeAppServer) connections() []*websocket.Conn {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.conns)
}

// push writes one message to every connection.
func (f *fakeAppServer) push(t *testing.T, typ websocket.MessageType, data []byte) {
	t.Helper()
	conns := f.connections()
	assert.Assert(t, len(conns) > 0, "nothing is connected to the fake app server")
	for _, c := range conns {
		assert.NilError(t, c.Write(t.Context(), typ, data))
	}
}

// waitGot blocks until the fake has been sent n messages and returns them.
func (f *fakeAppServer) waitGot(t *testing.T, n int) []frame {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		f.mu.Lock()
		got := slices.Clone(f.got)
		f.mu.Unlock()
		if len(got) >= n || time.Now().After(deadline) {
			assert.Equal(t, len(got), n, "the fake app server was sent %d messages", len(got))
			return got
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// waitEnded blocks until a connection to the fake ended and returns why.
func (f *fakeAppServer) waitEnded(t *testing.T) error {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		ended := slices.Clone(f.ended)
		f.mu.Unlock()
		if len(ended) > 0 {
			return ended[0]
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("no connection to the fake app server ended")
	return nil
}

// dialUnix opens a WebSocket to the listener on the unix socket at path, the
// way a remote Codex TUI does.
func dialUnix(t *testing.T, path string) *websocket.Conn {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		},
	}}
	conn, _, err := websocket.Dial(t.Context(), "ws://localhost/", &websocket.DialOptions{
		HTTPClient: client,
	})
	assert.NilError(t, err)
	conn.SetReadLimit(-1)
	t.Cleanup(func() { _ = conn.CloseNow() })
	return conn
}

func readFrame(t *testing.T, conn *websocket.Conn) frame {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	typ, data, err := conn.Read(ctx)
	assert.NilError(t, err)
	return frame{Type: typ, Data: data}
}

func writeText(t *testing.T, conn *websocket.Conn, data []byte) {
	t.Helper()
	assert.NilError(t, conn.Write(t.Context(), websocket.MessageText, data))
}

// requestId is the id member of msg, as its JSON.
func requestId(t *testing.T, msg []byte) string {
	t.Helper()
	var req struct {
		Id json.RawMessage `json:"id"`
	}
	assert.NilError(t, json.Unmarshal(msg, &req))
	return string(req.Id)
}

func methodOf(t *testing.T, msg []byte) string {
	t.Helper()
	var req struct {
		Method string `json:"method"`
	}
	assert.NilError(t, json.Unmarshal(msg, &req))
	return req.Method
}

func decoded(t *testing.T, msg []byte) map[string]any {
	t.Helper()
	var v map[string]any
	assert.NilError(t, json.Unmarshal(msg, &v))
	return v
}

// withToken is the request msg as the proxy stamps it: params.config gains
// the token key, created with params and config when the TUI sent neither.
func withToken(t *testing.T, msg []byte, token string) map[string]any {
	t.Helper()
	req := decoded(t, msg)
	params, _ := req["params"].(map[string]any)
	if params == nil {
		params = map[string]any{}
		req["params"] = params
	}
	config, _ := params["config"].(map[string]any)
	if config == nil {
		config = map[string]any{}
		params["config"] = config
	}
	config["mcp_servers.crabswarm-mcp.http_headers.X-Crabswarm-Token"] = token
	return req
}
