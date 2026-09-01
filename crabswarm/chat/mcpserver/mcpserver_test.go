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

	return startSessionWith(t, svc, nil)
}

// startSessionWith is [startSession] for a harness that listens for what the
// bridge announces, rather than only calling it.
func startSessionWith(
	t *testing.T, svc *fakeChatService, opts *mcp.ClientOptions,
) *mcp.ClientSession {
	t.Helper()

	bridge, err := New(slog.New(slog.DiscardHandler), serveTestDaemon(t, svc), testToken)
	assert.NilError(t, err)

	serverSide, clientSide := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return bridge.serve(ctx, serverSide) })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-harness", Version: "v0"}, opts)
	session, err := client.Connect(t.Context(), clientSide, nil)
	assert.NilError(t, err)

	// Waiting for serve to return keeps the startup goroutine from outliving
	// the test that owns the stub it is calling.
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
			member("backend", "alice", testRoom),
			member("frontend", "bob", testRoom),
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
			want: "backend/alice\nfrontend/bob\n",
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
	assert.Equal(t, fake.lastJoin().GetAgent(), true)
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
	// The read was refused, not a join: the bridge still counted itself a
	// member when it went out, which is the state the refusal has to undo.
	assert.Equal(t, fake.joinCount(), 1)

	fake.setErr(nil)
	res, err = session.CallTool(t.Context(), read)
	assert.NilError(t, err)
	assert.Assert(t, !res.IsError, "tool failed: %s", textOf(t, res))
	assert.Equal(t, fake.joinCount(), 2)
}

func TestNew_RejectsEmptySocketPath(t *testing.T) {
	_, err := New(nil, "", testToken)
	assert.Assert(t, err != nil)
}
