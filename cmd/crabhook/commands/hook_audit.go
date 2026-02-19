package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	hookCmd.AddCommand(hookAuditCmd)
}

// hookAuditCmd is the audit subcommand under hook.
var hookAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit hook events",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("hook audit called")
		return nil
	},
}
