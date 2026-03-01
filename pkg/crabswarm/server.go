package crabswarm

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"syscall"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/claude_hook/v1"
	impl "github.com/ngicks/crabswarm/pkg/api/impl/proto/go/claude_hook/v1"
	"google.golang.org/grpc"
)

// Server is the crabswarm server.
type Server struct {
	logger   *slog.Logger
	sockPath string
}

// NewServer returns a new Server.
func NewServer(
	logger *slog.Logger,
	sockPath string,
) *Server {
	return &Server{
		logger:   logger,
		sockPath: sockPath,
	}
}

func listenUnixDomainSocket(sockPath string) (net.Listener, error) {
	// Ensure parent directory exists.
	err := os.MkdirAll(filepath.Dir(sockPath), 0o700)
	if err != nil {
		return nil, err
	}

	// Remove stale socket file if it exists.
	err = os.Remove(sockPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return net.Listen("unix", sockPath)
}

func (s *Server) listen() (net.Listener, error) {
	if s.sockPath != "" {
		return listenUnixDomainSocket(s.sockPath)
	}

	return nil, fmt.Errorf("server listen target not specified")
}

func (s *Server) Serve(ctx context.Context) error {
	// Acquire exclusive lock on <sockPath>.lock to prevent duplicate servers.
	lockPath := s.sockPath + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer lockFile.Close()

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("server already running (lock held on %s)", lockPath)
	}
	defer syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN)

	lis, err := s.listen()
	if err != nil {
		return err
	}
	defer lis.Close()

	s.logger.Info(
		"server listening",
		slog.String("addr", lis.Addr().String()),
	)

	srv := grpc.NewServer()
	pb.RegisterAuditServiceServer(srv, &impl.AuditServiceImpl{Logger: s.logger})

	// Graceful shutdown when context is cancelled (e.g. SIGINT).
	go func() {
		<-ctx.Done()
		s.logger.Info("shutting down server")
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}
