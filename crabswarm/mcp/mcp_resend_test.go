package mcp

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// What a feed last said is repeated for as long as the session runs, and these
// cases are about the three things that decides: the repeat happens, it happens
// after a report the daemon turned down, and it does not happen before a feed
// has said anything at all.

// testResend is the pace the repeat runs at here — short enough that a case
// sees several inside its timeout, long enough that the first one does not race
// the feed's own report.
const testResend = 20 * time.Millisecond

// attendingStub is the stub as these cases read it, which is the stub itself
// and anything wrapping it.
type attendingStub interface {
	chatv1.ChatServiceServer
	attendCount() int
	reportedStates() []chatv1.HarnessState
}

// watchedBridge serves a bridge whose harness carries a feed, already attending
// and repeating at testResend, and returns the feed the case speaks through.
func watchedBridge(t *testing.T, svc attendingStub) *feedingHarness {
	t.Helper()

	bridge := newTestBridge(t, svc)
	bridge.harnessStateResend = testResend
	feed := &feedingHarness{states: make(chan chatv1.HarnessState)}
	runsOn(bridge, feed)
	serveBridge(t, bridge)

	waitFor(t, "the server never attended", func() bool { return svc.attendCount() == 1 })
	return feed
}

// assertOnly asserts the member reported want and nothing else, however many
// times the repeat said it.
func assertOnly(t *testing.T, got []chatv1.HarnessState, want chatv1.HarnessState) {
	t.Helper()

	for i, state := range got {
		assert.Equal(t, state, want, "report %d of %d", i+1, len(got))
	}
}

// refusingChat is the stub with the first of the state reports turned down,
// which is the daemon refusing one — a restart caught mid-report, a member it
// does not have. The attempts are counted because a refused report records
// nothing, so the trail alone would not say one was ever made.
type refusingChat struct {
	*fakeChatService

	// refusals is how many of the first reports are turned down.
	refusals int64
	attempts atomic.Int64
}

func (f *refusingChat) ReportState(
	ctx context.Context, req *chatv1.ReportStateRequest,
) (*chatv1.ReportStateResponse, error) {
	if f.attempts.Add(1) <= f.refusals {
		return nil, status.Error(codes.Unavailable, "the daemon is going away")
	}
	return f.fakeChatService.ReportState(ctx, req)
}

// A state a feed reported once keeps reaching the daemon. The feed speaks on
// change alone, so without this a daemon that came back to a member it records
// as done would hold that until the agent's next transition.
func TestServer_SaysTheLastHarnessStateAgain(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	feed := watchedBridge(t, fake)

	feed.says(t, chatv1.HarnessState_HARNESS_STATE_WORKING)

	// The feed is handed nothing else, so every report past the first is the
	// repeat saying that one again.
	waitFor(t, "the state the feed reported was never said again", func() bool {
		return len(fake.reportedStates()) > 1
	})
	assertOnly(t, fake.reportedStates(), chatv1.HarnessState_HARNESS_STATE_WORKING)
	assert.Equal(t, feed.watches.Load(), int64(1))
}

// A report the daemon turned down is made again, rather than left until the
// feed's next change: the daemon holds the wrong state meanwhile, and a
// change-only feed would never say this one twice.
func TestServer_SaysAgainWhatTheDaemonRefused(t *testing.T) {
	fake := &refusingChat{
		fakeChatService: &fakeChatService{self: doneSelf("backend", "alice", testRoom)},
		refusals:        1,
	}
	feed := watchedBridge(t, fake)

	// The feed speaks once, and that report is the one refused — it is the first
	// attempt, since nothing is repeated until a feed has spoken. Whatever the
	// daemon went on to record is therefore the repeat.
	feed.says(t, chatv1.HarnessState_HARNESS_STATE_WORKING)

	waitFor(t, "the refused report was never made again", func() bool {
		return len(fake.reportedStates()) > 0
	})
	assertOnly(t, fake.reportedStates(), chatv1.HarnessState_HARNESS_STATE_WORKING)
}

// A session whose feed never said anything says nothing again. The attendance
// came back carrying a state, and repeating that would be this server telling
// the daemon what the daemon had just told it.
func TestServer_SaysNothingAgainUntilAFeedHasSpoken(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	watchedBridge(t, fake)

	// Several intervals, so the quiet is the repeat finding nothing to say
	// rather than the repeat not having run yet.
	time.Sleep(10 * testResend)
	assert.Equal(t, len(fake.reportedStates()), 0)
}
