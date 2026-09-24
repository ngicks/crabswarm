package harnessctl

import (
	"bytes"
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

// codexEnv is a harness started by a launcher that pointed it at addr.
func codexEnv(addr string) func(string) string {
	return func(name string) string {
		if name == CodexAppServerEnv {
			return addr
		}
		return ""
	}
}

// syncLog collects what a harness logged while its own goroutines were running.
type syncLog struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *syncLog) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

func (l *syncLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

// newCodexHarness is one codex harness talking to f, with its log captured.
func newCodexHarness(f *codexFake) (*codex, *syncLog) {
	log := &syncLog{}
	return &codex{
		path:   f.path,
		logger: slog.New(slog.NewTextHandler(log, nil)),
	}, log
}

// threadId is the thread the recorded session ran on, read out of the first
// status the server pushed about it. Taking it from the recording rather than
// spelling it again keeps a re-captured session from needing an edit here.
func (r codexRecording) threadId(t *testing.T) string {
	t.Helper()
	var frame struct {
		Params struct {
			ThreadId string `json:"threadId"`
		} `json:"params"`
	}
	assert.NilError(t, json.Unmarshal(r.notification(t, codexStatusChanged, 0), &frame))
	assert.Assert(t, frame.Params.ThreadId != "", "the recording names no thread")
	return frame.Params.ThreadId
}

// watchCodex runs h's feed for the length of the test and hands back the states
// it reports.
//
// The report never blocks: a full buffer is a case that stopped reading, and
// the assertion that follows says so far better than a cleanup that hangs.
func watchCodex(t *testing.T, h *codex) <-chan chatv1.HarnessState {
	t.Helper()
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
	return states
}

// nextState takes the next state the feed reported.
func nextState(t *testing.T, states <-chan chatv1.HarnessState) chatv1.HarnessState {
	t.Helper()
	select {
	case state := <-states:
		return state
	case <-time.After(10 * time.Second):
		t.Fatal("the codex feed reported nothing")
		return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED
	}
}

// A Codex pointed at an app server delivers its own mentions; one started
// without the variable that names it is woken through its terminal, which is
// what a harness with no channel falls back to.
func TestDetect_CodexTakesItsAppServer(t *testing.T) {
	h := Detect("codex-mcp-client", codexEnv("unix:///tmp/codex.sock"))
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CODEX)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	for _, addr := range []string{"", "   ", "tcp://127.0.0.1:1234"} {
		t.Run("address "+addr, func(t *testing.T) {
			h := Detect("codex-mcp-client", codexEnv(addr))
			assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CODEX)
			assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
		})
	}
}

// The listener address is read as the socket behind it, with or without the
// scheme the launcher spells it with.
func TestCodexSocket_ReadsTheListenerAddress(t *testing.T) {
	for _, tc := range []struct {
		addr string
		want string
		ok   bool
	}{
		{"unix:///run/codex.sock", "/run/codex.sock", true},
		{" unix:///run/codex.sock ", "/run/codex.sock", true},
		{"/run/codex.sock", "/run/codex.sock", true},
		{"", "", false},
		{"unix://", "", false},
		{"tcp://127.0.0.1:1234", "", false},
	} {
		t.Run(tc.addr, func(t *testing.T) {
			path, ok := codexSocket(tc.addr)
			assert.Equal(t, ok, tc.ok)
			assert.Equal(t, path, tc.want)
		})
	}
}

// One loaded thread is the session, taken without reading it, and the notice
// reaches it as a turn of its own — the whole exchange in order, with the
// notice as the user's input.
func TestCodex_DeliversTheNoticeAsATurn(t *testing.T) {
	f := startCodexFake(t)
	thread := f.rec.threadId(t)
	f.loaded = []string{thread}
	h, _ := newCodexHarness(f)

	notice := Notice{From: "alpha/bob", Text: "[crabswarm chat] new message from alpha/bob"}
	assert.NilError(t, h.Deliver(t.Context(), notice))

	assert.DeepEqual(t, f.methods(), []string{
		"initialize", "initialized", "thread/loaded/list", "thread/resume", "turn/start",
	})

	var started struct {
		ThreadId string `json:"threadId"`
		Input    []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"input"`
	}
	assert.NilError(t, json.Unmarshal(f.paramsOf(t, "turn/start"), &started))
	assert.Equal(t, started.ThreadId, thread)
	assert.Equal(t, len(started.Input), 1)
	assert.Equal(t, started.Input[0].Type, "text")
	assert.Equal(t, started.Input[0].Text, notice.Text)

	// The notice is what the agent reads, never the message: nothing the sender
	// wrote is carried into the turn.
	assert.Equal(t, strings.Contains(started.Input[0].Text, "\n"), false)
}

// deliveredTo is the thread the delivery started its turn on.
func deliveredTo(t *testing.T, f *codexFake) string {
	t.Helper()
	var started struct {
		ThreadId string `json:"threadId"`
	}
	assert.NilError(t, json.Unmarshal(f.paramsOf(t, "turn/start"), &started))
	return started.ThreadId
}

// Codex loads helper threads beside the session, and the notice goes to the
// one thread a person started, whichever helper sits beside it. The recorded
// helper answers `system`; any other source is a helper too. The helper is the
// most recently used thread loaded, as the recorded one was, so the session is
// found by its source before recency is weighed.
func TestCodex_DeliversPastTheHelperThreads(t *testing.T) {
	for _, helper := range []string{"system", "guardian_review", "memory_consolidation"} {
		t.Run(helper, func(t *testing.T) {
			f := startCodexFake(t)
			thread := f.rec.threadId(t)
			f.loaded = []string{"helper-" + helper, thread}
			f.sources = map[string]string{"helper-" + helper: helper}
			f.recency = map[string]int64{"helper-" + helper: 200, thread: 100}
			h, _ := newCodexHarness(f)

			assert.NilError(t, h.Deliver(t.Context(), Notice{Text: "hi"}))

			assert.DeepEqual(t, f.methods(), []string{
				"initialize", "initialized", "thread/loaded/list",
				"thread/read", "thread/read", "thread/resume", "turn/start",
			})
			assert.Equal(t, deliveredTo(t, f), thread)
		})
	}
}

// Two of a person's threads as Codex names them: time-ordered UUIDs, so the
// earlier one sorts first. They are the two TUI threads of the helpers
// recording.
const (
	codexEarlierSession = "01a0c777-0a11-7713-a00f-f61ed05c0e9f"
	codexLaterSession   = "01a0c77a-cf11-7122-9a8d-af74926d81a4"
)

// codexSessionOrders are both orders an app server may list the two in.
var codexSessionOrders = []struct {
	name string
	ids  []string
}{
	{"listed earlier first", []string{codexEarlierSession, codexLaterSession}},
	{"listed later first", []string{codexLaterSession, codexEarlierSession}},
}

// deliverAmongSessions delivers one notice through an app server holding
// loaded, each thread a person's and used at the second recency gives it, and
// returns the thread the turn was started on.
func deliverAmongSessions(t *testing.T, loaded []string, recency map[string]int64) string {
	t.Helper()
	f := startCodexFake(t, loaded...)
	f.recency = recency
	h, _ := newCodexHarness(f)

	assert.NilError(t, h.Deliver(t.Context(), Notice{Text: "hi"}))

	assert.DeepEqual(t, f.methods(), []string{
		"initialize", "initialized", "thread/loaded/list",
		"thread/read", "thread/read", "thread/resume", "turn/start",
	})
	return deliveredTo(t, f)
}

// A stopped TUI leaves its thread loaded for about two minutes beside the
// thread of the TUI that replaced it. The notice goes to the thread used last,
// whichever order the app server lists them in, and even when that is the
// earlier thread a person resumed.
func TestCodex_DeliversToTheSessionUsedLast(t *testing.T) {
	for _, tc := range []struct {
		name    string
		recency map[string]int64
		want    string
	}{
		{
			name:    "the later thread was used last",
			recency: map[string]int64{codexEarlierSession: 100, codexLaterSession: 200},
			want:    codexLaterSession,
		},
		{
			name:    "the earlier thread was resumed and used last",
			recency: map[string]int64{codexEarlierSession: 200, codexLaterSession: 100},
			want:    codexEarlierSession,
		},
	} {
		for _, order := range codexSessionOrders {
			t.Run(tc.name+"/"+order.name, func(t *testing.T) {
				assert.Equal(t, deliverAmongSessions(t, order.ids, tc.recency), tc.want)
			})
		}
	}
}

// Two threads used in the same second go to the one created later, which is
// the greater id, whichever order the app server lists them in.
func TestCodex_BreaksARecencyTieOnTheLaterThread(t *testing.T) {
	recency := map[string]int64{codexEarlierSession: 100, codexLaterSession: 100}
	for _, order := range codexSessionOrders {
		t.Run(order.name, func(t *testing.T) {
			assert.Equal(t, deliverAmongSessions(t, order.ids, recency), codexLaterSession)
		})
	}
}

// An app server holding no thread a person started has no session to hand the
// notice to: nothing is started, the warning says how many threads it held,
// and the server that asked keeps the mention outstanding.
func TestCodex_SkipsADeliveryItCannotBind(t *testing.T) {
	for _, tc := range []struct {
		name    string
		loaded  []string
		sources map[string]string
		held    string
	}{
		{
			name: "no thread is loaded",
			held: "holds 0, 0 of them a person's",
		},
		{
			name:   "only helpers are loaded",
			loaded: []string{"helper-system", "helper-guardian"},
			sources: map[string]string{
				"helper-system":   "system",
				"helper-guardian": "guardian_review",
			},
			held: "holds 2, 0 of them a person's",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := startCodexFake(t, tc.loaded...)
			f.sources = tc.sources
			h, log := newCodexHarness(f)

			err := h.Deliver(t.Context(), Notice{Text: "hi"})
			assert.ErrorContains(t, err, "loaded codex thread")
			assert.Assert(t, !slices.Contains(f.methods(), "turn/start"),
				"the fake was asked %v", f.methods())
			assert.Assert(t, strings.Contains(log.String(), "skipping a codex delivery"),
				"the harness logged %q", log.String())
			assert.Assert(t, strings.Contains(log.String(), tc.held),
				"the harness logged %q", log.String())
		})
	}
}

// The app server is the session's own account of itself, so the feed says what
// a hook used to: working while a turn runs, waiting while a dialog is open,
// done when the thread goes idle and done again however the turn ended.
func TestCodex_WatchReportsWhatTheAppServerSays(t *testing.T) {
	f := startCodexFake(t)
	thread := f.rec.threadId(t)
	f.loaded = []string{thread}
	h, _ := newCodexHarness(f)

	states := watchCodex(t, h)
	f.waitAsked(t, "thread/resume")

	// The recorded frames go over the wire as they were captured: no version
	// marker, a stamp beside the parameters, nothing rewritten.
	f.push(f.rec.notification(t, codexStatusChanged, 0))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WORKING)

	f.push(codexWaitingFrame(thread))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WAITING)

	f.push(f.rec.notification(t, codexStatusChanged, 1))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)

	// The last recorded turn was interrupted, which ends it as surely as
	// finishing it does.
	f.push(f.rec.notification(t, codexTurnCompleted, 1))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
}

// A feed that drops is opened again: an app server restarted under a running
// session leaves the room hearing nothing about that agent otherwise.
func TestCodex_WatchConnectsAgainAfterADrop(t *testing.T) {
	f := startCodexFake(t)
	thread := f.rec.threadId(t)
	f.loaded = []string{thread}
	h, _ := newCodexHarness(f)

	states := watchCodex(t, h)
	f.waitAsked(t, "thread/resume")
	f.disconnect()

	waitCodexHandshakes(t, f, 2)
	f.push(f.rec.notification(t, codexStatusChanged, 1))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
}

// waitCodexHandshakes blocks until the fake has been introduced to want times,
// which is how a case counts the connections that were made to it.
func waitCodexHandshakes(t *testing.T, f *codexFake, want int) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if f.handshakes() >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the fake was introduced to fewer than %d times: %v", want, f.methods())
}

// A delivery made while the feed is up rides the feed's own connection, so the
// turn it starts is one the connection watching the session sees through — and
// it asks for nothing that connection already has.
func TestCodex_DeliveryRidesTheWatchedConnection(t *testing.T) {
	f := startCodexFake(t)
	thread := f.rec.threadId(t)
	f.loaded = []string{thread}
	h, _ := newCodexHarness(f)

	watchCodex(t, h)
	f.waitAsked(t, "thread/resume")

	assert.NilError(t, h.Deliver(t.Context(), Notice{Text: "the migration needs you"}))
	assert.Equal(t, f.handshakes(), 1, "the delivery opened a second connection")
	assert.Equal(t, count(f.methods(), "thread/resume"), 1,
		"the delivery resumed a thread this connection already follows")
	assert.Equal(t, count(f.methods(), "turn/start"), 1)
}

// count is how many of methods are one.
func count(methods []string, one string) int {
	var n int
	for _, method := range methods {
		if method == one {
			n++
		}
	}
	return n
}
