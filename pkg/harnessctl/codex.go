package harnessctl

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// CodexAppServerEnv names the app server hosting this Codex session, as the
// listener address it was started with — `unix:///path/to.sock`. A bare path is
// read the same way.
//
// A Codex session in a swarm runs as `codex app-server --listen unix://PATH`
// with the terminal attached to it, so the app server is both the way in and
// the way to hear what the session is doing. The launcher puts the address in
// the environment it starts this server in; a Codex started any other way
// carries none, and its member is woken through its terminal instead.
const CodexAppServerEnv = "CRABSWARM_CODEX_APP_SERVER"

// newCodex builds the codex channel, which is the app server the session is
// hosted by. A Codex started without one is a terminal member.
func newCodex(getenv func(string) string) Harness {
	path, ok := codexSocket(getenv(CodexAppServerEnv))
	if !ok {
		return nil
	}
	return &codex{path: path, logger: slog.Default(), binding: semaphore.NewWeighted(1)}
}

// codexSocket reads the unix socket path out of a listener address, reporting
// whether it named one. An address with a scheme this cannot speak names no
// socket: dialing its remainder would reach some unrelated path, or nothing.
func codexSocket(addr string) (string, bool) {
	path, _ := strings.CutPrefix(strings.TrimSpace(addr), "unix://")
	if path == "" || strings.Contains(path, "://") {
		return "", false
	}
	return path, true
}

// codexDeliverTimeout bounds one delivery. The caller hands notices over on the
// same goroutine that reads the room's feed, so a delivery that waited on a
// wedged app server would stop the member hearing anything at all.
const codexDeliverTimeout = 30 * time.Second

// codex delivers through the app server hosting one Codex session, and reports
// what that session is doing off the same connection.
type codex struct {
	path   string
	logger *slog.Logger

	// mu guards live, the connection [codex.Watch] holds. A delivery rides it
	// when there is one, since a turn started over a connection that is then
	// closed is a turn nobody is left watching. Without a watcher a delivery
	// opens one of its own.
	//
	// A delivery that borrowed the connection just as the watcher gave it up
	// fails with whatever the closing connection reports, and the server that
	// asked leaves the mention outstanding and delivers it again — which is the
	// same answer as for any other channel that refused.
	mu   sync.Mutex
	live *codexClient

	// binding orders the changes to a connection's subscription. A delivery and
	// the feed's move to a new session thread both list the loaded threads, pick
	// the session among them and subscribe to it, and the move also unsubscribes
	// the thread it leaves. Two such sequences that overlap act on two different
	// listings. A delivery that listed the threads just before a new TUI started
	// would resume the stopped TUI's thread after the feed had moved to the new
	// one. The feed would then follow a thread nobody is in front of and read
	// past every state about the new one. So each sequence holds binding for its
	// whole length: the one that goes second lists the threads the first left
	// behind, and both settle on the same thread.
	//
	// It is a semaphore of one so that a delivery waiting on it still gives up
	// at its own deadline. The feed's calls run for as long as the session does,
	// and a lock that ignored the deadline would let a wedged app server hold
	// the delivery, and the room's feed behind it, for good.
	binding *semaphore.Weighted
}

func (c *codex) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_CODEX }

func (*codex) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// Deliver hands one notice to the session as a turn of its own.
func (c *codex) Deliver(ctx context.Context, n Notice) error {
	ctx, cancel := context.WithTimeout(ctx, codexDeliverTimeout)
	defer cancel()

	if cli := c.borrow(); cli != nil {
		return c.deliverOn(ctx, cli, n)
	}
	cli, err := dialCodex(ctx, c.path, nil)
	if err != nil {
		return err
	}
	defer cli.close()
	if err := cli.handshake(ctx); err != nil {
		return err
	}
	return c.deliverOn(ctx, cli, n)
}

// deliverOn binds the thread this session is and starts a turn on it carrying
// the notice.
//
// Binding happens per delivery rather than once, because the thread a Codex
// session is running changes under it: a new one is loaded when the session is
// resumed or forked, and a connection that remembered the first would keep
// talking to a thread nobody is looking at.
//
// It holds [codex.binding] for its whole length, for the reason given there.
func (c *codex) deliverOn(ctx context.Context, cli *codexClient, n Notice) error {
	if err := c.binding.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("waiting for the codex feed to finish binding: %w", err)
	}
	defer c.binding.Release(1)
	id, err := c.bind(ctx, cli)
	if err != nil {
		if errors.Is(err, errCodexUnbound) {
			c.logger.Warn("skipping a codex delivery", "socket", c.path, "error", err)
		}
		return err
	}
	// Subscribing before the turn is what the recorded session does, and it is
	// also what lets a delivery made over a connection nothing was watching
	// hear how the turn it started ends.
	//
	// A refused subscription does not stop the turn. Resuming a thread reads its
	// rollout, and a session's first turn is what writes one: a thread a TUI has
	// only just started is loaded and takes a turn, and answers resume with
	// "no rollout found" until that turn has run. A delivery that gave up here
	// would never reach a fresh session at all — the notice waits for a state
	// report, and a session nobody starts a turn on reports none. The feed
	// subscribes on its own once the turn exists.
	if _, err := cli.subscribe(ctx, id); err != nil {
		c.logger.Warn("starting the codex turn without a subscription",
			"socket", c.path, "thread", id, "error", err)
	}
	return cli.startTurn(ctx, id, n.Text)
}

// errCodexUnbound is the refusal for an app server holding no thread a person
// started. The notice stays unread in the room, which is where the member's own
// next read finds it.
var errCodexUnbound = errors.New("no loaded codex thread to deliver to")

// bind names the thread a delivery goes to: the session thread a person is
// sitting in front of.
//
// The app server does not hold that thread alone. Codex opens helper threads
// of its own beside a session: a `system` thread appears after the session's
// first turn, when Codex names the session, and unloads within about a minute;
// the guardian that reviews an approval runs as a thread of its own too. A
// helper is loaded just like the session, so counting loaded threads does not
// find the session. The fields a helper shares with the session do not tell
// them apart either: where the thread was started from, its parent, its
// originator and whether it takes direct input read the same on both. The
// thread's own source does differ. A thread a person started says `user`, and
// a helper says something else or nothing.
//
// A lone loaded thread is taken as the session without reading it: a helper
// only ever exists beside the session it serves.
//
// Two of a person's threads can be loaded at once. A stopped TUI's thread stays
// loaded for about two minutes beside the thread of the TUI that replaced it.
// The person is in front of the one used last, so the greatest recencyAt wins.
// A thread used in the same second as another loses to the one created later.
// Codex thread ids are time-ordered UUIDs, so the greater id is the later
// thread; the order the app server lists its threads in says nothing about
// their age.
//
// Two TUIs driven on purpose on one app server are not supported. The one used
// last gets the mentions.
func (c *codex) bind(ctx context.Context, cli *codexClient) (string, error) {
	loaded, err := cli.loadedThreads(ctx)
	if err != nil {
		return "", err
	}
	if len(loaded) == 1 {
		return loaded[0], nil
	}
	type session struct {
		id        string
		recencyAt int64
	}
	var sessions []session
	for _, id := range loaded {
		thread, err := cli.readThread(ctx, id)
		if err != nil {
			return "", err
		}
		if thread.Source == codexUserThread {
			sessions = append(sessions, session{id: id, recencyAt: thread.RecencyAt})
		}
	}
	if len(sessions) == 0 {
		return "", fmt.Errorf("%w: the app server on %s holds %d threads, none of them a person's",
			errCodexUnbound, c.path, len(loaded))
	}
	return slices.MaxFunc(sessions, func(a, b session) int {
		return cmp.Or(cmp.Compare(a.recencyAt, b.recencyAt), strings.Compare(a.id, b.id))
	}).id, nil
}

// lend publishes the watcher's connection for a delivery to ride.
func (c *codex) lend(cli *codexClient) {
	c.mu.Lock()
	c.live = cli
	c.mu.Unlock()
}

// withdraw takes back the connection cli, leaving whatever replaced it alone.
func (c *codex) withdraw(cli *codexClient) {
	c.mu.Lock()
	if c.live == cli {
		c.live = nil
	}
	c.mu.Unlock()
}

// borrow is the connection a delivery may ride, or nil when nothing watches.
func (c *codex) borrow() *codexClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.live
}
