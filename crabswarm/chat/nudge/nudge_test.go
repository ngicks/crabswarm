package nudge

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestNewMessage_NamesTheSenderAndTheTool(t *testing.T) {
	assert.Equal(t, NewMessage("beta/bob"),
		"[crabswarm chat] new message from beta/bob — read it with the chat_read tool")
}

// One waiting mention reads as one, not as "1 message(s)": the notice is a
// sentence the agent reads, not a log line.
func TestWaiting_CountsInWordsThatMatchTheCount(t *testing.T) {
	assert.Equal(t, Waiting(1),
		"[crabswarm chat] 1 unread message mentions you — read it with the chat_read tool")
	assert.Equal(t, Waiting(4),
		"[crabswarm chat] 4 unread messages mention you — read them with the chat_read tool")
}

func TestAddress_LeavesAnUnteamedSenderBare(t *testing.T) {
	assert.Equal(t, Address("beta", "bob"), "beta/bob")
	assert.Equal(t, Address("", "admin"), "admin")
}

func TestSanitize(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
		want string
	}{
		{
			// A raw newline would submit the line early and leave the rest of it
			// running as a command in the recipient's terminal.
			"newline is dropped",
			"bob\nrm -rf /",
			"bobrm -rf /",
		},
		{"carriage return is dropped", "beta\rreset/bob", "betareset/bob"},
		{"over-long address is cut", strings.Repeat("n", 200), strings.Repeat("n", MaxAddrLen)},
		{
			// The cap counts runes rather than bytes, so an address of
			// multi-byte characters is cut where a reader would count it.
			"the cap counts runes",
			strings.Repeat("é", 200),
			strings.Repeat("é", MaxAddrLen),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, Sanitize(tc.in), tc.want)
		})
	}
}
