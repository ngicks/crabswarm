package harnessctl

import (
	"context"
	"errors"
	"log/slog"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The app server is not only how a mention reaches a Codex session; it is also
// where that session says what it is doing. Everything here is one agent's half
// of that — the thread it follows on the connection [codexHub] holds, and the
// states it forwards.

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

// codexEvent is one state the app server gave, the thread it gave it for, and
// the connection it came in on — which is what says whether the agent follows
// that thread.
type codexEvent struct {
	cli    *codexClient
	thread string
	state  chatv1.HarnessState
}

// Watch follows the app server's own account of the agent's thread for as long
// as ctx runs. The connection is the hub's, opened again whenever it drops.
//
// Only the thread the agent follows is reported on. The thread moves when the
// agent is seen in front of another one, when the app server announces a new
// thread, and every few seconds while the agent follows none.
func (c *codex) Watch(ctx context.Context, report func(chatv1.HarnessState)) {
	w := codexWatcher{
		events: make(chan codexEvent, codexStateBuffer),
		rebind: make(chan struct{}, 1),
	}
	c.hub.join(c, w)
	defer c.hub.leave(c)

	var changed <-chan struct{}
	if c.threads != nil {
		changed = c.threads.Changed()
	}
	var failing bool
	bind := func() {
		cli := c.hub.borrow()
		if cli == nil {
			// The hub asks again once it has a connection.
			return
		}
		err := c.rebind(ctx, cli, w.events)
		// Holding no session thread is no failure at all: it is an app server
		// whose TUI has not attached yet. Of the failures, only the first of a
		// run is worth a line, for the reason [codexHub.run] gives.
		failed := err != nil && !errors.Is(err, errCodexUnbound) && ctx.Err() == nil
		if failed && !failing {
			c.logger.Warn("binding the codex feed to the session thread failed",
				"socket", c.hub.path, "error", err)
		}
		failing = failed
	}

	retry := time.NewTicker(codexRebindInterval)
	defer retry.Stop()
	for {
		// A nil channel is never ready, so the timer only counts while the
		// agent follows nothing on a connection it has.
		var retried <-chan time.Time
		if cli := c.hub.borrow(); cli != nil && cli.following(c) == "" {
			retried = retry.C
		}
		select {
		case <-ctx.Done():
			return
		case <-w.rebind:
			bind()
		case <-changed:
			bind()
		case <-retried:
			bind()
		case ev := <-w.events:
			if ev.cli.follows(c, ev.thread) {
				report(ev.state)
			}
		}
	}
}

// rebind binds the agent's thread and moves what the agent follows on cli to
// it.
//
// A bind that fails leaves the agent following what it followed, since that
// thread is still the best one this connection knows of.
//
// It holds [codex.binding] for its whole length, for the reason given there.
func (c *codex) rebind(ctx context.Context, cli *codexClient, events chan codexEvent) error {
	if err := c.binding.Acquire(ctx, 1); err != nil {
		return err
	}
	defer c.binding.Release(1)
	id, err := c.target(ctx, cli)
	if err != nil {
		return err
	}
	return c.move(ctx, cli, id, events)
}

// move has the agent follow id on cli: the thread it leaves is unsubscribed
// when no other agent follows it, and id is resumed. What the resume answer
// says id is doing is queued on events like any state a notification carried;
// nil events drops it.
//
// That state is what moves the room off the thread the agent leaves behind. A
// stopped TUI's thread stays loaded for a while, and the room still holds
// whatever it last said; a replacement sitting idle says nothing until its
// first turn, so a room left believing the old thread was mid-turn would hold
// back every mention until then.
//
// A resume that fails after the old thread was let go of leaves the agent
// following nothing, which is the state its feed retries from on its timer.
//
// The caller holds [codex.binding].
func (c *codex) move(
	ctx context.Context,
	cli *codexClient,
	id string,
	events chan codexEvent,
) error {
	old := cli.following(c)
	if id == old {
		return nil
	}
	if old != "" {
		cli.follow(c, "")
		// Leaving the old thread subscribed costs nothing but its events, which
		// name a thread this agent no longer follows and are read past.
		if cli.followers(old) == 0 {
			if err := cli.unsubscribe(ctx, old); err != nil {
				c.logger.Warn("unsubscribing from the abandoned codex thread failed",
					"socket", c.hub.path, "thread", old, "error", err)
			}
		}
	}
	status, err := cli.subscribe(ctx, id)
	if err != nil {
		return err
	}
	cli.follow(c, id)
	if state, ok := status.state(); ok && events != nil {
		queueCodexEvent(c.logger, events, codexEvent{cli: cli, thread: id, state: state})
	}
	return nil
}

// queueCodexEvent hands one state to the goroutine reporting them, dropping
// the oldest waiting state when the queue is full.
//
// It never blocks: a notification handler calls it while holding the JSON-RPC
// client's lock, and every other frame received and every call sent waits on
// that lock. The oldest goes rather than the newest because the newest is the
// one that is still true.
func queueCodexEvent(logger *slog.Logger, events chan codexEvent, ev codexEvent) {
	for {
		select {
		case events <- ev:
			return
		default:
		}
		select {
		case dropped := <-events:
			logger.Warn("dropping a codex state nothing took in time",
				"thread", dropped.thread, "state", dropped.state.String())
		default:
		}
	}
}
