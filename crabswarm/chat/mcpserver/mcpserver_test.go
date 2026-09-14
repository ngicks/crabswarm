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
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// testToken is spelled out at every call site rather than left to resolution:
// this test binary may itself run under cmdman, and an empty token would
// quietly resolve to $CMDMAN_CMD_ID and talk as whoever is running the suite.
const testToken = "tok-a"

const testRoom = "/work/proj"

// startSession builds a bridge onto the stub and drives it over an in-memory
// pipe, returning the session a harness would hold. The transport is the only
// thing swapped out: the bridge runs its real startup, so every test here sees
// the attendance it holds.
func startSession(t *testing.T, svc *fakeChatService) *mcp.ClientSession {
	t.Helper()

	return serveBridge(t, newTestBridge(t, svc), nil)
}

// heldPace is the retry loop's wait for a case that is not about the loop: the
// first attempt happens as it does in production and the next is due long after
// the case has finished, so what the stub recorded is what the case itself
// asked for. The cases about the loop set their own pace.
const heldPace = time.Hour

// newTestBridge builds a bridge onto the stub with its retry loop held at
// [heldPace].
func newTestBridge(t *testing.T, svc *fakeChatService) *Server {
	t.Helper()

	bridge, err := New(slog.New(slog.DiscardHandler), serveTestDaemon(t, svc), testToken)
	assert.NilError(t, err)
	runsAt(bridge, heldPace, heldPace)
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
// model may supply, and which of it it must.
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

// The harness sees the three member verbs and nothing else. Reporting harness
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
	assert.DeepEqual(t, names, []string{"chat_members", "chat_read", "chat_send"})

	// Both halves of a send are required: who a message is for is the decision
	// the sender is making, and a board post is the empty target written out
	// rather than the target left out.
	send := schemaOf(t, byName["chat_send"])
	assert.Equal(t, send.Type, "object")
	assert.DeepEqual(t, propertyNames(send), []string{"message", "to"})
	assert.DeepEqual(t, send.Required, []string{"to", "message"})

	// A read is all defaults: every argument left out is the daemon's own, and
	// an argument the model must supply to read its unread would be one more
	// thing it can get wrong.
	read := schemaOf(t, byName["chat_read"])
	assert.Equal(t, read.Type, "object")
	assert.DeepEqual(t, propertyNames(read),
		[]string{"cursor", "range", "since", "to", "until"})
	assert.Equal(t, len(read.Required), 0, "a read requires %v", read.Required)

	members := schemaOf(t, byName["chat_members"])
	assert.Equal(t, members.Type, "object")
	assert.Equal(t, len(members.Properties), 0, "chat_members takes arguments")
}

// A tool answers with exactly what the matching `crabswarm chat` verb prints,
// down to the trailing newline. That is the promise the bridge makes: a member
// wired through MCP reads its room in the same words as one typing commands.
func TestServer_ToolsAnswerWithTheCLIWording(t *testing.T) {
	fake := &fakeChatService{
		self:      member("backend", "alice", testRoom),
		mentioned: []*chatv1.Member{member("frontend", "bob", testRoom)},
		absent:    []*chatv1.Member{member("ops", "carol", testRoom)},
		messages: []*chatv1.Message{{
			Seq:          41,
			From:         member("frontend", "bob", testRoom),
			Target:       rolesTarget(member("backend", "alice", testRoom)),
			MentionedYou: true,
			Text:         "rebased onto main",
			SentAt:       timestamppb.New(time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)),
		}},
		remaining: 2,
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
			// The warning rides in the tool's own text: a mention nobody is
			// attending under is what its reader has to act on, and a tool
			// result has no second stream to put it on.
			want: "mentioned frontend/bob\n" +
				"warning: ops/carol is not attending; the mention waits\n",
		},
		{
			name: "chat_read",
			want: "41 2026-08-31T12:00:00Z frontend/bob -> backend/alice " +
				"[mentioned you]: rebased onto main\n2 more unread\n",
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

	// The bridge attends before it acts, under the name the daemon derives
	// from the token, and always as an agent: a harness is the only thing that
	// starts a bridge, and a message reaching it should be typed into the
	// terminal it runs in.
	assert.Assert(t, fake.lastAttend() != nil)
	assert.Equal(t, fake.lastAttend().GetName(), "")
	assert.Equal(t, fake.lastAttend().GetKind(), chatv1.MemberKind_MEMBER_KIND_AGENT)

	// The bare name goes out as a role for the daemon to resolve, and a read
	// with no arguments asks for no cursor at all, which is how it takes the
	// unread default.
	assert.Equal(t, fake.lastSend().GetText(), "PR is ready")
	assert.Equal(t, cli.TargetString(fake.lastSend().GetTarget()), "bob")
	assert.Equal(t, fake.lastRead().GetFilter().GetCursor(),
		chatv1.ReadCursor_READ_CURSOR_UNSPECIFIED)
	assert.Equal(t, fake.readCount(), 1)
}

// An empty target is a board post: the message is in the room and mentions
// nobody. The tool passes it on as the absent target the wire spells one with,
// rather than as a role nobody holds.
func TestServer_SendsABoardPostForAnEmptyTarget(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	session := startSession(t, fake)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "chat_send",
		Arguments: map[string]any{"to": "", "message": "fyi: rebased main"},
	})
	assert.NilError(t, err)
	assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	// A post names no role, so there is nothing to report about delivery.
	assert.Equal(t, textOf(t, res), "")

	assert.Equal(t, fake.lastSend().GetText(), "fyi: rebased main")
	assert.Assert(t, fake.lastSend().GetTarget() == nil, "the post carried a target")
}

// A read carries the arguments it was given, so the agent can ask for a stretch
// of the room the way a person types the same flags.
func TestServer_ReadsFromTheCursorItIsGiven(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	session := startSession(t, fake)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name: "chat_read",
		Arguments: map[string]any{
			"cursor": "tail", "range": -5, "to": "everyone", "since": 10, "until": 90,
		},
	})
	assert.NilError(t, err)
	assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	assert.Equal(t, textOf(t, res), "no pending messages\n")

	filter := fake.lastRead().GetFilter()
	assert.Equal(t, filter.GetCursor(), chatv1.ReadCursor_READ_CURSOR_TAIL)
	assert.Equal(t, filter.GetRange(), int32(-5))
	assert.Equal(t, filter.GetSince(), int64(10))
	assert.Equal(t, filter.GetUntil(), int64(90))
	assert.Assert(t, filter.GetTo().GetEveryone() != nil, "the read narrowed to %v", filter.GetTo())
}

// A cursor word the daemon does not know is refused before anything is asked of
// it: the refusal names the words that work, which is what the model needs, and
// a round trip spent on a typo would come back saying less.
func TestServer_RefusesAnUnknownCursorWithoutReading(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	session := startSession(t, fake)

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "chat_read",
		Arguments: map[string]any{"cursor": "everything"},
	})
	assert.NilError(t, err)
	assert.Assert(t, res.IsError)
	text := textOf(t, res)
	for _, want := range []string{"everything", "unread", "head", "tail"} {
		assert.Assert(t, strings.Contains(text, want), "%q is missing from %q", want, text)
	}
	assert.Equal(t, fake.readCount(), 0)
}

// A daemon that refuses the token leaves the bridge running: the harness keeps
// a live MCP server that says why every call fails, rather than a subprocess
// that died during startup with nothing to read about it.
func TestServer_RefusedAttendanceDegradesToErroringTools(t *testing.T) {
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
	// member the room does not have would be asking for a second no.
	assert.Assert(t, fake.lastSend() == nil)

	// The session is still there to be asked again.
	tools, err := session.ListTools(t.Context(), nil)
	assert.NilError(t, err)
	assert.Equal(t, len(tools.Tools), 3)
}

// Attendance is one stream held for the whole session, so a tool call declares
// nothing. Re-declaring it per call would spend a round trip on a question the
// daemon has already answered, on every message an agent sends.
func TestServer_AttendsOnceForTheWholeSession(t *testing.T) {
	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	session := startSession(t, fake)

	for range 2 {
		res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "chat_read"})
		assert.NilError(t, err)
		assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	}

	assert.Equal(t, fake.readCount(), 2)
	assert.Equal(t, fake.openCount(), 1)
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

	// More attempts than a bounded retry would have made.
	waitFor(t, "the bridge stopped asking to attend", func() bool {
		return fake.openCount() > 5
	})

	fake.setErr(nil)
	waitFor(t, "the bridge never attended once the daemon admitted it", func() bool {
		return fake.attendCount() == 1
	})
}

// The attendance is the stream, so a daemon that closes it has taken the
// membership with it — a restart, a reaped command, a watcher dropped for
// falling behind. The bridge opens it again with nothing asking it to, which is
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
	serveBridge(t, bridge, nil)

	waitFor(t, "the bridge never attended", func() bool {
		return fake.attendCount() == 1
	})

	dropFeed(t, fake, status.Error(codes.Unavailable, "the daemon is going away"))

	waitFor(t, "the bridge never attended again", func() bool {
		return fake.attendCount() == 2
	})
	assert.Equal(t, fake.readCount(), 0, "no tool call was made")
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
	assert.Equal(t, fake.openCount(), 0)
}

func TestNew_RejectsEmptySocketPath(t *testing.T) {
	_, err := New(nil, "", testToken)
	assert.Assert(t, err != nil)
}

// The retry loop comes out of New with a pace of its own. The fields exist so a
// test can slow it down or hold it still, and a bridge nobody tuned would
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
