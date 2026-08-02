package commands

import (
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/preview"
	"github.com/ngicks/crabswarm/web"
)

// previewServeCmd wires the hidden `preview __serve` command. It is the process
// cmdman launches for the daemon (see preview.EnsureDaemon / daemon.go): the
// contract is `<self> preview __serve --addr <addr> [--config <path>]`, so this
// command MUST accept exactly those flags. It runs the previewer in the
// foreground and cmdman owns daemonization, logs and restart; it is Hidden
// because users never invoke it directly.
func previewServeCmd(parent *cobra.Command, flagConfig *string) {
	var flagAddr string

	cmd := &cobra.Command{
		Use:               "__serve",
		Short:             "Run the preview HTTP server in the foreground (daemon entry point)",
		Hidden:            true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPreviewServe(cmd, args, *flagConfig, flagAddr)
		},
	}

	cmd.Flags().StringVar(&flagAddr, "addr", "",
		"TCP listen address override (default from config, e.g. 0.0.0.0:6419)")

	parent.AddCommand(cmd)
}

func runPreviewServe(cmd *cobra.Command, _ []string, flagConfig, flagAddr string) error {
	ctx := cmd.Context()
	logger := commandLogger(cmd)

	pcfg, err := resolvePreviewConfig(cmd, flagConfig, flagAddr)
	if err != nil {
		return err
	}

	svc, err := preview.New(logger, pcfg)
	if err != nil {
		return err
	}
	// Inject the embedded (or dev-FS) SPA before serving; SetStaticFS must be
	// called before Serve.
	svc.SetStaticFS(web.FS())

	// ctx is the signal-aware context main installs (cmdsignals.NotifyContext);
	// Serve blocks until it is cancelled, then shuts the HTTP server down.
	return svc.Serve(ctx)
}
