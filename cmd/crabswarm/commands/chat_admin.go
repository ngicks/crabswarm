package commands

import "github.com/spf13/cobra"

func chatAdminCmd(parent *cobra.Command, flags *chatFlags) {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Inspect and edit the rooms from outside them (admin)",
		Long: `admin is the host's plane: which rooms exist, who attends them, which team
each member belongs to, and what is said into them.

These verbs are authenticated by the age identity file named by --identity
rather than by a member token, so a participant cannot reshape the rooms it
lives in. The admin attends no room and carries no token to be recognized by,
so every room-scoped verb names its room as its first argument.`,
		Example: `  crabswarm chat admin list --identity ~/.config/crabswarm/chat_admin.key
  crabswarm chat admin send /work/proj backend/alice "ship it" \
    --identity ~/.config/crabswarm/chat_admin.key`,
	}

	chatAdminListCmd(cmd, flags)
	chatAdminRegisterCmd(cmd, flags)
	chatAdminMoveCmd(cmd, flags)
	chatAdminSendCmd(cmd, flags)

	parent.AddCommand(cmd)
}
