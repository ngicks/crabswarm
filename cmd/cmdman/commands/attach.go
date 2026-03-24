package commands

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/cmdman/v1"
)

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.Flags().Bool("no-stdin", false, "Output-only mode")
	attachCmd.Flags().Bool("sig-proxy", true, "Forward signals to command")
	attachCmd.Flags().String("detach-keys", "ctrl-p,ctrl-q", "Key sequence to detach")
}

var attachCmd = &cobra.Command{
	Use:   "attach [flags] ID|NAME",
	Short: "Attach to a running command's PTY",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAttach(cmd, args[0])
	},
}

func runAttach(cmd *cobra.Command, idOrName string) error {
	noStdin, _ := cmd.Flags().GetBool("no-stdin")
	sigProxy, _ := cmd.Flags().GetBool("sig-proxy")

	store, err := cmdman.OpenStore(cmdman.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	id, err := store.ResolveID(idOrName)
	if err != nil {
		return fmt.Errorf("resolve command: %w", err)
	}

	_, _, stateJSON, err := store.GetCommandState(id)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	if stateJSON.SocketPath == "" {
		return fmt.Errorf("no socket path for command %s", id)
	}

	conn, err := grpc.NewClient(
		"unix://"+stateJSON.SocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connect to monitor: %w", err)
	}
	defer conn.Close()

	client := pb.NewCommandMonitorClient(conn)
	ctx := cmd.Context()

	stream, err := client.Attach(ctx)
	if err != nil {
		return fmt.Errorf("attach: %w", err)
	}

	// Put terminal into raw mode so keystrokes pass through to the remote PTY.
	// Restore on exit to avoid leaving the terminal in a broken state.
	if !noStdin {
		fd := int(os.Stdin.Fd())
		if term.IsTerminal(fd) {
			oldState, err := term.MakeRaw(fd)
			if err == nil {
				defer term.Restore(fd, oldState)
			}
		}
	}

	// Signal proxy.
	if sigProxy {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			for sig := range sigCh {
				sigNum, ok := sig.(syscall.Signal)
				if !ok {
					continue
				}
				client.Signal(ctx, &pb.SignalRequest{Signal: int32(sigNum)})
			}
		}()
		defer signal.Stop(sigCh)
	}

	// Send terminal size.
	sendResize(stream)

	// Watch for SIGWINCH.
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)
	go func() {
		for range winchCh {
			sendResize(stream)
		}
	}()
	defer signal.Stop(winchCh)

	// Read from stream -> stdout.
	errCh := make(chan error, 2)
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			os.Stdout.Write(msg.Stdout)
		}
	}()

	// Read stdin -> stream.
	if !noStdin {
		go func() {
			buf := make([]byte, 32*1024)
			for {
				n, err := os.Stdin.Read(buf)
				if n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					if sendErr := stream.Send(&pb.AttachInput{
						Input: &pb.AttachInput_Stdin{Stdin: data},
					}); sendErr != nil {
						errCh <- sendErr
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						errCh <- err
					}
					return
				}
			}
		}()
	}

	// Wait for either direction to finish.
	select {
	case err := <-errCh:
		if err == io.EOF {
			return nil
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func sendResize(stream pb.CommandMonitor_AttachClient) {
	rows, cols := getTerminalSize()
	if rows > 0 && cols > 0 {
		stream.Send(&pb.AttachInput{
			Input: &pb.AttachInput_Resize{
				Resize: &pb.ResizeEvent{
					Rows: uint32(rows),
					Cols: uint32(cols),
				},
			},
		})
	}
}

func getTerminalSize() (rows, cols int) {
	return getTerminalSizeImpl()
}
