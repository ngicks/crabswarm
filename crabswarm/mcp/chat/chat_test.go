package chat

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
	crabmcp "github.com/ngicks/crabswarm/crabswarm/mcp"
)

// testToken is spelled out at every call site rather than left to resolution:
// this test binary may itself run under cmdman, and an empty token would
// quietly resolve to $CMDMAN_CMD_ID and talk as whoever is running the suite.
const testToken = "tok-a"

const testRoom = "/work/proj"

// startSession builds a server onto the stub and drives it over an in-memory
// pipe, returning the session a harness would hold. The transport is the only
// thing swapped out: the server runs its real startup, so every test here sees
// the attendance it holds.
func startSession(t *testing.T, svc *fakeChatService) *mcp.ClientSession {
	t.Helper()

	return serveBridge(t, newTestBridge(t, svc), nil)
}

// newTestBridge builds a server onto the stub with the chat family registered,
// which is the whole of what a harness gets.
func newTestBridge(t *testing.T, svc *fakeChatService) *crabmcp.Server {
	t.Helper()

	server, err := crabmcp.New(
		slog.New(slog.DiscardHandler), serveTestDaemon(t, svc), testToken)
	assert.NilError(t, err)
	Register(server)
	return server
}

// serveBridge runs server over an in-memory pipe and returns the session a
// harness would hold.
func serveBridge(
	t *testing.T, server *crabmcp.Server, opts *mcp.ClientOptions,
) *mcp.ClientSession {
	t.Helper()

	serverSide, clientSide := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return server.Serve(ctx, serverSide) })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-harness", Version: "v0"}, opts)
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
// down to the trailing newline. That is the promise the family makes: a member
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
			want: "backend/alice  agent  working  -  -\n" +
				"frontend/bob  human  done  -  -\n",
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

	// The server attends before a tool acts, under the name the daemon derives
	// from the token, and always as an agent: a harness is the only thing that
	// starts one, and a message reaching it should be typed into the terminal
	// it runs in.
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

// A daemon that refuses the token leaves the server running: the harness keeps
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

// A server whose harness handed it no identity at all — no --token, and an
// environment stripped of both variables one could arrive in — still serves.
// The alternative is a subprocess that exits before the handshake, which the
// harness reports as a closed connection and nobody can read a reason out of.
func TestServer_WithoutATokenServesToolsThatSayWhatIsMissing(t *testing.T) {
	// This test binary may itself run under cmdman, whose variable an empty
	// token would resolve through.
	t.Setenv("CRABSWARM_CHAT_TOKEN", "")
	t.Setenv("CMDMAN_CMD_ID", "")

	fake := &fakeChatService{self: member("backend", "alice", testRoom)}
	server, err := crabmcp.New(slog.New(slog.DiscardHandler), serveTestDaemon(t, fake), "")
	assert.NilError(t, err)
	Register(server)
	session := serveBridge(t, server, nil)

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
