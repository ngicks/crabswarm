package crabswarm_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// Claude Code pushes nothing about what a session is doing, but it does keep a
// registry of every session on the host and prints it on request, so the
// `crabswarm mcp` a Claude Code spawns reads `claude agents --json` every couple
// of seconds and reports the entry whose sessionId is the one Claude Code set in
// its environment.
//
// The suite cannot run Claude Code, so what stands in for it here is a shim
// named `claude` on the bridge's PATH. It prints one of the listings this file
// plants beside it, and a pointer file says which — rewriting that file is how a
// case moves the session from one status to the next without restarting
// anything.
//
// The listings are the recorded one with a single field changed, so what the
// feed decodes keeps the shape the real command printed.
// testdata/harness/README.md records how that capture was taken.

// claudeAgentsShimScript answers the one invocation the feed makes and refuses
// everything else in words that reach the bridge's stderr, so a feed asking for
// something this shim was never written for says so rather than looking like a
// listing that simply never named the session.
const claudeAgentsShimScript = `#!/bin/sh
d=$(dirname "$0")
if [ "$1" = "agents" ] && [ "$2" = "--json" ]; then
	cat "$d/agents-$(cat "$d/showing").json"
	exit 0
fi
echo "stub claude: unsupported invocation: $*" >&2
exit 1
`

// claudeAgentsShim is the planted `claude` together with the session its
// listings call live, which is the session a bridge has to be pointed at to see
// anything at all.
type claudeAgentsShim struct {
	dir       string
	sessionID string
}

// startClaudeAgentsShim plants the shim and the listings it can print, pointed
// at the busy one.
func startClaudeAgentsShim(t *testing.T) *claudeAgentsShim {
	t.Helper()

	shim := &claudeAgentsShim{dir: t.TempDir()}
	for _, listing := range []struct{ name, status, waitingFor string }{
		{"busy", "busy", ""},
		{"idle", "idle", ""},
		// A waiting session names what it is waiting on. The reason is display
		// text rather than a state of its own, so it rides along here only
		// because the real listing carries it beside the status.
		{"waiting", "waiting", "dialog open"},
	} {
		rendered, session := claudeAgentsListing(t, listing.status, listing.waitingFor)
		writeFile(t, filepath.Join(shim.dir, "agents-"+listing.name+".json"), rendered)
		shim.sessionID = session
	}

	path := filepath.Join(shim.dir, "claude")
	writeFile(t, path, claudeAgentsShimScript)
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod the claude shim: %v", err)
	}
	shim.show(t, "busy")
	return shim
}

// show switches the listing the shim prints from here on.
//
// The name is written beside the pointer and renamed onto it, so a read never
// lands on half a name: the feed runs the shim on a clock of its own, with no
// regard for what the case is doing.
func (s *claudeAgentsShim) show(t *testing.T, listing string) {
	t.Helper()

	staged := filepath.Join(s.dir, ".showing")
	writeFile(t, staged, listing)
	if err := os.Rename(staged, filepath.Join(s.dir, "showing")); err != nil {
		t.Fatalf("point the claude shim at the %s listing: %v", listing, err)
	}
}

// claudeAgentsListing renders the recorded listing with the live session's
// status replaced, and answers with that session's id.
//
// One field moves and nothing else does: the finished session beside it, the
// keys each entry carries and the ids are all as the capture recorded them, so
// what the feed decodes here is what `claude agents --json` printed. The live
// session is the one the capture gives a status — a session whose process the
// listing cannot see is printed without one, whatever it is doing.
//
// waitingFor is left out entirely for a status that names no reason.
func claudeAgentsListing(t *testing.T, status, waitingFor string) (listing, sessionID string) {
	t.Helper()

	var entries []map[string]any
	if err := json.Unmarshal(harnessFixtureBytes(t, "claude-agents.json"), &entries); err != nil {
		t.Fatalf("reading the recorded agents listing: %v", err)
	}
	live := -1
	for i, entry := range entries {
		if _, ok := entry["status"]; ok {
			live = i
		}
	}
	if live < 0 {
		t.Fatal("the recorded agents listing holds no session with a status")
	}

	entries[live]["status"] = status
	if waitingFor != "" {
		entries[live]["waitingFor"] = waitingFor
	} else {
		delete(entries[live], "waitingFor")
	}
	id, _ := entries[live]["sessionId"].(string)
	if id == "" {
		t.Fatal("the live session in the recorded agents listing carries no sessionId")
	}
	rendered, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("rendering the agents listing: %v", err)
	}
	return string(rendered), id
}

// claudeClientInfo is how Claude Code named itself in the handshake that was
// recorded, which is the whole of what a bridge has to recognise it by.
func claudeClientInfo(t *testing.T) *mcp.Implementation {
	t.Helper()

	var frame struct {
		Params struct {
			ClientInfo mcp.Implementation `json:"clientInfo"`
		} `json:"params"`
	}
	if err := json.Unmarshal(harnessFixtureBytes(t, "initialize-claude.json"), &frame); err != nil {
		t.Fatalf("reading the recorded claude handshake: %v", err)
	}
	if frame.Params.ClientInfo.Name == "" {
		t.Fatal("the recorded claude handshake names no client")
	}
	return &frame.Params.ClientInfo
}

// startClaudeAgentsBridge starts `crabswarm mcp` the way Claude Code starts it:
// named as Claude Code in the handshake, carrying the session id Claude Code
// sets in the environment of every MCP server it spawns, and with the shim first
// on PATH so the feed's `claude agents --json` reaches it.
//
// The suite may itself be running under Claude Code, which carries a session id
// and a claude of its own. Both are appended after the scrubbed environment,
// where os/exec keeps the last value of a duplicate key, so the answers this
// case planted are the ones the bridge sees.
//
// No $CRABSWARM_HARNESS_SINK: the sink replaces the harness it was detected as,
// and it would take the feed with it.
func startClaudeAgentsBridge(
	t *testing.T, cfgPath, token string, shim *claudeAgentsShim,
) *mcp.ClientSession {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), crabswarmBin,
		"mcp", "--config", cfgPath, "--token", token)
	cmd.Env = append(chatEnviron(),
		harnessctl.ClaudeSessionEnv+"="+shim.sessionID,
		"PATH="+shim.dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	// Everything the bridge says goes to stderr by design, since stdout carries
	// the protocol; forwarding it is what makes a failing case readable.
	cmd.Stderr = os.Stderr

	client := mcp.NewClient(claudeClientInfo(t), nil)
	session, err := client.Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to the claude bridge for %q: %v", token, err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// waitClaudeMemberState blocks until the room lists address in state, read as
// the member itself.
//
// A refusal is waited through rather than failed on: between a daemon going away
// and the bridge attending again there is no member for the daemon to answer
// about, and one of the cases below walks straight through that gap.
func waitClaudeMemberState(
	t *testing.T, cfgPath, token, address, state string, timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var listed string
	for time.Now().Before(deadline) {
		rendered, _, err := execChat(t, cfgPath, token, "members")
		if err == nil {
			listed = rendered
			for _, l := range lines(rendered) {
				fields := strings.Fields(l)
				if len(fields) > 2 && fields[0] == address && fields[2] == state {
					return
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s did not reach the state %s within %s; the room listed:\n%s",
		address, state, timeout, listed)
}

// waitClaudeScreenCaptures blocks until the daemon has read token's screen want
// times, which is how many sweeps of the screen poller have been round.
func waitClaudeScreenCaptures(
	t *testing.T, cfgPath, token string, want int, timeout time.Duration,
) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	var captured []string
	for time.Now().Before(deadline) {
		captured = stubCaptures(t, cfgPath)
		read := 0
		for _, c := range captured {
			if c == token {
				read++
			}
		}
		if read >= want {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("the daemon read %s's screen fewer than %d times within %s; it captured %q",
		token, want, timeout, captured)
}

// The listing is the whole of a Claude Code member's state: no hook reports
// anything here, and the room follows the session from busy through idle to
// waiting with nothing changing but the answer the shim gives.
func TestChatClaudeAgents_TheListingBecomesTheMemberState(t *testing.T) {
	shim := startClaudeAgentsShim(t)
	cfg := writeChatConfig(t, 0, []stubCommand{
		{token: "tok-ana", dir: chatRoom, project: "alpha"},
	})
	startChatServe(t, cfg)
	startClaudeAgentsBridge(t, cfg, "tok-ana", shim)
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)

	// The handshake said which CLI this is. The launcher gave it no channel to
	// push a mention through, so the daemon still types at its terminal — a feed
	// says what the session is doing and nothing about how it is woken.
	row := memberRow(t, cfg, "tok-ana", chatBridgeAna)
	if len(row) != 5 || row[3] != "claude-code" || row[4] != "terminal" {
		t.Errorf("members row = %v, want claude-code reached through its terminal", row)
	}

	// Busy is the session working. The first state can arrive two ways — the
	// feed reads the listing as soon as the handshake names the harness, which
	// is early enough to race the attendance a report needs, and the bridge
	// says the last state it read again every ten seconds — so the wait is
	// generous rather than one poll wide.
	waitClaudeMemberState(t, cfg, "tok-ana", chatBridgeAna, "working", 30*time.Second)

	// Idle is the session done. The entry still says its state is working: a
	// status is what the session is doing at this moment and is read before the
	// state it keeps about its own progress, so this is the listing
	// disagreeing with itself and the live answer winning.
	shim.show(t, "idle")
	waitClaudeMemberState(t, cfg, "tok-ana", chatBridgeAna, "done", 30*time.Second)

	// Waiting is the session holding something somebody has to answer, whatever
	// it names as the reason.
	shim.show(t, "waiting")
	waitClaudeMemberState(t, cfg, "tok-ana", chatBridgeAna, "waiting", 30*time.Second)
}

// A member with a feed is left to it. The daemon reads screens for the harnesses
// nobody recognised, and reading this one beside its listing would only lay a
// wrong done over it: the composer line stands empty while a subagent running in
// the background keeps the session working.
func TestChatClaudeAgents_TheDaemonReadsNoScreenBesideTheListing(t *testing.T) {
	shim := startClaudeAgentsShim(t)
	cfg := startScreenPollDaemon(t, []stubCommand{
		{token: "tok-ana", dir: chatRoom, project: "alpha"},
		{token: "tok-bob", dir: chatRoom, project: "alpha"},
	})
	startClaudeAgentsBridge(t, cfg, "tok-ana", shim)
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)

	// Bob attends afterwards and runs a harness the daemon has no feed for, so
	// every sweep counted here had both of them on the roster: bob's screens are
	// what say the poller went round at all rather than not having reached
	// anybody yet.
	attendChatBridges(t, cfg, "tok-bob")
	waitClaudeScreenCaptures(t, cfg, "tok-bob", 3, 30*time.Second)

	// The stub answers a capture for whatever token it is handed, so a token
	// missing from its log is the daemon declining to read that screen rather
	// than the stub refusing to show one.
	if captured := stubCaptures(t, cfg); slices.Contains(captured, "tok-ana") {
		t.Errorf("capture-screen was asked for tok-ana; it captured %q", captured)
	}

	// And the member is where its own listing put it, with no screen read for it.
	waitClaudeMemberState(t, cfg, "tok-ana", chatBridgeAna, "working", 30*time.Second)
}

// A daemon that went away and came back has forgotten what everyone was doing:
// attendance is held in the daemon's memory, so the bridge attending again is
// recorded as a member with nothing to do. The listing has not moved and a feed
// speaks on change alone, so nothing there would say otherwise — what puts the
// member back at working is the bridge saying the last state it read again.
func TestChatClaudeAgents_ARestartedDaemonIsToldTheStateAgain(t *testing.T) {
	shim := startClaudeAgentsShim(t)
	cfg := writeChatConfig(t, 0, []stubCommand{
		{token: "tok-ana", dir: chatRoom, project: "alpha"},
	})
	serve := startChatServe(t, cfg)
	startClaudeAgentsBridge(t, cfg, "tok-ana", shim)
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)
	waitClaudeMemberState(t, cfg, "tok-ana", chatBridgeAna, "working", 30*time.Second)

	// The daemon alone goes. The bridge keeps running and the shim keeps
	// answering busy, which is what leaves the repeat as the only way the state
	// can come back.
	restartChatDaemon(t, cfg, serve)
	waitChatAttendance(t, cfg, "tok-ana", 30*time.Second)

	// Waited out from the attendance rather than from the restart: nothing is
	// said again while the room is missing the member, so a repeat that fell
	// into that gap is one interval late.
	waitClaudeMemberState(t, cfg, "tok-ana", chatBridgeAna, "working", 30*time.Second)
}
