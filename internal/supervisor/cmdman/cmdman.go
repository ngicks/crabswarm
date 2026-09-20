// Package cmdman implements supervisor.Supervisor on top of the cmdman
// executable.
package cmdman

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ngicks/crabswarm/internal/supervisor"
)

// defaultBin is the executable Cmdman shells out to. It is expected on PATH.
const defaultBin = "cmdman"

// Cmdman drives commands through the cmdman executable.
type Cmdman struct {
	bin string
}

var _ supervisor.Supervisor = (*Cmdman)(nil)

// New returns a Cmdman that shells out to bin. An empty bin resolves "cmdman"
// on PATH.
func New(bin string) *Cmdman {
	if bin == "" {
		bin = defaultBin
	}
	return &Cmdman{bin: bin}
}

// Inspect reports the state cmdman prints for the name, e.g. "running" or
// "exited".
//
// Any failure reads as supervisor.ErrNotFound: cmdman exits non-zero for a name
// it does not know, and a cmdman broken for any other reason reports it when Run
// runs next. The exec error is wrapped alongside, so a caller logging the error
// still sees what actually went wrong.
func (c *Cmdman) Inspect(ctx context.Context, name string) (supervisor.State, error) {
	out, err := exec.CommandContext(ctx, c.bin, "inspect", name, "--format", "{{.State}}").
		Output()
	if err != nil {
		return "", fmt.Errorf("cmdman inspect %q: %w: %w", name, supervisor.ErrNotFound, err)
	}
	return supervisor.State(strings.TrimSpace(string(out))), nil
}

// Remove drops a cmdman command so its name is free again.
func (c *Cmdman) Remove(ctx context.Context, name string) error {
	if out, err := exec.CommandContext(ctx, c.bin, "rm", name).CombinedOutput(); err != nil {
		return fmt.Errorf("cmdman rm %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Run starts command under cmdman. An empty name starts an anonymous command.
func (c *Cmdman) Run(ctx context.Context, name string, command []string) error {
	args := []string{"run"}
	if name != "" {
		args = append(args, "--name", name)
	}
	// The separator keeps cmdman from reading the supervised command's own flags
	// as its own.
	args = append(args, "--")
	args = append(args, command...)

	if out, err := exec.CommandContext(ctx, c.bin, args...).CombinedOutput(); err != nil {
		return fmt.Errorf("cmdman run %q: %w: %s",
			describe(name, command), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// describe labels a run in error messages: the name when the caller gave one,
// the command line otherwise.
func describe(name string, command []string) string {
	if name != "" {
		return name
	}
	return strings.Join(command, " ")
}
