// Package supervisor starts a command under a process supervisor and, when a
// readiness target is given, waits until that target answers. The supervisor
// itself is an interface; internal/supervisor/cmdman implements it.
package supervisor

import (
	"context"
	"errors"
	"log/slog"

	"github.com/ngicks/crabswarm/pkg/util/poll"
)

// State is the lifecycle state a supervisor reports for a command. Start only
// tells StateRunning from everything else, so no other state is spelled out
// here: anything else belongs to a command that is no longer live and whose
// name Start may reclaim.
type State string

// StateRunning is the state a supervisor reports for a live command.
const StateRunning State = "running"

// ErrNotFound reports that the supervisor knows no command under the given
// name. An Inspect error matching it is what Start reads as "nothing to
// reclaim".
var ErrNotFound = errors.New("supervisor: command not found")

// Supervisor is the process supervisor Start drives.
type Supervisor interface {
	// Inspect reports the state of the named command, or an error matching
	// ErrNotFound when the supervisor knows no such name.
	Inspect(ctx context.Context, name string) (State, error)
	// Remove drops a command so its name is free again.
	Remove(ctx context.Context, name string) error
	// Run starts command, program first, under the given name. An empty name
	// starts an anonymous command the supervisor names for itself.
	Run(ctx context.Context, name string, command []string) error
}

// StartOption configures Start.
type StartOption struct {
	// Name is the supervisor-side name of the command. An empty Name starts an
	// anonymous command, which is a new command on every call.
	Name string
	// Command is the argv to start, program first. Start rejects an empty Command.
	Command []string
	// Poll is the readiness target to wait for once the command is started. Nil
	// returns as soon as the supervisor accepted the command.
	Poll *poll.Target
	// PollOptions tunes that wait. It is ignored while Poll is nil.
	PollOptions poll.WaitOption
}

// Start starts opt.Command under sup and returns once the command is started
// and the readiness target, when one is given, answers. logger may be nil.
func Start(ctx context.Context, logger *slog.Logger, sup Supervisor, opt StartOption) error {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if len(opt.Command) == 0 {
		return errors.New("no command to run")
	}

	if err := start(ctx, logger, sup, opt.Name, opt.Command); err != nil {
		return err
	}
	if opt.Poll == nil {
		return nil
	}
	// poll.Wait names the target in its errors already, so its error travels up
	// unwrapped.
	return poll.Wait(ctx, logger, *opt.Poll, opt.PollOptions)
}

// start hands the command to the supervisor, leaving an already running one
// alone.
//
// The inspect / remove dance exists because starting under a fixed name needs
// that name to be free: a command that exited keeps its name, so rerunning
// under it fails until the old entry is dropped. An anonymous command owns no
// name to collide with and goes straight to Run.
func start(
	ctx context.Context,
	logger *slog.Logger,
	sup Supervisor,
	name string,
	command []string,
) error {
	if name != "" {
		state, err := sup.Inspect(ctx, name)
		switch {
		case err == nil && state == StateRunning:
			logger.Debug("supervised command already running", "name", name)
			return nil
		case err == nil:
			logger.Debug("removing stale supervised command", "name", name, "state", state)
			// Best-effort: the Run below reports the real problem if the name is
			// still taken, and a remove that failed for any other reason does not
			// stop this start.
			if err := sup.Remove(ctx, name); err != nil {
				logger.Debug("removing stale supervised command failed",
					"name", name, "error", err)
			}
		default:
			// Every inspect failure, ErrNotFound or not, reads as "no command to
			// reclaim": there is nothing to remove, and a supervisor broken for any
			// other reason reports it when Run follows.
			logger.Debug("supervisor inspect: treating command as absent",
				"name", name, "error", err)
		}
	}

	logger.Debug("starting supervised command", "name", name, "command", command)
	return sup.Run(ctx, name, command)
}
