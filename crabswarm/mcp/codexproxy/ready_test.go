package codexproxy

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

var probe = []string{"initialize", "initialized", "account/read"}

// The command starts only after the app server answered account/read with a
// result; every refusal before that is retried on a session of its own.
func TestRun_WaitsForAccountRead(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", shortDir(t))
	t.Setenv(cli.TokenEnvVar, "")
	t.Setenv(cli.CmdIDEnvVar, "")
	app := startFakeAppServer(t)
	app.refuse(1)
	start := fixtureLines(t, "codex-tui-thread-start.client.ndjson")[0]
	t.Setenv(tuiOutEnv, filepath.Join(shortDir(t), "answer"))
	t.Setenv(tuiSendEnv, string(start))

	err := Run(t.Context(), testLogger(t), Config{
		Argv:         []string{os.Args[0], "--remote", "unix://" + app.path},
		ReadyTimeout: 10 * time.Second,
	})
	exitErr, ok := errors.AsType[*ExitError](err)
	assert.Assert(t, ok, "Run returned %v", err)
	assert.Equal(t, exitErr.Code, tuiExit)

	want := append(append(append([]string{}, probe...), probe...), "thread/start")
	assert.DeepEqual(t, app.methods(), want)
}

// An app server that never answers account/read, or is not there at all, does
// not keep the command from starting once the timeout passes.
func TestRun_StartsTheCommandPastTheReadyTimeout(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh to run a command with")
	}
	refusing := startFakeAppServer(t)
	refusing.refuse(1 << 30)
	for _, tc := range []struct {
		name string
		path string
	}{
		{name: "refusing", path: refusing.path},
		{name: "absent", path: filepath.Join(shortDir(t), "absent.sock")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_RUNTIME_DIR", shortDir(t))
			err := Run(t.Context(), testLogger(t), Config{
				Argv:         []string{sh, "-c", "exit 0", "sh", "--remote", "unix://" + tc.path},
				ReadyTimeout: 200 * time.Millisecond,
			})
			assert.NilError(t, err)
		})
	}
}

// A Run cancelled while it waits returns without running the command.
func TestRun_CancelledWhileWaitingRunsNothing(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", shortDir(t))
	app := startFakeAppServer(t)
	app.refuse(1 << 30)
	ran := filepath.Join(shortDir(t), "ran")
	t.Setenv("CODEXPROXY_TEST_RAN", ran)
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh to run a command with")
	}

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	err = Run(ctx, testLogger(t), Config{
		Argv: []string{
			sh, "-c", `touch "$CODEXPROXY_TEST_RAN"`, "sh", "--remote", "unix://" + app.path,
		},
		ReadyTimeout: time.Minute,
	})
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = os.Stat(ran)
	assert.Assert(t, errors.Is(err, fs.ErrNotExist), "the command ran")
}
