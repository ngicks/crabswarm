package harnessctl

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// relayEnv is the environment an OpenCode running the crabswarm plugin starts
// this server in: the plugin's own loopback URL and nothing else.
func relayEnv(url string) func(string) string {
	return func(name string) string {
		if name == OpenCodeRelayEnv {
			return url
		}
		return ""
	}
}

// fakeRelay is the plugin's side of the relay, recording what it was posted.
type fakeRelay struct {
	server *httptest.Server
	status int

	mu      sync.Mutex
	method  string
	path    string
	content string
	bodies  []openCodeNotice
}

// startFakeRelay answers every post with status and keeps the notices.
func startFakeRelay(t *testing.T, status int, answer string) *fakeRelay {
	t.Helper()
	r := &fakeRelay{status: status}
	r.server = httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter, req *http.Request,
	) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var notice openCodeNotice
		_ = json.Unmarshal(body, &notice)
		r.mu.Lock()
		r.method = req.Method
		r.path = req.URL.Path
		r.content = req.Header.Get("Content-Type")
		r.bodies = append(r.bodies, notice)
		r.mu.Unlock()
		w.WriteHeader(r.status)
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(r.server.Close)
	return r
}

func (r *fakeRelay) url() string { return r.server.URL + "/nudge" }

func (r *fakeRelay) delivered() []openCodeNotice {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bodies
}

// A harness whose OpenCode carries no plugin carries no relay either, and the
// member attends as a terminal one the daemon types at.
func TestOpenCode_WithoutTheRelayThereIsNoChannel(t *testing.T) {
	h := Detect("opencode", noEnv, nil)
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_OPENCODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
}

// What the plugin is handed: one post of the notice as the plugin reads it, the
// whole line under content, the sender beside it — empty on the notice that
// counts what waits, which is about the room rather than about one message —
// and the room the member attends, which every notice carries because an agent
// can be looking at more than one crabswarm at a time.
func TestOpenCode_PostsTheNoticeToTheRelay(t *testing.T) {
	relay := startFakeRelay(t, http.StatusAccepted, "")

	h := Detect("opencode", relayEnv(relay.url()), nil)
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_OPENCODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	assert.NilError(t, h.Deliver(t.Context(), Notice{
		From: "beta/bob",
		Room: "/work",
		Text: "[crabswarm chat] new message from beta/bob",
	}))
	assert.NilError(t, h.Deliver(t.Context(), Notice{
		Room: "/work",
		Text: "[crabswarm chat] 2 unread messages mention you",
	}))

	assert.DeepEqual(t, relay.delivered(), []openCodeNotice{
		{
			Content: "[crabswarm chat] new message from beta/bob",
			From:    "beta/bob",
			Room:    "/work",
		},
		{Content: "[crabswarm chat] 2 unread messages mention you", Room: "/work"},
	})

	relay.mu.Lock()
	defer relay.mu.Unlock()
	assert.Equal(t, relay.method, http.MethodPost)
	assert.Equal(t, relay.path, "/nudge")
	assert.Equal(t, relay.content, "application/json")
}

// The plugin refuses while it knows no session to prompt, and a refusal is an
// error rather than a delivery: the mention is still unread, and the next
// report that ends a turn comes back to it.
func TestOpenCode_ARefusedNoticeIsNotDelivered(t *testing.T) {
	relay := startFakeRelay(t, http.StatusConflict, "no session yet")

	err := Detect("opencode", relayEnv(relay.url()), nil).
		Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorContains(t, err, "409")
	assert.ErrorContains(t, err, "no session yet")
}

// A relay that took the connection and then said nothing is given up on, so one
// wedged plugin costs the delivery rather than the feed every delivery runs on.
func TestOpenCode_ARelayThatNeverAnswersIsGivenUpOn(t *testing.T) {
	// The handler is released by the case rather than by the client giving up:
	// httptest's Close waits for the handlers it is serving to return, and the
	// cleanups run in reverse, so the release happens first.
	release := make(chan struct{})
	blocked := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	t.Cleanup(blocked.Close)
	t.Cleanup(func() { close(release) })

	h := openCode{relay: blocked.URL + "/nudge", timeout: 50 * time.Millisecond}
	err := h.Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
