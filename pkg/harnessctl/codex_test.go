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
	"golang.org/x/sync/semaphore"
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
		path:    f.path,
		logger:  slog.New(slog.NewTextHandler(log, nil)),
		binding: semaphore.NewWeighted(1),
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
			held: "holds 0 threads, none of them a person's",
		},
		{
			name:   "only helpers are loaded",
			loaded: []string{"helper-system", "helper-guardian"},
			sources: map[string]string{
				"helper-system":   "system",
				"helper-guardian": "guardian_review",
			},
			held: "holds 2 threads, none of them a person's",
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

	// Binding says what the thread is doing before anything is pushed: the
	// recorded resume answer holds it idle.
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)

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
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
	f.disconnect()

	// The connection made again binds again, and says what the thread is doing
	// the way the first one did.
	waitCodexHandshakes(t, f, 2)
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
	f.push(f.rec.notification(t, codexStatusChanged, 0))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WORKING)
}

// A feed that connected before the session had a thread follows none, and
// binds the moment the app server announces one: no delivery is needed to find
// the thread, and no turn is started on it.
func TestCodex_WatchBindsTheThreadAStartAnnounces(t *testing.T) {
	f := startCodexFake(t)
	thread := f.rec.threadId(t)
	h, _ := newCodexHarness(f)

	states := watchCodex(t, h)
	f.waitAsked(t, "thread/loaded/list")
	f.update(func() { f.loaded = []string{thread} })
	f.push(codexThreadStartedFrame(thread))

	// Well inside the first retry, so the bind is the announcement's doing and
	// not the timer's; the trail then holds nothing the timer asked for.
	select {
	case state := <-states:
		assert.Equal(t, state, chatv1.HarnessState_HARNESS_STATE_DONE)
	case <-time.After(codexRebindInterval / 2):
		t.Fatalf("the feed did not bind on the announcement; the fake was asked %v",
			f.methods())
	}
	assert.DeepEqual(t, f.methods(), []string{
		"initialize", "initialized", "thread/loaded/list",
		"thread/started", "thread/loaded/list", "thread/resume",
	})
}

// A thread loaded without an announcement is found all the same: a feed that
// follows nothing looks again every few seconds, and stops looking once it
// follows a thread.
func TestCodex_WatchLooksAgainUntilItFollowsAThread(t *testing.T) {
	f := startCodexFake(t)
	thread := f.rec.threadId(t)
	h, _ := newCodexHarness(f)

	states := watchCodex(t, h)
	f.waitAsked(t, "thread/loaded/list")
	f.update(func() { f.loaded = []string{thread} })

	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.DeepEqual(t, f.threadsOf(t, "thread/resume"), []string{thread})
	listed := count(f.methods(), "thread/loaded/list")

	time.Sleep(codexRebindInterval + codexRebindInterval/2)
	assert.Equal(t, count(f.methods(), "thread/loaded/list"), listed,
		"the feed kept looking for a thread it already follows: %v", f.methods())
	assert.Assert(t, !slices.Contains(f.methods(), "turn/start"))
}

// A TUI stopped and started again leaves its old thread loaded beside the new
// one for a while. The new thread's announcement moves the feed to it: the old
// thread is unsubscribed, the new one resumed, and what the new one's resume
// answer says is reported on its own, since an idle replacement pushes nothing
// until its first turn and the room would otherwise keep the old thread's
// working.
func TestCodex_WatchFollowsTheSessionToItsReplacement(t *testing.T) {
	f := startCodexFake(t)
	old := f.rec.threadId(t)
	f.loaded = []string{old}
	f.resumeStatus = map[string]string{old: `{"type":"active","activeFlags":[]}`}
	h, _ := newCodexHarness(f)

	states := watchCodex(t, h)
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WORKING)

	f.update(func() {
		f.loaded = []string{old, codexLaterSession}
		f.recency = map[string]int64{old: 100, codexLaterSession: 200}
	})
	f.push(codexThreadStartedFrame(codexLaterSession))

	// The new thread keeps the recorded resume answer, which is idle.
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.DeepEqual(t, f.methods(), []string{
		"initialize", "initialized", "thread/loaded/list", "thread/resume",
		"thread/started", "thread/loaded/list", "thread/read", "thread/read",
		"thread/unsubscribe", "thread/resume",
	})
	assert.DeepEqual(t, f.threadsOf(t, "thread/unsubscribe"), []string{old})
	assert.DeepEqual(t, f.threadsOf(t, "thread/resume"), []string{old, codexLaterSession})

	// From here on the old thread is one the feed reads past.
	f.push(f.rec.notification(t, codexStatusChanged, 0))
	f.push(codexWaitingFrame(codexLaterSession))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WAITING)
}

// A helper Codex runs beside the session goes active and idle on its own, and
// none of that is the session's: a state about any thread but the one the feed
// follows, loaded or not, reports nothing.
func TestCodex_WatchReadsPastThreadsItDoesNotFollow(t *testing.T) {
	f := startCodexFake(t)
	thread := f.rec.threadId(t)
	const helper = "helper-system"
	f.loaded = []string{helper, thread}
	f.sources = map[string]string{helper: "system"}
	h, _ := newCodexHarness(f)

	states := watchCodex(t, h)
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)

	for _, other := range []string{helper, codexEarlierSession} {
		f.push(codexFrameAbout(t, f.rec.notification(t, codexStatusChanged, 0), other))
		f.push(codexFrameAbout(t, f.rec.notification(t, codexTurnCompleted, 0), other))
	}
	// The session's own state follows twice. Each frame is handed on in its own
	// goroutine, so a state about another thread that got through could land on
	// either side of the first; it cannot hide from both.
	f.push(codexWaitingFrame(thread))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WAITING)
	f.push(codexWaitingFrame(thread))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WAITING)
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

// A delivery riding the feed's connection and the feed moving to a new session
// thread each list the loaded threads and then subscribe, and they take turns
// doing it. Here the delivery is still reading the threads when a new TUI's
// thread is announced. The feed waits for the delivery, which starts its turn
// on the thread a person was in when it listed them, and then moves to the new
// thread. The delivery resumes nothing over the top of the move.
func TestCodex_DeliveryAndTheFeedTakeTurnsBinding(t *testing.T) {
	f := startCodexFake(t)
	old := f.rec.threadId(t)
	const helper = "helper-system"
	f.loaded = []string{old}
	h, _ := newCodexHarness(f)

	states := watchCodex(t, h)
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)

	// A helper beside the session makes the delivery read the threads, and its
	// first read is kept back so the announcement lands in the middle of it.
	f.update(func() {
		f.loaded = []string{old, helper}
		f.sources = map[string]string{helper: "system"}
	})
	release := f.holdNext(t, "thread/read")
	var delivering errgroup.Group
	delivering.Go(func() error {
		return h.Deliver(t.Context(), Notice{Text: "the migration needs you"})
	})
	f.waitAsked(t, "thread/read")

	f.update(func() {
		f.loaded = []string{old, helper, codexLaterSession}
		f.recency = map[string]int64{old: 100, codexLaterSession: 200}
	})
	f.push(codexThreadStartedFrame(codexLaterSession))

	// A feed that did not wait would list the threads again in this window. The
	// delivery would then resume the thread it had listed over the top of the
	// feed's move, and leave the feed following the stopped TUI's thread.
	time.Sleep(codexRebindInterval / 4)
	assert.Equal(t, count(f.methods(), "thread/loaded/list"), 2,
		"the feed bound again while a delivery was binding: %v", f.methods())
	release()
	assert.NilError(t, delivering.Wait())

	// The new thread keeps the recorded resume answer, which is idle.
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_DONE)
	assert.Equal(t, f.handshakes(), 1, "the delivery opened a second connection")
	assert.Equal(t, deliveredTo(t, f), old)
	assert.Equal(t, h.borrow().subscribed(), codexLaterSession)
	assert.DeepEqual(t, f.methods(), []string{
		"initialize", "initialized", "thread/loaded/list", "thread/resume",
		"thread/loaded/list", "thread/read", "thread/started", "thread/read", "turn/start",
		"thread/loaded/list", "thread/read", "thread/read", "thread/read",
		"thread/unsubscribe", "thread/resume",
	})
	assert.DeepEqual(t, f.threadsOf(t, "thread/resume"), []string{old, codexLaterSession})

	// The feed hears the new thread, which is what the move was for.
	f.push(codexWaitingFrame(codexLaterSession))
	assert.Equal(t, nextState(t, states), chatv1.HarnessState_HARNESS_STATE_WAITING)
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
