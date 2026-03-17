package cmdman

import (
	"context"
	"io"
	"syscall"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/cmdman/v1"
)

type monitorServer struct {
	pb.UnimplementedCommandMonitorServer
	monitor *Monitor
}

func (s *monitorServer) Attach(stream pb.CommandMonitor_AttachServer) error {
	// Send scrollback first.
	scrollback := s.monitor.ring.Bytes()
	if len(scrollback) > 0 {
		if err := stream.Send(&pb.AttachOutput{Stdout: scrollback}); err != nil {
			return err
		}
	}

	// Subscribe to live output.
	ch, unsub := s.monitor.fanout.Subscribe()
	defer unsub()

	// Read stdin from client in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			switch input := msg.Input.(type) {
			case *pb.AttachInput_Stdin:
				s.monitor.SendStdin(input.Stdin)
			case *pb.AttachInput_Resize:
				s.monitor.Resize(
					uint16(input.Resize.Rows),
					uint16(input.Resize.Cols),
				)
			}
		}
	}()

	// Send live output to client.
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&pb.AttachOutput{Stdout: data}); err != nil {
				return err
			}
		case err := <-errCh:
			if err == io.EOF {
				return nil
			}
			return err
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *monitorServer) Logs(req *pb.LogsRequest, stream pb.CommandMonitor_LogsServer) error {
	// Send scrollback.
	scrollback := s.monitor.ring.Bytes()
	if len(scrollback) > 0 {
		if err := stream.Send(&pb.LogsOutput{Data: scrollback}); err != nil {
			return err
		}
	}

	if !req.Follow {
		return nil
	}

	// Follow live output.
	ch, unsub := s.monitor.fanout.Subscribe()
	defer unsub()

	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&pb.LogsOutput{Data: data}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

func (s *monitorServer) Signal(_ context.Context, req *pb.SignalRequest) (*pb.SignalResponse, error) {
	if err := s.monitor.SignalProcess(syscall.Signal(req.Signal)); err != nil {
		return nil, err
	}
	return &pb.SignalResponse{}, nil
}

func (s *monitorServer) Status(_ context.Context, _ *pb.StatusRequest) (*pb.StatusResponse, error) {
	state, exitCode, pid := s.monitor.GetState()
	return &pb.StatusResponse{
		State:    state,
		ExitCode: int32(exitCode),
		Pid:      int32(pid),
	}, nil
}
