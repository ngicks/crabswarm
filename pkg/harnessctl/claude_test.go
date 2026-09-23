package harnessctl

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// A fakechat plugin belonging to a live Claude Code session holds port 8787 on
// a developer's own host, and these cases run on that host. No case here may
// probe or deliver through a harness it did not hand a port of its own: the
// default port is checked by reading the URL the harness would have called,
// never by calling it.

// fakechatEnv is the environment a Claude Code launched with the plugin starts
// this server in.
func fakechatEnv(channel, port string) func(string) string {
	return func(name string) string {
		switch name {
		case ClaudeChannelEnv:
			return channel
		case FakechatPortEnv:
			return port
		default:
			return ""
		}
	}
}

// recordedFakechatPage is the page the real plugin answered GET / with, byte for
// byte as its own server wrote it.
//
// Read rather than spelled out: a probe recognises the plugin by one string it
// looks for in that page, and a stand-in serving a page written by hand would
// agree with the string this package happens to hold instead of with the page the
// plugin serves.
func recordedFakechatPage(t *testing.T) string {
	t.Helper()

	path := filepath.Join(fixtureDir, "fakechat-page.html")
	b, err := os.ReadFile(path)
	assert.NilError(t, err)
	// A recording that read as nothing would be a page every probe rejects, which
	// would pass the cases that expect a refusal.
	assert.Assert(t, len(b) > 0, "%s is empty", path)
	return string(b)
}

// uploadID is the shape of an id the plugin is posted. It numbers the uploads
// of this process, so a case can say what it looks like but not what it is.
var uploadID = regexp.MustCompile(`^crabswarm-[1-9][0-9]*$`)

// recordedUpload is one request as the plugin read it.
type recordedUpload struct {
	method string
	path   string
	fields []string
	id     string
	text   string
}

// readUpload enumerates one multipart request. Both the stand-in plugin and the
// recorded request go through this, so comparing them compares two readings
// made the same way.
func readUpload(req *http.Request) (recordedUpload, error) {
	up := recordedUpload{method: req.Method, path: req.URL.Path}
	parts, err := req.MultipartReader()
	if err != nil {
		return up, err
	}
	for {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			return up, nil
		}
		if err != nil {
			return up, err
		}
		value, err := io.ReadAll(part)
		if err != nil {
			return up, err
		}
		up.fields = append(up.fields, part.FormName())
		switch part.FormName() {
		case "id":
			up.id = string(value)
		case "text":
			up.text = string(value)
		}
	}
}

// recordedFakechatUpload is the upload the real plugin answered 204 to, read
// back out of the bytes curl put on the wire.
func recordedFakechatUpload(t *testing.T) recordedUpload {
	t.Helper()

	path := filepath.Join(fixtureDir, "fakechat-upload.http")
	f, err := os.Open(path)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	req, err := http.ReadRequest(bufio.NewReader(f))
	assert.NilError(t, err, "reading %s", path)
	up, err := readUpload(req)
	assert.NilError(t, err, "reading the form of %s", path)
	// A recording that read as an empty form would agree with anything.
	assert.Assert(t, len(up.fields) > 0, "%s carries no form fields", path)
	return up
}

// fakePlugin is the plugin's loopback server: its page on GET, and a record of
// every upload it was posted.
type fakePlugin struct {
	server *httptest.Server

	mu      sync.Mutex
	uploads []recordedUpload
}

// startFakePlugin serves page on GET / — or 404s when page is empty — and
// answers every upload with status and answer.
func startFakePlugin(t *testing.T, page string, status int, answer string) *fakePlugin {
	t.Helper()

	p := &fakePlugin{}
	p.server = httptest.NewServer(http.HandlerFunc(func(
		w http.ResponseWriter, req *http.Request,
	) {
		if req.Method == http.MethodGet {
			if page == "" {
				http.NotFound(w, req)
				return
			}
			_, _ = io.WriteString(w, page)
			return
		}
		up, err := readUpload(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		p.mu.Lock()
		p.uploads = append(p.uploads, up)
		p.mu.Unlock()
		w.WriteHeader(status)
		if answer != "" {
			_, _ = io.WriteString(w, answer)
		}
	}))
	t.Cleanup(p.server.Close)
	return p
}

// port is the loopback port this stand-in holds, which is the only port a case
// here may point a harness at.
func (p *fakePlugin) port(t *testing.T) string {
	t.Helper()

	u, err := url.Parse(p.server.URL)
	assert.NilError(t, err)
	return u.Port()
}

func (p *fakePlugin) delivered() []recordedUpload {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.uploads
}

// closedPort is a loopback port this process held and gave up, which is the
// only honest way to name one nothing answers on: a number picked by hand could
// be anything by the time the case runs.
func closedPort(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	_, port, err := net.SplitHostPort(ln.Addr().String())
	assert.NilError(t, err)
	assert.NilError(t, ln.Close())
	return port
}

// Whether the session registered the plugin is decided at launch and never said
// again, so the variable the launcher sets beside the flag is the whole of what
// this server has to go on. Anything but the launcher's own value leaves the
// member terminal, where the daemon still reaches it.
func TestNewClaudeCode_TakesTheChannelOnlyFromItsLauncher(t *testing.T) {
	for _, value := range []string{"", "0", "true", "yes", "2"} {
		t.Run(value, func(t *testing.T) {
			assert.Assert(t, newClaudeCode(fakechatEnv(value, "1234")) == nil)
		})
	}

	h := newClaudeCode(fakechatEnv("1", "1234"))
	assert.Assert(t, h != nil)
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)
}

// A launcher that named no port and one that exported an empty one arrive here
// as the same thing, and both mean the port the plugin binds on its own. The
// URL is read rather than called: a host running this suite may well have a
// plugin of its own on that port.
func TestNewClaudeCode_FallsBackToThePortThePluginBindsItself(t *testing.T) {
	for _, tc := range []struct {
		name   string
		getenv func(string) string
	}{
		{"unset", func(name string) string {
			if name == ClaudeChannelEnv {
				return "1"
			}
			return ""
		}},
		{"empty", fakechatEnv("1", "")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newClaudeCode(tc.getenv)
			assert.Equal(t, h.(channelledClaudeCode).channel.url("/"), "http://127.0.0.1:8787/")
		})
	}
}

// The whole of it through [Detect], which is the one caller: a Claude Code
// launched with the plugin attends as a member the daemon never types at, and
// as one the server can ask whether its channel is there.
func TestDetect_TheClaudeChannelIsAskedBeforeItAttends(t *testing.T) {
	plugin := startFakePlugin(t, recordedFakechatPage(t), http.StatusNoContent, "")

	h := Detect("claude-code", fakechatEnv("1", plugin.port(t)))
	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	prober, ok := h.(Prober)
	assert.Assert(t, ok, "%T cannot be asked whether its channel is there", h)
	assert.NilError(t, prober.Probe(t.Context()))
}

// A port nothing holds is the launcher that never started the plugin, or
// started it somewhere else. The refusal names the port, since the port is the
// one thing whoever reads it has to go and fix.
func TestFakechat_ProbeRefusesAPortNothingAnswersOn(t *testing.T) {
	port := closedPort(t)

	err := newClaudeCode(fakechatEnv("1", port)).(Prober).Probe(t.Context())
	assert.ErrorContains(t, err, port)
}

// Something else holding the port is the same failure with a worse ending: a
// channel would post its mentions into whatever that is. Only the plugin's own
// page is taken for the plugin.
func TestFakechat_ProbeRefusesWhateverElseHoldsThePort(t *testing.T) {
	for _, tc := range []struct {
		name string
		page string
	}{
		{"nothing on the path", ""},
		{"another server's page", "<!doctype html>\n<title>grafana</title>\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plugin := startFakePlugin(t, tc.page, http.StatusNoContent, "")
			port := plugin.port(t)

			err := newClaudeCode(fakechatEnv("1", port)).(Prober).Probe(t.Context())
			assert.ErrorContains(t, err, port)
		})
	}
}

// What the plugin is handed, checked against the request the real one took: the
// same method on the same path, carrying the same fields. The values are this
// room's, and the id is this process's own.
func TestFakechat_PostsTheNoticeAsAnUpload(t *testing.T) {
	recorded := recordedFakechatUpload(t)
	plugin := startFakePlugin(t, recordedFakechatPage(t), http.StatusNoContent, "")

	h := Detect("claude-code", fakechatEnv("1", plugin.port(t)))
	text := "[crabswarm chat] new message from team/bob"
	assert.NilError(t, h.Deliver(t.Context(), Notice{
		From: "team/bob",
		Room: "/work/proj",
		Text: text,
	}))

	got := plugin.delivered()
	assert.Equal(t, len(got), 1, "the plugin took %d uploads", len(got))
	assert.Equal(t, got[0].method, recorded.method)
	assert.Equal(t, got[0].path, recorded.path)
	// A set rather than a sequence: the plugin reads the form as a whole, and
	// what matters is that no field was added to it or left out of it.
	assert.DeepEqual(t,
		slices.Sorted(slices.Values(got[0].fields)),
		slices.Sorted(slices.Values(recorded.fields)))
	assert.Equal(t, got[0].text, text)
	assert.Assert(t, uploadID.MatchString(got[0].id), "%q is not an upload id", got[0].id)
}

// Two mentions are two messages, and the plugin tells them apart by the id it
// is posted.
func TestFakechat_GivesEachUploadAnIDOfItsOwn(t *testing.T) {
	plugin := startFakePlugin(t, recordedFakechatPage(t), http.StatusNoContent, "")

	h := Detect("claude-code", fakechatEnv("1", plugin.port(t)))
	assert.NilError(t, h.Deliver(t.Context(), Notice{Text: "first"}))
	assert.NilError(t, h.Deliver(t.Context(), Notice{Text: "second"}))

	got := plugin.delivered()
	assert.Equal(t, len(got), 2, "the plugin took %d uploads", len(got))
	for _, up := range got {
		assert.Assert(t, uploadID.MatchString(up.id), "%q is not an upload id", up.id)
	}
	assert.Assert(t, got[0].id != got[1].id, "both uploads went out as %s", got[0].id)
}

// A plugin that refused says so rather than reporting a delivery that never
// reached the session: the caller leaves the mention outstanding and comes back
// to it.
func TestFakechat_ARefusedUploadIsNotDelivered(t *testing.T) {
	plugin := startFakePlugin(t, recordedFakechatPage(t), http.StatusBadRequest, "missing id")

	err := Detect("claude-code", fakechatEnv("1", plugin.port(t))).
		Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorContains(t, err, "400")
	assert.ErrorContains(t, err, "missing id")
}

// A plugin that took the connection and then said nothing is given up on, so
// one wedged plugin costs the attendance or the delivery rather than the feed
// every delivery runs on.
func TestFakechat_APluginThatNeverAnswersIsGivenUpOn(t *testing.T) {
	// The handler is released by the case rather than by the client giving up:
	// httptest's Close waits for the handlers it is serving to return, and the
	// cleanups run in reverse, so the release happens first.
	release := make(chan struct{})
	blocked := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	t.Cleanup(blocked.Close)
	t.Cleanup(func() { close(release) })

	u, err := url.Parse(blocked.URL)
	assert.NilError(t, err)
	h := fakechat{port: u.Port(), timeout: 50 * time.Millisecond}

	assert.ErrorIs(t, h.Probe(t.Context()), context.DeadlineExceeded)
	assert.ErrorIs(t, h.Deliver(t.Context(), Notice{Text: "hi"}), context.DeadlineExceeded)
}
