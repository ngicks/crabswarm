package commands

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/pkg/util/poll"
)

func utilPollWaitCmd(parent *cobra.Command) {
	var (
		flagStartPeriod time.Duration
		flagInterval    time.Duration
		flagRetries     int
	)

	cmd := &cobra.Command{
		Use:   "wait <uri>",
		Short: "Wait until a unix socket, a file or an HTTP endpoint is ready",
		Long: `wait probes <uri> until it is ready and then exits 0, so a script can order
start-up around a daemon it just launched.

The scheme selects the probe:

  <path> or unix://<path>   the unix socket accepts a connection
  file://<path>             the path exists on disk
  http://... https://...    a GET answers 2xx

--interval and --retries carry the meaning the Docker healthcheck flags of the
same names have: probes follow one another every --interval, and --retries
consecutive failures end the wait. --start-period is a delay before the first
probe. Nothing is probed until it elapses, and every failed probe counts toward
--retries.`,
		Example: `  crabswarm util poll wait /run/user/1000/crabswarm/default.sock
  crabswarm util poll wait unix://./run/daemon.sock --interval 200ms
  crabswarm util poll wait file://./dist/index.html
  crabswarm util poll wait http://127.0.0.1:6419/healthz --start-period 5s`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: utilPollWaitCompletion,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUtilPollWait(cmd, args, flagStartPeriod, flagInterval, flagRetries)
		},
	}

	cmd.Flags().DurationVar(&flagStartPeriod, "start-period", 0,
		"delay before the first probe; nothing is probed until it elapses")
	cmd.Flags().DurationVar(&flagInterval, "interval", time.Second,
		"cadence of probes; a probe that takes longer than this is followed at once")
	cmd.Flags().IntVar(&flagRetries, "retries", 10,
		"consecutive failed probes at which the wait gives up; "+
			"0 selects the default, negative means unlimited, waiting until interrupted")

	parent.AddCommand(cmd)
}

// utilPollWaitCompletion completes the target with default file completion,
// because a bare target is a socket path on disk, and offers nothing for a
// second argument, because wait takes exactly one.
func utilPollWaitCompletion(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveDefault
}

func runUtilPollWait(
	cmd *cobra.Command,
	args []string,
	startPeriod, interval time.Duration,
	retries int,
) error {
	target, err := poll.ParseTarget(args[0])
	if err != nil {
		return err
	}
	return poll.Wait(cmd.Context(), commandLogger(cmd), target, poll.WaitOption{
		StartPeriod: startPeriod,
		Interval:    interval,
		Retries:     retries,
	})
}
