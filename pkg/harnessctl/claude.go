package harnessctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The Claude Code channel contract, as the official fakechat plugin serves it.
//
// A channel is something Claude Code lets push a turn into a running session. A
// session launched with `--channels plugin:fakechat@claude-plugins-official`
// runs that plugin's loopback HTTP server beside itself, and everything the
// server is posted reaches the model as a <channel> element it starts a turn on
// when it is idle. This process is not the channel; it is only the sender.
const (
	// ClaudeChannelEnv is set to "1" by whoever launched Claude Code with the
	// plugin registered as a channel.
	//
	// Registration is decided at launch and nothing in the MCP session reports
	// it, so this server cannot tell a session carrying the plugin from an
	// ordinary one. The launcher sets this beside the flag, which makes the
	// variable the only honest answer available: without it the uploads would
	// go somewhere no model reads and the member would wait on a delivery
	// nobody could make.
	ClaudeChannelEnv = "CRABSWARM_CLAUDE_CHANNEL"

	// FakechatPortEnv names the loopback port the plugin's server listens on.
	// The launcher sets it beside the channel flag and the plugin reads the
	// same variable out of the launch environment, which is what puts the two
	// ends on one port.
	FakechatPortEnv = "FAKECHAT_PORT"
)

// fakechatDefaultPort is where the plugin listens when its launcher named no
// port, straight out of the plugin's own `Number(process.env.FAKECHAT_PORT ??
// 8787)`.
//
// An unset variable and an empty one are one case here, and not only because
// getenv cannot tell them apart. This server's MCP declaration forwards the
// port as "${FAKECHAT_PORT}", which arrives as the empty string when the
// launcher named none, while the plugin inherits the launch environment
// unexpanded and sees the variable genuinely unset — so the empty string this
// end reads describes the same launch the plugin answers on 8787.
//
// A launcher that exports an empty FAKECHAT_PORT on purpose is a different
// story: the plugin's Number("") is 0, which binds a random port, and the probe
// below then fails and the member never attends. That is the loud failure it
// should be, rather than a channel quietly posting at whatever else holds 8787.
const fakechatDefaultPort = "8787"

// fakechatTimeout bounds one call on the plugin's server. A probe holds up the
// attendance it gates, and a delivery runs on the goroutine that reads the
// room's feed, so a plugin that took the connection and then said nothing costs
// that one call rather than everything the member would have heard after it.
const fakechatTimeout = 5 * time.Second

// fakechatTitle is what the plugin's page calls itself, and the whole of what a
// probe recognises it by. Loopback ports get reused, and a 200 from whatever
// else took this one is not a channel.
const fakechatTitle = "<title>fakechat</title>"

// ClaudeChannelEnabled reports whether this process was launched beside a
// Claude Code session that registered the fakechat plugin as a channel.
func ClaudeChannelEnabled(getenv func(string) string) bool {
	return getenv(ClaudeChannelEnv) == "1"
}

// newClaudeCode builds the fakechat channel, or answers nil for a session that
// registered no plugin and so has to be woken through its terminal.
func newClaudeCode(getenv func(string) string) Harness {
	if !ClaudeChannelEnabled(getenv) {
		return nil
	}
	port := getenv(FakechatPortEnv)
	if port == "" {
		port = fakechatDefaultPort
	}
	return fakechat{port: port, timeout: fakechatTimeout}
}

// fakechat delivers a notice by posting it to the loopback server the fakechat
// plugin runs beside its MCP server. The plugin turns each upload into a
// channel event of the Claude Code session that registered it, so the session
// this process was launched beside is reached without this process holding a
// channel of its own.
type fakechat struct {
	port    string
	timeout time.Duration
}

func (fakechat) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_CLAUDE_CODE }

func (fakechat) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// url is one endpoint of the plugin's server. Loopback and nothing else: the
// plugin binds 127.0.0.1, and a channel that could be pointed at another host
// would be a way to post into a session nobody here is running.
func (f fakechat) url(path string) string {
	return "http://127.0.0.1:" + f.port + path
}

// Probe reports whether the plugin is listening where this channel would post.
//
// Nothing in the MCP session says the plugin is there. The channel variable
// says the launcher meant to register one and the port says where, and a
// launcher that got either wrong leaves a member attending as native — which
// stops the daemon typing at it — with every mention it is handed dropped.
// Asking the page first turns that into a refusal to attend, which somebody
// reads.
func (f fakechat) Probe(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url("/"), nil)
	if err != nil {
		return fmt.Errorf("building a request for the fakechat plugin on port %s: %w",
			f.port, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("reaching the fakechat plugin on port %s: %w", f.port, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("port %s does not serve the fakechat plugin: it answered %s",
			f.port, resp.Status)
	}
	// Bounded: the page is a few kilobytes of the plugin's own HTML, and
	// whatever else may be holding the port is not this process's to trust with
	// how much it sends.
	page, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return fmt.Errorf("reading the page on port %s: %w", f.port, err)
	}
	if !bytes.Contains(page, []byte(fakechatTitle)) {
		return fmt.Errorf("port %s answers, but the page it serves is not the fakechat plugin's",
			f.port)
	}
	return nil
}

// fakechatUploads numbers the uploads this process makes. The plugin takes the
// id as the message it delivers under and echoes it back in the event's meta,
// so two mentions sharing an id would be two the transcript cannot tell apart.
var fakechatUploads atomic.Uint64

// Deliver posts the notice to the plugin as one upload.
//
// Two fields and no more. [Notice.From] and [Notice.Room] stay behind: the
// plugin writes the event's meta itself — which chat it came from, the id, the
// user and the time — and the line the room worded already says who wrote and
// which tool answers them, so a third field would have nothing to carry.
//
// Anything but a 2xx is an error, as it is for the opencode relay: the mention
// is still unread, and the caller comes back to it rather than counting it as
// delivered to nobody.
func (f fakechat) Deliver(ctx context.Context, n Notice) error {
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	id := "crabswarm-" + strconv.FormatUint(fakechatUploads.Add(1), 10)
	if err := errors.Join(
		form.WriteField("id", id),
		form.WriteField("text", n.Text),
		form.Close(),
	); err != nil {
		return fmt.Errorf("encoding the upload %s: %w", id, err)
	}

	ctx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.url("/upload"), &body)
	if err != nil {
		return fmt.Errorf("building an upload for the fakechat plugin on port %s: %w",
			f.port, err)
	}
	req.Header.Set("Content-Type", form.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting an upload to the fakechat plugin on port %s: %w",
			f.port, err)
	}
	defer func() { _ = resp.Body.Close() }()
	// Bounded for the same reason the probe bounds the page it reads.
	answer, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("the fakechat plugin on port %s refused the upload %s: %s: %s",
			f.port, id, resp.Status, bytes.TrimSpace(answer))
	}
	return nil
}
