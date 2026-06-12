package commands

import "github.com/spf13/cobra"

func hookCmd(parent *cobra.Command, flagSock, flagConfig *string) {
	cmd := &cobra.Command{
		Use:   "hook",
		Short: "Hook management commands",
	}

	hookAuditCmd(cmd, flagSock, flagConfig)
	hookExecCmd(cmd, flagConfig)

	parent.AddCommand(cmd)
}
