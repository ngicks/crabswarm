package commands

import "github.com/spf13/cobra"

func init() {
	rootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage the crabswarm server",
}
