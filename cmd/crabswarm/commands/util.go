package commands

import "github.com/spf13/cobra"

func utilCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "util",
		Short: "Utility verbs for scripts and hook commands",
	}

	utilPollCmd(cmd)

	parent.AddCommand(cmd)
}
