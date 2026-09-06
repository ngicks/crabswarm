package commands

import (
	"github.com/spf13/cobra"

	issuescli "github.com/ngicks/crabswarm/crabswarm/issues/cli"
)

// previewListCmd wires `preview list`: a thin ConnectRPC client that prints the
// file roots and issue sources the running daemon has registered.
func previewListCmd(parent *cobra.Command, flagConfig *string) {
	var flagAddr string

	cmd := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls"},
		Short:             "List the roots and issue sources registered with the preview daemon",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreviewList(cmd, args, *flagConfig, flagAddr)
		},
	}

	cmd.Flags().StringVar(&flagAddr, "addr", "",
		"TCP address of the preview daemon to query (default from config)")

	parent.AddCommand(cmd)
}

func runPreviewList(cmd *cobra.Command, _ []string, flagConfig, flagAddr string) error {
	ctx := cmd.Context()

	pcfg, err := resolvePreviewConfig(cmd, flagConfig, flagAddr)
	if err != nil {
		return err
	}

	regs, err := previewRegistrations(ctx, pcfg)
	if err != nil {
		return err
	}
	return issuescli.RenderRegistrations(cmd.OutOrStdout(), regs)
}
