package mcp

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// testToken is spelled out at every call site rather than left to resolution:
// this test binary may itself run under cmdman, and an empty token would
// quietly resolve to $CMDMAN_CMD_ID and talk as whoever is running the suite.
const testToken = "tok-a"

const testRoom = "/work/proj"

// eventTimeout bounds how long a test waits on something the server does off
// its own goroutines — attending again, ending an attendance. Generous enough
// to ride out a loaded machine and the retry backoff, short enough that a
// server that never does it fails instead of hanging the suite.
const eventTimeout = 5 * time.Second

// newTestBridge builds a server onto the stub. It carries no tool family: what
// this package does on its own is attend, and every case here is about that.
func newTestBridge(t *testing.T, svc *fakeChatService) *Server {
	t.Helper()

	bridge, err := New(slog.New(slog.DiscardHandler), serveTestDaemon(t, svc), testToken)
	assert.NilError(t, err)
	return bridge
}

// runsAt sets the loop going at a pace a case can wait for, for the cases that
// are about it attending again rather than about what one attendance did.
func runsAt(bridge *Server, base, max time.Duration) {
	bridge.attendBackoffBase = base
	bridge.attendBackoffMax = max
}

// serveBridge runs bridge over an in-memory pipe and returns the session a
// harness would hold.
func serveBridge(t *testing.T, bridge *Server) *mcpsdk.ClientSession {
	t.Helper()

	serverSide, clientSide := mcpsdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return bridge.Serve(ctx, serverSide) })

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-harness", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), clientSide, nil)
	assert.NilError(t, err)

	// Waiting for Serve to return keeps the server's own goroutines from
	// outliving the test that owns the stub they are calling.
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		_ = served.Wait()
	})
	return session
}

// waitFor blocks until want reports true, failing with why when it never does.
// The wait is what a case spends on something the server does off its own
// goroutines; the poll is short because everything here is local.
func waitFor(t *testing.T, why string, want func() bool) {
	t.Helper()

	deadline := time.Now().Add(eventTimeout)
	for time.Now().Before(deadline) {
		if want() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("%s within %s", why, eventTimeout)
}

// dropFeed ends the attendance the server is holding, the way the daemon does
// when it goes away or drops a reader that fell behind. It returns once the
// stub has taken the error, so the attendance the next assertion is about is
// the one that comes after this.
func dropFeed(t *testing.T, svc *fakeChatService, err error) {
	t.Helper()

	select {
	case svc.drops <- err:
	case <-time.After(eventTimeout):
		t.Fatal("nothing is attending the room")
	}
}

// A server whose daemon refuses it keeps asking, past any handful of attempts.
// A harness starts its MCP subprocesses before the services they talk to, so
// the daemon regularly arrives late; a server that stopped asking would leave
// the room a member short until the agent happened to call a tool.
func TestServer_KeepsAttendingUntilTheDaemonAdmitsIt(t *testing.T) {
	fake := &fakeChatService{
		self: member("backend", "alice", testRoom),
		err:  status.Error(codes.Unauthenticated, "unknown identity token"),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge)

	// More attempts than a bounded retry would have made.
	waitFor(t, "the server stopped asking to attend", func() bool {
		return fake.openCount() > 5
	})

	fake.setErr(nil)
	waitFor(t, "the server never attended once the daemon admitted it", func() bool {
		return fake.attendCount() == 1
	})
}

// The attendance is the stream, so a daemon that closes it has taken the
// membership with it — a restart, a reaped command, a watcher dropped for
// falling behind. The server opens it again with nothing asking it to, which is
// what the agent's hooks need: they report harness state through the CLI and
// are refused for as long as the room is missing the member.
func TestServer_AttendsAgainAfterTheDaemonEndsTheStream(t *testing.T) {
	fake := &fakeChatService{
		self:   member("backend", "alice", testRoom),
		events: make(chan *chatv1.RoomEvent),
		drops:  make(chan error),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge)

	waitFor(t, "the server never attended", func() bool {
		return fake.attendCount() == 1
	})

	dropFeed(t, fake, status.Error(codes.Unavailable, "the daemon is going away"))

	waitFor(t, "the server never attended again", func() bool {
		return fake.attendCount() == 2
	})
}

// watched is a resource registered the way a tool family registers one, so the
// gate below has something it may announce.
const watched = "crabswarm://test/watched"

// unwatchable is a URI nothing registered, spelled as one a harness might
// plausibly have reached for.
const unwatchable = "crabswarm://test/missing"

// registerWatched adds the resource a family would, which is what makes the
// server willing to announce it.
func registerWatched(bridge *Server) {
	bridge.AddResource(
		&mcpsdk.Resource{Name: "watched", URI: watched, MIMEType: "application/json"},
		func(context.Context, *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
			return &mcpsdk.ReadResourceResult{}, nil
		},
	)
}

// assertNotFound asserts uri was refused as the missing resource the SDK
// spells. Pinned as that error rather than as any error at all: it is what
// tells a harness the URI is not one this server has, so it stops waiting on
// news that could never come.
func assertNotFound(t *testing.T, uri string, err error) {
	t.Helper()

	assert.Assert(t, errors.Is(err, mcpsdk.ResourceNotFoundError(uri)),
		"%s was not refused as a missing resource: %v", uri, err)
}

// Only a registered resource may be subscribed to. A URI no family registered
// is refused because the SDK would otherwise record a subscription for whatever
// it was handed and leave the harness waiting.
//
// The handler is exercised directly because the protocol the SDK negotiates
// here opens a subscription without waiting for the answer, so a refusal never
// reaches the client as the error of a call.
func TestServer_RefusesToWatchWhatItCannotAnnounce(t *testing.T) {
	bridge, err := New(slog.New(slog.DiscardHandler),
		serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })
	registerWatched(bridge)

	assertNotFound(t, unwatchable, bridge.subscribed(t.Context(), &mcpsdk.SubscribeRequest{
		Params: &mcpsdk.SubscribeParams{URI: unwatchable},
	}))
	assert.NilError(t, bridge.subscribed(t.Context(), &mcpsdk.SubscribeRequest{
		Params: &mcpsdk.SubscribeParams{URI: watched},
	}))
}

// Withdrawing is answered by the same gate as asking: a harness that was
// refused a subscription has none to withdraw, so telling it the withdrawal
// succeeded would say it had had one.
//
// Exercised directly for the reason the subscribe side is: the SDK does not
// hand either refusal back as the error of a client call.
func TestServer_RefusesToUnwatchWhatItCannotAnnounce(t *testing.T) {
	bridge, err := New(slog.New(slog.DiscardHandler),
		serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })
	registerWatched(bridge)

	unsubscribe := func(uri string) error {
		return bridge.unsubscribed(t.Context(), &mcpsdk.UnsubscribeRequest{
			Params: &mcpsdk.UnsubscribeParams{URI: uri},
		})
	}

	assert.NilError(t, unsubscribe(watched))
	assertNotFound(t, unwatchable, unsubscribe(unwatchable))
}

func TestNew_RejectsEmptySocketPath(t *testing.T) {
	_, err := New(nil, "", testToken)
	assert.Assert(t, err != nil)
}

// The retry loop comes out of New with a pace of its own. The fields exist so a
// test can slow it down or hold it still, and a server nobody tuned would
// otherwise run it at a zero wait — asking the daemon as fast as it can answer
// for the whole session.
func TestNew_SeedsTheRetryPace(t *testing.T) {
	bridge, err := New(nil, serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })

	assert.Assert(t, bridge.attendBackoffBase > 0,
		"attending starts at %s", bridge.attendBackoffBase)
	assert.Assert(t, bridge.attendBackoffMax >= bridge.attendBackoffBase,
		"attending climbs to %s, below its first wait of %s",
		bridge.attendBackoffMax, bridge.attendBackoffBase)
}
