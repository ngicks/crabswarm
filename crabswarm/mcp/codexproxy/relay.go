package codexproxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/sync/errgroup"
)

// relay bridges every TUI connection on the proxy's socket to the app server.
// The app server speaks JSON-RPC over a WebSocket, one message per frame, so
// the relay terminates WebSocket on both sides and handles whole messages.
type relay struct {
	logger *slog.Logger
	// upstream is the app server's socket path.
	upstream string
	client   *http.Client
	// server is the MCP server name the token is stamped for.
	server string
	// token is the identity to stamp. Empty forwards every message unchanged.
	token string

	// mu orders a connection's bridge being started against serve waiting for
	// every bridge, so none starts after the wait began.
	mu     sync.Mutex
	closed bool
	conns  errgroup.Group
}

func newRelay(logger *slog.Logger, upstream, server string) *relay {
	return &relay{
		logger:   logger,
		upstream: upstream,
		// The transport dials the app server's socket for every address, so the
		// URL a bridge dials only has to be a valid one.
		client: &http.Client{Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", upstream)
			},
		}},
		server: server,
	}
}

// serve relays the connections accepted on ln until ctx ends, then waits for
// every bridge to end. It closes ln.
func (r *relay) serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			r.accept(ctx, w, req)
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	stop := context.AfterFunc(ctx, func() { _ = srv.Close() })
	defer stop()

	err := srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	} else {
		r.logger.Warn("codex proxy: the TUI can no longer connect", "err", err)
	}

	r.mu.Lock()
	r.closed = true
	r.mu.Unlock()
	_ = r.conns.Wait()

	if err != nil {
		return fmt.Errorf("serving the proxy socket: %w", err)
	}
	return nil
}

// accept upgrades one TUI connection and bridges it outside the handler: a
// hijacked connection is one the HTTP server no longer tracks, so serve waits
// for the bridge itself.
func (r *relay) accept(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	tui, err := websocket.Accept(w, req, nil)
	if err != nil {
		r.logger.Debug("codex proxy: refused a connection", "err", err)
		return
	}
	uri := req.URL.RequestURI()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		_ = tui.CloseNow()
		return
	}
	r.conns.Go(func() error {
		r.bridge(ctx, tui, uri)
		return nil
	})
}

// bridge connects tui to the app server at the path the TUI asked for and
// relays messages both ways until either side leaves.
func (r *relay) bridge(ctx context.Context, tui *websocket.Conn, uri string) {
	defer func() { _ = tui.CloseNow() }()
	// A resumed thread comes back with its whole history in one message, well
	// past the library's default limit of 32 KiB.
	tui.SetReadLimit(-1)

	server, _, err := websocket.Dial(ctx, "ws://localhost"+uri, &websocket.DialOptions{
		HTTPClient: r.client,
	})
	if err != nil {
		r.logger.Warn("codex proxy: cannot reach the app server",
			"socket", r.upstream, "err", err)
		_ = tui.Close(websocket.StatusBadGateway, "the app server is unreachable")
		return
	}
	defer func() { _ = server.CloseNow() }()
	server.SetReadLimit(-1)

	s := &stamper{logger: r.logger, server: r.server, token: r.token}
	var g errgroup.Group
	g.Go(func() error { return pipe(ctx, tui, server, s.rewrite) })
	g.Go(func() error { return pipe(ctx, server, tui, r.dropOwnAnswer) })
	_ = g.Wait()
}

// pipe relays the messages src sends to dst, each as transform says, until
// either connection ends. A close src receives is passed on with its status,
// which is what ends the pipe running the other way.
func pipe(
	ctx context.Context,
	src, dst *websocket.Conn,
	transform func(websocket.MessageType, []byte) [][]byte,
) error {
	for {
		typ, msg, err := src.Read(ctx)
		if err != nil {
			if ce, ok := errors.AsType[websocket.CloseError](err); ok {
				_ = dst.Close(ce.Code, ce.Reason)
			} else {
				_ = dst.CloseNow()
			}
			return err
		}
		for _, out := range transform(typ, msg) {
			if err := dst.Write(ctx, typ, out); err != nil {
				_ = src.CloseNow()
				return err
			}
		}
	}
}

// dropOwnAnswer passes on every app server message except the answers to the
// requests the proxy made itself, which the TUI never asked for.
func (r *relay) dropOwnAnswer(typ websocket.MessageType, msg []byte) [][]byte {
	if typ == websocket.MessageText && ownAnswer(msg) {
		r.logger.Debug("codex proxy: dropped the answer to its own request",
			"answer", string(msg))
		return nil
	}
	return [][]byte{msg}
}

// stamper rewrites the thread requests of one TUI connection.
type stamper struct {
	logger *slog.Logger
	server string
	token  string
	// focused counts the focus calls this connection made, which numbers the
	// next one's id.
	focused int
}

// rewrite returns what goes to the app server for one TUI message: the
// message itself, or a thread request stamped with the token. A stamped
// thread/resume is followed by a call to the focus tool on the resumed thread:
// resuming a thread that is still loaded keeps its MCP session as it was, so
// the MCP server has no other way to learn that this TUI switched to it.
func (s *stamper) rewrite(typ websocket.MessageType, msg []byte) [][]byte {
	if s.token == "" || typ != websocket.MessageText {
		return [][]byte{msg}
	}
	method := threadRequest(msg)
	if method == "" {
		return [][]byte{msg}
	}
	stamped, err := stamp(msg, tokenKey(s.server), s.token)
	if err != nil {
		s.logger.Warn("codex proxy: forwarding a thread request without the chat identity",
			"method", method, "err", err)
		return [][]byte{msg}
	}
	if method != methodThreadResume {
		return [][]byte{stamped}
	}
	threadId := resumedThread(msg)
	if threadId == "" {
		return [][]byte{stamped}
	}
	s.focused++
	focus, err := focusCall(s.focused, s.server, threadId)
	if err != nil {
		s.logger.Warn("codex proxy: not focusing the resumed thread",
			"thread", threadId, "err", err)
		return [][]byte{stamped}
	}
	return [][]byte{stamped, focus}
}
