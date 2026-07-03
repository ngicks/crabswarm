package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	crabpreviewv1 "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/crabpreview/v1"
	"github.com/ngicks/crabswarm/pkg/crabswarm/cli"
	"github.com/ngicks/crabswarm/pkg/crabswarm/preview"
)

func previewCmd(parent *cobra.Command, flagConfig *string) {
	var flagAddr string

	cmd := &cobra.Command{
		Use:   "preview [ROOT]",
		Short: "Preview GitHub-flavored markdown in a browser",
		Long: `preview adds ROOT (default the current directory) to the browser-based
markdown previewer and prints the URL to open. If the preview daemon is not
already running it is started under cmdman; subsequent invocations just add
another root to the running daemon.

The server listens on preview.addr (default 0.0.0.0:6419) so phones and tablets
on the tailnet can reach it — the printed URL uses this machine's hostname for
wildcard binds. Because it exposes the registered roots' file contents to
anyone who can reach the port, set preview.addr to 127.0.0.1:6419 on untrusted
networks.`,
		Example: `  crabswarm preview
  crabswarm preview ./docs
  crabswarm preview --addr 127.0.0.1:6419 .`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completePreviewRootDir,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreview(cmd, args, *flagConfig, flagAddr)
		},
	}

	cmd.Flags().StringVar(&flagAddr, "addr", "",
		"TCP listen address override (default from config, e.g. 0.0.0.0:6419)")

	previewServeCmd(cmd, flagConfig)
	previewListCmd(cmd, flagConfig)
	previewRemoveCmd(cmd, flagConfig)

	parent.AddCommand(cmd)
}

func runPreview(cmd *cobra.Command, args []string, flagConfig, flagAddr string) error {
	ctx := cmd.Context()
	logger := commandLogger(cmd)

	root := "."
	if len(args) > 0 {
		root = args[0]
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("preview root %q: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("preview root %q is not a directory", abs)
	}

	pcfg, err := resolvePreviewConfig(cmd, flagConfig, flagAddr)
	if err != nil {
		return err
	}

	// Bring the daemon up (or confirm it is healthy) before adding the root:
	// EnsureDaemon starts `preview __serve` under cmdman with the same --config
	// and polls /healthz.
	if err := preview.EnsureDaemon(ctx, logger, pcfg, flagConfig); err != nil {
		return err
	}

	client := preview.NewClient(pcfg.Addr)
	resp, err := client.AddRoot(ctx, connect.NewRequest(&crabpreviewv1.AddRootRequest{Path: abs}))
	if err != nil {
		return cli.PreviewDaemonError(err, pcfg.DaemonName)
	}

	fmt.Fprintln(cmd.OutOrStdout(), cli.PreviewURL(pcfg.Addr, resp.Msg.GetRoot().GetId()))
	return nil
}
