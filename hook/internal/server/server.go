// Package server provides the interactive permission server.
package server

import (
	"context"
	"fmt"
	"io"
	"net"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	impl "github.com/ngicks/crabswarm/hook/api/impl/go/permission/v1"
	"google.golang.org/grpc"
)

// Server is the interactive permission server.
type Server struct {
	prompter   *Prompter
	grpcServer *grpc.Server
	listener   net.Listener
}

// Config holds the server configuration.
type Config struct {
	// Address is the address to listen on (e.g., "localhost:50051").
	Address string
	// Reader is the input source for prompts (typically os.Stdin).
	Reader io.Reader
	// Writer is the output destination for prompts (typically os.Stdout).
	Writer io.Writer
}

// New creates a new Server with the given configuration.
func New(cfg Config) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", cfg.Address, err)
	}

	prompter := NewPrompter(cfg.Reader, cfg.Writer)
	grpcServer := grpc.NewServer()

	server := &Server{
		prompter:   prompter,
		grpcServer: grpcServer,
		listener:   listener,
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

// Serve starts the gRPC server and blocks until stopped.
func (s *Server) Serve() error {
	return s.grpcServer.Serve(s.listener)
}

// Address returns the address the server is listening on.
func (s *Server) Address() string {
	return s.listener.Addr().String()
}

// Stop gracefully stops the server.
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}
