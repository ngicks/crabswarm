package notify

import (
	"context"
	"log/slog"
	"maps"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/internal/cmdman"
)

// DefaultScreenPollInterval is how often a screen is read when the
// configuration names no interval. Short enough that an operator watching the
// roster sees a turn end about when it ends, long enough that a room of agents
// costs a handful of cmdman invocations a second rather than a flood.
const DefaultScreenPollInterval = 3 * time.Second

// screenCaptureTimeout bounds one capture. A cmdman that hangs would otherwise
// hold the whole sweep, and every member behind it would go unread for as long
// as it hung.
const screenCaptureTimeout = 2 * time.Second

// ClassifyScreen reads a terminal snapshot as the harness state it shows, and
// reports whether it shows one at all.
//
// The markers and their order are [cmdman.ScreenMarkers]; the first that
// matches settles the screen. A snapshot matching none of them is a screen this
// classifier cannot read — a harness mid-repaint, a session showing something
// nobody wrote a marker for — and gets no verdict rather than a guessed one,
// because a wrong "done" is what puts a keystroke into a terminal mid-turn.
func ClassifyScreen(snapshot string) (chat.MemberState, bool) {
	for _, m := range cmdman.ScreenMarkers {
		if m.Matches(snapshot) {
			return m.State, true
		}
	}
	return "", false
}

// AttendingMembers lists everyone attending the broker right now, across every
// room. [chat.Store] is what the daemon passes.
type AttendingMembers interface {
	Attending(ctx context.Context) ([]chat.Member, error)
}

// StateRecorder records the state a member is observed in, the same way a
// harness hook's report is recorded: the store keeps it, the status display
// follows it, and the room hears about a change. [chat.Service] is what the
// daemon passes.
type StateRecorder interface {
	RecordState(ctx context.Context, token string, state chat.MemberState) error
}

// screenCapturer snapshots what a member's terminal is showing. It is declared
// here, at its consumer, so a test can hand the poller a screen without a
// cmdman to exec; [cmdman.Terminal] is what the daemon runs.
type screenCapturer interface {
	CaptureScreen(ctx context.Context, token string) (string, error)
}

// ScreenPoller keeps the recorded state of an attending session whose harness
// nobody recognised in step with what its terminal is actually showing.
//
// Every harness this daemon names reports on a feed of its own that its bridge
// watches: Codex its app server, OpenCode its plugin events, Claude Code the
// agents listing its own `crabswarm mcp` server polls. Reading a screen beside
// such a feed only reintroduces a wrong "done" — the empty composer line reads
// as idle while a subagent running in the background keeps the session working
// — so a harness with a feed is left to that feed.
//
// This is the fallback for a harness that has no feed, so only agents attending
// as [chat.HarnessOther] are polled: for those, the screen is the only thing
// that says what the session is doing. A member that declared no harness at all
// is left alone too — nothing says a harness is sitting in that terminal — and a
// member that is not an agent has no terminal to read.
type ScreenPoller struct {
	interval time.Duration
	capture  screenCapturer
	members  AttendingMembers
	states   StateRecorder
	logger   *slog.Logger
	// failing holds the members whose last capture failed, so a terminal that
	// cannot be read reports itself once instead of at every interval. Only
	// [ScreenPoller.Run]'s goroutine touches it.
	failing map[string]struct{}
}

// NewScreenPoller returns a poller that captures screens through the cmdman
// binary named by bin, lists its subjects through members and records what it
// reads through states. An empty bin means "cmdman", resolved on PATH, the same
// default the token resolver uses; a nil logger discards logs.
//
// A non-positive interval takes [DefaultScreenPollInterval]. Turning the
// polling off is the caller's decision rather than this constructor's: a poller
// that should not run is one that is never built.
func NewScreenPoller(
	bin string,
	interval time.Duration,
	members AttendingMembers,
	states StateRecorder,
	logger *slog.Logger,
) *ScreenPoller {
	if interval <= 0 {
		interval = DefaultScreenPollInterval
	}
	term := cmdman.NewTerminal(bin, logger)
	return &ScreenPoller{
		interval: interval,
		capture:  term,
		members:  members,
		states:   states,
		// The Terminal's logger, not the argument: it has already been defaulted,
		// so a nil logger is turned into a discarding one in exactly one place.
		logger:  term.Logger(),
		failing: make(map[string]struct{}),
	}
}

// Run reads every watched screen once per interval until ctx ends.
//
// Nothing is reported back: a capture that fails and a screen that says nothing
// are both ordinary — a session ends, a terminal repaints — and the answer to
// either is the next tick. A sweep that outlasts its interval simply skips the
// ticks it ran through.
func (p *ScreenPoller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

// poll reads the screen of every attending agent whose harness has no feed of
// its own once, in roster order.
//
// One at a time: a room holds a handful of agents, and capturing them in
// parallel would spend goroutines and cmdman processes to shave milliseconds
// off an interval measured in seconds.
func (p *ScreenPoller) poll(ctx context.Context) {
	members, err := p.members.Attending(ctx)
	if err != nil {
		p.logger.Debug("chat: listing attendance to poll screens failed", "err", err)
		return
	}
	watched := make(map[string]struct{}, len(members))
	for _, m := range members {
		if m.Kind != chat.KindAgent || m.Harness != chat.HarnessOther {
			continue
		}
		watched[m.Token] = struct{}{}
		p.pollMember(ctx, m)
	}
	// A member that stopped attending takes its failure record with it, so the
	// next session under the same token reports its first failure as a first one.
	maps.DeleteFunc(p.failing, func(token string, _ struct{}) bool {
		_, ok := watched[token]
		return !ok
	})
}

// pollMember reads one member's screen and records what it says. A screen that
// could not be captured or could not be read records nothing: the state already
// held is a better answer than a guess, and the next tick asks again.
func (p *ScreenPoller) pollMember(ctx context.Context, m chat.Member) {
	who := m.Team + "/" + m.Name

	captureCtx, cancel := context.WithTimeout(ctx, screenCaptureTimeout)
	defer cancel()
	snapshot, err := p.capture.CaptureScreen(captureCtx, m.Token)
	if err != nil {
		p.reportCaptureFailure(m.Token, who, err)
		return
	}
	delete(p.failing, m.Token)

	state, ok := ClassifyScreen(snapshot)
	if !ok {
		p.logger.Debug("chat: the screen says nothing about the member's state",
			"member", who)
		return
	}
	if err := p.states.RecordState(ctx, m.Token, state); err != nil {
		p.logger.Debug("chat: recording the state read off the screen failed",
			"member", who, "state", state, "err", err)
	}
}

// reportCaptureFailure logs a capture that failed, loudly the first time and
// quietly after that. The first failure is worth an operator's attention — a
// member whose screen cannot be read is one whose state stops being corrected —
// while the same failure at every interval would be noise.
func (p *ScreenPoller) reportCaptureFailure(token, who string, err error) {
	if _, repeat := p.failing[token]; repeat {
		p.logger.Debug("chat: capturing the member's screen failed again",
			"member", who, "err", err)
		return
	}
	p.failing[token] = struct{}{}
	p.logger.Info("chat: capturing the member's screen failed",
		"member", who, "err", err)
}
