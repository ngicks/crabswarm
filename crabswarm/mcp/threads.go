package mcp

import (
	"crypto/rand"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// Which of a member's sessions it is reached through. A member of an HTTP
// server may hold several sessions at once, one per thread its TUIs opened,
// and a mention goes to the one it was last seen in. Everything here runs
// with [HTTPServer.mu] held unless it says otherwise, and none of it waits on
// anything: it is called from the requests of every session and from the app
// server's notification handler.

// nextSeen numbers one moment a member is seen somewhere.
func (s *HTTPServer) nextSeen() uint64 {
	s.seen++
	return s.seen
}

// seenAt is the last moment the member was seen in b: in the session itself, or
// on the thread it serves.
func (s *HTTPServer) seenAt(b *binding) uint64 {
	if b.thread == "" {
		return b.seen
	}
	return max(b.seen, s.turns[b.thread])
}

// threadBound reports whether some session serves thread.
func (s *HTTPServer) threadBound(thread string) bool {
	for _, b := range s.sessions {
		if b.thread == thread {
			return true
		}
	}
	return false
}

// retarget points a's member at the session it was last seen in, among those
// whose harness is known, and tells every harness scoped to a that the
// thread it follows may have moved.
//
// A member whose OpenCode TUI holds a notice stream open is reached through
// that stream instead: it is what the TUI's plugin opened to be told things,
// however many MCP sessions carry the same token besides.
func (s *HTTPServer) retarget(a *attendee) {
	var last *binding
	var at uint64
	for _, b := range s.sessions {
		if b.attendee != a || b.harness == nil {
			continue
		}
		if seen := s.seenAt(b); last == nil || seen > at {
			last, at = b, seen
		}
	}
	for _, sc := range a.scoped {
		poke(sc.threads.changed)
	}
	if a.streams > 0 {
		a.member.retarget(a.openCode)
		return
	}
	if last == nil {
		return
	}
	harness := last.harness
	if base, ok := harness.(harnessctl.ThreadScoper); ok {
		harness = s.scope(a, base)
	}
	a.member.retarget(harness)
}

// scope is the harness a is reached through on the harness server base stands
// for, made the first time a is seen there.
func (s *HTTPServer) scope(a *attendee, base harnessctl.ThreadScoper) harnessctl.Harness {
	if sc, ok := a.scoped[base]; ok {
		return sc.harness
	}
	threads := &memberThreads{
		server:   s,
		attendee: a,
		base:     base,
		changed:  make(chan struct{}, 1),
	}
	sc := scopedHarness{harness: base.ScopeTo(threads), threads: threads}
	a.scoped[base] = sc
	return sc.harness
}

// learn records that b serves thread, where a call named one, and reports
// whether that changed what b serves. authority says the call was addressed to
// thread by the harness server itself, which is the one account of a session's
// thread that overrides what an earlier call said. A call a turn made names the
// thread that turn ran on, and a session keeps the first of those.
func (b *binding) learn(thread string, authority bool) bool {
	if thread == "" || b.thread == thread || (b.thread != "" && !authority) {
		return false
	}
	b.thread = thread
	return true
}

// toolCalled is [callWatcher.toolCalled]: the session serves the thread the
// call named, if it had not learned one yet.
func (s *HTTPServer) toolCalled(session *mcpsdk.ServerSession, thread string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.sessions[session]
	if b == nil || !b.learn(thread, false) || b.attendee == nil {
		return
	}
	s.retarget(b.attendee)
}

// focused is [callWatcher.focused]: the member was seen in session just now.
func (s *HTTPServer) focused(session *mcpsdk.ServerSession, thread string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.sessions[session]
	if b == nil {
		return
	}
	b.learn(thread, true)
	b.seen = s.nextSeen()
	if b.attendee != nil {
		s.retarget(b.attendee)
	}
}

// pinged is [callWatcher.pinged]: session serves the thread the ping was sent
// to. A ping this server sent names that thread by its nonce too, for a
// harness server that sent no metadata with it.
func (s *HTTPServer) pinged(session *mcpsdk.ServerSession, thread, nonce string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := s.sessions[session]
	if b == nil {
		return
	}
	if thread == "" {
		thread = s.pings[nonce]
	}
	if b.learn(thread, true) && b.attendee != nil {
		s.retarget(b.attendee)
	}
}

// turnStarted records a turn starting on thread, which is a member seen on it.
// It is handed every turn the app server connection reports, from that
// connection's notification handler.
func (s *HTTPServer) turnStarted(thread string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.turns[thread] = s.nextSeen()
	retargeted := map[*attendee]bool{}
	for _, b := range s.sessions {
		if b.thread != thread || b.attendee == nil || retargeted[b.attendee] {
			continue
		}
		retargeted[b.attendee] = true
		s.retarget(b.attendee)
	}
}

// memberThreads is one member's threads on one harness server, as the harness
// scoped to them asks about them. It is the only part of the book a harness
// reaches, and every method takes the server's lock for itself.
type memberThreads struct {
	server   *HTTPServer
	attendee *attendee
	// base is the harness server's unscoped harness: a session whose handshake
	// named it is one of the member's sessions there.
	base harnessctl.ThreadScoper
	// changed is signalled whenever the member is seen somewhere new.
	changed chan struct{}
}

var _ harnessctl.AgentThreads = (*memberThreads)(nil)

// serves reports whether b is a session of the member on the harness server.
func (t *memberThreads) serves(b *binding) bool {
	return b.attendee == t.attendee && b.harness == harnessctl.Harness(t.base)
}

func (t *memberThreads) Target() (string, bool) {
	s := t.server
	s.mu.Lock()
	defer s.mu.Unlock()
	var thread string
	var at uint64
	for _, b := range s.sessions {
		if !t.serves(b) || b.thread == "" {
			continue
		}
		if seen := s.seenAt(b); thread == "" || seen > at {
			thread, at = b.thread, seen
		}
	}
	return thread, thread != ""
}

func (t *memberThreads) Unbound() bool {
	s := t.server
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.sessions {
		if t.serves(b) && b.thread == "" {
			return true
		}
	}
	return false
}

func (t *memberThreads) Bound(thread string) bool {
	s := t.server
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.threadBound(thread)
}

func (t *memberThreads) Changed() <-chan struct{} {
	return t.changed
}

// Ping is the reserved ping call, carrying a nonce that names thread for as
// long as the ping is in flight.
func (t *memberThreads) Ping(thread string) (harnessctl.ToolCall, func()) {
	s := t.server
	nonce := rand.Text()
	s.mu.Lock()
	s.pings[nonce] = thread
	s.mu.Unlock()
	call := harnessctl.ToolCall{
		Server:    serverName,
		Tool:      pingTool,
		Arguments: map[string]any{pingNonceArg: nonce},
	}
	return call, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.pings, nonce)
	}
}

// poke leaves one signal on ch unless one is already waiting there.
func poke(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}
