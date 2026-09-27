package chat

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	crabmcp "github.com/ngicks/crabswarm/crabswarm/mcp"
)

// serveHTTPBridge runs a shared server with the chat family registered on a
// loopback port and returns the endpoint a harness server would be given. The
// environment it was launched in is empty, so no channel of the host running
// the suite is looked for.
func serveHTTPBridge(t *testing.T, svc *fakeChatService) string {
	t.Helper()

	server, err := crabmcp.NewHTTP(slog.New(slog.DiscardHandler), serveTestDaemon(t, svc),
		func(string) string { return "" })
	assert.NilError(t, err)
	Register(server)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return server.Serve(ctx, ln) })
	t.Cleanup(func() {
		cancel()
		_ = served.Wait()
	})
	return "http://" + ln.Addr().String() + crabmcp.HTTPPath
}

// stampsToken stamps every request of a session with the token of the agent it
// belongs to, the way a launcher configures its harness server per agent. The
// empty token stamps nothing.
type stampsToken struct {
	token string
}

func (s stampsToken) RoundTrip(req *http.Request) (*http.Response, error) {
	if s.token != "" {
		req = req.Clone(req.Context())
		req.Header.Set(crabmcp.TokenHeader, s.token)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// connectAs opens one session on endpoint as the agent token names.
func connectAs(t *testing.T, endpoint, token string) *mcp.ClientSession {
	t.Helper()

	transport := &mcp.StreamableClientTransport{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Transport: stampsToken{token: token}},
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "codex-mcp-client", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), transport, nil)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// Two agents behind one shared server are two members, and each one's message
// goes out as its own: the token its session named is the one every call it
// makes carries.
func TestHTTPServer_EachSessionSendsAsItsOwnMember(t *testing.T) {
	fake := &fakeChatService{
		selves: map[string]*chatv1.Member{
			"tok-a": member("backend", "alice", testRoom),
			"tok-b": member("backend", "bob", testRoom),
		},
	}
	endpoint := serveHTTPBridge(t, fake)
	alice := connectAs(t, endpoint, "tok-a")
	bob := connectAs(t, endpoint, "tok-b")

	for session, text := range map[*mcp.ClientSession]string{
		alice: "from alice", bob: "from bob",
	} {
		res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
			Name: "chat_send", Arguments: map[string]any{"to": "", "message": text},
		})
		assert.NilError(t, err)
		assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	}

	got := fake.sends()
	slices.SortFunc(got, func(a, b sent) int { return strings.Compare(a.token, b.token) })
	want := []sent{
		{token: "tok-a", text: "from alice"},
		{token: "tok-b", text: "from bob"},
	}
	assert.Assert(t, slices.Equal(got, want), "the daemon received %+v, want %+v", got, want)

	// One attendance per member, each under its own token.
	attended := fake.attendedTokens()
	slices.Sort(attended)
	assert.DeepEqual(t, attended, []string{"tok-a", "tok-b"})
	// The harness each member declares is the one its session named.
	assert.Equal(t, fake.lastAttend().GetHarness(), chatv1.Harness_HARNESS_CODEX)
}

// A session on a shared server that named no token acts as nobody. Its tools say
// which header was missing, and nothing is asked of the daemon on its behalf —
// not even under the identity the server process itself may have inherited.
func TestHTTPServer_ASessionWithoutATokenSaysWhatIsMissing(t *testing.T) {
	t.Setenv("CRABSWARM_CHAT_TOKEN", "tok-of-the-process")

	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	session := connectAs(t, serveHTTPBridge(t, fake), "")

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "chat_members"})
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	text := textOf(t, res)
	for _, want := range []string{"no chat identity token", crabmcp.TokenHeader} {
		assert.Assert(t, strings.Contains(text, want), "%q is missing from %q", want, text)
	}
	assert.Equal(t, fake.openCount(), 0)
}
