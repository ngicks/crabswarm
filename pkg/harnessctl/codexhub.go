package harnessctl

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/creachadair/jrpc2"
	"golang.org/x/sync/errgroup"
)

// codexHub is one app server as the agents of one process share it: the one
// connection every agent's feed rides, held open for as long as any of them
// watches.
//
// One connection rather than one per agent, because what arrives on it is
// shared as well: a thread starting, a turn starting, a status changing. Each
// agent follows one thread on it, and reads past what the others follow.
type codexHub struct {
	path   string
	logger *slog.Logger
	// turnStarted hears the thread of every turn the connection sees start. It
	// is called from the notification handler, so it returns at once. Nil hears
	// nothing.
	turnStarted func(thread string)

	// mu guards everything below.
	//
	// A delivery rides live when there is one, since a turn started over a
	// connection that is then closed is a turn nobody is left watching. Without
	// it a delivery opens one of its own.
	//
	// A delivery that borrowed the connection just as the loop gave it up fails
	// with whatever the closing connection reports, and the server that asked
	// leaves the mention outstanding and delivers it again — which is the same
	// answer as for any other channel that refused.
	mu   sync.Mutex
	live *codexClient
	// watching are the agents whose feeds run, and where each one's states and
	// requests to bind again go.
	watching map[*codex]codexWatcher
	// stop ends the connection loop, and running is that loop. Both are nil
	// while no agent watches.
	stop    context.CancelFunc
	running *errgroup.Group
}

// codexWatcher is where the hub hands one watching agent what it hears.
type codexWatcher struct {
	events chan codexEvent
	// rebind asks the agent to bind its thread again. One waiting request is
	// as good as ten: a bind reads the threads afresh whatever asked for it.
	rebind chan struct{}
}

func newCodexHub(path string, logger *slog.Logger, turnStarted func(string)) *codexHub {
	return &codexHub{
		path:        path,
		logger:      logger,
		turnStarted: turnStarted,
		watching:    map[*codex]codexWatcher{},
	}
}

// join starts handing agent what the connection hears, opening the connection
// when agent is the first to watch.
func (h *codexHub) join(agent *codex, w codexWatcher) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.watching[agent] = w
	poke(w.rebind)
	if h.stop != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	var g errgroup.Group
	g.Go(func() error {
		h.run(ctx)
		return nil
	})
	h.stop, h.running = cancel, &g
}

// leave stops handing agent anything, and closes the connection when agent was
// the last to watch.
//
// A connection other agents still watch is kept, and the thread agent followed
// on it is let go of when nobody else follows it. Holding it would keep the app
// server sending its events for no one.
func (h *codexHub) leave(agent *codex) {
	h.mu.Lock()
	delete(h.watching, agent)
	cli := h.live
	var stop context.CancelFunc
	var running *errgroup.Group
	if len(h.watching) == 0 {
		stop, running = h.stop, h.running
		h.stop, h.running = nil, nil
	}
	h.mu.Unlock()

	if stop != nil {
		stop()
		_ = running.Wait()
		return
	}
	if cli == nil {
		return
	}
	old := cli.following(agent)
	cli.follow(agent, "")
	if old == "" || cli.followers(old) > 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), codexLeaveTimeout)
	defer cancel()
	err := cli.unsubscribe(ctx, old)
	select {
	case <-cli.dropped():
		// The connection went, and every subscription with it; agents leaving
		// together as the server shuts down meet this, and it is no failure.
		return
	default:
	}
	if err != nil {
		h.logger.Warn("unsubscribing from the codex thread an agent left failed",
			"socket", h.path, "thread", old, "error", err)
	}
}

// codexLeaveTimeout bounds the unsubscribe an agent leaving a shared
// connection makes. The agent is on its way out, and nothing waits for the
// answer but its own goroutine.
const codexLeaveTimeout = 5 * time.Second

// run holds the connection until ctx is done, opening it again whenever it
// drops.
func (h *codexHub) run(ctx context.Context) {
	backoff := codexBackoffBase
	failures := 0
	for {
		followed, err := h.follow(ctx)
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
			h.logger.Warn("the codex app server feed ended; connecting again",
				"socket", h.path, "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, codexBackoffMax)
	}
}

// follow holds one connection until it drops, reporting whether it ever
// carried anything and why it is over.
//
// Every agent is asked to bind as the connection comes up, since a new
// connection follows nothing. What the connection carried before it dropped
// stays queued with each agent: it is still true, and the last of it is
// regularly the turn ending.
func (h *codexHub) follow(ctx context.Context) (bool, error) {
	cli, err := dialCodex(ctx, h.path, h.notified)
	if err != nil {
		return false, err
	}
	defer cli.close()
	if err := cli.handshake(ctx); err != nil {
		return false, err
	}
	h.lend(cli)
	defer h.withdraw(cli)
	select {
	case <-ctx.Done():
		return true, ctx.Err()
	case <-cli.dropped():
		return true, cli.stopErr
	}
}

// notified hands one notification on to whoever it concerns. It runs while the
// JSON-RPC client's lock is held, so nothing here waits.
//
// A state goes to every agent, each of which reads past the ones about a
// thread it does not follow on cli. The thread is checked as an agent takes
// the state rather than here: a state that arrived while the agent was binding
// is judged against the thread the bind settled on, not against the one
// before it.
func (h *codexHub) notified(cli *codexClient, req *jrpc2.Request) {
	if req.Method() == codexThreadStarted {
		h.mu.Lock()
		for _, w := range h.watching {
			poke(w.rebind)
		}
		h.mu.Unlock()
		return
	}
	if thread, ok := codexTurnThread(req); ok {
		if h.turnStarted != nil {
			h.turnStarted(thread)
		}
		return
	}
	thread, state, ok := codexState(req)
	if !ok {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, w := range h.watching {
		queueCodexEvent(h.logger, w.events, codexEvent{cli: cli, thread: thread, state: state})
	}
}

// lend publishes the connection for deliveries to ride, and asks every agent
// to bind on it.
func (h *codexHub) lend(cli *codexClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.live = cli
	for _, w := range h.watching {
		poke(w.rebind)
	}
}

// withdraw takes back the connection cli, leaving whatever replaced it alone.
func (h *codexHub) withdraw(cli *codexClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.live == cli {
		h.live = nil
	}
}

// borrow is the connection a delivery may ride, or nil when nothing watches.
func (h *codexHub) borrow() *codexClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.live
}

// events is where agent's states go while it watches, or nil while it does
// not.
func (h *codexHub) events(agent *codex) chan codexEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.watching[agent].events
}

// poke leaves one signal on ch unless one is already waiting there.
func poke(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
