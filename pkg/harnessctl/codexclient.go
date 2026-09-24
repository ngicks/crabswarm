package harnessctl

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/creachadair/jrpc2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/internal/libver"
)

// The Codex app server is a JSON-RPC 2.0 peer reached over a WebSocket on a
// unix socket: the listener answers an ordinary HTTP upgrade, and every message
// after that is one text frame. This file is the client half of it — the dial,
// the framing, and the handful of calls a delivery and a state feed need.

// codexClientName is how this client names itself in the app server's
// handshake. The server records it in the user agent it answers with, so the
// name is what an operator reading a Codex session's log sees asking for turns.
const codexClientName = "crabswarm-mcp"

// codexReadLimit caps one frame. The default of the WebSocket library is 32
// KiB, and a resumed thread comes back with its whole history in one message,
// which passes that on a session of any length.
const codexReadLimit = 32 << 20

// codexWriteTimeout bounds one frame going out. Nothing on the far end of a
// unix socket takes this long unless it has stopped reading altogether, at
// which point the call it belongs to is over anyway.
const codexWriteTimeout = 10 * time.Second

// codexClient is one connection to a Codex app server.
type codexClient struct {
	conn *websocket.Conn
	rpc  *jrpc2.Client
	// stopped is closed when the JSON-RPC client stops, which is how a dropped
	// WebSocket reaches the loop holding the connection. stopErr is why; it is
	// written before the close, so a read after the close sees it.
	stopped chan struct{}
	stopErr error

	// resumedMu guards resumed, the thread this connection is subscribed to and
	// the one thread whose states the feed reports. The feed and a delivery both
	// ask for the subscription and either may be first, so it is recorded rather
	// than assumed.
	//
	// The lock keeps one read or write of the record whole and nothing more. A
	// caller that reads the record and then changes the subscription orders that
	// sequence itself, as [codex.binding] does.
	resumedMu sync.Mutex
	resumed   string
}

// dialCodex opens one connection to the app server listening on the unix socket
// at path and finishes the JSON-RPC handshake. onNotify, when not nil, is
// handed every notification the server pushes.
//
// The JSON-RPC client hands every frame it receives to a goroutine of its own,
// and that goroutine runs the handler while holding the client's lock. A
// handler that blocks stalls every other frame received and every call sent
// meanwhile. Callers hand the event on and return.
func dialCodex(
	ctx context.Context, path string, onNotify func(*jrpc2.Request),
) (*codexClient, error) {
	// The HTTP client dials the unix socket for every address, so the URL below
	// only has to be a syntactically valid one; the host in it is never
	// resolved.
	httpClient := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		},
	}}
	// The upgrade response carries no body of its own to close: on a successful
	// handshake the body is the hijacked connection, and conn owns it from here.
	//
	// ctx outlives the handshake, because the request it was made with is what
	// the transport keeps the connection open for. A caller that ends ctx ends
	// the connection with it, which is how a watcher stops.
	conn, _, err := websocket.Dial(ctx, "ws://localhost/", &websocket.DialOptions{
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, fmt.Errorf("dialing the codex app server on %s: %w", path, err)
	}
	conn.SetReadLimit(codexReadLimit)

	c := &codexClient{conn: conn, stopped: make(chan struct{})}
	c.rpc = jrpc2.NewClient(codexChannel{conn: conn}, &jrpc2.ClientOptions{
		OnNotify: onNotify,
		OnStop: func(_ *jrpc2.Client, err error) {
			c.stopErr = err
			close(c.stopped)
		},
	})
	return c, nil
}

// handshake introduces this client and declares it ready, which is the pair of
// messages every session in the recorded exchange opens with.
func (c *codexClient) handshake(ctx context.Context) error {
	_, err := c.rpc.Call(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    codexClientName,
			"title":   "crabswarm MCP server",
			"version": libver.Version,
		},
	})
	if err != nil {
		return fmt.Errorf("initializing the codex app server session: %w", err)
	}
	if err := c.rpc.Notify(ctx, "initialized", nil); err != nil {
		return fmt.Errorf("declaring the codex app server session ready: %w", err)
	}
	return nil
}

// loadedThreads lists the threads the app server holds in memory, as bare ids
// rather than thread objects. The order is none a caller can rely on: the
// recorded app server listed its threads oldest first.
func (c *codexClient) loadedThreads(ctx context.Context) ([]string, error) {
	var res struct {
		Data []string `json:"data"`
	}
	if err := c.rpc.CallResult(ctx, "thread/loaded/list", map[string]any{}, &res); err != nil {
		return nil, fmt.Errorf("listing the loaded codex threads: %w", err)
	}
	return res.Data, nil
}

// codexThread is what a delivery needs to know about one loaded thread: whose
// it is and when it was last used.
type codexThread struct {
	Id     string `json:"id"`
	Source string `json:"threadSource"`
	// RecencyAt is the Unix second the app server orders its threads by
	// recency with. The protocol allows null, which reads as 0.
	RecencyAt int64 `json:"recencyAt"`
}

// codexThreadStatus is what a thread is doing: `idle`, `active` with the flags
// saying what it waits on, or a state that makes no claim about a turn.
type codexThreadStatus struct {
	Type        string   `json:"type"`
	ActiveFlags []string `json:"activeFlags"`
}

// codexUserThread is the threadSource of a thread a person started. Any other
// value, and no value at all, marks a thread Codex runs by itself beside the
// session; the recorded helper answered `system`.
const codexUserThread = "user"

// readThread reads one loaded thread without its turns, which is all a binding
// needs and far less than the whole history a session of any length carries.
func (c *codexClient) readThread(ctx context.Context, threadId string) (codexThread, error) {
	var res struct {
		Thread codexThread `json:"thread"`
	}
	if err := c.rpc.CallResult(ctx, "thread/read", map[string]any{
		"threadId":     threadId,
		"includeTurns": false,
	}, &res); err != nil {
		return codexThread{}, fmt.Errorf("reading the codex thread %s: %w", threadId, err)
	}
	return res.Thread, nil
}

// subscribe subscribes this connection to a thread's events, once per thread,
// and returns what the thread was doing when the server answered.
//
// The answer is the whole thread, and only its status is kept. The call is made
// for the feed it opens, which is where the states this harness reports come
// from; the status is the state the feed starts that thread at. It is made once
// because a second one would ask a live session to be resumed again for no
// gain, and both the feed and a delivery ask for the same thread. A thread this
// connection already follows answers the zero status, which reads as no state.
//
// Resuming a thread moves the record to it. Whatever was resumed before keeps
// sending its events until [codexClient.unsubscribe] ends them, and they are
// read past because they name another thread.
func (c *codexClient) subscribe(ctx context.Context, threadId string) (codexThreadStatus, error) {
	if c.subscribed() == threadId {
		return codexThreadStatus{}, nil
	}
	// thread/resume answers with the thread under `thread`, and its status in
	// the shape thread/status/changed carries one: the recorded answer holds
	// `"status":{"type":"idle"}` there.
	var res struct {
		Thread struct {
			Status codexThreadStatus `json:"status"`
		} `json:"thread"`
	}
	if err := c.rpc.CallResult(
		ctx, "thread/resume", map[string]any{"threadId": threadId}, &res,
	); err != nil {
		return codexThreadStatus{}, fmt.Errorf("resuming the codex thread %s: %w", threadId, err)
	}
	c.resumedMu.Lock()
	c.resumed = threadId
	c.resumedMu.Unlock()
	return res.Thread.Status, nil
}

// unsubscribe ends this connection's subscription to a thread, which clears the
// record [codexClient.subscribe] keeps of it.
func (c *codexClient) unsubscribe(ctx context.Context, threadId string) error {
	if _, err := c.rpc.Call(
		ctx, "thread/unsubscribe", map[string]any{"threadId": threadId},
	); err != nil {
		return fmt.Errorf("unsubscribing from the codex thread %s: %w", threadId, err)
	}
	c.resumedMu.Lock()
	if c.resumed == threadId {
		c.resumed = ""
	}
	c.resumedMu.Unlock()
	return nil
}

// subscribed is the thread this connection is subscribed to, or "" for none.
func (c *codexClient) subscribed() string {
	c.resumedMu.Lock()
	defer c.resumedMu.Unlock()
	return c.resumed
}

// follows reports whether threadId names the thread this connection is
// subscribed to. A notification that names no thread is about none of them.
func (c *codexClient) follows(threadId string) bool {
	return threadId != "" && threadId == c.subscribed()
}

// startTurn hands text to a thread as one user input, which is a turn the agent
// takes as if the line had been typed at it.
func (c *codexClient) startTurn(ctx context.Context, threadId, text string) error {
	_, err := c.rpc.Call(ctx, "turn/start", map[string]any{
		"threadId": threadId,
		"input":    []any{map[string]any{"type": "text", "text": text}},
	})
	if err != nil {
		return fmt.Errorf("starting a codex turn on thread %s: %w", threadId, err)
	}
	return nil
}

// close ends the connection. Closing the JSON-RPC client first lets the pending
// calls fail with its own reason rather than with a read error off a socket
// that went away underneath them.
func (c *codexClient) close() {
	_ = c.rpc.Close()
	_ = c.conn.Close(websocket.StatusNormalClosure, "")
}

// dropped is closed once the connection is no longer carrying messages, for
// whatever reason.
func (c *codexClient) dropped() <-chan struct{} { return c.stopped }

// codexChannel carries JSON-RPC messages as WebSocket text frames, one message
// per frame, which is the framing the app server speaks.
//
// It takes no context because the JSON-RPC library's channel does not offer
// one. Sending is bounded by [codexWriteTimeout] and receiving is unbounded:
// the loop reading it is waiting for the next server event, and closing the
// connection is what ends that wait.
type codexChannel struct {
	conn *websocket.Conn
}

func (c codexChannel) Send(msg []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), codexWriteTimeout)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageText, msg)
}

func (c codexChannel) Recv() ([]byte, error) {
	_, data, err := c.conn.Read(context.Background())
	if err != nil {
		return nil, err
	}
	return codexFrame(data), nil
}

func (c codexChannel) Close() error {
	return c.conn.Close(websocket.StatusNormalClosure, "")
}

// codexFrame rewrites one message from the app server into the strict JSON-RPC
// 2.0 shape the library parses.
//
// The app server leaves the version marker off everything it sends and hangs an
// emittedAtMs stamp on its notifications. The library treats a missing marker
// and an undefined member as protocol errors, and a reply carrying either comes
// back to the caller as a failed call rather than as its result — so the repair
// belongs here, at the point where the two spellings meet, rather than in every
// call. Anything this cannot read is passed through for the library to refuse
// with its own words.
func codexFrame(raw []byte) []byte {
	var msg map[string]json.RawMessage
	if json.Unmarshal(raw, &msg) != nil {
		return raw
	}
	kept := make(map[string]json.RawMessage, len(msg))
	for name, value := range msg {
		// The version marker is written below whether or not one arrived, since
		// the only one the protocol allows is the one being written.
		switch name {
		case "id", "method", "params", "result", "error":
			kept[name] = value
		}
	}
	kept["jsonrpc"] = json.RawMessage(`"2.0"`)
	repaired, err := json.Marshal(kept)
	if err != nil {
		return raw
	}
	return repaired
}

// The notifications the app server reports a session's own progress through.
// Everything else it pushes is about what the agent is doing rather than about
// whether it is busy.
const (
	codexStatusChanged = "thread/status/changed"
	codexTurnCompleted = "turn/completed"
)

// codexThreadStarted is the notification a thread being started pushes to
// every connection, subscribed or not, which is how a feed hears that a TUI
// attached or was replaced.
//
// Its parameters are never read. No such frame was recorded, and the feed
// binds again from the loaded threads anyway, so the notification is only a
// signal that the answer may have changed.
const codexThreadStarted = "thread/started"

// codexWaitingFlags are the reasons an active thread is not working but
// waiting on the person in front of it. The protocol's other flags, and any it
// gains later, read as work in progress.
var codexWaitingFlags = []string{"waitingOnApproval", "waitingOnUserInput"}

// state reads what a thread status says about the agent, and reports whether it
// said anything at all.
//
// A status this does not recognise — one that was never loaded, one the server
// has given up on, the zero status of a thread nobody read — leaves the member
// in whatever it last reported, since none of them is a claim about a turn.
func (s codexThreadStatus) state() (chatv1.HarnessState, bool) {
	switch s.Type {
	case "idle":
		return chatv1.HarnessState_HARNESS_STATE_DONE, true
	case "active":
		for _, flag := range s.ActiveFlags {
			if slices.Contains(codexWaitingFlags, flag) {
				return chatv1.HarnessState_HARNESS_STATE_WAITING, true
			}
		}
		return chatv1.HarnessState_HARNESS_STATE_WORKING, true
	}
	return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, false
}

// codexState reads what a server notification says about the agent and the
// thread it says it about, and reports whether it said anything at all.
//
// Which thread that is matters, because a helper Codex runs beside the session
// flips between active and idle on its own; the caller reads past a state about
// any thread but the one it follows. The thread is handed back rather than
// checked here: this runs as the notification arrives, and a notification that
// arrived while the thread was still being bound would be checked against no
// thread and lost.
//
// A turn that ended is done however it ended: interrupted and failed both leave
// the agent back at its prompt, which is the only thing a room needs to know
// before it interrupts.
func codexState(req *jrpc2.Request) (string, chatv1.HarnessState, bool) {
	switch req.Method() {
	case codexStatusChanged:
		var params struct {
			ThreadId string            `json:"threadId"`
			Status   codexThreadStatus `json:"status"`
		}
		if req.UnmarshalParams(&params) != nil {
			break
		}
		state, ok := params.Status.state()
		return params.ThreadId, state, ok
	case codexTurnCompleted:
		var params struct {
			ThreadId string `json:"threadId"`
			Turn     struct {
				Status string `json:"status"`
			} `json:"turn"`
		}
		if req.UnmarshalParams(&params) != nil {
			break
		}
		switch params.Turn.Status {
		case "completed", "interrupted", "failed":
			return params.ThreadId, chatv1.HarnessState_HARNESS_STATE_DONE, true
		}
	}
	return "", chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, false
}
