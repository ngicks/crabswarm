package commands

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	crabpreviewv1 "github.com/ngicks/crabswarm/api/gen/proto/go/crabpreview/v1"
	"github.com/ngicks/crabswarm/crabswarm"
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

// completePreviewRootDir completes the optional ROOT positional of `preview`
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

// completePreviewRoots completes the NAME|ID positional of `preview remove` with
// the roots the running daemon reports. It is best-effort: any failure (daemon
// down, config error) degrades to no suggestions rather than surfacing an error,
// and the RPC is bounded by a short timeout so completion never hangs the shell.
func completePreviewRoots(
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

	resp, err := preview.NewClient(pcfg.Addr).
		ListRoots(ctx, connect.NewRequest(&crabpreviewv1.ListRootsRequest{}))
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(resp.Msg.GetRoots()))
	for _, r := range resp.Msg.GetRoots() {
		names = append(names, r.GetName())
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}
