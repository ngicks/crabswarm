package commands

import "github.com/spf13/cobra"

func hookCmd(parent *cobra.Command, flagSock *string) {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Hook management commands",
	}

	hookAuditCmd(cmd, flagSock)
	hookAutoApproveCmd(cmd)

	parent.AddCommand(cmd)
}
