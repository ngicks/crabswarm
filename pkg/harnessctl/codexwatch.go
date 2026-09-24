package harnessctl

import (
	"context"
	"errors"
	"time"

	"github.com/creachadair/jrpc2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The app server is not only how a mention reaches a Codex session; it is also
// where that session says what it is doing. Everything here is that half — the
// connection held open for the feed, and the states it forwards.

// How long the harness waits before opening the app server connection again.
// The first retries are quick, for the app server still binding its socket as
// the session starts; the ceiling keeps one that is gone from being dialled in
// a loop.
const (
	codexBackoffBase = 200 * time.Millisecond
	codexBackoffMax  = 5 * time.Second
)

// codexStateBuffer is how many states may wait for the caller's report before
// the oldest of them is dropped. It is generous because dropping is the bad
// outcome and a state is a few dozen bytes; the queue only grows while the
// daemon is slow to take one.
const codexStateBuffer = 64

// codexRebindInterval is how often a feed that follows no thread looks for the
// session thread again. The app server announces a thread a TUI starts, and
// that announcement is what normally ends the wait. Only that one way of
// loading a thread was seen announced, though, and a bind can also fail for a
// reason of the moment; asking again every few seconds keeps either from
// leaving the room deaf to the session until its next thread starts.
const codexRebindInterval = 2 * time.Second

// codexEvent is one state the app server gave, and the thread it gave it for.
type codexEvent struct {
	thread string
	state  chatv1.HarnessState
}

// Watch follows the app server's own account of the session for as long as ctx
// runs, opening the connection again whenever it drops.
func (c *codex) Watch(ctx context.Context, report func(chatv1.HarnessState)) {
	backoff := codexBackoffBase
	failures := 0
	for {
		followed, err := c.follow(ctx, report)
		if ctx.Err() != nil {
			return
		}
		// A connection that stood for a while and then dropped is not the
		// trouble one that never opened is, so it starts its retries over
		// rather than inheriting the wait the last failure had climbed to.
		if followed {
			backoff = codexBackoffBase
			failures = 0
		}
		failures++
		// Only the first of a run: it is the one that says what broke, and a
		// line per retry for the rest of the session would bury everything else
		// on the harness's stderr.
		if failures == 1 {
			c.logger.Warn("the codex app server feed ended; connecting again",
				"socket", c.path, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, codexBackoffMax)
	}
}

// follow holds one connection and reports what it carries until it drops,
// reporting whether it ever carried anything and why it is over.
//
// Only the thread the connection is subscribed to is reported on. The thread is
// checked as a state is taken off the queue rather than as it arrives: a state
// that arrived while the thread was being bound is judged against the thread
// the bind settled on, not against the one before it.
func (c *codex) follow(
	ctx context.Context, report func(chatv1.HarnessState),
) (bool, error) {
	events := make(chan codexEvent, codexStateBuffer)
	// One waiting request to bind again is as good as ten: a bind reads the
	// loaded threads afresh whichever announcement asked for it.
	rebind := make(chan struct{}, 1)
	cli, err := dialCodex(ctx, c.path, func(req *jrpc2.Request) {
		if req.Method() == codexThreadStarted {
			select {
			case rebind <- struct{}{}:
			default:
			}
			return
		}
		if thread, state, ok := codexState(req); ok {
			c.queue(events, codexEvent{thread: thread, state: state})
		}
	})
	if err != nil {
		return false, err
	}
	defer cli.close()
	if err := cli.handshake(ctx); err != nil {
		return false, err
	}
	// Subscribing is what makes the feed a feed. An app server that cannot say
	// which thread this session is has nothing to subscribe to yet; the feed
	// binds again when a thread starts, and every few seconds until one does.
	var failing bool
	bind := func() {
		err := c.rebind(ctx, cli, events)
		// Holding no session thread is no failure at all: it is an app server
		// whose TUI has not attached yet. Of the failures, only the first of a
		// run is worth a line, for the reason [codex.Watch] gives.
		failed := err != nil && !errors.Is(err, errCodexUnbound) && ctx.Err() == nil
		if failed && !failing {
			c.logger.Warn("binding the codex feed to the session thread failed",
				"socket", c.path, "error", err)
		}
		failing = failed
	}
	bind()

	c.lend(cli)
	defer c.withdraw(cli)
	retry := time.NewTicker(codexRebindInterval)
	defer retry.Stop()
	for {
		// A nil channel is never ready, so the timer only counts while the
		// connection follows nothing.
		var retried <-chan time.Time
		if cli.subscribed() == "" {
			retried = retry.C
		}
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		case <-cli.dropped():
			// What the connection carried before it went is still true, and the
			// last of it is regularly the turn ending.
			c.drain(cli, events, report)
			return true, cli.stopErr
		case <-rebind:
			bind()
		case <-retried:
			bind()
		case ev := <-events:
			if cli.follows(ev.thread) {
				report(ev.state)
			}
		}
	}
}

// rebind binds the session thread and, when the connection follows another,
// moves the subscription to it: the old thread is unsubscribed and the new one
// resumed. What the resume answer says the new thread is doing is queued like
// any state a notification carried.
//
// That state is what moves the room off the thread it leaves behind. A stopped
// TUI's thread stays loaded for a while, and the room still holds whatever it
// last said; a replacement sitting idle says nothing until its first turn, so a
// room left believing the old thread was mid-turn would hold back every
// mention until then.
//
// A bind that fails leaves the subscription where it was, since the thread
// followed until now is still the best one this connection knows of. A resume
// that fails after the old thread was unsubscribed leaves the connection
// following nothing, which is the state the feed retries from on its timer.
func (c *codex) rebind(ctx context.Context, cli *codexClient, events chan codexEvent) error {
	id, err := c.bind(ctx, cli)
	if err != nil {
		return err
	}
	old := cli.subscribed()
	if id == old {
		return nil
	}
	if old != "" {
		// Leaving the old thread subscribed costs nothing but its events, which
		// name a thread this connection no longer follows and are read past.
		if err := cli.unsubscribe(ctx, old); err != nil {
			c.logger.Warn("unsubscribing from the abandoned codex thread failed",
				"socket", c.path, "thread", old, "error", err)
		}
	}
	status, err := cli.subscribe(ctx, id)
	if err != nil {
		return err
	}
	if state, ok := status.state(); ok {
		c.queue(events, codexEvent{thread: id, state: state})
	}
	return nil
}

// queue hands one state to the goroutine reporting them, dropping the oldest
// waiting state when the queue is full.
//
// It never blocks: it runs on the JSON-RPC client's delivery goroutine, which
// holds that client's lock, so waiting here would stop every reply behind it.
// The oldest goes rather than the newest because the newest is the one that is
// still true.
func (c *codex) queue(events chan codexEvent, ev codexEvent) {
	for {
		select {
		case events <- ev:
			return
		default:
		}
		select {
		case dropped := <-events:
			c.logger.Warn("dropping a codex state nothing took in time",
				"thread", dropped.thread, "state", dropped.state.String())
		default:
		}
	}
}

// drain reports the states about the followed thread that were waiting when
// the connection went.
func (c *codex) drain(cli *codexClient, events chan codexEvent, report func(chatv1.HarnessState)) {
	for {
		select {
		case ev := <-events:
			if cli.follows(ev.thread) {
				report(ev.state)
			}
		default:
			return
		}
	}
}
