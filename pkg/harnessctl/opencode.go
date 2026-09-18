package harnessctl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// OpenCodeRelayEnv names the loopback URL the crabswarm plugin listens for
// notices on, which the plugin puts into the environment of the MCP server it
// declares.
//
// The plugin delivers rather than this server, and the relay is how the two
// meet. Only the plugin knows which session the person is driving, and
// OpenCode's own way of typing at a session — appending to the composer and
// submitting it — would submit whatever draft the person had half-written. So
// the notice travels the other way: the server hands it to the plugin, and the
// plugin prompts the session it last saw the person write in.
const OpenCodeRelayEnv = "CRABSWARM_OPENCODE_RELAY"

// openCodeRelayTimeout bounds one call on the relay. The plugin answers as soon
// as it has handed the notice to the session — it never waits for the turn that
// follows — so this is only the point past which no answer is coming, and the
// feed the delivery runs on must not be held any longer than that.
const openCodeRelayTimeout = 5 * time.Second

// newOpenCode builds the opencode channel, which exists only where the plugin
// that serves it does: an OpenCode started without the plugin starts this server
// without the relay URL, and the member then attends as a terminal one.
func newOpenCode(getenv func(string) string, _ Session) Harness {
	relay := getenv(OpenCodeRelayEnv)
	if relay == "" {
		return nil
	}
	return openCode{relay: relay, timeout: openCodeRelayTimeout}
}

// openCode delivers a notice by posting it to the plugin's relay.
type openCode struct {
	relay   string
	timeout time.Duration
}

func (openCode) Kind() chatv1.Harness { return chatv1.Harness_HARNESS_OPENCODE }

func (openCode) Nudge() chatv1.NudgeDelivery {
	return chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE
}

// openCodeNotice is what the relay is handed, as the plugin reads it. The room
// rides along for a plugin that labels a notice with where it came from; the
// one this package ships prompts the session with the content alone and ignores
// the rest, which costs nothing and leaves the field there for one that does
// not.
type openCodeNotice struct {
	Content string `json:"content"`
	From    string `json:"from"`
	Room    string `json:"room"`
}

// Deliver posts one notice to the relay.
//
// Anything but a 2xx is an error, the plugin's refusal included: a plugin that
// has not seen the person write yet has no session to prompt, and reporting
// that leaves the mention waiting for the next report that ends a turn instead
// of counting it as delivered to nobody.
func (o openCode) Deliver(ctx context.Context, n Notice) error {
	body, err := json.Marshal(openCodeNotice{Content: n.Text, From: n.From, Room: n.Room})
	if err != nil {
		return fmt.Errorf("encoding a chat notice for the opencode relay: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.relay, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building a request for the opencode relay %s: %w", o.relay, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting a chat notice to the opencode relay %s: %w", o.relay, err)
	}
	defer func() { _ = resp.Body.Close() }()
	// Bounded: the answer is a word or two of why, and the relay is not this
	// process's to trust with how much it sends.
	answer, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("the opencode relay %s refused a chat notice: %s: %s",
			o.relay, resp.Status, bytes.TrimSpace(answer))
	}
	return nil
}
