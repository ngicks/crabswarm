package mcp

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// stampsHeaders is a launcher that sets more than the token on every request a
// session makes: what the session declares about how it is woken.
type stampsHeaders struct {
	headers map[string]string
}

func (s stampsHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	for k, v := range s.headers {
		req.Header.Set(k, v)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// connectDeclaring opens one MCP session on endpoint carrying headers.
func connectDeclaring(
	t *testing.T, endpoint string, headers map[string]string,
) *mcpsdk.ClientSession {
	t.Helper()

	transport := &mcpsdk.StreamableClientTransport{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Transport: stampsHeaders{headers: headers}},
	}
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-harness", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), transport, nil)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestParseNudge(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  chatv1.NudgeDelivery
	}{
		{"", chatv1.NudgeDelivery_NUDGE_DELIVERY_UNSPECIFIED},
		{"terminal", chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL},
		{"native", chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE},
		{"none", chatv1.NudgeDelivery_NUDGE_DELIVERY_NONE},
		{" none ", chatv1.NudgeDelivery_NUDGE_DELIVERY_NONE},
	} {
		t.Run(tc.value, func(t *testing.T) {
			got, err := ParseNudge(tc.value)
			assert.NilError(t, err)
			assert.Equal(t, got, tc.want)
		})
	}
	for _, value := range []string{"off", "None", "unspecified", "1"} {
		t.Run(value, func(t *testing.T) {
			_, err := ParseNudge(value)
			assert.ErrorContains(t, err, "want terminal, native or none")
		})
	}
}

// What a session declared and this server cannot honour is refused with the
// header named, since the header is what its launcher has to go and fix. The
// direct channel is among them: a shared server has no way to push one.
func TestParseDeclaration_NamesTheHeaderItRefuses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{"bad nudge", map[string]string{NudgeHeader: "off"}, NudgeHeader},
		{"bad channel", map[string]string{ChannelHeader: "fakechat"}, ChannelHeader},
		{"direct", map[string]string{ChannelHeader: "direct"}, ChannelHeader},
	} {
		t.Run(tc.name, func(t *testing.T) {
			header := http.Header{}
			for k, v := range tc.headers {
				header.Set(k, v)
			}
			_, _, err := parseDeclaration(header)
			assert.ErrorContains(t, err, tc.want)
		})
	}

	header := http.Header{}
	header.Set(NudgeHeader, "none")
	header.Set(ChannelHeader, "fakechat;port=8788")
	nudge, ch, err := parseDeclaration(header)
	assert.NilError(t, err)
	assert.Equal(t, nudge, chatv1.NudgeDelivery_NUDGE_DELIVERY_NONE)
	assert.Equal(
		t,
		ch,
		harnessctl.ClaudeChannel{Kind: harnessctl.ClaudeChannelFakechat, Port: "8788"},
	)
}

// A session that asked not to be woken attends saying so, which is what keeps
// the daemon from typing at it.
func TestHTTPServer_AttendsWithTheDeclaredNudge(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	connectDeclaring(t, serveHTTP(t, srv).endpoint, map[string]string{
		TokenHeader: testToken,
		NudgeHeader: "none",
	})

	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetNudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NONE)
}

// A session whose declaration does not parse acts as nobody: attending it with
// the declaration dropped would wake it in a way its launcher said not to.
func TestHTTPServer_ASessionWithABadDeclarationActsAsNobody(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	addMembersTool(srv)
	session := connectDeclaring(t, serveHTTP(t, srv).endpoint, map[string]string{
		TokenHeader: testToken,
		NudgeHeader: "off",
	})

	res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "chat_members"})
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	assert.Assert(t, strings.Contains(toolText(t, res), NudgeHeader),
		"the tool answered %q", toolText(t, res))
	assert.Equal(t, fake.openCount(), 0)
}

// Sessions sharing a token may declare different deliveries, and the member
// declares the one of the session it was last seen in: it attends again when
// that changes, and once more when the session it moved to closes.
func TestHTTPServer_FollowsTheNudgeOfTheSessionLastSeen(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	srv.host.pace.attendBackoffBase = 10 * time.Millisecond
	srv.host.pace.attendBackoffMax = 20 * time.Millisecond
	endpoint := serveHTTP(t, srv).endpoint

	connectDeclaring(t, endpoint, map[string]string{TokenHeader: testToken})
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetNudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)

	second := connectDeclaring(t, endpoint, map[string]string{
		TokenHeader: testToken,
		NudgeHeader: "none",
	})
	waitFor(t, "the member never declared the nudge of the session it was last seen in",
		func() bool {
			return fake.attendCount() == 2 &&
				fake.lastAttend().GetNudge() == chatv1.NudgeDelivery_NUDGE_DELIVERY_NONE
		})

	assert.NilError(t, second.Close())
	waitFor(t, "the member never declared the nudge of the session it is back in",
		func() bool {
			return fake.attendCount() == 3 &&
				fake.lastAttend().GetNudge() == chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL
		})
}

// Native delivery declared for a harness with no channel would stop the daemon
// typing at the member with nothing delivering in its place, so the member
// stays out of the room and its tools say why.
// A native attendance that opened on one session's channel is not kept when
// the member is seen in another session declaring native too: that session may
// carry no channel, and only a new opening checks it. Kept, the daemon would go
// on typing nothing at a member nothing delivers to.
func TestHTTPServer_ANativeSessionWithoutAChannelEndsTheNativeAttendance(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	srv.host.pace.attendBackoffBase = 10 * time.Millisecond
	srv.host.pace.attendBackoffMax = 20 * time.Millisecond
	var mu sync.Mutex
	harnesses := []harnessctl.Harness{
		newFeedHarness(chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE),
		newFeedHarness(chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL),
	}
	srv.host.detect = func(string, func(string) string, harnessctl.Options) harnessctl.Harness {
		mu.Lock()
		defer mu.Unlock()
		h := harnesses[0]
		harnesses = harnesses[1:]
		return h
	}
	endpoint := serveHTTP(t, srv).endpoint
	declares := map[string]string{TokenHeader: testToken, NudgeHeader: "native"}

	connectDeclaring(t, endpoint, declares)
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })
	assert.Equal(t, fake.lastAttend().GetNudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	connectDeclaring(t, endpoint, declares)
	waitFor(t, "the native attendance was kept for a session with no channel",
		func() bool { return fake.cancelCount() == 1 })
	// A few backoffs: the member keeps asking and keeps being refused before
	// the daemon is asked.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, fake.openCount(), 1)
}

// failingProber is a native harness whose channel never answers a probe.
type failingProber struct{ *feedHarness }

var errProbeRefused = errors.New("the plugin is not there")

func (failingProber) Probe(context.Context) error { return errProbeRefused }

// A member that keeps its native attendance across a move to another session
// is reprobed on the channel of the session it moved to, not on the one the
// attendance opened on, so a dead channel there ends the attendance.
func TestHTTPServer_ReprobesTheChannelOfTheSessionLastSeen(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	srv.host.pace.reprobeInterval = 10 * time.Millisecond
	var mu sync.Mutex
	harnesses := []harnessctl.Harness{
		newFeedHarness(chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE),
		failingProber{newFeedHarness(chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)},
	}
	srv.host.detect = func(string, func(string) string, harnessctl.Options) harnessctl.Harness {
		mu.Lock()
		defer mu.Unlock()
		h := harnesses[0]
		harnesses = harnesses[1:]
		return h
	}
	endpoint := serveHTTP(t, srv).endpoint

	connectDeclaring(t, endpoint, map[string]string{TokenHeader: testToken})
	waitFor(t, "the server never attended", func() bool { return fake.attendCount() == 1 })

	connectDeclaring(t, endpoint, map[string]string{TokenHeader: testToken})
	waitFor(t, "the dead channel of the session last seen never ended the attendance",
		func() bool { return fake.cancelCount() == 1 })
}

func TestHTTPServer_RefusesNativeWithoutAChannel(t *testing.T) {
	fake := &fakeChatService{self: doneSelf("backend", "alice", testRoom)}
	srv := newTestHTTP(t, fake)
	addMembersTool(srv)
	session := connectDeclaring(t, serveHTTP(t, srv).endpoint, map[string]string{
		TokenHeader: testToken,
		NudgeHeader: "native",
	})

	res, err := session.CallTool(t.Context(), &mcpsdk.CallToolParams{Name: "chat_members"})
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	assert.Equal(t, toolText(t, res), errNativeWithoutChannel.Error())
	assert.Equal(t, fake.openCount(), 0)
}
