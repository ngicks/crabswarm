package commands

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/internal/supervisor"
	"github.com/ngicks/crabswarm/internal/supervisor/cmdman"
	"github.com/ngicks/crabswarm/pkg/util/poll"
)

func utilSupervisedCmd(parent *cobra.Command) {
	var (
		flagSupervisor      string
		flagName            string
		flagPoll            string
		flagPollStartPeriod time.Duration
		flagPollInterval    time.Duration
		flagPollRetries     int
	)

	cmd := &cobra.Command{
		Use:   "supervised [flags] -- COMMAND [ARGS...]",
		Short: "Start a command under a supervisor and optionally wait until it is ready",
		Long: `supervised hands COMMAND to a process supervisor and returns once the
supervisor accepted it. cmdman is the only supervisor today.

--name gives the supervised command a stable name. A named command already
running is left alone and a finished one is dropped before the new run, so
rerunning the same command is safe. Without --name every run starts a new
anonymous command.

--poll waits for a readiness target once the command is started, and also when
an already running --name command was adopted instead of started. The target is
a unix socket path (bare or unix://), a file://PATH or an http(s):// URL that
answers 2xx. --poll-start-period delays the first probe, so a command known to
need a moment is left alone until it elapses; --poll-interval and
--poll-retries carry the meaning the Docker healthcheck flags of the same names
have, and every failed probe counts toward --poll-retries.

COMMAND must follow "--", so its own flags are never parsed as this command's.`,
		Example: `  crabswarm util supervised -- npm run dev
  crabswarm util supervised --name web --poll http://127.0.0.1:5173 -- npm run dev
  crabswarm util supervised --name api --poll unix:///tmp/api.sock -- ./api --sock /tmp/api.sock`,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: utilSupervisedCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUtilSupervised(
				cmd, args,
				flagSupervisor, flagName, flagPoll,
				flagPollStartPeriod, flagPollInterval, flagPollRetries,
			)
		},
	}

	f := cmd.Flags()
	f.StringVar(&flagSupervisor, "supervisor", "cmdman",
		`Supervisor that owns the command (only "cmdman" today)`)
	f.StringVar(&flagName, "name", "",
		"Supervisor-side name of the command (empty starts an anonymous command)")
	f.StringVar(&flagPoll, "poll", "",
		"Readiness target to wait for after starting: a unix socket path, "+
			"file://PATH or an http(s):// URL (empty starts without waiting)")
	f.DurationVar(&flagPollStartPeriod, "poll-start-period", 0,
		"Delay before the first readiness probe (nothing is probed until it elapses)")
	f.DurationVar(&flagPollInterval, "poll-interval", time.Second,
		"Cadence of readiness probes")
	f.IntVar(&flagPollRetries, "poll-retries", 10,
		"Consecutive failed probes at which the wait gives up (0 also means 10, "+
			"negative retries until cancelled)")

	parent.AddCommand(cmd)
}

// utilSupervisedCompletion leaves every positional to the shell: the first one
// is a program to run and the rest are its own arguments, neither of which this
// command can enumerate.
func utilSupervisedCompletion(
	_ *cobra.Command,
	_ []string,
	_ string,
) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveDefault
}

func runUtilSupervised(
	cmd *cobra.Command, args []string,
	supervisorName, name, pollTarget string,
	pollStartPeriod, pollInterval time.Duration, pollRetries int,
) error {
	// A dash at any other position means the command was written without "--"
	// or mixed with our own flags; either way pflag may have eaten a flag meant
	// for the supervised command.
	if cmd.ArgsLenAtDash() != 0 {
		return errors.New(
			`the command to supervise must follow "--", ` +
				`e.g. crabswarm util supervised --name web -- npm run dev`)
	}

	var sup supervisor.Supervisor
	switch supervisorName {
	case "", "cmdman":
		sup = cmdman.New("")
	default:
		// Reject before spawning anything, so a typo cannot start a command under
		// the wrong supervisor.
		return fmt.Errorf("unsupported supervisor %q", supervisorName)
	}

	opt := supervisor.StartOption{
		Name:    name,
		Command: args,
		PollOptions: poll.WaitOption{
			StartPeriod: pollStartPeriod,
			Interval:    pollInterval,
			Retries:     pollRetries,
		},
	}
	if pollTarget != "" {
		target, err := poll.ParseTarget(pollTarget)
		if err != nil {
			return err
		}
		opt.Poll = &target
	}

	return supervisor.Start(cmd.Context(), commandLogger(cmd), sup, opt)
}
