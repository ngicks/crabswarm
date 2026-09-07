package mcpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// testToken is spelled out at every call site rather than left to resolution:
// this test binary may itself run under cmdman, and an empty token would
// quietly resolve to $CMDMAN_CMD_ID and talk as whoever is running the suite.
const testToken = "tok-a"

const testRoom = "/work/proj"

// startSession builds a bridge onto the stub and drives it over an in-memory
// pipe, returning the session a harness would hold. The transport is the only
// thing swapped out: the bridge runs its real startup, so every test here sees
// the attendance it declares.
func startSession(t *testing.T, svc *fakeChatService) *mcp.ClientSession {
	t.Helper()

	return serveBridge(t, newTestBridge(t, svc), nil)
}

// heldPace is a retry loop's wait for a case that is not about the loop: the
// first attempt happens as it does in production and the next is due long after
// the case has finished, so what the stub recorded is what the case itself
// asked for. The cases about a loop set their own pace.
const heldPace = time.Hour

// newTestBridge builds a bridge onto the stub with both its retry loops — the
// one that attends and the one that watches the room — held at [heldPace].
func newTestBridge(t *testing.T, svc *fakeChatService) *Server {
	t.Helper()

	bridge, err := New(slog.New(slog.DiscardHandler), serveTestDaemon(t, svc), testToken)
	assert.NilError(t, err)
	bridge.joinBackoffBase = heldPace
	bridge.joinBackoffMax = heldPace
	bridge.watchBackoffBase = heldPace
	bridge.watchBackoffMax = heldPace
	return bridge
}

// runsAt sets both loops going at a pace a case can wait for, for the cases
// that are about a loop retrying rather than about what one attempt did.
func runsAt(bridge *Server, base, max time.Duration) {
	bridge.joinBackoffBase = base
	bridge.joinBackoffMax = max
	bridge.watchBackoffBase = base
	bridge.watchBackoffMax = max
}

// serveBridge runs bridge over an in-memory pipe and returns the session a
// harness would hold.
func serveBridge(
	t *testing.T, bridge *Server, opts *mcp.ClientOptions,
) *mcp.ClientSession {
	t.Helper()

	serverSide, clientSide := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return bridge.serve(ctx, serverSide) })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-harness", Version: "v0"}, opts)
	session, err := client.Connect(t.Context(), clientSide, nil)
	assert.NilError(t, err)

	// Waiting for serve to return keeps the bridge's own goroutines from
	// outliving the test that owns the stub they are calling.
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		_ = served.Wait()
	})
	return session
}

// waitFor blocks until want reports true, failing with why when it never does.
// The wait is what a case spends on something the bridge does off its own
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

// textOf unwraps the one text block a chat tool answers with.
func textOf(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()

	assert.Equal(t, len(res.Content), 1)
	text, ok := res.Content[0].(*mcp.TextContent)
	assert.Assert(t, ok, "content is %T, not text", res.Content[0])
	return text.Text
}

// toolSchema is as much of a tool's input schema as these tests pin: what the
// model has to supply, and by which names.
type toolSchema struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

func schemaOf(t *testing.T, tool *mcp.Tool) toolSchema {
	t.Helper()

	raw, err := json.Marshal(tool.InputSchema)
	assert.NilError(t, err)
	var schema toolSchema
	assert.NilError(t, json.Unmarshal(raw, &schema))
	return schema
}

func propertyNames(schema toolSchema) []string {
	return slices.Sorted(maps.Keys(schema.Properties))
}

// The harness sees the four member verbs and nothing else. Reporting harness
// state is missing on purpose: it reaches the daemon through hooks, and a
// model asked for its own state would be reporting the one thing it cannot
// observe about itself.
func TestServer_ServesTheMemberVerbsAsTools(t *testing.T) {
	session := startSession(t, &fakeChatService{self: member("backend", "alice", testRoom)})

	res, err := session.ListTools(t.Context(), nil)
	assert.NilError(t, err)

	byName := map[string]*mcp.Tool{}
	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	assert.DeepEqual(t, names, []string{
		"chat_broadcast", "chat_members", "chat_read", "chat_send",
	})

	send := schemaOf(t, byName["chat_send"])
	assert.Equal(t, send.Type, "object")
	assert.DeepEqual(t, propertyNames(send), []string{"message", "to"})
	assert.DeepEqual(t, send.Required, []string{"to", "message"})

	broadcast := schemaOf(t, byName["chat_broadcast"])
	assert.DeepEqual(t, propertyNames(broadcast), []string{"message"})
	assert.DeepEqual(t, broadcast.Required, []string{"message"})

	for _, name := range []string{"chat_read", "chat_members"} {
		schema := schemaOf(t, byName[name])
		assert.Equal(t, schema.Type, "object")
		assert.Equal(t, len(schema.Properties), 0, "%s takes no arguments", name)
	}
}

// A tool answers with exactly what the matching `crabswarm chat` verb prints,
// down to the trailing newline. That is the promise the bridge makes: a member
// wired through MCP reads its room in the same words as one typing commands.
func TestServer_ToolsAnswerWithTheCLIWording(t *testing.T) {
	fake := &fakeChatService{
		self:      member("backend", "alice", testRoom),
		recipient: member("frontend", "bob", testRoom),
		delivered: 2,
		messages: []*chatv1.Message{{
			From:   member("frontend", "bob", testRoom),
			Text:   "rebased onto main",
			SentAt: timestamppb.New(time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)),
		}},
		members: []*chatv1.Member{
			memberOnRoster("backend", "alice", testRoom,
				chatv1.MemberKind_MEMBER_KIND_AGENT,
				chatv1.HarnessState_HARNESS_STATE_WORKING),
			memberOnRoster("frontend", "bob", testRoom,
				chatv1.MemberKind_MEMBER_KIND_HUMAN,
				chatv1.HarnessState_HARNESS_STATE_DONE),
		},
	}
	session := startSession(t, fake)

	for _, tc := range []struct {
		name string
		args map[string]any
		want string
	}{
		{
			name: "chat_send",
			args: map[string]any{"to": "bob", "message": "PR is ready"},
			want: "sent to frontend/bob\n",
		},
		{
			name: "chat_broadcast",
			args: map[string]any{"message": "starting the release"},
			want: "broadcast to 2 members\n",
		},
		{
			name: "chat_read",
			want: "[2026-08-31T12:00:00Z] frontend/bob: rebased onto main\n",
		},
		{
			name: "chat_members",
			want: "backend/alice  agent  working\nfrontend/bob  human  done\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
				Name: tc.name, Arguments: tc.args,
			})
			assert.NilError(t, err)
			assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
			assert.Equal(t, textOf(t, res), tc.want)
		})
	}

	// The bridge attends before it acts, and forwards the arguments untouched:
	// resolving an address is the daemon's job, not the bridge's.
	assert.Assert(t, fake.lastJoin() != nil)
	assert.Equal(t, fake.lastJoin().GetName(), "")
	// As an agent: a harness is the only thing that starts a bridge, and a
	// message reaching it should be typed into the terminal it runs in.
	assert.Equal(t, fake.lastJoin().GetKind(),
		chatv1.MemberKind_MEMBER_KIND_AGENT)
	assert.Equal(t, fake.lastSend().GetTo(), "bob")
	assert.Equal(t, fake.lastSend().GetText(), "PR is ready")
	assert.Equal(t, fake.lastBroadcast().GetText(), "starting the release")
	assert.Equal(t, fake.readCount(), 1)
}

// A daemon that refuses the token leaves the bridge running: the harness keeps
// a live MCP server that says why every call fails, rather than a subprocess
// that died during startup with nothing to read about it.
func TestServer_JoinFailureDegradesToErroringTools(t *testing.T) {
	const refusal = "unknown identity token"
	fake := &fakeChatService{err: status.Error(codes.Unauthenticated, refusal)}
	session := startSession(t, fake)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "chat_send", Arguments: map[string]any{"to": "bob", "message": "PR is ready"},
	})
	// A tool error, not a transport one: the model is meant to read it.
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	assert.Assert(t, strings.Contains(textOf(t, res), refusal), "got %q", textOf(t, res))

	// The refusal stopped the call at attendance: sending on behalf of a
	// member the daemon does not acknowledge would be asking for a second no.
	assert.Assert(t, fake.lastSend() == nil)

	// The session is still there to be asked again.
	tools, err := session.ListTools(t.Context(), nil)
	assert.NilError(t, err)
	assert.Equal(t, len(tools.Tools), 4)
}

// Attendance is declared once and remembered. A join per tool call would spend
// a round trip re-asking a question the daemon has already answered, on every
// message an agent sends for the rest of its session.
func TestServer_AttendsOnceForTheWholeSession(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	session := startSession(t, fake)

	for range 2 {
		res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "chat_read"})
		assert.NilError(t, err)
		assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	}

	assert.Equal(t, fake.readCount(), 2)
	assert.Equal(t, fake.joinCount(), 1)
}

// A daemon can stop counting this bridge as a member while the session is still
// running — it reaps a command the team-info provider stopped knowing, a human
// types `crabswarm chat leave`, a restarted daemon comes back on a fresh
// database. The refusal that follows is what tells the bridge, so the call after
// it attends again and works, rather than every call failing that way until the
// harness is restarted.
func TestServer_AttendsAgainAfterTheDaemonForgetsTheMember(t *testing.T) {
	// The daemon's own wording for a caller it cannot resolve to a member.
	const refusal = "token is not attending any room; join first"
	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	session := startSession(t, fake)

	read := &mcp.CallToolParams{Name: "chat_read"}
	res, err := session.CallTool(t.Context(), read)
	assert.NilError(t, err)
	assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	assert.Equal(t, fake.joinCount(), 1)

	fake.setErr(status.Error(codes.Unauthenticated, refusal))
	res, err = session.CallTool(t.Context(), read)
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	assert.Assert(t, strings.Contains(textOf(t, res), refusal), "got %q", textOf(t, res))
	// The read went out and came back refused: the bridge still counted itself
	// a member, which is the state the refusal has to undo.
	assert.Equal(t, fake.readCount(), 2)

	fake.setErr(nil)
	res, err = session.CallTool(t.Context(), read)
	assert.NilError(t, err)
	assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	// Twice in all: the join this session opened with, and the one the refusal
	// sent it back for. Both loops are held, so nothing else asked.
	assert.Equal(t, fake.joinCount(), 2, "the bridge never attended again")
}

// A bridge whose daemon refuses it keeps asking, past any handful of attempts.
// A harness starts its MCP subprocesses before the services they talk to, so
// the daemon regularly arrives late; a bridge that stopped asking would leave
// the room a member short until the agent happened to call a tool.
func TestServer_KeepsAttendingUntilTheDaemonAdmitsIt(t *testing.T) {
	fake := &fakeChatService{
		self: member("backend", "alice", testRoom),
		err:  status.Error(codes.Unauthenticated, "unknown identity token"),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge, nil)

	// More attempts than a bounded retry would have made. Both the attend loop
	// and the room's feed declare attendance, so this counts what the bridge
	// asked for rather than which goroutine asked.
	waitFor(t, "the bridge stopped asking to attend", func() bool {
		return fake.joinCount() > 5
	})

	fake.setErr(nil)
	waitFor(t, "the bridge never attended once the daemon admitted it", func() bool {
		return fake.attendCount() == 1
	})
}

// A daemon that forgets this member mid-session — it came back on a fresh
// database, or reaped a command its provider stopped knowing — is answered by
// attending again, with nothing asking the bridge to. That matters because the
// agent is not the one who noticed: its hooks report state through the CLI, and
// they fail as Unauthenticated for as long as the room is missing the member.
func TestServer_AttendsAgainWithoutBeingAsked(t *testing.T) {
	fake := &fakeChatService{
		self:   member("backend", "alice", testRoom),
		events: make(chan *chatv1.RoomEvent),
		drops:  make(chan error),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge, nil)

	waitFor(t, "the bridge never attended", func() bool {
		return fake.attendCount() == 1
	})

	// The feed is what carries the news: the daemon refuses the watcher as a
	// caller it does not count as a member.
	dropFeed(t, fake, status.Error(codes.Unauthenticated,
		"token is not attending any room; join first"))

	waitFor(t, "the bridge never attended again", func() bool {
		return fake.attendCount() == 2
	})
	assert.Equal(t, fake.readCount(), 0, "no tool call was made")
}

// The daemon publishes a member's departure into the feed that member is
// reading, and it authorises a feed once and never again — so a bridge whose
// membership was taken away is left holding a stream that works and a
// membership that does not. The departure names it, and that is enough: it
// attends again without anything asking, which is what the agent's hooks need,
// since they report harness state through the CLI and are refused for as long
// as the room is missing the member.
func TestServer_AttendsAgainWhenTheRoomSaysItLeft(t *testing.T) {
	self := member("backend", "alice", testRoom)
	fake := &fakeChatService{
		self:   self,
		events: make(chan *chatv1.RoomEvent),
		drops:  make(chan error),
	}
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge, nil)

	waitFor(t, "the bridge never attended", func() bool {
		return fake.attendCount() == 1
	})

	pushEvent(t, fake, leftEvent(self))

	waitFor(t, "the bridge never attended again", func() bool {
		return fake.attendCount() == 2
	})
	assert.Equal(t, fake.readCount(), 0, "no tool call was made")
}

// Someone else leaving is the ordinary traffic of a room, and the bridge stays
// exactly as it was: its own membership is untouched, so re-declaring it would
// spend a round trip on every departure the room ever sees.
func TestServer_StaysAttendingWhenAnotherMemberLeaves(t *testing.T) {
	fake := &fakeChatService{
		self:   member("backend", "alice", testRoom),
		events: make(chan *chatv1.RoomEvent),
		drops:  make(chan error),
	}
	// Both loops run quickly, so a departure taken for this bridge's own would
	// have been acted on well inside the settle below rather than waiting out a
	// backoff the case would never see the end of.
	bridge := newTestBridge(t, fake)
	runsAt(bridge, 10*time.Millisecond, 20*time.Millisecond)
	serveBridge(t, bridge, nil)

	waitFor(t, "the bridge never attended", func() bool {
		return fake.attendCount() == 1
	})

	// A teammate, and a member of another team carrying this bridge's own name:
	// a departure is matched on both halves of an identity, not on either alone.
	pushEvent(t, fake, leftEvent(member("backend", "bob", testRoom)))
	pushEvent(t, fake, leftEvent(member("frontend", "alice", testRoom)))

	// The feed the bridge is reading is still the one it opened, and the
	// membership behind it is still the one it declared.
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, fake.joinCount(), 1, "the bridge attended again")
	assert.Equal(t, fake.watchCount(), 1, "the bridge watched the room again")
}

// A bridge whose harness handed it no identity at all — no --token, and an
// environment stripped of both variables one could arrive in — still serves.
// The alternative is a subprocess that exits before the handshake, which the
// harness reports as a closed connection and nobody can read a reason out of.
func TestServer_WithoutATokenServesToolsThatSayWhatIsMissing(t *testing.T) {
	// This test binary may itself run under cmdman, whose variable an empty
	// token would resolve through.
	t.Setenv("CRABSWARM_CHAT_TOKEN", "")
	t.Setenv("CMDMAN_CMD_ID", "")

	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	bridge, err := New(slog.New(slog.DiscardHandler), serveTestDaemon(t, fake), "")
	assert.NilError(t, err)
	session := serveBridge(t, bridge, nil)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "chat_members"})
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	// The words every member verb answers a missing identity with, naming what
	// to pass and what to set.
	text := textOf(t, res)
	for _, want := range []string{
		"no chat identity token", "--token", "CRABSWARM_CHAT_TOKEN", "CMDMAN_CMD_ID",
	} {
		assert.Assert(t, strings.Contains(text, want), "%q is missing from %q", want, text)
	}

	// Nothing was asked of the daemon on behalf of nobody.
	assert.Equal(t, fake.joinCount(), 0)
}

func TestNew_RejectsEmptySocketPath(t *testing.T) {
	_, err := New(nil, "", testToken)
	assert.Assert(t, err != nil)
}

// Both retry loops come out of New with a pace of their own. The fields exist so
// a test can slow a loop down or hold it still, and a bridge nobody tuned would
// otherwise run them at a zero wait — asking the daemon as fast as it can answer
// for the whole session.
func TestNew_SeedsBothRetryPaces(t *testing.T) {
	bridge, err := New(nil, serveTestDaemon(t, &fakeChatService{}), testToken)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = bridge.client.Close() })

	for _, tc := range []struct {
		what      string
		base, max time.Duration
	}{
		{what: "attending", base: bridge.joinBackoffBase, max: bridge.joinBackoffMax},
		{what: "watching", base: bridge.watchBackoffBase, max: bridge.watchBackoffMax},
	} {
		assert.Assert(t, tc.base > 0, "%s starts at %s", tc.what, tc.base)
		assert.Assert(t, tc.max >= tc.base,
			"%s climbs to %s, below its first wait of %s", tc.what, tc.max, tc.base)
	}
}
