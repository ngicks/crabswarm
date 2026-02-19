package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

// serveCmd is the serve subcommand.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the crabhook server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("serve called")
		return nil
	},
}
