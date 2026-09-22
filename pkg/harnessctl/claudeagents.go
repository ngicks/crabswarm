package harnessctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Claude Code has no feed of its own to push a session's state on, but it does
// keep a registry of every session on the host and prints it on request. This
// file is the feed built on that: the listing polled, the session picked out of
// it by id, and the state it shows reported when it changes.
//
// The listing is what Claude Code itself says the session is doing, which is
// the one account that stays right while a background subagent keeps the
// session busy after the main turn ended, or after a turn was interrupted with
// nothing left to fire a hook.

// claudeAgentsInterval is how often the listing is read. Each read spawns a
// Claude Code process, which costs a good part of a second, so the interval
// stays well above that; it sits under the daemon's screen-poll cadence so the
// feed is never the slower of the two.
const claudeAgentsInterval = 2 * time.Second

// The listing's fields, one type per field. They stay strings so a value
// outside the vocabulary below still decodes and is read as the one thing that
// is true of it — that this listing says nothing the feed can map.
type (
	claudeAgentKind       string
	claudeAgentStatus     string
	claudeAgentState      string
	claudeAgentWaitingFor string
)

// kind is on every entry.
const (
	claudeKindInteractive claudeAgentKind = "interactive"
	claudeKindBackground  claudeAgentKind = "background"
)

// status is what the session is doing at this moment, printed only while its
// process is alive.
const (
	claudeStatusBusy    claudeAgentStatus = "busy"
	claudeStatusWaiting claudeAgentStatus = "waiting"
	claudeStatusIdle    claudeAgentStatus = "idle"
)

// state is the session's own account of its progress, printed only for
// background sessions.
const (
	claudeStateWorking claudeAgentState = "working"
	claudeStateBlocked claudeAgentState = "blocked"
	claudeStateDone    claudeAgentState = "done"
	claudeStateFailed  claudeAgentState = "failed"
	claudeStateStopped claudeAgentState = "stopped"
)

// waitingFor names what a waiting session waits on. It accompanies a waiting
// status and is display text for a person, so nothing reads it beyond the
// waiting the status already said.
const (
	claudeWaitingPermission claudeAgentWaitingFor = "permission prompt"
	claudeWaitingInput      claudeAgentWaitingFor = "input needed"
	claudeWaitingSandbox    claudeAgentWaitingFor = "sandbox request"
	claudeWaitingWorker     claudeAgentWaitingFor = "worker request"
	claudeWaitingDialog     claudeAgentWaitingFor = "dialog open"
)

// claudeAgent is one entry of `claude agents --json`, the fields a state is
// read from. e2e/crabswarm/testdata/harness/claude-agents.json is a recorded
// listing, and its README says where the vocabulary above comes from.
type claudeAgent struct {
	SessionID  string                `json:"sessionId"`
	Kind       claudeAgentKind       `json:"kind"`
	Status     claudeAgentStatus     `json:"status"`
	WaitingFor claudeAgentWaitingFor `json:"waitingFor"`
	State      claudeAgentState      `json:"state"`
}

// claudeAgentsLister answers the current listing. The harness runs the real
// command; a test hands in whatever listing it wants the feed to see.
type claudeAgentsLister func(ctx context.Context) ([]claudeAgent, error)

// listClaudeAgents runs `claude agents --json`, resolving claude on PATH and in
// the environment this server inherited from its harness, which is what points
// the command at the same registry the harness writes to.
func listClaudeAgents(ctx context.Context) ([]claudeAgent, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "claude", "agents", "--json")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running claude agents --json: %w: %s",
			err, bytes.TrimSpace(stderr.Bytes()))
	}
	var agents []claudeAgent
	if err := json.Unmarshal(stdout.Bytes(), &agents); err != nil {
		return nil, fmt.Errorf("decoding claude agents --json: %w", err)
	}
	return agents, nil
}

// claudeAgentHarnessState maps one entry onto the state the room records,
// reporting whether the entry said anything it could map.
//
// status is the live answer and is read first: it says what the session is
// doing at this moment, and waiting is waiting whatever the entry names as the
// reason. An entry with no status is a session whose process the listing could
// not see, and its state is the session's own account of its progress, which is
// the fallback. Anything outside the documented vocabulary is no verdict rather
// than a guess.
func claudeAgentHarnessState(a claudeAgent) (chatv1.HarnessState, bool) {
	switch a.Status {
	case claudeStatusBusy:
		return chatv1.HarnessState_HARNESS_STATE_WORKING, true
	case claudeStatusWaiting:
		return chatv1.HarnessState_HARNESS_STATE_WAITING, true
	case claudeStatusIdle:
		return chatv1.HarnessState_HARNESS_STATE_DONE, true
	case "":
	default:
		return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, false
	}
	switch a.State {
	case claudeStateWorking:
		return chatv1.HarnessState_HARNESS_STATE_WORKING, true
	case claudeStateBlocked:
		return chatv1.HarnessState_HARNESS_STATE_WAITING, true
	case claudeStateDone, claudeStateFailed, claudeStateStopped:
		return chatv1.HarnessState_HARNESS_STATE_DONE, true
	}
	return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, false
}

// errClaudeSessionUnlisted is the failure of a listing that came back without
// this session in it.
var errClaudeSessionUnlisted = errors.New("the agents listing does not carry this session")

// Watch follows the listing for as long as ctx runs, reporting the session's
// state each time it changes. A harness that knows no session id has nothing to
// follow and returns at once.
//
// A read that failed, or that came back without the session, reports nothing:
// the member keeps its last state, which is the answer that interrupts nobody
// mid-turn, and the next read replaces it. The first failure of a run is
// logged and the rest are not, since a line per tick for the rest of the
// session would bury everything else on the harness's stderr.
func (c claudeCode) Watch(ctx context.Context, report func(chatv1.HarnessState)) {
	if c.sessionID == "" {
		return
	}
	last := chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED
	failures := 0
	poll := func() {
		state, err := c.read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			failures++
			if failures == 1 {
				c.logger.Warn("reading the claude agents listing failed; trying again each tick",
					"session", c.sessionID, "error", err)
			}
			return
		}
		failures = 0
		if state != last {
			last = state
			report(state)
		}
	}
	// Once before the first tick: the member would otherwise have no state at
	// all for the first interval of every attendance.
	poll()
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			poll()
		}
	}
}

// read runs one listing and picks this session's state out of it. The read is
// bounded by the interval: a claude that hung would otherwise hold the feed for
// the rest of the session.
func (c claudeCode) read(ctx context.Context) (chatv1.HarnessState, error) {
	ctx, cancel := context.WithTimeout(ctx, c.interval)
	defer cancel()
	agents, err := c.list(ctx)
	if err != nil {
		return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, err
	}
	for _, a := range agents {
		if a.SessionID != c.sessionID {
			continue
		}
		state, ok := claudeAgentHarnessState(a)
		if !ok {
			return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, fmt.Errorf(
				"the agents listing shows this session with status %q and state %q, which map to nothing",
				a.Status,
				a.State,
			)
		}
		return state, nil
	}
	return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED, errClaudeSessionUnlisted
}
