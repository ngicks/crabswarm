package commands

import (
	"fmt"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
	previewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1"
	"github.com/ngicks/crabswarm/crabswarm/cli"
	issuescli "github.com/ngicks/crabswarm/crabswarm/issues/cli"
	"github.com/ngicks/crabswarm/crabswarm/preview"
)

// previewRemoveCmd wires `preview remove NAME|ID`: a thin ConnectRPC client
// that drops a file root (and its watches) or an issue source from the running
// daemon, whichever the argument names.
func previewRemoveCmd(parent *cobra.Command, flagConfig *string) {
	var flagAddr string

	cmd := &cobra.Command{
		Use:               "remove NAME|ID",
		Aliases:           []string{"rm"},
		Short:             "Remove a root or an issue source from the running preview daemon",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePreviewRegistrations,
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

	// Both registries are listed first so the argument is matched against every
	// ID, root name and issue prefix at once. The daemon cannot do it: a source
	// is removed by ID while the name a user reads off `preview list` is its
	// prefix, and neither registry sees the other, so an argument naming a root
	// and a source alike looks unambiguous from inside either one.
	regs, err := previewRegistrations(ctx, pcfg)
	if err != nil {
		return err
	}
	reg, err := issuescli.ResolveRegistration(regs, args[0])
	if err != nil {
		return err
	}

	if reg.Kind == issuescli.KindSource {
		_, err = preview.NewIssuesClient(pcfg.Addr).RemoveSource(ctx,
			connect.NewRequest(&issuesv1.RemoveSourceRequest{SourceId: reg.ID}))
	} else {
		_, err = preview.NewClient(pcfg.Addr).RemoveRoot(ctx,
			connect.NewRequest(&previewv1.RemoveRootRequest{RootId: reg.ID}))
	}
	if err != nil {
		return cli.PreviewDaemonError(err, pcfg.DaemonName)
	}

	// Stdout, like every other preview subcommand's result: cobra's Print family
	// writes to stderr unless an out writer is installed.
	fmt.Fprintf(cmd.OutOrStdout(), "removed %s %s\n", reg.Kind, reg.Name)
	return nil
}
