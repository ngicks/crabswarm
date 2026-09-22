package crabswarm_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// The OpenCode channel runs the other way round from the rest: the plugin
// inside OpenCode opens a loopback relay, hands its address to the MCP server
// it declares, and the server posts a mention there for the plugin to prompt
// into the session the person is driving. What follows drives the server's half
// of that against a relay standing in for the plugin, which is the half this
// repository ships as Go — the plugin's own half needs an OpenCode to run in,
// and the case beside this one drives that.

// relayNotice is one notice as the plugin reads it off the relay.
type relayNotice struct {
	Content string `json:"content"`
	From    string `json:"from"`
}

// fakeOpenCodeRelay is the plugin's listener, recording what the bridge posted.
type fakeOpenCodeRelay struct {
	url string

	mu       sync.Mutex
	notices  []relayNotice
	requests []string
}

// startFakeOpenCodeRelay answers every notice the way the plugin does once it
// has handed one to a session: accepted, with nothing to read.
func startFakeOpenCodeRelay(t *testing.T) *fakeOpenCodeRelay {
	t.Helper()
	relay := &fakeOpenCodeRelay{}
	server := httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter, r *http.Request,
	) {
		var notice relayNotice
		if err := json.NewDecoder(r.Body).Decode(&notice); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		relay.mu.Lock()
		relay.notices = append(relay.notices, notice)
		relay.requests = append(relay.requests, r.Method+" "+r.URL.Path)
		relay.mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)
	relay.url = server.URL + "/nudge"
	return relay
}

// delivered is what the relay was handed, oldest first.
func (r *fakeOpenCodeRelay) delivered() []relayNotice {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.notices)
}

// calls is how each of those arrived, oldest first.
func (r *fakeOpenCodeRelay) calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.requests)
}

// waitNotices blocks until the relay has been handed exactly want, so a case
// pins both what reached the plugin and that nothing else did.
func (r *fakeOpenCodeRelay) waitNotices(t *testing.T, want ...relayNotice) {
	t.Helper()
	var got []relayNotice
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		got = r.delivered()
		if slices.Equal(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the relay was handed\n%+v\nwant\n%+v", got, want)
}

// An OpenCode bridge started by the plugin attends as a member the daemon never
// types at and posts its mentions to the relay the plugin opened — while the
// harness is idle, never mid-turn, and once the turn ends as a count of what
// piled up meanwhile.
func TestChatOpenCode_ABridgeDeliversThroughThePluginsRelay(t *testing.T) {
	cfg := startChatDaemon(t)
	relay := startFakeOpenCodeRelay(t)
	// Appended after the scrubbed environment, which drops every CRABSWARM_
	// variable the suite is itself running with. This is the one the plugin
	// puts into the environment of the server it declares.
	startChatBridgeAs(t, cfg, "tok-ana", "opencode",
		append(chatEnviron(), harnessctl.OpenCodeRelayEnv+"="+relay.url))
	attendChatBridges(t, cfg, "tok-bob")
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitChatRosterHas(t, cfg, "tok-bob", chatBridgeAna, 30*time.Second)

	// The daemon has it from the handshake and the relay in the environment:
	// which CLI the member runs, and that its mentions are its own server's to
	// deliver.
	roster := lines(runChat(t, cfg, "tok-bob", "members"))
	slices.Sort(roster)
	want := []string{
		chatBridgeAna + "  agent  done  opencode  native",
		chatBridgeBob + "  agent  done  other  terminal",
	}
	if !slices.Equal(roster, want) {
		t.Errorf("members = %v, want %v", roster, want)
	}

	// Mentioned while idle: one post carries the notice and who wrote it, and
	// no keystroke is typed anywhere.
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "the migration needs you")
	arrival := relayNotice{
		From: chatBridgeBob,
		Content: "[crabswarm chat] new message from " + chatBridgeBob +
			" — read it with the chat_read tool and respond with chat_send," +
			" both from crabswarm-mcp",
	}
	relay.waitNotices(t, arrival)
	// A nudge is typed while the send is being served, so by now there would be
	// one to read.
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// Mentioned mid-turn: nothing is posted at a harness that is working, and
	// the mention waits in the room where it already is.
	runChat(t, cfg, "tok-ana", "report-state", "working")
	runChat(t, cfg, "tok-bob", "send", chatBridgeAna, "and the rebase after it")
	if got := relay.delivered(); !slices.Equal(got, []relayNotice{arrival}) {
		t.Errorf("the relay was handed %+v mid-turn, want only the arrival", got)
	}

	// The turn ends and what waited is posted as one line, naming no sender:
	// by then which of them wrote is what the read itself says. Both messages
	// are still unread — a notice is not a read — so the count is two.
	runChat(t, cfg, "tok-ana", "report-state", "done")
	relay.waitNotices(t, arrival, relayNotice{
		Content: "[crabswarm chat] 2 unread messages mention you — " +
			"read them with the chat_read tool and respond with chat_send," +
			" both from crabswarm-mcp",
	})
	if keys := stubSendKeys(t, cfg); keys != nil {
		t.Errorf("cmdman send-keys invocations = %q, want none", keys)
	}

	// One post per notice, on the route the plugin listens on.
	if got, want := relay.calls(), []string{
		"POST /nudge",
		"POST /nudge",
	}; !slices.Equal(
		got,
		want,
	) {
		t.Errorf("the relay was called %q, want %q", got, want)
	}

	// Delivered is not read: both messages are waiting where the agent's own
	// read finds them.
	read := runChat(t, cfg, "tok-ana", "read")
	for _, want := range []string{"the migration needs you", "and the rebase after it"} {
		if !strings.Contains(read, want) {
			t.Errorf("read as %s = %q, want it to carry %q", chatBridgeAna, read, want)
		}
	}
}
