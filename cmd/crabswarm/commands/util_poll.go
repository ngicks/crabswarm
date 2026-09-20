package commands

import "github.com/spf13/cobra"

func utilPollCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "poll",
		Short: "Probe a target until it becomes ready",
	}

	utilPollWaitCmd(cmd)

	parent.AddCommand(cmd)
}
