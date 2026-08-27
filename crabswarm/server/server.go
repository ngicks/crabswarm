package server

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

	pb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/hook/v1"
	"google.golang.org/grpc"
)

// Server is the crabswarm server.
type Server struct {
	logger   *slog.Logger
	sockPath string
}

type auditServiceServer struct {
	pb.UnimplementedAuditServiceServer
	logger *slog.Logger
}

// New returns a new Server.
func New(
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
	pb.RegisterAuditServiceServer(srv, &auditServiceServer{logger: s.logger})

	// Graceful shutdown when context is cancelled (e.g. SIGINT).
	go func() {
		<-ctx.Done()
		s.logger.Info("shutting down server")
		srv.GracefulStop()
	}()

	return srv.Serve(lis)
}

func (s *auditServiceServer) ReportHookInputEvent(
	ctx context.Context,
	req *pb.ReportHookInputEventRequest,
) (*pb.ReportHookInputEventResponse, error) {
	attrs := []any{}
	if req.GetTimestamp() != nil {
		attrs = append(
			attrs,
			slog.String(
				"timestamp",
				req.GetTimestamp().AsTime().Format("2006-01-02T15:04:05.999999999Z07:00"),
			),
		)
	}
	if req.GetHookInput() != nil {
		attrs = append(attrs, slog.String("hook_input", string(req.GetHookInput())))
	}
	s.logger.InfoContext(ctx, "audit hook input event", attrs...)
	return &pb.ReportHookInputEventResponse{}, nil
}
