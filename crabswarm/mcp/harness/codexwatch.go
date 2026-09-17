package harness

import (
	"context"
	"time"

	"github.com/creachadair/jrpc2"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The app server is not only how a mention reaches a Codex session; it is also
// where that session says what it is doing. Everything here is that half — the
// connection held open for the feed, and the states it forwards.

// StateSource is a harness that says what its agent is doing without being
// asked, which is a harness whose CLI has a feed of its own to say it on.
//
// Every other harness reports through its hooks: a hook runs on each event the
// harness announces and tells the daemon what changed. A harness with a feed
// needs none of them, and hooks reporting alongside it would race it with a
// slower, coarser answer.
//
// The server is what joins the two ends: it holds one harness per session, and
// a harness that implements this gets watched for as long as that session runs.
// See [Harness] for the other half of what a harness does.
type StateSource interface {
	// Watch follows the harness until ctx is done, handing each state it
	// reports to report. It blocks, and it does not give up: a feed that
	// dropped is opened again, since an agent nobody hears about is one the
	// room stops delivering to.
	Watch(ctx context.Context, report func(chatv1.HarnessState))
}

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
func (c *codex) follow(
	ctx context.Context, report func(chatv1.HarnessState),
) (bool, error) {
	states := make(chan chatv1.HarnessState, codexStateBuffer)
	cli, err := dialCodex(ctx, c.path, func(req *jrpc2.Request) {
		if state, ok := codexState(req); ok {
			c.queue(states, state)
		}
	})
	if err != nil {
		return false, err
	}
	defer cli.close()
	if err := cli.handshake(ctx); err != nil {
		return false, err
	}
	// Subscribing here is what makes the feed a feed. It is best effort: an app
	// server that cannot say which thread this session is has nothing to
	// subscribe to yet, and the next delivery binds and subscribes again.
	if id, err := c.bind(ctx, cli); err == nil {
		if err := cli.subscribe(ctx, id); err != nil {
			c.logger.Warn("subscribing to the codex thread failed",
				"socket", c.path, "error", err)
		}
	}

	c.lend(cli)
	defer c.withdraw(cli)
	for {
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		case <-cli.dropped():
			// What the connection carried before it went is still true, and the
			// last of it is regularly the turn ending.
			c.drain(states, report)
			return true, cli.stopErr
		case state := <-states:
			report(state)
		}
	}
}

// queue hands one state to the goroutine reporting them, dropping the oldest
// waiting state when the queue is full.
//
// It never blocks: it runs on the JSON-RPC client's delivery goroutine, which
// holds that client's lock, so waiting here would stop every reply behind it.
// The oldest goes rather than the newest because the newest is the one that is
// still true.
func (c *codex) queue(states chan chatv1.HarnessState, state chatv1.HarnessState) {
	for {
		select {
		case states <- state:
			return
		default:
		}
		select {
		case dropped := <-states:
			c.logger.Warn("dropping a codex state nothing took in time",
				"state", dropped.String())
		default:
		}
	}
}

// drain reports the states that were waiting when the connection went.
func (c *codex) drain(states chan chatv1.HarnessState, report func(chatv1.HarnessState)) {
	for {
		select {
		case state := <-states:
			report(state)
		default:
			return
		}
	}
}
