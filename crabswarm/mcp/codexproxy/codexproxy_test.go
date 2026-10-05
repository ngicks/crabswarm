package codexproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// A test binary started with tuiOutEnv set acts as a remote Codex TUI instead
// of running the tests: it attaches to the app server its --remote names,
// sends the message in tuiSendEnv, writes the answer to the file tuiOutEnv
// names, and exits with tuiExit.
const (
	tuiOutEnv  = "CODEXPROXY_TEST_TUI_OUT"
	tuiSendEnv = "CODEXPROXY_TEST_TUI_SEND"
	tuiExit    = 7
)

func TestMain(m *testing.M) {
	if out := os.Getenv(tuiOutEnv); out != "" {
		os.Exit(fakeTUI(out, os.Getenv(tuiSendEnv), os.Args[1:]))
	}
	os.Exit(m.Run())
}

func fakeTUI(out, send string, args []string) int {
	if err := attach(out, send, args); err != nil {
		fmt.Fprintln(os.Stderr, "fake TUI:", err)
		return 2
	}
	return tuiExit
}

func attach(out, send string, args []string) error {
	if len(args) != 2 || args[0] != "--remote" {
		return fmt.Errorf("want --remote ADDR, got %q", args)
	}
	path, ok := strings.CutPrefix(args[1], "unix://")
	if !ok {
		return fmt.Errorf("%q is no unix:// address", args[1])
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		},
	}}
	conn, _, err := websocket.Dial(ctx, "ws://localhost/", &websocket.DialOptions{
		HTTPClient: client,
	})
	if err != nil {
		return err
	}
	defer func() { _ = conn.CloseNow() }()
	if err := conn.Write(ctx, websocket.MessageText, []byte(send)); err != nil {
		return err
	}
	_, answer, err := conn.Read(ctx)
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, answer, 0o600); err != nil {
		return err
	}
	return conn.Close(websocket.StatusNormalClosure, "")
}

// proxySocket is where Run serves the TUI under runtimeDir.
func proxySocket(runtimeDir string) string {
	return filepath.Join(runtimeDir, "crabswarm-codex-proxy", strconv.Itoa(os.Getpid())+".sock")
}

// Run hands the command a socket of its own, relays what the command sends
// through it to the app server with the token it resolved, exits with the
// command's status, and leaves no socket behind.
func TestRun_RelaysTheCommandsConnection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		flag      string
		env       string
		server    string
		wantKey   string
		wantToken string
	}{
		{
			name:      "token from the environment",
			env:       "tok-env",
			wantKey:   headerKey,
			wantToken: "tok-env",
		},
		{
			name:      "token from the flag",
			flag:      "tok-flag",
			env:       "tok-env",
			wantKey:   headerKey,
			wantToken: "tok-flag",
		},
		{
			name:      "another server name",
			env:       "tok-env",
			server:    "cs",
			wantKey:   "mcp_servers.cs.http_headers.X-Crabswarm-Token",
			wantToken: "tok-env",
		},
		{name: "no token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtimeDir := shortDir(t)
			t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
			t.Setenv(cli.TokenEnvVar, tc.env)
			t.Setenv(cli.CmdIDEnvVar, "")
			app := startFakeAppServer(t)
			start := fixtureLines(t, "codex-tui-thread-start.client.ndjson")[0]
			out := filepath.Join(shortDir(t), "answer")
			t.Setenv(tuiOutEnv, out)
			t.Setenv(tuiSendEnv, string(start))

			err := Run(t.Context(), testLogger(t), Config{
				Token:  tc.flag,
				Argv:   []string{os.Args[0], "--remote", "unix://" + app.path},
				Server: tc.server,
			})
			exitErr, ok := errors.AsType[*ExitError](err)
			assert.Assert(t, ok, "Run returned %v", err)
			assert.Equal(t, exitErr.Code, tuiExit)

			got := app.waitGot(t, 1)
			if tc.wantToken == "" {
				assert.DeepEqual(t, got[0].Data, start)
			} else {
				want := decoded(t, start)
				want["params"].(map[string]any)["config"].(map[string]any)[tc.wantKey] =
					tc.wantToken
				assert.DeepEqual(t, decoded(t, got[0].Data), want)
			}
			answer, err := os.ReadFile(out)
			assert.NilError(t, err)
			assert.DeepEqual(t, answer,
				answerTo(json.RawMessage(requestId(t, start)), "thread/start"))

			_, err = os.Stat(proxySocket(runtimeDir))
			assert.Assert(t, errors.Is(err, fs.ErrNotExist), "the socket is still there: %v", err)
		})
	}
}

// runSh runs Run over `sh -c script sh args...`, skipping where there is no sh.
func runSh(t *testing.T, script string, args ...string) error {
	t.Helper()
	return runShWith(t, t.Context(), script, args...)
}

// runShWith is [runSh] under ctx.
func runShWith(t *testing.T, ctx context.Context, script string, args ...string) error {
	t.Helper()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh to run a command with")
	}
	argv := append([]string{sh, "-c", script, "sh"}, args...)
	return Run(ctx, testLogger(t), Config{Argv: argv})
}

// The command is started with the proxy's socket already served, sees it in
// place of the address it was given, in the spelling it was given, and its
// exit status is what Run reports.
func TestRun_HandsTheCommandTheProxySocketAndPassesItsStatus(t *testing.T) {
	runtimeDir := shortDir(t)
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	out := filepath.Join(shortDir(t), "argv")
	t.Setenv("CODEXPROXY_TEST_ARGV", out)

	err := runSh(t,
		`test -S "${1#--remote=unix://}" || exit 99
printf '%s\n' "$@" > "$CODEXPROXY_TEST_ARGV"
exit 3`,
		"--remote=unix:///nowhere.sock", "-m", "gpt")
	exitErr, ok := errors.AsType[*ExitError](err)
	assert.Assert(t, ok, "Run returned %v", err)
	assert.Equal(t, exitErr.Code, 3)

	argv, err := os.ReadFile(out)
	assert.NilError(t, err)
	assert.Equal(t, string(argv), "--remote=unix://"+proxySocket(runtimeDir)+"\n-m\ngpt\n")
	_, err = os.Stat(proxySocket(runtimeDir))
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "the socket is still there: %v", err)
}

func TestRun_ReturnsNilForAZeroStatus(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", shortDir(t))
	assert.NilError(t, runSh(t, "exit 0", "--remote", "unix:///nowhere.sock"))
}

// A command a signal ended exits the way a shell reports it: 128 plus the
// signal number.
func TestRun_ReportsASignalledCommandAsAShellWould(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", shortDir(t))
	err := runSh(t, "kill -TERM $$", "--remote", "unix:///nowhere.sock")
	exitErr, ok := errors.AsType[*ExitError](err)
	assert.Assert(t, ok, "Run returned %v", err)
	assert.Equal(t, exitErr.Code, 128+15)
}

// A command line the proxy cannot sit in front of is refused before the
// command runs.
func TestRun_RefusesBeforeRunningTheCommand(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", shortDir(t))
	ran := filepath.Join(shortDir(t), "ran")
	t.Setenv("CODEXPROXY_TEST_RAN", ran)

	err := runSh(t, `touch "$CODEXPROXY_TEST_RAN"`, "--remote", "ws://127.0.0.1:1")
	assert.ErrorContains(t, err, "unix://")
	_, err = os.Stat(ran)
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "the command ran")
}

// A socket directory the caller names is where the command is handed its
// socket, created on the way, and the fallback under $XDG_RUNTIME_DIR is left
// alone.
func TestRun_ServesInTheGivenSockDir(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh to run a command with")
	}
	runtimeDir := shortDir(t)
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	sockDir := filepath.Join(shortDir(t), "given")
	out := filepath.Join(shortDir(t), "argv")
	t.Setenv("CODEXPROXY_TEST_ARGV", out)

	err = Run(t.Context(), testLogger(t), Config{
		Argv: []string{sh, "-c",
			`test -S "${2#unix://}" || exit 99
printf '%s\n' "$2" > "$CODEXPROXY_TEST_ARGV"`,
			"sh", "--remote", "unix:///nowhere.sock"},
		SockDir: sockDir,
	})
	assert.NilError(t, err)

	argv, err := os.ReadFile(out)
	assert.NilError(t, err)
	want := filepath.Join(sockDir, strconv.Itoa(os.Getpid())+".sock")
	assert.Equal(t, string(argv), "unix://"+want+"\n")
	info, err := os.Stat(sockDir)
	assert.NilError(t, err)
	assert.Equal(t, info.Mode().Perm(), fs.FileMode(0o700))
	_, err = os.Stat(filepath.Join(runtimeDir, "crabswarm-codex-proxy"))
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "the default directory was made: %v", err)
}

// A given socket directory that was there before with wider permissions is
// taken over by its owner alone.
func TestSocketPath_SecuresTheGivenDir(t *testing.T) {
	dir := filepath.Join(shortDir(t), "wide")
	assert.NilError(t, os.Mkdir(dir, 0o755))

	path, err := socketPath(dir)
	assert.NilError(t, err)
	assert.Equal(t, path, filepath.Join(dir, strconv.Itoa(os.Getpid())+".sock"))
	info, err := os.Stat(dir)
	assert.NilError(t, err)
	assert.Equal(t, info.Mode().Perm(), fs.FileMode(0o700))
}

// With no socket directory given and no $XDG_RUNTIME_DIR, the socket goes
// under the temporary directory, in a directory only its owner can enter, even
// one that was there before with wider permissions.
func TestSocketPath_FallsBackToTheTempDir(t *testing.T) {
	tmp := shortDir(t)
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", tmp)
	assert.NilError(t, os.Mkdir(filepath.Join(tmp, "crabswarm-codex-proxy"), 0o755))

	path, err := socketPath("")
	assert.NilError(t, err)
	assert.Equal(t, path, proxySocket(tmp))
	info, err := os.Stat(filepath.Dir(path))
	assert.NilError(t, err)
	assert.Equal(t, info.Mode().Perm(), fs.FileMode(0o700))
}

// A socket file nobody answers on is replaced; one somebody serves is not.
func TestListen_ReplacesOnlyASocketNobodyServes(t *testing.T) {
	path := filepath.Join(shortDir(t), "p.sock")
	stale, err := net.Listen("unix", path)
	assert.NilError(t, err)
	stale.(*net.UnixListener).SetUnlinkOnClose(false)
	assert.NilError(t, stale.Close())

	ln, err := listen(path)
	assert.NilError(t, err)
	defer func() { _ = ln.Close() }()

	_, err = listen(path)
	assert.ErrorContains(t, err, "serving the proxy socket")
}
