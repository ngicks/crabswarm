package harness

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// fixtureDir holds the first frame each harness wrote to a real MCP server. The
// names this package recognises are read out of those recordings rather than
// spelled here a second time: a harness that renamed itself has to be
// re-captured, and the case below then fails until the mapping follows.
const fixtureDir = "../../../e2e/crabswarm/testdata/harness"

// clientNameOf reads the name a client gave itself out of one recorded
// initialize frame.
func clientNameOf(t *testing.T, fixture string) string {
	t.Helper()

	path := filepath.Join(fixtureDir, fixture)
	b, err := os.ReadFile(path)
	assert.NilError(t, err)

	var frame struct {
		Params struct {
			ClientInfo struct {
				Name string `json:"name"`
			} `json:"clientInfo"`
		} `json:"params"`
	}
	assert.NilError(t, json.Unmarshal(b, &frame), "reading %s", path)
	assert.Assert(t, frame.Params.ClientInfo.Name != "", "%s names no client", path)
	return frame.Params.ClientInfo.Name
}

// noEnv is a harness started with nothing in its environment, which is how a
// harness whose channel is not wired up starts one.
func noEnv(string) string { return "" }

func TestDetect_ReadsTheHarnessOffTheHandshake(t *testing.T) {
	for _, tc := range []struct {
		fixture string
		want    chatv1.Harness
	}{
		{"initialize-claude.json", chatv1.Harness_HARNESS_CLAUDE_CODE},
		{"initialize-codex.json", chatv1.Harness_HARNESS_CODEX},
		{"initialize-opencode.json", chatv1.Harness_HARNESS_OPENCODE},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			h := Detect(clientNameOf(t, tc.fixture), noEnv, nil)
			assert.Equal(t, h.Kind(), tc.want)
			// Until each harness's own channel is wired up, knowing which one it
			// is changes nothing about how it is woken.
			assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
		})
	}
}

// A client this package does not recognise is still a harness: the server is
// serving something, and a roster saying "other" is worth more to whoever reads
// it than a blank column. A client that named itself nothing at all is the same
// case.
func TestDetect_CallsAnUnknownClientOther(t *testing.T) {
	for _, name := range []string{"", "some-other-agent", "Claude-Code"} {
		t.Run(name, func(t *testing.T) {
			h := Detect(name, noEnv, nil)
			assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_OTHER)
			assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL)
		})
	}
}

// A terminal harness refuses a delivery rather than dropping it: the caller
// that asked believed this member reaches its own agent, and the daemon is
// typing at it on the understanding that it does not.
func TestDetect_ATerminalHarnessDeliversNothing(t *testing.T) {
	err := Detect("claude-code", noEnv, nil).Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorContains(t, err, "no channel")
}

// The sink makes the member a native one whatever it runs, which is what lets a
// test drive the delivery path: the real channels talk to a running harness,
// and a test suite has none to talk to.
func TestDetect_TheSinkTakesTheDelivery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notices.log")
	h := Detect("claude-code", func(name string) string {
		if name == SinkEnv {
			return path
		}
		return ""
	}, nil)

	assert.Equal(t, h.Kind(), chatv1.Harness_HARNESS_CLAUDE_CODE)
	assert.Equal(t, h.Nudge(), chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE)

	assert.NilError(t, h.Deliver(t.Context(), Notice{From: "beta/bob", Text: "first"}))
	assert.NilError(t, h.Deliver(t.Context(), Notice{Text: "second"}))

	// One line per notice, in the order they were delivered: the file is the
	// record of everything that reached the agent.
	b, err := os.ReadFile(path)
	assert.NilError(t, err)
	assert.DeepEqual(t, strings.Split(strings.TrimSuffix(string(b), "\n"), "\n"),
		[]string{"first", "second"})
}

// A sink that cannot be written is reported rather than swallowed: the caller
// logs it and the member's mentions wait for the next chance to be delivered.
func TestDetect_TheSinkReportsAFileItCannotOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing-dir", "notices.log")
	err := Detect("opencode", func(string) string { return path }, nil).
		Deliver(t.Context(), Notice{Text: "hi"})
	assert.ErrorContains(t, err, path)
}
