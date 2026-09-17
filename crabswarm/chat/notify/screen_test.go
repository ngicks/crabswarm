package notify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"gotest.tools/v3/assert"
)

// harnessFixtureDir holds the recorded harness screens the classifier is pinned
// against. They are read from the e2e suite's testdata rather than copied here:
// one recording of a real session, read by whatever reads screens.
const harnessFixtureDir = "../../../e2e/crabswarm/testdata/harness"

// harnessScreen reads one recorded screen.
func harnessScreen(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(harnessFixtureDir, name))
	assert.NilError(t, err)
	return string(b)
}

func TestClassifyScreen_Fixtures(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		want    chat.MemberState
	}{
		// The prompt stands empty and no turn is running above it.
		{"claude-screen-idle.txt", chat.StateDone},
		// A tool is running in the foreground and the spinner is counting. The
		// prompt below it is empty too, and the turn that ended before this one
		// still reads "done" on screen, so neither of those may win.
		{"claude-screen-working.txt", chat.StateWorking},
		// A permission dialog has replaced the whole footer, status line
		// included, and is waiting for an answer.
		{"claude-screen-dialog.txt", chat.StateWaiting},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			got, ok := ClassifyScreen(harnessScreen(t, tc.fixture))
			assert.Assert(t, ok, "the fixture must classify")
			assert.Equal(t, got, tc.want)
		})
	}
}

func TestClassifyScreen_Unclassifiable(t *testing.T) {
	for _, tc := range []struct {
		name     string
		snapshot string
	}{
		// A terminal that answered nothing is not a terminal at a prompt.
		{"empty", ""},
		{"blank lines", "\n\n\n"},
		// A plain shell prompt is not the harness composer, so nothing here
		// claims to know what the session is doing.
		{"a shell prompt", "$ echo hello\nhello\n$ "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ClassifyScreen(tc.snapshot)
			assert.Assert(t, !ok, "classified as %q", got)
			assert.Equal(t, got, chat.MemberState(""))
		})
	}
}

// The spinner is read off its elapsed counter, because a configured status line
// replaces the footer the "(esc to interrupt)" hint belongs to. The same
// ellipsis ends an over-long status line, which must not read as a running turn.
func TestClassifyScreen_SpinnerAndTruncation(t *testing.T) {
	spinner, ok := ClassifyScreen("* Generating… (19s · ↓ 193 tokens)\n────\n❯ \n────\n")
	assert.Assert(t, ok)
	assert.Equal(t, spinner, chat.StateWorking)

	truncated, ok := ClassifyScreen(
		"────\n❯ \n────\n Haiku 4.5 | 16% used | /tmp/claude-0/-home-watage-gitrepo-ngic…\n" +
			"  ⏸ manual mode on · ← for agents\n")
	assert.Assert(t, ok)
	assert.Equal(t, truncated, chat.StateDone)
}

// Done is the whole prompt line standing empty. A "❯" anywhere is not it: the
// dialog's first choice carries one, and so does every prompt already answered.
func TestClassifyScreen_PromptMustBeEmpty(t *testing.T) {
	answered, ok := ClassifyScreen("❯ Reply with the single word ok.\n")
	assert.Assert(t, !ok, "classified as %q", answered)
}

// fakeScreens answers each token's captures from a script, one entry per
// capture, holding the last once the script runs out — the way a terminal keeps
// showing its last screen. An entry that is an error is a capture that failed.
type fakeScreens struct {
	mu     sync.Mutex
	script map[string][]screenAnswer
	taken  map[string]int
}

type screenAnswer struct {
	screen string
	err    error
}

func newFakeScreens(script map[string][]screenAnswer) *fakeScreens {
	return &fakeScreens{script: script, taken: make(map[string]int)}
}

func (f *fakeScreens) CaptureScreen(_ context.Context, token string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	answers, ok := f.script[token]
	if !ok || len(answers) == 0 {
		return "", errors.New("no command found matching " + token)
	}
	f.taken[token]++
	a := answers[min(f.taken[token], len(answers))-1]
	return a.screen, a.err
}

// captured reports how many times token's screen was read.
func (f *fakeScreens) captured(token string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.taken[token]
}

// fakeAttendance is the roster the poller walks.
type fakeAttendance struct {
	members []chat.Member
	err     error
}

func (f fakeAttendance) Attending(context.Context) ([]chat.Member, error) {
	return f.members, f.err
}

// fakeRecorder collects what the poller recorded, as "token=state" lines, and
// fails the calls whose token is in refuse — a member whose session ended
// between the listing and the reading.
type fakeRecorder struct {
	mu       sync.Mutex
	recorded []string
	refuse   map[string]struct{}
}

func (f *fakeRecorder) RecordState(_ context.Context, token string, state chat.MemberState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.refuse[token]; ok {
		return errors.New("not attending")
	}
	f.recorded = append(f.recorded, token+"="+string(state))
	return nil
}

func (f *fakeRecorder) states() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.recorded)
}

// claudeAgent is an attending Claude Code session, the one kind of member the
// poller reads.
func claudeAgent(token, name string) chat.Member {
	return chat.Member{
		Token:   token,
		Name:    name,
		Team:    "alpha",
		Room:    "/work",
		Kind:    chat.KindAgent,
		Harness: chat.HarnessClaudeCode,
		State:   chat.StateDone,
	}
}

// pollOnce runs one sweep of a poller built around the fakes, which is what the
// ticker would have run.
func pollOnce(
	t *testing.T,
	members []chat.Member,
	screens *fakeScreens,
	recorder *fakeRecorder,
	sweeps int,
) {
	t.Helper()
	poller := NewScreenPoller("", time.Hour, fakeAttendance{members: members}, recorder, nil)
	poller.capture = screens
	for range sweeps {
		poller.poll(t.Context())
	}
}

// Each sweep records what that member's screen showed, so a state nobody
// reported still lands — which is the whole point of reading the screen.
func TestScreenPoller_RecordsWhatTheScreenShows(t *testing.T) {
	screens := newFakeScreens(map[string][]screenAnswer{
		"tok-ana": {
			{screen: harnessScreen(t, "claude-screen-idle.txt")},
			{screen: harnessScreen(t, "claude-screen-working.txt")},
			{screen: harnessScreen(t, "claude-screen-dialog.txt")},
		},
	})
	recorder := &fakeRecorder{}
	pollOnce(t, []chat.Member{claudeAgent("tok-ana", "ana")}, screens, recorder, 3)

	assert.DeepEqual(t, recorder.states(), []string{
		"tok-ana=done",
		"tok-ana=working",
		"tok-ana=waiting",
	})
}

// Only an agent running Claude Code is read. Every other harness reports its own
// state, and a member that is not an agent has no terminal to read at all.
func TestScreenPoller_ReadsOnlyClaudeCodeAgents(t *testing.T) {
	codex := claudeAgent("tok-bob", "bob")
	codex.Harness = chat.HarnessCodex
	human := claudeAgent("tok-cid", "cid")
	human.Kind = chat.KindHuman
	human.Harness = ""
	unnamed := claudeAgent("tok-dee", "dee")
	unnamed.Harness = ""

	screens := newFakeScreens(map[string][]screenAnswer{
		"tok-ana": {{screen: harnessScreen(t, "claude-screen-working.txt")}},
		"tok-bob": {{screen: harnessScreen(t, "claude-screen-working.txt")}},
		"tok-cid": {{screen: harnessScreen(t, "claude-screen-working.txt")}},
		"tok-dee": {{screen: harnessScreen(t, "claude-screen-working.txt")}},
	})
	recorder := &fakeRecorder{}
	pollOnce(t, []chat.Member{claudeAgent("tok-ana", "ana"), codex, human, unnamed},
		screens, recorder, 1)

	assert.DeepEqual(t, recorder.states(), []string{"tok-ana=working"})
	for _, token := range []string{"tok-bob", "tok-cid", "tok-dee"} {
		assert.Equal(t, screens.captured(token), 0,
			"%s must never have its screen read", token)
	}
}

// A screen that could not be captured, one nothing in the table matches, and a
// member whose session ended before the reading landed all record nothing. The
// state already held is a better answer than a guess, and the next sweep asks
// again.
func TestScreenPoller_RecordsNothingItCannotRead(t *testing.T) {
	screens := newFakeScreens(map[string][]screenAnswer{
		"tok-ana": {{err: errors.New("no command found matching tok-ana")}},
		"tok-bob": {{screen: "$ echo hello\nhello\n$ "}},
		"tok-cid": {{screen: harnessScreen(t, "claude-screen-idle.txt")}},
	})
	recorder := &fakeRecorder{refuse: map[string]struct{}{"tok-cid": {}}}
	pollOnce(t, []chat.Member{
		claudeAgent("tok-ana", "ana"),
		claudeAgent("tok-bob", "bob"),
		claudeAgent("tok-cid", "cid"),
	}, screens, recorder, 2)

	assert.Equal(t, len(recorder.states()), 0, "recorded %v", recorder.states())
	// Every one of them is still read at the next sweep: none of these is a
	// reason to stop watching a member.
	for _, token := range []string{"tok-ana", "tok-bob", "tok-cid"} {
		assert.Equal(t, screens.captured(token), 2)
	}
}

// A member that stopped attending is forgotten, so the next session under its
// token reports its first failure as a first one rather than as a repeat.
func TestScreenPoller_ForgetsMembersThatLeft(t *testing.T) {
	screens := newFakeScreens(map[string][]screenAnswer{
		"tok-ana": {{err: errors.New("no command found matching tok-ana")}},
	})
	poller := NewScreenPoller("", time.Hour,
		fakeAttendance{members: []chat.Member{claudeAgent("tok-ana", "ana")}},
		&fakeRecorder{}, nil)
	poller.capture = screens

	poller.poll(t.Context())
	assert.Equal(t, len(poller.failing), 1)

	poller.members = fakeAttendance{}
	poller.poll(t.Context())
	assert.Equal(t, len(poller.failing), 0)
}

// A roster that cannot be listed ends the sweep rather than the poller: the
// ticker asks again.
func TestScreenPoller_SurvivesAFailedListing(t *testing.T) {
	screens := newFakeScreens(nil)
	poller := NewScreenPoller("", time.Hour,
		fakeAttendance{err: errors.New("store is closed")}, &fakeRecorder{}, nil)
	poller.capture = screens

	poller.poll(t.Context())

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	poller.Run(ctx) // returns rather than blocking on a cancelled context
}

func TestNewScreenPoller_DefaultsTheInterval(t *testing.T) {
	roster := fakeAttendance{}
	recorder := &fakeRecorder{}
	assert.Equal(t,
		NewScreenPoller("", 0, roster, recorder, nil).interval, DefaultScreenPollInterval)
	assert.Equal(t,
		NewScreenPoller("", 250*time.Millisecond, roster, recorder, nil).interval,
		250*time.Millisecond)
}
