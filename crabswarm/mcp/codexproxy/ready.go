package codexproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/coder/websocket"

	"github.com/ngicks/crabswarm/internal/libver"
)

// readyInterval is the pause between two attempts of [relay.waitReady].
const readyInterval = time.Second

// The ids the readiness probe numbers its two requests with. The probe runs on
// a connection of its own, so they cannot meet the TUI's.
const (
	readyInitializeId  = "1"
	readyAccountReadId = "2"
)

// waitReady blocks until the app server answers account/read with a result,
// for at most timeout. A remote Codex TUI calls account/read while it boots and
// exits when that call fails, which an app server still discovering its
// workspace routing answers with an error. Running out of time is no reason to
// keep the TUI from starting, so waitReady only warns then; the TUI reports
// whatever the app server still refuses.
//
// It returns early when ctx ends, and the caller then has nothing to start.
func (r *relay) waitReady(ctx context.Context, timeout time.Duration) {
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	warned := false
	for {
		err := r.readAccount(deadline)
		if err == nil {
			return
		}
		if ctx.Err() != nil {
			return
		}
		if !warned {
			r.logger.Warn("codex proxy: waiting for the app server to answer account/read",
				"socket", r.upstream, "timeout", timeout, "err", err)
			warned = true
		} else {
			r.logger.Debug("codex proxy: the app server is not ready yet", "err", err)
		}
		select {
		case <-deadline.Done():
			if ctx.Err() == nil {
				r.logger.Warn(
					"codex proxy: starting the TUI before the app server answered account/read",
					"timeout",
					timeout,
					"err",
					err,
				)
			}
			return
		case <-time.After(readyInterval):
		}
	}
}

// readAccount opens a connection to the app server, opens a session on it the
// way the TUI does and calls account/read. It returns nil once that call
// answers with a result.
func (r *relay) readAccount(ctx context.Context) error {
	conn, _, err := websocket.Dial(ctx, "ws://localhost/", &websocket.DialOptions{
		HTTPClient: r.client,
	})
	if err != nil {
		return fmt.Errorf("dialing the app server: %w", err)
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(-1)

	initialize, err := marshal(map[string]any{
		"id":     json.RawMessage(readyInitializeId),
		"method": "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{
				"name":    "crabswarm-codex-proxy",
				"title":   "crabswarm codex proxy",
				"version": libver.Version,
			},
		},
	})
	if err != nil {
		return err
	}
	for _, msg := range [][]byte{
		initialize,
		[]byte(`{"method":"initialized"}`),
		[]byte(`{"id":` + readyAccountReadId + `,"method":"account/read","params":{}}`),
	} {
		if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
			return fmt.Errorf("writing to the app server: %w", err)
		}
	}

	for {
		typ, msg, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("reading from the app server: %w", err)
		}
		if typ != websocket.MessageText {
			continue
		}
		var answer struct {
			Id     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if json.Unmarshal(msg, &answer) != nil || answer.Method != "" {
			continue
		}
		var method string
		switch {
		case bytes.Equal(answer.Id, []byte(readyInitializeId)):
			method = "initialize"
		case bytes.Equal(answer.Id, []byte(readyAccountReadId)):
			method = "account/read"
		default:
			continue
		}
		if failed := answerError(msg); failed != nil {
			return fmt.Errorf("%s: %s", method, failed)
		}
		if method == "account/read" {
			// The close handshake is a courtesy: the app server logs a
			// connection dropped without one as a warning.
			_ = conn.Close(websocket.StatusNormalClosure, "")
			return nil
		}
	}
}
