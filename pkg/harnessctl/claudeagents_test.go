package harnessctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// claudeAgentsFixture reads the recorded listing, and hands back the one live
// session in it: the entry the feed is meant to find its own session as.
func claudeAgentsFixture(t *testing.T) (listing []claudeAgent, live claudeAgent) {
	t.Helper()

	path := filepath.Join(fixtureDir, "claude-agents.json")
	b, err := os.ReadFile(path)
	assert.NilError(t, err)
	assert.NilError(t, json.Unmarshal(b, &listing), "reading %s", path)

	var lives []claudeAgent
	for _, a := range listing {
		if a.Status != "" {
			lives = append(lives, a)
		}
	}
	assert.Equal(t, len(lives), 1, "%s holds %d live sessions", path, len(lives))
	return listing, lives[0]
}

// scriptedLister hands the feed one listing per read, in order, and keeps
// answering the last one once the script has run out.
type scriptedLister struct {
	mu      sync.Mutex
	answers []func() ([]claudeAgent, error)
	reads   int
}

func (s *scriptedLister) list(context.Context) ([]claudeAgent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	i := min(s.reads-1, len(s.answers)-1)
	return s.answers[i]()
}

func listingOf(agents ...claudeAgent) func() ([]claudeAgent, error) {
	return func() ([]claudeAgent, error) { return agents, nil }
}

func listingErr(err error) func() ([]claudeAgent, error) {
	return func() ([]claudeAgent, error) { return nil, err }
}

// watchClaude runs the feed of a Claude Code following sessionID over the
// scripted listings for the length of the test, at a cadence a test can wait
// on, and hands back the states it reports and the log it wrote.
func watchClaude(
	t *testing.T, sessionID string, script *scriptedLister,
) (<-chan chatv1.HarnessState, *bytes.Buffer) {
	t.Helper()

	var log bytes.Buffer
	h := claudeCode{
		sessionID: sessionID,
		list:      script.list,
		interval:  20 * time.Millisecond,
		logger:    slog.New(slog.NewTextHandler(&log, nil)),
	}
	ctx, cancel := context.WithCancel(t.Context())
	states := make(chan chatv1.HarnessState, 16)
	var watching errgroup.Group
	watching.Go(func() error {
		h.Watch(ctx, func(state chatv1.HarnessState) {
			select {
			case states <- state:
			default:
			}
		})
		return nil
	})
	t.Cleanup(func() {
		cancel()
		_ = watching.Wait()
	})
	return states, &log
}

// nextClaudeState takes the next state the feed reported.
func nextClaudeState(t *testing.T, states <-chan chatv1.HarnessState) chatv1.HarnessState {
	t.Helper()
	select {
	case state := <-states:
		return state
	case <-time.After(5 * time.Second):
		t.Fatal("the claude agents feed reported nothing")
		return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED
	}
}

// noClaudeState asserts the feed reports nothing over several ticks.
func noClaudeState(t *testing.T, states <-chan chatv1.HarnessState) {
	t.Helper()
	select {
	case state := <-states:
		t.Fatalf("the claude agents feed reported %s", state)
	case <-time.After(200 * time.Millisecond):
	}
}

// The listing's status is the live answer and is read first; state is the
// session's own account, read only when the listing shows no status. Anything
// outside either vocabulary is no verdict, and an unrecognised status does not
// fall through to the state beside it.
func TestClaudeAgentHarnessState_ReadsStatusThenState(t *testing.T) {
	for _, tc := range []struct {
		status claudeAgentStatus
		state  claudeAgentState
		want   chatv1.HarnessState
		ok     bool
	}{
		{claudeStatusBusy, claudeStateWorking, chatv1.HarnessState_HARNESS_STATE_WORKING, true},
		{claudeStatusWaiting, claudeStateWorking, chatv1.HarnessState_HARNESS_STATE_WAITING, true},
		{claudeStatusIdle, claudeStateWorking, chatv1.HarnessState_HARNESS_STATE_DONE, true},
		// status wins over a state that disagrees with it.
		{claudeStatusBusy, claudeStateDone, chatv1.HarnessState_HARNESS_STATE_WORKING, true},
		{claudeStatusIdle, claudeStateBlocked, chatv1.HarnessState_HARNESS_STATE_DONE, true},
		{"", claudeStateWorking, chatv1.HarnessState_HARNESS_STATE_WORKING, true},
		{"", claudeStateBlocked, chatv1.HarnessState_HARNESS_STATE_WAITING, true},
		{"", claudeStateDone, chatv1.HarnessState_HARNESS_STATE_DONE, true},
		{"", claudeStateFailed, chatv1.HarnessState_HARNESS_STATE_DONE, true},
		{"", claudeStateStopped, chatv1.HarnessState_HARNESS_STATE_DONE, true},
		{"", "", chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, false},
		{"", "paused", chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, false},
		{"thinking", claudeStateWorking, chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, false},
	} {
		t.Run(string(tc.status)+"/"+string(tc.state), func(t *testing.T) {
			got, ok := claudeAgentHarnessState(claudeAgent{Status: tc.status, State: tc.state})
			assert.Equal(t, ok, tc.ok)
			assert.Equal(t, got, tc.want)
		})
	}
}

// An interactive session carries no state of its own, so its status is the
// whole of what the listing says about it.
func TestClaudeAgentHarnessState_AnInteractiveSessionIsItsStatus(t *testing.T) {
	got, ok := claudeAgentHarnessState(claudeAgent{
		Kind:   claudeKindInteractive,
		Status: claudeStatusIdle,
	})
	assert.Assert(t, ok)
	assert.Equal(t, got, chatv1.HarnessState_HARNESS_STATE_DONE)
}

// The reason a session waits is display text, not a state of its own: every
// waitingFor is the same waiting.
func TestClaudeAgentHarnessState_WaitingWhateverTheReason(t *testing.T) {
	for _, reason := range []claudeAgentWaitingFor{
		claudeWaitingPermission,
		claudeWaitingInput,
		claudeWaitingSandbox,
		claudeWaitingWorker,
		claudeWaitingDialog,
		"",
	} {
		got, ok := claudeAgentHarnessState(claudeAgent{
			Status:     claudeStatusWaiting,
			WaitingFor: reason,
			State:      claudeStateWorking,
		})
		assert.Assert(t, ok)
		assert.Equal(t, got, chatv1.HarnessState_HARNESS_STATE_WAITING, "waitingFor %q", reason)
	}
}

// The recorded listing parses into the fields the feed reads, and the one live
// session in it is a busy one, which is a session working.
func TestClaudeAgents_TheFixtureParses(t *testing.T) {
	listing, live := claudeAgentsFixture(t)
	assert.Equal(t, len(listing), 2)
	assert.Equal(t, live.Kind, claudeKindBackground)
	assert.Equal(t, live.Status, claudeStatusBusy)
	assert.Equal(t, live.State, claudeStateWorking)

	state, ok := claudeAgentHarnessState(live)
	assert.Assert(t, ok)
	assert.Equal(t, state, chatv1.HarnessState_HARNESS_STATE_WORKING)
	for _, a := range listing {
		assert.Assert(t, a.SessionID != "", "an entry names no session")
	}
}

// The feed follows its own session through the listing: the state the entry
// shows is reported when it changes and not again while it holds, so the room
// hears each transition once.
func TestClaudeCode_WatchReportsTheListingOnChange(t *testing.T) {
	listing, live := claudeAgentsFixture(t)

	idle := live
	idle.Status = claudeStatusIdle
	waiting := live
	waiting.Status = claudeStatusWaiting
	waiting.WaitingFor = claudeWaitingDialog
	script := &scriptedLister{answers: []func() ([]claudeAgent, error){
		listingOf(listing...),
		listingOf(listing...),
		listingOf(listing[0], idle),
		listingOf(listing[0], idle),
		listingOf(waiting),
	}}
	states, log := watchClaude(t, live.SessionID, script)

	assert.Equal(t, nextClaudeState(t, states), chatv1.HarnessState_HARNESS_STATE_WORKING)
	assert.Equal(t, nextClaudeState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.Equal(t, nextClaudeState(t, states), chatv1.HarnessState_HARNESS_STATE_WAITING)
	noClaudeState(t, states)
	assert.Equal(t, log.String(), "")
}

// A listing without the session says nothing about it: the member keeps its
// last state rather than being moved by a guess, and the feed says once that
// it found nothing.
func TestClaudeCode_WatchReportsNothingForAnUnlistedSession(t *testing.T) {
	listing, _ := claudeAgentsFixture(t)
	script := &scriptedLister{answers: []func() ([]claudeAgent, error){
		listingOf(listing...),
	}}
	states, log := watchClaude(t, "00000000-0000-0000-0000-000000000000", script)

	noClaudeState(t, states)
	script.mu.Lock()
	reads := script.reads
	script.mu.Unlock()
	assert.Assert(t, reads >= 3, "the feed read the listing %d times", reads)
	assert.Equal(t, strings.Count(log.String(), "reading the claude agents listing failed"), 1,
		"log:\n%s", log.String())
}

// A listing that cannot be read is logged on the first failure of a run and
// not on the rest, and the feed goes on to report what the next good read
// shows.
func TestClaudeCode_WatchLogsAFailingListingOnce(t *testing.T) {
	listing, live := claudeAgentsFixture(t)
	broken := errors.New("claude: command not found")
	script := &scriptedLister{answers: []func() ([]claudeAgent, error){
		listingErr(broken),
		listingErr(broken),
		listingErr(broken),
		listingOf(listing...),
	}}
	states, log := watchClaude(t, live.SessionID, script)

	assert.Equal(t, nextClaudeState(t, states), chatv1.HarnessState_HARNESS_STATE_WORKING)
	assert.Equal(t, strings.Count(log.String(), "reading the claude agents listing failed"), 1,
		"log:\n%s", log.String())
	assert.Assert(t, strings.Contains(log.String(), broken.Error()), "log:\n%s", log.String())
}

// A harness that knows no session has nothing to follow: Watch returns at once
// rather than polling for a session it could never find.
func TestClaudeCode_WatchWithoutASessionReturns(t *testing.T) {
	script := &scriptedLister{answers: []func() ([]claudeAgent, error){
		listingOf(),
	}}
	h := claudeCode{sessionID: "", list: script.list, interval: time.Hour, logger: slog.Default()}
	done := make(chan struct{})
	go func() {
		h.Watch(t.Context(), func(chatv1.HarnessState) { t.Error("reported a state") })
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Watch did not return")
	}
	assert.Equal(t, script.reads, 0)
}

// The session id alone makes a Claude Code harness: one launched without the
// channel is still followed, and still woken through its terminal.
func TestNewClaudeCode_TakesTheFeedFromTheSessionId(t *testing.T) {
	env := func(name string) string {
		if name == ClaudeSessionEnv {
			return "7eda2488-ab16-4b7a-98cc-b8b8b314a225"
		}
		return ""
	}
	h := newClaudeCode(env)
	assert.Assert(t, h != nil)
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
	assert.ErrorIs(t, h.Deliver(t.Context(), Notice{Text: "hi"}), errNoChannel)
	_, ok := h.(StateSource)
	assert.Assert(t, ok, "a Claude Code with a session id is no state source")
	c := h.(claudeCode)
	assert.Equal(t, c.sessionID, "7eda2488-ab16-4b7a-98cc-b8b8b314a225")
	assert.Equal(t, c.interval, claudeAgentsInterval)
}

// The production shape of a session launched with the plugin: the channel and
// the feed on one member, each taken from its own variable, so the member is
// native and followed at once.
func TestNewClaudeCode_TakesTheChannelAndTheFeedTogether(t *testing.T) {
	env := func(name string) string {
		switch name {
		case ClaudeChannelEnv:
			return "1"
		case FakechatPortEnv:
			return "1234"
		case ClaudeSessionEnv:
			return "7eda2488-ab16-4b7a-98cc-b8b8b314a225"
		}
		return ""
	}
	h := newClaudeCode(env)
	assert.Assert(t, h != nil)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
	_, ok := h.(StateSource)
	assert.Assert(t, ok, "a Claude Code with a session id is no state source")
	_, ok = h.(Prober)
	assert.Assert(t, ok, "a Claude Code with the channel cannot be asked whether it is there")
	c := h.(channelledClaudeCode)
	assert.Equal(t, c.sessionID, "7eda2488-ab16-4b7a-98cc-b8b8b314a225")
	assert.Equal(t, c.channel.url("/"), "http://127.0.0.1:1234/")
}

// A Claude Code that knows no session says so on its stderr: the daemon reads
// no screen for a Claude Code member, so nothing else would report its state.
func TestClaudeCode_WatchWithoutASessionSaysSo(t *testing.T) {
	var log bytes.Buffer
	h := claudeCode{logger: slog.New(slog.NewTextHandler(&log, nil)), interval: time.Hour}
	h.Watch(t.Context(), func(chatv1.HarnessState) { t.Error("reported a state") })
	assert.Assert(
		t,
		strings.Contains(log.String(), "no claude code session"),
		"log:\n%s",
		log.String(),
	)
}

// The whole of it through [Detect]: a Claude Code that set its session id in
// the server's environment attends as a member whose state the feed reports,
// and, having no channel, as one the server asks nothing before it attends.
func TestDetect_ClaudeCodeWithASessionIsAStateSource(t *testing.T) {
	env := func(name string) string {
		if name == ClaudeSessionEnv {
			return "7eda2488-ab16-4b7a-98cc-b8b8b314a225"
		}
		return ""
	}
	h := Detect("claude-code", env)
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
	_, ok := h.(StateSource)
	assert.Assert(t, ok)
	_, ok = h.(Prober)
	assert.Assert(t, !ok, "%T is probed for a channel it does not have", h)
}
