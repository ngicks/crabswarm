package harnessctl

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/creachadair/jrpc2"
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

// codexPingTimeout bounds one ping. A ping is answered by an MCP server out of
// memory, so one that takes longer is hung, and waiting on it would spend the
// delivery's whole budget on one thread.
const codexPingTimeout = 5 * time.Second

// codex delivers through the app server hosting one agent's Codex sessions,
// and reports what that agent is doing off the connection its hub holds.
type codex struct {
	hub    *codexHub
	logger *slog.Logger

	// threads names the thread the agent is in front of. Nil is an agent that
	// has the app server to itself, whose thread is the one a person used last.
	threads AgentThreads

	// binding orders the changes to what this agent follows on a connection. A
	// delivery and the feed's move to a new session thread both pick the thread
	// and subscribe to it, and the move also unsubscribes the thread it leaves.
	// Two such sequences that overlap act on two different answers. A delivery
	// that listed the threads just before a new TUI started would resume the
	// stopped TUI's thread after the feed had moved to the new one. The feed
	// would then follow a thread nobody is in front of and read past every state
	// about the new one. So each sequence holds binding for its whole length:
	// the one that goes second picks from what the first left behind, and both
	// settle on the same thread.
	//
	// It is a semaphore of one so that a delivery waiting on it still gives up
	// at its own deadline. The feed's calls run for as long as the session does,
	// and a lock that ignored the deadline would let a wedged app server hold
	// the delivery, and the room's feed behind it, for good.
	binding *semaphore.Weighted
}

func newCodexAgent(hub *codexHub, threads AgentThreads) *codex {
	return &codex{
		hub:     hub,
		logger:  hub.logger,
		threads: threads,
		binding: semaphore.NewWeighted(1),
	}
}

func (c *codex) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_CODEX }

func (*codex) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// ScopeTo returns the agent threads names, on the connection c's hub holds.
func (c *codex) ScopeTo(threads AgentThreads) Harness {
	return newCodexAgent(c.hub, threads)
}

// Deliver hands one notice to the agent's thread as a turn of its own.
func (c *codex) Deliver(ctx context.Context, n Notice) error {
	ctx, cancel := context.WithTimeout(ctx, codexDeliverTimeout)
	defer cancel()

	// Only an agent whose feed runs rides the hub's connection: what it follows
	// there is its feed's to keep, and an agent nothing watches would leave a
	// subscription behind that nobody lets go of.
	if events := c.hub.events(c); events != nil {
		if cli := c.hub.borrow(); cli != nil {
			return c.deliverOn(ctx, cli, n, events)
		}
	}
	cli, err := dialCodex(ctx, c.hub.path, nil)
	if err != nil {
		return err
	}
	defer cli.close()
	if err := cli.handshake(ctx); err != nil {
		return err
	}
	return c.deliverOn(ctx, cli, n, nil)
}

// deliverOn binds the thread this agent is in and starts a turn on it carrying
// the notice.
//
// Binding happens per delivery rather than once, because the thread an agent
// is in changes under it: a new one is loaded when the session is resumed or
// forked, and a connection that remembered the first would keep talking to a
// thread nobody is looking at.
//
// It holds [codex.binding] for its whole length, for the reason given there.
func (c *codex) deliverOn(
	ctx context.Context, cli *codexClient, n Notice, events chan codexEvent,
) error {
	if err := c.binding.Acquire(ctx, 1); err != nil {
		return fmt.Errorf("waiting for the codex feed to finish binding: %w", err)
	}
	defer c.binding.Release(1)
	id, err := c.target(ctx, cli)
	if err != nil {
		if errors.Is(err, errCodexUnbound) {
			c.logger.Warn("skipping a codex delivery", "socket", c.hub.path, "error", err)
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
	if err := c.move(ctx, cli, id, events); err != nil {
		c.logger.Warn("starting the codex turn without a subscription",
			"socket", c.hub.path, "thread", id, "error", err)
	}
	return cli.startTurn(ctx, id, n.Text)
}

// errCodexUnbound is the refusal for an app server holding no thread the agent
// is known to be in. The notice stays unread in the room, which is where the
// member's own next read finds it.
var errCodexUnbound = errors.New("no loaded codex thread to deliver to")

// target names the thread a delivery goes to and the feed follows.
func (c *codex) target(ctx context.Context, cli *codexClient) (string, error) {
	if c.threads == nil {
		return c.bind(ctx, cli)
	}
	return c.resolve(ctx, cli)
}

// bind names the session thread a person is sitting in front of, for an agent
// that has the app server to itself.
//
// Two of a person's threads can be loaded at once. A stopped TUI's thread stays
// loaded for about two minutes beside the thread of the TUI that replaced it.
// The person is in front of the one used last, so the greatest recencyAt wins.
// A thread used in the same second as another loses to the one created later.
// Codex thread ids are time-ordered UUIDs, so the greater id is the later
// thread; the order the app server lists its threads in says nothing about
// their age.
//
// Several agents on one app server are told apart by [codex.resolve] instead;
// here the one used last would get every agent's mentions.
func (c *codex) bind(ctx context.Context, cli *codexClient) (string, error) {
	persons, loaded, err := c.personThreads(ctx, cli)
	if err != nil {
		return "", err
	}
	if len(persons) == 0 {
		return "", fmt.Errorf("%w: the app server on %s holds %d threads, none of them a person's",
			errCodexUnbound, c.hub.path, loaded)
	}
	return slices.MaxFunc(persons, func(a, b codexThread) int {
		return cmp.Or(cmp.Compare(a.RecencyAt, b.RecencyAt), strings.Compare(a.Id, b.Id))
	}).Id, nil
}

// resolve names the thread the agent was last seen in front of, among the
// threads its sessions are known to serve.
//
// A session that has not named its thread may be the one the agent is in
// front of, so first every person's thread no session has named is pinged
// until none of the agent's sessions is left unbound. A ping arrives on the
// session serving that thread, which is how that session learns it.
//
// A session that stays unbound — its thread did not answer the ping, or is not
// a person's — is left out rather than waited on: the agent's other sessions
// still say where it was last seen. An agent none of whose sessions is bound is
// given the one person's thread the app server holds, when it holds exactly
// one and no session claims it, since that is the only thread the agent can be
// in.
func (c *codex) resolve(ctx context.Context, cli *codexClient) (string, error) {
	if !c.threads.Unbound() {
		if id, ok := c.threads.Target(); ok {
			return id, nil
		}
	}
	persons, loaded, err := c.personThreads(ctx, cli)
	if err != nil {
		if id, ok := c.threads.Target(); ok {
			return id, nil
		}
		return "", err
	}
	for _, t := range persons {
		if !c.threads.Unbound() {
			break
		}
		if c.threads.Bound(t.Id) {
			continue
		}
		c.ping(ctx, cli, t.Id)
	}
	if id, ok := c.threads.Target(); ok {
		return id, nil
	}
	if len(persons) == 1 && !c.threads.Bound(persons[0].Id) {
		return persons[0].Id, nil
	}
	return "", fmt.Errorf("%w: the app server on %s holds %d threads, %d of them a person's, "+
		"and none is known to be this agent's", errCodexUnbound, c.hub.path, loaded, len(persons))
}

// personThreads lists the loaded threads a person started, and how many
// threads were loaded in all.
//
// The app server does not hold a person's threads alone. Codex opens helper
// threads of its own beside a session: a `system` thread appears after the
// session's first turn, when Codex names the session, and unloads within about
// a minute; the guardian that reviews an approval runs as a thread of its own
// too. A helper is loaded just like the session, so counting loaded threads does
// not find the session. The fields a helper shares with the session do not
// tell them apart either: where the thread was started from, its parent, its
// originator and whether it takes direct input read the same on both. The
// thread's own source does differ. A thread a person started says `user`, and
// a helper says something else or nothing.
//
// A lone loaded thread is taken as a person's without reading it: a helper only
// ever exists beside the session it serves. Its recency stays unread, since
// nothing is weighed against it.
func (c *codex) personThreads(ctx context.Context, cli *codexClient) ([]codexThread, int, error) {
	loaded, err := cli.loadedThreads(ctx)
	if err != nil {
		return nil, 0, err
	}
	if len(loaded) == 1 {
		return []codexThread{{Id: loaded[0]}}, 1, nil
	}
	var persons []codexThread
	for _, id := range loaded {
		thread, err := cli.readThread(ctx, id)
		if err != nil {
			return nil, len(loaded), err
		}
		if thread.Source == codexUserThread {
			thread.Id = id
			persons = append(persons, thread)
		}
	}
	return persons, len(loaded), nil
}

// ping has the app server call the agent's ping tool on thread's MCP server,
// which binds the session it reaches to thread.
//
// A thread whose server refused the call is skipped quietly: a server that
// does not know the tool is one this agent's server is not, and says nothing
// about the agent. Anything else is worth a line, since a ping that never
// arrives leaves the agent's newest session unbound.
func (c *codex) ping(ctx context.Context, cli *codexClient, thread string) {
	call, done := c.threads.Ping(thread)
	defer done()
	pctx, cancel := context.WithTimeout(ctx, codexPingTimeout)
	defer cancel()
	err := cli.callTool(pctx, thread, call)
	if err == nil || ctx.Err() != nil {
		return
	}
	if _, refused := errors.AsType[*jrpc2.Error](err); refused {
		c.logger.Debug("a codex thread's MCP server refused the ping",
			"socket", c.hub.path, "thread", thread, "error", err)
		return
	}
	c.logger.Warn("pinging a codex thread's MCP server failed",
		"socket", c.hub.path, "thread", thread, "error", err)
}
