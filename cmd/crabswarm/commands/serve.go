package commands

import (
	"log/slog"
	"net"
	"os"
	"path/filepath"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/claude_hook/v1"
	impl "github.com/ngicks/crabswarm/pkg/api/impl/proto/go/claude_hook/v1"
	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

// serveCmd is the serve subcommand.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the crabswarm server",
	RunE:  runServeCmd,
}

func runServeCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	logger, _ := contextkey.ValueSlogLogger(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	sockPath := resolveSocketPath(cmd)

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(sockPath), 0o700); err != nil {
		return err
	}

	// Remove stale socket file if it exists.
	if err := os.Remove(sockPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	lis, err := net.Listen("unix", sockPath)
	if err != nil {
		return err
	}
	defer lis.Close()

	logger.Info("server listening", slog.String("socket", sockPath))

	srv := grpc.NewServer()
	pb.RegisterAuditServiceServer(srv, &impl.AuditServiceImpl{Logger: logger})

	// Graceful shutdown when context is cancelled (e.g. SIGINT).
	go func() {
		<-ctx.Done()
		logger.Info("shutting down server")
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}
