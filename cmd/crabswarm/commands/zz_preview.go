package commands

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	issuesv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/issues/v1"
	previewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/preview/v1"
	"github.com/ngicks/crabswarm/crabswarm"
	"github.com/ngicks/crabswarm/crabswarm/cli"
	issuescli "github.com/ngicks/crabswarm/crabswarm/issues/cli"
	"github.com/ngicks/crabswarm/crabswarm/preview"
)

// commandLogger returns the request-scoped logger the root command stashes in
// the context, falling back to the default logger when it is absent.
func commandLogger(cmd *cobra.Command) *slog.Logger {
	if logger, _ := contextkey.ValueSlogLogger(cmd.Context()); logger != nil {
		return logger
	}
	return slog.Default()
}

// resolvePreviewConfig loads the layered crabswarm config (defaults < file <
// env) via flagConfig and returns its preview sub-config, overlaying the
// explicitly-set --addr flag on top so flags win. It is shared by every preview
// subcommand so they agree on which daemon address to use.
func resolvePreviewConfig(cmd *cobra.Command, flagConfig, flagAddr string) (preview.Config, error) {
	cfg, err := crabswarm.LoadConfig(flagConfig)
	if err != nil {
		return preview.Config{}, err
	}
	pcfg := cfg.Preview
	if cmd.Flags().Changed("addr") {
		pcfg.Addr = flagAddr
	}
	return pcfg, nil
}

// previewRegistrations lists everything the daemon at pcfg.Addr has registered
// — file roots and issue sources — as the one view `preview list` prints and
// `preview remove` matches its argument against.
func previewRegistrations(
	ctx context.Context,
	pcfg preview.Config,
) ([]issuescli.Registration, error) {
	roots, err := preview.NewClient(pcfg.Addr).
		ListRoots(ctx, connect.NewRequest(&previewv1.ListRootsRequest{}))
	if err != nil {
		return nil, cli.PreviewDaemonError(err, pcfg.DaemonName)
	}
	sources, err := preview.NewIssuesClient(pcfg.Addr).
		ListSources(ctx, connect.NewRequest(&issuesv1.ListSourcesRequest{}))
	if err != nil {
		return nil, cli.PreviewDaemonError(err, pcfg.DaemonName)
	}

	pbRoots, pbSources := roots.Msg.GetRoots(), sources.Msg.GetSources()
	regs := make([]issuescli.Registration, 0, len(pbRoots)+len(pbSources))
	for _, r := range pbRoots {
		regs = append(regs, issuescli.Registration{
			Kind: issuescli.KindRoot,
			ID:   r.GetId(),
			Name: r.GetName(),
			Path: r.GetPath(),
		})
	}
	for _, s := range pbSources {
		// The .beads path rather than the registering directory: it is what the
		// source is keyed by, so two worktrees of one repository still read as
		// the single source they are.
		regs = append(regs, issuescli.Registration{
			Kind: issuescli.KindSource,
			ID:   s.GetId(),
			Name: s.GetPrefix(),
			Path: s.GetBeadsPath(),
		})
	}
	return regs, nil
}

// completePreviewRootDir completes the optional DIR positional of `preview`
// with directories only; once the single argument is present there is nothing
// more to complete.
func completePreviewRootDir(
	_ *cobra.Command,
	args []string,
	_ string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveFilterDirs
}

// completePreviewRegistrations completes the NAME|ID positional of
// `preview remove` with the root names and issue-source prefixes the running
// daemon reports. It is best-effort: any failure (daemon down, config error)
// degrades to no suggestions rather than surfacing an error, and the RPCs are
// bounded by a short timeout so completion never hangs the shell.
func completePreviewRegistrations(
	cmd *cobra.Command,
	args []string,
	_ string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	flagConfig, _ := cmd.Flags().GetString("config")
	flagAddr, _ := cmd.Flags().GetString("addr")
	pcfg, err := resolvePreviewConfig(cmd, flagConfig, flagAddr)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), time.Second)
	defer cancel()

	regs, err := previewRegistrations(ctx, pcfg)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(regs))
	for _, r := range regs {
		names = append(names, r.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
