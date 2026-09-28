package mcp

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// newTestHTTP builds an HTTP server onto the stub, launched in an empty
// environment for the reason [newTestBridge] is.
func newTestHTTP(t *testing.T, svc chatv1.ChatServiceServer) *HTTPServer {
	t.Helper()

	srv, err := NewHTTP(slog.New(slog.DiscardHandler), serveTestDaemon(t, svc),
		func(string) string { return "" })
	assert.NilError(t, err)
	return srv
}

// servingHTTP is an HTTP server running on a loopback port, and the way to stop
// it.
type servingHTTP struct {
	endpoint string
	stop     func() error
}

// serveHTTP runs srv on a loopback port until the test ends or the case stops
// it. Loopback, because the SDK refuses a request that reached a loopback
// listener under any other host name.
func serveHTTP(t *testing.T, srv *HTTPServer) servingHTTP {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return srv.Serve(ctx, ln) })
	stop := func() error {
		cancel()
		return served.Wait()
	}
	// Waiting for Serve to return keeps the server's own goroutines from
	// outliving the test that owns the stub they are calling.
	t.Cleanup(func() { _ = stop() })
	return servingHTTP{endpoint: "http://" + ln.Addr().String() + HTTPPath, stop: stop}
}

// stampsToken is the launcher's side of a shared server: every request a
// session makes carries the token of the agent it belongs to. The empty token
// stamps nothing. A nil base sends through [http.DefaultTransport].
type stampsToken struct {
	token string
	base  http.RoundTripper
}

func (s stampsToken) RoundTrip(req *http.Request) (*http.Response, error) {
	if s.token != "" {
		req = req.Clone(req.Context())
		req.Header.Set(TokenHeader, s.token)
	}
	if s.base == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return s.base.RoundTrip(req)
}

// connectHTTP opens one MCP session on endpoint as the agent token names.
func connectHTTP(t *testing.T, endpoint, token string) *mcpsdk.ClientSession {
	t.Helper()
	return connectHTTPAs(t, endpoint, token, "test-harness")
}

// connectHTTPAs is [connectHTTP] under the name a client gives itself in the
// handshake.
func connectHTTPAs(t *testing.T, endpoint, token, clientName string) *mcpsdk.ClientSession {
	t.Helper()

	transport := &mcpsdk.StreamableClientTransport{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Transport: stampsToken{token: token}},
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: clientName, Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), transport, nil)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// sessionsOf is how many open sessions keep the member token names in the
// room, zero once it has left.
func sessionsOf(srv *HTTPServer, token string) int {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	if a := srv.members[token]; a != nil {
		return a.sessions
	}
	return 0
}

// Sessions naming one token are one member: it attends once, stays while any of
// them is open, and leaves the room when the last one closes — which the daemon
// sees as the attendance let go of.
//
// The SDK's client probes server/discover before it initializes, and the
// handler serves that probe on a session of its own that nobody closes. The
// member still leaves, because a probe binds nothing.
func TestHTTPServer_AMemberLeavesWhenItsLastSessionCloses(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	addMembersTool(srv)
	endpoint := serveHTTP(t, srv).endpoint

	first := connectHTTP(t, endpoint, testToken)
	second := connectHTTP(t, endpoint, testToken)
	for _, session := range []*mcpsdk.ClientSession{first, second} {
		res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "chat_members"})
		assert.NilError(t, err)
		assert.Assert(t, !res.IsError, "the tool answered %q", toolText(t, res))
	}
	assert.Equal(t, fake.openCount(), 1)
	assert.Equal(t, sessionsOf(srv, testToken), 2)

	assert.NilError(t, first.Close())
	waitFor(t, "the closed session still keeps the member", func() bool {
		return sessionsOf(srv, testToken) == 1
	})
	// One session is still open, so the member is still in the room.
	assert.Equal(t, fake.cancelCount(), 0)
	res, err := second.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "chat_members"})
	assert.NilError(t, err)
	assert.Assert(t, !res.IsError, "the tool answered %q", toolText(t, res))

	assert.NilError(t, second.Close())
	waitFor(t, "the member outlived its last session", func() bool {
		return fake.cancelCount() == 1
	})
	assert.Equal(t, sessionsOf(srv, testToken), 0)
}

// vanishingNetwork is the network a client reaches the server over, which the
// case can cut: every connection the client holds closes and every one it
// opens after is refused, the way a killed client or a host gone away leaves
// its server.
type vanishingNetwork struct {
	mu    sync.Mutex
	cut   bool
	conns []net.Conn
}

func (n *vanishingNetwork) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.cut {
		return nil, errors.New("the client is gone")
	}
	conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	n.conns = append(n.conns, conn)
	return conn, nil
}

func (n *vanishingNetwork) vanish() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.cut = true
	for _, conn := range n.conns {
		_ = conn.Close()
	}
}

// testKeepAlive is how often the server pings in the case below: a few
// intervals fit in the case's timeout, and a pong on loopback takes far less
// than the half interval the SDK waits for one.
const testKeepAlive = 100 * time.Millisecond

// A client that vanishes without the DELETE that ends its session answers no
// ping from then on, and the session is closed after a few of them, which
// takes its member out of the room. A client that is still there answers every
// ping and keeps its session.
func TestHTTPServer_AMemberLeavesWhenItsClientVanishes(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv, err := newHTTP(slog.New(slog.DiscardHandler), serveTestDaemon(t, fake),
		func(string) string { return "" }, testKeepAlive)
	assert.NilError(t, err)
	endpoint := serveHTTP(t, srv).endpoint

	network := &vanishingNetwork{}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-harness", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), &mcpsdk.StreamableClientTransport{
		Endpoint: endpoint,
		HTTPClient: &http.Client{Transport: stampsToken{
			token: testToken,
			base:  &http.Transport{DialContext: network.dial},
		}},
	}, nil)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })

	time.Sleep(5 * testKeepAlive)
	assert.Equal(t, sessionsOf(srv, testToken), 1)
	assert.Equal(t, fake.cancelCount(), 0)

	network.vanish()
	waitFor(t, "the member outlived its vanished client", func() bool {
		return fake.cancelCount() == 1
	})
	assert.Equal(t, sessionsOf(srv, testToken), 0)
}

// A session that comes back after its token's member left attends again as a
// new member.
func TestHTTPServer_AttendsAgainForASessionAfterTheMemberLeft(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	srv.host.pace.attendBackoffBase = 10 * time.Millisecond
	srv.host.pace.attendBackoffMax = 20 * time.Millisecond
	endpoint := serveHTTP(t, srv).endpoint

	first := connectHTTP(t, endpoint, testToken)
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.NilError(t, first.Close())
	waitFor(t, "the member outlived its last session", func() bool {
		return fake.cancelCount() == 1
	})

	connectHTTP(t, endpoint, testToken)
	waitFor(t, "the server never attended again", func() bool {
		return fake.attendCount() == 2
	})
}

// A session that names no token is served and acts as nobody: nothing attends
// on its behalf, and the tools say which header was missing.
func TestHTTPServer_ASessionWithoutATokenActsAsNobody(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	addMembersTool(srv)
	session := connectHTTP(t, serveHTTP(t, srv).endpoint, "")

	res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "chat_members"})
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	assert.Equal(t, toolText(t, res), errNoTokenHeader.Error())
	assert.Equal(t, fake.openCount(), 0)
}

// A server shutting down takes every member out of the room before it lets go
// of the daemon, and says it stopped because it was asked to.
func TestHTTPServer_ShuttingDownTakesItsMembersOutOfTheRoom(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	serving := serveHTTP(t, srv)
	connectHTTP(t, serving.endpoint, testToken)
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })

	err := serving.stop()
	assert.Assert(t, errors.Is(err, context.Canceled), "Serve returned %v", err)
	waitFor(t, "the member outlived the server", func() bool {
		return fake.cancelCount() == 1
	})
}

// countingHarness is a harness with a channel of its own that counts what it
// was handed, so a case can tell which session's harness a notice went through.
type countingHarness struct {
	delivered atomic.Int64
}

func (*countingHarness) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_OTHER }

func (*countingHarness) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

func (h *countingHarness) Deliver(context.Context, harnessctl.Notice) error {
	h.delivered.Add(1)
	return nil
}

// A member with several sessions hands a mention to the harness of the session
// that handshook last: that is the window of the agent a person opened most
// recently.
func TestHTTPServer_DeliversThroughTheLatestSessionsHarness(t *testing.T) {
	self := doneSelf("backend", "alice", testRoom)
	fake := &fakeChatService{self: self, events: make(chan *chatv1.RoomEvent)}
	srv := newTestHTTP(t, fake)
	byClient := map[string]*countingHarness{"first": {}, "second": {}}
	srv.host.detect = func(client string, _ func(string) string) harnessctl.Harness {
		return byClient[client]
	}
	endpoint := serveHTTP(t, srv).endpoint

	connectHTTPAs(t, endpoint, testToken, "first")
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	connectHTTPAs(t, endpoint, testToken, "second")

	pushEvent(t, fake, mentionOf(member("frontend", "bob", testRoom), self, "rebase please"))
	waitFor(t, "the mention was never delivered", func() bool {
		return byClient["second"].delivered.Load() == 1
	})
	assert.Equal(t, byClient["first"].delivered.Load(), int64(0))
	// Still one member: the second session joined the attendance the first opened.
	assert.Equal(t, fake.openCount(), 1)
}
