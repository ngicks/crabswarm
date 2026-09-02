package crabswarm_test

import (
	"context"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	chatcli "github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli/tui"
)

// The watch screen is driven here rather than through the binary: a terminal
// program with nothing but pipes around it needs its input and output handed
// to it, which the process cannot be told to do from the outside. Everything
// under it is real — a daemon started by the suite, the same client the verb
// dials with, the same challenge on every call — so what is exercised is the
// screen against a live room. The failure paths, which never reach the screen,
// are asserted through the binary further down.

// chatSock is the socket the daemon config written by startChatDaemonKeeping
// names.
func chatSock(cfgPath string) string {
	return filepath.Join(filepath.Dir(cfgPath), "chat.sock")
}

// ansiEscape matches the control sequences a terminal program writes around the
// text it draws.
var ansiEscape = regexp.MustCompile(
	`\x1b\][^\a\x1b]*(?:\a|\x1b\\)` + // OSC, ended by BEL or ST
		`|\x1b\[[0-9;:?<>=!]*[ -/]*[@-~]` + // CSI
		`|\x1b[()*+][A-Za-z0-9]` + // character set designation
		`|\x1b[=>78MDEHc]`) // the odd two-byte ones

// screen collects what the program drew. The program renders from its own
// goroutine while the test reads, so the writes are guarded.
type screen struct {
	mu      sync.Mutex
	written strings.Builder
}

func (s *screen) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.written.Write(p)
}

// text is everything drawn so far with the escape sequences taken out. The
// renderer repaints only what changed, so this is the whole session rather than
// the current frame — enough to ask whether something ever reached the screen.
func (s *screen) text() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ansiEscape.ReplaceAllString(s.written.String(), "")
}

// screenTimeout is how long a test waits for something to be drawn. Generous
// against the screen's own poll interval, since a message has to be written,
// polled for and drawn.
const screenTimeout = 30 * time.Second

// waitScreen blocks until want has been drawn, and fails the test if it never
// is.
func waitScreen(t *testing.T, s *screen, want string) {
	t.Helper()
	deadline := time.Now().Add(screenTimeout)
	for time.Now().Before(deadline) {
		if strings.Contains(s.text(), want) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%q was never drawn; the screen so far:\n%s", want, s.text())
}

// typeOnScreen hands the program keystrokes the way a terminal would.
func typeOnScreen(t *testing.T, keys io.Writer, s string) {
	t.Helper()
	if _, err := io.WriteString(keys, s); err != nil {
		t.Fatalf("type %q: %v", s, err)
	}
}

// The operator opens the screen on a room already talking, reads back what was
// said before they arrived, watches a message land without touching anything,
// and sends one of their own — which reaches the pane the way every other
// message does, by being in the room's log.
func TestChatTUI_WatchesARoomAndSendsIntoIt(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	runChat(t, cfg, "tok-ana", "join", "--name", "ana")
	runChat(t, cfg, "tok-bob", "join", "--name", "bob")
	runChat(t, cfg, "tok-ana", "broadcast", "rebasing onto main")

	client, err := chatcli.Dial(chatSock(cfg))
	if err != nil {
		t.Fatalf("dial the daemon: %v", err)
	}
	defer client.Close()
	admin := client.Admin(identity)

	keys, typing := io.Pipe()
	defer typing.Close()
	drawn := &screen{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	quit := make(chan error, 1)
	go func() {
		quit <- tui.Run(ctx, tui.Deps{
			Room:   chatRoom,
			Log:    admin,
			Roster: admin,
			Sender: admin,
		}, tea.WithInput(keys), tea.WithOutput(drawn), tea.WithWindowSize(100, 24))
	}()

	// What the room said before the screen existed is on it, and so is who is
	// attending — neither cost a keypress.
	waitScreen(t, drawn, "alpha/ana → *: rebasing onto main")
	// The panes are framed and titled, and the members pane's title carries the
	// attendance the sidebar used to head itself with.
	waitScreen(t, drawn, "members (2)")

	// A message sent while the screen is open arrives on its own.
	runChat(t, cfg, "tok-bob", "broadcast", "the branch is green")
	waitScreen(t, drawn, "alpha/bob → *: the branch is green")

	// The operator steps in, addressing a member the way `chat send` does. The
	// screen opens on the conversation, so reaching the message pane is a focus
	// move down into it — ctrl+j, which a terminal sends as a bare line feed.
	typeOnScreen(t, typing, "\x0a")
	typeOnScreen(t, typing, "alpha/ana: hold the deploy\r")

	// It is in the room's log, attributed to the host, and it reaches the pane
	// from there rather than from the screen echoing itself.
	waitScreen(t, drawn, "admin/admin → alpha/ana: hold the deploy")
	logged := runChat(t, cfg, "", "admin", "log", chatRoom, "--identity", identity)
	if !strings.Contains(logged, "admin/admin → alpha/ana: hold the deploy") {
		t.Errorf("the room log = %q, want the admin's message in it", logged)
	}
	if got := runChat(t, cfg, "tok-ana", "read"); !strings.Contains(
		got, "admin/admin: hold the deploy") {
		t.Errorf("ana's inbox = %q, want the admin's message delivered", got)
	}

	// ctrl-c leaves, and the program says it left rather than being torn down.
	typeOnScreen(t, typing, "\x03")
	select {
	case err := <-quit:
		if err != nil {
			t.Fatalf("the screen exited with %v, want a clean quit", err)
		}
	case <-time.After(screenTimeout):
		t.Fatal("the screen did not quit on ctrl-c")
	}
}

// Everything that can stop the screen from opening stops it before it opens:
// each of these exits with a message on stderr rather than taking the terminal
// and showing nothing.
func TestChatTUI_FailuresExitBeforeTheScreen(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	runChat(t, cfg, "tok-ana", "join", "--name", "ana")

	stranger, _ := newChatIdentityFile(t)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "a room the daemon does not know, which names the ones it does",
			args: []string{"admin", "tui", "--room", "/work/nowhere", "--identity", identity},
			want: `no room "/work/nowhere": the daemon knows ` + chatRoom,
		},
		{
			name: "no identity to authenticate with",
			args: []string{"admin", "tui", "--room", chatRoom},
			want: "no admin age identity file",
		},
		{
			name: "an identity the daemon does not challenge",
			args: []string{"admin", "tui", "--room", chatRoom, "--identity", stranger},
			want: "decrypting the admin challenge",
		},
		{
			name: "nothing listening on the socket",
			args: []string{
				"admin", "tui", "--room", chatRoom, "--identity", identity,
				"--sock", filepath.Join(t.TempDir(), "absent.sock"),
			},
			want: "chat daemon unreachable",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := execChat(t, cfg, "", tc.args...)
			if err == nil {
				t.Fatalf("the screen opened; want a refusal.\nstdout:\n%s", stdout)
			}
			if !strings.Contains(stderr, tc.want) {
				t.Errorf("stderr = %q, want it to carry %q", stderr, tc.want)
			}
		})
	}
}
