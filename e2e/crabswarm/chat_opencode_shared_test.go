package crabswarm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Agent replicas that share one `opencode serve`, end to end: the daemon, the
// `crabswarm mcp --transport http` beside the server, the real server with its
// plugin and a mock model, and one real `opencode attach` per replica, each with
// the TUI plugin and a token of its own. chat_opencode_test.go holds the
// helpers and the story of one TUI.

// openCodeSharedNotice is how the notice of a mention from bob reads.
const openCodeSharedNotice = "new message from alpha/bob"

// openCodeSharedTimeout bounds every wait here past the TUI's first
// attendance.
const openCodeSharedTimeout = 60 * time.Second

// openCodeShared is what the cases share: one daemon, one person in the room,
// one MCP server and one OpenCode server in front of the mock model.
type openCodeShared struct {
	opencode string
	cfg      string
	// bob is the token of the person writing to the replicas.
	bob     string
	mcpAddr string
	mcp     *exec.Cmd
	server  *opencodeServer
	mock    *mockProvider
}

// openCodeReplica is one TUI attached to the shared server.
type openCodeReplica struct {
	token   string
	address string
	// session is the session the TUI shows, whose first prompt names address.
	session string
	tui     *exec.Cmd
}

// startOpenCodeShared starts everything but the TUIs, with answer as the model,
// and returns once the OpenCode server is connected to the MCP server. It skips
// the case when opencode is not on PATH.
//
// The stub cmdman vouches for two replicas of the compose command opencode and
// for the server, which runs as tok-srv.
func startOpenCodeShared(
	t *testing.T, answer func(ctx context.Context, req mockRequest) mockReply,
) *openCodeShared {
	t.Helper()
	opencode, err := exec.LookPath("opencode")
	if err != nil {
		t.Skip("opencode is not on PATH")
	}

	identity, recipient := newChatIdentityFile(t)
	cfg := writeChatConfig(t, 0, []stubCommand{
		{token: "tok-oc1", dir: chatRoom, project: "alpha", command: "opencode", scaleIndex: "1"},
		{token: "tok-oc2", dir: chatRoom, project: "alpha", command: "opencode", scaleIndex: "2"},
		{token: "tok-srv", dir: chatRoom, project: "alpha", command: "opencode-serve"},
	}, recipient)
	startChatServe(t, cfg)
	c := &openCodeShared{
		opencode: opencode,
		cfg:      cfg,
		bob:      registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob"),
		mcpAddr:  freeAddr(t),
		mock:     newMockProvider(t, answer),
	}
	c.mcp = startOpenCodeMCPOn(t, cfg, c.mcpAddr)
	c.server = startOpenCodeServe(t, opencode, cfg, c.mcpAddr, c.mock)
	waitFor(t, openCodeSharedTimeout, "opencode connecting to crabswarm mcp", func() bool {
		_, answer := c.server.try(t, http.MethodGet, "/mcp", "", 10*time.Second)
		return strings.Contains(answer, `"connected"`)
	})
	return c
}

// attach creates a session whose first turn tells the model it is address,
// attaches a TUI running as token to it, and returns once the TUI attends as a
// native OpenCode member and has registered the session.
//
// The first turn is taken before the TUI attaches, so the TUI's plugin reports
// nothing for it.
func (c *openCodeShared) attach(t *testing.T, token, address string) openCodeReplica {
	t.Helper()
	session := c.server.createSession(t)
	c.server.takeTurn(t, session, "you are "+address)
	r := openCodeReplica{
		token:   token,
		address: address,
		session: session,
		tui:     startOpenCodeTUI(t, c.opencode, c.server, c.cfg, token, session),
	}
	c.waitNative(t, address)
	waitFor(t, openCodeSharedTimeout, address+"'s TUI registering the session it shows",
		func() bool {
			return openCodeRead(t, c.mcpAddr, session) == http.StatusNoContent
		})
	return r
}

// waitNative blocks until address attends as a native OpenCode member.
func (c *openCodeShared) waitNative(t *testing.T, address string) {
	t.Helper()
	waitFor(t, 120*time.Second, address+" attending as a native OpenCode member", func() bool {
		row := c.row(t, address)
		return len(row) == 5 && row[3] == "opencode" && row[4] == "native"
	})
}

// row is the fields of address's line in bob's `chat members`: address, kind,
// state, harness and nudge. It is nil while address does not attend.
func (c *openCodeShared) row(t *testing.T, address string) []string {
	t.Helper()
	for _, line := range lines(runChat(t, c.cfg, c.bob, "members")) {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == address {
			return fields
		}
	}
	return nil
}

// mention has bob mention the replica named name.
func (c *openCodeShared) mention(t *testing.T, name, text string) {
	t.Helper()
	runChat(t, c.cfg, c.bob, "send", name, text)
}

// waitBobReads blocks until bob has read want, reading as often as it takes.
func (c *openCodeShared) waitBobReads(t *testing.T, want string) {
	t.Helper()
	var read strings.Builder
	waitFor(t, openCodeSharedTimeout, "bob reading "+want, func() bool {
		read.WriteString(runChat(t, c.cfg, c.bob, "read"))
		return strings.Contains(read.String(), want)
	})
}

// states returns the states the daemon published for token's member through the
// stub cmdman, oldest first.
func (c *openCodeShared) states(t *testing.T, token string) []string {
	t.Helper()
	var states []string
	for _, line := range stubStatus(t, c.cfg) {
		// set <state> <token> --detail ...
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "set" && fields[2] == token {
			states = append(states, fields[1])
		}
	}
	return states
}

// waitTurn blocks until token's member, which had published since states, has
// been published working and then done.
func (c *openCodeShared) waitTurn(t *testing.T, token string, since int) {
	t.Helper()
	waitFor(t, openCodeSharedTimeout, token+"'s turn being reported working and then done",
		func() bool {
			states := c.states(t, token)
			if len(states) <= since {
				return false
			}
			states = states[since:]
			return slices.Contains(states, "working") && states[len(states)-1] == "done"
		})
}

// notices counts the prompts of session that carry the notice of a mention.
func (c *openCodeShared) notices(t *testing.T, session string) int {
	t.Helper()
	n := 0
	for _, prompt := range c.server.prompts(t, session) {
		if strings.Contains(prompt, openCodeSharedNotice) {
			n++
		}
	}
	return n
}

// prompts returns the text of every part of every user message in session,
// oldest first.
func (s *opencodeServer) prompts(t *testing.T, session string) []string {
	t.Helper()
	var messages []struct {
		Info struct {
			Role string `json:"role"`
		} `json:"info"`
		Parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	answer := s.call(t, http.MethodGet, "/session/"+session+"/message", "")
	if err := json.Unmarshal([]byte(answer), &messages); err != nil {
		t.Fatalf("decode the messages of %s: %v\n%s", session, err, answer)
	}
	var prompts []string
	for _, msg := range messages {
		if msg.Info.Role != "user" {
			continue
		}
		for _, part := range msg.Parts {
			if part.Type == "text" {
				prompts = append(prompts, part.Text)
			}
		}
	}
	return prompts
}

// promptAsync sends one user message to session and returns once the server
// has queued it, before the model has answered.
func (s *opencodeServer) promptAsync(t *testing.T, session, prompt string) {
	t.Helper()
	message, err := json.Marshal(map[string]any{
		"model": map[string]string{"providerID": "mock", "modelID": "mock"},
		"parts": []map[string]string{{"type": "text", "text": prompt}},
	})
	if err != nil {
		t.Fatal(err)
	}
	status, answer := s.try(t, http.MethodPost, "/session/"+session+"/prompt_async",
		string(message), 30*time.Second)
	if status != http.StatusNoContent && status != http.StatusOK {
		t.Fatalf("POST /session/%s/prompt_async: status %d\n%s", session, status, answer)
	}
}

// waitIdle blocks until the server lists session as running no turn: absent
// from its session statuses, or idle there.
func (s *opencodeServer) waitIdle(t *testing.T, session string) {
	t.Helper()
	waitFor(t, openCodeSharedTimeout, session+" ending its turn", func() bool {
		var statuses map[string]struct {
			Type string `json:"type"`
		}
		status, answer := s.try(t, http.MethodGet, "/session/status", "", 10*time.Second)
		if status != http.StatusOK || json.Unmarshal([]byte(answer), &statuses) != nil {
			return false
		}
		st, ok := statuses[session]
		return !ok || st.Type == "idle"
	})
}

// answerAsReplica answers the notice of a mention with a chat_send to bob that
// names the replica whose session the request came from, which the session's
// first prompt names, and anything else with "ok". The message sent is
// "<address> answers".
func answerAsReplica(req mockRequest) mockReply {
	role, last := req.last()
	if role != "user" || !strings.Contains(last, openCodeSharedNotice) ||
		!req.offers(opencodeSendTool) {
		return mockReply{}
	}
	for _, msg := range req.Messages {
		for _, address := range []string{"alpha/opencode-1", "alpha/opencode-2"} {
			if strings.Contains(string(msg), "you are "+address) {
				return mockReply{
					tool: opencodeSendTool,
					args: fmt.Sprintf(`{"to":"bob","message":"%s answers"}`, address),
				}
			}
		}
	}
	return mockReply{}
}

// sentBy is the line bob reads for the chat_send answerAsReplica makes as
// address.
func sentBy(address string) string {
	return address + " -> alpha/bob [mentioned you]: " + address + " answers"
}

// Two TUIs attached to one server are two members. Each attends as a native
// OpenCode member under its own token and registers the session it shows. A
// turn in one session is reported for its own member and leaves the other's
// state alone. A mention of one is prompted into its session and no other, and
// the chat_send the model answers it with there goes out as that replica, with
// that replica's unread in its result.
func TestChatOpenCodeShared_EachReplicaIsAMemberOfItsOwn(t *testing.T) {
	// The model holds its answer to a prompt asking it to take its time until
	// the case lets it go.
	holding := make(chan struct{})
	hold := make(chan struct{})
	release := sync.OnceFunc(func() { close(hold) })
	var held sync.Once
	c := startOpenCodeShared(t, func(ctx context.Context, req mockRequest) mockReply {
		if role, last := req.last(); role == "user" && strings.Contains(last, "take your time") &&
			req.offers(opencodeSendTool) {
			held.Do(func() {
				close(holding)
				select {
				case <-hold:
				case <-ctx.Done():
				}
			})
		}
		return answerAsReplica(req)
	})
	t.Cleanup(release)

	one := c.attach(t, "tok-oc1", "alpha/opencode-1")
	two := c.attach(t, "tok-oc2", "alpha/opencode-2")

	// A turn in one's session, held mid-way.
	since1, before2 := len(c.states(t, one.token)), c.states(t, two.token)
	c.server.promptAsync(t, one.session, "take your time")
	select {
	case <-holding:
	case <-time.After(openCodeSharedTimeout):
		t.Fatal("the model never received the prompt of one's turn")
	}
	waitFor(t, openCodeSharedTimeout, one.address+" being listed working", func() bool {
		row := c.row(t, one.address)
		return len(row) == 5 && row[2] == "working"
	})
	if row := c.row(t, two.address); len(row) != 5 || row[2] != "done" {
		t.Errorf("while %s works, %s is listed %q, want done", one.address, two.address, row)
	}
	release()
	c.waitTurn(t, one.token, since1)
	if got := c.states(t, two.token); !slices.Equal(got, before2) {
		t.Errorf("%s's turn published %q for %s, which had %q before it",
			one.address, got, two.token, before2)
	}

	// A turn in two's session.
	since2, before1 := len(c.states(t, two.token)), c.states(t, one.token)
	c.server.takeTurn(t, two.session, "carry on")
	c.waitTurn(t, two.token, since2)
	if got := c.states(t, one.token); !slices.Equal(got, before1) {
		t.Errorf("%s's turn published %q for %s, which had %q before it",
			two.address, got, one.token, before1)
	}

	// A mention of one.
	since1 = len(c.states(t, one.token))
	c.mention(t, "opencode-1", "ping one")
	c.waitBobReads(t, sentBy(one.address))
	c.waitTurn(t, one.token, since1)
	if n := c.notices(t, one.session); n != 1 {
		t.Errorf("%s's session holds %d prompts of a notice, want 1", one.address, n)
	}
	if n := c.notices(t, two.session); n != 0 {
		t.Errorf("%s's session holds %d prompts of a notice for %s, want 0",
			two.address, n, one.address)
	}
	if !c.mock.sawRequest(chatDeliveryNotices[0], "ping one") {
		t.Errorf("no chat_send result carried the mention of %s read mid-turn", one.address)
	}

	// A mention of two.
	since2 = len(c.states(t, two.token))
	c.mention(t, "opencode-2", "ping two")
	c.waitBobReads(t, sentBy(two.address))
	c.waitTurn(t, two.token, since2)
	if n := c.notices(t, two.session); n != 1 {
		t.Errorf("%s's session holds %d prompts of a notice, want 1", two.address, n)
	}
	if n := c.notices(t, one.session); n != 1 {
		t.Errorf("%s's session holds %d prompts of a notice, want only its own", one.address, n)
	}
	if !c.mock.sawRequest(chatDeliveryNotices[0], "ping two") {
		t.Errorf("no chat_send result carried the mention of %s read mid-turn", two.address)
	}

	for _, r := range []openCodeReplica{one, two} {
		if got := runChat(t, c.cfg, r.token, "read"); got != "no pending messages\n" {
			t.Errorf("%s's read after its turn = %q, want its mention handed over", r.address, got)
		}
	}
	if keys := stubSendKeys(t, c.cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}
}

// A replica whose TUI dies takes only its own member out of the room, and the
// session it showed stops acting as anybody. The other replica stays, and a
// mention still reaches it in its own session.
func TestChatOpenCodeShared_AStoppedReplicaLeavesAlone(t *testing.T) {
	c := startOpenCodeShared(t, func(_ context.Context, req mockRequest) mockReply {
		return answerAsReplica(req)
	})
	one := c.attach(t, "tok-oc1", "alpha/opencode-1")
	two := c.attach(t, "tok-oc2", "alpha/opencode-2")

	// Killed rather than asked to quit, so nothing but the connection closing
	// tells the MCP server.
	if err := two.tui.Process.Kill(); err != nil {
		t.Fatalf("kill %s's TUI: %v", two.address, err)
	}
	waitFor(t, openCodeSharedTimeout, two.address+" leaving the room", func() bool {
		return c.row(t, two.address) == nil
	})
	if c.row(t, one.address) == nil {
		t.Fatalf("%s left the room with %s", one.address, two.address)
	}
	if _, stderr, err := execChat(t, c.cfg, two.token, "members"); err == nil {
		t.Errorf("members as %s succeeded after its TUI died; want a refusal\nstderr:\n%s",
			two.token, stderr)
	}
	if status := openCodeRead(t, c.mcpAddr, two.session); status != http.StatusNotFound {
		t.Errorf("the read for the session %s's TUI showed answered %d, want %d",
			two.address, status, http.StatusNotFound)
	}

	since := len(c.states(t, one.token))
	c.mention(t, "opencode-1", "still there?")
	c.waitBobReads(t, sentBy(one.address))
	c.waitTurn(t, one.token, since)
	if n := c.notices(t, one.session); n != 1 {
		t.Errorf("%s's session holds %d prompts of a notice, want 1", one.address, n)
	}
}

// A session stays with the replica whose TUI showed it first. A turn runs in
// two's session, and while it runs, two's TUI moves to one's session. The
// server answers two's registration that the session is another TUI's, and
// two's plugin reports two done, because the end of the turn it left no longer
// reaches it. A mention of one is prompted into the session both TUIs show,
// the chat_send the model answers it with there goes out as one, and two
// reports nothing for that turn. A mention of two is prompted nowhere and stays
// unread. Once one's TUI dies, the session passes to two, whose plugin hears
// that it is its own when it registers the session again.
func TestChatOpenCodeShared_ASessionStaysWithTheTUIThatShowedItFirst(t *testing.T) {
	// The model holds its answer to a prompt asking it to take its time until
	// the case lets it go.
	holding := make(chan struct{})
	hold := make(chan struct{})
	release := sync.OnceFunc(func() { close(hold) })
	var held sync.Once
	c := startOpenCodeShared(t, func(ctx context.Context, req mockRequest) mockReply {
		if role, last := req.last(); role == "user" && strings.Contains(last, "take your time") &&
			req.offers(opencodeSendTool) {
			held.Do(func() {
				close(holding)
				select {
				case <-hold:
				case <-ctx.Done():
				}
			})
		}
		return answerAsReplica(req)
	})
	t.Cleanup(release)

	one := c.attach(t, "tok-oc1", "alpha/opencode-1")
	two := c.attach(t, "tok-oc2", "alpha/opencode-2")

	// A turn in two's session, held mid-way.
	since2, before1 := len(c.states(t, two.token)), c.states(t, one.token)
	c.server.promptAsync(t, two.session, "take your time")
	select {
	case <-holding:
	case <-time.After(openCodeSharedTimeout):
		t.Fatal("the model never received the prompt of two's turn")
	}
	waitFor(t, openCodeSharedTimeout, two.address+" being listed working", func() bool {
		row := c.row(t, two.address)
		return len(row) == 5 && row[2] == "working"
	})

	// Every attached TUI follows the selection, and one's already shows the
	// session, so two's alone moves.
	c.server.call(t, http.MethodPost, "/tui/select-session",
		fmt.Sprintf(`{"sessionID":%q}`, one.session))
	waitFor(t, openCodeSharedTimeout, two.address+"'s TUI hearing the session is another's",
		func() bool {
			return strings.Contains(c.server.output.String(),
				one.session+" is another TUI's session in the crabswarm room")
		})
	// Still mid-turn: the done is the plugin letting the session go, not the
	// end of the turn.
	c.waitTurn(t, two.token, since2)
	release()
	c.server.waitIdle(t, two.session)
	if got := c.states(t, one.token); !slices.Equal(got, before1) {
		t.Errorf("%s's move published %q for %s, which had %q before it",
			two.address, got, one.token, before1)
	}

	// A mention of one, answered in the session both TUIs show.
	since1, after2 := len(c.states(t, one.token)), c.states(t, two.token)
	c.mention(t, "opencode-1", "ping one")
	c.waitBobReads(t, sentBy(one.address))
	c.waitTurn(t, one.token, since1)
	if n := c.notices(t, one.session); n != 1 {
		t.Errorf("%s's session holds %d prompts of a notice, want 1", one.address, n)
	}
	if got := c.states(t, two.token); !slices.Equal(got, after2) {
		t.Errorf("the turns in the sessions %s's TUI left and moved to published %q for it, "+
			"which had %q before them", two.address, got, after2)
	}

	// A mention of two, which its TUI has no session of its own to prompt.
	c.mention(t, "opencode-2", "ping two")
	waitFor(t, openCodeSharedTimeout, two.address+"'s plugin dropping the notice", func() bool {
		return strings.Contains(c.server.output.String(),
			"a chat notice from alpha/bob arrived while this TUI shows "+one.session+
				", a session it does not own")
	})
	if n := c.notices(t, one.session); n != 1 {
		t.Errorf("%s's session holds %d prompts of a notice, want only %s's own",
			one.address, n, one.address)
	}
	if n := c.notices(t, two.session); n != 0 {
		t.Errorf("the session %s's TUI left holds %d prompts of a notice, want 0", two.address, n)
	}
	if c.mock.sawRequest("ping two") {
		t.Errorf("the model was handed the mention of %s", two.address)
	}

	// The session passes to two once one's TUI dies. Only two's plugin is left
	// to say so past this point.
	mark := len(c.server.output.String())
	if err := one.tui.Process.Kill(); err != nil {
		t.Fatalf("kill %s's TUI: %v", one.address, err)
	}
	waitFor(t, openCodeSharedTimeout, one.address+" leaving the room", func() bool {
		return c.row(t, one.address) == nil
	})
	waitFor(t, openCodeSharedTimeout, two.address+"'s TUI hearing the session is its own",
		func() bool {
			return strings.Contains(c.server.output.String()[mark:],
				one.session+" is this TUI's session in the crabswarm room")
		})

	if got := runChat(t, c.cfg, two.token, "read"); !strings.Contains(got, "ping two") {
		t.Errorf("%s's read = %q, want the mention its TUI dropped still unread", two.address, got)
	}
	if keys := stubSendKeys(t, c.cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}
}

// A subagent's session is never one a TUI shows, so a crabswarm call from it
// acts as nobody. The model in the replica's session hands a task to a
// subagent, whose model calls chat_send: the server's plugin names the
// subagent's session on the call, and the MCP server refuses it for having no
// crabswarm identity. Nothing is sent, and the replica is still the only
// member its TUI makes.
func TestChatOpenCodeShared_ASubagentsCallActsAsNobody(t *testing.T) {
	const task = "subagent: tell bob hello"
	c := startOpenCodeShared(t, func(_ context.Context, req mockRequest) mockReply {
		role, last := req.last()
		switch {
		case role == "user" && strings.Contains(last, "delegate") && req.offers("task"):
			return mockReply{tool: "task", args: fmt.Sprintf(
				`{"description":"tell bob hello","prompt":%q,"subagent_type":"general"}`, task)}
		case role == "user" && strings.Contains(last, task) && req.offers(opencodeSendTool):
			return mockReply{
				tool: opencodeSendTool,
				args: `{"to":"bob","message":"from a subagent"}`,
			}
		}
		return mockReply{}
	})
	one := c.attach(t, "tok-oc1", "alpha/opencode-1")

	c.server.takeTurn(t, one.session, "delegate")

	var children []struct {
		ID       string `json:"id"`
		ParentID string `json:"parentID"`
	}
	answer := c.server.call(t, http.MethodGet, "/session/"+one.session+"/children", "")
	if err := json.Unmarshal([]byte(answer), &children); err != nil {
		t.Fatalf("decode the children of %s: %v\n%s", one.session, err, answer)
	}
	if len(children) != 1 || children[0].ParentID != one.session {
		t.Fatalf("children of %s = %s, want the one session of the subagent", one.session, answer)
	}
	child := children[0].ID
	// The refusal names the session as the tool result's JSON string carries it.
	refusal := fmt.Sprintf(`(\"%s\") has no crabswarm identity`, child)
	if !c.mock.sawRequest(`"role":"tool"`, refusal) {
		t.Errorf("the subagent's model never got chat_send refused for its session %s "+
			"having no crabswarm identity", child)
	}

	if got := runChat(t, c.cfg, c.bob, "read"); strings.Contains(got, "from a subagent") {
		t.Errorf("bob's read = %q, want nothing the subagent sent", got)
	}
	members := memberAddresses(runChat(t, c.cfg, c.bob, "members"))
	slices.Sort(members)
	if want := []string{"alpha/bob", one.address}; !slices.Equal(members, want) {
		t.Errorf("members = %v, want %v", members, want)
	}
}

// The TUI's plugin outlives the MCP server it streams from. The server stops
// and the replica's member leaves with it; another starts on the same address,
// and the plugin opens its notice stream there again and registers the session
// it shows. A mention after that is prompted into the session, and the model's
// chat_send there goes out as the replica again.
func TestChatOpenCodeShared_TheTUIAttendsAgainOnARestartedMCP(t *testing.T) {
	c := startOpenCodeShared(t, func(_ context.Context, req mockRequest) mockReply {
		return answerAsReplica(req)
	})
	one := c.attach(t, "tok-oc1", "alpha/opencode-1")

	stopProcess(t, c.mcp)
	waitFor(t, openCodeSharedTimeout, one.address+" leaving with the MCP server", func() bool {
		return c.row(t, one.address) == nil
	})
	c.mcp = startOpenCodeMCPOn(t, c.cfg, c.mcpAddr)
	c.waitNative(t, one.address)
	waitFor(t, openCodeSharedTimeout, "the TUI registering its session again", func() bool {
		return openCodeRead(t, c.mcpAddr, one.session) == http.StatusNoContent
	})

	since := len(c.states(t, one.token))
	c.mention(t, "opencode-1", "after the restart")
	c.waitBobReads(t, sentBy(one.address))
	c.waitTurn(t, one.token, since)
	if n := c.notices(t, one.session); n != 1 {
		t.Errorf("%s's session holds %d prompts of a notice, want 1", one.address, n)
	}
	if !c.mock.sawRequest(chatDeliveryNotices[0], "after the restart") {
		t.Errorf("no chat_send result carried the mention read mid-turn")
	}
}

// A TUI that has shown no session yet still attends, and has nowhere to prompt
// a notice: the mention stays unread. Once the TUI shows a session, the read
// that ends the first turn there hands it over.
func TestChatOpenCodeShared_AMentionBeforeASessionWaitsForATurn(t *testing.T) {
	c := startOpenCodeShared(t, func(context.Context, mockRequest) mockReply {
		return mockReply{}
	})
	startOpenCodeTUI(t, c.opencode, c.server, c.cfg, "tok-oc1", "")
	c.waitNative(t, "alpha/opencode-1")

	c.mention(t, "opencode-1", "before any session")
	waitFor(t, openCodeSharedTimeout, "the TUI's plugin dropping the notice", func() bool {
		return strings.Contains(c.server.output.String(),
			"a chat notice from alpha/bob arrived before this TUI showed a session")
	})
	if c.mock.sawRequest(openCodeSharedNotice) {
		t.Errorf("the notice was prompted into a session the TUI does not show")
	}

	// Registration is watched through a call of chat_members, which reads
	// nothing: the plugin's read route would take the mention.
	probe := dialOpenCodeMCP(t, c.mcpAddr)
	session := c.server.createSession(t)
	c.server.call(t, http.MethodPost, "/tui/select-session",
		fmt.Sprintf(`{"sessionID":%q}`, session))
	waitFor(t, openCodeSharedTimeout, "the TUI registering the session it moved to", func() bool {
		_, failed := chatToolResult(t, probe, "chat_members",
			map[string]any{"_crabswarm_session": session})
		return !failed
	})

	since := len(c.states(t, "tok-oc1"))
	c.server.takeTurn(t, session, "first turn")
	waitFor(t, openCodeSharedTimeout, "the mention being handed over at the end of the turn",
		func() bool {
			return c.mock.sawRequest(chatDeliveryNotices[1], "before any session")
		})
	c.waitTurn(t, "tok-oc1", since)
	if got := runChat(t, c.cfg, "tok-oc1", "read"); got != "no pending messages\n" {
		t.Errorf("the member's read after the turn = %q, want the mention handed over", got)
	}
}

// dialOpenCodeMCP opens an MCP session on the MCP server at addr the way
// opencode serve does: under the name OpenCode gives itself and with no token
// header, so a call on it acts as the member that owns the session the call
// names.
func dialOpenCodeMCP(t *testing.T, addr string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "opencode", Version: "1.18.32"}, nil)
	session, err := client.Connect(t.Context(),
		&mcp.StreamableClientTransport{Endpoint: "http://" + addr + "/mcp"}, nil)
	if err != nil {
		t.Fatalf("connect to crabswarm mcp as opencode: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}
