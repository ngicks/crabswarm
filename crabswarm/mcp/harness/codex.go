package harness

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

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
func newCodex(getenv func(string) string, _ Session) Harness {
	path, ok := codexSocket(getenv(CodexAppServerEnv))
	if !ok {
		return nil
	}
	return &codex{path: path, logger: slog.Default()}
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
func (c *codex) deliverOn(ctx context.Context, cli *codexClient, n Notice) error {
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
	if err := cli.subscribe(ctx, id); err != nil {
		return err
	}
	return cli.startTurn(ctx, id, n.Text)
}

// errCodexUnbound is the refusal for an app server whose loaded threads do not
// say which one this session is.
var errCodexUnbound = errors.New("no single loaded codex thread to deliver to")

// bind names the thread a delivery goes to: the one thread the app server
// holds in memory.
//
// Anything else is refused rather than guessed at. An app server hosting
// several loaded threads cannot say which of them the agent is sitting in
// front of, and a turn started on the wrong one would interrupt a session
// nobody addressed. The notice stays unread in the room, which is where the
// member's own next read finds it.
func (c *codex) bind(ctx context.Context, cli *codexClient) (string, error) {
	loaded, err := cli.loadedThreads(ctx)
	if err != nil {
		return "", err
	}
	if len(loaded) != 1 {
		return "", fmt.Errorf("%w: the app server on %s holds %d of them",
			errCodexUnbound, c.path, len(loaded))
	}
	return loaded[0], nil
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
