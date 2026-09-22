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
	"time"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	pb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/hook/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat"
	"github.com/ngicks/crabswarm/crabswarm/chat/auth"
	"github.com/ngicks/crabswarm/crabswarm/chat/notify"
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
	"google.golang.org/grpc"
)

// shutdownGrace is how long a shutting-down daemon waits for its open streams
// to end on their own before closing them. Long enough for a watcher to finish
// the event it is handling, short enough that stopping the daemon stays a
// keystroke rather than a wait.
const shutdownGrace = 5 * time.Second

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

// listenUnixDomainSocket binds sockPath. Its directory is [Server.Serve]'s to
// create, which it has done by the time this runs: the lock file lives in that
// same directory and is taken first.
func listenUnixDomainSocket(sockPath string) (net.Listener, error) {
	// Remove stale socket file if it exists.
	err := os.Remove(sockPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	return net.Listen("unix", sockPath)
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
	return chat.NewStore(ctx, path, s.chatCfg.HistoryLimit)
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
	// Refused here rather than left to the listener, because everything below
	// derives a path from this one: the directory of "" is ".", so a daemon
	// started without a socket would create the lock file in whatever directory
	// it happened to be run from and hold a lock nobody else looks for.
	if s.sockPath == "" {
		return fmt.Errorf("server listen target not specified")
	}
	// The lock file sits beside the socket, so its directory has to exist
	// before the lock is taken; on a fresh boot the runtime directory holds
	// no crabswarm/ yet and the first daemon creates it.
	if err := os.MkdirAll(filepath.Dir(s.sockPath), 0o700); err != nil {
		return fmt.Errorf("creating the socket directory: %w", err)
	}

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

	lis, err := listenUnixDomainSocket(s.sockPath)
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
	// moment the operator first tries an admin call. No recipients at all is not
	// a misspelling: it leaves the admin half with no authenticator, which is
	// what makes it refuse every call with "configure a key first".
	var adminAuth chat.AdminAuthenticator
	if len(s.chatCfg.AdminRecipients) > 0 {
		ageAuth, err := auth.NewAgeNonce(s.chatCfg.AdminRecipients...)
		if err != nil {
			return err
		}
		adminAuth = ageAuth
	}
	// One notifier for both halves: a recipient is nudged the same way whether
	// the message came from a peer or from the operator. The team-info provider
	// goes to the member half alone, which is the only one that has a token to
	// place.
	notifier := notify.NewSendKeys(s.chatCfg.CmdmanBin, s.logger)
	provider := resolver.NewCmdmanCompose(s.chatCfg.CmdmanBin)
	adminSvc := chat.NewAdminService(chatStore, adminAuth, notifier, s.logger)
	chatSvc := chat.NewService(
		chatStore,
		provider,
		notifier,
		chat.NewCmdmanStatusMirror(s.chatCfg.CmdmanBin, s.logger),
		s.logger,
	)

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(chat.UnaryTokenInterceptor()),
		// Attend is a stream, and the unary interceptor never sees one.
		grpc.ChainStreamInterceptor(chat.StreamTokenInterceptor()),
	)
	pb.RegisterAuditServiceServer(srv, &auditServiceServer{logger: s.logger})
	chatv1.RegisterChatServiceServer(srv, chatSvc)
	// The admin half shares the socket with the member half: it is gated by the
	// credential its own calls carry, not by the token interceptor. With no
	// admin recipient configured it registers anyway and refuses every call,
	// which tells an operator that they have a key to configure — an
	// Unimplemented would read as "this daemon is too old".
	chatv1.RegisterChatAdminServiceServer(srv, adminSvc)

	// A harness the daemon recognises reports on a feed of its own. A harness
	// nobody recognised says nothing about its turns, so the daemon reads the
	// state off that member's terminal instead. The poller runs for as long as
	// the server does; a negative interval is the operator saying not to run it
	// at all.
	if s.chatCfg.ScreenPollInterval >= 0 {
		// Derived from the server's context rather than being it: a daemon that
		// stopped because its listener failed never cancels that context, and the
		// poller would keep exec'ing cmdman for a server that is gone.
		pollCtx, stopPolling := context.WithCancel(ctx)
		defer stopPolling()
		poller := notify.NewScreenPoller(
			s.chatCfg.CmdmanBin,
			s.chatCfg.ScreenPollInterval,
			chatStore,
			chatSvc,
			s.logger,
		)
		go poller.Run(pollCtx)
	}

	// Graceful shutdown when context is cancelled (e.g. SIGINT).
	//
	// GracefulStop waits for every in-flight RPC, and Attend is a stream that
	// ends only when its client does — an attendee would hold the daemon open
	// through SIGINT for as long as it kept attending. Attendees get
	// shutdownGrace to notice the closing connection and hang up; after that
	// what is left is cut.
	go func() {
		<-ctx.Done()
		s.logger.Info("shutting down server")
		cut := time.AfterFunc(shutdownGrace, func() {
			s.logger.Warn("shutdown grace elapsed, closing open streams",
				slog.Duration("grace", shutdownGrace))
			srv.Stop()
		})
		defer cut.Stop()
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
