package cmdman_test

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/creack/pty/v2"
	"github.com/ngicks/crabswarm/pkg/cmdman"
)

func TestAttach_DetachKeysExitWithoutStoppingCommand(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	env := newTestEnv(t)

	id := env.run(ctx, "run", "-n", "attach-detach", "--", "/bin/sh", "-c", "sleep 300")
	t.Cleanup(func() { env.cleanupCommand(ctx, id) })
	env.waitForState(ctx, id, "running", defaultTimeout)

	attach := exec.CommandContext(ctx, cmdmanBin, "attach", id)
	attach.Env = append(
		os.Environ(),
		cmdman.ENV_CMDMAN_DATA_DIR+"="+env.dataHome,
		cmdman.ENV_CMDMAN_RUNTIME_DIR+"="+env.runtimeDir,
	)

	ptmx, err := pty.Start(attach)
	if err != nil {
		t.Fatalf("start attach pty: %v", err)
	}
	defer ptmx.Close()

	time.Sleep(300 * time.Millisecond)
	if _, err := ptmx.Write([]byte{0x10, 0x11}); err != nil {
		t.Fatalf("send detach keys: %v", err)
	}

	waitAttachExit(t, attach, 3*time.Second)
	env.waitForState(ctx, id, "running", defaultTimeout)
}

func TestAttach_ExitsWhenCommandStopsFromCtrlC(t *testing.T) {
	t.Parallel()
	ctx := testContext(t)
	env := newTestEnv(t)

	id := env.run(ctx, "run", "-n", "attach-sigint", "--", "/bin/sh", "-c", "sleep 300")
	env.waitForState(ctx, id, "running", defaultTimeout)

	attach := exec.CommandContext(ctx, cmdmanBin, "attach", id)
	attach.Env = append(
		os.Environ(),
		cmdman.ENV_CMDMAN_DATA_DIR+"="+env.dataHome,
		cmdman.ENV_CMDMAN_RUNTIME_DIR+"="+env.runtimeDir,
	)

	ptmx, err := pty.Start(attach)
	if err != nil {
		t.Fatalf("start attach pty: %v", err)
	}
	defer ptmx.Close()

	time.Sleep(300 * time.Millisecond)
	if _, err := ptmx.Write([]byte{0x03}); err != nil {
		t.Fatalf("send ctrl-c: %v", err)
	}

	waitAttachExit(t, attach, 3*time.Second)
	env.waitForState(ctx, id, "exited", defaultTimeout)
}

func waitAttachExit(t *testing.T, cmd *exec.Cmd, timeout time.Duration) {
	t.Helper()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("attach exited with error: %v", err)
		}
	case <-time.After(timeout):
		t.Fatal("attach did not exit")
	}
}
