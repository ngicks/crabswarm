package harnessctl

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Several agents on one app server, each with TUIs of its own. The cases here
// play the MCP server beside the harness with codexSessions: every thread the
// app server opened has a session, the session belongs to one agent, and it
// names its thread once a call reaches it.

// Two agents' threads as Codex names them, and a second thread of the first.
const (
	codexThreadA  = "01a0e508-d67b-7a83-beea-9bbe77546150"
	codexThreadB  = "01a0e508-f1f5-74d0-9e01-cdc8b500a958"
	codexThreadA2 = "01a0e509-39ae-7be3-981a-74678ea1f8d9"
)

// codexSessions plays the MCP server's sessions, one per thread the app server
// opened, as every agent's [AgentThreads] reads them.
type codexSessions struct {
	mu       sync.Mutex
	seen     int
	sessions []*codexSession
	// pings is every ping the harnesses made, in order: the thread it was sent
	// to and the nonce it carried.
	pings []codexPing
}

// codexSession is one MCP session: the agent it serves, the thread the app
// server opened it for, and the thread it named, "" until a call reached it.
type codexSession struct {
	agent  *codexAgentThreads
	serves string
	thread string
	seen   int
}

type codexPing struct {
	thread, nonce string
}

// open is a session of agent opening on serves, which is the agent seen in
// it. A bound one has named its thread already.
func (s *codexSessions) open(agent *codexAgentThreads, serves string, bound bool) *codexSession {
	s.mu.Lock()
	s.seen++
	session := &codexSession{agent: agent, serves: serves, seen: s.seen}
	if bound {
		session.thread = serves
	}
	s.sessions = append(s.sessions, session)
	s.mu.Unlock()
	poke(agent.changed)
	return session
}

// focus is the agent seen in session again.
func (s *codexSessions) focus(session *codexSession) {
	s.mu.Lock()
	s.seen++
	session.seen = s.seen
	s.mu.Unlock()
	poke(session.agent.changed)
}

// toolCalled is the fake app server's MCP servers: a call reaches the session
// the app server opened for the thread, which names that thread. A thread no
// session serves has a server that knows no such tool.
func (s *codexSessions) toolCalled(thread string, params json.RawMessage) bool {
	var call struct {
		Server    string `json:"server"`
		Tool      string `json:"tool"`
		Arguments struct {
			Nonce string `json:"nonce"`
		} `json:"arguments"`
	}
	if json.Unmarshal(params, &call) != nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pings = append(s.pings, codexPing{thread: thread, nonce: call.Arguments.Nonce})
	i := slices.IndexFunc(s.sessions, func(session *codexSession) bool {
		return session.serves == thread
	})
	if i < 0 {
		return false
	}
	s.sessions[i].thread = thread
	return true
}

// pinged is the threads the harnesses pinged, in order.
func (s *codexSessions) pinged() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, p := range s.pings {
		out = append(out, p.thread)
	}
	return out
}

// codexAgentThreads is one agent's [AgentThreads] over codexSessions.
type codexAgentThreads struct {
	sessions *codexSessions
	changed  chan struct{}
}

func newCodexAgentThreads(sessions *codexSessions) *codexAgentThreads {
	return &codexAgentThreads{sessions: sessions, changed: make(chan struct{}, 1)}
}

func (a *codexAgentThreads) Target() (string, bool) {
	a.sessions.mu.Lock()
	defer a.sessions.mu.Unlock()
	var last *codexSession
	for _, s := range a.sessions.sessions {
		if s.agent == a && s.thread != "" && (last == nil || s.seen > last.seen) {
			last = s
		}
	}
	if last == nil {
		return "", false
	}
	return last.thread, true
}

func (a *codexAgentThreads) Unbound() bool {
	a.sessions.mu.Lock()
	defer a.sessions.mu.Unlock()
	return slices.ContainsFunc(a.sessions.sessions, func(s *codexSession) bool {
		return s.agent == a && s.thread == ""
	})
}

func (a *codexAgentThreads) Bound(thread string) bool {
	a.sessions.mu.Lock()
	defer a.sessions.mu.Unlock()
	return slices.ContainsFunc(a.sessions.sessions, func(s *codexSession) bool {
		return s.thread == thread
	})
}

func (a *codexAgentThreads) Changed() <-chan struct{} { return a.changed }

func (a *codexAgentThreads) Ping(thread string) (ToolCall, func()) {
	return ToolCall{
		Server:    "crabswarm-mcp",
		Tool:      "crabswarm_ping",
		Arguments: map[string]any{"nonce": "nonce-" + thread},
	}, func() {}
}

// sharedCodex is one app server's hub and a way to scope agents to it, with
// the hub's log captured.
func sharedCodex(f *codexFake, sessions *codexSessions) (*codex, *syncLog) {
	log := &syncLog{}
	f.toolCalled = sessions.toolCalled
	hub := newCodexHub(f.path, slog.New(slog.NewTextHandler(log, nil)), nil)
	return newCodexAgent(hub, nil), log
}

// scopedTo is the agent threads names on base's app server.
func scopedTo(base *codex, threads AgentThreads) *codex {
	return base.ScopeTo(threads).(*codex)
}

// waitCodexTurns blocks until the fake has started a turn on exactly want,
// oldest first.
func waitCodexTurns(t *testing.T, f *codexFake, want ...string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var got []string
	for time.Now().Before(deadline) {
		got = f.threadsOf(t, "turn/start")
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("turns were started on %v, want %v", got, want)
}

// Two agents share one app server and one connection to it, and each one's
// notice becomes a turn on its own thread, whichever thread a person used
// last.
func TestCodex_AgentsSharingAnAppServerDeliverToTheirOwnThreads(t *testing.T) {
	f := startCodexFake(t, codexThreadA, codexThreadB)
	f.recency = map[string]int64{codexThreadA: 100, codexThreadB: 200}
	sessions := &codexSessions{}
	base, _ := sharedCodex(f, sessions)
	threadsA, threadsB := newCodexAgentThreads(sessions), newCodexAgentThreads(sessions)
	sessions.open(threadsA, codexThreadA, true)
	sessions.open(threadsB, codexThreadB, true)
	a, b := scopedTo(base, threadsA), scopedTo(base, threadsB)

	statesA, statesB := watchCodex(t, a), watchCodex(t, b)
	assert.Equal(t, nextState(t, statesA), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.Equal(t, nextState(t, statesB), chatv1.HarnessState_HARNESS_STATE_DONE)

	assert.NilError(t, a.Deliver(t.Context(), Notice{Text: "for a"}))
	waitCodexTurns(t, f, codexThreadA)
	assert.NilError(t, b.Deliver(t.Context(), Notice{Text: "for b"}))
	waitCodexTurns(t, f, codexThreadA, codexThreadB)

	assert.Equal(t, f.handshakes(), 1, "the agents did not share one connection")
	// Each agent's thread was named by a session, so nothing was pinged or
	// listed to find it.
	assert.Equal(t, len(sessions.pinged()), 0)
	assert.Assert(t, !slices.Contains(f.methods(), "thread/loaded/list"),
		"the fake was asked %v", f.methods())
}

// Each agent on a shared connection hears its own thread and reads past the
// other's.
func TestCodex_AgentsSharingAnAppServerHearTheirOwnThreads(t *testing.T) {
	f := startCodexFake(t, codexThreadA, codexThreadB)
	sessions := &codexSessions{}
	base, _ := sharedCodex(f, sessions)
	threadsA, threadsB := newCodexAgentThreads(sessions), newCodexAgentThreads(sessions)
	sessions.open(threadsA, codexThreadA, true)
	sessions.open(threadsB, codexThreadB, true)
	a, b := scopedTo(base, threadsA), scopedTo(base, threadsB)

	statesA, statesB := watchCodex(t, a), watchCodex(t, b)
	assert.Equal(t, nextState(t, statesA), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.Equal(t, nextState(t, statesB), chatv1.HarnessState_HARNESS_STATE_DONE)

	f.push(codexFrameAbout(t, f.rec.notification(t, codexStatusChanged, 0), codexThreadB))
	assert.Equal(t, nextState(t, statesB), chatv1.HarnessState_HARNESS_STATE_WORKING)
	f.push(codexWaitingFrame(codexThreadA))
	assert.Equal(t, nextState(t, statesA), chatv1.HarnessState_HARNESS_STATE_WAITING)

	// Nothing about B's thread reached A: the next thing A heard was its own.
	select {
	case state := <-statesA:
		t.Fatalf("A heard %s about a thread it does not follow", state)
	case <-time.After(50 * time.Millisecond):
	}
	assert.Equal(t, f.handshakes(), 1, "the agents did not share one connection")
}

// An agent that stops watching lets go of its thread on a connection other
// agents still hold, and the connection stays for them.
func TestCodex_AnAgentLeavingASharedConnectionLetsGoOfItsThread(t *testing.T) {
	f := startCodexFake(t, codexThreadA, codexThreadB)
	sessions := &codexSessions{}
	base, _ := sharedCodex(f, sessions)
	threadsA, threadsB := newCodexAgentThreads(sessions), newCodexAgentThreads(sessions)
	sessions.open(threadsA, codexThreadA, true)
	sessions.open(threadsB, codexThreadB, true)
	a, b := scopedTo(base, threadsA), scopedTo(base, threadsB)

	ctx, cancel := context.WithCancel(t.Context())
	statesA := make(chan chatv1.HarnessState, 16)
	var watching errgroup.Group
	watching.Go(func() error {
		a.Watch(ctx, func(state chatv1.HarnessState) { statesA <- state })
		return nil
	})
	statesB := watchCodex(t, b)
	assert.Equal(t, nextState(t, statesA), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.Equal(t, nextState(t, statesB), chatv1.HarnessState_HARNESS_STATE_DONE)

	cancel()
	assert.NilError(t, watching.Wait())
	assert.DeepEqual(t, f.threadsOf(t, "thread/unsubscribe"), []string{codexThreadA})

	f.push(codexWaitingFrame(codexThreadB))
	assert.Equal(t, nextState(t, statesB), chatv1.HarnessState_HARNESS_STATE_WAITING)
	assert.Equal(t, f.handshakes(), 1, "the connection did not stay for the other agent")
}

// A session that has not named its thread yet is found by pinging the
// person's threads no session named, in the order the app server lists them,
// until none of the agent's sessions is left unbound. A ping that reaches
// another agent's session binds that one, and is not taken for the agent's.
func TestCodex_PingsTheThreadsNoSessionNamed(t *testing.T) {
	f := startCodexFake(t, codexThreadB, codexThreadA)
	sessions := &codexSessions{}
	base, _ := sharedCodex(f, sessions)
	threadsA, threadsB := newCodexAgentThreads(sessions), newCodexAgentThreads(sessions)
	sessions.open(threadsA, codexThreadA, false)
	sessions.open(threadsB, codexThreadB, false)
	a := scopedTo(base, threadsA)

	assert.NilError(t, a.Deliver(t.Context(), Notice{Text: "for a"}))

	assert.Equal(t, deliveredTo(t, f), codexThreadA)
	assert.DeepEqual(t, sessions.pinged(), []string{codexThreadB, codexThreadA})
	// The ping asked the crabswarm server of the thread for the ping tool, with
	// the nonce the agent's side issued.
	var call struct {
		ThreadId  string         `json:"threadId"`
		Server    string         `json:"server"`
		Tool      string         `json:"tool"`
		Arguments map[string]any `json:"arguments"`
	}
	assert.NilError(t, json.Unmarshal(f.paramsOf(t, "mcpServer/tool/call"), &call))
	assert.Equal(t, call.ThreadId, codexThreadB)
	assert.Equal(t, call.Server, "crabswarm-mcp")
	assert.Equal(t, call.Tool, "crabswarm_ping")
	assert.DeepEqual(t, call.Arguments, map[string]any{"nonce": "nonce-" + codexThreadB})
	// B's session learned its thread from the ping that reached it.
	thread, ok := threadsB.Target()
	assert.Assert(t, ok)
	assert.Equal(t, thread, codexThreadB)
}

// A thread some session already named is never pinged, and pinging stops once
// the agent has no unbound session left.
func TestCodex_PingsNoMoreThanItNeeds(t *testing.T) {
	f := startCodexFake(t, codexThreadB, codexThreadA, codexThreadA2)
	sessions := &codexSessions{}
	base, _ := sharedCodex(f, sessions)
	threadsA, threadsB := newCodexAgentThreads(sessions), newCodexAgentThreads(sessions)
	sessions.open(threadsB, codexThreadB, true)
	sessions.open(threadsA, codexThreadA, false)
	a := scopedTo(base, threadsA)

	assert.NilError(t, a.Deliver(t.Context(), Notice{Text: "for a"}))

	assert.Equal(t, deliveredTo(t, f), codexThreadA)
	assert.DeepEqual(t, sessions.pinged(), []string{codexThreadA})
}

// An agent is followed to the session it was last seen in: a newer session
// takes the next notice at once, and the feed moves with it; an older session
// the agent is seen in again takes it back.
func TestCodex_FollowsTheSessionTheAgentWasLastSeenIn(t *testing.T) {
	f := startCodexFake(t, codexThreadA)
	f.resumeStatus = map[string]string{codexThreadA: `{"type":"active","activeFlags":[]}`}
	sessions := &codexSessions{}
	base, _ := sharedCodex(f, sessions)
	threads := newCodexAgentThreads(sessions)
	first := sessions.open(threads, codexThreadA, true)
	a := scopedTo(base, threads)

	states := watchCodex(t, a)
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WORKING)

	// A new thread opens a session of its own, which has named nothing yet.
	f.update(func() { f.loaded = []string{codexThreadA, codexThreadA2} })
	sessions.open(threads, codexThreadA2, false)

	// The new thread keeps the recorded resume answer, which is idle; the feed
	// found it by pinging, and let go of the thread it left.
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.DeepEqual(t, sessions.pinged(), []string{codexThreadA2})
	assert.DeepEqual(t, f.threadsOf(t, "thread/unsubscribe"), []string{codexThreadA})
	assert.Equal(t, followed(a), codexThreadA2)
	assert.NilError(t, a.Deliver(t.Context(), Notice{Text: "to the new one"}))
	waitCodexTurns(t, f, codexThreadA2)

	// The agent is seen in the first session again.
	sessions.focus(first)
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WORKING)
	assert.Equal(t, followed(a), codexThreadA)
	assert.NilError(t, a.Deliver(t.Context(), Notice{Text: "back to the first"}))
	waitCodexTurns(t, f, codexThreadA2, codexThreadA)
}

// An agent none of whose sessions answered a ping is in the one person's
// thread the app server holds, when there is one and no session claims it.
// Two leave nothing to tell them apart, and nothing is started.
func TestCodex_AnAgentNoSessionNamedTakesTheLonePersonThread(t *testing.T) {
	t.Run("one person's thread", func(t *testing.T) {
		const helper = "helper-system"
		f := startCodexFake(t, helper, codexThreadA)
		f.sources = map[string]string{helper: "system"}
		sessions := &codexSessions{}
		base, _ := sharedCodex(f, sessions)
		threads := newCodexAgentThreads(sessions)
		sessions.open(threads, "a thread nothing forwards to", false)

		assert.NilError(t, scopedTo(base, threads).Deliver(t.Context(), Notice{Text: "hi"}))
		assert.Equal(t, deliveredTo(t, f), codexThreadA)
	})

	t.Run("two people's threads", func(t *testing.T) {
		f := startCodexFake(t, codexThreadA, codexThreadB)
		sessions := &codexSessions{}
		base, log := sharedCodex(f, sessions)
		threads := newCodexAgentThreads(sessions)
		sessions.open(threads, "a thread nothing forwards to", false)

		err := scopedTo(base, threads).Deliver(t.Context(), Notice{Text: "hi"})
		assert.ErrorContains(t, err, "none is known to be this agent's")
		assert.Assert(t, !slices.Contains(f.methods(), "turn/start"),
			"the fake was asked %v", f.methods())
		assert.Assert(t, strings.Contains(log.String(), "skipping a codex delivery"),
			"the harness logged %q", log.String())
	})
}

// A turn starting on any thread the connection hears of is handed on by its
// thread, which is how the server beside the harness learns where an agent
// was last seen.
func TestCodex_TheHubHandsOnEveryTurnStart(t *testing.T) {
	f := startCodexFake(t, codexThreadA)
	started := make(chan string, 4)
	hub := newCodexHub(f.path, slog.New(slog.DiscardHandler), func(thread string) {
		started <- thread
	})
	watchCodex(t, newCodexAgent(hub, nil))
	f.waitAsked(t, "thread/resume")

	f.push(codexTurnStartedFrame(t, f.rec, codexThreadB))
	select {
	case thread := <-started:
		assert.Equal(t, thread, codexThreadB)
	case <-time.After(10 * time.Second):
		t.Fatal("the turn start was never handed on")
	}
}

// One detector hands every session of one app server the same harness, so the
// agents it is scoped to share its connection; another app server gets its
// own.
func TestDetector_SharesOneHarnessPerAppServer(t *testing.T) {
	d := NewDetector(nil, nil)
	first := d.Detect("codex-mcp-client", codexEnv("unix:///tmp/one.sock"))
	again := d.Detect("codex-mcp-client", codexEnv("unix:///tmp/one.sock"))
	other := d.Detect("codex-mcp-client", codexEnv("unix:///tmp/other.sock"))
	assert.Assert(t, first == again, "one app server was given two harnesses")
	assert.Assert(t, first != other, "two app servers were given one harness")

	scoper, ok := first.(ThreadScoper)
	assert.Assert(t, ok, "a codex harness cannot be scoped")
	sessions := &codexSessions{}
	a := scoper.ScopeTo(newCodexAgentThreads(sessions)).(*codex)
	b := scoper.ScopeTo(newCodexAgentThreads(sessions)).(*codex)
	assert.Assert(t, a != b)
	assert.Assert(t, a.hub == first.(*codex).hub && b.hub == a.hub,
		"a scoped harness has a connection of its own")
}
