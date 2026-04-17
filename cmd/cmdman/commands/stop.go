package commands

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringArrayP("label", "l", nil, "Target commands matching labels")
	stopCmd.Flags().StringP("signal", "s", "SIGTERM", "Signal to send")
}

var stopCmd = &cobra.Command{
	Use:   "stop [flags] [ID|NAME]",
	Short: "Send signal to a running command",
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	sigName, _ := cmd.Flags().GetString("signal")
	labelSlice, _ := cmd.Flags().GetStringArray("label")

	sig := parseSignal(sigName)
	labels, err := parseLabels(labelSlice)
	if err != nil {
		return err
	}

	svc, err := cmdmanService()
	if err != nil {
		return err
	}

	results, err := svc.Stop(cmd.Context(), cmdman.StopRequest{
		Targets: args,
		Labels:  labels,
		Signal:  sig,
	})
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.Err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "stop %s: %v\n", result.ID, result.Err)
		}
	}
	return nil
}

func parseSignal(s string) int32 {
	s = strings.ToUpper(s)
	s = strings.TrimPrefix(s, "SIG")
	switch s {
	case "HUP":
		return int32(syscall.SIGHUP)
	case "INT":
		return int32(syscall.SIGINT)
	case "QUIT":
		return int32(syscall.SIGQUIT)
	case "KILL":
		return int32(syscall.SIGKILL)
	case "TERM":
		return int32(syscall.SIGTERM)
	case "USR1":
		return int32(syscall.SIGUSR1)
	case "USR2":
		return int32(syscall.SIGUSR2)
	default:
		return int32(syscall.SIGTERM)
	}
}
