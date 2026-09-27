package harnessctl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// OpenCode has no channel a server beside it can push a notice through on its
// own. Its TUI carries a plugin that can prompt the session the person is
// driving, so the notice travels to the plugin: the plugin of a TUI attached to
// a shared `opencode serve` holds a notice stream open on the crabswarm MCP
// server that serves it, and the server writes every notice there. Detect never
// hands one out, since an MCP session does not say which TUI it is for; the
// server attaches the channel to the member whose token opened the stream. An
// OpenCode that reached the server any other way is a terminal harness.

// OpenCodeNoticeContentType is what a notice stream is answered as: one JSON
// object per line, each one notice.
const OpenCodeNoticeContentType = "application/x-ndjson"

// openCodeWriteTimeout bounds one notice written to a stream. The plugin reads
// the stream as it arrives, so a write only waits when the connection's window
// is full; past this the plugin is not reading, and the feed the delivery runs
// on must not be held any longer than that.
const openCodeWriteTimeout = 5 * time.Second

// OpenCodeNotices is the channel one OpenCode member is reached through: every
// notice is written as one JSON line to each stream attached to it, and the
// plugin holding a stream prompts the session its TUI shows with it.
//
// Its zero value is not usable; make one with [NewOpenCodeNotices].
type OpenCodeNotices struct {
	mu      sync.Mutex
	streams map[*OpenCodeStream]struct{}
}

var _ Harness = (*OpenCodeNotices)(nil)

// NewOpenCodeNotices returns a channel with no stream attached yet.
func NewOpenCodeNotices() *OpenCodeNotices {
	return &OpenCodeNotices{streams: map[*OpenCodeStream]struct{}{}}
}

func (*OpenCodeNotices) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_OPENCODE }

func (*OpenCodeNotices) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// openCodeNotice is one line of a notice stream, as the plugin reads it. From
// is empty on a notice about the room rather than about one message.
type openCodeNotice struct {
	Content string `json:"content"`
	From    string `json:"from"`
}

// errNoOpenCodeStream is what a delivery is refused with while no plugin holds
// a stream open.
var errNoOpenCodeStream = errors.New(
	"no opencode notice stream is open to deliver a mention through",
)

// Deliver writes n to every stream attached, and reports an error when none of
// them took it: the mention is still unread, and the caller comes back to it
// rather than counting it as delivered to nobody.
func (o *OpenCodeNotices) Deliver(_ context.Context, n Notice) error {
	line, err := json.Marshal(openCodeNotice{Content: n.Text, From: n.From})
	if err != nil {
		return fmt.Errorf("encoding a chat notice for an opencode notice stream: %w", err)
	}
	line = append(line, '\n')

	o.mu.Lock()
	streams := make([]*OpenCodeStream, 0, len(o.streams))
	for s := range o.streams {
		streams = append(streams, s)
	}
	o.mu.Unlock()
	if len(streams) == 0 {
		return errNoOpenCodeStream
	}
	var errs []error
	for _, s := range streams {
		if err := s.write(line); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == len(streams) {
		return errors.Join(errs...)
	}
	return nil
}

// OpenCodeStream is one notice stream: the response to one request a plugin
// holds open.
type OpenCodeStream struct {
	notices *OpenCodeNotices
	w       http.ResponseWriter
	rc      *http.ResponseController
	// broken is closed once a notice could not be written.
	broken    chan struct{}
	breakOnce sync.Once

	// mu serializes what is written to w, which a delivery does from the
	// member's own goroutine rather than from the handler holding the request.
	mu sync.Mutex
	// started says the response header went out, with [OpenCodeStream.Open] or
	// with the first notice, whichever came first.
	started bool
	// detached says the handler is done with w, which may not be touched after.
	detached bool
}

// Attach makes w a stream of o: every notice delivered until
// [OpenCodeStream.Detach] is written to it. Nothing is written yet. A caller
// attaches the stream before it lets anything deliver through o, so the first
// notice finds it, and answers the request with [OpenCodeStream.Open] after.
func (o *OpenCodeNotices) Attach(w http.ResponseWriter) *OpenCodeStream {
	w.Header().Set("Content-Type", OpenCodeNoticeContentType)
	s := &OpenCodeStream{
		notices: o,
		w:       w,
		rc:      http.NewResponseController(w),
		broken:  make(chan struct{}),
	}
	o.mu.Lock()
	o.streams[s] = struct{}{}
	o.mu.Unlock()
	return s
}

// Open sends the response header now, unless a notice already has, so the
// plugin learns the stream is up before anything is said on it.
func (s *OpenCodeStream) Open() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.detached || s.started {
		return nil
	}
	s.started = true
	s.w.WriteHeader(http.StatusOK)
	if err := s.rc.Flush(); err != nil {
		return fmt.Errorf("opening an opencode notice stream: %w", err)
	}
	return nil
}

// Broken is closed once a notice could not be written to the stream. The
// stream is worth nothing after that, and the handler holding the request may
// end it: the plugin opens another.
func (s *OpenCodeStream) Broken() <-chan struct{} {
	return s.broken
}

// Detach takes the stream off its channel. Nothing is written to it once
// Detach has returned, so the handler holding the request may return then.
func (s *OpenCodeStream) Detach() {
	s.notices.mu.Lock()
	delete(s.notices.streams, s)
	s.notices.mu.Unlock()
	s.mu.Lock()
	s.detached = true
	s.mu.Unlock()
}

// write puts one line on the stream and flushes it to the plugin.
func (s *OpenCodeStream) write(line []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.detached {
		return errors.New("the opencode notice stream has closed")
	}
	s.started = true
	// A writer that cannot take a deadline is written without one: the
	// connection the plugin holds is one that can.
	_ = s.rc.SetWriteDeadline(time.Now().Add(openCodeWriteTimeout))
	defer func() { _ = s.rc.SetWriteDeadline(time.Time{}) }()
	_, err := s.w.Write(line)
	if err == nil {
		err = s.rc.Flush()
	}
	if err != nil {
		s.breakOnce.Do(func() { close(s.broken) })
		return fmt.Errorf("writing a chat notice to an opencode notice stream: %w", err)
	}
	return nil
}
