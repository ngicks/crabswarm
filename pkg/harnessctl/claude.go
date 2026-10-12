package harnessctl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// The Claude Code channel contracts.
//
// A channel is something Claude Code lets push a turn into a running session,
// and a session can be reached through one of two:
//
//   - fakechat: a session launched with
//     `--channels plugin:fakechat@claude-plugins-official` runs that plugin's
//     loopback HTTP server beside itself, and everything the server is posted
//     reaches the model as a <channel> element it starts a turn on when it is
//     idle. This process is not the channel; it is only the sender.
//   - direct: a session launched with this server itself registered as a
//     channel listens for [claudeChannelMethod] notifications on the MCP
//     session it spawned the server on. The server declares
//     [ClaudeChannelCapability] and pushes the notice down that session.
//
// Registration is decided at launch and nothing in the MCP session reports it,
// so this server cannot tell a registered session from an ordinary one. The
// launcher declares which channel it registered: in the environment for a
// server it spawned per session (see [ClaudeChannelFromEnv]), and in a request
// header for a session of a shared server (see [ParseClaudeChannel]). Without
// the declaration the notices would go somewhere no model reads and the member
// would wait on a delivery nobody could make.
const (
	// ClaudeChannelEnv declares the channel of the Claude Code session that
	// spawned this server: "1" or "fakechat" for the fakechat plugin on
	// [FakechatPortEnv], "direct" for this server registered as the channel.
	ClaudeChannelEnv = "CRABSWARM_CLAUDE_CHANNEL"

	// ClaudeChannelCapability is the experimental capability a direct channel
	// server declares. Declaring it is what makes Claude Code listen.
	ClaudeChannelCapability = "claude/channel"

	// claudeChannelMethod carries one direct channel event.
	claudeChannelMethod = "notifications/claude/channel"

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

// ClaudeSessionEnv names this bridge's Claude Code session, as Claude Code sets
// it in the environment of every MCP server it spawns. It is the id the agents
// listing prints as sessionId at spawn time; the feed falls back to it when no
// entry carries the pid of the process that spawned this server.
const ClaudeSessionEnv = "CLAUDE_CODE_SESSION_ID"

// ClaudeChannelKind names which channel a Claude Code session registered.
type ClaudeChannelKind int

const (
	// ClaudeChannelNone is a session that registered no channel. It is woken
	// through its terminal.
	ClaudeChannelNone ClaudeChannelKind = iota
	// ClaudeChannelFakechat is the fakechat plugin's loopback server.
	ClaudeChannelFakechat
	// ClaudeChannelDirect is this server registered as the channel itself.
	ClaudeChannelDirect
)

// ClaudeChannel is the channel a Claude Code session registered, and where it
// listens. Port is set for [ClaudeChannelFakechat] alone.
type ClaudeChannel struct {
	Kind ClaudeChannelKind
	Port string
}

// The spellings of a [ClaudeChannel], shared by [ParseClaudeChannel] and
// [ClaudeChannelFromEnv].
const (
	claudeChannelFakechat = "fakechat"
	claudeChannelDirect   = "direct"
	fakechatPortParam     = "port"
)

// ParseClaudeChannel reads a channel declaration in the form a request header
// carries it: "direct", or "fakechat;port=N". The empty string is
// [ClaudeChannelNone].
//
// The fakechat port is mandatory here, unlike in [ClaudeChannelFromEnv]. The
// declaration comes from one session among the many a shared server serves,
// and two sessions defaulting to one port would post each other's notices into
// whichever session holds it.
func ParseClaudeChannel(s string) (ClaudeChannel, error) {
	kind, params, _ := strings.Cut(strings.TrimSpace(s), ";")
	switch strings.TrimSpace(kind) {
	case "":
		if params != "" {
			return ClaudeChannel{}, fmt.Errorf("channel %q names parameters but no channel", s)
		}
		return ClaudeChannel{}, nil
	case claudeChannelDirect:
		if strings.TrimSpace(params) != "" {
			return ClaudeChannel{}, fmt.Errorf("channel %q: direct takes no parameters", s)
		}
		return ClaudeChannel{Kind: ClaudeChannelDirect}, nil
	case claudeChannelFakechat:
		var port string
		for param := range strings.SplitSeq(params, ";") {
			key, value, _ := strings.Cut(param, "=")
			switch key = strings.TrimSpace(key); key {
			case "":
			case fakechatPortParam:
				port = strings.TrimSpace(value)
			default:
				return ClaudeChannel{}, fmt.Errorf("channel %q: unknown parameter %q", s, key)
			}
		}
		if port == "" {
			return ClaudeChannel{}, fmt.Errorf(
				"channel %q: fakechat needs the plugin's port, as fakechat;port=N", s)
		}
		if err := validPort(port); err != nil {
			return ClaudeChannel{}, fmt.Errorf("channel %q: %w", s, err)
		}
		return ClaudeChannel{Kind: ClaudeChannelFakechat, Port: port}, nil
	default:
		return ClaudeChannel{}, fmt.Errorf(
			"unknown channel %q: want %s or %s;%s=N",
			s, claudeChannelDirect, claudeChannelFakechat, fakechatPortParam)
	}
}

// validPort refuses anything but a TCP port number. The port lands in a URL, and
// a value that could carry a host or a path would point the channel at
// something other than the loopback plugin.
func validPort(port string) error {
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil || n == 0 {
		return fmt.Errorf("port %q is not a TCP port number", port)
	}
	return nil
}

// ClaudeChannelFromEnv reads the channel the launcher declared in the
// environment of a server it spawned for one Claude Code session.
//
// "1" and "fakechat" name the plugin on [FakechatPortEnv], which defaults to
// the plugin's own default; "direct" names this server. Anything else set is
// an error rather than no channel, since a misspelled declaration would leave
// the member typed at while its launcher believes otherwise.
func ClaudeChannelFromEnv(getenv func(string) string) (ClaudeChannel, error) {
	switch v := getenv(ClaudeChannelEnv); v {
	case "":
		return ClaudeChannel{}, nil
	case "1", claudeChannelFakechat:
		port := getenv(FakechatPortEnv)
		if port == "" {
			port = fakechatDefaultPort
		}
		if err := validPort(port); err != nil {
			return ClaudeChannel{}, fmt.Errorf("%s: %w", FakechatPortEnv, err)
		}
		return ClaudeChannel{Kind: ClaudeChannelFakechat, Port: port}, nil
	case claudeChannelDirect:
		return ClaudeChannel{Kind: ClaudeChannelDirect}, nil
	default:
		return ClaudeChannel{}, fmt.Errorf("%s=%q: want 1, %s or %s",
			ClaudeChannelEnv, v, claudeChannelFakechat, claudeChannelDirect)
	}
}

// newClaudeCode builds the Claude Code harness: the channel the session
// registered, and the state feed, when the environment names the session to
// follow. It answers nil when it has neither, and the member is then a terminal
// one with nothing Claude-specific about it.
//
// The two halves are independent. The channel needs the launcher's
// declaration, and a session that registered none has to be woken through its
// terminal. The feed needs only the session id, which Claude Code sets itself,
// so a session launched without the channel still reports its state.
//
// A direct channel needs session, the MCP session it pushes through; with none
// it is no channel at all, since one with nowhere to push would claim a
// delivery it could never make.
//
// A member with the fakechat channel comes back as a [Prober], since the
// plugin is a server somewhere other than this session and can be asked
// whether it is there. A direct channel is the session itself and is asked
// nothing, which is why the two are two types rather than one with a probe
// that always says yes.
func newClaudeCode(getenv func(string) string, ch ClaudeChannel, session Session) Harness {
	c := claudeCode{
		sessionID: getenv(ClaudeSessionEnv),
		pid:       os.Getppid(),
		list:      listClaudeAgents,
		interval:  claudeAgentsInterval,
		logger:    slog.Default(),
	}
	switch ch.Kind {
	case ClaudeChannelFakechat:
		f := &fakechat{port: ch.Port, timeout: fakechatTimeout}
		c.channel = f
		return channelledClaudeCode{claudeCode: c, plugin: f}
	case ClaudeChannelDirect:
		if session != nil {
			c.channel = directChannel{session: session}
			return c
		}
	}
	if c.sessionID == "" {
		return nil
	}
	return c
}

// claudeChannel is what a Claude Code session is pushed a notice through.
type claudeChannel interface {
	Deliver(ctx context.Context, n Notice) error
}

// claudeCode delivers a notice through the channel the session registered,
// when it registered one, and reports what the session is doing off the agents
// listing.
type claudeCode struct {
	// channel is nil for a session that registered none.
	channel claudeChannel

	// sessionID is the session the feed follows, empty for a feed that has
	// nothing to follow.
	sessionID string
	// pid is the Claude Code process that spawned this server. The feed
	// follows the process first: /clear and /resume switch the session id
	// inside the same process, while sessionID keeps the id this server was
	// started under.
	pid      int
	list     claudeAgentsLister
	interval time.Duration
	logger   *slog.Logger
}

func (claudeCode) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_CLAUDE_CODE }

func (c claudeCode) Nudge() chatv1.NudgeDelivery {
	if c.channel == nil {
		return chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL
	}
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// Deliver posts the notice to the plugin. A session that registered none
// refuses, the way every terminal harness does.
func (c claudeCode) Deliver(ctx context.Context, n Notice) error {
	if c.channel == nil {
		return errNoChannel
	}
	return c.channel.Deliver(ctx, n)
}

// channelledClaudeCode is a [claudeCode] whose session registered the plugin,
// which is the one that can be asked whether its channel is there.
type channelledClaudeCode struct {
	claudeCode
	plugin *fakechat
}

// Probe asks the plugin's server, see [fakechat.Probe].
func (c channelledClaudeCode) Probe(ctx context.Context) error {
	return c.plugin.Probe(ctx)
}

// directChannel pushes a notice as a channel event of the MCP session the
// server already holds with Claude Code. There is no second connection: the
// harness spawned this server, and the stream it spawned it on is the channel.
type directChannel struct {
	session Session
}

// The meta keys one direct channel event carries. Claude Code renders them as
// attributes of the <channel> element the model reads, and accepts
// [A-Za-z0-9_] in a key.
const (
	claudeMetaFrom = "from"
	claudeMetaRoom = "room"
)

// claudeChannelParams is one direct channel event: the line the model reads,
// and the keys Claude Code renders around it.
type claudeChannelParams struct {
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta,omitzero"`
}

// Deliver pushes the notice into the session as a channel event.
func (d directChannel) Deliver(ctx context.Context, n Notice) error {
	params := claudeChannelParams{Content: n.Text}
	// Only the keys there is something to say with: a notice about how much
	// waits names no sender, and an empty value would render as from="" in the
	// element the model reads.
	put := func(key, value string) {
		if value == "" {
			return
		}
		if params.Meta == nil {
			params.Meta = map[string]string{}
		}
		params.Meta[key] = value
	}
	put(claudeMetaFrom, n.From)
	put(claudeMetaRoom, n.Room)
	if err := d.session.Notify(ctx, claudeChannelMethod, params); err != nil {
		return fmt.Errorf("pushing a %s notification: %w", claudeChannelMethod, err)
	}
	return nil
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
// Anything but a 2xx is an error: the mention is still unread, and the caller
// comes back to it rather than counting it as delivered to nobody.
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
