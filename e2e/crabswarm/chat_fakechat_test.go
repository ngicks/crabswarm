package crabswarm_test

import (
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// Claude Code holds no channel this process could write down: the official
// fakechat plugin runs a loopback HTTP server beside the session that registered
// it, and everything that server is posted reaches the model as a turn it starts
// once it is idle. The bridge is therefore only a sender, and what follows drives
// it against a server standing in for the plugin. The whole contract between the
// two is in that server: the page it answers a probe with, and the upload route
// it takes a notice on.

// fakechatUpload is one notice as the plugin reads it off its own form: the id
// it delivers the message under, and the text the model is handed.
type fakechatUpload struct {
	ID   string
	Text string
}

// fakechatRequest is how one upload arrived — the request line and the multipart
// fields it carried, sorted — which is what the recording of the real plugin's
// traffic is compared against.
type fakechatRequest struct {
	Call   string
	Fields []string
}

// fakeFakechat is the plugin's loopback server, recording what the bridge
// posted.
type fakeFakechat struct {
	port string
	// page is what a probe reads, which is the plugin's own page as it was
	// recorded. A page written by hand here would only agree with whatever the
	// bridge happens to look for, where this agrees with the plugin.
	page []byte

	mu       sync.Mutex
	uploaded []fakechatUpload
	arrived  []fakechatRequest
}

// startFakeFakechat serves as the plugin on a port of the kernel's choosing,
// which is what every case wants but the one about a plugin that is not there.
func startFakeFakechat(t *testing.T) *fakeFakechat {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen as the fakechat plugin: %v", err)
	}
	return serveFakeFakechat(t, ln)
}

// startFakeFakechatOn serves as the plugin on the port the caller already told a
// bridge about, which is how a plugin that came up after the session beside it
// is played.
func startFakeFakechatOn(t *testing.T, port string) *fakeFakechat {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		t.Fatalf("listen as the fakechat plugin on port %s: %v", port, err)
	}
	return serveFakeFakechat(t, ln)
}

// serveFakeFakechat answers on ln for the rest of the test.
func serveFakeFakechat(t *testing.T, ln net.Listener) *fakeFakechat {
	t.Helper()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("read the port off %s: %v", ln.Addr(), err)
	}
	// Read once here rather than per request: the recording does not change while
	// a case runs, and reading it off disk on the request a probe is waiting for
	// would put a file read in the middle of an attendance.
	fake := &fakeFakechat{port: port, page: harnessFixtureBytes(t, "fakechat-page.html")}
	server := httptest.NewUnstartedServer(fake.handler())
	// The test server opens a listener of its own, and the one that matters is
	// the port the bridge was told about.
	_ = server.Listener.Close()
	server.Listener = ln
	server.Start()
	t.Cleanup(server.Close)
	return fake
}

// handler answers the two routes the plugin's server holds that this end uses:
// the page a probe reads, and the upload a notice is posted as. Everything else
// is a 404, the way it is on the real one — a bridge writing anywhere else is a
// bridge writing to nobody.
func (f *fakeFakechat) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(f.page)
		case r.Method == http.MethodPost && r.URL.Path == "/upload":
			f.take(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// take records one upload and answers 204, as the plugin does once the notice is
// on its way to the session. Probes are not recorded: the bridge asks before
// every attendance attempt, so a case that counts uploads would be counting the
// retries instead.
func (f *fakeFakechat) take(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uploaded = append(f.uploaded, fakechatUpload{
		ID:   r.FormValue("id"),
		Text: r.FormValue("text"),
	})
	f.arrived = append(f.arrived, fakechatRequest{
		Call:   r.Method + " " + r.URL.Path,
		Fields: slices.Sorted(maps.Keys(r.MultipartForm.Value)),
	})
	w.WriteHeader(http.StatusNoContent)
}

// uploads is what the plugin was handed, oldest first.
func (f *fakeFakechat) uploads() []fakechatUpload {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.uploaded)
}

// requests is how each of those arrived, oldest first.
func (f *fakeFakechat) requests() []fakechatRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.arrived)
}

// waitUploads blocks until the plugin has been handed exactly the texts want, so
// a case pins both what reached the session and that nothing else did. The ids
// are left to [assertFakechatIDs]: they are numbered by the sending process, and
// a case that spelled them would be asserting on its own position in that count.
func (f *fakeFakechat) waitUploads(t *testing.T, want ...string) {
	t.Helper()
	var got []string
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		got = nil
		for _, up := range f.uploads() {
			got = append(got, up.Text)
		}
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the plugin was handed\n%q\nwant\n%q", got, want)
}

// fakechatUploadIDPattern is the shape of the id the bridge numbers an upload
// with.
var fakechatUploadIDPattern = regexp.MustCompile(`^crabswarm-[1-9][0-9]*$`)

// assertFakechatIDs checks every upload carries an id of that shape and its own.
// The plugin delivers a message under the id it was handed and repeats it in the
// event's meta, so two mentions sharing one would be two the transcript cannot
// tell apart.
func assertFakechatIDs(t *testing.T, fake *fakeFakechat) {
	t.Helper()
	seen := map[string]bool{}
	for _, up := range fake.uploads() {
		if !fakechatUploadIDPattern.MatchString(up.ID) {
			t.Errorf("upload id = %q, want it to match %s", up.ID, fakechatUploadIDPattern)
		}
		if seen[up.ID] {
			t.Errorf("two uploads carry the id %q", up.ID)
		}
		seen[up.ID] = true
	}
}

// fakechatArrival is the line the room words a fresh mention as, which is the
// whole of what the plugin is posted for one.
func fakechatArrival(from string) string {
	return "[crabswarm chat] new message from " + from +
		" — read it with the chat_read tool and respond with chat_send," +
		" both from crabswarm-mcp"
}

// startFakechatBridge starts `crabswarm mcp` the way a Claude Code that
// registered the plugin as a channel starts it: named as Claude Code in the
// handshake, and carrying both launch-time variables — that a plugin was
// registered, and the loopback port it was registered on.
//
// The port is always passed. A bridge left without one posts to the plugin's
// default 8787, which on a developer host is the live session they are reading
// this in.
func startFakechatBridge(t *testing.T, cfgPath, token, port string) *mcp.ClientSession {
	t.Helper()
	// Appended after the scrubbed environment, which drops every CRABSWARM_
	// variable the suite is itself running with, and the plugin's port with them.
	return startChatBridgeAs(t, cfgPath, token, "claude-code", append(chatEnviron(),
		harnessctl.ClaudeChannelEnv+"=1",
		harnessctl.FakechatPortEnv+"="+port))
}

// A Claude Code whose launcher registered the fakechat plugin attends as a
// member the daemon never types at, and is handed its mentions as uploads the
// plugin turns into turns — while it is idle, never mid-turn, and once the turn
// ends as a count of what piled up meanwhile.
func TestChat_TheFakechatPluginTakesTheMentionsItsDaemonNoLongerTypes(t *testing.T) {
	cfg := startChatDaemon(t)
	fake := startFakeFakechat(t)
	session := startFakechatBridge(t, cfg, "tok-ana", fake.port)
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	// Nothing of the channel is in the handshake, because the channel is the
	// plugin's. A capability left over from a session this server pushed turns
	// into itself would have Claude Code listening where nothing writes.
	res := session.InitializeResult()
	if res.Capabilities != nil && len(res.Capabilities.Experimental) > 0 {
		t.Errorf("the handshake declares the experimental capabilities %v, want none",
			res.Capabilities.Experimental)
	}

	// The daemon has it from the handshake and the variables in the environment:
	// which CLI the member runs, and that its mentions are its own server's to
	// deliver.
	roster := lines(runChat(t, cfg, "tok-bob", "members"))
	slices.Sort(roster)
	want := []string{
		chatBridgeAna + "  agent  done  claude-code  native",
		chatBridgeBob + "  agent  done  other  terminal",
	}
	if !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}

	// Mentioned while idle: one upload carries the notice, and no keystroke is
	// typed anywhere.
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	arrival := fakechatArrival(chatBridgeBob)
	fake.waitUploads(t, arrival)
	assertFakechatIDs(t, fake)
	// A nudge is typed while the send is being served, so by now there would be
	// one to read.
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Mentioned mid-turn: nothing is posted at a harness that is working, and the
	// mention waits in the room where it already is.
	runChat(t, cfg, "tok-ana", "report-state", "working")
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "and the rebase after it")
	if got := fake.uploads(); len(got) != 1 {
		t.Errorf("the plugin was handed %+v mid-turn, want only the arrival", got)
	}

	// The turn ends and what waited is posted as one line. Both messages are
	// still unread — an upload is not a read — so the count is two.
	runChat(t, cfg, "tok-ana", "report-state", "done")
	fake.waitUploads(t, arrival, "[crabswarm chat] 2 unread messages mention you"+
		" — read them with the chat_read tool and respond with chat_send,"+
		" both from crabswarm-mcp")
	assertFakechatIDs(t, fake)
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Delivered is not read: both messages are waiting where the agent's own read
	// finds them.
	read := runChat(t, cfg, "tok-ana", "read")
	for _, want := range []string{"the migration needs you", "and the rebase after it"} {
		if !strings.Contains(read, want) {
			t.Errorf("read as %s = %q, want it to carry %q", chatBridgeAna, read, want)
		}
	}
}

// What the bridge posts has the shape of the request the real plugin's server
// answered: the same route, and the same multipart fields. The plugin reads the
// form by field name, so a field renamed on this end is a notice it delivers
// without its text or under no id at all.
func TestChat_TheFakechatUploadHasTheRecordedShape(t *testing.T) {
	cfg := startChatDaemon(t)
	fake := startFakeFakechat(t)
	startFakechatBridge(t, cfg, "tok-ana", fake.port)
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	fake.waitUploads(t, fakechatArrival(chatBridgeBob))

	recorded := harnessFixtureRequest(t, "fakechat-upload.http")
	if err := recorded.ParseMultipartForm(1 << 20); err != nil {
		t.Fatalf("parse the recorded upload's form: %v", err)
	}
	want := []fakechatRequest{{
		Call:   recorded.Method + " " + recorded.URL.Path,
		Fields: slices.Sorted(maps.Keys(recorded.MultipartForm.Value)),
	}}
	if got := fake.requests(); !slices.EqualFunc(got, want, func(
		got, want fakechatRequest,
	) bool {
		return got.Call == want.Call && slices.Equal(got.Fields, want.Fields)
	}) {
		t.Errorf("the bridge posted %+v, want the recorded %+v", got, want)
	}
}

// A launch that promised a plugin and started none leaves the member out of the
// room rather than in it as one the daemon does not type at: the bridge asks the
// port for the plugin's page before every attendance attempt, and attends on the
// first answer that is it. So a plugin that comes up after the process beside it
// is picked up by the same loop that refused while it was missing.
func TestChat_AMemberWhoseFakechatIsMissingDoesNotAttend(t *testing.T) {
	cfg := startChatDaemon(t)
	port := freeLoopbackPort(t)
	startFakechatBridge(t, cfg, "tok-ana", port)
	attendChatBridges(t, cfg, "tok-bob")

	// Watched for a while rather than asked once: the bridge retries its
	// attendance with a backoff that tops out at two seconds, so an attendance
	// the missing plugin failed to stop would land inside this window.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		listed := memberAddresses(runChat(t, cfg, "tok-bob", "members"))
		if slices.Contains(listed, chatBridgeAna) {
			t.Fatalf("%s attended with nothing listening on port %s; the room held %v",
				chatBridgeAna, port, listed)
		}
		time.Sleep(200 * time.Millisecond)
	}

	startFakeFakechatOn(t, port)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
	// And it attends as what it is: a Claude Code whose mentions are its own
	// server's to post.
	want := chatBridgeAna + "  agent  done  claude-code  native"
	if roster := lines(runChat(t, cfg, "tok-bob", "members")); !slices.Contains(roster, want) {
		t.Errorf("members = %v, want one of them to be %q", roster, want)
	}
}

// freeLoopbackPort takes a loopback port and gives it straight back, so the
// caller holds a port number nothing is listening on. Whatever may grab it in
// between is the race this has to live with, and it is as close as a test gets to
// a plugin its launcher promised and never started.
func freeLoopbackPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("take a loopback port: %v", err)
	}
	_, port, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatalf("read the port off %s: %v", ln.Addr(), err)
	}
	if err := ln.Close(); err != nil {
		t.Fatalf("give the loopback port back: %v", err)
	}
	return port
}

// One host runs as many sessions as the person opened, each with a plugin of its
// own on a port of its own. A mention therefore has to reach the session it names
// and no other, which is the port its own bridge was launched with and nothing
// the room knows about.
func TestChat_TwoFakechatSessionsOnOneHostEachGetTheirOwn(t *testing.T) {
	// A third agent beside the default roster's teammates: the two sessions have
	// to share a room for one message to be able to reach either of them.
	cfg := startChatDaemonWith(t, append(defaultStubCommands(),
		stubCommand{token: "tok-dave", dir: chatRoom, project: "alpha"}))
	anaPlugin := startFakeFakechat(t)
	davePlugin := startFakeFakechat(t)
	startFakechatBridge(t, cfg, "tok-ana", anaPlugin.port)
	startFakechatBridge(t, cfg, "tok-dave", davePlugin.port)
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatAttendance(t, cfg, "tok-dave", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeDave, 30*time.Second)

	arrival := fakechatArrival(chatBridgeBob)

	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	anaPlugin.waitUploads(t, arrival)
	if got := davePlugin.uploads(); len(got) != 0 {
		t.Errorf("dave's plugin was handed %+v, want nothing: ana was the one mentioned", got)
	}

	runChat(t, cfg, "tok-bob", "send", chatBridgeDave, "and the rebase is yours")
	davePlugin.waitUploads(t, arrival)
	if got := anaPlugin.uploads(); len(got) != 1 {
		t.Errorf("ana's plugin was handed %+v, want its own mention alone", got)
	}
}

// chatBridgeDave is the address of the third agent the case above adds, spelled
// the way the daemon derives it from the token.
const chatBridgeDave = "alpha/agent-tok-dave"
