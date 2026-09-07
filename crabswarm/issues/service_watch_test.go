package issues

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"gotest.tools/v3/assert"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
	"github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1/issuesv1connect"
)

// TestWatchIssuesStreamsPolledChanges drives the change feed end to end, over
// the connect handler a browser talks to: a subscribed client sees the source
// registration, then the single diff the poll finds between two bd listings,
// and cancelling the client's context ends the stream and the handler with it.
func TestWatchIssuesStreamsPolledChanges(t *testing.T) {
	invocations := installFakeBd(t)
	// The first listing is the poller's baseline and the second differs from
	// it in one issue, crabswarm-jp7, whose update time moved. Every further
	// poll replays the second listing, so the diff happens once.
	t.Setenv("FAKE_BD_LIST_SEQUENCE", "list.json:list_changed.json")

	svc := newTestService(t, WithPollInterval(20*time.Millisecond))

	mux := http.NewServeMux()
	mux.Handle(issuesv1connect.NewIssuesServiceHandler(svc))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := issuesv1connect.NewIssuesServiceClient(srv.Client(), srv.URL)

	runCtx, cancelRun := context.WithCancel(t.Context())
	defer cancelRun()
	runDone := make(chan error, 1)
	go func() { runDone <- svc.Run(runCtx) }()
	// A source registered before Run installed its supervisor starts no
	// poller, and this test is about what the poller reports.
	waitRunning(t, svc)

	// The whole stream is driven from one goroutine: opening it blocks until
	// the handler sends its first message, so the call cannot sit on the test
	// goroutine ahead of the event that unblocks it.
	streamCtx, cancelStream := context.WithCancel(t.Context())
	defer cancelStream()
	events := make(chan *issuesv1.WatchIssuesResponse, subBuffer)
	streamErr := make(chan error, 1)
	go func() {
		stream, err := client.WatchIssues(streamCtx,
			connect.NewRequest(&issuesv1.WatchIssuesRequest{}))
		if err != nil {
			streamErr <- err
			return
		}
		defer func() { _ = stream.Close() }()
		for stream.Receive() {
			events <- stream.Msg()
		}
		streamErr <- stream.Err()
	}()
	// The handler subscribes as it starts, before anything is sent back;
	// publishing until then would be publishing to nobody.
	waitSubscribed(t, svc.hub)

	_, err := client.AddSource(t.Context(),
		connect.NewRequest(&issuesv1.AddSourceRequest{Dir: t.TempDir()}))
	assert.NilError(t, err)

	// Registering a source is a registry event: it names neither a source nor
	// issue ids, and a client answers it by re-listing the sources.
	assert.Assert(t, nextEvent(t, events, streamErr).GetSourcesChanged() != nil)

	changed := nextEvent(t, events, streamErr).GetIssuesChanged()
	assert.Assert(t, changed != nil)
	assert.Assert(t, changed.GetSourceId() != "")
	assert.DeepEqual(t, changed.GetIssueIds(), []string{"crabswarm-jp7"})

	// The polls keep running against an unchanged listing. None of them
	// reports a change, so the diff above was the only one.
	waitListings(t, invocations, 5)
	select {
	case ev := <-events:
		t.Fatalf("an unchanged listing published %v", ev)
	default:
	}

	cancelStream()
	assert.Equal(t, connect.CodeOf(waitStreamEnd(t, events, streamErr)), connect.CodeCanceled)

	// Close blocks until the handlers return, so it is what proves the server
	// side of the stream noticed the cancelled client rather than leaking a
	// goroutine parked on the hub.
	closed := make(chan struct{})
	go func() {
		srv.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("WatchIssues did not return after the client cancelled")
	}

	cancelRun()
	select {
	case err := <-runDone:
		assert.NilError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after context cancel")
	}
}

// TestWatchIssuesReportsChangeARequestFound drives the change feed with no
// poll behind it: [Service.Run] never runs, so the two ListIssues calls are
// the only listings this source ever gets. The second one is where the
// backlog moves, and a subscriber hears about it from that request rather
// than from a tick that has not happened.
func TestWatchIssuesReportsChangeARequestFound(t *testing.T) {
	installFakeBd(t)
	// The first listing is the baseline and the second differs from it in one
	// issue, crabswarm-jp7, whose update time moved.
	t.Setenv("FAKE_BD_LIST_SEQUENCE", "list.json:list_changed.json")

	svc := newTestService(t)

	mux := http.NewServeMux()
	mux.Handle(issuesv1connect.NewIssuesServiceHandler(svc))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	client := issuesv1connect.NewIssuesServiceClient(srv.Client(), srv.URL)

	streamCtx, cancelStream := context.WithCancel(t.Context())
	defer cancelStream()
	events := make(chan *issuesv1.WatchIssuesResponse, subBuffer)
	streamErr := make(chan error, 1)
	go func() {
		stream, err := client.WatchIssues(streamCtx,
			connect.NewRequest(&issuesv1.WatchIssuesRequest{}))
		if err != nil {
			streamErr <- err
			return
		}
		defer func() { _ = stream.Close() }()
		for stream.Receive() {
			events <- stream.Msg()
		}
		streamErr <- stream.Err()
	}()
	waitSubscribed(t, svc.hub)

	added, err := client.AddSource(t.Context(),
		connect.NewRequest(&issuesv1.AddSourceRequest{Dir: t.TempDir()}))
	assert.NilError(t, err)
	assert.Assert(t, nextEvent(t, events, streamErr).GetSourcesChanged() != nil)

	sourceID := added.Msg.GetSource().GetId()
	list := connect.NewRequest(&issuesv1.ListIssuesRequest{SourceId: sourceID})
	// A source read before any Run still has its poller, so this listing
	// records the baseline.
	_, err = client.ListIssues(t.Context(), list)
	assert.NilError(t, err)

	_, err = client.ListIssues(t.Context(), list)
	assert.NilError(t, err)

	changed := nextEvent(t, events, streamErr).GetIssuesChanged()
	assert.Assert(t, changed != nil)
	assert.Equal(t, changed.GetSourceId(), sourceID)
	assert.DeepEqual(t, changed.GetIssueIds(), []string{"crabswarm-jp7"})
}

// nextEvent takes the next stream message, failing the test rather than
// hanging when none arrives or the stream ends first.
func nextEvent(
	t *testing.T,
	events <-chan *issuesv1.WatchIssuesResponse,
	streamErr <-chan error,
) *issuesv1.WatchIssuesResponse {
	t.Helper()
	select {
	case ev := <-events:
		return ev
	case err := <-streamErr:
		t.Fatalf("the stream ended before an event arrived: %v", err)
		return nil
	case <-time.After(10 * time.Second):
		t.Fatal("no event received")
		return nil
	}
}

// waitStreamEnd returns the error the receive loop ended with, draining a
// message still in flight so the loop can reach the end at all.
func waitStreamEnd(
	t *testing.T,
	events <-chan *issuesv1.WatchIssuesResponse,
	streamErr <-chan error,
) error {
	t.Helper()
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-events:
		case err := <-streamErr:
			return err
		case <-deadline:
			t.Fatal("the stream did not end after the client cancelled")
			return nil
		}
	}
}

// waitRunning waits until Run installed the errgroup pollers start under.
func waitRunning(t *testing.T, s *Service) {
	t.Helper()
	waitFor(t, "Run did not start in time", func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return s.group != nil
	})
}

// waitSubscribed waits until a WatchIssues handler registered with the hub.
func waitSubscribed(t *testing.T, h *eventHub) {
	t.Helper()
	waitFor(t, "WatchIssues did not subscribe in time", func() bool {
		h.mu.Lock()
		defer h.mu.Unlock()
		return len(h.subs) > 0
	})
}

// waitListings waits until the fake bd recorded n whole-source listings, one
// per poll.
func waitListings(t *testing.T, invocations func() []invocation, n int) {
	t.Helper()
	waitFor(t, fmt.Sprintf("fewer than %d listings in time", n), func() bool {
		var listings int
		for _, inv := range invocations() {
			if strings.HasPrefix(inv.args, "list ") {
				listings++
			}
		}
		return listings >= n
	})
}

// waitFor polls cond until it holds, failing with msg when it does not.
func waitFor(t *testing.T, msg string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}
