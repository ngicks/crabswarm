// Package server provides the interactive permission server.
package server

import (
	"context"
	"fmt"
	"io"
	"net"

	tea "github.com/charmbracelet/bubbletea"
	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	impl "github.com/ngicks/crabswarm/hook/api/impl/go/permission/v1"
	"google.golang.org/grpc"
)

// Prompter is the interface for prompting the user for permission decisions.
type Prompter interface {
	Prompt(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error)
}

// Server is the interactive permission server.
type Server struct {
	prompter   Prompter
	grpcServer *grpc.Server
	listener   net.Listener
	program    *tea.Program
}

// Config holds the server configuration.
type Config struct {
	// Address is the address to listen on (e.g., "localhost:50051").
	Address string
	// Prompter is the prompter to use for handling permission requests.
	// If nil, a plain text prompter is created using Reader/Writer.
	Prompter Prompter
	// Program is the bubbletea program for TUI mode. If set, Serve() runs
	// gRPC in a background goroutine and bubbletea as the main loop.
	Program *tea.Program
	// Reader is the input source for prompts (only used when Prompter is nil).
	Reader io.Reader
	// Writer is the output destination for prompts (only used when Prompter is nil).
	Writer io.Writer
}

// New creates a new Server with the given configuration.
func New(cfg Config) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", cfg.Address, err)
	}

	prompter := cfg.Prompter
	if prompter == nil {
		prompter = NewPlainPrompter(cfg.Reader, cfg.Writer)
	}

	grpcServer := grpc.NewServer()

	server := &Server{
		prompter:   prompter,
		grpcServer: grpcServer,
		listener:   listener,
		program:    cfg.Program,
	}

	// Register the permission service
	service := impl.NewService(server)
	pb.RegisterPermissionServiceServer(grpcServer, service)

	return server, nil
}

// HandlePermissionRequest implements the impl.PermissionHandler interface.
func (s *Server) HandlePermissionRequest(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	return s.prompter.Prompt(ctx, req)
}

// Serve starts the server and blocks until stopped.
// In TUI mode, gRPC runs in a background goroutine and bubbletea runs as the main loop.
// In plain mode, gRPC runs directly.
func (s *Server) Serve() error {
	if s.program != nil {
		// TUI mode: run gRPC in background, bubbletea as main loop
		errCh := make(chan error, 1)
		go func() {
			errCh <- s.grpcServer.Serve(s.listener)
		}()

		// Run bubbletea (blocks until user quits with ctrl+c)
		if _, err := s.program.Run(); err != nil {
			s.grpcServer.GracefulStop()
			return fmt.Errorf("TUI error: %w", err)
		}

		// TUI exited, stop gRPC
		s.grpcServer.GracefulStop()

		// Check if gRPC had an error
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("gRPC error: %w", err)
			}
		default:
		}

		return nil
	}

	// Plain mode
	return s.grpcServer.Serve(s.listener)
}

// Address returns the address the server is listening on.
func (s *Server) Address() string {
	return s.listener.Addr().String()
}

// Stop gracefully stops the server.
func (s *Server) Stop() {
	if s.program != nil {
		s.program.Quit()
	}
	s.grpcServer.GracefulStop()
}
