package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	"github.com/spf13/cobra"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
	previewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1"
	"github.com/ngicks/crabswarm/crabswarm/cli"
	"github.com/ngicks/crabswarm/crabswarm/preview"
)

func previewCmd(parent *cobra.Command, flagConfig *string) {
	var (
		flagAddr  string
		flagRoot  bool
		flagIssue bool
	)

	cmd := &cobra.Command{
		Use:   "preview [DIR]",
		Short: "Preview GitHub-flavored markdown and beads issues in a browser",
		Long: `preview registers DIR (default the current directory) with the browser-based
previewer and prints the URL to open. DIR becomes a file root and, when a beads
database governs it, an issues source; --root and --issue register only one of
the two. If the preview daemon is not already running it is started under
cmdman; subsequent invocations just register another directory with the running
daemon.

The server listens on preview.addr (default 0.0.0.0:6419) so phones and tablets
on the tailnet can reach it — the printed URL uses this machine's hostname for
wildcard binds. Because it exposes the registered roots' file contents to
anyone who can reach the port, set preview.addr to 127.0.0.1:6419 on untrusted
networks.`,
		Example: `  crabswarm preview
  crabswarm preview ./docs
  crabswarm preview --root ./docs
  crabswarm preview --issue .
  crabswarm preview --addr 127.0.0.1:6419 .`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completePreviewRootDir,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreview(cmd, args, *flagConfig, flagAddr, flagRoot, flagIssue)
		},
	}

	cmd.Flags().StringVar(&flagAddr, "addr", "",
		"TCP listen address override (default from config, e.g. 0.0.0.0:6419)")
	cmd.Flags().BoolVar(&flagRoot, "root", false,
		"register DIR as a file root only, leaving its issues unregistered")
	cmd.Flags().BoolVar(&flagIssue, "issue", false,
		"register DIR as an issues source only; fails when no beads database governs it")
	cmd.MarkFlagsMutuallyExclusive("root", "issue")

	previewServeCmd(cmd, flagConfig)
	previewListCmd(cmd, flagConfig)
	previewRemoveCmd(cmd, flagConfig)

	parent.AddCommand(cmd)
}

func runPreview(
	cmd *cobra.Command,
	args []string,
	flagConfig, flagAddr string,
	flagRoot, flagIssue bool,
) error {
	ctx := cmd.Context()
	logger := commandLogger(cmd)

	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("preview directory %q: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("preview directory %q is not a directory", abs)
	}

	pcfg, err := resolvePreviewConfig(cmd, flagConfig, flagAddr)
	if err != nil {
		return err
	}

	// Bring the daemon up (or confirm it is healthy) before registering
	// anything: EnsureDaemon starts `preview __serve` under cmdman with the same
	// --config and polls /healthz.
	if err := preview.EnsureDaemon(ctx, logger, pcfg, flagConfig); err != nil {
		return err
	}

	// Neither flag means both, so each side registers unless the other flag
	// asked for it alone.
	if !flagIssue {
		if err := addPreviewRoot(ctx, cmd, pcfg, abs); err != nil {
			return err
		}
	}
	if !flagRoot {
		if err := addPreviewSource(ctx, cmd, logger, pcfg, abs, flagIssue); err != nil {
			return err
		}
	}
	return nil
}

// addPreviewRoot registers abs as a file root and prints the URL that opens it.
func addPreviewRoot(
	ctx context.Context,
	cmd *cobra.Command,
	pcfg preview.Config,
	abs string,
) error {
	resp, err := preview.NewClient(pcfg.Addr).
		AddRoot(ctx, connect.NewRequest(&previewv1.AddRootRequest{Path: abs}))
	if err != nil {
		return cli.PreviewDaemonError(err, pcfg.DaemonName)
	}
	fmt.Fprintln(cmd.OutOrStdout(), cli.PreviewURL(pcfg.Addr, resp.Msg.GetRoot().GetId()))
	return nil
}

// addPreviewSource registers the beads database governing abs as an issues
// source and prints its prefix and ID.
//
// required says what a directory outside every beads workspace means. The
// daemon answers NotFound for one; with --issue the user asked for the source
// by itself, so that is the error they get, while the default registration
// treats it as a directory that simply has no issues to serve.
func addPreviewSource(
	ctx context.Context,
	cmd *cobra.Command,
	logger *slog.Logger,
	pcfg preview.Config,
	abs string,
	required bool,
) error {
	resp, err := preview.NewIssuesClient(pcfg.Addr).
		AddSource(ctx, connect.NewRequest(&issuesv1.AddSourceRequest{Dir: abs}))
	if err != nil {
		if !required && connect.CodeOf(err) == connect.CodeNotFound {
			logger.Debug("preview: no beads database, registering no issues source",
				"dir", abs, "err", err)
			return nil
		}
		return cli.PreviewDaemonError(err, pcfg.DaemonName)
	}
	src := resp.Msg.GetSource()
	fmt.Fprintf(cmd.OutOrStdout(), "issue source %s (%s)\n", src.GetPrefix(), src.GetId())
	return nil
}
