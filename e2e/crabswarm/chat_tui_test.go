package crabswarm_test

import (
	"context"
	"fmt"
	"io"
	"os"
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

// chatSock is the socket the daemon config written by writeChatConfig names.
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
	return s.since(0)
}

// mark is how much has been drawn so far, which is where the next frame starts.
func (s *screen) mark() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.written.Len()
}

// since is everything drawn after mark, with the escape sequences taken out.
func (s *screen) since(mark int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ansiEscape.ReplaceAllString(s.written.String()[mark:], "")
}

// screenTimeout is how long a test waits for something to be drawn. Generous
// against the screen's own poll interval, since a message has to be written,
// polled for and drawn.
const screenTimeout = 30 * time.Second

// The size the screen is driven at: wide enough for both columns beside each
// other and for every field of the status bar, and the height a terminal is
// conventionally assumed to have.
const (
	screenWidth  = 100
	screenHeight = 24
)

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

// sizeReport is a terminal answering how big it is (XTWINOPS), with the size
// the screen was opened at. Nothing on the screen moves — it is the size it
// already had — but a reported size makes the renderer throw its diff away and
// paint every cell again, which is what [waitFrame] is after.
var sizeReport = fmt.Sprintf("\x1b[8;%d;%dt", screenHeight, screenWidth)

// framePause is how long a repaint is given to arrive before the frame is read
// back: a few of the renderer's own frames, and retried until the deadline
// rather than relied on.
const framePause = 300 * time.Millisecond

// waitFrame asks the screen to draw itself whole and blocks until one such
// frame carries every want at once, which it returns.
//
// [waitScreen] asks whether something was ever drawn, which is all a new line
// of conversation needs. This asks what the screen says now. The renderer
// writes only the cells that changed, so a line edited in place is never in
// the output as a line: switching from /work/other to /work/proj spells the
// status bar as "\x1b[24;13Hproj \x1b[P" — the new name, then a delete of the
// cell the shorter name freed. Repainting is what puts the whole line there.
func waitFrame(t *testing.T, s *tuiScreen, want ...string) string {
	t.Helper()
	deadline := time.Now().Add(screenTimeout)
	var frame, missing string
	for time.Now().Before(deadline) {
		mark := s.drawn.mark()
		typeOnScreen(t, s.keys, sizeReport)
		time.Sleep(framePause)
		frame, missing = s.drawn.since(mark), ""
		for _, w := range want {
			if !strings.Contains(frame, w) {
				missing = w
				break
			}
		}
		if missing == "" {
			return frame
		}
	}
	t.Fatalf("%q was never on the screen; the last frame drawn:\n%s", missing, frame)
	return ""
}

// tuiScreen is one run of the watch screen: the pipe the program reads its
// keystrokes from, everything it has drawn, and how it exited.
type tuiScreen struct {
	keys  *os.File
	drawn *screen
	quit  <-chan error
}

// startTUI opens the watch screen on cfg's daemon as the holder of identity,
// over the same client the verb dials with. deps arrives holding what the
// operator chose — the room, the editor — and the three daemon-facing halves
// are filled in here.
//
// The keystrokes travel down an os.Pipe rather than an io.Pipe, because ctrl+g
// hands the terminal to a child process and takes it back: a read on a file
// descriptor can be cancelled, so the program's own reader stops rather than
// staying blocked on the pipe and racing the restored one for the keys after
// it; and a child is handed the descriptor itself rather than a goroutine
// copying into it, which would swallow a keystroke to notice the child is gone.
func startTUI(t *testing.T, cfg, identity string, deps tui.Deps) *tuiScreen {
	t.Helper()
	client, err := chatcli.Dial(chatSock(cfg))
	if err != nil {
		t.Fatalf("dial the daemon: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	keys, typing, err := os.Pipe()
	if err != nil {
		t.Fatalf("open the keystroke pipe: %v", err)
	}
	t.Cleanup(func() {
		_ = typing.Close()
		_ = keys.Close()
	})

	admin := client.Admin(identity)
	deps.Log, deps.Roster, deps.Sender = admin, admin, admin
	drawn := &screen{}
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	quit := make(chan error, 1)
	go func() {
		quit <- tui.Run(ctx, deps,
			tea.WithInput(keys), tea.WithOutput(drawn),
			tea.WithWindowSize(screenWidth, screenHeight))
	}()
	return &tuiScreen{keys: typing, drawn: drawn, quit: quit}
}

// quitScreen leaves the screen with ctrl+c and waits for the program to say it
// left, rather than for the test to tear it down.
func quitScreen(t *testing.T, s *tuiScreen) {
	t.Helper()
	typeOnScreen(t, s.keys, "\x03")
	select {
	case err := <-s.quit:
		if err != nil {
			t.Fatalf("the screen exited with %v, want a clean quit", err)
		}
	case <-time.After(screenTimeout):
		t.Fatal("the screen did not quit on ctrl-c")
	}
}

// The operator opens the screen on a room already talking, reads back what was
// said before they arrived, watches a message land without touching anything,
// and sends messages of their own — which reach the pane the way every other
// message does, by being in the room's log.
//
// The three sends are the three things a message can be for: the whole room, a
// list of roles, and nobody at all. The pane's target column is where they are
// told apart.
func TestChatTUI_WatchesARoomAndSendsIntoIt(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	one := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "claude-1")
	two := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "claude-2")
	runChat(t, cfg, one, "send", "everyone", "rebasing onto main")

	s := startTUI(t, cfg, identity, tui.Deps{Room: chatRoom})

	// What the room said before the screen existed is on it, and so is who is
	// attending — neither cost a keypress.
	waitScreen(t, s.drawn, "alpha/claude-1 → everyone: rebasing onto main")
	// The panes are framed and titled, and the members pane's title carries the
	// attendance the sidebar used to head itself with.
	waitScreen(t, s.drawn, "members (2)")

	// A message sent while the screen is open arrives on its own.
	runChat(t, cfg, two, "send", "everyone", "the branch is green")
	waitScreen(t, s.drawn, "alpha/claude-2 → everyone: the branch is green")

	// The operator steps in. The screen opens on the conversation, so reaching
	// the message pane is a focus move down into it — ctrl+j, which a terminal
	// sends as a bare line feed. enter is a newline there, so each message is
	// sent with ctrl+x, which is byte 0x18.
	typeOnScreen(t, s.keys, "\x0a")

	// @everyone is the whole room, and the pane says so.
	typeOnScreen(t, s.keys, "@everyone hi\x18")
	waitScreen(t, s.drawn, "admin → everyone: @everyone hi")

	// Several tokens are one list of roles, resolved to the members they name.
	// This is the longest line the room draws, so it is read off a whole frame
	// rather than out of everything drawn so far: the renderer writes only the
	// cells that changed, and the message pane shrinking back to one row after
	// the send redraws part of this line on its own.
	typeOnScreen(t, s.keys, "@claude-1 @claude-2 hi\x18")
	waitFrame(t, s,
		"admin → alpha/claude-1,alpha/claude-2: @claude-1",
		"sent to alpha/claude-1,alpha/claude-2")

	// No token at all names nobody, which is a board post: the target column is
	// a dash, and nobody was mentioned.
	typeOnScreen(t, s.keys, "hi\x18")
	waitScreen(t, s.drawn, "admin → -: hi")
	waitFrame(t, s, "sent a post")

	// All three are in the room's log, attributed to the host, and they reached
	// the pane from there rather than from the screen echoing itself. The
	// tokens that addressed them travel with the text: they are also the
	// mentions that name who was asked.
	logged := runChat(t, cfg, "", "admin", "log", chatRoom, "--identity", identity)
	for _, want := range []string{
		"admin -> everyone: @everyone hi",
		"admin -> alpha/claude-1,alpha/claude-2: @claude-1 @claude-2 hi",
		"admin -> -: hi",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("the room log = %q, want %q in it", logged, want)
		}
	}
	// The roles send reached the member it named; the post mentioned nobody and
	// so is not waiting in anybody's unread.
	got := runChat(t, cfg, one, "read")
	if !strings.Contains(got, "@claude-1 @claude-2 hi") {
		t.Errorf("claude-1's inbox = %q, want the message that named them", got)
	}
	if strings.Contains(got, "admin -> -:") {
		t.Errorf("claude-1's inbox = %q, want the board post to mention nobody", got)
	}

	// ctrl-c leaves, and the program says it left rather than being torn down.
	quitScreen(t, s)
}

// A daemon knowing two rooms, and the operator moving between them: the screen
// opens on the one the daemon lists first when nothing names one, and the rooms
// pane takes the whole screen — conversation, attendance and status bar — to
// whichever room the cursor is on.
func TestChatTUI_OpensAndSwitchesBetweenRooms(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	ana := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "ana")
	registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob")
	registerChatHuman(t, cfg, identity, chatRoom, "beta", "cid")
	zed := registerChatHuman(t, cfg, identity, chatOtherRoom, "gamma", "zed")
	runChat(t, cfg, ana, "send", "everyone", "the proj room is talking")
	runChat(t, cfg, zed, "send", "everyone", "the other room is talking")

	// The daemon lists its rooms by name, so /work/other is the first of the
	// two and the one a screen that was told no room opens on.
	t.Run("no --room opens on the first room listed", func(t *testing.T) {
		s := startTUI(t, cfg, identity, tui.Deps{})
		waitScreen(t, s.drawn, "gamma/zed → everyone: the other room is talking")

		frame := waitFrame(t, s, "room "+chatOtherRoom, "members (1)")
		if strings.Contains(frame, "the proj room is talking") {
			t.Errorf("the screen carries the other room's conversation:\n%s", frame)
		}
		quitScreen(t, s)
	})

	t.Run("enter in the rooms pane takes the screen to that room", func(t *testing.T) {
		s := startTUI(t, cfg, identity, tui.Deps{Room: chatOtherRoom})
		waitScreen(t, s.drawn, "gamma/zed → everyone: the other room is talking")

		// The screen opens on the conversation. ctrl+h lands on the members
		// pane, which is what lies to its left at this split, and ctrl+k on the
		// rooms pane above that. The cursor opens on the room being watched, so
		// one j reaches the other one.
		typeOnScreen(t, s.keys, "\x08")
		typeOnScreen(t, s.keys, "\x0b")
		typeOnScreen(t, s.keys, "j")
		typeOnScreen(t, s.keys, "\r")

		// The room switched: its conversation is on the screen, its attendance
		// is in the members pane, and the status bar names it.
		waitScreen(t, s.drawn, "alpha/ana → everyone: the proj room is talking")
		frame := waitFrame(t, s,
			"room "+chatRoom, "members (3)", "alpha", " ana ", " bob ", "beta", " cid ")
		for _, left := range []string{"the other room is talking", " zed "} {
			if strings.Contains(frame, left) {
				t.Errorf("the screen still carries %q of the room it left:\n%s", left, frame)
			}
		}
		quitScreen(t, s)
	})
}

// The members pane addresses the message: enter on a member writes that
// member's address in front of what is written and follows it into the message
// pane. A team heading is not an address — a team is not something a message
// can be sent to — and enter on one says so.
func TestChatTUI_MembersPaneAddressesTheMessage(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	ana := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "ana")
	registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob")
	registerChatHuman(t, cfg, identity, chatRoom, "beta", "cid")
	runChat(t, cfg, ana, "send", "everyone", "who is awake?")

	t.Run("enter on a member writes @team/name", func(t *testing.T) {
		s := startTUI(t, cfg, identity, tui.Deps{Room: chatRoom})
		waitScreen(t, s.drawn, "alpha/ana → everyone: who is awake?")

		// ctrl+h into the members pane, whose cursor opens on the first row —
		// the heading of the first team — so one j is that team's first member.
		typeOnScreen(t, s.keys, "\x08")
		typeOnScreen(t, s.keys, "j")
		typeOnScreen(t, s.keys, "\r")
		typeOnScreen(t, s.keys, "the deploy is yours\x18")

		// The address the pane wrote is part of the message, so what ana was
		// delivered is the whole line, mention included.
		waitFrame(t, s, "sent to alpha/ana")
		if got := runChat(t, cfg, ana, "read"); !strings.Contains(
			got, "admin -> alpha/ana") {
			t.Errorf("ana's inbox = %q, want the whole addressed line", got)
		}
		quitScreen(t, s)
	})

	t.Run("enter on a team heading says a team is not a target", func(t *testing.T) {
		s := startTUI(t, cfg, identity, tui.Deps{Room: chatRoom})
		waitScreen(t, s.drawn, "alpha/ana → everyone: who is awake?")

		// The cursor opens on the alpha heading, which names a team.
		typeOnScreen(t, s.keys, "\x08")
		typeOnScreen(t, s.keys, "\r")

		// Nothing was written into the message and the focus stayed put; the
		// system line names what to write instead.
		waitFrame(t, s, "a team is not a target")
		quitScreen(t, s)
	})
}

// Tab completes the `@token` under the cursor against the room's attendance and
// the whole room beside it: one match is the answer and is applied, and more
// than one is a list to pick from.
func TestChatTUI_TabCompletesAnAddress(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	ana := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "ana")
	registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob")
	registerChatHuman(t, cfg, identity, chatRoom, "beta", "cid")
	runChat(t, cfg, ana, "send", "everyone", "standup in five")

	t.Run("one match completes in place", func(t *testing.T) {
		s := startTUI(t, cfg, identity, tui.Deps{Room: chatRoom})
		waitScreen(t, s.drawn, "alpha/ana → everyone: standup in five")

		// ctrl+j into the message pane, then a prefix only ana answers to. Tab
		// is byte 0x09.
		typeOnScreen(t, s.keys, "\x0a")
		typeOnScreen(t, s.keys, "@an\t")
		waitFrame(t, s, "> @alpha/ana")

		// The token is completed with the space that ends it, which the next
		// word lands after rather than inside.
		typeOnScreen(t, s.keys, "ready?")
		waitFrame(t, s, "> @alpha/ana ready?")
		quitScreen(t, s)
	})

	t.Run("the whole room is offered like any other address", func(t *testing.T) {
		s := startTUI(t, cfg, identity, tui.Deps{Room: chatRoom})
		waitScreen(t, s.drawn, "alpha/ana → everyone: standup in five")

		// Nobody in the room answers to a name or an address starting with "e",
		// so the whole room is the one match.
		typeOnScreen(t, s.keys, "\x0a")
		typeOnScreen(t, s.keys, "@e\t")
		waitFrame(t, s, "> @everyone")
		quitScreen(t, s)
	})

	t.Run("an ambiguous prefix opens the dropdown", func(t *testing.T) {
		s := startTUI(t, cfg, identity, tui.Deps{Room: chatRoom})
		waitScreen(t, s.drawn, "alpha/ana → everyone: standup in five")

		// "a" is both of alpha's members, so the list is offered instead of one
		// of them being chosen.
		typeOnScreen(t, s.keys, "\x0a")
		typeOnScreen(t, s.keys, "@a\t")
		waitFrame(t, s, "@alpha/ana", "@alpha/bob", "> @a")

		// Tab walks the list and enter takes the row the highlight is on, which
		// is the second of the two.
		typeOnScreen(t, s.keys, "\t")
		typeOnScreen(t, s.keys, "\r")
		frame := waitFrame(t, s, "> @alpha/bob")
		if strings.Contains(frame, "@alpha/ana") {
			t.Errorf("the dropdown is still open after accepting:\n%s", frame)
		}
		quitScreen(t, s)
	})
}

// editorLine is what the stand-in editor leaves in the file it is handed, which
// is how a test tells the draft that went in from what came back.
const editorLine = "appended by the editor"

// editorScript writes an executable stand-in for the operator's editor: a shell
// script that appends [editorLine] to the file it is handed and exits with
// code. The line opens with a newline of its own, since the draft is written to
// the file without a trailing one — and the failing script appends it too, so
// what an editor that exited non-zero left behind is refused rather than
// missing.
func editorScript(t *testing.T, code int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "editor")
	writeFile(t, path, fmt.Sprintf(
		"#!/bin/sh\nprintf '\\n%s\\n' >> \"$1\"\nexit %d\n", editorLine, code))
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatalf("chmod the editor stand-in: %v", err)
	}
	return path
}

// ctrl+g hands the draft to the operator's own editor and takes back whatever
// it left, and nothing is sent by it. The editor is resolved the way the
// command resolves it — out of $VISUAL, else $EDITOR — so what the screen is
// handed here is what the operator's environment would hand it.
func TestChatTUI_EditsTheDraftInAnEditor(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	ana := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "ana")
	runChat(t, cfg, ana, "send", "everyone", "a long one is coming")

	const draft = "@alpha/ana the draft"

	t.Run("what the editor leaves comes back into the pane", func(t *testing.T) {
		t.Setenv(chatcli.VisualEnvVar, editorScript(t, 0))
		s := startTUI(t, cfg, identity, tui.Deps{
			Room: chatRoom, Editor: chatcli.EditorFromEnv()})
		waitScreen(t, s.drawn, "alpha/ana → everyone: a long one is coming")

		typeOnScreen(t, s.keys, "\x0a")
		typeOnScreen(t, s.keys, draft)
		typeOnScreen(t, s.keys, "\x07")

		// The draft and the editor's own line are both in the pane, which has
		// grown the row that took.
		waitFrame(t, s, "> "+draft, "> "+editorLine)
		if logged := runChat(
			t, cfg, "", "admin", "log", chatRoom, "--identity", identity,
		); strings.Contains(logged, editorLine) {
			t.Errorf("the room log = %q, want the editor to have sent nothing", logged)
		}
		quitScreen(t, s)
	})

	t.Run("an editor that fails leaves the draft alone", func(t *testing.T) {
		t.Setenv(chatcli.VisualEnvVar, editorScript(t, 1))
		s := startTUI(t, cfg, identity, tui.Deps{
			Room: chatRoom, Editor: chatcli.EditorFromEnv()})
		waitScreen(t, s.drawn, "alpha/ana → everyone: a long one is coming")

		typeOnScreen(t, s.keys, "\x0a")
		typeOnScreen(t, s.keys, draft)
		typeOnScreen(t, s.keys, "\x07")

		// The hand-off did not happen, so the draft is exactly as it was typed
		// — the file the editor wrote before failing is not taken.
		frame := waitFrame(t, s, "editor exited:", "> "+draft)
		if strings.Contains(frame, editorLine) {
			t.Errorf("what the failed editor wrote reached the pane:\n%s", frame)
		}
		quitScreen(t, s)
	})

	t.Run("no VISUAL and no EDITOR is reported", func(t *testing.T) {
		t.Setenv(chatcli.VisualEnvVar, "")
		t.Setenv(chatcli.EditorEnvVar, "")
		s := startTUI(t, cfg, identity, tui.Deps{
			Room: chatRoom, Editor: chatcli.EditorFromEnv()})
		waitScreen(t, s.drawn, "alpha/ana → everyone: a long one is coming")

		typeOnScreen(t, s.keys, "\x0a")
		typeOnScreen(t, s.keys, draft)
		typeOnScreen(t, s.keys, "\x07")

		// Nothing is guessed at and nothing is run; the draft stays where it is.
		waitFrame(t, s, "no VISUAL or EDITOR set", "> "+draft)
		quitScreen(t, s)
	})
}

// enter writes a line in the message pane and never sends: a message to a
// harness is often a paragraph. Sending is ctrl+enter where the terminal can
// report it, and ctrl+x everywhere else.
func TestChatTUI_EnterWritesALineAndTheSendKeysSend(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)

	ana := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "ana")
	bob := registerChatHuman(t, cfg, identity, chatRoom, "alpha", "bob")
	runChat(t, cfg, ana, "send", "everyone", "two lines please")

	s := startTUI(t, cfg, identity, tui.Deps{Room: chatRoom})
	waitScreen(t, s.drawn, "alpha/ana → everyone: two lines please")

	// ctrl+j into the message pane, and enter — a carriage return from a
	// terminal — between the two lines.
	typeOnScreen(t, s.keys, "\x0a")
	typeOnScreen(t, s.keys, "@alpha/ana first line")
	typeOnScreen(t, s.keys, "\r")
	typeOnScreen(t, s.keys, "second line")

	// Both lines are in the pane, which has grown to hold them, and the room
	// has been told nothing.
	waitFrame(t, s, "> @alpha/ana first line", "> second line")
	if logged := runChat(
		t, cfg, "", "admin", "log", chatRoom, "--identity", identity,
	); strings.Contains(logged, "second line") {
		t.Errorf("the room log = %q, want enter to have sent nothing", logged)
	}

	// ctrl+x sends the whole of it, both lines and the address they were
	// written under.
	typeOnScreen(t, s.keys, "\x18")
	waitFrame(t, s, "sent to alpha/ana")
	got := runChat(t, cfg, ana, "read")
	for _, want := range []string{"admin -> alpha/ana", "@alpha/ana first line", "second line"} {
		if !strings.Contains(got, want) {
			t.Errorf("ana's inbox = %q, want it to carry %q", got, want)
		}
	}

	// And ctrl+enter sends where the terminal can say ctrl+enter: the kitty
	// keyboard protocol spells it as CSI 13 ; 5 u, which is what a terminal
	// that answered the screen's disambiguation request would send.
	typeOnScreen(t, s.keys, "@alpha/bob and one for bob")
	typeOnScreen(t, s.keys, "\x1b[13;5u")
	waitFrame(t, s, "sent to alpha/bob")
	if got := runChat(t, cfg, bob, "read"); !strings.Contains(
		got, "@alpha/bob and one for bob") {
		t.Errorf("bob's inbox = %q, want the message ctrl+enter sent", got)
	}

	quitScreen(t, s)
}

// Everything that can stop the screen from opening stops it before it opens:
// each of these exits with a message on stderr rather than taking the terminal
// and showing nothing.
func TestChatTUI_FailuresExitBeforeTheScreen(t *testing.T) {
	identity, recipient := newChatIdentityFile(t)
	cfg := startChatDaemonWith(t, defaultStubCommands(), recipient)
	registerChatHuman(t, cfg, identity, chatRoom, "alpha", "ana")

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
