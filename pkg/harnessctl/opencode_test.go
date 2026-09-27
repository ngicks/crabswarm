package harnessctl

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// An OpenCode MCP session names the harness and nothing about a TUI, so it is a
// terminal member whatever its environment says: the one channel OpenCode has
// is the notice stream its plugin opens, which an environment cannot point at.
func TestOpenCode_ASessionAloneIsTerminal(t *testing.T) {
	h := Detect("opencode", func(name string) string {
		if name == "CRABSWARM_OPENCODE_RELAY" {
			return "http://127.0.0.1:1/nudge"
		}
		return ""
	})
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_OPENCODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
}

// noticeListener is a plugin holding one notice stream open.
type noticeListener struct {
	lines chan openCodeNotice
}

// serveNotices serves notices on a loopback server, each request one stream
// attached until its client goes away, and returns the URL a plugin requests.
func serveNotices(t *testing.T, notices *OpenCodeNotices) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stream := notices.Attach(w)
		defer stream.Detach()
		if err := stream.Open(); err != nil {
			return
		}
		select {
		case <-r.Context().Done():
		case <-stream.Broken():
		}
	}))
	t.Cleanup(server.Close)
	return server.URL
}

// listen opens a stream on url and returns once it is attached, which is when
// its header has arrived. The stream closes when the test ends.
func listen(t *testing.T, url string) *noticeListener {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	assert.NilError(t, err)
	resp, err := http.DefaultClient.Do(req)
	assert.NilError(t, err)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, resp.Header.Get("Content-Type"), OpenCodeNoticeContentType)

	l := &noticeListener{lines: make(chan openCodeNotice, 8)}
	var reading errgroup.Group
	reading.Go(func() error {
		defer close(l.lines)
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			var n openCodeNotice
			if err := json.Unmarshal(scanner.Bytes(), &n); err != nil {
				return err
			}
			l.lines <- n
		}
		return nil
	})
	t.Cleanup(func() {
		_ = resp.Body.Close()
		assert.NilError(t, reading.Wait())
	})
	return l
}

// next is the next notice on the stream.
func (l *noticeListener) next(t *testing.T) openCodeNotice {
	t.Helper()
	select {
	case n := <-l.lines:
		return n
	case <-time.After(5 * time.Second):
		t.Fatal("no notice arrived on the stream")
		return openCodeNotice{}
	}
}

// waitStreams blocks until n streams are attached to notices.
func waitStreams(t *testing.T, notices *OpenCodeNotices, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		notices.mu.Lock()
		got := len(notices.streams)
		notices.mu.Unlock()
		if got == n {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("%d notice streams never became attached", n)
}

// Every stream open for the member carries every notice, one JSON line each:
// the whole line under content, and the sender beside it — empty on the notice
// that counts what waits, which is about the room rather than about one
// message.
func TestOpenCode_DeliversToEveryOpenStream(t *testing.T) {
	notices := NewOpenCodeNotices()
	assert.Equal(t, notices.Kind(), chatv1.Harness_HARNESS_OPENCODE)
	assert.Equal(t, notices.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	url := serveNotices(t, notices)
	first, second := listen(t, url), listen(t, url)
	waitStreams(t, notices, 2)

	assert.NilError(t, notices.Deliver(t.Context(), Notice{
		From: "beta/bob",
		Room: "/work",
		Text: "[crabswarm chat] new message from beta/bob",
	}))
	assert.NilError(t, notices.Deliver(t.Context(), Notice{
		Room: "/work",
		Text: "[crabswarm chat] 2 unread messages mention you",
	}))

	for _, l := range []*noticeListener{first, second} {
		assert.DeepEqual(t, l.next(t), openCodeNotice{
			Content: "[crabswarm chat] new message from beta/bob",
			From:    "beta/bob",
		})
		assert.DeepEqual(t, l.next(t), openCodeNotice{
			Content: "[crabswarm chat] 2 unread messages mention you",
		})
	}
}

// With no stream open there is nobody to tell, and a refusal is an error rather
// than a delivery: the mention is still unread, and the next report that ends a
// turn comes back to it.
func TestOpenCode_WithoutAStreamNothingIsDelivered(t *testing.T) {
	err := NewOpenCodeNotices().Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorIs(t, err, errNoOpenCodeStream)
}

// A stream the plugin let go of is taken off the channel, so what follows
// reaches the streams still open and nothing is written to the one that ended.
func TestOpenCode_AClosedStreamIsLeftOut(t *testing.T) {
	notices := NewOpenCodeNotices()
	url := serveNotices(t, notices)
	kept := listen(t, url)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, nil)
	assert.NilError(t, err)
	gone, err := http.DefaultClient.Do(req)
	assert.NilError(t, err)
	waitStreams(t, notices, 2)
	assert.NilError(t, gone.Body.Close())
	waitStreams(t, notices, 1)

	assert.NilError(t, notices.Deliver(t.Context(), Notice{Text: "still here"}))
	assert.DeepEqual(t, kept.next(t), openCodeNotice{Content: "still here"})
}
