// Package server hosts the crabswarm daemon: one gRPC server on a Unix socket
// serving the hook audit service beside both halves of the chat broker, held to
// a single instance per socket by a lock file next to it.
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
	"strings"
	"syscall"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	pb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/hook/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
	"github.com/ngicks/crabswarm/crabswarm/chat/notify"
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
	"google.golang.org/grpc"
)

// Server is the crabswarm server.
type Server struct {
	logger   *slog.Logger
	sockPath string
	chatCfg  chat.Config
}

type auditServiceServer struct {
	pb.UnimplementedAuditServiceServer
	logger *slog.Logger
}

// New returns a new Server. chatCfg configures the chat broker the server
// hosts beside the audit service.
func New(
	logger *slog.Logger,
	sockPath string,
	chatCfg chat.Config,
) *Server {
	return &Server{
		logger:   logger,
		sockPath: sockPath,
		chatCfg:  chatCfg,
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

// openChatStore opens the SQLite store backing the chat broker, creating its
// directory the way the socket's is created. It runs after the flock: two
// daemons writing one chat database is exactly what the lock prevents.
func (s *Server) openChatStore(ctx context.Context) (*chat.Store, error) {
	if s.chatCfg.Db == "" {
		return nil, fmt.Errorf("chat db path not specified")
	}
	path, err := expandHome(s.chatCfg.Db)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return chat.NewStore(ctx, path)
}

// expandHome resolves a leading "~" against the user's home directory. The
// config layers keep paths as they were written so the `config` subcommand
// prints them back unchanged; expansion belongs here, where the path is opened.
func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("expanding %q: %w", path, err)
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~")), nil
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

	chatStore, err := s.openChatStore(ctx)
	if err != nil {
		return err
	}
	defer chatStore.Close()

	lis, err := s.listen()
	if err != nil {
		return err
	}
	defer lis.Close()

	s.logger.Info(
		"server listening",
		slog.String("addr", lis.Addr().String()),
	)

	// Built before the listener is served so a misspelled admin recipient stops
	// the daemon here, with the config key named, instead of at whatever later
	// moment the operator first tries an admin call. No recipient at all is not
	// a misspelling: it leaves the admin half with no authenticator, which is
	// what makes it refuse every call with "configure a key first".
	var adminAuth chat.AdminAuthenticator
	if s.chatCfg.AdminRecipient != "" {
		ageAuth, err := auth.NewAgeNonce(s.chatCfg.AdminRecipient)
		if err != nil {
			return err
		}
		adminAuth = ageAuth
	}
	adminSvc := chat.NewAdminService(chatStore, adminAuth, s.logger)

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(chat.UnaryTokenInterceptor()))
	pb.RegisterAuditServiceServer(srv, &auditServiceServer{logger: s.logger})
	chatv1.RegisterChatServiceServer(srv, chat.NewService(
		chatStore,
		resolver.NewCmdmanCompose(s.chatCfg.CmdmanBin),
		notify.NewSendKeys(s.chatCfg.CmdmanBin, s.logger),
		chat.NewCmdmanStatusMirror(s.chatCfg.CmdmanBin, s.logger),
		s.logger,
	))
	// The admin half shares the socket with the member half: it is gated by the
	// credential its own calls carry, not by the token interceptor. With no
	// admin recipient configured it registers anyway and refuses every call,
	// which tells an operator that they have a key to configure — an
	// Unimplemented would read as "this daemon is too old".
	chatv1.RegisterChatAdminServiceServer(srv, adminSvc)

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
