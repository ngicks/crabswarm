package commands

import (
	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	previewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1"
	"github.com/ngicks/crabswarm/crabswarm/cli"
	"github.com/ngicks/crabswarm/crabswarm/preview"
)

// previewListCmd wires `preview list`: a thin ConnectRPC client that prints the
// roots the running daemon has registered.
func previewListCmd(parent *cobra.Command, flagConfig *string) {
	var flagAddr string

	cmd := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls"},
		Short:             "List the roots registered with the running preview daemon",
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

	client := preview.NewClient(pcfg.Addr)
	resp, err := client.ListRoots(ctx, connect.NewRequest(&previewv1.ListRootsRequest{}))
	if err != nil {
		return cli.PreviewDaemonError(err, pcfg.DaemonName)
	}

	roots := make([]cli.PreviewRoot, 0, len(resp.Msg.GetRoots()))
	for _, r := range resp.Msg.GetRoots() {
		roots = append(roots, cli.PreviewRoot{ID: r.GetId(), Name: r.GetName(), Path: r.GetPath()})
	}
	return cli.RenderRoots(cmd.OutOrStdout(), roots)
}
