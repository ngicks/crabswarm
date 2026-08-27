package commands

import (
	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	previewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1"
	"github.com/ngicks/crabswarm/crabswarm/cli"
	"github.com/ngicks/crabswarm/crabswarm/preview"
)

// previewRemoveCmd wires `preview remove NAME|ID`: a thin ConnectRPC client that
// drops a root (and its watches) from the running daemon.
func previewRemoveCmd(parent *cobra.Command, flagConfig *string) {
	var flagAddr string

	cmd := &cobra.Command{
		Use:               "remove NAME|ID",
		Aliases:           []string{"rm"},
		Short:             "Remove a root from the running preview daemon",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePreviewRoots,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreviewRemove(cmd, args, *flagConfig, flagAddr)
		},
	}

	cmd.Flags().StringVar(&flagAddr, "addr", "",
		"TCP address of the preview daemon to query (default from config)")

	parent.AddCommand(cmd)
}

func runPreviewRemove(cmd *cobra.Command, args []string, flagConfig, flagAddr string) error {
	ctx := cmd.Context()

	pcfg, err := resolvePreviewConfig(cmd, flagConfig, flagAddr)
	if err != nil {
		return err
	}

	client := preview.NewClient(pcfg.Addr)
	_, err = client.RemoveRoot(ctx, connect.NewRequest(&previewv1.RemoveRootRequest{
		RootId: args[0],
	}))
	if err != nil {
		return cli.PreviewDaemonError(err, pcfg.DaemonName)
	}

	cmd.Printf("removed %s\n", args[0])
	return nil
}
